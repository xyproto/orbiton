package frame

import (
	"github.com/kpfaulkner/jxl-go/bundle"
	"github.com/kpfaulkner/jxl-go/jxlio"
	"github.com/kpfaulkner/jxl-go/util"
)

const (
	REGULAR_FRAME    = 0
	LF_FRAME         = 1
	REFERENCE_ONLY   = 2
	SKIP_PROGRESSIVE = 3

	// lossy
	VARDCT = 0

	// lossless
	MODULAR = 1

	NOISE                      = 1
	PATCHES                    = 2
	SPLINES                    = 16
	USE_LF_FRAME               = 32
	SKIP_ADAPTIVE_LF_SMOOTHING = 128

	BLEND_REPLACE = 0
	BLEND_ADD     = 1
	BLEND_BLEND   = 2
	BLEND_MULADD  = 3
	BLEND_MULT    = 4
)

type FrameHeader struct {
	jpegUpsamplingX   []int32
	jpegUpsamplingY   []int32
	EcUpsampling      []uint32
	EcBlendingInfo    []BlendingInfo
	name              string
	Bounds            *util.Rectangle
	restorationFilter *RestorationFilter
	extensions        *bundle.Extensions
	passes            *PassesInfo
	BlendingInfo      *BlendingInfo
	Flags             uint64
	FrameType         uint32
	Width             uint32
	Height            uint32
	Upsampling        uint32
	LfLevel           uint32
	groupDim          uint32
	Encoding          uint32
	groupSizeShift    uint32
	lfGroupDim        uint32
	logGroupDim       uint32
	logLFGroupDIM     uint32
	xqmScale          uint32
	bqmScale          uint32
	Duration          uint32
	timecode          uint32
	SaveAsReference   uint32
	SaveBeforeCT      bool
	DoYCbCr           bool
	haveCrop          bool
	IsLast            bool
}

func NewFrameHeaderWithReader(reader jxlio.BitReader, parent *bundle.ImageHeader) (*FrameHeader, error) {
	fh := &FrameHeader{}
	var err error
	var allDefault bool
	if allDefault, err = reader.ReadBool(); err != nil {
		return nil, err
	}
	if allDefault {
		fh.FrameType = REGULAR_FRAME
		fh.Encoding = VARDCT
		fh.Flags = 0
	} else {
		if frameType, err := reader.ReadBits(2); err != nil {
			return nil, err
		} else {
			fh.FrameType = uint32(frameType)
		}

		if encoding, err := reader.ReadBits(1); err != nil {
			return nil, err
		} else {
			fh.Encoding = uint32(encoding)
		}
		if fh.Flags, err = reader.ReadU64(); err != nil {
			return nil, err
		}
	}

	if !allDefault && !parent.XybEncoded {
		if fh.DoYCbCr, err = reader.ReadBool(); err != nil {
			return nil, err
		}
	} else {
		fh.DoYCbCr = false
	}
	//fh.jpegUpsampling = make([]util.IntPoint, 3)
	fh.jpegUpsamplingX = make([]int32, 3)
	fh.jpegUpsamplingY = make([]int32, 3)
	if fh.DoYCbCr && (fh.Flags&USE_LF_FRAME) == 0 {
		for i := 0; i < 3; i++ {
			var mode uint64
			if mode, err = reader.ReadBits(2); err != nil {
				return nil, err
			}
			//y := reader.MustReadBits(1)
			//x := reader.MustReadBits(1)
			//fh.jpegUpsampling[i] = util.NewIntPointWithXY(uint32(x^y), uint32(y))
			switch mode {
			case 1:
				fh.jpegUpsamplingY[i] = 1
				fh.jpegUpsamplingX[i] = 1
				break
			case 2:
				fh.jpegUpsamplingY[i] = 0
				fh.jpegUpsamplingX[i] = 1
			case 3:
				fh.jpegUpsamplingY[i] = 1
				fh.jpegUpsamplingX[i] = 0
			default:
				break
			}
		}
	}

	fh.EcUpsampling = make([]uint32, len(parent.ExtraChannelInfo))
	if !allDefault && (fh.Flags&USE_LF_FRAME) == 0 {
		if upsampling, err := reader.ReadBits(2); err != nil {
			return nil, err
		} else {
			fh.Upsampling = 1 << upsampling
		}
		for i := 0; i < len(fh.EcUpsampling); i++ {
			if ecUpsampling, err := reader.ReadBits(2); err != nil {
				return nil, err
			} else {
				fh.EcUpsampling[i] = 1 << ecUpsampling
			}
		}
	} else {
		fh.Upsampling = 1
		//fh.EcUpsampling = []uint32{1}
		for i := 0; i < len(fh.EcUpsampling); i++ {
			fh.EcUpsampling[i] = 1
		}
	}

	if fh.Encoding == MODULAR {
		if groupSizeShift, err := reader.ReadBits(2); err != nil {
			return nil, err
		} else {
			fh.groupSizeShift = uint32(groupSizeShift)
		}
	} else {
		fh.groupSizeShift = 1
	}
	fh.groupDim = 128 << fh.groupSizeShift
	fh.lfGroupDim = fh.groupDim << 3
	fh.logGroupDim = uint32(util.CeilLog2(int64(fh.groupDim)))
	fh.logLFGroupDIM = uint32(util.CeilLog2(int64(fh.lfGroupDim)))
	if parent.XybEncoded && fh.Encoding == VARDCT {
		if !allDefault {
			// TODO(kpfaulkner) 20241026 getting 0's for xqmScale and bqmScale where as JXLatte gets 2 for both?!?
			// REALLY confused how this is happening...
			xqmScale, err := reader.ReadBits(3)
			if err != nil {
				return nil, err
			}
			fh.xqmScale = uint32(xqmScale)
			bqmScale, err := reader.ReadBits(3)
			if err != nil {
				return nil, err
			}
			fh.bqmScale = uint32(bqmScale)
		} else {
			fh.xqmScale = 3
			fh.bqmScale = 2
		}
	} else {
		fh.xqmScale = 2
		fh.bqmScale = 2
	}

	if !allDefault && fh.FrameType != REFERENCE_ONLY {
		fh.passes, err = NewPassesInfoWithReader(reader)
		if err != nil {
			return nil, err
		}
	} else {
		fh.passes = NewPassesInfo()
	}

	if fh.FrameType == LF_FRAME {
		if lfLevel, err := reader.ReadBits(2); err != nil {
			return nil, err
		} else {
			fh.LfLevel = uint32(lfLevel) + 1
		}
	} else {
		fh.LfLevel = 0
	}
	if !allDefault && fh.FrameType != LF_FRAME {
		if fh.haveCrop, err = reader.ReadBool(); err != nil {
			return nil, err
		}
	} else {
		fh.haveCrop = false
	}

	if fh.haveCrop && fh.FrameType != REFERENCE_ONLY {
		var err error
		var x0 uint32
		if x0, err = reader.ReadU32(0, 8, 256, 11, 2304, 14, 18688, 30); err != nil {
			return nil, err
		}

		var y0 uint32
		if y0, err = reader.ReadU32(0, 8, 256, 11, 2304, 14, 18688, 30); err != nil {
			return nil, err
		}

		x0Signed := jxlio.UnpackSigned(x0)
		y0Signed := jxlio.UnpackSigned(y0)
		if fh.Bounds == nil {
			fh.Bounds = &util.Rectangle{
				Origin: util.Point{},
				Size:   util.Dimension{},
			}
		}
		fh.Bounds.Origin.X = x0Signed
		fh.Bounds.Origin.Y = y0Signed
	}

	if fh.haveCrop {
		if width, err := reader.ReadU32(0, 8, 256, 11, 2304, 14, 18688, 30); err != nil {
			return nil, err
		} else {
			fh.Width = width
			fh.Bounds.Size.Width = width
		}

		if height, err := reader.ReadU32(0, 8, 256, 11, 2304, 14, 18688, 30); err != nil {
			return nil, err
		} else {
			fh.Height = height
			fh.Bounds.Size.Height = height
		}
	} else {
		if fh.Bounds == nil {
			fh.Bounds = &util.Rectangle{
				Origin: util.Point{X: 0, Y: 0},
				Size:   util.Dimension{},
			}
		}
		fh.Bounds.Size = parent.Size
	}

	normalFrame := !allDefault && (fh.FrameType == REGULAR_FRAME || fh.FrameType == SKIP_PROGRESSIVE)
	lowerCorner := fh.Bounds.ComputeLowerCorner()
	//fullFrame := fh.Bounds.Origin.X <= 0 && fh.Bounds.Origin.Y <= 0 &&
	//	(fh.Width+uint32(fh.Bounds.Origin.X) >= parent.size.Width && (fh.Height+uint32(fh.Bounds.Origin.Y) >= parent.size.Height))
	fullFrame := fh.Bounds.Origin.Y <= 0 && fh.Bounds.Origin.X <= 0 &&
		lowerCorner.Y >= int32(parent.Size.Height) && lowerCorner.X >= int32(parent.Size.Width)

	fh.Bounds.Size.Height = util.CeilDiv(fh.Bounds.Size.Height, fh.Upsampling)
	fh.Bounds.Size.Width = util.CeilDiv(fh.Bounds.Size.Width, fh.Upsampling)
	fh.Bounds.Size.Height = util.CeilDiv(fh.Bounds.Size.Height, 1<<(3*fh.LfLevel))
	fh.Bounds.Size.Width = util.CeilDiv(fh.Bounds.Size.Width, 1<<(3*fh.LfLevel))

	fh.EcBlendingInfo = make([]BlendingInfo, len(parent.ExtraChannelInfo))
	if normalFrame {
		fh.BlendingInfo, err = NewBlendingInfoWithReader(reader, len(fh.EcBlendingInfo) > 0, fullFrame)
		if err != nil {
			return nil, err
		}
		for i := 0; i < len(fh.EcBlendingInfo); i++ {
			bi, err := NewBlendingInfoWithReader(reader, true, fullFrame)
			if err != nil {
				return nil, err
			}
			// store value not pointer. TODO(kpfaulkner) check this is fine.
			fh.EcBlendingInfo[i] = *bi
		}
	} else {
		fh.BlendingInfo = NewBlendingInfo()
		for i := 0; i < len(fh.EcBlendingInfo); i++ {
			fh.EcBlendingInfo[i] = *fh.BlendingInfo
		}
	}

	if normalFrame && parent.AnimationHeader != nil {
		// dont care about animation
		panic("animation")
		dur, err := reader.ReadU32(0, 0, 1, 0, 0, 8, 0, 32)
		if err != nil {
			return nil, err
		}
		fh.Duration = dur
	} else {
		fh.Duration = 0
	}
	if normalFrame && parent.AnimationHeader != nil && parent.AnimationHeader.HaveTimeCodes {
		// dont care about animation
		tc, err := reader.ReadBits(32)
		if err != nil {
			return nil, err
		}
		fh.timecode = uint32(tc)
	} else {
		fh.timecode = 0
	}

	if normalFrame {
		if fh.IsLast, err = reader.ReadBool(); err != nil {
			return nil, err
		}
	} else {
		fh.IsLast = fh.FrameType == REGULAR_FRAME
	}

	if !allDefault && fh.FrameType != LF_FRAME && !fh.IsLast {
		if saveAsReference, err := reader.ReadBits(2); err != nil {
			return nil, err
		} else {
			fh.SaveAsReference = uint32(saveAsReference)
		}
	} else {
		fh.SaveAsReference = 0
	}

	if !allDefault && (fh.FrameType == REFERENCE_ONLY || fullFrame &&
		(fh.FrameType == REGULAR_FRAME || fh.FrameType == SKIP_PROGRESSIVE) &&
		(fh.Duration == 0 || fh.SaveAsReference != 0) &&
		!fh.IsLast && fh.BlendingInfo.Mode == BLEND_REPLACE) {

		if fh.SaveBeforeCT, err = reader.ReadBool(); err != nil {
			return nil, err
		}
	} else {
		fh.SaveBeforeCT = false
	}

	if allDefault {
		fh.name = ""
	} else {
		var nameLen uint32
		if nameLen, err = reader.ReadU32(0, 0, 0, 4, 16, 5, 48, 10); err != nil {
			return nil, err
		}
		buffer := make([]byte, nameLen)
		for i := 0; i < int(nameLen); i++ {
			buffer[i], err = reader.ReadByte()
			if err != nil {
				return nil, err
			}
		}
		fh.name = string(buffer)
	}
	if allDefault {
		fh.restorationFilter = NewRestorationFilter()
	} else {
		fh.restorationFilter, err = NewRestorationFilterWithReader(reader, fh.Encoding)
		if err != nil {
			return nil, err
		}
	}

	if allDefault {
		fh.extensions = bundle.NewExtensions()
	} else {
		fh.extensions, err = bundle.NewExtensionsWithReader(reader)
		if err != nil {
			return nil, err
		}
	}

	maxJPY := util.Max(fh.jpegUpsamplingY...)
	maxJPX := util.Max(fh.jpegUpsamplingX...)
	fh.Bounds.Size.Height = util.CeilDiv(fh.Bounds.Size.Height, 1<<maxJPY) << maxJPY
	fh.Bounds.Size.Width = util.CeilDiv(fh.Bounds.Size.Width, 1<<maxJPX) << maxJPX

	for i := 0; i < 3; i++ {
		fh.jpegUpsamplingY[i] = maxJPY - fh.jpegUpsamplingY[i]
		fh.jpegUpsamplingX[i] = maxJPX - fh.jpegUpsamplingX[i]
	}

	return fh, nil
}
