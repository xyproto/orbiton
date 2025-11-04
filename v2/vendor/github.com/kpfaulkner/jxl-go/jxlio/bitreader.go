package jxlio

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// BitReader is interface to BitStreamReader... am concerned (without measurements)
// that interface calling might have impact. But will convert to interface and measure
// later.
type BitReader interface {
	Seek(offset int64, whence int) (int64, error)
	Reset() error
	AtEnd() bool
	ReadBytesToBuffer(buffer []uint8, numBytes uint32) error
	ReadBits(bits uint32) (uint64, error)
	ReadByteArrayWithOffsetAndLength(buffer []byte, offset int64, length uint32) error
	ReadByte() (uint8, error)
	ReadEnum() (int32, error)
	ReadF16() (float32, error)
	ReadICCVarint() (int32, error)
	ReadU32(c0 int, u0 int, c1 int, u1 int, c2 int, u2 int, c3 int, u3 int) (uint32, error)
	ReadBool() (bool, error)
	ReadU64() (uint64, error)
	ReadU8() (int, error)
	GetBitsCount() uint64
	ShowBits(bits int) (uint64, error)
	SkipBits(bits uint32) error
	Skip(bytes uint32) (int64, error)
	ReadBytesUint64(noBytes int) (uint64, error)
	ZeroPadToByte() error
	BitsRead() uint64
}

// BitStreamReader is the key struct for reading bits from a byte "stream".
type BitStreamReader struct {
	//buffer []byte
	// stream/reader we're using most of the time
	stream      io.ReadSeeker
	bitsRead    uint64
	tempIndex   int
	index       uint8
	currentByte uint8
}

func NewBitStreamReaderWithIndex(in io.ReadSeeker, index int) *BitStreamReader {

	br := NewBitStreamReader(in)
	br.tempIndex = index
	//br.buffer = make([]byte, 1)
	return br
}

func NewBitStreamReader(in io.ReadSeeker) *BitStreamReader {

	br := &BitStreamReader{}
	//br.buffer = make([]byte, 1)
	br.stream = in
	return br
}

// utter hack to seek about the place. TODO(kpfaulkner) confirm this really works.
func (br *BitStreamReader) Seek(offset int64, whence int) (int64, error) {
	n, err := br.stream.Seek(offset, whence)
	if err != nil {
		return 0, err
	}
	return n, err
}

func (br *BitStreamReader) Reset() error {

	_, err := br.stream.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	// reset tracking
	br.index = 0
	br.currentByte = 0
	return nil
}

func (br *BitStreamReader) AtEnd() bool {

	_, err := br.ShowBits(1)
	if err != nil {
		return true
	}
	return false
}

// ReadBytesToBuffer
// If part way through a byte then fail. Need to be aligned for this to work.
func (br *BitStreamReader) ReadBytesToBuffer(buffer []uint8, numBytes uint32) error {

	if br.index != 0 {
		return errors.New("BitStreamReader cache not aligned")
	}

	n, err := br.stream.Read(buffer[:numBytes])
	if err != nil {
		return err
	}

	if n != int(numBytes) {
		return errors.New("unable to read all bytes")
	}
	return nil
}

// read single bit and will cache the current byte we're working on.
func (br *BitStreamReader) readBit() (uint8, error) {
	if br.index == 0 {
		buffer := make([]byte, 1)
		_, err := br.stream.Read(buffer)
		if err != nil {
			return 0, err
		}
		br.currentByte = buffer[0]
	}

	v := (br.currentByte & (1 << br.index)) != 0
	br.index = (br.index + 1) % 8

	br.bitsRead++
	if v {
		return 1, nil
	} else {
		return 0, nil
	}
}

func (br *BitStreamReader) ReadBits(bits uint32) (uint64, error) {

	if bits == 0 {
		return 0, nil
	}

	if bits < 1 || bits > 64 {

		return 0, errors.New("num bits must be between 1 and 64")
	}
	var v uint64
	for i := uint32(0); i < bits; i++ {
		bit, err := br.readBit()
		if err != nil {
			return 0, err
		}
		v |= uint64(bit) << i

	}
	return v, nil
}

func (br *BitStreamReader) ReadByteArrayWithOffsetAndLength(buffer []byte, offset int64, length uint32) error {
	if length == 0 {
		return nil
	}

	_, err := br.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}

	err = br.ReadBytesToBuffer(buffer, length)
	if err != nil {
		return err
	}
	return nil
}

func (br *BitStreamReader) ReadByte() (uint8, error) {
	v, err := br.ReadBits(8)
	if err != nil {
		return 0, err
	}
	return uint8(v), nil
}

func (br *BitStreamReader) ReadEnum() (int32, error) {
	constant, err := br.ReadU32(0, 0, 1, 0, 2, 4, 18, 6)
	if err != nil {
		return 0, err
	}
	if constant > 63 {
		return 0, errors.New("enum constant > 63")
	}
	return int32(constant), nil
}

func (br *BitStreamReader) ReadF16() (float32, error) {
	bits16, err := br.ReadBits(16)
	if err != nil {
		return 0, err
	}

	mantissa := bits16 & 0x3FF
	biased_exp := (uint32(bits16) >> 10) & 0x1F
	sign := (bits16 >> 15) & 1
	if biased_exp == 31 {
		return 0, errors.New("illegal infinite/NaN float16")
	}

	if biased_exp == 0 {
		return (1.0 - 2.0*float32(sign)) * float32(mantissa) / 16777216.0, nil
	}

	biased_exp += 127 - 15
	mantissa = mantissa << 13
	sign = sign << 31

	total := uint32(sign) | biased_exp<<23 | uint32(mantissa)
	return math.Float32frombits(total), nil
}

func (br *BitStreamReader) ReadICCVarint() (int32, error) {
	value := int32(0)
	for shift := 0; shift < 63; shift += 7 {
		b, err := br.ReadBits(8)
		if err != nil {
			return 0, err
		}
		value |= int32(b) & 127 << shift
		if b <= 127 {
			break
		}
	}
	if value > math.MaxInt32 {
		return 0, errors.New("ICC varint overflow")

	}
	return value, nil
}

func (br *BitStreamReader) ReadU32(c0 int, u0 int, c1 int, u1 int, c2 int, u2 int, c3 int, u3 int) (uint32, error) {
	choice, err := br.ReadBits(2)
	if err != nil {
		return 0, err
	}

	c := []int{c0, c1, c2, c3}
	u := []int{u0, u1, u2, u3}
	b, err := br.ReadBits(uint32(u[choice]))
	if err != nil {
		return 0, err
	}
	return uint32(c[choice]) + uint32(b), nil
}

func (br *BitStreamReader) ReadBool() (bool, error) {
	v, err := br.readBit()
	if err != nil {
		return false, err
	}
	return v == 1, nil
}

func (br *BitStreamReader) ReadU64() (uint64, error) {
	index, err := br.ReadBits(2)
	if err != nil {
		return 0, err
	}

	if index == 0 {
		return 0, nil
	}

	if index == 1 {
		b, err := br.ReadBits(4)
		if err != nil {
			return 0, err
		}
		return 1 + uint64(b), nil
	}

	if index == 2 {
		b, err := br.ReadBits(8)
		if err != nil {
			return 0, err
		}
		return 17 + uint64(b), nil
	}

	value2, err := br.ReadBits(12)
	if err != nil {
		return 0, err
	}
	value := uint64(value2)

	shift := 12
	var boolCheck bool
	for {
		if boolCheck, err = br.ReadBool(); err != nil {
			return 0, err
		}
		if !boolCheck {
			break
		}
		if shift == 60 {

			if data, err := br.ReadBits(4); err != nil {
				return 0, err
			} else {
				value |= data << shift
			}
			break
		}
		if data, err := br.ReadBits(8); err != nil {
			return 0, err
		} else {
			value |= data << shift
		}
		shift += 8
	}
	return value, nil
}

func (br *BitStreamReader) ReadU8() (int, error) {

	b, err := br.ReadBool()
	if err != nil {
		return 0, err
	}

	if !b {
		return 0, nil
	}
	n, err := br.ReadBits(3)
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 1, nil
	}

	nn, err := br.ReadBits(uint32(n))
	if err != nil {
		return 0, err
	}
	return int(nn + 1<<n), nil
}

func (br *BitStreamReader) GetBitsCount() uint64 {
	return br.bitsRead
}

func (br *BitStreamReader) ShowBits(bits int) (uint64, error) {

	curPos, err := br.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	oldCur := br.currentByte
	oldIndex := br.index
	oldBitsRead := br.bitsRead

	b, err := br.ReadBits(uint32(bits))
	if err != nil {
		return 0, err
	}

	_, err = br.Seek(curPos, io.SeekStart)
	if err != nil {
		return 0, err
	}
	br.currentByte = oldCur
	br.index = oldIndex
	br.bitsRead = oldBitsRead

	return b, nil
}

func (br *BitStreamReader) SkipBits(bits uint32) error {

	numBytes := bits / 8
	if numBytes > 0 {
		buffer := make([]byte, numBytes)
		_, err := br.stream.Read(buffer)
		if err != nil {
			return err
		}
		br.currentByte = buffer[numBytes-1]
	}

	// read bits so we can keep track of where we are.
	for i := numBytes * 8; i < bits; i++ {
		_, err := br.readBit()
		if err != nil {
			return err
		}
	}
	return nil
}

func (br *BitStreamReader) Skip(bytes uint32) (int64, error) {
	err := br.SkipBits(bytes << 3)
	if err != nil {
		return 0, err
	}
	return int64(bytes), nil
}

func (br *BitStreamReader) ReadBytesUint64(noBytes int) (uint64, error) {
	if noBytes < 1 || noBytes > 8 {
		return 0, fmt.Errorf("number of bytes number should be between 1 and 8.")
	}

	ba := make([]byte, 8)
	err := br.ReadBytesToBuffer(ba, uint32(noBytes))
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(ba), nil
}

func (br *BitStreamReader) ZeroPadToByte() error {

	if br.index == 0 {
		return nil
	}
	remaining := 8 - br.index
	if remaining > 0 {
		_, err := br.ReadBits(uint32(remaining))
		if err != nil {
			return err
		}
	}
	return nil
}

func (br *BitStreamReader) BitsRead() uint64 {
	return br.bitsRead
}

// JPEGXL spec states unpackedsigned is
// equivalent to u / 2 if u is even, and -(u + 1) / 2 if u is odd
func UnpackSigned(value uint32) int32 {
	if value&1 == 0 {
		return int32(value >> 1)
	}

	return -(int32(value) + 1) >> 1
}

func UnpackSigned64(value uint64) int64 {
	if value&1 == 0 {
		return int64(value >> 1)
	}

	return -(int64(value) + 1) >> 1
}
