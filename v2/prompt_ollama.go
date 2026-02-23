package main

import (
	"fmt"
	"strings"

	"github.com/xyproto/mode"
)

const (
	ollamaPreamble   = "You are an expert %s programmer with at least one PhD in Computer Science and are gifted when it comes to explaining things clearly. Be truthful and concise. If you are unsure of anything, then skip it. "
	ollamaPlainText  = " Use plain text only (no Markdown):\n\n%s"
	ollamaPseudoText = " Also explain how the instruction works by adding example pseudocode."
)

// Prompt returns the Ollama prompt used for function descriptions
func (req FunctionDescriptionRequest) Prompt() string {
	m := req.editor.mode
	lang := m.String()
	if m == mode.GoAssembly || m == mode.Assembly {
		line := strings.TrimSpace(req.funcBody)
		if strings.HasSuffix(line, ":") {
			return fmt.Sprintf(ollamaPreamble+"The following line is a label (jump target) in %s assembly, not an instruction. Briefly explain what this label name likely represents, in 1 short sentence."+ollamaPlainText, lang, lang, line)
		}
		return fmt.Sprintf(ollamaPreamble+"Describe and explain what the following %s assembly instruction does, in 1-2 short sentences."+ollamaPseudoText+ollamaPlainText, lang, lang, req.funcBody)
	}
	return fmt.Sprintf(ollamaPreamble+"Describe and explain what the following %q function in the %s programming language does, in 1-4 short sentences."+ollamaPseudoText+ollamaPlainText, lang, req.funcName, lang, req.funcBody)
}

// buildErrorExplanationPrompt builds the Ollama prompt for explaining a build error
func buildErrorExplanationPrompt(language, functionBody string, lineNumber int, lineText, compilerError string) string {
	return fmt.Sprintf(
		ollamaPreamble+"A user got this compiler error:\n\n%s\n\nThe error is on line %d:\n%s\n\nHere is the surrounding code:\n\n%s\n\nWhat is the most likely cause and fix? Focus on syntax and logic errors in the code shown. Do not suggest importing modules unless truly missing. Answer in 1-3 short sentences. Use plain text only (no Markdown, no code blocks).",
		language,
		strings.TrimSpace(compilerError),
		lineNumber,
		strings.TrimSpace(lineText),
		strings.TrimSpace(functionBody),
	)
}
