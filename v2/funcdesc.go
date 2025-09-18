package main

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/xyproto/vt"
)

// FunctionDescriptionRequest represents a queued function description request
type FunctionDescriptionRequest struct {
	funcName string
	funcBody string
	bodyHash string
	canvas   *vt.Canvas
	editor   *Editor
}

// OllamaQueue manages the queue of function description requests and processing state
type OllamaQueue struct {
	// LIFO queue system for function description requests
	stack        []FunctionDescriptionRequest
	queuedHashes map[string]bool // Track what's already queued
	mutex        sync.Mutex

	// Worker control
	workerStarted bool
	signal        chan struct{} // Signal when new items are added

	// Processing state
	isThinking         bool   // Whether Ollama is currently processing
	processingFunction string // The function currently being processed by Ollama

	// Atomic cache for Ollama responses - map from function body hash to description
	responseCache atomic.Value // stores map[string]string
}

var (
	// Function description text from Ollama
	functionDescription strings.Builder
	// Track current function for continuous descriptions
	currentDescribedFunction string // The function we have a description for
	actualCurrentFunction    string // The function the cursor is actually on
	functionDescriptionReady bool

	// Global queue instance
	queue = &OllamaQueue{
		queuedHashes: make(map[string]bool),
		signal:       make(chan struct{}, 1),
	}
)

// init initializes the atomic cache
func init() {
	queue.responseCache.Store(make(map[string]string))
}

// hashFunctionBody creates a SHA256 hash of the function body for caching
func hashFunctionBody(funcBody string) string {
	h := sha256.Sum256([]byte(funcBody))
	return fmt.Sprintf("%x", h)
}

// IsThinking returns whether Ollama is currently processing a request
func (q *OllamaQueue) IsThinking() bool {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	return q.isThinking
}

// getFromCache retrieves a cached description
func (q *OllamaQueue) getFromCache(hash string) (string, bool) {
	cached := q.responseCache.Load()
	if cached == nil {
		return "", false
	}
	cache, ok := cached.(map[string]string)
	if !ok {
		return "", false
	}
	desc, exists := cache[hash]
	return desc, exists
}

// addToCache adds a description to the cache
func (q *OllamaQueue) addToCache(hash, description string) {
	for {
		oldCached := q.responseCache.Load()
		var oldCache map[string]string
		if oldCached == nil {
			oldCache = make(map[string]string)
		} else {
			var ok bool
			oldCache, ok = oldCached.(map[string]string)
			if !ok {
				oldCache = make(map[string]string)
			}
		}

		newCache := make(map[string]string, len(oldCache)+1)
		for k, v := range oldCache {
			newCache[k] = v
		}
		newCache[hash] = description

		if oldCached == nil {
			if q.responseCache.CompareAndSwap(nil, newCache) {
				break
			}
		} else {
			if q.responseCache.CompareAndSwap(oldCached, newCache) {
				break
			}
		}
	}
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

	go func() {
		for range q.signal {
			for {
				// Get the next request from the stack (LIFO)
				q.mutex.Lock()
				if len(q.stack) == 0 {
					q.mutex.Unlock()
					break
				}
				// Pop from the end (most recent)
				req := q.stack[len(q.stack)-1]
				q.stack = q.stack[:len(q.stack)-1]
				delete(q.queuedHashes, req.bodyHash)
				q.mutex.Unlock()

				// Check cache first (in case it was added while queued)
				if cachedDescription, exists := q.getFromCache(req.bodyHash); exists {
					// Only show cached response if this is still the current function
					if actualCurrentFunction == req.funcName {
						currentDescribedFunction = req.funcName
						functionDescriptionReady = true
						functionDescription.Reset()
						functionDescription.WriteString(strings.TrimSpace(cachedDescription))
						// Description will be drawn by main redraw cycle
					}
					continue
				}

				// Set thinking state before processing
				q.mutex.Lock()
				q.isThinking = true
				q.processingFunction = req.funcName
				q.mutex.Unlock()

				// Draw ellipsis immediately when processing starts
				req.editor.WriteCurrentFunctionName(req.canvas)
				req.canvas.HideCursorAndDraw()

				// Always process the request (for caching)
				q.processRequest(req)

				// Clear thinking state after processing
				q.mutex.Lock()
				q.isThinking = false
				q.processingFunction = ""
				q.mutex.Unlock()

				// Clear ellipsis when processing is done
				req.editor.WriteCurrentFunctionName(req.canvas)
				req.canvas.HideCursorAndDraw()

				// Only process one request at a time, then check for newer ones
				break
			}
		}
	}()
}

// processRequest handles the actual Ollama request and response
func (q *OllamaQueue) processRequest(req FunctionDescriptionRequest) {
	prompt := fmt.Sprintf("You have a PhD in Computer Science and are gifted when it comes to explaning things clearly. Be truthful and consise. If you are unsure of anything, then skip it. Describe and explain what the following %q function does, in 1-5 sentences:\n\n%s", req.funcName, req.funcBody)

	if description, err := ollama.GetSimpleResponse(prompt); err == nil {
		// Cache the response
		q.addToCache(req.bodyHash, description)

		// Only update if this is still the actual current function
		if actualCurrentFunction == req.funcName && ollama.Loaded() {
			currentDescribedFunction = req.funcName
			functionDescription.Reset()
			functionDescription.WriteString(strings.TrimSpace(description))
			functionDescriptionReady = true
			//logf("Ollama response ready for %s", req.funcName)
			// Description will be drawn by main redraw cycle
		}
		// Note: thinking state is managed by the worker, not here
	} else {
		// If error and this was for current function, clear ellipsis
		if actualCurrentFunction == req.funcName && ollama.Loaded() {
			// Clear ellipsis by overwriting with space
			ellipsisX := req.canvas.Width() - 1
			req.canvas.Write(ellipsisX, 0, req.editor.Foreground, req.editor.Background, " ")
			// Redraw function name area
			req.editor.WriteCurrentFunctionName(req.canvas)
			req.canvas.HideCursorAndDraw()
		}
		// Note: thinking state is managed by the worker, not here
	}
}

// enqueue adds a function description request to the queue
func (q *OllamaQueue) enqueue(funcName, funcBody, bodyHash string, c *vt.Canvas, e *Editor) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	// Don't queue if currently being processed
	if q.processingFunction == funcName {
		return
	}

	if !q.queuedHashes[bodyHash] {
		// Add to the stack (LIFO - most recent at the end)
		q.stack = append(q.stack, FunctionDescriptionRequest{
			funcName: funcName,
			funcBody: funcBody,
			bodyHash: bodyHash,
			canvas:   c,
			editor:   e,
		})
		q.queuedHashes[bodyHash] = true

		// Signal the worker
		select {
		case q.signal <- struct{}{}:
		default:
		}
	} else {
		// Function is already in queue - move it to the top (end of slice)
		// Find the existing request and move it to the end
		for i, req := range q.stack {
			if req.bodyHash == bodyHash {
				// Remove from current position
				q.stack = append(q.stack[:i], q.stack[i+1:]...)
				// Add to the end (top of LIFO stack)
				q.stack = append(q.stack, FunctionDescriptionRequest{
					funcName: funcName,
					funcBody: funcBody,
					bodyHash: bodyHash,
					canvas:   c,
					editor:   e,
				})
				break
			}
		}

		// Signal the worker in case it's waiting
		select {
		case q.signal <- struct{}{}:
		default:
		}
	}
}

// RequestFunctionDescription requests a description for a function using the queue system
func (e *Editor) RequestFunctionDescription(funcName, funcBody string, c *vt.Canvas) {
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

	// Check cache first
	if cachedDescription, exists := queue.getFromCache(bodyHash); exists {
		// Use cached response immediately
		currentDescribedFunction = funcName
		functionDescriptionReady = true
		functionDescription.Reset()
		functionDescription.WriteString(strings.TrimSpace(cachedDescription))
		// Force immediate redraw
		e.DrawFunctionDescriptionContinuous(c, false)
		c.HideCursorAndDraw()
		return
	}

	// Update the actual current function
	actualCurrentFunction = funcName

	// Clear description if it's for a different function
	if currentDescribedFunction != funcName {
		functionDescriptionReady = false
		functionDescription.Reset()
	}

	// Add to queue using the OllamaQueue method
	queue.enqueue(funcName, funcBody, bodyHash, c, e)
}

// DrawFunctionDescriptionContinuous draws the function description panel if in continuous mode
func (e *Editor) DrawFunctionDescriptionContinuous(c *vt.Canvas, repositionCursor bool) {
	// Only show description box if we have a ready description
	if !ollama.Loaded() || currentDescribedFunction == "" || !functionDescriptionReady {
		return
	}

	// Description is ready - show it
	title := fmt.Sprintf("Function: %s", currentDescribedFunction)
	descriptionText := strings.TrimSpace(functionDescription.String())
	if len(descriptionText) == 0 {
		descriptionText = "No description available"
	}

	e.drawFunctionDescriptionPopup(c, title, descriptionText, repositionCursor)
}

// drawFunctionDescriptionPopup draws a panel with the function description text
func (e *Editor) drawFunctionDescriptionPopup(c *vt.Canvas, title, descriptionText string, repositionCursorAfterDrawing bool) {
	// Create a box the size of the entire canvas
	canvasBox := NewCanvasBox(c)

	minWidth := 40

	// Position the description panel exactly like tutorial
	descriptionBox := NewBox()
	descriptionBox.EvenLowerRightPlacement(canvasBox, minWidth)
	// Move box up 3 positions and increase height by 1
	descriptionBox.Y -= 3
	descriptionBox.H += 1
	e.redraw.Store(true)

	// Create a list box inside
	listBox := NewBox()
	listBox.FillWithMargins(descriptionBox, 2, 2)

	// Get the current theme for the description box (exactly like tutorial)
	bt := e.NewBoxTheme()
	//bt.Foreground = &e.BoxTextColor
	bt.Foreground = &e.ItalicsColor
	bt.Background = &e.BoxBackground

	// First figure out how many lines of text this will be after word wrap (like tutorial)
	const dryRun = true
	addedLines := e.DrawText(bt, c, listBox, descriptionText, dryRun)

	if addedLines > listBox.H {
		// Then adjust the box height and text position (addedLines could very well be 0)
		descriptionBox.Y -= addedLines
		descriptionBox.H += addedLines
		listBox.Y -= addedLines
	}

	// Then draw the box with the text (like tutorial but non-blocking)
	e.DrawBox(bt, c, descriptionBox)
	e.DrawTitle(bt, c, descriptionBox, title, true)
	e.DrawText(bt, c, listBox, descriptionText, false)

	// Reposition the cursor, if needed
	if repositionCursorAfterDrawing {
		e.EnableAndPlaceCursor(c)
	}
}
