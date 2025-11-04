package util

// Deque double ended queue to mimic Java version (or at least the functionality I need).
type Deque[T any] struct {
	data []T
}

func NewDeque[T any]() *Deque[T] {
	dq := &Deque[T]{}

	return dq
}

func (dq *Deque[T]) AddFirst(item T) {
	dq.data = append([]T{item}, dq.data...)
}

func (dq *Deque[T]) AddLast(item T) {
	dq.data = append(dq.data, item)
}

func (dq *Deque[T]) RemoveFirst() *T {
	if len(dq.data) == 0 {
		return nil
	}

	item := dq.data[0]
	dq.data = dq.data[1:]
	return &item
}

func (dq *Deque[T]) RemoveLast() *T {
	if len(dq.data) == 0 {
		return nil
	}

	item := dq.data[len(dq.data)-1]
	dq.data = dq.data[:len(dq.data)-1]
	return &item
}

func (dq *Deque[T]) IsEmpty() bool {
	return len(dq.data) == 0
}
