package main

import (
	"errors"
	"regexp"
	"strings"
	"sync"

	"github.com/xyproto/mode"
	"github.com/xyproto/syntax"
)

// shaderRegexpCache caches compiled regexps on first use.
var shaderRegexpCache struct {
	once     sync.Once
	patterns map[string]*regexp.Regexp
}

// getShaderRegexp returns a compiled regexp for the given pattern, using a cache.
func getShaderRegexp(pattern string) *regexp.Regexp {
	shaderRegexpCache.once.Do(func() {
		shaderRegexpCache.patterns = make(map[string]*regexp.Regexp)
		for _, keywords := range shaderKeywordPatterns {
			for _, p := range keywords {
				shaderRegexpCache.patterns[p] = regexp.MustCompile(p)
			}
		}
	})
	return shaderRegexpCache.patterns[pattern]
}

// shaderKeywordPatterns maps shader types to regexp patterns for detection.
var shaderKeywordPatterns = map[string][]string{
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

// glslIndicators are patterns that strongly suggest GLSL source inside a C string.
var glslIndicators = []string{
	"#version ", "vec2", "vec3", "vec4",
	"mat3", "mat4", "uniform ", "layout(",
	"gl_", "sampler2D",
}

// isGLSLStringLine reports whether a trimmed C string literal line contains GLSL-specific code.
func isGLSLStringLine(trimmedLine string) bool {
	if len(trimmedLine) < 2 || trimmedLine[0] != '"' {
		return false
	}
	// Strip trailing comma or semicolon for the purpose of finding the closing quote
	check := trimmedLine
	if last := check[len(check)-1]; last == ',' || last == ';' {
		check = check[:len(check)-1]
	}
	if len(check) < 2 || check[len(check)-1] != '"' {
		return false
	}
	inner := check[1 : len(check)-1]
	for _, ind := range glslIndicators {
		if strings.Contains(inner, ind) {
			return true
		}
	}
	return false
}

// isCStringLiteral reports whether the trimmed line is a C string literal (starts and ends with ",
// possibly followed by a trailing comma or semicolon).
func isCStringLiteral(trimmedLine string) bool {
	if len(trimmedLine) < 2 || trimmedLine[0] != '"' {
		return false
	}
	last := trimmedLine[len(trimmedLine)-1]
	return last == '"' ||
		((last == ',' || last == ';') && len(trimmedLine) >= 3 && trimmedLine[len(trimmedLine)-2] == '"')
}

// detectShaderType tries to detect which type of shader the given source code could be
func detectShaderType(shaderCode string) (string, error) {
	detectedTypes := make(map[string]int)

	// Check for shader type keywords using cached compiled regexps
	for shaderType, keywords := range shaderKeywordPatterns {
		for _, pattern := range keywords {
			if re := getShaderRegexp(pattern); re != nil && re.MatchString(shaderCode) {
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

// hasGLSLContent reports whether a line (without surrounding quotes) contains GLSL-specific code.
func hasGLSLContent(inner string) bool {
	for _, ind := range glslIndicators {
		if strings.Contains(inner, ind) {
			return true
		}
	}
	return false
}

// canHaveShaderStrings reports whether the given mode is one where shader code may be embedded in strings.
func canHaveShaderStrings(m mode.Mode) bool {
	return m == mode.Arduino || m == mode.C || m == mode.Cpp || m == mode.ObjC || m == mode.Go ||
		m == mode.ObjectPascal || m == mode.Zig || m == mode.Rust || m == mode.Odin
}

// shaderStringLines returns a set of line indices (within the given range) that belong
// to shader string blocks. A shader string block is a consecutive run of C string
// literal lines where at least one line contains GLSL indicators.
func (e *Editor) shaderStringLines(from, to LineIndex) map[LineIndex]bool {
	result := make(map[LineIndex]bool)
	i := from
	for i < to {
		trimmedLine := strings.TrimSpace(e.Line(i))
		if !isCStringLiteral(trimmedLine) {
			i++
			continue
		}
		// Found the start of a consecutive block of C string literals
		blockStart := i
		hasGLSL := false
		for i < to {
			trimmedLine = strings.TrimSpace(e.Line(i))
			if !isCStringLiteral(trimmedLine) {
				break
			}
			if isGLSLStringLine(trimmedLine) {
				hasGLSL = true
			}
			i++
		}
		// If any line in the block had GLSL, mark the entire block
		if hasGLSL {
			for li := blockStart; li < i; li++ {
				result[li] = true
			}
		}
	}
	return result
}

// highlightShaderInCString takes a line that is a C string literal (e.g. `"vec4 color;\n"`)
// and returns a syntax-highlighted version where the string content is highlighted as shader code.
func (e *Editor) highlightShaderInCString(line, trimmedLine string) string {
	// Find the first and last quote in the original line to preserve indentation
	firstQ := strings.IndexByte(line, '"')
	lastQ := strings.LastIndexByte(line, '"')
	if firstQ < 0 || lastQ <= firstQ {
		return ""
	}
	prefix := line[:firstQ]         // leading whitespace before the opening quote
	inner := line[firstQ+1 : lastQ] // string content between quotes
	suffix := line[lastQ+1:]        // anything after the closing quote (e.g. comma, semicolon)

	// Separate trailing escape sequences (like \n) from the shader code
	shaderCode := inner
	trailingEscape := ""
	if before, ok := strings.CutSuffix(shaderCode, `\n`); ok {
		shaderCode = before
		trailingEscape = `\n`
	}

	// Highlight the inner shader code
	highlighted, err := syntax.AsText([]byte(Escape(shaderCode)), mode.Shader)
	if err != nil {
		return ""
	}
	coloredInner := UnEscape(tout.DarkTags(string(highlighted)))

	// Reconstruct the line with quotes and escape sequences shown in the string color
	stringColor := syntax.DefaultTextConfig.String
	return prefix + tout.DarkTags("<"+stringColor+">"+"\"<off>") +
		coloredInner + tout.DarkTags("<"+stringColor+">"+trailingEscape+"\"<off>") + suffix
}

// highlightShaderInBacktickString takes a line that is inside a Go backtick string
// and returns a syntax-highlighted version as shader code.
func (e *Editor) highlightShaderInBacktickString(line string) string {
	highlighted, err := syntax.AsText([]byte(Escape(line)), mode.Shader)
	if err != nil {
		return ""
	}
	return UnEscape(tout.DarkTags(string(highlighted)))
}

// isPascalShaderStringLine reports whether a trimmed line is an Object Pascal string literal
// that forms part of a shader string block. Pascal shader strings look like:
// '#version 330'#10 +
// 'in vec3 vertexPosition;'#10 +
// #10 +
func isPascalShaderStringLine(trimmedLine string) bool {
	if len(trimmedLine) < 2 {
		return false
	}
	// Lines that are just #10 + or #10; (representing bare newlines in the shader)
	if strings.HasPrefix(trimmedLine, "#10") {
		rest := strings.TrimSpace(trimmedLine[3:])
		return rest == "" || rest == "+" || rest == ";" || rest == "+ " || strings.HasPrefix(rest, "+")
	}
	// Lines that start with a single quote (regular shader string content)
	if trimmedLine[0] != '\'' {
		return false
	}
	return strings.Contains(trimmedLine, "'#10") || strings.Contains(trimmedLine, "' +")
}

// isPascalGLSLStringLine reports whether a Pascal string literal line contains GLSL-specific code.
func isPascalGLSLStringLine(trimmedLine string) bool {
	if !isPascalShaderStringLine(trimmedLine) {
		return false
	}
	for _, ind := range glslIndicators {
		if strings.Contains(trimmedLine, ind) {
			return true
		}
	}
	return false
}

// pascalShaderStringLines returns a set of line indices (within the given range) that belong
// to Pascal shader string blocks. A shader string block is a consecutive run of Pascal string
// literal lines where at least one line contains GLSL indicators.
func (e *Editor) pascalShaderStringLines(from, to LineIndex) map[LineIndex]bool {
	result := make(map[LineIndex]bool)
	i := from
	for i < to {
		trimmedLine := strings.TrimSpace(e.Line(i))
		if !isPascalShaderStringLine(trimmedLine) {
			i++
			continue
		}
		blockStart := i
		hasGLSL := false
		for i < to {
			trimmedLine = strings.TrimSpace(e.Line(i))
			if !isPascalShaderStringLine(trimmedLine) {
				break
			}
			if isPascalGLSLStringLine(trimmedLine) {
				hasGLSL = true
			}
			i++
		}
		if hasGLSL {
			for li := blockStart; li < i; li++ {
				result[li] = true
			}
		}
	}
	return result
}

// highlightShaderInPascalString takes a line that is a Pascal string literal with shader code
// and returns a syntax-highlighted version.
func (e *Editor) highlightShaderInPascalString(line string) string {
	trimmed := strings.TrimSpace(line)

	// Handle bare #10 lines (blank lines in the shader)
	if strings.HasPrefix(trimmed, "#10") {
		stringColor := syntax.DefaultTextConfig.String
		return tout.DarkTags("<" + stringColor + ">" + line + "<off>")
	}

	// Extract shader code from between single quotes
	firstQ := strings.IndexByte(line, '\'')
	if firstQ < 0 {
		return ""
	}
	lastQ := strings.LastIndexByte(line, '\'')
	if lastQ <= firstQ {
		return ""
	}
	prefix := line[:firstQ]
	inner := line[firstQ+1 : lastQ]
	suffix := line[lastQ+1:]

	// Highlight the inner shader code
	highlighted, err := syntax.AsText([]byte(Escape(inner)), mode.Shader)
	if err != nil {
		return ""
	}
	coloredInner := UnEscape(tout.DarkTags(string(highlighted)))

	// Reconstruct the line with quotes shown in the string color
	stringColor := syntax.DefaultTextConfig.String
	return prefix + tout.DarkTags("<"+stringColor+">"+"'<off>") +
		coloredInner + tout.DarkTags("<"+stringColor+">"+"'<off>") + suffix
}

// isZigMultilineStringLine reports whether a trimmed line is a Zig multiline string (starts with \\).
func isZigMultilineStringLine(trimmedLine string) bool {
	return strings.HasPrefix(trimmedLine, `\\`)
}

// highlightShaderInZigString takes a line that is a Zig multiline string (starting with \\)
// and returns a syntax-highlighted version as shader code.
func (e *Editor) highlightShaderInZigString(line string) string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, `\\`) {
		return ""
	}
	// Find the \\ prefix in the original line
	before, after, _ := strings.Cut(line, `\\`)
	prefix := before
	shaderCode := after // skip the \\

	highlighted, err := syntax.AsText([]byte(Escape(shaderCode)), mode.Shader)
	if err != nil {
		return ""
	}
	coloredInner := UnEscape(tout.DarkTags(string(highlighted)))

	stringColor := syntax.DefaultTextConfig.String
	return prefix + tout.DarkTags("<"+stringColor+">"+`\\`+"<off>") + coloredInner
}

// zigShaderStringLines returns a set of line indices that belong to Zig shader multiline
// string blocks (consecutive lines starting with \\, where at least one contains GLSL).
func (e *Editor) zigShaderStringLines(from, to LineIndex) map[LineIndex]bool {
	result := make(map[LineIndex]bool)
	i := from
	for i < to {
		trimmedLine := strings.TrimSpace(e.Line(i))
		if !isZigMultilineStringLine(trimmedLine) {
			i++
			continue
		}
		blockStart := i
		hasGLSL := false
		for i < to {
			trimmedLine = strings.TrimSpace(e.Line(i))
			if !isZigMultilineStringLine(trimmedLine) {
				break
			}
			// Check inner content (after \\)
			inner := trimmedLine[2:]
			if hasGLSLContent(inner) {
				hasGLSL = true
			}
			i++
		}
		if hasGLSL {
			for li := blockStart; li < i; li++ {
				result[li] = true
			}
		}
	}
	return result
}

// rustShaderStringLines returns a set of line indices that belong to Rust shader string blocks.
// Rust uses the same "..." string literal syntax as C for shader embedding.
func (e *Editor) rustShaderStringLines(from, to LineIndex) map[LineIndex]bool {
	return e.shaderStringLines(from, to)
}
