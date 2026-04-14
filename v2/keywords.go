package main

var (
	cTypes          = []string{"bool", "char", "const", "constexpr", "double", "float", "inline", "int", "int16_t", "int32_t", "int64_t", "int8_t", "long", "short", "signed", "size_t", "static", "uint", "uint16_t", "uint32_t", "uint64_t", "uint8_t", "unsigned", "void", "volatile"}
	cCompositeTypes = []string{"enum", "struct", "union"}
	cExtensions     = []string{".c", ".cc", ".cpp", ".cxx", ".h", ".hpp", ".m", ".mm"}
	cControlFlow    = []string{"break", "case", "catch", "continue", "default", "do", "else", "for", "goto", "if", "return", "switch", "while"}
	cModifiers      = []string{"explicit", "extern", "noexcept", "override", "virtual"}
)
