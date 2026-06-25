package lookslikegoasm

import (
	"regexp"
	"strings"
)

var (
	goDirectivePattern      = regexp.MustCompile(`^\s*(TEXT|DATA|GLOBL|FUNCDATA|PCDATA)\b`)
	goInstructionPattern    = regexp.MustCompile(`^\s*[A-Z]{3,5}[BWLQ]\b`)
	goRegisterPattern       = regexp.MustCompile(`\b([ABCD][XHL]|R[0-9]+|SP|BP|SI|DI|FP)\b`)
	goImmediatePattern      = regexp.MustCompile(`\$\d+`)
	intelInstructionPattern = regexp.MustCompile(`^\s*(mov|add|sub|lea|push|pop|jmp|call|ret|movl|addl|subl|leal|pushl|popl|jmpl|calll)\b`)
	atntRegisterPattern     = regexp.MustCompile(`%[re][abcdsi][xip]`)
	intelRegisterPattern    = regexp.MustCompile(`\b(e?[abcd][xhl]|r[abcd][x]|r[0-9]+[bwd]?)\b`)
)

// SeveralSemicolonComments returns true if at least two non-empty lines start
// with ";" (after leading whitespace). ";" starts a comment in Intel/MASM-style
// Assembly, but not in Go/Plan9 Assembly (which uses "//"), so this is a hint
// that the source is regular Assembly rather than Go/Plan9 Assembly.
func SeveralSemicolonComments(sourceCode string) bool {
	count := 0
	for _, line := range strings.Split(sourceCode, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), ";") {
			count++
			if count >= 2 {
				return true
			}
		}
	}
	return false
}

// Consider checks if the given source code looks more like Go/Plan9 Assembly
// than Intel or AT&T style Assembly. Returns true if it looks like Go/Plan9 Assembly.
// Note that this is not an exact science!
func Consider(sourceCode string) bool {
	goCount := 0
	intelCount := 0
	for _, line := range strings.Split(sourceCode, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, ";") {
			// ";" starts a comment in Intel/MASM-style Assembly, but not in Go/Plan9 Assembly (which uses "//")
			intelCount++
			continue
		}
		if goDirectivePattern.MatchString(line) {
			goCount += 2
			continue
		}
		if goInstructionPattern.MatchString(line) {
			goCount++
			if goRegisterPattern.MatchString(line) || goImmediatePattern.MatchString(line) {
				goCount++
			}
			continue
		}
		if intelInstructionPattern.MatchString(line) {
			intelCount++
			if atntRegisterPattern.MatchString(line) || intelRegisterPattern.MatchString(line) {
				intelCount++
			}
			continue
		}
		if atntRegisterPattern.MatchString(line) || intelRegisterPattern.MatchString(line) {
			intelCount++
			continue
		}
	}
	return (goCount > 0 || intelCount > 0) && goCount > intelCount
}
