package entropy

import (
	"errors"
	bbits "math/bits"

	"github.com/kpfaulkner/jxl-go/jxlio"
	"github.com/kpfaulkner/jxl-go/util"
)

type VLCTable struct {
	table [][]int32
	bits  int32
}

func NewVLCTable(bits int32, table [][]int32) (rcvr *VLCTable) {
	rcvr = &VLCTable{}
	rcvr.bits = bits
	rcvr.table = table
	return
}
func NewVLCTableWithSymbols(bits int32, lengths []int32, symbols []int32) (*VLCTable, error) {
	rcvr := &VLCTable{}
	rcvr.bits = bits
	table := util.MakeMatrix2D[int32](1<<bits, 2)
	codes := make([]int, len(lengths))
	nLengths := make([]int32, len(lengths))
	nSymbols := make([]int32, len(lengths))
	count := 0
	code := 0
	for i := int32(0); i < int32(len(lengths)); i++ {
		currentLen := lengths[i]
		if currentLen > 0 {
			nLengths[count] = currentLen
			if len(symbols) > 0 {
				nSymbols[count] = symbols[i]
			} else {
				nSymbols[count] = i
			}
			codes[count] = int(code)
			count++
		} else if currentLen < 0 {
			currentLen = -currentLen
		} else {
			continue
		}
		code += 1 << (32 - currentLen)
		if code > 1<<32 {
			return nil, errors.New("Too many VLC codes")
		}
	}
	if code != 1<<32 {
		return nil, errors.New("Not enough VLC codes")
	}
	for i := 0; i < count; i++ {
		if nLengths[i] <= bits {
			index := bbits.Reverse32(uint32(codes[i]))
			number := 1 << (bits - nLengths[i])
			offset := 1 << nLengths[i]
			for j := 0; j < number; j++ {
				oldSymbol := table[index][0]
				oldLen := table[index][1]
				if (oldLen > 0 || oldSymbol > 0) && (oldLen != nLengths[i] || oldSymbol != nSymbols[i]) {
					return nil, errors.New("Illegal VLC codes")
				}
				table[index][0] = nSymbols[i]
				table[index][1] = nLengths[i]
				index += uint32(offset)
			}
		} else {
			return nil, errors.New("Table size too small")
		}
	}
	for i := 0; i < len(table); i++ {
		if table[i][1] == 0 {
			table[i][0] = -1
		}
	}
	rcvr.table = table
	return rcvr, nil
}
func (rcvr *VLCTable) GetVLC(reader jxlio.BitReader) (int32, error) {

	index, err := reader.ShowBits(int(rcvr.bits))
	if err != nil {
		return 0, err
	}
	symbol := rcvr.table[index][0]
	length := rcvr.table[index][1]
	err = reader.SkipBits(uint32(length))
	if err != nil {
		return 0, err
	}

	return symbol, nil
}
