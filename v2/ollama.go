package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/mode"
	"github.com/xyproto/ollamaclient/v2"
	"github.com/xyproto/usermodel"
	"github.com/xyproto/vt"
)

// Ollama holds a model name, a boolean for if the model name was found and an Ollama client struct
type Ollama struct {
	// Ollama client, used for tab completion
	ollamaClient *ollamaclient.Config

	// The Ollama model that is used for code completion
	ModelName string

	// found a model name?
	foundModel bool
}

var (
	buildErrorExplanation         strings.Builder
	buildErrorExplanationActive   bool
	buildErrorExplanationThinking bool
	buildErrorExplanationReady    bool
	buildErrorExplanationFunction string
	buildErrorExplanationCache    = make(map[string]string)
	buildErrorExplanationMutex    sync.RWMutex
)

// NewOllama returns the pointer to an empty Ollama struct
func NewOllama() *Ollama {
	return &Ollama{}
}

// FindModel checks if a code completion model was specified either in $OLLAMA_MODEL,
// ~/.config/llm-manager/llm.conf or /etc/llm.conf.
// See https://github.com/xyproto/usermodel for more info.
// Returns true if an Ollama model name[:tag] was found.
func (cc *Ollama) FindModel() bool {
	cc.ModelName = env.Str("OLLAMA_MODEL", usermodel.GetCodeModel())
	cc.foundModel = cc.ModelName != ""
	return cc.foundModel
}

// LoadModel tries to load the cc.ModelName by using the Ollama client
func (cc *Ollama) LoadModel() error {
	if !cc.foundModel {
		return fmt.Errorf("could not find a code completion model name to use")
	}
	cc.ollamaClient = ollamaclient.New(cc.ModelName)
	cc.ollamaClient.Verbose = false
	const verbosePull = true
	if err := cc.ollamaClient.PullIfNeeded(verbosePull); err != nil {
		if ollamaHost := env.Str("OLLAMA_HOST"); ollamaHost != "" {
			return fmt.Errorf("could not fetch the %s model, check if Ollama is up and running at %s", cc.ModelName, ollamaHost)
		}
		return fmt.Errorf("could not fetch the %s model, check if Ollama is up and running locally or at $OLLAMA_HOST", cc.ModelName)
	}
	cc.ollamaClient.SetReproducible()
	return nil
}

// Loaded returns true if the ollama client could be used and the code completion model could be loaded
func (cc *Ollama) Loaded() bool {
	return cc.ollamaClient != nil
}

// GetSimpleResponse gets a simple text response from Ollama for a given prompt
func (cc *Ollama) GetSimpleResponse(prompt string) (string, error) {
	// Use the same approach as CompleteBetween but with empty end
	response, err := cc.ollamaClient.GetBetweenResponse(prompt, "")
	if err != nil {
		return "", err
	}
	return response.Response, nil
}

// clearBuildErrorExplanationState clears the build error explanation state
func clearBuildErrorExplanationState() {
	buildErrorExplanationMutex.Lock()
	buildErrorExplanationActive = false
	buildErrorExplanationThinking = false
	buildErrorExplanationReady = false
	buildErrorExplanationFunction = ""
	buildErrorExplanation.Reset()
	buildErrorExplanationMutex.Unlock()
}

// setBuildErrorExplanationPending marks that a build error explanation is being fetched
func setBuildErrorExplanationPending() {
	buildErrorExplanationMutex.Lock()
	buildErrorExplanationActive = true
	buildErrorExplanationThinking = true
	buildErrorExplanationReady = false
	buildErrorExplanationFunction = ""
	buildErrorExplanation.Reset()
	buildErrorExplanationMutex.Unlock()
}

// hasBuildErrorExplanation checks if there is an active build error explanation
func hasBuildErrorExplanation() bool {
	buildErrorExplanationMutex.RLock()
	active := buildErrorExplanationActive
	buildErrorExplanationMutex.RUnlock()
	return active
}

// hasBuildErrorExplanationThinking checks if a build error explanation is being fetched
func hasBuildErrorExplanationThinking() bool {
	buildErrorExplanationMutex.RLock()
	thinking := buildErrorExplanationThinking
	buildErrorExplanationMutex.RUnlock()
	return thinking
}

// setBuildErrorExplanation sets the build error explanation state
func setBuildErrorExplanation(functionName string, explanationText string) {
	buildErrorExplanationMutex.Lock()
	if !buildErrorExplanationActive {
		buildErrorExplanationMutex.Unlock()
		return
	}
	buildErrorExplanationActive = true
	buildErrorExplanationThinking = false
	buildErrorExplanationFunction = functionName
	buildErrorExplanationReady = true
	buildErrorExplanation.Reset()
	buildErrorExplanation.WriteString(strings.TrimSpace(sanitizeOllamaText(explanationText)))
	buildErrorExplanationMutex.Unlock()
}

func buildErrorExplanationCacheKey(sourceText, functionBody, lineText, compilerError string) string {
	return hashFunctionBody(sourceText + "\n" + functionBody + "\n" + lineText + "\n" + compilerError)
}

// trimExplanationToMaxLines trims and limits explanation text to maxLines lines
func trimExplanationToMaxLines(text string, maxLines int) string {
	trimmed := strings.TrimSpace(sanitizeOllamaText(text))
	if trimmed == "" || maxLines <= 0 {
		return ""
	}
	fields := strings.Split(trimmed, "\n")
	lines := make([]string, 0, len(fields))
	for _, line := range fields {
		line = strings.TrimSpace(line)
		lines = append(lines, line)
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n")
}

// ExplainBuildErrorWithOllama asks Ollama to explain the build error for the current function
func (e *Editor) ExplainBuildErrorWithOllama(c *vt.Canvas, err error) {
	if c == nil || err == nil || !ollama.Loaded() {
		return
	}
	keepBuildErrorExplanation := false
	defer func() {
		if !keepBuildErrorExplanation {
			clearBuildErrorExplanationState()
		}
	}()

	functionName := e.FindCurrentFunctionName()
	if functionName == "" {
		functionName = "current context"
	}

	lineIndex := e.LineIndex()
	functionBody, functionErr := e.FunctionBlock(lineIndex)
	if functionErr != nil {
		functionBody = e.Block(lineIndex)
	}
	functionBody = strings.TrimSpace(functionBody)
	if functionBody == "" || (e.mode == mode.Haskell && len(strings.Split(functionBody, "\n")) < 3) {
		// for Haskell, walk backwards to find the function start (top-level definition)
		// then include everything from there through the error line
		start := int(lineIndex)
		if e.mode == mode.Haskell {
			for i := int(lineIndex) - 1; i >= 0; i-- {
				line := e.Line(LineIndex(i))
				if strings.TrimSpace(line) == "" {
					start = i + 1
					break
				}
				start = i
			}
		} else {
			start = int(lineIndex) - 5
		}
		if start < 0 {
			start = 0
		}
		var lines []string
		for i := start; i <= int(lineIndex); i++ {
			lines = append(lines, e.Line(LineIndex(i)))
		}
		functionBody = strings.TrimSpace(strings.Join(lines, "\n"))
	}
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

	sourceText := e.String()
	cacheKey := buildErrorExplanationCacheKey(sourceText, functionBody, lineText, compilerError)

	buildErrorExplanationMutex.RLock()
	cachedExplanation, hasCached := buildErrorExplanationCache[cacheKey]
	buildErrorExplanationMutex.RUnlock()

	if hasCached {
		setBuildErrorExplanation(functionName, cachedExplanation)
		keepBuildErrorExplanation = true
		descriptionPopupDrawn = false
		e.DrawBuildErrorExplanationContinuous(c, false)
		c.HideCursorAndDraw()
		return
	}

	prompt := buildErrorExplanationPrompt(e.mode.String(), functionBody, int(lineIndex)+1, lineText, compilerError)
	explanationText, ollamaErr := ollama.GetSimpleResponse(prompt)
	if ollamaErr != nil {
		return
	}

	explanationText = strings.TrimSpace(sanitizeOllamaText(explanationText))
	if explanationText == "" {
		return
	}

	buildErrorExplanationMutex.Lock()
	buildErrorExplanationCache[cacheKey] = explanationText
	buildErrorExplanationMutex.Unlock()

	setBuildErrorExplanation(functionName, explanationText)
	keepBuildErrorExplanation = true
	descriptionPopupDrawn = false
	e.DrawBuildErrorExplanationContinuous(c, false)
	c.HideCursorAndDraw()
}

// ExplainBuildErrorWithOllamaBackground asks Ollama to explain one build error, in the background
func (e *Editor) ExplainBuildErrorWithOllamaBackground(c *vt.Canvas, err error) {
	if c == nil || err == nil || !ollama.Loaded() {
		return
	}
	setBuildErrorExplanationPending()
	e.drawFuncName.Store(true)
	e.redraw.Store(true)
	e.WriteCurrentFunctionName(c)
	c.HideCursorAndDraw()
	go e.ExplainBuildErrorWithOllama(c, err)
}

// DrawBuildErrorExplanationContinuous draws the build error explanation panel
func (e *Editor) DrawBuildErrorExplanationContinuous(c *vt.Canvas, repositionCursor bool) {
	if c == nil || !ollama.Loaded() {
		return
	}

	buildErrorExplanationMutex.RLock()
	ready := buildErrorExplanationReady
	functionName := buildErrorExplanationFunction
	descriptionText := strings.TrimSpace(buildErrorExplanation.String())
	buildErrorExplanationMutex.RUnlock()

	if !ready || descriptionText == "" {
		return
	}

	title := "Build Error"
	if functionName != "" {
		title = fmt.Sprintf("Build Error in %s", functionName)
	}
	e.drawFunctionDescriptionPopup(c, title, descriptionText, repositionCursor)
}
