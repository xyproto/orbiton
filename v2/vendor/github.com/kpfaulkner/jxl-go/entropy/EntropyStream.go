package entropy

import (
	"errors"

	"github.com/kpfaulkner/jxl-go/jxlio"
)

var (
	SPECIAL_DISTANCES = [][]int32{
		{0, 1}, {1, 0}, {1, 1}, {-1, 1}, {0, 2}, {2, 0}, {1, 2}, {-1, 2}, {2, 1}, {-2, 1}, {2, 2},
		{-2, 2}, {0, 3}, {3, 0}, {1, 3}, {-1, 3}, {3, 1}, {-3, 1}, {2, 3}, {-2, 3}, {3, 2},
		{-3, 2}, {0, 4}, {4, 0}, {1, 4}, {-1, 4}, {4, 1}, {-4, 1}, {3, 3}, {-3, 3}, {2, 4},
		{-2, 4}, {4, 2}, {-4, 2}, {0, 5}, {3, 4}, {-3, 4}, {4, 3}, {-4, 3}, {5, 0}, {1, 5},
		{-1, 5}, {5, 1}, {-5, 1}, {2, 5}, {-2, 5}, {5, 2}, {-5, 2}, {4, 4}, {-4, 4}, {3, 5},
		{-3, 5}, {5, 3}, {-5, 3}, {0, 6}, {6, 0}, {1, 6}, {-1, 6}, {6, 1}, {-6, 1}, {2, 6},
		{-2, 6}, {6, 2}, {-6, 2}, {4, 5}, {-4, 5}, {5, 4}, {-5, 4}, {3, 6}, {-3, 6}, {6, 3},
		{-6, 3}, {0, 7}, {7, 0}, {1, 7}, {-1, 7}, {5, 5}, {-5, 5}, {7, 1}, {-7, 1}, {4, 6},
		{-4, 6}, {6, 4}, {-6, 4}, {2, 7}, {-2, 7}, {7, 2}, {-7, 2}, {3, 7}, {-3, 7}, {7, 3},
		{-7, 3}, {5, 6}, {-5, 6}, {6, 5}, {-6, 5}, {8, 0}, {4, 7}, {-4, 7}, {7, 4}, {-7, 4},
		{8, 1}, {8, 2}, {6, 6}, {-6, 6}, {8, 3}, {5, 7}, {-5, 7}, {7, 5}, {-7, 5}, {8, 4}, {6, 7},
		{-6, 7}, {7, 6}, {-7, 6}, {8, 5}, {7, 7}, {-7, 7}, {8, 6}, {8, 7}}
)

type EntropyStream struct {
	window         []int32
	clusterMap     []int
	dists          []SymbolDistribution
	lzLengthConfig *HybridIntegerConfig
	ansState       *ANSState

	logAlphabetSize int32
	lz77MinSymbol   int32
	lz77MinLength   int32
	numToCopy77     int32
	copyPos77       int32
	numDecoded77    int32
	usesLZ77        bool
}

// Creating function types to make easier to pass in functions to functions (for later mocking)
type ReadClusterMapFunc func(reader jxlio.BitReader, clusterMap []int, maxClusters int) (int, error)
type EntropyStreamWithReaderAndNumDistsFunc func(reader jxlio.BitReader, numDists int, readClusterMapFunc ReadClusterMapFunc) (*EntropyStream, error)

func NewEntropyStreamWithReaderAndNumDists(reader jxlio.BitReader, numDists int, readClusterMapFunc ReadClusterMapFunc) (*EntropyStream, error) {
	return NewEntropyStreamWithReader(reader, numDists, false, readClusterMapFunc)
}

func NewEntropyStreamWithStream(stream *EntropyStream) *EntropyStream {
	es := &EntropyStream{}
	es.usesLZ77 = stream.usesLZ77
	es.lz77MinLength = stream.lz77MinLength
	es.lz77MinSymbol = stream.lz77MinSymbol
	es.lzLengthConfig = stream.lzLengthConfig
	es.clusterMap = stream.clusterMap
	es.dists = stream.dists
	es.logAlphabetSize = stream.logAlphabetSize
	if es.usesLZ77 {
		es.window = make([]int32, 1<<20)
	}
	es.ansState = &ANSState{State: -1, HasState: false}
	return es
}

func NewEntropyStreamWithReader(reader jxlio.BitReader, numDists int, disallowLZ77 bool, readClusterMapFunc func(reader jxlio.BitReader, clusterMap []int, maxClusters int) (int, error)) (*EntropyStream, error) {

	es := &EntropyStream{}

	var err error
	if numDists <= 0 {
		return nil, errors.New("Num Dists must be positive")
	}

	if es.usesLZ77, err = reader.ReadBool(); err != nil {
		return nil, err
	}
	es.ansState = &ANSState{State: -1, HasState: false}
	if es.usesLZ77 {
		if disallowLZ77 {
			return nil, errors.New("Nested distributions cannot use LZ77")
		}
		if lz77MinSymbol, err := reader.ReadU32(224, 0, 512, 0, 4096, 0, 8, 15); err != nil {
			return nil, err
		} else {
			es.lz77MinSymbol = int32(lz77MinSymbol)
		}

		if lz77MinLength, err := reader.ReadU32(3, 0, 4, 0, 5, 2, 9, 8); err != nil {
			return nil, err
		} else {
			es.lz77MinLength = int32(lz77MinLength)
		}
		numDists++
		es.lzLengthConfig, err = NewHybridIntegerConfigWithReader(reader, 8)
		if err != nil {
			return nil, err
		}
		es.window = make([]int32, 1<<20)
	}

	es.clusterMap = make([]int, numDists)
	numClusters, err := readClusterMapFunc(reader, es.clusterMap, numDists)
	if err != nil {
		return nil, err
	}

	es.dists = make([]SymbolDistribution, numClusters)
	var prefixCodes bool

	if prefixCodes, err = reader.ReadBool(); err != nil {
		return nil, err
	}
	if prefixCodes {
		es.logAlphabetSize = 15
	} else {
		if logAlphabetSize, err := reader.ReadBits(2); err != nil {
			return nil, err
		} else {
			es.logAlphabetSize = 5 + int32(logAlphabetSize)
		}
	}

	configs := make([]*HybridIntegerConfig, len(es.dists))
	for i := 0; i < len(configs); i++ {
		configs[i], err = NewHybridIntegerConfigWithReader(reader, es.logAlphabetSize)
		if err != nil {
			return nil, err
		}
	}

	if prefixCodes {
		alphabetSizes := make([]int32, len(es.dists))
		for i := 0; i < len(es.dists); i++ {
			var readBits bool
			if readBits, err = reader.ReadBool(); err != nil {
				return nil, err
			}
			if readBits {
				var n uint64
				if n, err = reader.ReadBits(4); err != nil {
					return nil, err
				}

				if alphaSize, err := reader.ReadBits(uint32(n)); err != nil {
					return nil, err
				} else {
					alphabetSizes[i] = 1 + int32(1<<n+alphaSize)
				}
			} else {
				alphabetSizes[i] = 1
			}
		}
		for i := 0; i < len(es.dists); i++ {
			es.dists[i], err = NewPrefixSymbolDistributionWithReader(reader, alphabetSizes[i])
			if err != nil {
				return nil, err
			}
		}
	} else {
		for i := 0; i < len(es.dists); i++ {
			d, err := NewANSSymbolDistribution(reader, es.logAlphabetSize)
			if err != nil {
				return nil, err
			}
			es.dists[i] = d
		}
	}

	for i := 0; i < len(es.dists); i++ {
		es.dists[i].SetConfig(configs[i])
	}

	return es, nil

}

func ReadClusterMap(reader jxlio.BitReader, clusterMap []int, maxClusters int) (int, error) {
	numDists := len(clusterMap)
	if numDists == 1 {
		clusterMap[0] = 0
	} else {
		var simpleClustering bool
		var err error
		if simpleClustering, err = reader.ReadBool(); err != nil {
			return 0, err
		}
		if simpleClustering {
			var nbits uint64

			if nbits, err = reader.ReadBits(2); err != nil {
				return 0, err
			}
			for i := 0; i < numDists; i++ {
				if cm, err := reader.ReadBits(uint32(nbits)); err != nil {
					return 0, err
				} else {
					clusterMap[i] = int(cm)
				}
			}
		} else {
			var useMtf bool
			var err error
			if useMtf, err = reader.ReadBool(); err != nil {
				return 0, err
			}
			nested, err := NewEntropyStreamWithReader(reader, 1, numDists <= 2, ReadClusterMap)
			if err != nil {
				return 0, err
			}

			for i := 0; i < numDists; i++ {
				c, err := nested.ReadSymbol(reader, 0)
				clusterMap[i] = int(c)
				if err != nil {
					return 0, err
				}
			}

			if !nested.ValidateFinalState() {
				return 0, errors.New("nested distribution")
			}

			if useMtf {
				mtf := make([]int, 256)
				for i := 0; i < 256; i++ {
					mtf[i] = i
				}
				for i := 0; i < numDists; i++ {
					index := clusterMap[i]
					clusterMap[i] = mtf[index]
					if index != 0 {
						value := mtf[index]
						for j := index; j > 0; j-- {
							mtf[j] = mtf[j-1]
						}
						mtf[0] = value
					}
				}
			}
		}
	}
	numClusters := 0
	for i := 0; i < numDists; i++ {
		if clusterMap[i] >= numClusters {
			numClusters = clusterMap[i] + 1
		}
	}
	if numClusters > maxClusters {
		return 0, errors.New("Too many clusters")
	}
	return numClusters, nil
}

func (es *EntropyStream) GetState() *ANSState {
	return es.ansState
}

func (es *EntropyStream) ReadSymbol(reader jxlio.BitReader, context int) (int32, error) {
	return es.ReadSymbolWithMultiplier(reader, context, 0)
}

func (es *EntropyStream) TryReadSymbol(reader jxlio.BitReader, context int) int32 {
	v, err := es.ReadSymbol(reader, context)
	if err != nil {
		panic(err)
	}
	return v
}

func (es *EntropyStream) ReadSymbolWithMultiplier(reader jxlio.BitReader, context int, distanceMultiplier int32) (int32, error) {
	if es.numToCopy77 > 0 {
		es.copyPos77++
		hybridInt := es.window[es.copyPos77&0xFFFFF]
		es.numToCopy77--
		es.numDecoded77++
		es.window[es.numDecoded77&0xFFFFF] = hybridInt
		return hybridInt, nil
	}
	if context >= len(es.clusterMap) {
		return 0, errors.New("Context cannot be bigger than bundle length")
	}
	if es.clusterMap[context] >= len(es.dists) {
		return 0, errors.New("Cluster Map points to nonexisted distribution")
	}

	dist := es.dists[es.clusterMap[context]]
	t, err := dist.ReadSymbol(reader, es.ansState)
	token := int32(t)
	if err != nil {
		return 0, err
	}

	if es.usesLZ77 && token >= es.lz77MinSymbol {
		lz77dist := es.dists[es.clusterMap[len(es.clusterMap)-1]]
		hi, err := es.readHybridInteger(reader, es.lzLengthConfig, token-es.lz77MinSymbol)
		if err != nil {
			return 0, err
		}
		es.numToCopy77 = es.lz77MinLength + hi
		token, err = lz77dist.ReadSymbol(reader, es.ansState)
		if err != nil {
			return 0, err
		}
		distance, err := es.readHybridInteger(reader, lz77dist.GetConfig(), token)
		if err != nil {
			return 0, err
		}
		if distanceMultiplier == 0 {
			distance++
		} else if distance < 120 {
			distance = SPECIAL_DISTANCES[distance][0] + distanceMultiplier*SPECIAL_DISTANCES[distance][1]
		} else {
			distance -= 119
		}
		if distance > (1 << 20) {
			distance = 1 << 20
		}
		if distance > es.numDecoded77 {
			distance = es.numDecoded77
		}
		es.copyPos77 = es.numDecoded77 - distance
		return es.ReadSymbolWithMultiplier(reader, context, distanceMultiplier)

	}
	hybridInt, err := es.readHybridInteger(reader, dist.GetConfig(), token)
	if err != nil {
		return 0, err
	}
	if es.usesLZ77 {
		es.numDecoded77++
		es.window[es.numDecoded77&0xFFFFF] = hybridInt
	}
	return hybridInt, nil
}

func (es *EntropyStream) readHybridInteger(reader jxlio.BitReader, config *HybridIntegerConfig, token int32) (int32, error) {
	split := 1 << config.SplitExponent
	if token < int32(split) {
		return token, nil
	}
	n := config.SplitExponent - config.LsbInToken - config.MsbInToken + (token-int32(split))>>(config.MsbInToken+config.LsbInToken)
	if n > 32 {
		return 0, errors.New("n is too large")
	}
	low := token & ((1 << config.LsbInToken) - 1)
	token = int32(uint32(token) >> config.LsbInToken)
	token &= (1 << config.MsbInToken) - 1
	token |= 1 << config.MsbInToken
	if data, err := reader.ReadBits(uint32(n)); err != nil {
		return 0, err
	} else {
		return ((int32(token<<n)|int32(data))<<int32(config.LsbInToken) | int32(low)), nil
	}
}

func (es *EntropyStream) ValidateFinalState() bool {
	return !es.ansState.HasState || es.ansState.State == 0x130000
}
