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
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
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

// LSPLocation represents a location in a file
type LSPLocation struct {
	URI   string `json:"uri"`
	Range struct {
		Start struct {
			Line      int `json:"line"`
			Character int `json:"character"`
		} `json:"start"`
		End struct {
			Line      int `json:"line"`
			Character int `json:"character"`
		} `json:"end"`
	} `json:"range"`
}

type scoredItem struct {
	item  LSPCompletionItem
	score int
}

const (
	lspInitTimeout       = 5 * time.Second
	lspCompletionTimeout = 2 * time.Second
	lspDefinitionTimeout = 200 * time.Millisecond // Fast timeout for go-to-definition
	lspShutdownTimeout   = 2 * time.Second
)

var (
	lspClients        = make(map[mode.Mode]*LSPClient)
	lspMutex          sync.Mutex
	lastOpenedURI     string
	lastOpenedVersion int
	rustTempDirs      = make(map[mode.Mode]string) // Track temp directories for cleanup
	rustFileMapping   = make(map[string]string)    // Map original file path -> temp workspace file path
)

// LSPConfig holds configuration for a language server
type LSPConfig struct {
	Command         string
	Args            []string
	LanguageID      string
	RootMarkerFiles []string
	FileExtensions  []string
}

// Language server configurations
var lspConfigs = map[mode.Mode]LSPConfig{
	mode.Go: {
		Command:         "gopls",
		Args:            []string{},
		LanguageID:      "go",
		RootMarkerFiles: []string{"go.mod", "go.work"},
		FileExtensions:  []string{".go"},
	},
	mode.Rust: {
		Command:         "rust-analyzer",
		Args:            []string{},
		LanguageID:      "rust",
		RootMarkerFiles: []string{"Cargo.toml", "Cargo.lock", "rust-project.json"},
		FileExtensions:  []string{".rs"},
	},
	mode.C: {
		Command:         "clangd",
		Args:            []string{"--background-index"},
		LanguageID:      "c",
		RootMarkerFiles: []string{"compile_commands.json", ".clangd", "CMakeLists.txt", "Makefile"},
		FileExtensions:  []string{".c", ".h"},
	},
	mode.Cpp: {
		Command:         "clangd",
		Args:            []string{"--background-index"},
		LanguageID:      "cpp",
		RootMarkerFiles: []string{"compile_commands.json", ".clangd", "CMakeLists.txt", "Makefile"},
		FileExtensions:  []string{".cpp", ".cc", ".cxx", ".c++", ".hpp", ".hh", ".hxx", ".h++", ".h"},
	},
}

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

// writeMessage writes a JSON-RPC message with proper headers
func (lsp *LSPClient) writeMessage(message map[string]interface{}) error {
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err := lsp.stdin.Write([]byte(header)); err != nil {
		return err
	}
	if _, err := lsp.stdin.Write(body); err != nil {
		return err
	}
	return nil
}

// sendNotification sends a JSON-RPC notification (no response expected)
func (lsp *LSPClient) sendNotification(method string, params interface{}) error {
	lsp.mutex.Lock()
	defer lsp.mutex.Unlock()

	if !lsp.running {
		return errors.New("LSP client not running")
	}
	notification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	return lsp.writeMessage(notification)
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
	if err := lsp.writeMessage(request); err != nil {
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
		"rootPath":  lsp.workspaceRoot, // Some servers (like rust-analyzer) prefer this
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
	if _, err := lsp.readResponse(lspInitTimeout); err != nil {
		return err
	}
	if err := lsp.sendNotification("initialized", map[string]interface{}{}); err != nil {
		return err
	}
	lsp.initialized = true
	return nil
}

// DidOpen notifies the language server that a document was opened
func (lsp *LSPClient) DidOpen(uri, languageID, text string) error {
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        uri,
			"languageId": languageID,
			"version":    1,
			"text":       text,
		},
	}
	return lsp.sendNotification("textDocument/didOpen", params)
}

// DidChange notifies the language server that a document was changed
func (lsp *LSPClient) DidChange(uri, text string, version int) error {
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":     uri,
			"version": version,
		},
		"contentChanges": []map[string]interface{}{
			{"text": text},
		},
	}
	return lsp.sendNotification("textDocument/didChange", params)
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
	if _, err := lsp.sendRequest("textDocument/completion", params); err != nil {
		return nil, err
	}
	response, err := lsp.readResponse(lspCompletionTimeout)
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

// GetDefinition requests the definition location for a symbol at the given position
func (lsp *LSPClient) GetDefinition(uri string, line, character int, timeout time.Duration) (*LSPLocation, error) {
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
	if _, err := lsp.sendRequest("textDocument/definition", params); err != nil {
		return nil, err
	}
	response, err := lsp.readResponse(timeout)
	if err != nil {
		return nil, err
	}
	resultData, ok := response["result"]
	if !ok {
		if errorData, hasError := response["error"]; hasError {
			return nil, fmt.Errorf("LSP error: %v", errorData)
		}
		return nil, errors.New("no result in definition response")
	}
	if resultData == nil {
		return nil, errors.New("no definition found")
	}
	resultBytes, err := json.Marshal(resultData)
	if err != nil {
		return nil, err
	}

	// Try single location
	var location LSPLocation
	if err := json.Unmarshal(resultBytes, &location); err == nil && location.URI != "" {
		return &location, nil
	}

	// Try array of locations (take first one)
	var locations []LSPLocation
	if err := json.Unmarshal(resultBytes, &locations); err == nil && len(locations) > 0 {
		return &locations[0], nil
	}

	return nil, errors.New("could not parse definition response")
}

// Shutdown cleanly shuts down the LSP client
func (lsp *LSPClient) Shutdown() error {
	lsp.mutex.Lock()
	if !lsp.running {
		lsp.mutex.Unlock()
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
	lsp.writeMessage(request)
	notification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "exit",
	}
	lsp.writeMessage(notification)
	lsp.stdin.Close()
	lsp.mutex.Unlock()

	done := make(chan error, 1)
	go func() {
		done <- lsp.cmd.Wait()
	}()

	select {
	case <-done:
	case <-time.After(lspShutdownTimeout):
		lsp.cmd.Process.Kill()
	}
	return nil
}

// GetReadyLSPClient returns the LSP client if it's already running and initialized, or nil
// This is a non-blocking check - does not create or wait for LSP
func GetReadyLSPClient(m mode.Mode) *LSPClient {
	lspMutex.Lock()
	defer lspMutex.Unlock()

	if client, exists := lspClients[m]; exists && client != nil && client.running && client.initialized {
		return client
	}
	return nil
}

// GetOrCreateLSPClient returns the LSP client for the given mode, creating it if necessary
func GetOrCreateLSPClient(m mode.Mode, workspaceRoot string) (*LSPClient, error) {
	lspMutex.Lock()
	defer lspMutex.Unlock()

	if client, exists := lspClients[m]; exists && client != nil && client.running {
		return client, nil
	}
	config, ok := lspConfigs[m]
	if !ok {
		return nil, fmt.Errorf("no LSP configuration for mode %v", m)
	}
	client, err := NewLSPClient(config.Command, config.Args, workspaceRoot)
	if err != nil {
		return nil, err
	}
	if err := client.Initialize(); err != nil {
		client.Shutdown()
		return nil, err
	}
	lspClients[m] = client
	return client, nil
}

// TriggerLSPInitialization starts LSP initialization in the background if not already running
// This is called on first Tab or first Ctrl+G to warm up the LSP server
func TriggerLSPInitialization(m mode.Mode, workspaceRoot string) {
	// Quick check without locking - if already exists, don't bother
	lspMutex.Lock()
	if client, exists := lspClients[m]; exists && client != nil {
		lspMutex.Unlock()
		return
	}
	lspMutex.Unlock()

	// Start LSP in background (fire and forget)
	go func() {
		GetOrCreateLSPClient(m, workspaceRoot)
	}()
}

// findWorkspaceRoot finds the workspace root directory based on marker files
func findWorkspaceRoot(startPath string, markerFiles []string) string {
	dir := filepath.Dir(startPath)
	for {
		for _, marker := range markerFiles {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root, use starting directory
			return filepath.Dir(startPath)
		}
		dir = parent
	}
}

// ensureRustWorkspace creates a minimal Cargo.toml for standalone Rust files
// For standalone files, it creates a temporary workspace to avoid cluttering user directories
// Returns (workspaceRoot, mappedFilePath) where mappedFilePath is the file path to use for LSP
func ensureRustWorkspace(workspaceRoot, filePath string) (string, string) {
	cargoPath := filepath.Join(workspaceRoot, "Cargo.toml")

	// Check if Cargo.toml already exists (real Rust project)
	if _, err := os.Stat(cargoPath); err == nil {
		return workspaceRoot, filePath // Use existing workspace, no mapping needed
	}

	// Check if we already created a temp workspace for this file
	lspMutex.Lock()
	if mappedPath, exists := rustFileMapping[filePath]; exists {
		// Already have a temp workspace
		tempDir := rustTempDirs[mode.Rust]
		lspMutex.Unlock()
		return tempDir, mappedPath
	}
	lspMutex.Unlock()

	// For standalone files, create a temporary workspace
	// This avoids cluttering user directories with auto-generated Cargo.toml
	tempDir, err := os.MkdirTemp("", "orbiton-rust-*")
	if err != nil {
		return workspaceRoot, filePath // Fallback to original if can't create temp
	}

	// Create minimal Cargo.toml in temp directory
	cargoPath = filepath.Join(tempDir, "Cargo.toml")
	minimalCargo := `[package]
name = "standalone"
version = "0.1.0"
edition = "2021"

[dependencies]
`

	if err := os.WriteFile(cargoPath, []byte(minimalCargo), 0644); err != nil {
		os.RemoveAll(tempDir)
		return workspaceRoot, filePath // Fallback if write fails
	}

	// Create src directory in temp workspace
	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		return workspaceRoot, filePath
	}

	// Symlink the source file into temp workspace
	// If symlink fails, copy the file instead
	targetPath := filepath.Join(srcDir, filepath.Base(filePath))
	if err := os.Symlink(filePath, targetPath); err != nil {
		// Symlink failed, try copying instead
		if content, readErr := os.ReadFile(filePath); readErr == nil {
			os.WriteFile(targetPath, content, 0644)
		}
	}

	// Track temp directory for cleanup on exit
	lspMutex.Lock()
	rustTempDirs[mode.Rust] = tempDir
	// Map original file path to temp workspace file path
	rustFileMapping[filePath] = targetPath
	lspMutex.Unlock()

	return tempDir, targetPath // Use temp workspace and mapped file path
}

// gatherCodebaseStatistics scans files to find usage frequency
func gatherCodebaseStatistics(workspaceRoot string, fileExtensions []string) map[string]int {
	stats := make(map[string]int)
	filepath.WalkDir(workspaceRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || name == "target" || name == "build" || name == ".git" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		// Check if file has one of the supported extensions
		matchesExtension := false
		for _, ext := range fileExtensions {
			if strings.HasSuffix(path, ext) && !strings.HasSuffix(path, "_test"+ext) {
				matchesExtension = true
				break
			}
		}
		if !matchesExtension {
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
func sortAndFilterCompletions(items []LSPCompletionItem, context string, workspaceRoot string, fileExtensions []string) []LSPCompletionItem {
	// Trim whitespace from context
	context = strings.TrimSpace(context)
	// Gather codebase statistics
	codebaseStats := gatherCodebaseStatistics(workspaceRoot, fileExtensions)
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
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})
	result := make([]LSPCompletionItem, len(scored))
	for i, s := range scored {
		result[i] = s.item
	}
	return result
}

// GetLSPCompletions is a function to get completions for any supported language
func (e *Editor) GetLSPCompletions() ([]LSPCompletionItem, error) {
	const maxCompletions = 15

	config, ok := lspConfigs[e.mode]
	if !ok {
		return nil, fmt.Errorf("LSP not supported for mode %v", e.mode)
	}

	absPath, err := filepath.Abs(e.filename)
	if err != nil {
		return nil, err
	}

	// Find workspace root
	workspaceRoot := findWorkspaceRoot(absPath, config.RootMarkerFiles)

	// For Rust standalone files, create a temporary workspace
	// and get the mapped file path that rust-analyzer will understand
	lspFilePath := absPath
	if e.mode == mode.Rust {
		workspaceRoot, lspFilePath = ensureRustWorkspace(workspaceRoot, absPath)
	}

	client, err := GetOrCreateLSPClient(e.mode, workspaceRoot)
	if err != nil {
		return nil, err
	}

	// Get current position (LSP uses 0-indexed lines and characters)
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

	// Use the LSP file path (which might be in temp workspace for Rust)
	uri := "file://" + lspFilePath
	if lastOpenedURI != uri {
		if err := client.DidOpen(uri, config.LanguageID, buf.String()); err != nil {
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

	items = sortAndFilterCompletions(items, currentLine, workspaceRoot, config.FileExtensions)
	if len(items) > maxCompletions {
		items = items[:maxCompletions]
	}

	return items, nil
}

// ShutdownAllLSPClients shuts down all running LSP clients
func ShutdownAllLSPClients() {
	lspMutex.Lock()
	defer lspMutex.Unlock()

	for mode, client := range lspClients {
		if client != nil {
			client.Shutdown()
		}
		delete(lspClients, mode)
	}

	// Clean up temporary Rust workspaces
	for mode, tempDir := range rustTempDirs {
		if tempDir != "" {
			os.RemoveAll(tempDir)
		}
		delete(rustTempDirs, mode)
	}
}

// ShouldOfferLSPCompletion checks if LSP completion should be offered based on context
func (e *Editor) ShouldOfferLSPCompletion() bool {
	// Check if LSP is supported for this mode
	if _, ok := lspConfigs[e.mode]; !ok {
		return false
	}

	// Check if syntax highlighting is enabled
	if !e.syntaxHighlight {
		return false
	}

	// Check if cursor position is valid
	if e.pos.sx <= 0 {
		return false
	}

	// Check if we're in a completion context (after identifier or dot)
	leftRune := e.LeftRune()
	if !unicode.IsLetter(leftRune) && !unicode.IsDigit(leftRune) && leftRune != '_' && leftRune != '.' {
		return false
	}

	return true
}

// handleLSPCompletion handles LSP-based code completion in the editor for any supported language
func (e *Editor) handleLSPCompletion(c *vt.Canvas, status *StatusBar, tty *vt.TTY, undo *Undo) bool {
	if !e.ShouldOfferLSPCompletion() {
		return false
	}
	items, err := e.GetLSPCompletions()
	if err != nil {
		// Show error briefly to help user understand what went wrong
		status.SetMessageAfterRedraw("LSP: " + err.Error())
		return false
	}
	if len(items) == 0 {
		return false
	}

	choices := make([]string, 0, len(items))
	for _, item := range items {
		label := item.Label
		if item.Detail != "" && len(item.Detail) < 40 {
			if envVT100 {
				label += " * " + item.Detail
			} else {
				label += " • " + item.Detail
			}
		}
		choices = append(choices, label)
	}

	currentWord := e.CurrentWord()
	choice, _ := e.Menu(status, tty, "Completions", choices, e.Background, e.MenuTitleColor, e.MenuArrowColor, e.MenuTextColor, e.MenuHighlightColor, e.MenuSelectedColor, 0, false)
	if choice < 0 || choice >= len(items) {
		return false
	}

	undo.Snapshot(e)

	insertText := items[choice].InsertText
	if insertText == "" {
		insertText = items[choice].Label
	}
	if items[choice].TextEdit != nil && items[choice].TextEdit.NewText != "" {
		insertText = items[choice].TextEdit.NewText
	}

	if parenIndex := strings.Index(insertText, "("); parenIndex > 0 {
		insertText = insertText[:parenIndex]
	}

	var charsToDelete int
	if items[choice].TextEdit != nil {
		rangeStart := items[choice].TextEdit.Range.Start.Character
		rangeEnd := items[choice].TextEdit.Range.End.Character
		charsToDelete = rangeEnd - rangeStart
	} else if currentWord != "" {
		charsToDelete = len([]rune(currentWord))
	}

	if charsToDelete > 0 {
		for i := 0; i < charsToDelete; i++ {
			e.Prev(c)
		}
		for i := 0; i < charsToDelete; i++ {
			e.Delete(c, false)
		}
	}

	e.InsertString(c, insertText)

	const drawLines = true
	e.FullResetRedraw(c, status, drawLines, false)
	e.redraw.Store(true)
	e.redrawCursor.Store(true)

	status.SetMessage("Completed: " + insertText)
	status.ShowNoTimeout(c, e)

	c.Draw()

	return true
}
