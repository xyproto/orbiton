package gfx

import "testing"

func TestNewBatch(t *testing.T) {
	b := NewBatch(nil, nil)

	if b.mat != IM {
		t.Fatalf("unexpected matrix")
	}
}

func TestBatchClear(t *testing.T) {
	b := NewBatch(&TrianglesData{}, nil)

	b.Clear()
}

func TestBatchDraw(t *testing.T) {
	b := NewBatch(nil, nil)

	m := NewImage(32, 32)

	b.Draw(NewDrawTarget(m))
}

func TestBatchMakeTriangles(t *testing.T) {
	b := NewBatch(nil, nil)

	b.MakeTriangles(&TrianglesData{})
}

func TestBatchMakePicture(t *testing.T) {
	b := NewBatch(nil, nil)

	b.MakePicture(nil)
}

func TestBatchTrianglesDraw(t *testing.T) {
	bt := &batchTriangles{
		tri: &TrianglesData{},
		tmp: MakeTrianglesData(0),
		dst: NewBatch(&TrianglesData{}, nil),
	}

	bt.Draw()
}
