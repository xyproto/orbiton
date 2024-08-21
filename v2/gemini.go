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
	geminiModel   = env.Str("GEMINI_MODEL", "gemini-1.5-flash")
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	streamEnded := make(chan struct{})
	tokenBuffer := []string{}

	go func() {
		geminiClient.SubmitToClientStreaming(ctx, func(token string) {
			if !(e.mode != mode.Markdown && strings.Contains(token, "```")) {
				// Log each token for debugging
				logf("Received token: %s", token)

				// Append token to buffer
				tokenBuffer = append(tokenBuffer, token)
				newToken(token)
			}
			if !e.generatingTokens {
				cancel()
			}
		})
		close(streamEnded)
	}()

	// Wait for the stream to end
	<-streamEnded

	// Ensure remaining tokens are processed
	for _, token := range tokenBuffer {
		newToken(token)
	}

	// Set generatingTokens to false when done
	e.generatingTokens = false

	return nil
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

	temperature := env.Float32("GEMINI_TEMPERATURE", 0.0)

	geminiClient, err := simplegemini.NewWithTimeout(geminiModel, temperature, 10*time.Second)
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
	maxTokens := 8192 - amountOfPromptTokens
	if maxTokens < 1 {
		status.SetErrorMessage("Gemini API request is too long")
		status.Show(c, e)
		return
	}

	e.generatingTokens = true
	currentLeadingWhitespace := e.LeadingWhitespaceAt(lineIndex)
	var generatedLine string

	fixLineMut.Lock()
	defer fixLineMut.Unlock()

	if err := e.GenerateTokens(geminiClient, prompt, maxTokens, temperature, func(token string) {
		lines := strings.Split(token, "\n")
		for i, line := range lines {
			if i > 0 {
				e.InsertLineBelow()
				e.pos.sy++
				generatedLine = ""
			}
			generatedLine += line
			e.SetCurrentLine(currentLeadingWhitespace + e.AddSpaceAfterComments(generatedLine))
			e.MakeConsistent()
			e.DrawLines(c, true, false)
			e.redrawCursor = true

			// Log each line insertion for debugging
			logf("Inserted line: %s", generatedLine)
		}
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

	// Final refresh to ensure all tokens are drawn
	e.MakeConsistent()
	e.DrawLines(c, true, false)
	e.redrawCursor = true
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
		}

		temperature := env.Float32("GEMINI_TEMPERATURE", 0.8)
		if generationType == generateCode || generationType == continueCode {
			temperature = env.Float32("GEMINI_TEMPERATURE", 0.0)
		}

		geminiClient, err := simplegemini.NewWithTimeout(geminiModel, temperature, 10*time.Second)
		if err != nil {
			status.SetErrorMessage("Failed to create Gemini client")
			status.Show(c, e)
			return
		}

		amountOfPromptTokens, err := geminiClient.CountTextTokens(prompt)
		if err != nil {
			status.SetErrorMessage("Failed to count tokens")
			status.Show(c, e)
			return
		}

		maxTokens := 8192 - amountOfPromptTokens
		if maxTokens < 1 {
			status.SetErrorMessage("Gemini API request is too long")
			status.Show(c, e)
			return
		}

		switch generationType {
		case generateCode:
			prompt += ". " + fmt.Sprintf(codePrompt, e.mode.String())
		case continueCode:
			prompt += ". " + fmt.Sprintf(continuePrompt, e.mode.String()) + "\n"
			startTokens := strings.Fields(e.String())
			gatherNTokens := 8192 - amountOfPromptTokens
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

		currentLeadingWhitespace := e.LeadingWhitespace()
		e.generatingTokens = true
		first := true
		var generatedLine string

		if err := e.GenerateTokens(geminiClient, prompt, maxTokens, temperature, func(token string) {
			lines := strings.Split(token, "\n")
			for i, line := range lines {
				if i > 0 {
					e.InsertLineBelow()
					e.pos.sy++
					generatedLine = ""
				}
				generatedLine += line
				e.SetCurrentLine(currentLeadingWhitespace + e.AddSpaceAfterComments(generatedLine))
				e.MakeConsistent()
				e.DrawLines(c, true, false)
				e.redrawCursor = true
				if first {
					e.DeleteCurrentLineMoveBookmark(bookmark)
					first = false
				}

				// Log each line insertion for debugging
				logf("Inserted line: %s", generatedLine)
			}
		}); err != nil {
			e.redrawCursor = true
			if !strings.Contains(err.Error(), "context") {
				e.End(c)
				status.SetError(err)
				status.Show(c, e)
				return
			}
		}

		// Final refresh to ensure all tokens are drawn
		e.MakeConsistent()
		e.DrawLines(c, true, false)
		e.redrawCursor = true

		// Ensure e.generatingTokens is set to false when done
		e.generatingTokens = false

		if first {
			status.SetMessageAfterRedraw("Nothing was generated")
		} else {
			status.SetMessageAfterRedraw("Done")
		}

		e.RedrawAtEndOfKeyLoop(c, status)
	}()
}
