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
	canvas   *vt.Canvas
	editor   *Editor
	funcName string
	funcBody string
	bodyHash string
}

var (
	// Function description text from Ollama
	functionDescription strings.Builder

	// Track current function for continuous descriptions
	currentDescribedFunction    string // The function we have a description for
	actualCurrentFunction       string // The function the cursor is actually on
	functionDescriptionReady    bool
	functionDescriptionThinking bool
	processingFunction          string // The function currently being processed by Ollama

	// Cache for Ollama responses - map from function body hash to description
	ollamaResponseCache = make(map[string]string)

	// Mutex for both cache operations and ensuring only one Ollama request at a time
	ollamaMutex sync.RWMutex

	// LIFO queue system for function description requests
	descriptionStack   []FunctionDescriptionRequest
	queuedHashes       = make(map[string]bool) // Track what's already queued
	queueMutex         sync.Mutex
	queueWorkerStarted bool
	queueSignal        = make(chan struct{}, 1) // Signal when new items are added
)

// hashFunctionBody creates a SHA256 hash of the function body for caching
func hashFunctionBody(funcBody string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(funcBody)))
}

// startQueueWorker starts the background worker that processes function description requests
func startQueueWorker() {
	queueMutex.Lock()
	if queueWorkerStarted {
		queueMutex.Unlock()
		return
	}
	queueWorkerStarted = true
	queueMutex.Unlock()

	go func() {
		for range queueSignal {
			for {
				// Get the next request from the stack (LIFO)
				queueMutex.Lock()
				if len(descriptionStack) == 0 {
					queueMutex.Unlock()
					break
				}
				// Pop from the end (most recent)
				req := descriptionStack[len(descriptionStack)-1]
				descriptionStack = descriptionStack[:len(descriptionStack)-1]
				delete(queuedHashes, req.bodyHash)
				queueMutex.Unlock()

				// Check cache first (in case it was added while queued)
				ollamaMutex.RLock()
				if cachedDescription, exists := ollamaResponseCache[req.bodyHash]; exists {
					ollamaMutex.RUnlock()
					// Only show cached response if this is still the current function
					if actualCurrentFunction == req.funcName {
						currentDescribedFunction = req.funcName
						functionDescriptionReady = true
						functionDescriptionThinking = false
						functionDescription.Reset()
						functionDescription.WriteString(strings.TrimSpace(cachedDescription))
						// Force immediate redraw
						req.editor.DrawFunctionDescriptionContinuous(req.canvas, false)
						req.canvas.HideCursorAndDraw()
					}
					continue
				}
				ollamaMutex.RUnlock()

				// Set thinking state before processing
				functionDescriptionThinking = true
				processingFunction = req.funcName

				// Draw ellipsis immediately when starting to process
				req.editor.WriteCurrentFunctionName(req.canvas)
				req.canvas.HideCursorAndDraw()

				// Process the request
				processDescriptionRequest(req)

				// Clear thinking state after processing
				functionDescriptionThinking = false
				processingFunction = ""

				// Clear ellipsis when processing is done
				req.editor.WriteCurrentFunctionName(req.canvas)
				req.canvas.HideCursorAndDraw()

				// Only process one request at a time, then check for newer ones
				break
			}
		}
	}()
}

// processDescriptionRequest handles the actual Ollama request and response
func processDescriptionRequest(req FunctionDescriptionRequest) {
	prompt := fmt.Sprintf("You have a PhD in Computer Science and are gifted when it comes to explaning things clearly. Be truthful and consise. If you are unsure of anything, then skip it. Describe and explain what the following %q function does, in 1-5 sentences:\n\n%s", req.funcName, req.funcBody)

	if description, err := ollama.GetSimpleResponse(prompt); err == nil {
		// Cache the response
		ollamaMutex.Lock()
		ollamaResponseCache[req.bodyHash] = description
		ollamaMutex.Unlock()

		// Only update and display if this is still the actual current function
		if actualCurrentFunction == req.funcName && ollama.Loaded() {
			currentDescribedFunction = req.funcName
			functionDescription.Reset()
			functionDescription.WriteString(strings.TrimSpace(description))
			functionDescriptionReady = true
			//logf("Ollama response ready for %s, drawing description box", req.funcName)
			// Clear ellipsis by overwriting with space
			ellipsisX := req.canvas.Width() - 1
			req.canvas.Write(ellipsisX, 0, req.editor.Foreground, req.editor.Background, " ")
			// Redraw function name area and description box
			req.editor.WriteCurrentFunctionName(req.canvas)
			req.editor.DrawFunctionDescriptionContinuous(req.canvas, false)
			req.canvas.HideCursorAndDraw()
		}
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
	startQueueWorker()

	// Generate hash for caching
	bodyHash := hashFunctionBody(funcBody)

	// Check cache first
	ollamaMutex.RLock()
	if cachedDescription, exists := ollamaResponseCache[bodyHash]; exists {
		ollamaMutex.RUnlock()
		// Use cached response immediately
		currentDescribedFunction = funcName
		functionDescriptionReady = true
		functionDescriptionThinking = false
		functionDescription.Reset()
		functionDescription.WriteString(strings.TrimSpace(cachedDescription))
		// Force immediate redraw
		e.DrawFunctionDescriptionContinuous(c, false)
		c.HideCursorAndDraw()
		return
	}
	ollamaMutex.RUnlock()

	// Update the actual current function
	actualCurrentFunction = funcName

	// Clear description if it's for a different function
	if currentDescribedFunction != funcName {
		functionDescriptionReady = false
		functionDescription.Reset()
	}

	// Add to queue or move to top if already queued
	queueMutex.Lock()

	// Don't queue if currently being processed
	if processingFunction == funcName {
		queueMutex.Unlock()
		return
	}

	if !queuedHashes[bodyHash] {
		// Add to the stack (LIFO - most recent at the end)
		descriptionStack = append(descriptionStack, FunctionDescriptionRequest{
			funcName: funcName,
			funcBody: funcBody,
			bodyHash: bodyHash,
			canvas:   c,
			editor:   e,
		})
		queuedHashes[bodyHash] = true

		// Signal the worker
		select {
		case queueSignal <- struct{}{}:
		default:
		}
	} else {
		// Function is already in queue - move it to the top (end of slice)
		// Find the existing request and move it to the end
		for i, req := range descriptionStack {
			if req.bodyHash == bodyHash {
				// Remove from current position
				descriptionStack = append(descriptionStack[:i], descriptionStack[i+1:]...)
				// Add to the end (top of LIFO stack)
				descriptionStack = append(descriptionStack, FunctionDescriptionRequest{
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
		case queueSignal <- struct{}{}:
		default:
		}
	}
	queueMutex.Unlock()
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
	maxHeightPercent := 0.8 // Use up to 80% of canvas height

	// Calculate maximum height based on canvas
	maxHeight := int(float64(canvasBox.H) * maxHeightPercent)

	// Position the description panel near the top instead of bottom
	descriptionBox := NewBox()
	descriptionBox.EvenLowerRightPlacement(canvasBox, minWidth)

	// Start from the top (line 2 to leave room for function name)
	descriptionBox.Y = 2
	descriptionBox.H = maxHeight
	e.redraw.Store(true)

	// Create a list box inside with margins
	listBox := NewBox()
	listBox.FillWithMargins(descriptionBox, 2, 2)

	// Get the current theme for the description box
	bt := e.NewBoxTheme()
	bt.Foreground = &e.ItalicsColor
	bt.Background = &e.BoxBackground

	// First figure out how many lines of text this will be after word wrap
	const dryRun = true
	addedLines := e.DrawText(bt, c, listBox, descriptionText, dryRun)

	// Adjust box height to fit content, but don't exceed maxHeight
	if addedLines > 0 && addedLines < listBox.H {
		// Content fits - shrink box to fit content (add 1 line margin at bottom)
		heightDiff := listBox.H - addedLines
		descriptionBox.H -= (heightDiff - 1)
		listBox.H -= (heightDiff - 1)
	}
	// If addedLines > listBox.H, keep the box at maxHeight (scrolling would be needed)

	// Draw the box with the text
	e.DrawBox(bt, c, descriptionBox)
	e.DrawTitle(bt, c, descriptionBox, title, true)
	e.DrawText(bt, c, listBox, descriptionText, false)

	// Reposition the cursor, if needed
	if repositionCursorAfterDrawing {
		e.EnableAndPlaceCursor(c)
	}
}
