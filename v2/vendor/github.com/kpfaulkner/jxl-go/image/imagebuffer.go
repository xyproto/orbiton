package image

import (
	"errors"

	"github.com/kpfaulkner/jxl-go/util"
)

const (
	TYPE_INT   = 0
	TYPE_FLOAT = 1
)

type ImageBuffer struct {
	Width      int32
	Height     int32
	BufferType int

	// image data can be either float or int based. Keep separate buffers and just
	// reference each one as required. If conversion will be required then that might get
	// expensive, but will optimise/revisit later.
	FloatBuffer [][]float32
	IntBuffer   [][]int32
}

func NewImageBuffer(bufferType int, height int32, width int32) (*ImageBuffer, error) {

	if bufferType != TYPE_INT && bufferType != TYPE_FLOAT {
		return nil, errors.New("Invalid buffer type")
	}
	if height < 0 || height > (1<<30) || width < 0 || width > (1<<30) {
		return nil, errors.New("Invalid height/width")
	}

	ib := &ImageBuffer{
		Width:      width,
		Height:     height,
		BufferType: bufferType,
	}

	if bufferType == TYPE_INT {
		ib.IntBuffer = util.MakeMatrix2D[int32](height, width)
	} else {
		ib.FloatBuffer = util.MakeMatrix2D[float32](height, width)
	}
	return ib, nil
}

func NewImageBufferFromInts(buffer [][]int32) *ImageBuffer {
	ib := &ImageBuffer{}
	ib.IntBuffer = buffer
	ib.BufferType = TYPE_INT
	ib.Height = int32(len(buffer))
	ib.Width = int32(len(buffer[0]))
	return ib
}

func NewImageBufferFromFloats(buffer [][]float32) *ImageBuffer {
	ib := &ImageBuffer{}
	ib.FloatBuffer = buffer
	ib.BufferType = TYPE_FLOAT
	ib.Height = int32(len(buffer))
	ib.Width = int32(len(buffer[0]))
	return ib
}

func NewImageBufferFromImageBuffer(imageBuffer *ImageBuffer) *ImageBuffer {
	ib := &ImageBuffer{}
	ib.IntBuffer = copyInt32Matrix2D(imageBuffer.IntBuffer)
	ib.FloatBuffer = copyFloat32Matrix2D(imageBuffer.FloatBuffer)
	ib.BufferType = imageBuffer.BufferType
	ib.Height = imageBuffer.Height
	ib.Width = imageBuffer.Width
	return ib
}

func copyInt32Matrix2D(src [][]int32) [][]int32 {
	duplicate := make([][]int32, len(src))
	for i := range src {
		duplicate[i] = make([]int32, len(src[i]))
		copy(duplicate[i], src[i])
	}
	return duplicate
}

func copyFloat32Matrix2D(src [][]float32) [][]float32 {
	duplicate := make([][]float32, len(src))
	for i := range src {
		duplicate[i] = make([]float32, len(src[i]))
		copy(duplicate[i], src[i])
	}
	return duplicate
}

// Equals compares two ImageBuffers and returns true if they are equal.
func (ib *ImageBuffer) Equals(other ImageBuffer) bool {

	if ib.Width != other.Width || ib.Height != other.Height {
		return false
	}

	if ib.BufferType != other.BufferType {
		return false
	}

	if ib.BufferType == TYPE_INT {
		return util.CompareMatrix2D(ib.IntBuffer, other.IntBuffer, func(aa int32, bb int32) bool {
			return aa == bb
		})
	}

	if ib.BufferType == TYPE_FLOAT {
		return util.CompareMatrix2D(ib.FloatBuffer, other.FloatBuffer, func(aa float32, bb float32) bool {
			return aa == bb
		})
	}

	return false
}

func (ib *ImageBuffer) IsFloat() bool {
	return ib.BufferType == TYPE_FLOAT
}

func (ib *ImageBuffer) IsInt() bool {
	return ib.BufferType == TYPE_INT
}

func (ib *ImageBuffer) CastToFloatIfInt(maxValue int32) error {
	if ib.BufferType == TYPE_FLOAT {
		return nil
	}
	return ib.castToFloatBuffer(maxValue)
}

func (ib *ImageBuffer) castToFloatBuffer(maxValue int32) error {

	if ib.BufferType == TYPE_FLOAT {
		return errors.New("Already a float buffer")
	}
	if maxValue < 1 {
		return errors.New("Invalid maxValue")
	}
	oldBuffer := ib.IntBuffer
	newBuffer := util.MakeMatrix2D[float32](ib.Height, ib.Width)
	scaleFactor := 1.0 / float32(maxValue)
	for y := 0; y < int(ib.Height); y++ {
		for x := 0; x < int(ib.Width); x++ {
			newBuffer[y][x] = float32(oldBuffer[y][x]) * scaleFactor
		}
	}
	ib.BufferType = TYPE_FLOAT
	ib.FloatBuffer = newBuffer
	return nil
}

func (ib *ImageBuffer) CastToIntIfFloat(maxValue int32) error {
	if ib.BufferType == TYPE_INT {
		return nil
	}

	err := ib.castToIntBuffer(maxValue)
	ib.BufferType = TYPE_INT
	return err
}

func (ib *ImageBuffer) castToIntBuffer(maxValue int32) error {

	if ib.BufferType == TYPE_INT {
		return errors.New("Already a int buffer")
	}
	if maxValue < 1 {
		return errors.New("Invalid maxValue")
	}

	oldBuffer := ib.FloatBuffer
	newBuffer := util.MakeMatrix2D[int32](ib.Height, ib.Width)
	scaleFactor := float32(maxValue)
	for y := 0; y < int(ib.Height); y++ {
		for x := 0; x < int(ib.Width); x++ {
			v := int32(oldBuffer[y][x]*scaleFactor + 0.5)
			var vv int32
			if v < 0 {
				vv = 0
			} else if v > maxValue {
				vv = maxValue
			} else {
				vv = v
			}
			newBuffer[y][x] = vv
		}
	}
	ib.IntBuffer = newBuffer
	ib.BufferType = TYPE_INT
	return nil
}

func (ib *ImageBuffer) Clamp(maxValue int32) error {
	if ib.IsFloat() {
		return errors.New("Clamp only supported for int buffers")
	}

	buf := util.MakeMatrix2D[int32](ib.Height, ib.Width)
	for y := 0; y < int(ib.Height); y++ {
		for x := 0; x < int(ib.Width); x++ {
			v := ib.IntBuffer[y][x]
			if v < 0 {
				buf[y][x] = 0
			} else if v > maxValue {
				buf[y][x] = maxValue
			} else {
				buf[y][x] = v
			}
		}
	}
	ib.IntBuffer = buf
	return nil
}

// Equals compares two ImageBuffer slices and returns true if they are equal.
// Need to have:
//   - same size
//   - expected that in same order.
//   - each ImageBuffer has same buffer type
//   - each ImageBuffer has same width/height
//   - each ImageBuffer has same buffer values
func ImageBufferSliceEquals(a []ImageBuffer, b []ImageBuffer) bool {

	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if !a[i].Equals(b[i]) {
			return false
		}
	}
	return true
}
