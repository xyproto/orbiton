package gfx

func ExampleLog() {
	Log("Foo: %d", 123)

	// Output:
	// Foo: 123
}

func ExampleDump() {
	Dump([]string{"foo", "bar"})

	// Output:
	// [foo bar]
}

func ExamplePrintf() {
	Printf("%q %.01f", "foo bar", 1.23)

	// Output:
	// "foo bar" 1.2
}
