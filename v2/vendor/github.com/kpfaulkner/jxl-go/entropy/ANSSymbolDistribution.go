package entropy

import (
	"errors"
	"fmt"

	"github.com/kpfaulkner/jxl-go/jxlio"
	"github.com/kpfaulkner/jxl-go/util"
)

var distPrefixTable = NewVLCTable(7, [][]int32{{10, 3}, {12, 7}, {7, 3}, {3, 4}, {6, 3}, {8, 3}, {9, 3}, {5, 4}, {10, 3}, {4, 4}, {7, 3}, {1, 4}, {6, 3}, {8, 3}, {9, 3}, {2, 4}, {10, 3}, {0, 5}, {7, 3}, {3, 4}, {6, 3}, {8, 3}, {9, 3}, {5, 4}, {10, 3}, {4, 4}, {7, 3}, {1, 4}, {6, 3}, {8, 3}, {9, 3}, {2, 4}, {10, 3}, {11, 6}, {7, 3}, {3, 4}, {6, 3}, {8, 3}, {9, 3}, {5, 4}, {10, 3}, {4, 4}, {7, 3}, {1, 4}, {6, 3}, {8, 3}, {9, 3}, {2, 4}, {10, 3}, {0, 5}, {7, 3}, {3, 4}, {6, 3}, {8, 3}, {9, 3}, {5, 4}, {10, 3}, {4, 4}, {7, 3}, {1, 4}, {6, 3}, {8, 3}, {9, 3}, {2, 4}, {10, 3}, {13, 7}, {7, 3}, {3, 4}, {6, 3}, {8, 3}, {9, 3}, {5, 4}, {10, 3}, {4, 4}, {7, 3}, {1, 4}, {6, 3}, {8, 3}, {9, 3}, {2, 4}, {10, 3}, {0, 5}, {7, 3}, {3, 4}, {6, 3}, {8, 3}, {9, 3}, {5, 4}, {10, 3}, {4, 4}, {7, 3}, {1, 4}, {6, 3}, {8, 3}, {9, 3}, {2, 4}, {10, 3}, {11, 6}, {7, 3}, {3, 4}, {6, 3}, {8, 3}, {9, 3}, {5, 4}, {10, 3}, {4, 4}, {7, 3}, {1, 4}, {6, 3}, {8, 3}, {9, 3}, {2, 4}, {10, 3}, {0, 5}, {7, 3}, {3, 4}, {6, 3}, {8, 3}, {9, 3}, {5, 4}, {10, 3}, {4, 4}, {7, 3}, {1, 4}, {6, 3}, {8, 3}, {9, 3}, {2, 4}})

var (
	// just for some debugging
	lastStateObj int32
)

type ANSSymbolDistribution struct {
	SymbolDistributionBase
	frequencies []int32
	cutoffs     []int32
	symbols     []int32
	offsets     []int32
}

func NewANSSymbolDistribution(reader jxlio.BitReader, logAlphabetSize int32) (*ANSSymbolDistribution, error) {
	asd := &ANSSymbolDistribution{}
	asd.logAlphabetSize = logAlphabetSize
	uniqPos := int32(-1)
	var simpleDistribution bool
	var err error
	if simpleDistribution, err = reader.ReadBool(); err != nil {
		return nil, err
	}

	if simpleDistribution {

		var dist1 bool
		var err error
		if dist1, err = reader.ReadBool(); err != nil {
			return nil, err
		}

		if dist1 {
			v1, err := reader.ReadU8()
			if err != nil {
				return nil, err
			}
			v2, err := reader.ReadU8()
			if err != nil {
				return nil, err
			}
			if v1 == v2 {
				return nil, errors.New("Overlapping dual peak distribution")
			}
			asd.alphabetSize = 1 + util.Max[int32](int32(v1), int32(v2))
			if asd.alphabetSize > (1 << asd.logAlphabetSize) {
				return nil, errors.New(fmt.Sprintf("Illegal Alphabet size : %d", asd.alphabetSize))
			}
			asd.frequencies = make([]int32, asd.alphabetSize)
			if freq, err := reader.ReadBits(12); err != nil {
				return nil, err
			} else {
				asd.frequencies[v1] = int32(freq)
			}
			asd.frequencies[v2] = 1<<12 - asd.frequencies[v1]
			if asd.frequencies[v1] == 0 {
				uniqPos = int32(v2)
			}
		} else {
			x, err := reader.ReadU8()
			if err != nil {
				return nil, err
			}
			asd.alphabetSize = 1 + int32(x)
			asd.frequencies = make([]int32, asd.alphabetSize)
			asd.frequencies[x] = 1 << 12
			uniqPos = int32(x)
		}
	} else {
		// flat distribution
		var flat bool
		if flat, err = reader.ReadBool(); err != nil {
			return nil, err
		}
		if flat {
			r, err := reader.ReadU8()
			if err != nil {
				return nil, err
			}
			asd.alphabetSize = 1 + int32(r)
			if asd.alphabetSize > (1 << asd.logAlphabetSize) {
				return nil, errors.New(fmt.Sprintf("Illegal Alphabet size : %d", asd.alphabetSize))
			}
			if asd.alphabetSize == 1 {
				uniqPos = 0
			}

			asd.frequencies = make([]int32, asd.alphabetSize)
			for i := int32(0); i < asd.alphabetSize; i++ {
				asd.frequencies[i] = (1 << 12) / asd.alphabetSize
			}
			for i := int32(0); i < (1<<12)%asd.alphabetSize; i++ {
				asd.frequencies[i]++
			}
		} else {
			var l int
			var err error
			var b bool
			for l = 0; l < 3; l++ {
				if b, err = reader.ReadBool(); err != nil {
					return nil, err
				}
				if !b {
					break
				}
			}
			var shift uint64
			if shift, err = reader.ReadBits(uint32(l)); err != nil {
				return nil, err
			} else {
				shift = (shift | 1<<l) - 1
			}
			if shift > 13 {
				return nil, errors.New("Shift > 13")
			}
			r, err := reader.ReadU8()
			if err != nil {
				return nil, err
			}
			asd.alphabetSize = 3 + int32(r)
			if asd.alphabetSize > (1 << asd.logAlphabetSize) {
				return nil, errors.New(fmt.Sprintf("Illegal Alphabet size : %d", asd.alphabetSize))
			}

			asd.frequencies = make([]int32, asd.alphabetSize)
			logCounts := make([]int32, asd.alphabetSize)
			same := make([]int, asd.alphabetSize)
			omitLog := int32(-1)
			omitPos := int32(-1)
			for i := int32(0); i < asd.alphabetSize; i++ {
				logCounts[i], err = distPrefixTable.GetVLC(reader)
				if err != nil {
					return nil, err
				}
				if logCounts[i] == 13 {
					rle, err := reader.ReadU8()
					if err != nil {
						return nil, err
					}
					same[i] = rle + 5
					i += int32(rle) + 3
					continue
				}
				if logCounts[i] > omitLog {
					omitLog = logCounts[i]
					omitPos = i
				}
			}

			if omitPos < 0 || omitPos+1 < asd.alphabetSize && logCounts[omitPos+1] == 13 {
				return nil, errors.New("Invalid OmitPos")
			}
			totalCount := int32(0)
			numSame := 0
			prev := int32(0)
			for i := int32(0); i < asd.alphabetSize; i++ {
				if same[i] != 0 {
					numSame = same[i] - 1
					if i > 0 {
						prev = asd.frequencies[i-1]
					} else {
						prev = 0
					}
				}
				if numSame != 0 {
					asd.frequencies[i] = prev
					numSame--
				} else {
					if i == omitPos || logCounts[i] == 0 {
						continue
					}
					if logCounts[i] == 1 {
						asd.frequencies[i] = 1
					} else {
						bitcount := int32(shift) - int32(uint32(12-logCounts[i]+1)>>1)
						if bitcount < 0 {
							bitcount = 0
						}
						if bitcount > int32(logCounts[i])-1 {
							bitcount = int32(logCounts[i] - 1)
						}
						if freq, err := reader.ReadBits(uint32(bitcount)); err != nil {
							return nil, err
						} else {
							asd.frequencies[i] = int32(1<<(logCounts[i]-1) + int32(freq)<<(logCounts[i]-1-bitcount))
						}

					}
				}
				totalCount += asd.frequencies[i]
			}
			asd.frequencies[omitPos] = (1 << 12) - totalCount
		}
	}
	asd.generateAliasMapping(uniqPos)
	return asd, nil
}

func (asd *ANSSymbolDistribution) generateAliasMapping(uniqPos int32) {
	asd.logBucketSize = 12 - asd.logAlphabetSize
	bucketSize := int32(1 << asd.logBucketSize)
	tableSize := int32(1 << asd.logAlphabetSize)
	overfull := util.NewDeque[int32]()
	underfull := util.NewDeque[int32]()

	asd.symbols = make([]int32, tableSize)
	asd.cutoffs = make([]int32, tableSize)
	asd.offsets = make([]int32, tableSize)
	if uniqPos >= 0 {
		for i := int32(0); i < tableSize; i++ {
			asd.symbols[i] = uniqPos
			asd.offsets[i] = i * bucketSize
			asd.cutoffs[i] = 0
		}
		return
	}

	for i := int32(0); i < asd.alphabetSize; i++ {
		asd.cutoffs[i] = asd.frequencies[i]
		if asd.cutoffs[i] > bucketSize {
			overfull.AddFirst(i)
		} else if asd.cutoffs[i] < bucketSize {
			underfull.AddFirst(i)
		}
	}
	for i := asd.alphabetSize; i < tableSize; i++ {
		underfull.AddFirst(i)
	}
	for !overfull.IsEmpty() {
		u := underfull.RemoveFirst()
		o := overfull.RemoveFirst()
		by := bucketSize - asd.cutoffs[*u]
		asd.cutoffs[*o] -= by
		asd.symbols[*u] = *o
		asd.offsets[*u] = asd.cutoffs[*o]
		if asd.cutoffs[*o] < bucketSize {
			underfull.AddFirst(*o)
		} else if asd.cutoffs[*o] > bucketSize {
			overfull.AddFirst(*o)
		}
	}
	for i := int32(0); i < tableSize; i++ {
		if asd.cutoffs[i] == bucketSize {
			asd.symbols[i] = i
			asd.offsets[i] = 0
			asd.cutoffs[i] = 0
		} else {
			asd.offsets[i] -= asd.cutoffs[i]
		}
	}
}

func (asd *ANSSymbolDistribution) ReadSymbol(reader jxlio.BitReader, stateObj *ANSState) (int32, error) {

	var state int32
	if !stateObj.HasState {
		s, err := reader.ReadBits(32)
		if err != nil {
			return 0, err
		}
		state = int32(s)
	} else {
		state = stateObj.State
	}

	index := state & 0xFFF
	i := uint32(index) >> asd.logBucketSize
	pos := index & ((1 << asd.logBucketSize) - 1)
	var symbol int32
	var offset int32
	if pos >= asd.cutoffs[i] {
		symbol = asd.symbols[i]
		offset = asd.offsets[i] + int32(pos)
	} else {
		symbol = int32(i)
		offset = int32(pos)
	}

	state = asd.frequencies[symbol]*int32(uint32(state)>>12) + offset
	if uint32(state)&0xFFFF0000 == 0 {
		state = (state << 16)
		data, err := reader.ReadBits(16)
		if err != nil {
			return 0, err
		}
		state = state | int32(data)
	}
	stateObj.SetState(state)
	return symbol, nil
}
