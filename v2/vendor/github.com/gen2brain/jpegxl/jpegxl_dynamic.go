//go:build (unix || darwin || windows) && !nodynamic

package jpegxl

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
)

func decodeDynamic(r io.Reader, configOnly, decodeAll bool) (*JXL, image.Config, error) {
	var err error
	var cfg image.Config
	var data []byte

	decoder := jxlDecoderCreate()
	defer jxlDecoderDestroy(decoder)

	if !jxlDecoderSubscribeEvents(decoder, jxlDecBasicInfo|jxlDecFrame|jxlDecFullImage) {
		return nil, cfg, ErrDecode
	}

	var info jxlBasicInfo
	var header jxlFrameHeader

	var format jxlPixelFormat
	format.NumChannels = 4
	format.DataType = jxlTypeUint8
	format.Endianness = jxlNativeEndian

	data, err = io.ReadAll(r)
	if err != nil {
		return nil, cfg, fmt.Errorf("read: %w", err)
	}

	jxlDecoderSetInput(decoder, data)
	jxlDecoderCloseInput(decoder)

	delay := make([]int, 0)
	images := make([]image.Image, 0)

	for {
		status := jxlDecoderProcessInput(decoder)

		switch status {
		case jxlDecError:
			return nil, cfg, ErrDecode
		case jxlDecNeedMoreInput:
			return nil, cfg, ErrDecode
		case jxlDecBasicInfo:
			if !jxlDecoderGetBasicInfo(decoder, &info) {
				return nil, cfg, ErrDecode
			}

			cfg.Width = int(info.Xsize)
			cfg.Height = int(info.Ysize)
			cfg.ColorModel = color.NRGBAModel

			if configOnly && info.HaveAnimation == 0 {
				return nil, cfg, nil
			}

			if info.BitsPerSample == 16 {
				format.DataType = jxlTypeUint16
				format.Endianness = jxlBigEndian
			}
		case jxlDecFrame:
			if !jxlDecoderGetFrameHeader(decoder, &header) {
				return nil, cfg, ErrDecode
			}

			delay = append(delay, int(header.Duration))
		case jxlDecNeedImageOutBuffer:
			if configOnly {
				jxlDecoderSkipCurrentFrame(decoder)

				continue
			}

			var bufSize uint64
			if !jxlDecoderImageOutBufferSize(decoder, &format, &bufSize) {
				return nil, cfg, ErrDecode
			}

			if info.BitsPerSample == 16 {
				img := image.NewNRGBA64(image.Rect(0, 0, cfg.Width, cfg.Height))
				images = append(images, img)

				if !jxlDecoderSetImageOutBuffer(decoder, &format, img.Pix, bufSize) {
					return nil, cfg, ErrDecode
				}
			} else {
				img := image.NewNRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))
				images = append(images, img)

				if !jxlDecoderSetImageOutBuffer(decoder, &format, img.Pix, bufSize) {
					return nil, cfg, ErrDecode
				}
			}
		case jxlDecFullImage:
			if !decodeAll || (info.HaveAnimation == 1 && header.IsLast == 1) {
				ret := &JXL{
					Image: images,
					Delay: delay,
				}

				return ret, cfg, nil
			}
		case jxlDecSuccess:
			runtime.KeepAlive(data)

			ret := &JXL{
				Image: images,
				Delay: delay,
			}

			return ret, cfg, nil
		}
	}
}

func encodeDynamic(w io.Writer, m image.Image, quality, effort int) error {
	img := imageToNRGBA(m)

	encoder := jxlEncoderCreate()
	defer jxlEncoderDestroy(encoder)

	var format jxlPixelFormat
	format.NumChannels = 4
	format.DataType = jxlTypeUint8
	format.Endianness = jxlNativeEndian

	var info jxlBasicInfo
	jxlEncoderInitBasicInfo(&info)
	info.Xsize = uint32(img.Bounds().Dx())
	info.Ysize = uint32(img.Bounds().Dy())
	info.BitsPerSample = 8
	info.AlphaBits = 8
	info.NumExtraChannels = 1

	if quality == 100 {
		info.UsesOriginalProfile = 1
	}

	if !jxlEncoderSetBasicInfo(encoder, &info) {
		return ErrEncode
	}

	var encoding jxlColorEncoding
	jxlColorEncodingSetToSRGB(&encoding, false)

	if !jxlEncoderSetColorEncoding(encoder, &encoding) {
		return ErrEncode
	}

	settings := jxlEncoderFrameSettingsCreate(encoder)
	jxlEncoderSetFrameDistance(settings, jxlEncoderDistanceFromQuality(quality))
	jxlEncoderFrameSettingsSetOption(settings, jxlEncFrameSettingEffort, effort)
	if quality == 100 {
		jxlEncoderSetFrameLossless(settings, true)
	}

	if !jxlEncoderAddImageFrame(settings, &format, img.Pix) {
		return ErrEncode
	}

	jxlEncoderCloseInput(encoder)

	bufSize := 4096
	buf := make([]byte, bufSize)
	out := make([]byte, 0)

	for {
		available := uint64(bufSize)
		status := jxlEncoderProcessOutput(encoder, &buf[0], &available)
		if status == jxlEncError {
			return ErrEncode
		}

		if status == jxlEncNeedMoreOutput {
			out = append(out, buf...)
			bufSize *= 2
			buf = make([]byte, bufSize)
		}

		if status == jxlEncSuccess {
			out = append(out, buf[:bufSize-int(available)]...)
			break
		}
	}

	_, err := w.Write(out)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

func init() {
	var err error
	defer func() {
		if r := recover(); r != nil {
			dynamic = false
			dynamicErr = fmt.Errorf("%v", r)
		}
	}()

	libjxl, err = loadLibrary()
	if err == nil {
		dynamic = true
	} else {
		dynamicErr = err

		return
	}

	purego.RegisterLibFunc(&_jxlDecoderCreate, libjxl, "JxlDecoderCreate")
	purego.RegisterLibFunc(&_jxlDecoderDestroy, libjxl, "JxlDecoderDestroy")
	purego.RegisterLibFunc(&_jxlDecoderSubscribeEvents, libjxl, "JxlDecoderSubscribeEvents")
	purego.RegisterLibFunc(&_jxlDecoderSetInput, libjxl, "JxlDecoderSetInput")
	purego.RegisterLibFunc(&_jxlDecoderCloseInput, libjxl, "JxlDecoderCloseInput")
	purego.RegisterLibFunc(&_jxlDecoderProcessInput, libjxl, "JxlDecoderProcessInput")
	purego.RegisterLibFunc(&_jxlDecoderGetBasicInfo, libjxl, "JxlDecoderGetBasicInfo")
	purego.RegisterLibFunc(&_jxlDecoderGetFrameHeader, libjxl, "JxlDecoderGetFrameHeader")
	purego.RegisterLibFunc(&_jxlDecoderSkipCurrentFrame, libjxl, "JxlDecoderSkipCurrentFrame")
	purego.RegisterLibFunc(&_jxlDecoderImageOutBufferSize, libjxl, "JxlDecoderImageOutBufferSize")
	purego.RegisterLibFunc(&_jxlDecoderSetImageOutBuffer, libjxl, "JxlDecoderSetImageOutBuffer")
	purego.RegisterLibFunc(&_jxlEncoderCreate, libjxl, "JxlEncoderCreate")
	purego.RegisterLibFunc(&_jxlEncoderInitBasicInfo, libjxl, "JxlEncoderInitBasicInfo")
	purego.RegisterLibFunc(&_jxlEncoderSetBasicInfo, libjxl, "JxlEncoderSetBasicInfo")
	purego.RegisterLibFunc(&_jxlEncoderDestroy, libjxl, "JxlEncoderDestroy")
	purego.RegisterLibFunc(&_jxlColorEncodingSetToSRGB, libjxl, "JxlColorEncodingSetToSRGB")
	purego.RegisterLibFunc(&_jxlEncoderCloseInput, libjxl, "JxlEncoderCloseInput")
	purego.RegisterLibFunc(&_jxlEncoderSetFrameDistance, libjxl, "JxlEncoderSetFrameDistance")
	purego.RegisterLibFunc(&_jxlEncoderSetFrameLossless, libjxl, "JxlEncoderSetFrameLossless")
	purego.RegisterLibFunc(&_jxlEncoderSetColorEncoding, libjxl, "JxlEncoderSetColorEncoding")
	purego.RegisterLibFunc(&_jxlEncoderFrameSettingsCreate, libjxl, "JxlEncoderFrameSettingsCreate")
	purego.RegisterLibFunc(&_jxlEncoderFrameSettingsSetOption, libjxl, "JxlEncoderFrameSettingsSetOption")
	purego.RegisterLibFunc(&_jxlEncoderAddImageFrame, libjxl, "JxlEncoderAddImageFrame")
	purego.RegisterLibFunc(&_jxlEncoderProcessOutput, libjxl, "JxlEncoderProcessOutput")
	purego.RegisterLibFunc(&_jxlEncoderDistanceFromQuality, libjxl, "JxlEncoderDistanceFromQuality")
}

var (
	libjxl     uintptr
	dynamic    bool
	dynamicErr error
)

const (
	jxlDecSuccess            = 0
	jxlDecError              = 1
	jxlDecNeedMoreInput      = 2
	jxlDecNeedImageOutBuffer = 5
	jxlDecBasicInfo          = 0x40
	jxlDecFrame              = 0x400
	jxlDecFullImage          = 0x1000

	jxlEncSuccess        = 0
	jxlEncError          = 1
	jxlEncNeedMoreOutput = 2

	jxlEncFrameSettingEffort = 0

	jxlTypeUint8  = 2
	jxlTypeUint16 = 3

	jxlNativeEndian = 0
	jxlBigEndian    = 2
)

var (
	_jxlDecoderCreate                 func(uintptr) *jxlDecoder
	_jxlDecoderDestroy                func(*jxlDecoder)
	_jxlDecoderSubscribeEvents        func(*jxlDecoder, int32) int
	_jxlDecoderSetInput               func(*jxlDecoder, *uint8, uint64) int
	_jxlDecoderCloseInput             func(*jxlDecoder)
	_jxlDecoderProcessInput           func(*jxlDecoder) int
	_jxlDecoderGetBasicInfo           func(*jxlDecoder, *jxlBasicInfo) int
	_jxlDecoderGetFrameHeader         func(*jxlDecoder, *jxlFrameHeader) int
	_jxlDecoderSkipCurrentFrame       func(*jxlDecoder)
	_jxlDecoderImageOutBufferSize     func(*jxlDecoder, *jxlPixelFormat, *uint64) int
	_jxlDecoderSetImageOutBuffer      func(*jxlDecoder, *jxlPixelFormat, *uint8, uint64) int
	_jxlEncoderCreate                 func(uintptr) *jxlEncoder
	_jxlEncoderDestroy                func(*jxlEncoder)
	_jxlEncoderInitBasicInfo          func(*jxlBasicInfo)
	_jxlEncoderSetBasicInfo           func(*jxlEncoder, *jxlBasicInfo) int
	_jxlColorEncodingSetToSRGB        func(*jxlColorEncoding, int)
	_jxlEncoderCloseInput             func(*jxlEncoder)
	_jxlEncoderSetFrameDistance       func(*jxlEncoderFrameSettings, float32)
	_jxlEncoderSetFrameLossless       func(*jxlEncoderFrameSettings, int)
	_jxlEncoderSetColorEncoding       func(*jxlEncoder, *jxlColorEncoding) int
	_jxlEncoderFrameSettingsCreate    func(*jxlEncoder, uintptr) *jxlEncoderFrameSettings
	_jxlEncoderFrameSettingsSetOption func(*jxlEncoderFrameSettings, int, int64)
	_jxlEncoderAddImageFrame          func(*jxlEncoderFrameSettings, *jxlPixelFormat, *uint8, int) int
	_jxlEncoderProcessOutput          func(*jxlEncoder, **uint8, *uint64) int
	_jxlEncoderDistanceFromQuality    func(float32) float32
)

func jxlDecoderCreate() *jxlDecoder {
	return _jxlDecoderCreate(0)
}

func jxlDecoderDestroy(decoder *jxlDecoder) {
	_jxlDecoderDestroy(decoder)
}

func jxlDecoderSubscribeEvents(decoder *jxlDecoder, wanted int) bool {
	ret := _jxlDecoderSubscribeEvents(decoder, int32(wanted))

	return ret == 0
}

func jxlDecoderSetInput(decoder *jxlDecoder, data []byte) bool {
	ret := _jxlDecoderSetInput(decoder, unsafe.SliceData(data), uint64(len(data)))

	return ret == 0
}

func jxlDecoderCloseInput(decoder *jxlDecoder) {
	_jxlDecoderCloseInput(decoder)
}

func jxlDecoderProcessInput(decoder *jxlDecoder) int {
	ret := _jxlDecoderProcessInput(decoder)

	return ret
}

func jxlDecoderGetBasicInfo(decoder *jxlDecoder, info *jxlBasicInfo) bool {
	ret := _jxlDecoderGetBasicInfo(decoder, info)

	return ret == 0
}

func jxlDecoderGetFrameHeader(decoder *jxlDecoder, header *jxlFrameHeader) bool {
	ret := _jxlDecoderGetFrameHeader(decoder, header)

	return ret == 0
}

func jxlDecoderSkipCurrentFrame(decoder *jxlDecoder) {
	_jxlDecoderSkipCurrentFrame(decoder)
}

func jxlDecoderImageOutBufferSize(decoder *jxlDecoder, format *jxlPixelFormat, size *uint64) bool {
	ret := _jxlDecoderImageOutBufferSize(decoder, format, size)

	return ret == 0
}

func jxlDecoderSetImageOutBuffer(decoder *jxlDecoder, format *jxlPixelFormat, buffer []byte, size uint64) bool {
	ret := _jxlDecoderSetImageOutBuffer(decoder, format, unsafe.SliceData(buffer), size)

	return ret == 0
}

func jxlEncoderCreate() *jxlEncoder {
	return _jxlEncoderCreate(0)
}

func jxlEncoderDestroy(encoder *jxlEncoder) {
	_jxlEncoderDestroy(encoder)
}

func jxlEncoderInitBasicInfo(info *jxlBasicInfo) {
	_jxlEncoderInitBasicInfo(info)
}

func jxlEncoderSetBasicInfo(encoder *jxlEncoder, info *jxlBasicInfo) bool {
	ret := _jxlEncoderSetBasicInfo(encoder, info)

	return ret == 0
}

func jxlColorEncodingSetToSRGB(encoding *jxlColorEncoding, isGray bool) {
	enable := 0
	if isGray {
		enable = 1
	}

	_jxlColorEncodingSetToSRGB(encoding, enable)
}

func jxlEncoderCloseInput(encoder *jxlEncoder) {
	_jxlEncoderCloseInput(encoder)
}

func jxlEncoderSetFrameDistance(settings *jxlEncoderFrameSettings, quality float32) {
	_jxlEncoderSetFrameDistance(settings, quality)
}

func jxlEncoderSetFrameLossless(settings *jxlEncoderFrameSettings, lossless bool) {
	enable := 0
	if lossless {
		enable = 1
	}

	_jxlEncoderSetFrameLossless(settings, enable)
}

func jxlEncoderSetColorEncoding(encoder *jxlEncoder, encoding *jxlColorEncoding) bool {
	ret := _jxlEncoderSetColorEncoding(encoder, encoding)

	return ret == 0
}

func jxlEncoderFrameSettingsCreate(encoder *jxlEncoder) *jxlEncoderFrameSettings {
	return _jxlEncoderFrameSettingsCreate(encoder, 0)
}

func jxlEncoderFrameSettingsSetOption(settings *jxlEncoderFrameSettings, option, value int) {
	_jxlEncoderFrameSettingsSetOption(settings, option, int64(value))
}

func jxlEncoderAddImageFrame(settings *jxlEncoderFrameSettings, format *jxlPixelFormat, data []byte) bool {
	ret := _jxlEncoderAddImageFrame(settings, format, unsafe.SliceData(data), len(data))

	return ret == 0
}

func jxlEncoderProcessOutput(encoder *jxlEncoder, next *uint8, available *uint64) int {
	return _jxlEncoderProcessOutput(encoder, &next, available)
}

func jxlEncoderDistanceFromQuality(quality int) float32 {
	return _jxlEncoderDistanceFromQuality(float32(quality))
}

type jxlBasicInfo struct {
	HaveContainer         int32
	Xsize                 uint32
	Ysize                 uint32
	BitsPerSample         uint32
	ExponentBitsPerSample uint32
	IntensityTarget       float32
	MinNits               float32
	RelativeToMaxDisplay  int32
	LinearBelow           float32
	UsesOriginalProfile   int32
	HavePreview           int32
	HaveAnimation         int32
	Orientation           uint32
	NumColorChannels      uint32
	NumExtraChannels      uint32
	AlphaBits             uint32
	AlphaExponentBits     uint32
	AlphaPremultiplied    int32
	Preview               jxlPreviewHeader
	Animation             jxlAnimationHeader
	IntrinsicXsize        uint32
	IntrinsicYsize        uint32
	Padding               [100]uint8
}

type jxlFrameHeader struct {
	Duration   uint32
	Timecode   uint32
	NameLength uint32
	IsLast     int32
	LayerInfo  jxlLayerInfo
}

type jxlPixelFormat struct {
	NumChannels uint32
	DataType    uint32
	Endianness  uint32
	Align       uint64
}

type jxlAnimationHeader struct {
	TpsNumerator   uint32
	TpsDenominator uint32
	NumLoops       uint32
	HaveTimecodes  int32
}

type jxlPreviewHeader struct {
	Xsize uint32
	Ysize uint32
}

type jxlLayerInfo struct {
	HaveCrop        int32
	CropX0          int32
	CropY0          int32
	Xsize           uint32
	Ysize           uint32
	BlendInfo       jxlBlendInfo
	SaveAsReference uint32
}

type jxlBlendInfo struct {
	Blendmode uint32
	Source    uint32
	Alpha     uint32
	Clamp     int32
}

type jxlColorEncoding struct {
	ColorSpace       uint32
	WhitePoint       uint32
	WhitePointXy     [2]float64
	Primaries        uint32
	PrimariesRedXy   [2]float64
	PrimariesGreenXy [2]float64
	PrimariesBlueXy  [2]float64
	TransferFunction uint32
	Gamma            float64
	RenderingIntent  uint32
	_                [4]byte
}

type jxlDecoder struct{}
type jxlEncoder struct{}
type jxlEncoderFrameSettings struct{}
