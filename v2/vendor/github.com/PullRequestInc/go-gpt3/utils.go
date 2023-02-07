package gpt3

// IntPtr converts an integer to an *int as a convenience
func IntPtr(i int) *int {
	return &i
}

// Float32Ptr converts a float32 to a *float32 as a convenience
func Float32Ptr(f float32) *float32 {
	return &f
}
