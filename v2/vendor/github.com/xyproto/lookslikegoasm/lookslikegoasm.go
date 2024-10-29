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
