package gfx

func ExampleVx() {
	vx := Vx(V(6, 122), ColorWhite, V(1, 1), 0.5)

	Dump(vx)

	// Output:
	// {Position:gfx.V(6.00000000, 122.00000000) Color:{R:255 G:255 B:255 A:255} Picture:gfx.V(1.00000000, 1.00000000) Intensity:0.5}
}
