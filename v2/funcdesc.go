package main

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"

	"github.com/xyproto/vt"
	"github.com/xyproto/wordwrap"
)

// FunctionDescriptionRequest holds one queued function description request
type FunctionDescriptionRequest struct {
	canvas   *vt.Canvas
	editor   *Editor
	funcName string
	funcBody string
	bodyHash string
	start    LineIndex
	end      LineIndex
	hasRange bool
}

var (
	// Function description text from Ollama
	functionDescription strings.Builder

	// Track current function description state
	currentDescribedFunction     string
	currentDescribedFunctionFrom LineIndex
	currentDescribedFunctionTo   LineIndex
	currentDescribedFunctionSpan bool
	actualCurrentFunction        string
	functionDescriptionReady     bool
	functionDescriptionThinking  bool
	processingFunction           string
	functionDescriptionDismissed bool
	dismissedFunctionDescription string
	functionDescriptionsDisabled bool
	descriptionBoxY              int
	descriptionBoxYFunction      string
	descriptionBoxYBoxHeight     int
	descriptionBoxYCanvasHeight  int
	descriptionBoxYLocked        bool

	// Cache Ollama responses by function body hash
	ollamaResponseCache = make(map[string]string)

	// Lock cache operations and limit Ollama processing to one request
	ollamaMutex sync.RWMutex

	// LIFO queue for function description requests
	descriptionStack   []FunctionDescriptionRequest
	queuedHashes       = make(map[string]bool)
	queueMutex         sync.Mutex
	queueWorkerStarted bool
	queueSignal        = make(chan struct{}, 1)
)

// functionDescriptionsAllowed checks if function descriptions can be requested and shown.
func functionDescriptionsAllowed() bool {
	return !functionDescriptionsDisabled && !hasBuildErrorExplanation()
}

// sanitizeOllamaText replaces code fence lines with blank lines.
func sanitizeOllamaText(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			lines[i] = ""
		}
	}
	return strings.Join(lines, "\n")
}

// wrappedLineCount returns how many visual lines DrawText would use for this text and width.
func wrappedLineCount(text string, maxWidth int) int {
	if maxWidth <= 0 {
		return 0
	}
	text = asciiFallback(text)
	totalLines := 0
	for line := range strings.SplitSeq(text, "\n") {
		if strings.TrimSpace(line) == "" {
			totalLines++
			continue
		}
		wrappedLines, err := wordwrap.WordWrap(line, maxWidth)
		if err != nil || len(wrappedLines) == 0 {
			totalLines++
			continue
		}
		totalLines += len(wrappedLines)
	}
	return totalLines
}

// hashFunctionBody returns a SHA256 hash of the function body
func hashFunctionBody(funcBody string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(funcBody)))
}

// clearFunctionDescriptionQueue clears pending description requests
func clearFunctionDescriptionQueue() {
	queueMutex.Lock()
	descriptionStack = descriptionStack[:0]
	for bodyHash := range queuedHashes {
		delete(queuedHashes, bodyHash)
	}
	queueMutex.Unlock()
}

// clearFunctionDescriptionState resets function description state
func clearFunctionDescriptionState() {
	currentDescribedFunction = ""
	currentDescribedFunctionFrom = 0
	currentDescribedFunctionTo = 0
	currentDescribedFunctionSpan = false
	actualCurrentFunction = ""
	functionDescriptionReady = false
	functionDescriptionThinking = false
	descriptionBoxY = 0
	descriptionBoxYFunction = ""
	descriptionBoxYBoxHeight = 0
	descriptionBoxYCanvasHeight = 0
	descriptionBoxYLocked = false
	functionDescription.Reset()
}

// DisableFunctionDescriptionsAfterBuildError turns off function descriptions after a build failure.
func DisableFunctionDescriptionsAfterBuildError() {
	functionDescriptionsDisabled = true
	clearFunctionDescriptionState()
	clearFunctionDescriptionQueue()
}

// EnableFunctionDescriptions turns function descriptions back on after build error mode.
func EnableFunctionDescriptions() {
	functionDescriptionsDisabled = false
}

func setCurrentDescribedFunction(functionName string, start, end LineIndex, hasRange bool) {
	if hasRange && end < start {
		end = start
	}

	functionChanged := currentDescribedFunction != functionName
	spanChanged := currentDescribedFunctionSpan != hasRange
	if hasRange {
		spanChanged = spanChanged || currentDescribedFunctionFrom != start || currentDescribedFunctionTo != end
	}

	currentDescribedFunction = functionName
	currentDescribedFunctionSpan = hasRange
	if hasRange {
		currentDescribedFunctionFrom = start
		currentDescribedFunctionTo = end
	} else {
		currentDescribedFunctionFrom = 0
		currentDescribedFunctionTo = 0
	}

	// Reset popup placement when the function or range changes
	if functionChanged || spanChanged {
		descriptionBoxY = 0
		descriptionBoxYFunction = ""
		descriptionBoxYBoxHeight = 0
		descriptionBoxYCanvasHeight = 0
		descriptionBoxYLocked = false
	}
}

func (e *Editor) functionRangeForCurrentFunction(functionName string) (LineIndex, LineIndex, bool) {
	if functionName == "" {
		return 0, 0, false
	}

	currentLine := e.LineIndex()
	foundName, functionStart := e.FunctionNameForLineIndex(currentLine)
	if foundName == "" || foundName != functionName {
		return 0, 0, false
	}

	functionEnd := max(functionStart, currentLine)

	// In brace-based languages, find a stable end line to avoid popup movement.
	if e.isBraceBasedLanguage() {
		openBraceLineIndex := LineIndex(-1)
		totalLines := LineIndex(e.Len())
		searchLimit := min(functionStart+20, totalLines)

		for i := functionStart; i < searchLimit; i++ {
			line := e.Line(i)
			trimmedLine := strings.TrimSpace(line)

			// Skip empty lines and comments while searching for the opening brace.
			if trimmedLine == "" || strings.HasPrefix(trimmedLine, "//") ||
				strings.HasPrefix(trimmedLine, "/*") || strings.HasPrefix(trimmedLine, "*") {
				continue
			}

			if strings.Contains(line, "{") {
				openBraceLineIndex = i
				break
			}

			if i > functionStart && (e.LooksLikeFunctionDef(line, e.FuncPrefix()) ||
				strings.Contains(trimmedLine, "class ") || strings.Contains(trimmedLine, "interface ") ||
				strings.Contains(trimmedLine, "enum ")) {
				break
			}
		}

		if openBraceLineIndex != LineIndex(-1) {
			if closeBraceLineIndex := e.findMatchingCloseBrace(openBraceLineIndex); closeBraceLineIndex != LineIndex(-1) {
				functionEnd = closeBraceLineIndex
			}
		}
	}

	if functionEnd < functionStart {
		functionEnd = functionStart
	}
	return functionStart, functionEnd, true
}

// DismissFunctionDescription clears and dismisses the current function description
func (e *Editor) DismissFunctionDescription() {
	dismissedFunctionDescription = e.FindCurrentFunctionName()
	if dismissedFunctionDescription == "" {
		dismissedFunctionDescription = currentDescribedFunction
	}
	functionDescriptionDismissed = dismissedFunctionDescription != ""
	functionDescriptionsDisabled = false
	clearFunctionDescriptionState()
	clearBuildErrorExplanationState()
	clearFunctionDescriptionQueue()
}

// describedFunctionScreenRange returns the visible line range for the described function
func (e *Editor) describedFunctionScreenRange(c *vt.Canvas, functionName string) (int, int, bool) {
	if c == nil || functionName == "" {
		return 0, 0, false
	}
	if !currentDescribedFunctionSpan || currentDescribedFunction != functionName {
		return 0, 0, false
	}

	offsetY := e.pos.OffsetY()
	top := int(currentDescribedFunctionFrom) - offsetY
	bottom := int(currentDescribedFunctionTo) - offsetY
	canvasHeight := int(c.Height())

	if bottom < 0 || top >= canvasHeight {
		return 0, 0, false
	}
	if top < 0 {
		top = 0
	}
	if bottom >= canvasHeight {
		bottom = canvasHeight - 1
	}

	return top, bottom, true
}

// preferredDescriptionBoxY picks a Y placement that avoids covering the described function
func (e *Editor) preferredDescriptionBoxY(c *vt.Canvas, functionName string, boxHeight, defaultY int) int {
	const minY = 2 // leave room for the function name at the top

	if c == nil || boxHeight <= 0 {
		return defaultY
	}

	canvasHeight := int(c.Height())
	maxY := canvasHeight - boxHeight
	if maxY < minY {
		return defaultY
	}
	if defaultY < minY {
		defaultY = minY
	}
	if defaultY > maxY {
		defaultY = maxY
	}

	if descriptionBoxYLocked &&
		descriptionBoxYFunction == functionName &&
		descriptionBoxYBoxHeight == boxHeight &&
		descriptionBoxYCanvasHeight == canvasHeight {
		if descriptionBoxY < minY {
			return minY
		}
		if descriptionBoxY > maxY {
			return maxY
		}
		return descriptionBoxY
	}

	boxY := defaultY
	functionTop, functionBottom, ok := e.describedFunctionScreenRange(c, functionName)
	if ok {
		spaceAbove := functionTop - minY
		spaceBelow := canvasHeight - (functionBottom + 1)
		fitsAbove := spaceAbove >= boxHeight
		fitsBelow := spaceBelow >= boxHeight

		if fitsAbove && fitsBelow {
			if spaceBelow >= spaceAbove {
				boxY = functionBottom + 1
			} else {
				boxY = functionTop - boxHeight
			}
		} else if fitsBelow {
			boxY = functionBottom + 1
		} else if fitsAbove {
			boxY = functionTop - boxHeight
		}
	}

	if boxY < minY {
		boxY = minY
	}
	if boxY > maxY {
		boxY = maxY
	}

	descriptionBoxY = boxY
	descriptionBoxYFunction = functionName
	descriptionBoxYBoxHeight = boxHeight
	descriptionBoxYCanvasHeight = canvasHeight
	descriptionBoxYLocked = true

	return boxY
}

// startQueueWorker starts the worker that processes function description requests
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
				queueMutex.Lock()
				if len(descriptionStack) == 0 {
					queueMutex.Unlock()
					break
				}
				req := descriptionStack[len(descriptionStack)-1]
				descriptionStack = descriptionStack[:len(descriptionStack)-1]
				delete(queuedHashes, req.bodyHash)
				queueMutex.Unlock()

				// Check cache in case another request filled it while queued.
				ollamaMutex.RLock()
				if cachedDescription, exists := ollamaResponseCache[req.bodyHash]; exists {
					ollamaMutex.RUnlock()
					if actualCurrentFunction == req.funcName && functionDescriptionsAllowed() {
						setCurrentDescribedFunction(req.funcName, req.start, req.end, req.hasRange)
						functionDescriptionReady = true
						functionDescriptionThinking = false
						functionDescription.Reset()
						functionDescription.WriteString(strings.TrimSpace(sanitizeOllamaText(cachedDescription)))
						req.editor.DrawFunctionDescriptionContinuous(req.canvas, false)
						req.canvas.HideCursorAndDraw()
					}
					continue
				}
				ollamaMutex.RUnlock()

				if !functionDescriptionsAllowed() {
					continue
				}

				functionDescriptionThinking = true
				processingFunction = req.funcName

				req.editor.WriteCurrentFunctionName(req.canvas)
				req.canvas.HideCursorAndDraw()

				processDescriptionRequest(req)

				functionDescriptionThinking = false
				processingFunction = ""

				req.editor.WriteCurrentFunctionName(req.canvas)
				req.canvas.HideCursorAndDraw()

				// Process one request at a time, then check for newer work.
				break
			}
		}
	}()
}

// processDescriptionRequest handles one Ollama request/response cycle
func processDescriptionRequest(req FunctionDescriptionRequest) {
	prompt := req.Prompt()

	if description, err := ollama.GetSimpleResponse(prompt); err == nil {
		description = sanitizeOllamaText(description)
		ollamaMutex.Lock()
		ollamaResponseCache[req.bodyHash] = description
		ollamaMutex.Unlock()

		if actualCurrentFunction == req.funcName && ollama.Loaded() && functionDescriptionsAllowed() {
			setCurrentDescribedFunction(req.funcName, req.start, req.end, req.hasRange)
			functionDescription.Reset()
			functionDescription.WriteString(strings.TrimSpace(description))
			functionDescriptionReady = true
			//logf("Ollama response ready for %s, drawing description box", req.funcName)
			ellipsisX := req.canvas.Width() - 1
			req.canvas.Write(ellipsisX, 0, req.editor.Foreground, req.editor.Background, " ")
			req.editor.WriteCurrentFunctionName(req.canvas)
			req.editor.DrawFunctionDescriptionContinuous(req.canvas, false)
			req.canvas.HideCursorAndDraw()
		}
	} else {
		// If this is still the current function, clear the ellipsis.
		if actualCurrentFunction == req.funcName && ollama.Loaded() && functionDescriptionsAllowed() {
			ellipsisX := req.canvas.Width() - 1
			req.canvas.Write(ellipsisX, 0, req.editor.Foreground, req.editor.Background, " ")
			req.editor.WriteCurrentFunctionName(req.canvas)
			req.canvas.HideCursorAndDraw()
		}
	}
}

// RequestFunctionDescription queues a function description request
func (e *Editor) RequestFunctionDescription(funcName, funcBody string, c *vt.Canvas) {
	if !ollama.Loaded() {
		return
	}
	if !functionDescriptionsAllowed() {
		return
	}

	if funcBody == "" {
		return
	}

	if functionDescriptionDismissed {
		if funcName == dismissedFunctionDescription {
			return
		}
		functionDescriptionDismissed = false
		dismissedFunctionDescription = ""
	}

	// Start queue worker if needed.
	startQueueWorker()

	// Hash function body for cache/queue lookup.
	bodyHash := hashFunctionBody(funcBody)
	functionStart, functionEnd, hasRange := e.functionRangeForCurrentFunction(funcName)

	ollamaMutex.RLock()
	if cachedDescription, exists := ollamaResponseCache[bodyHash]; exists {
		ollamaMutex.RUnlock()
		setCurrentDescribedFunction(funcName, functionStart, functionEnd, hasRange)
		functionDescriptionReady = true
		functionDescriptionThinking = false
		functionDescription.Reset()
		functionDescription.WriteString(strings.TrimSpace(sanitizeOllamaText(cachedDescription)))
		e.DrawFunctionDescriptionContinuous(c, false)
		c.HideCursorAndDraw()
		return
	}
	ollamaMutex.RUnlock()

	actualCurrentFunction = funcName

	if currentDescribedFunction != funcName {
		functionDescriptionReady = false
		functionDescription.Reset()
	}

	queueMutex.Lock()

	if processingFunction == funcName {
		queueMutex.Unlock()
		return
	}

	if !queuedHashes[bodyHash] {
		descriptionStack = append(descriptionStack, FunctionDescriptionRequest{
			funcName: funcName,
			funcBody: funcBody,
			bodyHash: bodyHash,
			canvas:   c,
			editor:   e,
			start:    functionStart,
			end:      functionEnd,
			hasRange: hasRange,
		})
		queuedHashes[bodyHash] = true

		select {
		case queueSignal <- struct{}{}:
		default:
		}
	} else {
		// Already queued: move request to top of LIFO stack.
		for i, req := range descriptionStack {
			if req.bodyHash == bodyHash {
				descriptionStack = append(descriptionStack[:i], descriptionStack[i+1:]...)
				descriptionStack = append(descriptionStack, FunctionDescriptionRequest{
					funcName: funcName,
					funcBody: funcBody,
					bodyHash: bodyHash,
					canvas:   c,
					editor:   e,
					start:    functionStart,
					end:      functionEnd,
					hasRange: hasRange,
				})
				break
			}
		}

		select {
		case queueSignal <- struct{}{}:
		default:
		}
	}
	queueMutex.Unlock()
}

// DrawFunctionDescriptionContinuous draws the function description panel in continuous mode
func (e *Editor) DrawFunctionDescriptionContinuous(c *vt.Canvas, repositionCursor bool) {
	if !functionDescriptionsAllowed() {
		return
	}
	if !ollama.Loaded() || currentDescribedFunction == "" || !functionDescriptionReady {
		return
	}
	if functionDescriptionDismissed && currentDescribedFunction == dismissedFunctionDescription {
		return
	}

	title := fmt.Sprintf("Function: %s", currentDescribedFunction)
	descriptionText := strings.TrimSpace(functionDescription.String())
	if len(descriptionText) == 0 {
		descriptionText = "No description available"
	}

	e.drawFunctionDescriptionPopup(c, title, descriptionText, repositionCursor)
}

// drawFunctionDescriptionPopup draws a panel with function description text
func (e *Editor) drawFunctionDescriptionPopup(c *vt.Canvas, title, descriptionText string, repositionCursorAfterDrawing bool) {
	canvasBox := NewCanvasBox(c)

	minWidth := 40
	defaultHeightPercent := 0.72 // preferred panel size for shorter descriptions
	maxHeightPercent := 0.95     // can expand if text needs more room

	// Start with default placement, then adjust vertically if needed.
	descriptionBox := NewBox()
	descriptionBox.EvenLowerRightPlacement(canvasBox, minWidth)

	// Start at line 2 to leave room for the function name.
	descriptionBox.Y = 2
	defaultHeight := int(float64(canvasBox.H) * defaultHeightPercent)
	maxHeight := max(min(int(float64(canvasBox.H)*maxHeightPercent), canvasBox.H-1), 6)
	if defaultHeight < 6 {
		defaultHeight = 6
	}
	if defaultHeight > maxHeight {
		defaultHeight = maxHeight
	}
	descriptionBox.H = defaultHeight
	e.redraw.Store(true)

	listBox := NewBox()
	listBox.FillWithMargins(descriptionBox, 2, 2)

	bt := e.NewBoxTheme()
	bt.Foreground = &e.BoxTextColor
	bt.Background = &e.BoxBackground

	neededTextLines := wrappedLineCount(descriptionText, listBox.W-5)

	// Expand if needed, without exceeding maxHeight.
	if neededTextLines > listBox.H {
		missingLines := neededTextLines - listBox.H
		availableGrowth := maxHeight - descriptionBox.H
		growBy := min(missingLines, availableGrowth)
		if growBy > 0 {
			descriptionBox.H += growBy
			listBox.H += growBy
		}
	}

	// Fit height to content when there is significant spare space.
	if neededTextLines > 0 && neededTextLines < listBox.H {
		heightDiff := listBox.H - neededTextLines
		descriptionBox.H -= (heightDiff - 1)
		listBox.H -= (heightDiff - 1)
	}

	// Visual adjustment
	descriptionBox.H--

	descriptionBox.Y = e.preferredDescriptionBoxY(c, currentDescribedFunction, descriptionBox.H, descriptionBox.Y)
	listBox.Y = descriptionBox.Y + 2

	e.DrawBox(bt, c, descriptionBox)
	e.DrawTitle(bt, c, descriptionBox, title, true)
	e.DrawText(bt, c, listBox, descriptionText, false)

	if repositionCursorAfterDrawing {
		e.EnableAndPlaceCursor(c)
	}
}
