package gfx

func ExampleIntAbs() {
	Dump(
		IntAbs(10),
		IntAbs(-5),
	)

	// Output:
	// 10
	// 5
}

func ExampleIntMin() {
	Dump(
		IntMin(1, 2),
		IntMin(2, 1),
		IntMin(-1, -2),
	)

	// Output:
	// 1
	// 1
	// -2
}

func ExampleIntMax() {
	Dump(
		IntMax(1, 2),
		IntMax(2, 1),
		IntMax(-1, -2),
	)

	// Output:
	// 2
	// 2
	// -1
}
