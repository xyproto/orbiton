package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/xyproto/mode"
)

// LSPClient manages communication with a language server
type LSPClient struct {
	stdin         io.WriteCloser
	stdout        io.ReadCloser
	stderr        io.ReadCloser
	cmd           *exec.Cmd
	workspaceRoot string
	requestID     int
	mutex         sync.Mutex
	running       bool
	initialized   bool
}

// LSPCompletionItem represents a single completion suggestion
type LSPCompletionItem struct {
	Documentation interface{} `json:"documentation"` // Can be string or object
	TextEdit      *struct {
		NewText string `json:"newText"`
		Range   struct {
			Start struct {
				Line      int `json:"line"`
				Character int `json:"character"`
			} `json:"start"`
			End struct {
				Line      int `json:"line"`
				Character int `json:"character"`
			} `json:"end"`
		} `json:"range"`
	} `json:"textEdit"`
	Label            string `json:"label"`
	Detail           string `json:"detail"`
	InsertText       string `json:"insertText"`
	FilterText       string `json:"filterText"`
	SortText         string `json:"sortText"`
	Kind             int    `json:"kind"`
	InsertTextFormat int    `json:"insertTextFormat"`
	Preselect        bool   `json:"preselect"`
}

// LSPCompletionList represents a list of completions
type LSPCompletionList struct {
	Items        []LSPCompletionItem `json:"items"`
	IsIncomplete bool                `json:"isIncomplete"`
}

type scoredItem struct {
	item  LSPCompletionItem
	score int
}

var (
	goLSPClient       *LSPClient
	lspMutex          sync.Mutex
	lastOpenedURI     string
	lastOpenedVersion int
)

// NewLSPClient creates a new LSP client for the given language server command
func NewLSPClient(serverCmd string, args []string, workspaceRoot string) (*LSPClient, error) {
	cmd := exec.Command(serverCmd, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	client := &LSPClient{
		cmd:           cmd,
		stdin:         stdin,
		stdout:        stdout,
		stderr:        stderr,
		requestID:     0,
		running:       true,
		initialized:   false,
		workspaceRoot: workspaceRoot,
	}
	// Start reading stderr in a goroutine (for debugging if needed)
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			// Silently consume stderr for now
			_ = scanner.Text()
		}
	}()
	return client, nil
}

// sendRequest sends a JSON-RPC request to the language server
func (lsp *LSPClient) sendRequest(method string, params interface{}) (int, error) {
	lsp.mutex.Lock()
	defer lsp.mutex.Unlock()

	if !lsp.running {
		return 0, errors.New("LSP client not running")
	}
	lsp.requestID++
	id := lsp.requestID
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
	body, err := json.Marshal(request)
	if err != nil {
		return 0, err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err := lsp.stdin.Write([]byte(header)); err != nil {
		return 0, err
	}
	if _, err := lsp.stdin.Write(body); err != nil {
		return 0, err
	}
	return id, nil
}

// readResponse reads a JSON-RPC response from the language server
func (lsp *LSPClient) readResponse(timeout time.Duration) (map[string]interface{}, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		done := make(chan struct{})
		var result map[string]interface{}
		var readErr error

		go func() {
			defer close(done)
			reader := bufio.NewReader(lsp.stdout)
			headers := make(map[string]string)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					readErr = err
					return
				}
				line = strings.TrimSpace(line)
				if line == "" {
					break
				}
				parts := strings.SplitN(line, ": ", 2)
				if len(parts) == 2 {
					headers[parts[0]] = parts[1]
				}
			}

			contentLength := 0
			if cl, ok := headers["Content-Length"]; ok {
				fmt.Sscanf(cl, "%d", &contentLength)
			}
			if contentLength == 0 {
				readErr = errors.New("no content length")
				return
			}
			body := make([]byte, contentLength)
			if _, err := io.ReadFull(reader, body); err != nil {
				readErr = err
				return
			}
			if err := json.Unmarshal(body, &result); err != nil {
				readErr = err
				return
			}
		}()

		select {
		case <-done:
			if readErr != nil {
				return nil, readErr
			}
			if _, hasID := result["id"]; hasID {
				return result, nil
			}
		case <-time.After(time.Until(deadline)):
			return nil, errors.New("timeout reading LSP response")
		}
	}

	return nil, errors.New("timeout reading LSP response")
}

// Initialize sends the initialize request to the language server
func (lsp *LSPClient) Initialize() error {
	params := map[string]interface{}{
		"processId": os.Getpid(),
		"rootUri":   "file://" + lsp.workspaceRoot,
		"capabilities": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"completion": map[string]interface{}{
					"completionItem": map[string]interface{}{
						"snippetSupport": false,
					},
				},
			},
		},
	}
	if _, err := lsp.sendRequest("initialize", params); err != nil {
		return err
	}
	if _, err := lsp.readResponse(5 * time.Second); err != nil {
		return err
	}
	notification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "initialized",
		"params":  map[string]interface{}{},
	}
	body, err := json.Marshal(notification)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))

	lsp.mutex.Lock()
	defer lsp.mutex.Unlock()

	if _, err := lsp.stdin.Write([]byte(header)); err != nil {
		return err
	}
	if _, err := lsp.stdin.Write(body); err != nil {
		return err
	}
	lsp.initialized = true
	return nil
}

// DidOpen notifies the language server that a document was opened
func (lsp *LSPClient) DidOpen(uri, languageID, text string) error {
	notification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri":        uri,
				"languageId": languageID,
				"version":    1,
				"text":       text,
			},
		},
	}
	body, err := json.Marshal(notification)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))

	lsp.mutex.Lock()
	defer lsp.mutex.Unlock()

	if !lsp.running {
		return errors.New("LSP client not running")
	}
	if _, err := lsp.stdin.Write([]byte(header)); err != nil {
		return err
	}
	if _, err := lsp.stdin.Write(body); err != nil {
		return err
	}
	return nil
}

// DidChange notifies the language server that a document was changed
func (lsp *LSPClient) DidChange(uri, text string, version int) error {
	notification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "textDocument/didChange",
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri":     uri,
				"version": version,
			},
			"contentChanges": []map[string]interface{}{
				{
					"text": text,
				},
			},
		},
	}
	body, err := json.Marshal(notification)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))

	lsp.mutex.Lock()
	defer lsp.mutex.Unlock()

	if !lsp.running {
		return errors.New("LSP client not running")
	}
	if _, err := lsp.stdin.Write([]byte(header)); err != nil {
		return err
	}
	if _, err := lsp.stdin.Write(body); err != nil {
		return err
	}
	return nil
}

// GetCompletions requests completions at the given position
func (lsp *LSPClient) GetCompletions(uri string, line, character int) ([]LSPCompletionItem, error) {
	if !lsp.initialized {
		return nil, errors.New("LSP client not initialized")
	}
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": uri,
		},
		"position": map[string]interface{}{
			"line":      line,
			"character": character,
		},
	}
	_, err := lsp.sendRequest("textDocument/completion", params)
	if err != nil {
		return nil, err
	}
	response, err := lsp.readResponse(2 * time.Second)
	if err != nil {
		return nil, err
	}
	resultData, ok := response["result"]
	if !ok {
		if errorData, hasError := response["error"]; hasError {
			return nil, fmt.Errorf("LSP error: %v", errorData)
		}
		return nil, errors.New("no result in completion response")
	}
	if resultData == nil {
		return []LSPCompletionItem{}, nil
	}
	resultBytes, err := json.Marshal(resultData)
	if err != nil {
		return nil, err
	}

	var completionList LSPCompletionList
	if err := json.Unmarshal(resultBytes, &completionList); err == nil {
		if len(completionList.Items) > 0 {
			return completionList.Items, nil
		}
		return []LSPCompletionItem{}, nil
	}

	var items []LSPCompletionItem
	if err := json.Unmarshal(resultBytes, &items); err == nil && len(items) > 0 {
		return items, nil
	}

	return []LSPCompletionItem{}, nil
}

// Shutdown cleanly shuts down the LSP client
func (lsp *LSPClient) Shutdown() error {
	lsp.mutex.Lock()
	defer lsp.mutex.Unlock()

	if !lsp.running {
		return nil
	}
	lsp.running = false
	lsp.requestID++
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      lsp.requestID,
		"method":  "shutdown",
		"params":  nil,
	}
	body, _ := json.Marshal(request)
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	lsp.stdin.Write([]byte(header))
	lsp.stdin.Write(body)
	notification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "exit",
	}
	body, _ = json.Marshal(notification)
	header = fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	lsp.stdin.Write([]byte(header))
	lsp.stdin.Write(body)
	lsp.stdin.Close()
	done := make(chan error, 1)

	go func() {
		done <- lsp.cmd.Wait()
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		lsp.cmd.Process.Kill()
	}
	return nil
}

// GetOrCreateGoLSPClient returns the global Go LSP client, creating it if necessary
func GetOrCreateGoLSPClient(workspaceRoot string) (*LSPClient, error) {
	lspMutex.Lock()
	defer lspMutex.Unlock()

	if goLSPClient != nil && goLSPClient.running {
		return goLSPClient, nil
	}
	client, err := NewLSPClient("gopls", []string{}, workspaceRoot)
	if err != nil {
		return nil, err
	}
	if err := client.Initialize(); err != nil {
		client.Shutdown()
		return nil, err
	}
	goLSPClient = client
	return goLSPClient, nil
}

// Common Go functions ranked by popularity, as a fallback
var commonGoFunctions = map[string]int{
	"Println": 100, "Printf": 99, "Print": 98,
	"Sprintf": 95, "Errorf": 94, "Error": 93,
	"Fprintf": 90, "Scanln": 85, "Scanf": 84,
	"Scan": 83, "New": 80, "Make": 75,
	"Append": 70, "Len": 68, "Close": 65,
	"Read": 63, "Write": 62, "String": 60,
	"Open": 58, "Create": 56, "Fatal": 55,
	"Fatalf": 54, "Log": 52, "Panic": 50,
	"Marshal": 48, "Unmarshal": 47, "Decode": 45,
	"Encode": 44, "Parse": 42, "Format": 40,
	"Atoi": 92, "Itoa": 91, "ParseInt": 88, "ParseFloat": 87,
	"FormatInt": 85, "FormatFloat": 84, "ParseBool": 82,
}

// gatherCodebaseStatistics scans *.go files to find usage frequency
func gatherCodebaseStatistics(workspaceRoot string) map[string]int {
	stats := make(map[string]int)
	filepath.Walk(workspaceRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == ".git" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		text := string(content)
		words := strings.FieldsFunc(text, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '.'
		})
		for _, word := range words {
			if strings.Contains(word, ".") {
				if parts := strings.Split(word, "."); len(parts) == 2 && len(parts[1]) > 0 {
					stats[parts[1]]++
				}
			} else if len(word) > 0 && unicode.IsUpper(rune(word[0])) {
				stats[word]++
			}
		}
		return nil
	})
	return stats
}

// sortAndFilterCompletions sorts completions by relevance
func sortAndFilterCompletions(items []LSPCompletionItem, context string, workspaceRoot string) []LSPCompletionItem {
	// Trim whitespace from context
	context = strings.TrimSpace(context)
	// Gather codebase statistics
	codebaseStats := gatherCodebaseStatistics(workspaceRoot)
	hasDot := strings.HasSuffix(context, ".")
	var packageName string
	if hasDot {
		if parts := strings.Split(context, "."); len(parts) >= 2 {
			packageName = parts[len(parts)-2]
		}
	}
	scored := make([]scoredItem, 0, len(items))
	for _, item := range items {
		score := 0
		if hasDot && item.Label == packageName {
			continue
		}
		if count, found := codebaseStats[item.Label]; found && count > 0 {
			score += count * 100
		}
		if popularity, ok := commonGoFunctions[item.Label]; ok {
			score += popularity / 2
		}
		if item.Preselect {
			score += 50
		}
		if item.SortText != "" {
			score += (50 - len(item.SortText)/2)
		}
		switch item.Kind {
		case 3:
			score += 20
		case 2:
			score += 18
		case 6:
			score += 10
		case 22:
			score += 8
		case 9:
			score += 5
		}
		scored = append(scored, scoredItem{item, score})
	}
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}
	// Extract sorted items
	result := make([]LSPCompletionItem, len(scored))
	for i, s := range scored {
		result[i] = s.item
	}
	return result
}

// GetGoCompletions is a convenience function to get completions for Go code
func (e *Editor) GetGoCompletions() ([]LSPCompletionItem, error) {
	const maxCompletions = 15
	if e.mode != mode.Go {
		return nil, errors.New("not a Go file")
	}
	absPath, err := filepath.Abs(e.filename)
	if err != nil {
		return nil, err
	}
	// Find workspace root (for go.mod)
	workspaceRoot := filepath.Dir(absPath)
	for {
		if _, err := os.Stat(filepath.Join(workspaceRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(workspaceRoot)
		if parent == workspaceRoot {
			// Reached root, use current directory
			workspaceRoot = filepath.Dir(absPath)
			break
		}
		workspaceRoot = parent
	}
	client, err := GetOrCreateGoLSPClient(workspaceRoot)
	if err != nil {
		return nil, err
	}
	// Get current position FIRST (LSP uses 0-indexed lines and characters)
	line := int(e.DataY())
	x, err := e.DataX()
	if err != nil {
		x = 0
		// If position is after data, use the line length
		if lineRunes, ok := e.lines[line]; ok {
			x = len(lineRunes)
		}
	}
	// Get the current line content for context BEFORE any LSP operations
	var currentLine string
	if lineRunes, ok := e.lines[line]; ok {
		// Use the full line up to position x for context
		if x >= 0 && x <= len(lineRunes) {
			currentLine = string(lineRunes[:x])
		} else {
			currentLine = string(lineRunes)
		}
	}
	var buf bytes.Buffer
	for i := 0; i < len(e.lines); i++ {
		if lineContent, ok := e.lines[i]; ok {
			buf.WriteString(string(lineContent))
		}
		buf.WriteRune('\n')
	}
	uri := "file://" + absPath
	if lastOpenedURI != uri {
		if err := client.DidOpen(uri, "go", buf.String()); err != nil {
			return nil, err
		}
		lastOpenedURI = uri
		lastOpenedVersion = 1
	} else {
		lastOpenedVersion++
		if err := client.DidChange(uri, buf.String(), lastOpenedVersion); err != nil {
			return nil, err
		}
	}
	items, err := client.GetCompletions(uri, line, x)
	if err != nil {
		return nil, err
	}
	items = sortAndFilterCompletions(items, currentLine, workspaceRoot)
	if len(items) > maxCompletions {
		items = items[:maxCompletions]
	}
	return items, nil
}

// ShutdownAllLSPClients shuts down all running LSP clients
func ShutdownAllLSPClients() {
	lspMutex.Lock()
	defer lspMutex.Unlock()

	if goLSPClient != nil {
		goLSPClient.Shutdown()
		goLSPClient = nil
	}
}
