package core

import (
	"errors"
	"fmt"
	"io"

	"github.com/kpfaulkner/jxl-go/bundle"
	"github.com/kpfaulkner/jxl-go/colour"
	"github.com/kpfaulkner/jxl-go/frame"
	image2 "github.com/kpfaulkner/jxl-go/image"
	"github.com/kpfaulkner/jxl-go/jxlio"
	"github.com/kpfaulkner/jxl-go/options"
	"github.com/kpfaulkner/jxl-go/util"
)

// Box information (not sure what this is yet)
type BoxInfo struct {
	boxSize   uint32
	posInBox  uint32
	container bool
}

// JXLCodestreamDecoder decodes the JXL image
type JXLCodestreamDecoder struct {
	// bit reader... the actual thing that will read the bits/U16/U32/U64 etc.

	reference   [][]image2.ImageBuffer
	lfBuffer    [][]image2.ImageBuffer
	canvas      []image2.ImageBuffer
	boxHeaders  []ContainerBoxHeader
	bitReader   jxlio.BitReader
	imageHeader *bundle.ImageHeader

	options        options.JXLOptions
	level          int
	foundSignature bool
}

func NewJXLCodestreamDecoder(br jxlio.BitReader, opts *options.JXLOptions) *JXLCodestreamDecoder {
	jxl := &JXLCodestreamDecoder{}
	jxl.bitReader = br
	jxl.foundSignature = false
	jxl.lfBuffer = make([][]image2.ImageBuffer, 5)
	if opts != nil {
		jxl.options = *options.NewJXLOptions(opts)
	}
	jxl.reference = make([][]image2.ImageBuffer, 4)
	return jxl
}

func (jxl *JXLCodestreamDecoder) atEnd() bool {
	if jxl.bitReader != nil {
		return jxl.bitReader.AtEnd()
	}
	return false
}

// GetImageHeader just duplicates the first chunk of code from decode. This is so we can get the image size
// and colour model.
func (jxl *JXLCodestreamDecoder) GetImageHeader() (*bundle.ImageHeader, error) {

	// read header to get signature
	err := jxl.readSignatureAndBoxes()
	if err != nil {
		return nil, err
	}

	for _, box := range jxl.boxHeaders {
		_, err := jxl.bitReader.Seek(box.Offset, io.SeekStart)
		if err != nil {
			return nil, err
		}

		if jxl.atEnd() {
			return nil, nil
		}

		level := int32(jxl.level)
		imageHeader, err := bundle.ParseImageHeader(jxl.bitReader, level)
		if err != nil {
			return nil, err
		}

		return imageHeader, nil
	}
	return nil, errors.New("unable to find image header")
}

func (jxl *JXLCodestreamDecoder) decode() (*JXLImage, error) {

	// read header to get signature
	err := jxl.readSignatureAndBoxes()
	if err != nil {
		return nil, err
	}

	box := jxl.boxHeaders[0]
	_, err = jxl.bitReader.Seek(box.Offset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	if jxl.atEnd() {
		return nil, nil
	}

	level := int32(jxl.level)
	imageHeader, err := bundle.ParseImageHeader(jxl.bitReader, level)
	if err != nil {
		return nil, err
	}
	jxl.imageHeader = imageHeader
	size := imageHeader.Size
	jxl.canvas = make([]image2.ImageBuffer, imageHeader.GetColourChannelCount()+len(imageHeader.ExtraChannelInfo))
	if imageHeader.AnimationHeader != nil {
		return nil, errors.New("animation not implemented")
	}

	if imageHeader.PreviewSize != nil {
		previewOptions := options.NewJXLOptions(&jxl.options)
		previewOptions.ParseOnly = true
		frame := frame.NewFrameWithReader(jxl.bitReader, jxl.imageHeader, previewOptions)
		frame.ReadFrameHeader()
		return nil, errors.New("not implemented preview yet")
	}

	var matrix *colour.OpsinInverseMatrix
	if imageHeader.XybEncoded {
		bundle := imageHeader.ColourEncoding
		matrix, err = imageHeader.OpsinInverseMatrix.GetMatrix(bundle.Prim, bundle.White)
		if err != nil {
			return nil, err
		}
	}

	frameCount := 0
	invisibleFrames := int64(0)
	visibleFrames := 0
	header := frame.FrameHeader{}

	// If we have multiple box headers...  then we want to loop over from the second one.
	// Need to also seek to that box offset.
	ii := 0
	shouldSeekBoxOffset := false
	if len(jxl.boxHeaders) > 1 {
		ii = 1
		shouldSeekBoxOffset = true
	}

	for _, box := range jxl.boxHeaders[ii:] {

		// only seek if we're not dealing with a single box.
		if shouldSeekBoxOffset {
			_, err = jxl.bitReader.Seek(box.Offset, io.SeekStart)
			if err != nil {
				return nil, err
			}
		}

		if jxl.atEnd() {
			return nil, nil
		}

		for {
			imgFrame := frame.NewFrameWithReader(jxl.bitReader, jxl.imageHeader, &jxl.options)
			header, err = imgFrame.ReadFrameHeader()
			if err != nil {
				return nil, err
			}
			frameCount++

			//showNextNBytes(jxl.bitReader, 4)
			if jxl.lfBuffer[header.LfLevel] == nil && header.Flags&frame.USE_LF_FRAME != 0 {
				return nil, errors.New("LF level too large")
			}

			err := imgFrame.ReadTOC()
			if err != nil {
				return nil, err
			}

			if jxl.options.ParseOnly {
				imgFrame.SkipFrameData()
				continue
			}
			err = imgFrame.DecodeFrame(jxl.lfBuffer[header.LfLevel])
			if err != nil {
				return nil, err
			}

			if header.LfLevel > 0 {
				jxl.lfBuffer[header.LfLevel-1] = imgFrame.Buffer
			}
			if header.FrameType == frame.LF_FRAME {
				continue
			}
			save := (header.SaveAsReference != 0 || header.Duration == 0) && !header.IsLast && header.FrameType != frame.LF_FRAME
			if imgFrame.IsVisible() {
				visibleFrames++
				invisibleFrames = 0
			} else {
				invisibleFrames++
			}

			err = imgFrame.Upsample()
			if err != nil {
				return nil, err
			}

			err = imgFrame.InitializeNoise(int64(visibleFrames<<32) | invisibleFrames)
			if err != nil {
				return nil, err
			}

			if save && header.SaveBeforeCT {
				jxl.reference[header.SaveAsReference] = imgFrame.Buffer
			}

			err = jxl.computePatches(imgFrame)
			if err != nil {
				return nil, err
			}

			err = imgFrame.RenderSplines()
			if err != nil {
				return nil, err
			}

			err = imgFrame.SynthesizeNoise()
			if err != nil {
				return nil, err
			}

			err = jxl.performColourTransforms(matrix, imgFrame)
			if err != nil {
				return nil, err
			}

			if header.Encoding == frame.VARDCT && jxl.options.RenderVarblocks {
				panic("VARDCT not implemented yet")
			}

			if jxl.canvas[0].Height == 0 && jxl.canvas[0].Width == 0 {
				for c := 0; c < len(jxl.canvas); c++ {
					canvas, err := image2.NewImageBuffer(imgFrame.Buffer[0].BufferType, int32(size.Height), int32(size.Width))
					if err != nil {
						return nil, err
					}
					jxl.canvas[c] = *canvas
				}
			}
			if header.FrameType == frame.REGULAR_FRAME || header.FrameType == frame.SKIP_PROGRESSIVE {
				found := false
				for i := uint32(0); i < 4; i++ {
					if image2.ImageBufferSliceEquals(jxl.reference[i], jxl.canvas) && i != header.SaveAsReference {
						found = true
						break
					}
				}

				if found {
					//FIXME(kpfaulkner)
					// dumb copy of canvas?

					canvas2 := make([]image2.ImageBuffer, len(jxl.canvas))
					for _, ib := range jxl.canvas {
						ib2 := image2.NewImageBufferFromImageBuffer(&ib)
						canvas2 = append(canvas2, *ib2)
					}

				}
				err = jxl.blendFrame(jxl.canvas, imgFrame)
				if err != nil {
					return nil, err
				}
			}

			if save && !header.SaveBeforeCT {
				jxl.reference[header.SaveAsReference] = jxl.canvas
			}

			if header.IsLast && header.Duration == 0 {
				break
			}
		}

		err = jxl.bitReader.ZeroPadToByte()
		if err != nil {
			return nil, err
		}

		// TOOD(kpfaulkner) unsure if need to perform similar drain cache functionality here. Don't think we do.
		if jxl.options.ParseOnly {
			return nil, nil
		}

		orientation := imageHeader.Orientation
		orientedCanvas := make([]image2.ImageBuffer, len(jxl.canvas))
		for i := 0; i < len(orientedCanvas); i++ {
			orientedCanvas[i], err = jxl.transposeBuffer(jxl.canvas[i], orientation)
			if err != nil {
				return nil, err
			}
		}

		// generate image and return.
		img, err := NewJXLImageWithBuffer(orientedCanvas, *imageHeader)
		if err != nil {
			return nil, err
		}

		return img, nil
	}

	panic("make JXL image here?")
	return nil, nil
}

func showNextNBytes(reader jxlio.BitReader, prefix string, n int) {
	b, _ := reader.ShowBits(8 * n)
	fmt.Printf(prefix + " ")
	for i := 0; i < n; i++ {
		fmt.Printf("%02x ", b&0xFF)
		b >>= 8
	}
	fmt.Printf("\n")
}

// Read signature
func (jxl *JXLCodestreamDecoder) readSignatureAndBoxes() error {

	br := NewBoxReader(jxl.bitReader)
	boxHeaders, err := br.ReadBoxHeader()
	if err != nil {
		return err
	}

	jxl.boxHeaders = boxHeaders
	jxl.level = br.level
	return nil
}

func (jxl *JXLCodestreamDecoder) computePatches(frame *frame.Frame) error {

	// do not support patches yet.
	return nil

	//header := frame.Header
	//frameBuffer := frame.Buffer
	//colourChannels := jxl.imageHeader.GetColourChannelCount()
	//extraChannels := len(jxl.imageHeader.ExtraChannelInfo)
	//patches := frame.LfGlobal.Patches
	//hasAlpha := jxl.imageHeader.HasAlpha()
	//for i := 0; i < len(patches); i++ {
	//	patch := patches[i]
	//	if patch.Ref > 3 {
	//		return errors.New("patch out of range")
	//	}
	//	refBuffer := jxl.reference[patch.Ref]
	//	if refBuffer == nil || len(refBuffer) == 0 {
	//		continue
	//	}
	//	lowerCorner := patch.Bounds.ComputeLowerCorner()
	//	if lowerCorner.Y > refBuffer[0].Height || lowerCorner.X > refBuffer[0].Width {
	//		return errors.New("patch too large")
	//	}
	//	for j := 0; i < len(patch.Positions); j++ {
	//		x0 := patch.Positions[j].X
	//		y0 := patch.Positions[j].Y
	//		if x0 < 0 || y0 < 0 {
	//			return errors.New("patch size out of bounds")
	//		}
	//
	//		if patch.Bounds.Size.Height+uint32(y0) > header.Bounds.Size.Height ||
	//			patch.Bounds.Size.Width+uint32(x0) > header.Bounds.Size.Width {
	//			return errors.New("patch size out of bounds")
	//		}
	//
	//		for d := int32(0); d < int32(colourChannels)+int32(extraChannels); d++ {
	//			var c int32
	//			if d < int32(colourChannels) {
	//				c = 0
	//			} else {
	//				c = d - int32(colourChannels) + 1
	//			}
	//			info := patch.BlendingInfos[j][c]
	//			if info.Mode == 0 {
	//				continue
	//			}
	//			var premult bool
	//			if jxl.imageHeader.HasAlpha() {
	//				premult = jxl.imageHeader.ExtraChannelInfo[info.AlphaChannel].AlphaAssociated
	//			} else {
	//				premult = true
	//			}
	//			isAlpha := c > 0 && jxl.imageHeader.ExtraChannelInfo[c-1].EcType == bundle.ALPHA
	//			if info.Mode > 0 && header.Upsampling > 1 && c > 0 && header.EcUpsampling[c-1]<<jxl.imageHeader.ExtraChannelInfo[c-1].DimShift != header.Upsampling {
	//				return errors.New("Alpha channel upsampling mismatch during patches")
	//			}
	//
	//			toFloat := true
	//			switch info.Mode {
	//			case 1:
	//				if refBuffer[0].IsInt() && frameBuffer[d].IsInt() {
	//					refBufferI := refBuffer[d].IntBuffer
	//					frameBufferI := frameBuffer[d].IntBuffer
	//					for y := uint32(0); y < patch.Bounds.Size.Height; y++ {
	//						copy(frameBufferI[y+uint32(patch.Bounds.Origin.Y)][patch.Bounds.Origin.X:], refBufferI[y0+int32(y)][x0:])
	//					}
	//					toFloat = false
	//				}
	//				break
	//			case 2:
	//				if refBuffer[0].IsInt() && frameBuffer[d].IsInt() {
	//					refBufferI := refBuffer[d].IntBuffer
	//					frameBufferI := frameBuffer[d].IntBuffer
	//					for y := int32(0); y < int32(patch.Bounds.Size.Height); y++ {
	//						for x := int32(0); x < int32(patch.Bounds.Size.Width); x++ {
	//							frameBufferI[int32(y0)+y][int32(x0)+x] += refBufferI[patch.Bounds.Origin.Y+y][patch.Bounds.Origin.X+x]
	//						}
	//					}
	//					toFloat = false
	//				}
	//				break
	//			}
	//
	//			if toFloat {
	//				var depth uint32
	//				if c == 0 {
	//					depth = jxl.imageHeader.BitDepth.BitsPerSample
	//				} else {
	//					depth = jxl.imageHeader.ExtraChannelInfo[c-1].BitDepth.BitsPerSample
	//				}
	//				max := ^(^int32(0) << depth)
	//				refBuffer[d].CastToFloatIfInt(max)
	//				frameBuffer[d].CastToFloatIfInt(max)
	//			}
	//			var refBufferF [][]float32
	//			var frameBufferF [][]float32
	//			if toFloat {
	//				refBufferF = refBuffer[d].FloatBuffer
	//				frameBufferF = frameBuffer[d].FloatBuffer
	//			} else {
	//				refBufferF = nil
	//				frameBufferF = nil
	//			}
	//			var alphaBufferOld [][]float32
	//			var alphaBufferNew [][]float32
	//			if info.Mode > 3 && hasAlpha {
	//				depth := jxl.imageHeader.ExtraChannelInfo[info.AlphaChannel].BitDepth.BitsPerSample
	//				if err := frameBuffer[colourChannels+int(info.AlphaChannel)].CastToFloatIfInt(^(^0 << depth)); err != nil {
	//					return err
	//				}
	//				if err := refBuffer[colourChannels+int(info.AlphaChannel)].CastToFloatIfInt(^(^0 << depth)); err != nil {
	//					return err
	//				}
	//				alphaBufferOld = frameBuffer[colourChannels+int(info.AlphaChannel)].FloatBuffer
	//				alphaBufferNew = refBuffer[colourChannels+int(info.AlphaChannel)].FloatBuffer
	//			} else {
	//				alphaBufferOld = nil
	//				alphaBufferNew = nil
	//			}
	//
	//			switch info.Mode {
	//			case 1:
	//				if !toFloat {
	//					break
	//				}
	//				for y := 0; y < int(patch.Bounds.Size.Height); y++ {
	//					copy(frameBufferF[y+int(patch.Bounds.Origin.Y)][int(patch.Bounds.Origin.X):], refBufferF[int(y0)+y][x0:])
	//				}
	//				break
	//			case 2:
	//				if !toFloat {
	//					break
	//				}
	//				for y := int32(0); y < int32(patch.Bounds.Size.Height); y++ {
	//					for x := int32(0); x < int32(patch.Bounds.Size.Width); x++ {
	//						frameBufferF[y0+y][x0+x] += refBufferF[patch.Bounds.Origin.Y+y][patch.Bounds.Origin.X+x]
	//					}
	//				}
	//				break
	//			case 3:
	//				for y := uint32(0); y < patch.Bounds.Size.Height; y++ {
	//					for x := uint32(0); x < patch.Bounds.Size.Width; x++ {
	//						frameBufferF[uint32(y0)+y][uint32(x0)+x] *= refBufferF[uint32(patch.Bounds.Origin.Y)+y][uint32(patch.Bounds.Origin.X)+x]
	//					}
	//				}
	//				break
	//			case 4:
	//				if isAlpha {
	//					for y := int32(0); y < int32(patch.Bounds.Size.Height); y++ {
	//						newY := y + patch.Bounds.Origin.Y
	//						oldY := y + int32(y0)
	//						for x := int32(0); x < int32(patch.Bounds.Size.Width); x++ {
	//							oldX := x + int32(x0)
	//							newX := x + patch.Bounds.Origin.X
	//							newAlpha := alphaBufferNew[newY][newX]
	//							if info.Clamp {
	//								if newAlpha < 0 {
	//									newAlpha = 0
	//								} else if newAlpha > 1 {
	//									newAlpha = 1
	//								}
	//							}
	//							frameBufferF[oldY][oldX] = alphaBufferOld[oldY][oldY] +
	//								newAlpha*(1-alphaBufferOld[oldY][oldX])
	//						}
	//					}
	//				} else if premult {
	//					for y := int32(0); y < int32(patch.Bounds.Size.Height); y++ {
	//						newY := y + patch.Bounds.Origin.Y
	//						oldY := y + int32(y0)
	//						for x := int32(0); x < int32(patch.Bounds.Size.Width); x++ {
	//							newX := x + patch.Bounds.Origin.X
	//							oldX := x + int32(x0)
	//							newAlpha := alphaBufferNew[newY][newX]
	//							if info.Clamp {
	//								if newAlpha < 0 {
	//									newAlpha = 0
	//								} else if newAlpha > 1 {
	//									newAlpha = 1
	//								}
	//							}
	//							frameBufferF[oldY][oldX] = refBufferF[newY][newX] + frameBufferF[oldY][oldX]*(1-newAlpha)
	//						}
	//					}
	//				} else {
	//					for y := int32(0); y < int32(patch.Bounds.Size.Height); y++ {
	//						newY := y + patch.Bounds.Origin.Y
	//						oldY := y + int32(y0)
	//						for x := int32(0); x < int32(patch.Bounds.Size.Width); x++ {
	//							newX := x + patch.Bounds.Origin.X
	//							oldX := x + int32(x0)
	//							var oldAlpha float32
	//							var newAlpha float32
	//							if hasAlpha {
	//								oldAlpha = alphaBufferOld[oldY][oldX]
	//								newAlpha = alphaBufferNew[newY][newX]
	//							} else {
	//								oldAlpha = 1
	//								newAlpha = 1
	//							}
	//							if info.Clamp {
	//								if newAlpha < 0 {
	//									newAlpha = 0
	//								} else {
	//									if newAlpha > 1 {
	//										newAlpha = 1
	//									}
	//								}
	//							}
	//							alpha := oldAlpha + newAlpha*(1-oldAlpha)
	//							frameBufferF[oldY][oldX] = (refBufferF[newY][newX]*newAlpha + frameBufferF[oldY][oldX]*oldAlpha*(1-newAlpha)) / alpha
	//						}
	//					}
	//				}
	//				break
	//			case 5:
	//				if isAlpha {
	//					for y := int32(0); y < int32(patch.Bounds.Size.Height); y++ {
	//						newY := y + patch.Bounds.Origin.Y
	//						oldY := y + int32(y0)
	//						for x := int32(0); x < int32(patch.Bounds.Size.Width); x++ {
	//							oldX := x + int32(x0)
	//							newX := x + patch.Bounds.Origin.X
	//							frameBufferF[oldY][oldX] = alphaBufferOld[oldY][oldY] +
	//								alphaBufferNew[newY][newX]*(1-alphaBufferOld[oldY][oldX])
	//						}
	//					}
	//				} else if premult {
	//					for y := int32(0); y < int32(patch.Bounds.Size.Height); y++ {
	//						newY := y + patch.Bounds.Origin.Y
	//						oldY := y + int32(y0)
	//						for x := int32(0); x < int32(patch.Bounds.Size.Width); x++ {
	//							newX := x + patch.Bounds.Origin.X
	//							oldX := x + int32(x0)
	//							newAlpha := alphaBufferNew[newY][newX]
	//							if info.Clamp {
	//								if newAlpha < 0 {
	//									newAlpha = 0
	//								} else if newAlpha > 1 {
	//									newAlpha = 1
	//								}
	//							}
	//							frameBufferF[oldY][oldX] = frameBufferF[oldY][oldX] + refBufferF[newY][newX]*(1-newAlpha)
	//						}
	//					}
	//				} else {
	//					for y := int32(0); y < int32(patch.Bounds.Size.Height); y++ {
	//						newY := y + patch.Bounds.Origin.Y
	//						oldY := y + int32(y0)
	//						for x := int32(0); x < int32(patch.Bounds.Size.Width); x++ {
	//							newX := x + patch.Bounds.Origin.X
	//							oldX := x + int32(x0)
	//							var oldAlpha float32
	//							var newAlpha float32
	//							if hasAlpha {
	//								oldAlpha = alphaBufferOld[oldY][oldX]
	//								newAlpha = alphaBufferNew[newY][newX]
	//							} else {
	//								oldAlpha = 1
	//								newAlpha = 1
	//							}
	//							alpha := oldAlpha + newAlpha*(1-oldAlpha)
	//							frameBufferF[oldY][oldX] = (frameBufferF[oldY][oldX]*newAlpha + refBufferF[newY][newX]*oldAlpha*(1-newAlpha)) / alpha
	//						}
	//					}
	//				}
	//				break
	//			case 6:
	//				if isAlpha {
	//					for y := int32(0); y < int32(patch.Bounds.Size.Height); y++ {
	//						newY := y + patch.Bounds.Origin.Y
	//						oldY := y + int32(y0)
	//						for x := int32(0); x < int32(patch.Bounds.Size.Width); x++ {
	//							oldX := x + int32(x0)
	//							newX := x + patch.Bounds.Origin.X
	//							newAlpha := alphaBufferNew[newY][newX]
	//							if info.Clamp {
	//								if newAlpha < 0 {
	//									newAlpha = 0
	//								} else if newAlpha > 1 {
	//									newAlpha = 1
	//								}
	//							}
	//							v := float32(1.0)
	//							if !hasAlpha {
	//								v = newAlpha
	//							}
	//							frameBufferF[oldY][oldX] = v
	//						}
	//					}
	//				} else {
	//					for y := int32(0); y < int32(patch.Bounds.Size.Height); y++ {
	//						newY := y + patch.Bounds.Origin.Y
	//						oldY := y + int32(y0)
	//						for x := int32(0); x < int32(patch.Bounds.Size.Width); x++ {
	//							newX := x + patch.Bounds.Origin.X
	//							oldX := x + int32(x0)
	//							newAlpha := alphaBufferNew[newY][newX]
	//							if info.Clamp {
	//								if newAlpha < 0 {
	//									newAlpha = 0
	//								} else if newAlpha > 1 {
	//									newAlpha = 1
	//								}
	//							}
	//							frameBufferF[oldY][oldX] += refBufferF[newY][newX]
	//						}
	//					}
	//				}
	//				break
	//			case 7:
	//				if isAlpha {
	//					for y := int32(0); y < int32(patch.Bounds.Size.Height); y++ {
	//						oldY := y + int32(y0)
	//						for x := int32(0); x < int32(patch.Bounds.Size.Width); x++ {
	//							oldX := x + int32(x0)
	//
	//							v := float32(1.0)
	//							if !hasAlpha {
	//								v = alphaBufferOld[oldY][oldX]
	//							}
	//							frameBufferF[oldY][oldX] = v
	//						}
	//					}
	//				} else {
	//					for y := int32(0); y < int32(patch.Bounds.Size.Height); y++ {
	//						newY := y + patch.Bounds.Origin.Y
	//						oldY := y + int32(y0)
	//						for x := int32(0); x < int32(patch.Bounds.Size.Width); x++ {
	//							newX := x + patch.Bounds.Origin.X
	//							oldX := x + int32(x0)
	//							var oldAlpha float32
	//							var newAlpha float32
	//							if hasAlpha {
	//								oldAlpha = alphaBufferOld[oldY][oldX]
	//								newAlpha = alphaBufferNew[newY][newX]
	//							} else {
	//								oldAlpha = 1
	//								newAlpha = 1
	//							}
	//							if info.Clamp {
	//								if newAlpha < 0 {
	//									newAlpha = 0
	//								} else if newAlpha > 1 {
	//									newAlpha = 1
	//								}
	//							}
	//							alpha := oldAlpha + newAlpha*(1-oldAlpha)
	//							frameBufferF[oldY][oldX] = refBufferF[newY][newX] + alpha*frameBufferF[oldY][oldX]
	//						}
	//					}
	//				}
	//				break
	//			default:
	//				return errors.New("unknown blending mode")
	//			}
	//		}
	//	}
	//}
	//return nil
}

func (jxl *JXLCodestreamDecoder) performColourTransforms(matrix *colour.OpsinInverseMatrix, frame *frame.Frame) error {

	if matrix == nil && !frame.Header.DoYCbCr {
		return nil
	}

	buffer := frame.Buffer
	buffers := util.MakeMatrix3D[float32](3, 0, 0)
	depth := jxl.imageHeader.BitDepth.BitsPerSample
	for c := 0; c < 3; c++ {
		if buffer[c].IsInt() {
			if err := buffer[c].CastToFloatIfInt(^(^0 << depth)); err != nil {
				return err
			}
		}
		buffers[c] = buffer[c].FloatBuffer
	}

	if matrix != nil {
		err := matrix.InvertXYB(buffers, jxl.imageHeader.ToneMapping.GetIntensityTarget())
		if err != nil {
			return err
		}
	}

	if frame.Header.DoYCbCr {
		size, err := frame.GetPaddedFrameSize()
		if err != nil {
			return err
		}
		for y := uint32(0); y < size.Height; y++ {
			for x := uint32(0); x < size.Width; x++ {
				cb := buffers[0][y][x]
				yh := buffers[1][y][x] + 0.50196078431372549019
				cr := buffers[2][y][x]
				buffers[0][y][x] = yh + 1.402*cr
				buffers[1][y][x] = yh - 0.34413628620102214650*cb - 0.71413628620102214650*cr
				buffers[2][y][x] = yh + 1.772*cb
			}
		}
	}
	return nil
}

func (jxl *JXLCodestreamDecoder) blendFrame(canvas []image2.ImageBuffer, imgFrame *frame.Frame) error {

	imageSize := jxl.imageHeader.GetSize()
	header := imgFrame.Header
	frameStartY := int32(0)
	if header.Bounds.Origin.X >= 0 {
		frameStartY = header.Bounds.Origin.Y
	}
	frameStartX := int32(0)
	if header.Bounds.Origin.X >= 0 {
		frameStartX = header.Bounds.Origin.X
	}
	frameOffsetY := frameStartY - header.Bounds.Origin.Y
	frameOffsetX := frameStartX - header.Bounds.Origin.X
	lowerCorner := header.Bounds.ComputeLowerCorner()
	frameHeight := util.Min(lowerCorner.Y, int32(imageSize.Height)) - frameStartY
	frameWidth := util.Min(lowerCorner.X, int32(imageSize.Width)) - frameStartX

	frameColours := imgFrame.GetColourChannelCount()
	imageColours := jxl.imageHeader.GetColourChannelCount()
	hasAlpha := jxl.imageHeader.HasAlpha()
	frameBuffers := imgFrame.Buffer
	for c := int32(0); c < int32(len(canvas)); c++ {
		var frameC int32
		if frameColours != imageColours {
			if c == 0 {
				frameC = 1
			} else {
				frameC = c + 2
			}
		} else {
			frameC = c
		}
		frameBuffer := imgFrame.Buffer[frameC]
		var info *frame.BlendingInfo
		if frameC < int32(frameColours) {
			info = imgFrame.Header.BlendingInfo
		} else {
			info = &imgFrame.Header.EcBlendingInfo[frameC-int32(frameColours)]
		}
		isAlpha := c >= int32(imageColours) && jxl.imageHeader.ExtraChannelInfo[c-int32(imageColours)].EcType == bundle.ALPHA
		premult := hasAlpha && jxl.imageHeader.ExtraChannelInfo[info.AlphaChannel].AlphaAssociated

		refBuffer := jxl.reference[info.Source]
		err := jxl.convertCanvasWithDifferentBufferType(canvas, c, &frameBuffer, imageColours, frameC, frameColours)
		if err != nil {
			return err
		}
		if info.Mode == frame.BLEND_REPLACE || refBuffer == nil && info.Mode == frame.BLEND_ADD {
			offY := frameStartY - header.Bounds.Origin.Y
			offX := frameStartX - header.Bounds.Origin.X
			jxl.copyToCanvas(&canvas[c], util.Point{Y: frameStartY, X: frameStartX}, util.Point{X: offX, Y: offY},
				util.Dimension{Width: uint32(frameWidth), Height: uint32(frameHeight)}, frameBuffer)
			continue
		}

		// check if just the zero value. Think dimension being 0 should be ok.
		if refBuffer[c].Width == 0 && refBuffer[c].Height == 0 {
			refBuf, err := image2.NewImageBuffer(frameBuffer.BufferType, canvas[c].Height, canvas[c].Width)
			if err != nil {
				return err
			}
			refBuffer[c] = *refBuf
		}
		ref := refBuffer[c]

		err2 := jxl.blendAlpha(canvas[c], hasAlpha, info, imageColours, refBuffer, frameBuffers)
		if err2 != nil {
			return err2
		}

		err3 := jxl.convertReferenceWithDifferentBufferType(canvas, &ref, &frameBuffer, info, c, imageColours, frameC, frameColours)
		if err3 != nil {
			return err3
		}

		err4 := jxl.performBlending(canvas, info, frameBuffer, c, ref, frameHeight, frameStartY, frameOffsetY, frameWidth, frameStartX, frameOffsetX, hasAlpha, refBuffer, imageColours, frameBuffers, frameColours, isAlpha, premult)
		if err4 != nil {
			return err4
		}
	}
	return nil
}

// perform blending...  was refactored to a separate function but the number of parameters
// is insane. Need to tidy this up... but will leave for now.
func (jxl *JXLCodestreamDecoder) performBlending(canvas []image2.ImageBuffer,
	info *frame.BlendingInfo, frameBuffer image2.ImageBuffer,
	canvasIdx int32, ref image2.ImageBuffer,
	frameHeight int32, frameStartY int32,
	frameOffsetY int32, frameWidth int32,
	frameStartX int32, frameOffsetX int32,
	hasAlpha bool,
	refBuffer []image2.ImageBuffer,
	imageColours int,
	frameBuffers []image2.ImageBuffer,
	frameColours int, isAlpha bool, premult bool) error {

	var cf, rf, ff, oaf, naf [][]float32
	if info.Mode != frame.BLEND_ADD || frameBuffer.IsFloat() {
		cf = canvas[canvasIdx].FloatBuffer
		rf = ref.FloatBuffer
		ff = frameBuffer.FloatBuffer
	} else {
		cf = nil
		rf = nil
		ff = nil
	}

	switch info.Mode {
	case frame.BLEND_ADD:
		if frameBuffer.IsInt() {
			ci := canvas[canvasIdx].IntBuffer
			ri := ref.IntBuffer
			fi := frameBuffer.IntBuffer
			for y := int32(0); y < frameHeight; y++ {
				cy := y + frameStartY
				fy := y + frameOffsetY
				for x := int32(0); x < frameWidth; x++ {
					cx := x + frameStartX
					fx := x + frameOffsetX
					ci[cy][cx] = ri[cy][cx] + fi[fy][fx]
				}
			}
		} else {
			for y := int32(0); y < frameHeight; y++ {
				cy := y + frameStartY
				fy := y + frameOffsetY
				for x := int32(0); x < frameWidth; x++ {
					cx := x + frameStartX
					fx := x + frameOffsetX
					cf[cy][cx] = rf[cy][cx] + ff[fy][fx]
				}
			}
		}
		break
	case frame.BLEND_MULT:
		for y := int32(0); y < frameHeight; y++ {
			cy := y + frameStartY
			fy := y + frameOffsetY
			for x := int32(0); x < frameWidth; x++ {
				cx := x + frameStartX
				fx := x + frameOffsetX
				newSample := ff[fy][fx]
				if info.Clamp {
					if newSample < 0 {
						newSample = 0
					} else if newSample > 1 {
						newSample = 1
					}
				}
				cf[cy][cx] = newSample * rf[cy][cx]
			}
		}
		break
	case frame.BLEND_BLEND:
		if hasAlpha {
			oaf = refBuffer[imageColours+int(info.AlphaChannel)].FloatBuffer
			naf = frameBuffers[frameColours+int(info.AlphaChannel)].FloatBuffer
		} else {
			oaf = nil
			naf = nil
		}

		for y := int32(0); y < frameHeight; y++ {
			cy := y + frameStartY
			fy := y + frameOffsetY
			for x := int32(0); x < frameWidth; x++ {
				cx := x + frameStartX
				fx := x + frameOffsetX
				var oldAlpha float32
				var newAlpha float32
				if hasAlpha {
					oldAlpha = oaf[cy][cx]
					newAlpha = naf[fy][fx]
				} else {
					oldAlpha = 1.0
					newAlpha = 1.0
				}
				if info.Clamp {
					if newAlpha < 0 {
						newAlpha = 0
					} else if newAlpha > 1 {
						newAlpha = 1
					}
				}
				alpha := float32(1)
				oldSample := rf[cy][cx]
				newSample := ff[fy][fx]
				if isAlpha || hasAlpha && !premult {
					alpha = oldAlpha + newAlpha*(1-oldAlpha)
				}
				if isAlpha {
					cf[cy][cx] = alpha
				} else if !hasAlpha || premult {
					cf[cy][cx] = newSample + oldSample*(1-newAlpha)
				} else {
					cf[cy][cx] = (newSample*newAlpha + oldSample*oldAlpha*(1-newAlpha)) / alpha
				}
			}
		}
		break
	case frame.BLEND_MULADD:
		if hasAlpha {
			oaf = refBuffer[imageColours+int(info.AlphaChannel)].FloatBuffer
			naf = frameBuffers[frameColours+int(info.AlphaChannel)].FloatBuffer
		} else {
			oaf = nil
			naf = nil
		}

		for y := int32(0); y < frameHeight; y++ {
			cy := y + frameStartY
			fy := y + frameOffsetY
			for x := int32(0); x < frameWidth; x++ {
				cx := x + frameStartX
				fx := x + frameOffsetX
				var oldAlpha float32
				var newAlpha float32
				if hasAlpha {
					oldAlpha = oaf[cy][cx]
					newAlpha = naf[fy][fx]
				} else {
					oldAlpha = 1.0
					newAlpha = 1.0
				}
				if info.Clamp {
					if newAlpha < 0 {
						newAlpha = 0
					} else if newAlpha > 1 {
						newAlpha = 1
					}
				}
				oldSample := rf[cy][cx]
				newSample := ff[fy][fx]
				alpha := float32(0)
				if isAlpha {
					alpha = oldAlpha
				} else {
					alpha = oldSample + newAlpha*newSample
				}
				cf[cy][cx] = alpha
			}
		}
		break
	default:
		return errors.New("Illegal blend Mode")
	}
	return nil
}

func (jxl *JXLCodestreamDecoder) convertReferenceWithDifferentBufferType(
	canvas []image2.ImageBuffer,
	ref *image2.ImageBuffer,
	frameBuffer *image2.ImageBuffer,
	info *frame.BlendingInfo,
	canvasIdx int32,
	imageColours int,
	frameC int32,
	frameColours int) error {
	if ref.BufferType != frameBuffer.BufferType || info.Mode != frame.BLEND_ADD {
		var depthCanvas int32
		var depthFrame int32
		if canvasIdx >= int32(imageColours) {
			depthCanvas = int32(jxl.imageHeader.ExtraChannelInfo[canvasIdx-int32(imageColours)].BitDepth.BitsPerSample)
		} else {
			depthCanvas = int32(jxl.imageHeader.BitDepth.BitsPerSample)
		}
		if frameC >= int32(frameColours) {
			depthFrame = int32(jxl.imageHeader.ExtraChannelInfo[frameC-int32(frameColours)].BitDepth.BitsPerSample)
		} else {
			depthFrame = int32(jxl.imageHeader.BitDepth.BitsPerSample)
		}
		if err := frameBuffer.CastToFloatIfInt(^(^0 << depthFrame)); err != nil {
			return err
		}
		if err := canvas[canvasIdx].CastToFloatIfInt(^(^0 << depthCanvas)); err != nil {
			return err
		}
		if err := ref.CastToFloatIfInt(^(^0 << depthCanvas)); err != nil {
			return err
		}
	}
	return nil
}

func (jxl *JXLCodestreamDecoder) blendAlpha(canvas image2.ImageBuffer, hasAlpha bool, info *frame.BlendingInfo, imageColours int, refBuffer []image2.ImageBuffer, frameBuffers []image2.ImageBuffer) error {
	if hasAlpha && (info.Mode == frame.BLEND_BLEND || info.Mode == frame.BLEND_MULADD) {
		depth := jxl.imageHeader.ExtraChannelInfo[info.AlphaChannel].BitDepth.BitsPerSample
		alphaIdx := imageColours + int(info.AlphaChannel)
		if refBuffer[alphaIdx].Width == 0 && refBuffer[alphaIdx].Height == 0 {
			refBuf, err := image2.NewImageBuffer(image2.TYPE_FLOAT, canvas.Height, canvas.Width)
			if err != nil {
				return err
			}
			refBuffer[alphaIdx] = *refBuf
		}
		if !refBuffer[alphaIdx].IsFloat() {
			refBuffer[alphaIdx].CastToFloatIfInt(^(^0 << depth))
		}
		if !frameBuffers[alphaIdx].IsFloat() {
			frameBuffers[alphaIdx].CastToFloatIfInt(^(^0 << depth))
		}
	}
	return nil
}

func (jxl *JXLCodestreamDecoder) convertCanvasWithDifferentBufferType(
	canvas []image2.ImageBuffer,
	channelNo int32,
	frameBuffer *image2.ImageBuffer,
	imageColours int,
	frameChannelNo int32,
	frameColours int) error {
	if canvas[channelNo].BufferType != frameBuffer.BufferType {
		var depthCanvas int32
		if channelNo >= int32(imageColours) {
			depthCanvas = int32(jxl.imageHeader.ExtraChannelInfo[channelNo-int32(imageColours)].BitDepth.BitsPerSample)
		} else {
			depthCanvas = int32(jxl.imageHeader.BitDepth.BitsPerSample)
		}
		var depthFrame int32
		if frameChannelNo >= int32(frameColours) {
			depthFrame = int32(jxl.imageHeader.ExtraChannelInfo[frameChannelNo-int32(frameColours)].BitDepth.BitsPerSample)
		} else {
			depthFrame = int32(jxl.imageHeader.BitDepth.BitsPerSample)
		}
		if err := frameBuffer.CastToFloatIfInt(^(^0 << depthFrame)); err != nil {
			return err
		}
		if err := canvas[channelNo].CastToFloatIfInt(^(^0 << depthCanvas)); err != nil {
			return err
		}
	}
	return nil
}

// needs to handle int and float buffers...
func (jxl *JXLCodestreamDecoder) copyToCanvas(canvas *image2.ImageBuffer, start util.Point, off util.Point,
	size util.Dimension, frameBuffer image2.ImageBuffer) error {

	// if buffer type different for canvas and frame, then fail
	if canvas.BufferType != frameBuffer.BufferType {
		return errors.New("Buffer type mismatch")
	}

	if canvas.IsInt() {
		for y := uint32(0); y < size.Height; y++ {
			copy(canvas.IntBuffer[y+uint32(start.Y)][start.X:], frameBuffer.IntBuffer[y+uint32(off.Y)][off.X:uint32(off.X)+size.Width])
		}
	} else {
		for y := uint32(0); y < size.Height; y++ {
			copy(canvas.FloatBuffer[y+uint32(start.Y)][start.X:], frameBuffer.FloatBuffer[y+uint32(off.Y)][off.X:uint32(off.X)+size.Width])
		}
	}
	return nil
}

func (jxl *JXLCodestreamDecoder) transposeBuffer(src image2.ImageBuffer, orientation uint32) (image2.ImageBuffer, error) {
	if src.IsInt() {
		ints, err := jxl.transposeBufferInt(src.IntBuffer, orientation)
		if err != nil {
			return image2.ImageBuffer{}, err
		}
		return *image2.NewImageBufferFromInts(ints), nil
	} else {
		floats, err := jxl.transposeBufferFloat(src.FloatBuffer, orientation)
		if err != nil {
			return image2.ImageBuffer{}, err
		}
		return *image2.NewImageBufferFromFloats(floats), nil
	}

	return image2.ImageBuffer{}, errors.New("unable to transpose buffer")
}

func (jxl *JXLCodestreamDecoder) transposeBufferInt(src [][]int32, orientation uint32) ([][]int32, error) {

	srcHeight := len(src)
	srcWidth := len(src[0])
	srcH1 := srcHeight - 1
	srcW1 := srcWidth - 1

	var dest [][]int32
	if orientation > 4 {
		dest = util.MakeMatrix2D[int32](srcWidth, srcHeight)
	} else if orientation > 1 {
		dest = util.MakeMatrix2D[int32](srcHeight, srcWidth)
	} else {
		dest = nil
	}

	switch orientation {
	case 1:
		return src, nil
	case 2:
		for y := 0; y < srcHeight; y++ {
			for x := 0; x < srcWidth; x++ {
				dest[y][srcW1-x] = src[y][x]
			}
		}
		return dest, nil
	case 3:
		for y := 0; y < srcHeight; y++ {
			for x := 0; x < srcWidth; x++ {
				dest[srcH1-y][srcW1-x] = src[y][x]
			}
		}
		return dest, nil
	case 4:
		for y := 0; y < srcHeight; y++ {
			copy(dest[srcH1-y], src[y])
		}
		return dest, nil
	case 5:
		for y := 0; y < srcHeight; y++ {
			for x := 0; x < srcWidth; x++ {
				dest[x][y] = src[y][x]
			}
		}
		return dest, nil
	case 6:
		for y := 0; y < srcHeight; y++ {
			for x := 0; x < srcWidth; x++ {
				dest[x][srcH1-y] = src[y][x]
			}
		}
		return dest, nil
	case 7:
		for y := 0; y < srcHeight; y++ {
			for x := 0; x < srcWidth; x++ {
				dest[srcW1-x][srcH1-y] = src[y][x]
			}
		}
		return dest, nil
	case 8:
		for y := 0; y < srcHeight; y++ {
			for x := 0; x < srcWidth; x++ {
				dest[srcW1-x][y] = src[y][x]
			}
		}
		return dest, nil
	default:
		return nil, errors.New("Invalid orientation")

	}
	return nil, nil
}

func (jxl *JXLCodestreamDecoder) transposeBufferFloat(src [][]float32, orientation uint32) ([][]float32, error) {

	srcHeight := len(src)
	srcWidth := len(src[0])
	srcH1 := srcHeight - 1
	srcW1 := srcWidth - 1

	var dest [][]float32
	if orientation > 4 {
		dest = util.MakeMatrix2D[float32](srcWidth, srcHeight)
	} else if orientation > 1 {
		dest = util.MakeMatrix2D[float32](srcHeight, srcWidth)
	} else {
		dest = nil
	}

	switch orientation {
	case 1:
		return src, nil
	case 2:
		for y := 0; y < srcHeight; y++ {
			for x := 0; x < srcWidth; x++ {
				dest[y][srcW1-x] = src[y][x]
			}
		}
		return dest, nil
	case 3:
		for y := 0; y < srcHeight; y++ {
			for x := 0; x < srcWidth; x++ {
				dest[srcH1-y][srcW1-x] = src[y][x]
			}
		}
		return dest, nil
	case 4:
		for y := 0; y < srcHeight; y++ {
			copy(dest[srcH1-y], src[y])
		}
		return dest, nil
	case 5:
		for y := 0; y < srcHeight; y++ {
			for x := 0; x < srcWidth; x++ {
				dest[x][y] = src[y][x]
			}
		}
		return dest, nil
	case 6:
		for y := 0; y < srcHeight; y++ {
			for x := 0; x < srcWidth; x++ {
				dest[x][srcH1-y] = src[y][x]
			}
		}
		return dest, nil
	case 7:
		for y := 0; y < srcHeight; y++ {
			for x := 0; x < srcWidth; x++ {
				dest[srcW1-x][srcH1-y] = src[y][x]
			}
		}
		return dest, nil
	case 8:
		for y := 0; y < srcHeight; y++ {
			for x := 0; x < srcWidth; x++ {
				dest[srcW1-x][y] = src[y][x]
			}
		}
		return dest, nil
	default:
		return nil, errors.New("Invalid orientation")

	}
	return nil, nil
}
