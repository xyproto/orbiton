package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/xyproto/vt"
)

var (
	buildErrorExplanation strings.Builder

	buildErrorExplanationActive   bool
	buildErrorExplanationReady    bool
	buildErrorExplanationFunction string
	buildErrorExplanationLine     LineIndex

	buildErrorExplanationCache = make(map[string]string)
	buildErrorExplanationMutex sync.RWMutex
)

// clearBuildErrorExplanationState clears the current build error explanation state.
func clearBuildErrorExplanationState() {
	buildErrorExplanationMutex.Lock()
	buildErrorExplanationActive = false
	buildErrorExplanationReady = false
	buildErrorExplanationFunction = ""
	buildErrorExplanationLine = 0
	buildErrorExplanation.Reset()
	buildErrorExplanationMutex.Unlock()
}

// setBuildErrorExplanationPending marks that a build error explanation is being fetched.
func setBuildErrorExplanationPending() {
	buildErrorExplanationMutex.Lock()
	buildErrorExplanationActive = true
	buildErrorExplanationReady = false
	buildErrorExplanationFunction = ""
	buildErrorExplanationLine = 0
	buildErrorExplanation.Reset()
	buildErrorExplanationMutex.Unlock()
}

// hasBuildErrorExplanation checks if there is an active build error explanation.
func hasBuildErrorExplanation() bool {
	buildErrorExplanationMutex.RLock()
	active := buildErrorExplanationActive
	buildErrorExplanationMutex.RUnlock()
	return active
}

// setBuildErrorExplanation sets the current build error explanation state.
func setBuildErrorExplanation(functionName string, lineIndex LineIndex, explanationText string) {
	buildErrorExplanationMutex.Lock()
	buildErrorExplanationActive = true
	buildErrorExplanationFunction = functionName
	buildErrorExplanationLine = lineIndex
	buildErrorExplanationReady = true
	buildErrorExplanation.Reset()
	buildErrorExplanation.WriteString(strings.TrimSpace(explanationText))
	buildErrorExplanationMutex.Unlock()
}

// buildErrorExplanationPrompt builds the Ollama prompt for explaining a build error.
func buildErrorExplanationPrompt(functionBody string, lineNumber int, lineText, compilerError string) string {
	return fmt.Sprintf(
		"For this function:\n\n%s\n\nThe user is currently looking at line %d:\n%s\n\nExplain to the user what should be done in order to resolve and/or understand this error:\n\n%s\n\nKeep it brief, but enlightening. Assume the user is an expert, but just forgot something. Use at most 5 short lines.\n\nYou are an expert programmer.",
		strings.TrimSpace(functionBody),
		lineNumber,
		strings.TrimSpace(lineText),
		strings.TrimSpace(compilerError),
	)
}

// trimExplanationToMaxLines trims and limits explanation text to maxLines lines.
func trimExplanationToMaxLines(text string, maxLines int) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || maxLines <= 0 {
		return ""
	}
	fields := strings.Split(trimmed, "\n")
	lines := make([]string, 0, len(fields))
	for _, line := range fields {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n")
}

// ExplainBuildErrorWithOllama asks Ollama to explain the current build error for the current function.
func (e *Editor) ExplainBuildErrorWithOllama(c *vt.Canvas, err error) {
	if c == nil || err == nil || !ollama.Loaded() || !buildErrorJumpedToSource(err) {
		return
	}
	setBuildErrorExplanationPending()
	keepBuildErrorExplanation := false
	defer func() {
		if !keepBuildErrorExplanation {
			clearBuildErrorExplanationState()
		}
	}()

	functionName := e.FindCurrentFunctionName()
	if functionName == "" {
		return
	}

	lineIndex := e.LineIndex()
	functionBody, functionErr := e.FunctionBlock(lineIndex)
	if functionErr != nil {
		functionBody = e.Block(lineIndex)
	}
	functionBody = strings.TrimSpace(functionBody)
	if functionBody == "" {
		return
	}

	lineText := strings.TrimSpace(e.Line(lineIndex))
	if lineText == "" {
		lineText = "<empty line>"
	}

	compilerError := strings.TrimSpace(err.Error())
	if compilerError == "" {
		return
	}

	cacheKey := hashFunctionBody(functionBody + "\n" + lineText + "\n" + compilerError)

	buildErrorExplanationMutex.RLock()
	cachedExplanation, hasCached := buildErrorExplanationCache[cacheKey]
	buildErrorExplanationMutex.RUnlock()

	if hasCached {
		setBuildErrorExplanation(functionName, lineIndex, cachedExplanation)
		keepBuildErrorExplanation = true
		e.DrawBuildErrorExplanationContinuous(c, false)
		c.HideCursorAndDraw()
		return
	}

	prompt := buildErrorExplanationPrompt(functionBody, int(lineIndex)+1, lineText, compilerError)
	explanationText, ollamaErr := ollama.GetSimpleResponse(prompt)
	if ollamaErr != nil {
		return
	}

	explanationText = trimExplanationToMaxLines(explanationText, 5)
	if explanationText == "" {
		return
	}

	if e.LineIndex() != lineIndex || e.FindCurrentFunctionName() != functionName {
		return
	}

	buildErrorExplanationMutex.Lock()
	buildErrorExplanationCache[cacheKey] = explanationText
	buildErrorExplanationMutex.Unlock()

	setBuildErrorExplanation(functionName, lineIndex, explanationText)
	keepBuildErrorExplanation = true
	e.DrawBuildErrorExplanationContinuous(c, false)
	c.HideCursorAndDraw()
}

// ExplainBuildErrorWithOllamaBackground asks Ollama to explain one build error, in the background.
func (e *Editor) ExplainBuildErrorWithOllamaBackground(c *vt.Canvas, err error) {
	if c == nil || err == nil || !ollama.Loaded() || !buildErrorJumpedToSource(err) {
		return
	}
	setBuildErrorExplanationPending()
	go e.ExplainBuildErrorWithOllama(c, err)
}

// DrawBuildErrorExplanationContinuous draws the current build error explanation panel.
func (e *Editor) DrawBuildErrorExplanationContinuous(c *vt.Canvas, repositionCursor bool) {
	if c == nil || !ollama.Loaded() {
		return
	}

	buildErrorExplanationMutex.RLock()
	ready := buildErrorExplanationReady
	functionName := buildErrorExplanationFunction
	lineIndex := buildErrorExplanationLine
	descriptionText := strings.TrimSpace(buildErrorExplanation.String())
	buildErrorExplanationMutex.RUnlock()

	if !ready || functionName == "" || descriptionText == "" {
		return
	}

	if e.LineIndex() != lineIndex || e.FindCurrentFunctionName() != functionName {
		clearBuildErrorExplanationState()
		return
	}

	title := fmt.Sprintf("Build Error in %s", functionName)
	e.drawFunctionDescriptionPopup(c, title, descriptionText, repositionCursor)
}
