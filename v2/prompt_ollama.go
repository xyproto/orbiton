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
func buildErrorExplanationPrompt(functionBody string, lineNumber int, lineText, compilerError string) string {
	return fmt.Sprintf(
		"For this function:\n\n%s\n\nThe user is currently looking at line %d:\n%s\n\nExplain to the user what should be done in order to resolve and/or understand this error:\n\n%s\n\nKeep it brief, but enlightening. Assume the user is an expert, but just forgot something. Use at most 4 short lines. Use plain text only (no Markdown).\n\nYou are an expert programmer.",
		strings.TrimSpace(functionBody),
		lineNumber,
		strings.TrimSpace(lineText),
		strings.TrimSpace(compilerError),
	)
}
