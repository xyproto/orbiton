package main

import (
	"sync"

	"github.com/xyproto/mode"
)

// highlightCacheInterval is the number of lines between cached checkpoints
const highlightCacheInterval = 1000

// highlightCache stores syntax highlighting state at regular intervals so that
// WriteLines does not need to scan from line 0 on every redraw.
// Separate maps are used for each kind of state to avoid cross-contamination
// between the independent scan loops in WriteLines.
type highlightCache struct {
	quote      map[LineIndex]QuoteState // multi-line quote/comment state
	codeBlock  map[LineIndex]bool       // for Markdown/Python: inside a code block
	backtick   map[LineIndex]bool       // for Go/Odin: inside a backtick string
	cachedMode mode.Mode                // the mode when entries were cached
	mu         sync.Mutex
}

// newHighlightCache creates an empty highlight cache
func newHighlightCache() *highlightCache {
	return &highlightCache{
		quote:     make(map[LineIndex]QuoteState),
		codeBlock: make(map[LineIndex]bool),
		backtick:  make(map[LineIndex]bool),
	}
}

// Invalidate clears all cached entries
func (hc *highlightCache) Invalidate() {
	hc.mu.Lock()
	hc.quote = make(map[LineIndex]QuoteState)
	hc.codeBlock = make(map[LineIndex]bool)
	hc.backtick = make(map[LineIndex]bool)
	hc.mu.Unlock()
}

// CheckMode invalidates the cache if the editor mode has changed
func (hc *highlightCache) CheckMode(m mode.Mode) {
	hc.mu.Lock()
	if hc.cachedMode != m {
		hc.quote = make(map[LineIndex]QuoteState)
		hc.codeBlock = make(map[LineIndex]bool)
		hc.backtick = make(map[LineIndex]bool)
		hc.cachedMode = m
	}
	hc.mu.Unlock()
}

// getNearest returns the cached value nearest to (but not exceeding) the
// target line from the given map. Returns ok=false if no suitable entry exists.
func getNearest[V any](m map[LineIndex]V, targetLine LineIndex) (value V, fromLine LineIndex, ok bool) {
	bestLine := LineIndex(-1)
	for k, v := range m {
		if k <= targetLine && k > bestLine {
			bestLine = k
			value = v
		}
	}
	if bestLine < 0 {
		var zero V
		return zero, 0, false
	}
	return value, bestLine, true
}

// GetQuote returns the nearest cached QuoteState at or before targetLine
func (hc *highlightCache) GetQuote(targetLine LineIndex) (q QuoteState, fromLine LineIndex, ok bool) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	return getNearest(hc.quote, targetLine)
}

// PutQuote stores a QuoteState checkpoint at the given line
func (hc *highlightCache) PutQuote(lineIndex LineIndex, q QuoteState) {
	hc.mu.Lock()
	hc.quote[lineIndex] = q
	hc.mu.Unlock()
}

// GetCodeBlock returns the nearest cached code-block state at or before targetLine
func (hc *highlightCache) GetCodeBlock(targetLine LineIndex) (inCodeBlock bool, fromLine LineIndex, ok bool) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	return getNearest(hc.codeBlock, targetLine)
}

// PutCodeBlock stores a code-block state checkpoint at the given line
func (hc *highlightCache) PutCodeBlock(lineIndex LineIndex, inCodeBlock bool) {
	hc.mu.Lock()
	hc.codeBlock[lineIndex] = inCodeBlock
	hc.mu.Unlock()
}

// GetBacktick returns the nearest cached backtick state at or before targetLine
func (hc *highlightCache) GetBacktick(targetLine LineIndex) (inBacktick bool, fromLine LineIndex, ok bool) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	return getNearest(hc.backtick, targetLine)
}

// PutBacktick stores a backtick state checkpoint at the given line
func (hc *highlightCache) PutBacktick(lineIndex LineIndex, inBacktick bool) {
	hc.mu.Lock()
	hc.backtick[lineIndex] = inBacktick
	hc.mu.Unlock()
}
