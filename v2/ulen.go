package main

func ulen[T string | []rune | []string](xs T) uint {
	return uint(len(xs))
}
