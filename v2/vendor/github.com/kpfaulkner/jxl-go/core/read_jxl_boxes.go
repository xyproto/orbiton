package core

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"

	"github.com/kpfaulkner/jxl-go/jxlio"
	log "github.com/sirupsen/logrus"
)

var (
	JPEGXL_CONTAINER_HEADER = [12]byte{0x00, 0x00, 0x00, 0x0C, 0x4A, 0x58, 0x4C, 0x20, 0x0D, 0x0A, 0x87, 0x0A}

	JXLL = makeTag([]byte{'j', 'x', 'l', 'l'}, 0, 4)
	JXLP = makeTag([]byte{'j', 'x', 'l', 'p'}, 0, 4)
	JXLC = makeTag([]byte{'j', 'x', 'l', 'c'}, 0, 4)
)

type ContainerBoxHeader struct {
	BoxType   uint64
	BoxSize   uint64
	IsLast    bool
	Offset    int64 // offset compared to very beginning of file.
	Processed bool  // indicated if finished with.
}

type BoxReader struct {
	reader jxlio.BitReader
	level  int
}

func NewBoxReader(reader jxlio.BitReader) *BoxReader {
	return &BoxReader{
		reader: reader,
		level:  5,
	}
}

func (br *BoxReader) ReadBoxHeader() ([]ContainerBoxHeader, error) {
	buffer := make([]byte, 12)
	err := br.reader.ReadByteArrayWithOffsetAndLength(buffer, 0, 12)
	if err != nil {
		return nil, err
	}

	var containerBoxHeaders []ContainerBoxHeader

	// Believe this header is used when performing lossless decoding. Need to verify
	if !bytes.Equal(buffer, JPEGXL_CONTAINER_HEADER[:]) {
		log.Errorf("invalid magic number: %+v", buffer)
		// setup fake box header (if we dont have a container...?)
		bh := ContainerBoxHeader{
			BoxType: JXLC,
			BoxSize: 0,
			IsLast:  true,
			Offset:  0,
		}
		containerBoxHeaders = append(containerBoxHeaders, bh)

		// reset reader to beginning of data...  as we've read the first 12 bytes.
		//br.reader.Reset()
		return containerBoxHeaders, nil
	}

	if containerBoxHeaders, err = br.readAllBoxes(); err != nil {
		return nil, err
	}

	return containerBoxHeaders, nil
}

func (br *BoxReader) readAllBoxes() ([]ContainerBoxHeader, error) {

	var boxHeaders []ContainerBoxHeader
	//boxSizeArray := make([]byte, 4)
	boxSizeArray := make([]byte, 8)
	boxTag := make([]byte, 4)
	for {
		err := br.reader.ReadBytesToBuffer(boxSizeArray, 4)
		if err != nil {
			if err == io.EOF {
				// simple end of file... return with boxHeaders
				return boxHeaders, nil
			}
			return nil, err
		}

		boxSize := makeTag(boxSizeArray, 0, 4)
		if boxSize == 1 {
			err = br.reader.ReadBytesToBuffer(boxSizeArray, 8)
			if err != nil {
				return nil, err
			}
			boxSize = makeTag(boxSizeArray, 0, 8)
			if boxSize > 0 {
				boxSize -= 8
			}
		}
		if boxSize > 0 {
			boxSize -= 8
		}
		if boxSize < 0 {
			return nil, errors.New("invalid box size")
		}

		err = br.reader.ReadBytesToBuffer(boxTag, 4)
		if err != nil {
			return nil, err
		}
		tag := makeTag(boxTag, 0, 4)

		// check boxType...  if we dont know the box type, just skip over the bytes and keep reading.
		switch tag {
		case JXLP:
			// reads next 4 bytes as additional tag?
			err = br.reader.ReadBytesToBuffer(boxTag, 4)
			if err != nil {
				return nil, err
			}
			boxSize -= 4

			// fileoffset...  directly from ReadSeeker?
			pos, err := br.reader.Seek(0, io.SeekCurrent)
			if err != nil {
				return nil, err
			}
			bh := ContainerBoxHeader{
				BoxType:   tag,
				BoxSize:   boxSize,
				IsLast:    false,
				Offset:    pos,
				Processed: false,
			}
			boxHeaders = append(boxHeaders, bh)
			// skip past this box.
			_, err = br.SkipFully(int64(boxSize))
			if err != nil {
				return nil, err
			}

		case JXLL:
			if boxSize != 1 {
				return nil, errors.New("JXLL box size should be 1")
			}
			l, err := br.reader.ReadByte()
			if err != nil {
				return nil, err
			}
			if l != 5 && l != 10 {
				return nil, errors.New("invalid level")
			}
			br.level = int(l)

		case JXLC:
			// fileoffset...  directly from ReadSeeker?
			pos, err := br.reader.Seek(0, io.SeekCurrent)
			if err != nil {
				return nil, err
			}
			bh := ContainerBoxHeader{
				BoxType:   tag,
				BoxSize:   boxSize,
				IsLast:    false,
				Offset:    pos,
				Processed: false,
			}
			boxHeaders = append(boxHeaders, bh)

			// skip past this box.
			_, err = br.SkipFully(int64(boxSize))
			if err != nil {
				return nil, err
			}

		default:
			// skip over the bytes
			if boxSize > 0 {
				s, err := br.SkipFully(int64(boxSize))
				if err != nil {
					return nil, err
				}
				if s != 0 {
					return nil, errors.New("truncated extra box")
				}
			} else {
				panic("java read supplyExceptionally... unsure why?")
			}
		}
	}

	return boxHeaders, nil
}

func (br *BoxReader) readBox() error {

	return nil
}

// returns number of bytes that were NOT skipped.
func (br *BoxReader) SkipFully(i int64) (int64, error) {
	n, err := br.reader.Skip(uint32(i))
	return i - n, err
}

func makeTag(bytes []uint8, offset int, length int) uint64 {
	tag := uint64(0)
	for i := offset; i < offset+length; i++ {
		tag = (tag << 8) | uint64(bytes[i])&0xFF
	}
	return tag
}

func fromBeToLe(le uint32) uint32 {
	return uint32(binary.BigEndian.Uint32([]byte{byte(le), byte(le >> 8), byte(le >> 16), byte(le >> 24)}))
}
