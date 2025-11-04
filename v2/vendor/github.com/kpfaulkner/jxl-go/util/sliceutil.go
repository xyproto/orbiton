package util

func AddToSlice[T any](slice []*T, pos int, item *T) []*T {

	if len(slice) == 0 {
		return []*T{item}
	}

	// just add to end.
	if pos == len(slice) {
		slice = append(slice, item)
		return slice
	}

	// having to screw around copying vs just cutting slices due to we have
	// slices of pointers and we'd end up with duplicate entries no matter what I tried! :(
	tempSlice := make([]*T, len(slice)+1)
	copy(tempSlice[:pos], slice[:pos])
	tempSlice[pos] = item
	if pos < len(slice) {
		copy(tempSlice[pos+1:], slice[pos:])
	}

	return tempSlice
}
