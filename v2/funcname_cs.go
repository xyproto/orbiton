package main

import "strings"

// csStarters lists the tokens that can begin a C# method or constructor definition.
var csStarters = []string{
	"public ", "private ", "protected ", "internal ",
	"static ", "override ", "virtual ", "abstract ", "sealed ", "async ", "partial ", "new ", "extern ",
	"void ", "int ", "uint ", "long ", "ulong ", "short ", "ushort ",
	"string ", "bool ", "double ", "float ", "decimal ",
	"byte ", "sbyte ", "char ", "object ", "dynamic ",
}

// csLooksLikeFunctionDef reports whether the trimmed line looks like a C# method
// or constructor definition.
func csLooksLikeFunctionDef(line string) bool {
	if !strings.Contains(line, "(") {
		return false
	}
	// Lambdas and abstract/interface method declarations are not bodies.
	if strings.Contains(line, "=>") || strings.HasSuffix(line, ";") {
		return false
	}
	for _, starter := range csStarters {
		if strings.HasPrefix(line, starter) {
			return true
		}
	}
	return false
}
