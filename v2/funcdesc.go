package main

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"

	"github.com/xyproto/vt"
)

// FunctionDescriptionRequest represents a queued function description request
type FunctionDescriptionRequest struct {
	funcName string
	funcBody string
	bodyHash string
}

// FunctionDescriptionResponse represents the result of processing a request
type FunctionDescriptionResponse struct {
	funcName    string
	description string
	bodyHash    string
	err         error
}

// OllamaQueue manages the queue of function description requests and processing state
type OllamaQueue struct {
	// Channel-based queue system
	requestChan  chan FunctionDescriptionRequest
	responseChan chan FunctionDescriptionResponse
	shutdownChan chan struct{}

	// Processing state with mutex protection
	mutex            sync.RWMutex
	isThinking       bool
	processingFunc   string
	currentFunc      string
	readyDescription string

	// Thread-safe cache using sync.Map
	responseCache sync.Map // map[string]string (hash -> description)

	// Worker control
	workerStarted bool
	shutdownOnce  sync.Once
}

var (
	// Global queue instance
	queue = &OllamaQueue{
		requestChan:  make(chan FunctionDescriptionRequest, 10),
		responseChan: make(chan FunctionDescriptionResponse, 10),
		shutdownChan: make(chan struct{}),
	}
)

// hashFunctionBody creates a SHA256 hash of the function body for caching
func hashFunctionBody(funcBody string) string {
	h := sha256.Sum256([]byte(funcBody))
	return fmt.Sprintf("%x", h)
}

// IsThinking returns whether Ollama is currently processing a request
func (q *OllamaQueue) IsThinking() bool {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return q.isThinking
}

// GetCurrentState returns the current processing state
func (q *OllamaQueue) GetCurrentState() (isThinking bool, currentFunc, readyDescription string) {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return q.isThinking, q.currentFunc, q.readyDescription
}

// SetCurrentFunction updates the current function and returns whether description is ready
func (q *OllamaQueue) SetCurrentFunction(funcName string) bool {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.currentFunc = funcName

	// If we have a ready description for this function, return true
	if funcName != "" && q.readyDescription != "" {
		return true
	}

	// Clear ready description if function changed
	if funcName == "" {
		q.readyDescription = ""
	}

	return false
}

// getFromCache retrieves a cached description
func (q *OllamaQueue) getFromCache(hash string) (string, bool) {
	value, exists := q.responseCache.Load(hash)
	if !exists {
		return "", false
	}
	desc, ok := value.(string)
	return desc, ok
}

// addToCache adds a description to the cache
func (q *OllamaQueue) addToCache(hash, description string) {
	q.responseCache.Store(hash, description)
}

// startWorker starts the background worker that processes function description requests
func (q *OllamaQueue) startWorker() {
	q.mutex.Lock()
	if q.workerStarted {
		q.mutex.Unlock()
		return
	}
	q.workerStarted = true
	q.mutex.Unlock()

	// Start request processor
	go func() {
		for {
			select {
			case req := <-q.requestChan:
				// Check cache first
				if cachedDescription, exists := q.getFromCache(req.bodyHash); exists {
					// Send cached response
					q.responseChan <- FunctionDescriptionResponse{
						funcName:    req.funcName,
						description: cachedDescription,
						bodyHash:    req.bodyHash,
					}
					continue
				}

				// Set thinking state
				q.mutex.Lock()
				q.isThinking = true
				q.processingFunc = req.funcName
				q.mutex.Unlock()

				// Process request
				prompt := fmt.Sprintf("You have a PhD in Computer Science and are gifted when it comes to explaining things clearly. Be truthful and concise. If you are unsure of anything, then skip it. Describe and explain what the following %q function does, in 1-5 sentences:\n\n%s", req.funcName, req.funcBody)

				var response FunctionDescriptionResponse
				if description, err := ollama.GetSimpleResponse(prompt); err == nil {
					// Cache the response
					q.addToCache(req.bodyHash, description)
					response = FunctionDescriptionResponse{
						funcName:    req.funcName,
						description: description,
						bodyHash:    req.bodyHash,
					}
				} else {
					response = FunctionDescriptionResponse{
						funcName: req.funcName,
						bodyHash: req.bodyHash,
						err:      err,
					}
				}

				// Clear thinking state
				q.mutex.Lock()
				q.isThinking = false
				q.processingFunc = ""
				q.mutex.Unlock()

				// Send response
				q.responseChan <- response

			case <-q.shutdownChan:
				return
			}
		}
	}()

	// Start response handler - this handles UI updates safely
	go func() {
		for {
			select {
			case resp := <-q.responseChan:
				q.mutex.Lock()
				if resp.err == nil && resp.funcName == q.currentFunc {
					q.readyDescription = strings.TrimSpace(resp.description)
				} else if resp.funcName == q.currentFunc {
					q.readyDescription = ""
				}
				q.mutex.Unlock()
			case <-q.shutdownChan:
				return
			}
		}
	}()
}

// enqueue adds a function description request to the queue
func (q *OllamaQueue) enqueue(funcName, funcBody, bodyHash string) {
	q.mutex.RLock()
	// Don't queue if currently being processed
	if q.processingFunc == funcName {
		q.mutex.RUnlock()
		return
	}
	q.mutex.RUnlock()

	// Send request to worker (non-blocking)
	req := FunctionDescriptionRequest{
		funcName: funcName,
		funcBody: funcBody,
		bodyHash: bodyHash,
	}

	select {
	case q.requestChan <- req:
		// Request queued successfully
	default:
		// Channel full, drop oldest and add new
		// Drain one request if channel is full
		select {
		case <-q.requestChan:
		default:
		}
		// Try to send again
		select {
		case q.requestChan <- req:
		default:
			// Still full, just drop the request
		}
	}
}

// RequestFunctionDescription requests a description for a function using the queue system
func (e *Editor) RequestFunctionDescription(funcName, funcBody string) {
	if !ollama.Loaded() {
		return
	}

	if funcBody == "" {
		return
	}

	// Start the queue worker if not already started
	queue.startWorker()

	// Generate hash for caching
	bodyHash := hashFunctionBody(funcBody)

	// Update current function and check if description is ready
	if queue.SetCurrentFunction(funcName) {
		// Description is already ready
		return
	}

	// Check cache first
	if cachedDescription, exists := queue.getFromCache(bodyHash); exists {
		// Update ready description immediately
		queue.mutex.Lock()
		queue.readyDescription = strings.TrimSpace(cachedDescription)
		queue.mutex.Unlock()
		return
	}

	// Add to queue
	queue.enqueue(funcName, funcBody, bodyHash)
}

// DrawFunctionDescriptionContinuous draws the function description panel if in continuous mode
func (e *Editor) DrawFunctionDescriptionContinuous(c *vt.Canvas, repositionCursor bool) {
	if !ollama.Loaded() {
		return
	}

	// Get current state
	_, currentFunc, readyDescription := queue.GetCurrentState()

	// Only show description box if we have a ready description
	if currentFunc == "" || readyDescription == "" {
		return
	}

	// Description is ready - show it
	title := fmt.Sprintf("Function: %s", currentFunc)
	e.drawFunctionDescriptionPopup(c, title, readyDescription, repositionCursor)
}

// drawFunctionDescriptionPopup draws a panel with the function description text on the right side
func (e *Editor) drawFunctionDescriptionPopup(c *vt.Canvas, title, descriptionText string, repositionCursorAfterDrawing bool) {
	// Create a box the size of the entire canvas
	canvasBox := NewCanvasBox(c)

	// Calculate right side positioning with margins
	margin := 2
	maxWidth := 50 // Maximum width for the description box
	minWidth := 30 // Minimum width

	// Calculate width based on canvas size, but enforce limits
	width := canvasBox.W / 3 // Use 1/3 of canvas width
	if width > maxWidth {
		width = maxWidth
	}
	if width < minWidth {
		width = minWidth
	}

	// Position on the right side with margin
	descriptionBox := NewBox()
	descriptionBox.X = canvasBox.W - width - margin
	descriptionBox.Y = margin + 2 // Leave space for function name at top
	descriptionBox.W = width
	descriptionBox.H = canvasBox.H - (margin * 2) - 3 // Leave margins and space for function name

	// Ensure we don't go off canvas
	if descriptionBox.X+descriptionBox.W > canvasBox.W {
		descriptionBox.X = canvasBox.W - descriptionBox.W
	}
	if descriptionBox.Y+descriptionBox.H > canvasBox.H {
		descriptionBox.H = canvasBox.H - descriptionBox.Y
	}

	e.redraw.Store(true)

	// Create a list box inside with margins
	listBox := NewBox()
	listBox.FillWithMargins(descriptionBox, 1, 2)

	// Get the current theme for the description box
	bt := e.NewBoxTheme()
	bt.Foreground = &e.ItalicsColor
	bt.Background = &e.BoxBackground

	// First figure out how many lines of text this will be after word wrap
	const dryRun = true
	addedLines := e.DrawText(bt, c, listBox, descriptionText, dryRun)

	// Adjust height if needed, but don't exceed canvas
	if addedLines > 0 && listBox.H < addedLines {
		neededHeight := addedLines + 4 // Add space for borders and title
		if descriptionBox.Y+neededHeight <= canvasBox.H {
			descriptionBox.H = neededHeight
			listBox.H = addedLines
		}
	}

	// Draw the box with the text
	e.DrawBox(bt, c, descriptionBox)
	e.DrawTitle(bt, c, descriptionBox, title, true)
	e.DrawText(bt, c, listBox, descriptionText, false)

	// Reposition the cursor, if needed
	if repositionCursorAfterDrawing {
		e.EnableAndPlaceCursor(c)
	}
}

// ExtractCompleteFunctionBody extracts the complete function body starting from the function definition line
// This ensures we get the entire function regardless of cursor position
func (e *Editor) ExtractCompleteFunctionBody(funcName string, funcDefLineIndex LineIndex) string {
	if funcName == "" {
		return ""
	}

	// For brace-based languages, find the complete function body
	if e.isBraceBasedLanguage() {
		return e.extractBraceBasedFunctionBody(funcDefLineIndex)
	}

	// For other languages, use the existing FunctionBlock method
	// but ensure we start from the function definition line
	originalPos := e.pos
	e.GoTo(funcDefLineIndex, nil, nil)
	funcBody, err := e.FunctionBlock(funcDefLineIndex)
	e.pos = originalPos // Restore original position

	if err != nil {
		return e.Block(funcDefLineIndex)
	}
	return funcBody
}

// extractBraceBasedFunctionBody extracts the complete function body for brace-based languages
func (e *Editor) extractBraceBasedFunctionBody(funcDefLineIndex LineIndex) string {
	var sb strings.Builder
	totalLines := LineIndex(e.Len())

	// Find the opening brace
	openBraceLineIndex := LineIndex(-1)
	searchLimit := funcDefLineIndex + 20
	if searchLimit > totalLines {
		searchLimit = totalLines
	}

	for i := funcDefLineIndex; i < searchLimit; i++ {
		line := e.Line(i)
		// Include all lines up to and including the opening brace
		sb.WriteString(line)
		sb.WriteRune('\n')

		if strings.Contains(line, "{") {
			openBraceLineIndex = i
			break
		}

		// Stop if we hit another function or class definition
		trimmedLine := strings.TrimSpace(line)
		if i > funcDefLineIndex && (e.LooksLikeFunctionDef(line, e.FuncPrefix()) ||
			strings.Contains(trimmedLine, "class ") || strings.Contains(trimmedLine, "interface ")) {
			break
		}
	}

	if openBraceLineIndex == -1 {
		// No opening brace found, return what we have
		return sb.String()
	}

	// Find the matching closing brace and include all content
	closeBraceLineIndex := e.findMatchingCloseBrace(openBraceLineIndex)
	if closeBraceLineIndex == -1 {
		// No matching close brace, include rest of file or until next function
		for i := openBraceLineIndex + 1; i < totalLines; i++ {
			line := e.Line(i)
			trimmedLine := strings.TrimSpace(line)

			// Stop at next top-level function or class definition
			if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") &&
				(e.LooksLikeFunctionDef(line, e.FuncPrefix()) ||
					strings.Contains(trimmedLine, "class ") || strings.Contains(trimmedLine, "interface ")) {
				break
			}

			sb.WriteString(line)
			sb.WriteRune('\n')
		}
	} else {
		// Include all lines up to and including the closing brace
		for i := openBraceLineIndex + 1; i <= closeBraceLineIndex; i++ {
			line := e.Line(i)
			sb.WriteString(line)
			sb.WriteRune('\n')
		}
	}

	return sb.String()
}
