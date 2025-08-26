package main

import (
	"errors"
	"regexp"
)

// detectShaderType tries to detect which type of shader the given source code could be
func detectShaderType(shaderCode string) (string, error) {

	shaderKeywords := map[string][]string{
		"vert":  {`gl_Position`, `gl_VertexID`},
		"frag":  {`gl_FragColor`, `gl_FragCoord`, `gl_FragDepth`, `sampler2D`, `texture`},
		"geom":  {`layout\s*\(\s*points\s*\|`, `layout\s*\(\s*lines\s*\|`, `layout\s*\(\s*triangles\s*\|`},
		"tesc":  {`layout\s*\(\s*vertices\s*=\s*\d+\s*\)`},
		"tese":  {`gl_TessCoord`, `gl_TessLevelInner`, `gl_TessLevelOuter`},
		"comp":  {`layout\s*\(\s*local_size_[xyz]`},
		"mesh":  {`mesh`, `gl_MeshVerticesEXT`},
		"task":  {`task`, `gl_TaskCountNV`},
		"rgen":  {`raygeneration`},
		"rint":  {`intersection`},
		"rahit": {`anyhit`},
		"rchit": {`closesthit`},
		"rmiss": {`miss`},
		"rcall": {`callable`},
	}

	detectedTypes := make(map[string]int)

	// Check for shader type keywords
	for shaderType, keywords := range shaderKeywords {
		for _, keyword := range keywords {
			matched, _ := regexp.MatchString(keyword, shaderCode)
			if matched {
				detectedTypes[shaderType]++
			}
		}
	}

	// Determine the shader type based on the keyword counts
	var maxType string
	var maxCount int
	for shaderType, count := range detectedTypes {
		if count > maxCount {
			maxType = shaderType
			maxCount = count
		}
	}

	if maxType == "" {
		return "", errors.New("unrecognized shader type")
	}

	return maxType, nil
}
