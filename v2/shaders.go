package main

import (
	"errors"
	"regexp"
)

// detectShaderType detects if the shader code is a vertex shader or a fragment shader
// returns "frag", "vert" or an error
func detectShaderType(shaderCode string) (string, error) {
	var (
		fragmentCount, vertexCount int
		vertexShaderKeywords       = []string{"gl_Position", "gl_VertexID"}
		fragmentShaderKeywords     = []string{"gl_FragColor", "gl_FragCoord", "gl_FragDepth", "sampler2D", "texture"}
	)

	// Count vertex shader keywords
	for _, keyword := range vertexShaderKeywords {
		matched, _ := regexp.MatchString(keyword, shaderCode)
		if matched {
			vertexCount++
		}
	}

	// Count fragment shader keywords
	for _, keyword := range fragmentShaderKeywords {
		matched, _ := regexp.MatchString(keyword, shaderCode)
		if matched {
			fragmentCount++
		}
	}

	// Determine the shader type based on the keyword counts
	if vertexCount > fragmentCount {
		return "vert", nil
	} else if fragmentCount > vertexCount {
		return "frag", nil
	}

	return "", errors.New("unrecognized type of shader")
}
