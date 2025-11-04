package frame

import (
	"errors"

	"github.com/kpfaulkner/jxl-go/entropy"
	"github.com/kpfaulkner/jxl-go/jxlio"
)

type MATree struct {
	parent          *MATree
	stream          *entropy.EntropyStream
	leftChildNode   *MATree
	rightChildNode  *MATree
	property        int32
	context         int32
	value           int32
	leftChildIndex  int32
	rightChildIndex int32
	predictor       int32
	offset          int32
	multiplier      int32
}

func NewMATreeWithReader(reader jxlio.BitReader) (*MATree, error) {
	mt := &MATree{}
	mt.parent = nil
	var nodes []*MATree

	stream, err := entropy.NewEntropyStreamWithReaderAndNumDists(reader, 6, entropy.ReadClusterMap)
	if err != nil {
		return nil, err
	}

	contextId := int32(0)
	nodesRemaining := 1

	for nodesRemaining > 0 {
		nodesRemaining--

		if len(nodes) > (1 << 20) {
			return nil, errors.New("Tree too large")
		}
		property, err := stream.ReadSymbol(reader, 1)
		if err != nil {
			return nil, err
		}
		property--

		var node *MATree
		if len(nodes) == 0 {
			node = mt
		} else {
			node = &MATree{}
		}

		if property >= 0 {
			value := jxlio.UnpackSigned(uint32(stream.TryReadSymbol(reader, 0)))
			leftChild := len(nodes) + nodesRemaining + 1
			node.property = property
			node.predictor = -1
			node.value = value
			node.leftChildIndex = int32(leftChild)
			node.rightChildIndex = int32(leftChild) + 1
			nodes = append(nodes, node)
			nodesRemaining += 2
		} else {
			context := contextId
			contextId++
			var predictor int32
			var err error
			if predictor, err = stream.ReadSymbol(reader, 2); err != nil {
				return nil, err
			}
			if predictor > 13 {
				return nil, errors.New("invalid predictor value")
			}

			offset := jxlio.UnpackSigned(uint32(stream.TryReadSymbol(reader, 3)))

			var mulLog int32
			if mulLog, err = stream.ReadSymbol(reader, 4); err != nil {
				return nil, err
			}
			if mulLog > 30 {
				return nil, errors.New("mulLog too large")
			}
			var mulBits int32
			if mulBits, err = stream.ReadSymbol(reader, 5); err != nil {
				return nil, err
			}
			if mulBits > (1<<(31-mulLog))-2 {
				return nil, errors.New("mulBits too large")
			}
			multiplier := (mulBits + 1) << uint(mulLog)
			node.context = context
			node.predictor = predictor
			node.multiplier = multiplier
			node.offset = offset
			node.property = -1
			nodes = append(nodes, node)
		}
	}

	if !stream.ValidateFinalState() {
		return nil, errors.New("illegal MA Tree Entropy Stream")
	}
	mt.stream, err = entropy.NewEntropyStreamWithReader(reader, (len(nodes)+1)/2, false, entropy.ReadClusterMap)
	if err != nil {
		return nil, err
	}

	for n, node := range nodes {
		nodes[n].stream = mt.stream
		if !node.isLeafNode() {
			node.leftChildNode = nodes[node.leftChildIndex]
			node.rightChildNode = nodes[node.rightChildIndex]
			node.leftChildNode.parent = node
			node.rightChildNode.parent = node
		}
	}
	return mt, nil
}

func (t *MATree) isLeafNode() bool {
	return t.property < 0
}

func (t *MATree) compactifyWithY(channelIndex int32, streamIndex int32, y int32) *MATree {
	var prop int32
	switch t.property {
	case 0:
		prop = channelIndex
		break
	case 1:
		prop = streamIndex
		break
	case 2:
		prop = y
		break
	default:
		return t
	}

	var branch *MATree
	if prop > t.value {
		branch = t.leftChildNode
	} else {
		branch = t.rightChildNode
	}
	return branch.compactifyWithY(channelIndex, streamIndex, y)
}

func (t *MATree) compactify(channelIndex int32, streamIndex int32) *MATree {

	var prop int32
	switch t.property {
	case 0:
		prop = channelIndex
		break
	case 1:
		prop = streamIndex
		break
	default:
		return t
	}

	var branch *MATree
	if prop > t.value {
		branch = t.leftChildNode
	} else {
		branch = t.rightChildNode
	}
	return branch.compactify(channelIndex, streamIndex)
}

func (t *MATree) useWeightedPredictor() bool {
	if t.isLeafNode() {
		return t.predictor == 6
	}

	return t.property == 15 ||
		t.leftChildNode.useWeightedPredictor() ||
		t.rightChildNode.useWeightedPredictor()
}

func (t *MATree) walk(walkerFunc func(inp int32) (int32, error)) (*MATree, error) {

	if t.isLeafNode() {
		return t, nil
	}

	value, err := walkerFunc(t.property)
	if err != nil {
		return nil, err
	}

	if value > t.value {
		return t.leftChildNode.walk(walkerFunc)
	}
	return t.rightChildNode.walk(walkerFunc)
}

func (t *MATree) getSize() int {
	size := 1
	if !t.isLeafNode() {
		size += t.leftChildNode.getSize()
		size += t.rightChildNode.getSize()
	}
	return size
}

// Prints the tree to the console. Used for comparing implementations
func DisplayTree(node *MATree, depth int) {
	if !node.leftChildNode.isLeafNode() {
		DisplayTree(node.leftChildNode, depth+1)
	}
	if !node.rightChildNode.isLeafNode() {
		DisplayTree(node.rightChildNode, depth+1)
	}
	return
}
