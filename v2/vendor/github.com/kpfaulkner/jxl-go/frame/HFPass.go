package frame

import (
	"errors"
	"slices"

	"github.com/kpfaulkner/jxl-go/entropy"
	"github.com/kpfaulkner/jxl-go/jxlio"
	"github.com/kpfaulkner/jxl-go/util"
)

var (
	orderLookup     = make([]TransformType, 13)
	parameterLookup = make([]TransformType, 17)
	typeLookup      = make([]TransformType, 27)
)

// not crazy about init functions...  but think this is probably the best way to handle this.
func init() {
	for i := int32(0); i < 13; i++ {
		tt, err := filterByOrderID(i)
		if err != nil {

			// intentionally panic. If we can't setup this data, fail immediately
			panic(err)
		}
		orderLookup[i] = tt
	}

	for i := int32(0); i < 17; i++ {
		tt, err := filterByParameterIndex(i)
		if err != nil {
			// intentionally panic. If we can't setup this data, fail immediately
			panic(err)
		}
		parameterLookup[i] = tt

	}

	for i := int32(0); i < 27; i++ {
		tt, err := filterByType(i)
		if err != nil {
			// intentionally panic. If we can't setup this data, fail immediately
			panic(err)
		}
		typeLookup[i] = tt
	}
}

func filterByType(typeIndex int32) (TransformType, error) {
	for _, tt := range allDCT {
		if tt.ttType == typeIndex {
			return tt, nil
		}
	}

	return TransformType{}, errors.New("Unable to find transform type for typeIndex")
}

func filterByParameterIndex(parameterIndex int32) (TransformType, error) {
	for _, tt := range allDCT {
		if tt.parameterIndex == parameterIndex && !tt.isVertical() {
			return tt, nil
		}
	}

	return TransformType{}, errors.New("Unable to find transform type for parameterIndex")
}

func filterByOrderID(orderID int32) (TransformType, error) {
	for _, tt := range allDCT {
		if tt.orderID == orderID && !tt.isVertical() {
			return tt, nil
		}
	}

	return TransformType{}, errors.New("Unable to find transform type for orderID")
}

type HFPass struct {
	order         [][][]util.Point
	naturalOrder  [][]util.Point
	contextStream *entropy.EntropyStream
	usedOrders    uint32
}

func NewHFPassWithReader(reader jxlio.BitReader, frame Framer, passIndex uint32,
	readClusterMapFunc entropy.ReadClusterMapFunc,
	newEntropyStreamWithReader entropy.EntropyStreamWithReaderAndNumDistsFunc,
	readPermutation ReadPermutationFunc) (*HFPass, error) {
	hfp := &HFPass{}
	hfp.naturalOrder = util.MakeMatrix2D[util.Point](13, 0)
	hfp.order = util.MakeMatrix3D[util.Point](13, 3, 0)
	usedOrders, err := reader.ReadU32(0x5F, 0, 0x13, 0, 0, 0, 0, 13)
	if err != nil {
		return nil, err
	}
	hfp.usedOrders = usedOrders
	var stream *entropy.EntropyStream
	if usedOrders != 0 {
		if stream, err = newEntropyStreamWithReader(reader, 8, readClusterMapFunc); err != nil {
			return nil, err
		}
	} else {
		stream = nil
	}

	for b := int32(0); b < 13; b++ {
		naturalOrder, err := hfp.getNaturalOrder(b)
		if err != nil {
			return nil, err
		}
		l := len(naturalOrder)

		for c := 0; c < 3; c++ {
			if usedOrders&(1<<uint32(b)) != 0 {
				hfp.order[b][c] = make([]util.Point, l)
				perm, err := readPermutation(reader, stream, uint32(l), uint32(l/64))
				if err != nil {
					return nil, err
				}
				for i := 0; i < len(hfp.order[b][c]); i++ {
					hfp.order[b][c][i] = naturalOrder[perm[i]]
				}
			} else {
				hfp.order[b][c] = naturalOrder
			}
		}
	}
	if stream != nil && !stream.ValidateFinalState() {
		return nil, errors.New("ANS state decoding error")
	}
	numContexts := 495 * frame.getHFGlobal().numHFPresets * frame.getLFGlobal().hfBlockCtx.numClusters
	contextStream, err := newEntropyStreamWithReader(reader, int(numContexts), readClusterMapFunc)
	if err != nil {
		return nil, err
	}

	hfp.contextStream = contextStream
	return hfp, nil
}

func (hfp *HFPass) getNaturalOrder(i int32) ([]util.Point, error) {
	if len(hfp.naturalOrder[i]) != 0 {
		return hfp.naturalOrder[i], nil
	}

	tt := getByOrderID(i)
	l := tt.pixelWidth * tt.pixelHeight
	hfp.naturalOrder[i] = make([]util.Point, l)
	for y := int32(0); y < tt.pixelHeight; y++ {
		for x := int32(0); x < tt.pixelWidth; x++ {
			hfp.naturalOrder[i][y*tt.pixelWidth+x] = util.Point{X: x, Y: y}
		}
	}

	sorterFunc, err := getNaturalOrderFunc(i)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(hfp.naturalOrder[i], sorterFunc)

	return hfp.naturalOrder[i], nil

}

func getNaturalOrderFunc(i int32) (func(a util.Point, b util.Point) int, error) {

	tt := getByOrderID(i)

	return func(a util.Point, b util.Point) int {
		maxDim := util.Max(tt.dctSelectHeight, tt.dctSelectWidth)
		aLLF := a.Y < tt.dctSelectHeight && a.X < tt.dctSelectWidth
		bLLF := b.Y < tt.dctSelectHeight && b.X < tt.dctSelectWidth
		if aLLF && !bLLF {
			return -1
		}
		if !aLLF && bLLF {
			return 1
		}
		if aLLF && bLLF {
			if b.Y != a.Y {
				return int(a.Y - b.Y)
			}
			return int(a.X - b.X)
		}

		heightDivider := maxDim / tt.dctSelectHeight
		widthDivider := maxDim / tt.dctSelectWidth
		aSY := a.Y * heightDivider
		aSX := a.X * widthDivider
		bSY := b.Y * heightDivider
		bSX := b.X * widthDivider
		aKey1 := aSY + aSX
		bKey1 := bSY + bSX
		if aKey1 != bKey1 {
			return int(aKey1 - bKey1)
		}

		aKey2 := aSX - aSY
		bKey2 := bSX - bSY
		if (aKey1 & 1) == 1 {
			aKey2 = -aKey2
		}
		if (bKey1 & 1) == 1 {
			bKey2 = -bKey2
		}
		return int(aKey2 - bKey2)
	}, nil
}
