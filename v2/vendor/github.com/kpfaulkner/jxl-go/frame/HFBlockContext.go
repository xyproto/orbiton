package frame

import (
	"errors"

	"github.com/kpfaulkner/jxl-go/jxlio"
	"github.com/kpfaulkner/jxl-go/util"
)

type HFBlockContext struct {
	lfThresholds  [][]int32
	clusterMap    []int
	numClusters   int32
	qfThresholds  []int32
	numLFContexts int32
}
type NewHFBlockContextFunc func(reader jxlio.BitReader, readClusterMap func(reader jxlio.BitReader, clusterMap []int, maxClusters int) (int, error)) (*HFBlockContext, error)

func NewHFBlockContextWithReader(reader jxlio.BitReader, readClusterMap func(reader jxlio.BitReader, clusterMap []int, maxClusters int) (int, error)) (*HFBlockContext, error) {
	hf := &HFBlockContext{}
	hf.lfThresholds = util.MakeMatrix2D[int32](3, 0)
	useDefault, err := reader.ReadBool()
	if err != nil {
		return nil, err
	}

	if useDefault {
		hf.clusterMap = []int{0, 1, 2, 2, 3, 3, 4, 5, 6, 6, 6, 6, 6,
			7, 8, 9, 9, 10, 11, 12, 13, 14, 14, 14, 14, 14,
			7, 8, 9, 9, 10, 11, 12, 13, 14, 14, 14, 14, 14}
		hf.numClusters = 15
		hf.qfThresholds = []int32{}
		hf.lfThresholds = util.MakeMatrix2D[int32](3, 0)
		hf.numLFContexts = 1
	}

	nbLFThresh := make([]int32, 3)
	lfCtx := int32(1)
	for i := 0; i < 3; i++ {
		nb, err := reader.ReadBits(4)
		if err != nil {
			return nil, err
		}
		nbLFThresh[i] = int32(nb)
		lfCtx *= nbLFThresh[i] + 1
		hf.lfThresholds[i] = make([]int32, nbLFThresh[i])
		for j := int32(0); j < nbLFThresh[i]; j++ {
			t, err := reader.ReadU32(0, 4, 16, 8, 272, 16, 65808, 32)
			if err != nil {
				return nil, err
			}
			hf.lfThresholds[i][j] = jxlio.UnpackSigned(t)
		}
	}

	hf.numLFContexts = lfCtx
	nbQfThread, err := reader.ReadBits(4)
	if err != nil {
		return nil, err
	}
	hf.qfThresholds = make([]int32, nbQfThread)
	for i := 0; i < int(nbQfThread); i++ {
		t, err := reader.ReadU32(0, 2, 4, 3, 12, 5, 44, 8)
		if err != nil {
			return nil, err
		}
		hf.qfThresholds[i] = int32(t) + 1
	}
	bSize := 39 * (int32(nbQfThread) + 1)
	for i := 0; i < 3; i++ {
		bSize *= nbLFThresh[i] + 1
	}
	if bSize > 39*64 {
		return nil, errors.New("HF block size too large")
	}

	hf.clusterMap = make([]int, bSize)
	//nc, err := entropy.ReadClusterMap(reader, hf.clusterMap, 16)
	nc, err := readClusterMap(reader, hf.clusterMap, 16)
	if err != nil {
		return nil, err
	}
	hf.numClusters = int32(nc)
	return hf, nil
}
