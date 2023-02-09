package main

import (
	"context"
	"strings"

	"github.com/PullRequestInc/go-gpt3"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
)

// For generating code with ChatGPT
var chatAPIKey *string // can be null, to not read environment variables needlessly when starting o

// For stopping ChatGTP from generating tokens when Esc is pressed
var continueGeneratingTokens bool

// ProgrammingLanguage returns true if the current mode appears to be a programming language (and not a markup language etc)
func (e *Editor) ProgrammingLanguage() bool {
	switch e.mode {
	case mode.Blank, mode.AIDL, mode.Amber, mode.Bazel, mode.Config, mode.Doc, mode.Email, mode.Git, mode.HIDL, mode.HTML, mode.JSON, mode.Log, mode.M4, mode.ManPage, mode.Markdown, mode.Nroff, mode.PolicyLanguage, mode.ReStructured, mode.Shader, mode.SQL, mode.Text, mode.XML:
		return false
	}
	return true
}

// AIFixups adds a space after single-line comments
func (e *Editor) AIFixups(generatedLine string) string {
	singleLineComment := e.SingleLineCommentMarker()
	trimmedLine := strings.TrimSpace(generatedLine)
	if len(trimmedLine) > 2 && e.ProgrammingLanguage() && strings.HasPrefix(trimmedLine, singleLineComment) && !strings.HasPrefix(trimmedLine, singleLineComment+" ") && !strings.HasPrefix(generatedLine, "#!") {
		return strings.Replace(generatedLine, singleLineComment, singleLineComment+" ", 1)
	}
	return generatedLine
}

// GenerateTokens uses the ChatGTP API to generate text. n is the maximum number of tokens.
// The global atomic Bool "ContinueGeneratingTokens" controls when the text generation should stop.
func GenerateTokens(apiKey, prompt string, n int, newToken func(string)) error {
	client := gpt3.NewClient(apiKey)
	chatContext, cancelFunction := context.WithCancel(context.Background())
	defer cancelFunction()
	err := client.CompletionStreamWithEngine(
		chatContext,
		gpt3.TextDavinci003Engine,
		gpt3.CompletionRequest{
			Prompt:      []string{prompt},
			MaxTokens:   gpt3.IntPtr(n),
			Temperature: gpt3.Float32Ptr(0.2),
		}, func(resp *gpt3.CompletionResponse) {
			newToken(resp.Choices[0].Text)
			if !continueGeneratingTokens {
				cancelFunction()
			}
		})
	return err
}

// GenerateCode will try to generate and insert text at the corrent position in the editor, given a ChatGPT prompt
func (e *Editor) GenerateCode(c *vt100.Canvas, status *StatusBar, bookmark *Position, chatAPIKey, chatPrompt string) {
	if chatAPIKey == "" {
		status.SetErrorMessage("ChatGPT API key is empty")
		status.Show(c, e)
		return
	}

	prompt := strings.TrimSpace(strings.TrimSuffix(chatPrompt, "!"))

	status.ClearAll(c)
	status.SetMessage("Generating code...")
	status.Show(c, e)

	currentLeadingWhitespace := e.LeadingWhitespace()

	approximateAmountOfPromptTokens := len(strings.Fields(prompt))

	// TODO: Find an exact way to find the number of tokens in the prompt, from a ChatGPT point of view
	maxTokens := 4097 - (approximateAmountOfPromptTokens + 100) // The user can press Esc when there are enough tokens
	if maxTokens < 1 {
		status.SetErrorMessage("ChatGPT API request is too long")
		status.Show(c, e)
		return
	}

	continueGeneratingTokens = true
	first := true
	var generatedLine string

	if err := GenerateTokens(chatAPIKey, prompt, maxTokens, func(word string) {
		generatedLine += word
		if strings.HasSuffix(generatedLine, "\n") {
			e.SetCurrentLine(currentLeadingWhitespace + e.AIFixups(generatedLine))
			if !first {
				if !e.EmptyTrimmedLine() {
					e.InsertLineBelow()
					e.pos.sy++
				}
			} else {
				e.DeleteCurrentLineMoveBookmark(bookmark)
				first = false
			}
			generatedLine = ""
		} else {
			e.SetCurrentLine(currentLeadingWhitespace + e.AIFixups(generatedLine))
		}
		// "refresh"
		e.DrawLines(c, true, false)
	}); err != nil {
		errorMessage := err.Error()
		if !strings.Contains(errorMessage, "context") {
			e.End(c)
			status.SetError(err)
			status.Show(c, e)
			return
		}
	}
	e.End(c)

	if continueGeneratingTokens {
		if first { // Nothing was generated
			status.SetMessageAfterRedraw("Nothing was generated")
			//logf("nothing was generated for this prompt: %s\n", prompt)
		} else {
			status.SetMessageAfterRedraw("Done")
		}
	} else {
		status.SetMessageAfterRedraw("Stopped")
	}

	e.RedrawAtEndOfKeyLoop(c, status)
}