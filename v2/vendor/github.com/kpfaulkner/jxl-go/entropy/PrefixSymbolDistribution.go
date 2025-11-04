package entropy

import (
	"errors"
	"slices"

	"github.com/kpfaulkner/jxl-go/jxlio"
	"github.com/kpfaulkner/jxl-go/util"
)

var level0Table = NewVLCTable(4, [][]int32{{0, 2}, {4, 2}, {3, 2}, {2, 3}, {0, 2}, {4, 2}, {3, 2}, {1, 4}, {0, 2}, {4, 2}, {3, 2}, {2, 3}, {0, 2}, {4, 2}, {3, 2}, {5, 4}})
var codelenMap = []int{1, 2, 3, 4, 0, 5, 17, 6, 16, 7, 8, 9, 10, 11, 12, 13, 14, 15}

type PrefixSymbolDistribution struct {
	*SymbolDistributionBase
	table         *VLCTable
	defaultSymbol int32
}

func NewPrefixSymbolDistributionWithReader(reader jxlio.BitReader, alphabetSize int32) (*PrefixSymbolDistribution, error) {
	rcvr := &PrefixSymbolDistribution{
		SymbolDistributionBase: NewSymbolDistributionBase(),
	}
	rcvr.alphabetSize = alphabetSize
	rcvr.logAlphabetSize = int32(util.CeilLog1p(int64(alphabetSize - 1)))
	if rcvr.alphabetSize == 1 {
		rcvr.table = nil
		rcvr.defaultSymbol = 0
		return rcvr, nil
	}

	var hskip uint64
	var err error
	if hskip, err = reader.ReadBits(2); err != nil {
		return nil, err
	}

	if hskip == 1 {
		rcvr.populateSimplePrefix(reader)
	} else {
		rcvr.populateComplexPrefix(reader, int32(hskip))
	}
	return rcvr, nil
}
func (rcvr *PrefixSymbolDistribution) populateComplexPrefix(reader jxlio.BitReader, hskip int32) error {

	level1Lengths := make([]int32, 18)
	level1Codecounts := make([]int32, 19)
	level1Codecounts[0] = hskip
	totalCode := 0
	numCodes := 0
	for i := hskip; i < 18; i++ {
		code, err := level0Table.GetVLC(reader)
		if err != nil {
			return err
		}
		level1Lengths[codelenMap[i]] = code
		level1Codecounts[code]++
		if code != 0 {
			totalCode += 32 >> code
			numCodes++
		}
		if totalCode >= 32 {
			level1Codecounts[0] += 17 - i
			break
		}
	}
	if totalCode != 32 && numCodes >= 2 || numCodes < 1 {
		return errors.New("Invalid Level 1 Prefix codes")
	}
	for i := 1; i < 19; i++ {
		level1Codecounts[i] += level1Codecounts[i-1]
	}
	level1LengthsScrambled := make([]int32, 18)
	level1Symbols := make([]int32, 18)
	for i := int32(17); i >= 0; i-- {
		level1Codecounts[level1Lengths[i]]--
		index := level1Codecounts[level1Lengths[i]]
		level1LengthsScrambled[index] = level1Lengths[i]
		level1Symbols[index] = i
	}

	var level1Table *VLCTable
	var err error
	if numCodes == 1 {
		level1Table = NewVLCTable(0, [][]int32{{level1Symbols[17], 0}})
	} else {
		level1Table, err = NewVLCTableWithSymbols(5, level1LengthsScrambled, level1Symbols)
		if err != nil {
			return err
		}
	}

	totalCode = 0
	var prevRepeatCount int32
	var prevZeroCount int32
	level2Lengths := make([]int32, rcvr.alphabetSize)
	level2Symbols := make([]int32, rcvr.alphabetSize)
	level2Counts := make([]int32, rcvr.alphabetSize+1)
	prev := int32(8)
	for i := int32(0); i < int32(rcvr.alphabetSize); i++ {
		code, err := level1Table.GetVLC(reader)
		if err != nil {
			return err
		}
		if code == 16 {
			e, err := reader.ReadBits(2)
			if err != nil {
				return err
			}
			extra := 3 + int32(e)
			if prevRepeatCount > 0 {
				extra = 4*(prevRepeatCount-2) - prevRepeatCount + extra
			}
			for j := int32(0); j < extra; j++ {
				level2Lengths[i+j] = prev
			}
			totalCode += int(uint32(32768)>>prev) * int(extra)
			i += extra - 1
			prevRepeatCount += extra
			prevZeroCount = 0
			level2Counts[prev] += extra
		} else if code == 17 {
			e, err := reader.ReadBits(3)
			if err != nil {
				return err
			}
			extra := 3 + int32(e)
			if prevZeroCount > 0 {
				extra = 8*(prevZeroCount-2) - prevZeroCount + extra
			}
			i += extra - 1
			prevRepeatCount = 0
			prevZeroCount += extra
			level2Counts[0] += extra
		} else {
			level2Lengths[i] = code
			prevRepeatCount = 0
			prevZeroCount = 0
			if code != 0 {
				// uint32 casting due to in Java its using unsigned shift, right? Zero fill? (>>>).
				// Go recommendation is cast to uint32 first.
				totalCode += int(uint32(32768) >> code)
				prev = code
			}
			level2Counts[code]++
		}
		if totalCode >= 32768 {
			level2Counts[0] += rcvr.alphabetSize - i - 1
			break
		}
	}
	if totalCode != 32768 && level2Counts[0] < rcvr.alphabetSize-1 {
		return errors.New("Invalid Level 2 Prefix Codes")
	}
	for i := int32(1); i <= rcvr.alphabetSize; i++ {
		level2Counts[i] += level2Counts[i-1]
	}
	level2LengthsScrambled := make([]int32, rcvr.alphabetSize)
	for i := rcvr.alphabetSize - 1; i >= 0; i-- {
		level2Counts[level2Lengths[i]]--
		index := level2Counts[level2Lengths[i]]
		level2LengthsScrambled[index] = level2Lengths[i]
		level2Symbols[index] = i
	}
	rcvr.table, err = NewVLCTableWithSymbols(15, level2LengthsScrambled, level2Symbols)
	if err != nil {
		return err
	}
	return nil
}

func (rcvr *PrefixSymbolDistribution) populateSimplePrefix(reader jxlio.BitReader) error {

	symbols := make([]int32, 4)
	var lens []int32 = nil
	n, err := reader.ReadBits(2)
	if err != nil {
		return err
	}
	nsym := int32(n) + 1
	treeSelect := false
	bits := int32(0)
	for i := 0; i < int(nsym); i++ {
		s, err := reader.ReadBits(uint32(rcvr.logAlphabetSize))
		if err != nil {
			return err
		}
		symbols[i] = int32(s)
	}
	if nsym == 4 {
		if treeSelect, err = reader.ReadBool(); err != nil {
			return err
		}
	}
	switch nsym {
	case 1:
		rcvr.table = nil
		rcvr.defaultSymbol = symbols[0]
		return nil
	case 2:
		bits = 1
		lens = []int32{1, 1, 0, 0}
		if symbols[0] > symbols[1] {
			symbols[0], symbols[1] = symbols[1], symbols[0]
		}
	case 3:
		bits = 2
		lens = []int32{1, 2, 2, 0}
		if symbols[1] > symbols[2] {
			symbols[1], symbols[2] = symbols[2], symbols[1]
		}
	case 4:
		if treeSelect {
			bits = 3
			lens = []int32{1, 2, 3, 3}
			if symbols[2] > symbols[3] {
				symbols[2], symbols[3] = symbols[3], symbols[2]
			}
		} else {
			bits = 2
			lens = []int32{2, 2, 2, 2}
			slices.Sort(symbols)
		}
	}
	rcvr.table, err = NewVLCTableWithSymbols(bits, lens, symbols)
	if err != nil {
		return err
	}
	return nil
}

func (rcvr *PrefixSymbolDistribution) ReadSymbol(reader jxlio.BitReader, state *ANSState) (int32, error) {
	if rcvr.table == nil {
		return rcvr.defaultSymbol, nil
	}
	return rcvr.table.GetVLC(reader)
}
