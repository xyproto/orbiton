package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
	"github.com/xyproto/wordwrap"
)

// FunctionDescriptionRequest holds a queued function description request
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

	descriptionPopupDrawn bool // only one description popup per redraw frame

	lastKeyTime       atomic.Int64
	lastSettleKeyTime atomic.Int64
)

const keySettleDelay = 250 * time.Millisecond

// recordKeyActivity stamps the time of the most recent keystroke.
func recordKeyActivity() {
	lastKeyTime.Store(time.Now().UnixNano())
}

// keysSettled reports whether input has been idle for at least keySettleDelay.
func keysSettled() bool {
	last := lastKeyTime.Load()
	if last == 0 {
		return true
	}
	return time.Since(time.Unix(0, last)) >= keySettleDelay
}

// shouldSettleRedraw fires true once per keystroke after keySettleDelay of idle.
func shouldSettleRedraw() bool {
	last := lastKeyTime.Load()
	if last == 0 || last == lastSettleKeyTime.Load() {
		return false
	}
	if time.Since(time.Unix(0, last)) < keySettleDelay {
		return false
	}
	return lastSettleKeyTime.CompareAndSwap(lastSettleKeyTime.Load(), last)
}

// promoteAt pushes the function at line `at` onto the top of the queue, if uncached.
func (e *Editor) promoteAt(at LineIndex, c *vt.Canvas) {
	if !ollama.Loaded() || !functionDescriptionsAllowed() {
		return
	}
	funcName := e.FunctionName(e.Line(at))
	if funcName == "" {
		return
	}
	body, err := e.FunctionBlock(at)
	if err != nil || body == "" {
		return
	}
	hash := hashFunctionBody(body)
	ollamaMutex.RLock()
	_, cached := ollamaResponseCache[hash]
	ollamaMutex.RUnlock()
	if cached {
		return
	}
	start, end, hasRange := e.functionRangeForLine(funcName, at)
	req := FunctionDescriptionRequest{
		funcName: funcName, funcBody: body, bodyHash: hash,
		canvas: c, editor: e,
		start: start, end: end, hasRange: hasRange,
	}
	startQueueWorker()
	queueMutex.Lock()
	if processingFunction != funcName {
		for i := 0; i < len(descriptionStack); i++ {
			if descriptionStack[i].bodyHash == hash {
				descriptionStack = append(descriptionStack[:i], descriptionStack[i+1:]...)
				break
			}
		}
		descriptionStack = append(descriptionStack, req)
		queuedHashes[hash] = true
	}
	queueMutex.Unlock()
	select {
	case queueSignal <- struct{}{}:
	default:
	}
}

// promoteNeighborsOf queues the functions above and below the one at startLine.
func (e *Editor) promoteNeighborsOf(startLine LineIndex, c *vt.Canvas) {
	total := LineIndex(e.Len())
	for i := startLine - 1; i >= 0; i-- {
		if e.FunctionName(e.Line(i)) != "" {
			e.promoteAt(i, c)
			break
		}
	}
	for i := startLine + 1; i < total; i++ {
		if e.FunctionName(e.Line(i)) != "" {
			e.promoteAt(i, c)
			break
		}
	}
}

// promoteNeighbors queues the cursor's function and its neighbors, with the current one on top.
func (e *Editor) promoteNeighbors(funcName, funcBody string, c *vt.Canvas) {
	y := e.DataY()
	total := LineIndex(e.Len())
	_, funcStart := e.FunctionNameForLineIndex(y)
	for i := funcStart - 1; i >= 0; i-- {
		if e.FunctionName(e.Line(i)) != "" {
			e.promoteAt(i, c)
			break
		}
	}
	for i := y + 1; i < total; i++ {
		if n := e.FunctionName(e.Line(i)); n != "" && n != funcName {
			e.promoteAt(i, c)
			break
		}
	}
	e.promoteCurrent(funcName, funcBody, c)
}

// promoteCurrent pushes the cursor's function onto the top of the queue, if uncached.
func (e *Editor) promoteCurrent(funcName, funcBody string, c *vt.Canvas) {
	if funcBody == "" || !ollama.Loaded() || !functionDescriptionsAllowed() {
		return
	}
	hash := hashFunctionBody(funcBody)
	ollamaMutex.RLock()
	_, cached := ollamaResponseCache[hash]
	ollamaMutex.RUnlock()
	if cached {
		return
	}
	start, end, hasRange := e.functionRangeForCurrentFunction(funcName)
	req := FunctionDescriptionRequest{
		funcName: funcName, funcBody: funcBody, bodyHash: hash,
		canvas: c, editor: e,
		start: start, end: end, hasRange: hasRange,
	}
	startQueueWorker()
	queueMutex.Lock()
	if processingFunction != funcName {
		if n := len(descriptionStack); n > 0 && descriptionStack[n-1].bodyHash == hash {
			queueMutex.Unlock()
			return
		}
		for i := 0; i < len(descriptionStack); i++ {
			if descriptionStack[i].bodyHash == hash {
				descriptionStack = append(descriptionStack[:i], descriptionStack[i+1:]...)
				break
			}
		}
		descriptionStack = append(descriptionStack, req)
		queuedHashes[hash] = true
	}
	queueMutex.Unlock()
	select {
	case queueSignal <- struct{}{}:
	default:
	}
}

// scheduleDescriptions rebuilds the queue with the closest-to-cursor uncached function on top.
func (e *Editor) scheduleDescriptions(c *vt.Canvas) {
	if !ollama.Loaded() || !functionDescriptionsAllowed() {
		return
	}

	cursorLine := e.LineIndex()
	type fnLoc struct {
		name string
		line LineIndex
	}
	var fns []fnLoc
	total := LineIndex(e.Len())
	for i := range total {
		if n := e.FunctionName(e.Line(i)); n != "" {
			fns = append(fns, fnLoc{n, i})
		}
	}
	sort.Slice(fns, func(i, j int) bool {
		di := int(fns[i].line - cursorLine)
		if di < 0 {
			di = -di
		}
		dj := int(fns[j].line - cursorLine)
		if dj < 0 {
			dj = -dj
		}
		return di < dj
	})

	var jobs []FunctionDescriptionRequest
	seen := map[string]bool{}
	for _, f := range fns {
		body, err := e.FunctionBlock(f.line)
		if err != nil || body == "" {
			continue
		}
		hash := hashFunctionBody(body)
		if seen[hash] {
			continue
		}
		seen[hash] = true
		ollamaMutex.RLock()
		_, cached := ollamaResponseCache[hash]
		ollamaMutex.RUnlock()
		if cached {
			continue
		}
		start, end, hasRange := e.functionRangeForLine(f.name, f.line)
		jobs = append(jobs, FunctionDescriptionRequest{
			funcName: f.name, funcBody: body, bodyHash: hash,
			canvas: c, editor: e,
			start: start, end: end, hasRange: hasRange,
		})
	}

	startQueueWorker()

	queueMutex.Lock()
	descriptionStack = descriptionStack[:0]
	for k := range queuedHashes {
		delete(queuedHashes, k)
	}
	for i := len(jobs) - 1; i >= 0; i-- {
		if jobs[i].funcName == processingFunction {
			continue
		}
		descriptionStack = append(descriptionStack, jobs[i])
		queuedHashes[jobs[i].bodyHash] = true
	}
	queueMutex.Unlock()

	select {
	case queueSignal <- struct{}{}:
	default:
	}
}

// showDescriptionIfCached draws the popup for funcName when its body is cached.
func (e *Editor) showDescriptionIfCached(funcName, funcBody string, c *vt.Canvas) {
	actualCurrentFunction = funcName
	if funcBody == "" {
		return
	}
	hash := hashFunctionBody(funcBody)
	ollamaMutex.RLock()
	desc, cached := ollamaResponseCache[hash]
	ollamaMutex.RUnlock()
	if !cached {
		return
	}
	start, end, hasRange := e.functionRangeForCurrentFunction(funcName)
	setCurrentDescribedFunction(funcName, start, end, hasRange)
	functionDescription.Reset()
	functionDescription.WriteString(strings.TrimSpace(sanitizeOllamaText(desc)))
	functionDescriptionReady = true
	descriptionPopupDrawn = false
	e.DrawFunctionDescriptionContinuous(c, false)
	c.HideCursorAndDraw()
}

// functionDescriptionsAllowed checks if function descriptions can be requested and shown.
func functionDescriptionsAllowed() bool {
	return !functionDescriptionsDisabled && !hasBuildErrorExplanation()
}

// functionDescriptionWorkPending reports whether any description work is queued or running.
func functionDescriptionWorkPending() bool {
	if functionDescriptionThinking {
		return true
	}
	queueMutex.Lock()
	pending := len(descriptionStack) > 0 || processingFunction != ""
	queueMutex.Unlock()
	return pending
}

// sanitizeOllamaText replaces code fence lines with blank lines and removes control characters
func sanitizeOllamaText(text string) string {
	text = strings.ReplaceAll(text, "\t", "    ")
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			lines[i] = ""
		}
	}
	return strings.Join(lines, "\n")
}

// wrappedLineCount returns how many visual lines DrawText would use for the given text and width
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

// hashFunctionBody returns a FNV-1a hash of the function body.
// FNV-1a is a fast, non-cryptographic hash function that produces consistent
// hashes suitable for detecting function body changes. It's pure Go with no
// external dependencies, keeping the executable smaller.
func hashFunctionBody(funcBody string) string {
	const (
		fnvOffset uint64 = 14695981039346656037
		fnvPrime  uint64 = 1099511628211
	)
	hash := fnvOffset
	for _, b := range funcBody {
		hash ^= uint64(b)
		hash *= fnvPrime
	}
	return fmt.Sprintf("%016x", hash)
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

	// reset popup placement when the function or range changes
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
	return e.functionRangeFromStart(functionStart, max(functionStart, currentLine))
}

func (e *Editor) functionRangeForLine(functionName string, functionStart LineIndex) (LineIndex, LineIndex, bool) {
	if functionName == "" {
		return 0, 0, false
	}
	return e.functionRangeFromStart(functionStart, functionStart)
}

func (e *Editor) functionRangeFromStart(functionStart, fallbackEnd LineIndex) (LineIndex, LineIndex, bool) {
	functionEnd := fallbackEnd

	if e.isBraceBasedLanguage() {
		openBraceLineIndex := LineIndex(-1)
		totalLines := LineIndex(e.Len())
		searchLimit := min(functionStart+20, totalLines)

		for i := functionStart; i < searchLimit; i++ {
			line := e.Line(i)
			trimmedLine := strings.TrimSpace(line)

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
	if c == nil || functionName == "" || !currentDescribedFunctionSpan || currentDescribedFunction != functionName {
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

				// check cache in case another request filled it while queued
				ollamaMutex.RLock()
				if cachedDescription, exists := ollamaResponseCache[req.bodyHash]; exists {
					ollamaMutex.RUnlock()
					if actualCurrentFunction == req.funcName && functionDescriptionsAllowed() {
						setCurrentDescribedFunction(req.funcName, req.start, req.end, req.hasRange)
						functionDescriptionReady = true
						functionDescriptionThinking = false
						functionDescription.Reset()
						functionDescription.WriteString(strings.TrimSpace(sanitizeOllamaText(cachedDescription)))
						descriptionPopupDrawn = false
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

				// queue neighbors so prefetching radiates outward
				if req.hasRange {
					req.editor.linesMut.Lock()
					req.editor.promoteNeighborsOf(req.start, req.canvas)
					req.editor.linesMut.Unlock()
				}

				req.editor.WriteCurrentFunctionName(req.canvas)
				req.canvas.HideCursorAndDraw()
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
			descriptionPopupDrawn = false
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

// RequestFunctionDescription shows a cached description if present and (re)builds
// the priority queue so the closest-to-cursor undescribed function is next.
func (e *Editor) RequestFunctionDescription(funcName, funcBody string, c *vt.Canvas) {
	if !ollama.Loaded() || !functionDescriptionsAllowed() {
		return
	}
	if functionDescriptionDismissed {
		if funcName == dismissedFunctionDescription {
			return
		}
		functionDescriptionDismissed = false
		dismissedFunctionDescription = ""
	}
	e.showDescriptionIfCached(funcName, funcBody, c)
	if currentDescribedFunction != funcName && !functionDescriptionReady {
		functionDescription.Reset()
	}
	e.promoteNeighbors(funcName, funcBody, c)
	if keysSettled() {
		e.scheduleDescriptions(c)
	}
}

// drawFuncDescText renders text inside the function description box, with
// inline `code` segments highlighted in bold. Backticks themselves are not
// drawn. Returns the number of additional lines created by wrapping.
func (e *Editor) drawFuncDescText(bt *BoxTheme, c *vt.Canvas, r *Box, text string) int {
	text = asciiFallback(text)
	maxWidth := r.W - 5
	if maxWidth < 1 {
		return 0
	}
	var (
		x0         = uint(r.X) + 1
		lineIndex  = 0
		addedLines = 0
		fg         = *bt.Foreground
		boldFG     = fg.Combine(vt.Bold)
		bg         = *bt.Background
	)
	for line := range strings.SplitSeq(text, "\n") {
		if strings.TrimSpace(line) == "" {
			lineIndex++
			continue
		}
		wrapped, err := wordwrap.WordWrap(line, maxWidth)
		if err != nil {
			if len(line) > maxWidth {
				line = line[:maxWidth]
			}
			wrapped = []string{line}
		} else {
			addedLines += len(wrapped) - 1
		}
		// keep inCode across wrapped continuations of the same source line
		inCode := false
		for _, wl := range wrapped {
			y := uint(r.Y + lineIndex)
			x := x0
			for _, ch := range wl {
				if ch == '`' {
					inCode = !inCode
					continue
				}
				color := fg
				if inCode {
					color = boldFG
				}
				c.WriteRune(x, y, color, bg, ch)
				x++
			}
			lineIndex++
		}
	}
	return addedLines
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

	titleLabel := "Function"
	if e.mode == mode.GoAssembly || e.mode == mode.Assembly {
		titleLabel = "Instruction"
	}
	title := fmt.Sprintf("%s: %s", titleLabel, strings.ReplaceAll(currentDescribedFunction, "\t", " "))
	descriptionText := strings.TrimSpace(functionDescription.String())
	if len(descriptionText) == 0 {
		descriptionText = "No description available"
	}

	e.drawFunctionDescriptionPopup(c, title, descriptionText, repositionCursor)
}

// drawFunctionDescriptionPopup draws a description popup, allowing at most one per redraw frame
func (e *Editor) drawFunctionDescriptionPopup(c *vt.Canvas, title, descriptionText string, repositionCursorAfterDrawing bool) {
	if descriptionPopupDrawn {
		return
	}
	descriptionPopupDrawn = true
	canvasBox := NewCanvasBox(c)

	minWidth := 40
	defaultHeightPercent := 0.72
	maxHeightPercent := 0.95

	// Start with default placement, then adjust vertically if needed.
	descriptionBox := NewBox()
	descriptionBox.EvenLowerRightPlacement(canvasBox, minWidth)

	// start at line 2 to leave room for the function name
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

	// expand height first if needed, without exceeding maxHeight
	if neededTextLines > listBox.H {
		missingLines := neededTextLines - listBox.H
		availableGrowth := maxHeight - descriptionBox.H
		growBy := min(missingLines, availableGrowth)
		if growBy > 0 {
			descriptionBox.H += growBy
			listBox.H += growBy
		}
	}

	// if still overflowing, widen the box (keeping the upper-left corner)
	// before falling back to growing past maxHeight
	if neededTextLines > listBox.H {
		maxRight := canvasBox.W - 1
		for neededTextLines > listBox.H && (descriptionBox.X+descriptionBox.W) < maxRight {
			descriptionBox.W++
			listBox.W++
			neededTextLines = wrappedLineCount(descriptionText, listBox.W-5)
		}
	}

	// last resort: grow height past maxHeight, up to the bottom of the canvas
	if neededTextLines > listBox.H {
		hardMaxH := canvasBox.H - descriptionBox.Y - 1
		missingLines := neededTextLines - listBox.H
		availableGrowth := hardMaxH - descriptionBox.H
		growBy := min(missingLines, availableGrowth)
		if growBy > 0 {
			descriptionBox.H += growBy
			listBox.H += growBy
		}
	}

	// fit height to content when there is significant spare space
	if neededTextLines > 0 && neededTextLines < listBox.H {
		heightDiff := listBox.H - neededTextLines
		descriptionBox.H -= (heightDiff - 1)
		listBox.H -= (heightDiff - 1)
	}

	// visual adjustment
	descriptionBox.H--

	descriptionBox.Y = e.preferredDescriptionBoxY(c, currentDescribedFunction, descriptionBox.H, descriptionBox.Y)
	listBox.Y = descriptionBox.Y + 2

	e.DrawBox(bt, c, descriptionBox)
	// Themes that need a different title color than the box edges set
	// BoxTitleColor; otherwise fall back to the existing BoxUpperEdge.
	if e.BoxTitleColor != 0 {
		titleColor := e.BoxTitleColor
		origEdge := bt.UpperEdge
		bt.UpperEdge = &titleColor
		e.DrawTitle(bt, c, descriptionBox, title, true)
		bt.UpperEdge = origEdge
	} else {
		e.DrawTitle(bt, c, descriptionBox, title, true)
	}
	e.drawFuncDescText(bt, c, listBox, descriptionText)

	if repositionCursorAfterDrawing {
		e.EnableAndPlaceCursor(c)
	}
}
