package main

import (
	"fmt"
	"strings"
)

// Prompt returns the Ollama prompt used for function descriptions
func (req FunctionDescriptionRequest) Prompt() string {
	return fmt.Sprintf("You have a PhD in Computer Science and are gifted when it comes to explaining things clearly. Be truthful and concise. If you are unsure of anything, then skip it. Describe and explain what the following %q function in the %s programming language does, in 1-4 short sentences. Use plain text only (no Markdown):\n\n%s", req.funcName, req.editor.mode.String(), req.funcBody)
}

// buildErrorExplanationPrompt builds the Ollama prompt for explaining a build error
func buildErrorExplanationPrompt(language, functionBody string, lineNumber int, lineText, compilerError string) string {
	return fmt.Sprintf(
		"You are an expert %s programmer. A user got this compiler error:\n\n%s\n\nThe error is on line %d:\n%s\n\nHere is the surrounding code:\n\n%s\n\nWhat is the most likely cause and fix? Focus on syntax and logic errors in the code shown. Do not suggest importing modules unless truly missing. Answer in 1-3 short sentences. Use plain text only (no Markdown, no code blocks).",
		language,
		strings.TrimSpace(compilerError),
		lineNumber,
		strings.TrimSpace(lineText),
		strings.TrimSpace(functionBody),
	)
}
