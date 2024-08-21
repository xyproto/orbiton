package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/mode"
	"github.com/xyproto/simplegemini"
	"github.com/xyproto/vt100"
)

const (
	codePrompt     = "Write it in %s and include comments where it makes sense. The code should be concise, correct, and expertly created. Comments above functions should start with the function name."
	continuePrompt = "Write the next 10 lines of this %s program:\n"
	textPrompt     = "Write it in %s. It should be expertly written, concise, and correct."
)

var (
	fixLineMut    sync.Mutex
	geminiEnabled = env.Has("GCP_PROJECT") || env.Has("PROJECT_ID")
)

func (e *Editor) ProgrammingLanguage() bool {
	switch e.mode {
	case mode.AIDL, mode.ASCIIDoc, mode.Amber, mode.Bazel, mode.Blank, mode.Config, mode.Email, mode.Git, mode.HIDL, mode.HTML, mode.JSON, mode.Log, mode.M4, mode.ManPage, mode.Markdown, mode.Nroff, mode.PolicyLanguage, mode.ReStructured, mode.SCDoc, mode.SQL, mode.Shader, mode.Text, mode.XML:
		return false
	}
	return true
}

func (e *Editor) AddSpaceAfterComments(generatedLine string) string {
	singleLineComment := e.SingleLineCommentMarker()
	trimmedLine := strings.TrimSpace(generatedLine)
	if len(trimmedLine) > 2 && e.ProgrammingLanguage() &&
		strings.HasPrefix(trimmedLine, singleLineComment) &&
		!strings.HasPrefix(trimmedLine, singleLineComment+" ") &&
		!strings.HasPrefix(generatedLine, "#!") {
		return strings.Replace(generatedLine, singleLineComment, singleLineComment+" ", 1)
	}
	return generatedLine
}

func (e *Editor) GenerateTokens(geminiClient *simplegemini.GeminiClient, prompt string, n int, temperature float32, newToken func(string)) error {
	if geminiClient == nil {
		return errors.New("no Gemini client")
	}
	ctx, cancel := context.WithTimeout(context.Background(), geminiClient.Timeout)
	defer cancel()

	_, err := geminiClient.SubmitToClientStreaming(ctx, func(token string) {
		newToken(token)
		if !e.generatingTokens {
			cancel()
		}
	})
	return err
}

func (e *Editor) FixLine(c *vt100.Canvas, status *StatusBar, lineIndex LineIndex, disableFixAsYouTypeOnError bool) {
	if !geminiEnabled {
		status.SetErrorMessage("Gemini is not enabled")
		status.Show(c, e)
		return
	}

	line := e.Line(lineIndex)
	if strings.TrimSpace(line) == "" {
		return
	}

	temperature := float32(0.0)

	geminiClient, err := simplegemini.NewWithTimeout("gemini-1.5-pro", temperature, 10*time.Second)
	if err != nil {
		status.SetErrorMessage("Failed to create Gemini client")
		status.Show(c, e)
		return
	}

	prompt := fmt.Sprintf("Make as few changes as possible to this line of %s in order to correct any typos or obvious grammatical errors, but only output EITHER the exact same line OR the corrected line! Here it is: %s", e.mode.String(), line)

	geminiClient.ClearParts()
	geminiClient.AddText(prompt)

	amountOfPromptTokens, err := geminiClient.CountTextTokens(prompt)
	if err != nil {
		status.SetErrorMessage("Failed to count tokens")
		status.Show(c, e)
		return
	}
	maxTokens := 16000 - amountOfPromptTokens
	if maxTokens < 1 {
		status.SetErrorMessage("Gemini API request is too long")
		status.Show(c, e)
		return
	}

	e.generatingTokens = true
	currentLeadingWhitespace := e.LeadingWhitespaceAt(lineIndex)
	var generatedLine, newContents, newTrimmedContents string

	fixLineMut.Lock()
	defer fixLineMut.Unlock()

	if err := e.GenerateTokens(geminiClient, prompt, maxTokens, temperature, func(word string) {
		generatedLine = strings.TrimSpace(generatedLine) + word
		newTrimmedContents = e.AddSpaceAfterComments(generatedLine)
		newContents = currentLeadingWhitespace + newTrimmedContents
	}); err != nil {
		e.redrawCursor = true
		if !strings.Contains(err.Error(), "context") {
			e.End(c)
			status.SetError(err)
			status.Show(c, e)
			if disableFixAsYouTypeOnError {
				e.fixAsYouType = false
			}
			return
		}
	}

	if e.TrimmedLineAt(lineIndex) != newTrimmedContents {
		e.SetLine(lineIndex, newContents)
	}
}

func (e *Editor) FixCodeOrText(c *vt100.Canvas, status *StatusBar, disableFixAsYouTypeOnError bool) {
	if !geminiEnabled {
		status.SetErrorMessage("Gemini is not enabled")
		status.Show(c, e)
		return
	}
	go e.FixLine(c, status, e.DataY(), disableFixAsYouTypeOnError)
}

func (e *Editor) GenerateCodeOrText(c *vt100.Canvas, status *StatusBar, bookmark *Position) {
	if !geminiEnabled {
		status.SetErrorMessage("Gemini is not enabled")
		status.Show(c, e)
		return
	}

	geminiClient, err := simplegemini.NewWithTimeout("gemini-1.5-pro", 0.8, 10*time.Second)
	if err != nil {
		status.SetErrorMessage("Failed to create Gemini client")
		status.Show(c, e)
		return
	}

	trimmedLine := e.TrimmedLine()

	go func() {
		prompt := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(trimmedLine, e.SingleLineCommentMarker()), "!"))

		const (
			generateText = iota
			generateCode
			continueCode
		)

		generationType := generateText
		if e.ProgrammingLanguage() {
			generationType = generateCode
			if prompt == "" {
				generationType = continueCode
			}
		}

		temperature := env.Float32("GEMINI_TEMPERATURE", 0.8)

		amountOfPromptTokens, err := geminiClient.CountTextTokens(prompt)
		if err != nil {
			status.SetErrorMessage("Failed to count tokens")
			status.Show(c, e)
			return
		}

		switch generationType {
		case generateCode:
			prompt += ". " + fmt.Sprintf(codePrompt, e.mode.String())
		case continueCode:
			prompt += ". " + fmt.Sprintf(continuePrompt, e.mode.String()) + "\n"
			startTokens := strings.Fields(e.String())
			gatherNTokens := 16000 - amountOfPromptTokens
			if len(startTokens) > gatherNTokens {
				startTokens = startTokens[len(startTokens)-gatherNTokens:]
			}
			prompt += strings.Join(startTokens, " ")
		case generateText:
			prompt += ". " + fmt.Sprintf(textPrompt, e.mode.String())
		}

		geminiClient.ClearParts()
		geminiClient.AddText(prompt)

		status.ClearAll(c)
		switch generationType {
		case generateText:
			status.SetMessage("Generating text...")
		case generateCode:
			status.SetMessage("Generating code...")
		case continueCode:
			status.SetMessage("Continuing code...")
		}
		status.Show(c, e)

		maxTokens := 16000 - amountOfPromptTokens
		if maxTokens < 1 {
			status.SetErrorMessage("Gemini API request is too long")
			status.Show(c, e)
			return
		}

		currentLeadingWhitespace := e.LeadingWhitespace()
		e.generatingTokens = true
		first := true
		var generatedLine string

		if err := e.GenerateTokens(geminiClient, prompt, maxTokens, temperature, func(word string) {
			generatedLine += word
			if strings.HasSuffix(generatedLine, "\n") {
				newContents := currentLeadingWhitespace + e.AddSpaceAfterComments(generatedLine)
				e.SetCurrentLine(newContents)
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
				e.SetCurrentLine(currentLeadingWhitespace + e.AddSpaceAfterComments(generatedLine))
			}
			e.MakeConsistent()
			e.DrawLines(c, true, false)
			e.redrawCursor = true
		}); err != nil {
			e.redrawCursor = true
			if !strings.Contains(err.Error(), "context") {
				e.End(c)
				status.SetError(err)
				status.Show(c, e)
				return
			}
		}
		e.End(c)

		if e.generatingTokens {
			if first {
				status.SetMessageAfterRedraw("Nothing was generated")
			} else {
				status.SetMessageAfterRedraw("Done")
			}
		} else {
			status.SetMessageAfterRedraw("Stopped")
		}

		e.RedrawAtEndOfKeyLoop(c, status)
	}()
}
