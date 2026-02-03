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

// Helper function to check if mode needs special LSP handling (temp workspace, indexing wait, etc.)
func needsWorkspaceSetup(m mode.Mode) bool {
	return m == mode.Rust || m == mode.C || m == mode.Cpp
}

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

// lspClientKey generates a unique key for an LSP client based on mode and workspace
func lspClientKey(m mode.Mode, workspaceRoot string) string {
	return fmt.Sprintf("%d:%s", m, workspaceRoot)
}

var (
	lspClients        = make(map[string]*LSPClient) // Key is "mode:workspaceRoot"
	lspMutex          sync.Mutex
	lastOpenedURI     string
	lastOpenedVersion int
	rustTempDirs      = make(map[string]string) // Map file path -> temp directory
	rustFileMapping   = make(map[string]string) // Map original file path -> temp workspace file path
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
	mode.Python: {
		Command:         "pylsp",
		Args:            []string{},
		LanguageID:      "python",
		RootMarkerFiles: []string{"setup.py", "pyproject.toml", "requirements.txt", ".git"},
		FileExtensions:  []string{".py"},
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

// TestReady sends a simple request to check if the language server is truly ready
// For rust-analyzer and clangd, we use the workspace/symbol request as a lightweight ping
func (lsp *LSPClient) TestReady(m mode.Mode) bool {
	if !lsp.initialized {
		return false
	}

	// For rust-analyzer and clangd, use workspace/symbol as a lightweight ping
	// This doesn't require any files to be open and responds quickly once ready
	if needsWorkspaceSetup(m) {
		// Send a simple workspace/symbol query
		params := map[string]interface{}{
			"query": "",
		}

		if _, err := lsp.sendRequest("workspace/symbol", params); err != nil {
			return false
		}

		// Try to read response with a short timeout
		_, err := lsp.readResponse(500 * time.Millisecond)

		// If we got a response (even an empty one), server is ready
		if err == nil {
			return true
		}

		return false
	}

	// For other languages, assume ready after initialization
	return true
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
func (lsp *LSPClient) GetCompletions(uri string, line, character int, triggerCharacter string) ([]LSPCompletionItem, error) {
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

	// Add completion context to help LSP server understand the trigger
	if triggerCharacter != "" {
		params["context"] = map[string]interface{}{
			"triggerKind":      2, // TriggerCharacter
			"triggerCharacter": triggerCharacter,
		}
	} else {
		params["context"] = map[string]interface{}{
			"triggerKind": 1, // Invoked manually
		}
	}

	// Debug: Log the request
	if os.Getenv("ORBITON_DEBUG_LSP") != "" {
		paramsJSON, _ := json.MarshalIndent(params, "", "  ")
		fmt.Fprintf(os.Stderr, "LSP Request params:\n%s\n", string(paramsJSON))
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
func GetReadyLSPClient(m mode.Mode, workspaceRoot string) *LSPClient {
	lspMutex.Lock()
	defer lspMutex.Unlock()

	key := lspClientKey(m, workspaceRoot)
	if client, exists := lspClients[key]; exists && client != nil && client.running && client.initialized {
		return client
	}
	return nil
}

// GetOrCreateLSPClient returns the LSP client for the given mode, creating it if necessary
// If a client exists for a different workspace, it will be shut down and recreated
// The cancel channel can be used to abort the operation early
func GetOrCreateLSPClient(m mode.Mode, workspaceRoot string, cancel <-chan bool) (*LSPClient, error) {
	key := lspClientKey(m, workspaceRoot)

	lspMutex.Lock()

	// Check if we have a client for the exact workspace
	if client, exists := lspClients[key]; exists && client != nil && client.running {
		// Check if it's initialized
		if client.initialized {
			lspMutex.Unlock()
			return client, nil
		}
		// Not initialized yet, release lock and wait a bit
		lspMutex.Unlock()

		// Wait up to 3 seconds for initialization
		for i := 0; i < 30; i++ {
			select {
			case <-cancel:
				return nil, errors.New("cancelled by user")
			default:
				time.Sleep(100 * time.Millisecond)
				lspMutex.Lock()
				if client.initialized {
					lspMutex.Unlock()
					return client, nil
				}
				lspMutex.Unlock()
			}
		}
		// Timed out waiting for initialization
		return nil, errors.New("LSP client initialization timeout")
	}

	// Check if there's a client for the same mode but different workspace
	// This can happen when switching between files that need different workspaces
	for existingKey, client := range lspClients {
		// Check if this key is for the same mode (key format is "mode:workspace")
		if strings.HasPrefix(existingKey, fmt.Sprintf("%d:", m)) && existingKey != key {
			// Found a client for same mode but different workspace
			if client != nil && client.running {
				// Shut it down
				client.Shutdown()
			}
			delete(lspClients, existingKey)
		}
	}

	lspMutex.Unlock()

	// Now create a new client for the correct workspace
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

	// For rust-analyzer and clangd, wait until it's truly ready by testing it
	// Try up to 50 times with 200ms intervals (10 seconds total)
	if m == mode.Rust || m == mode.C || m == mode.Cpp {
		ready := false
		for i := 0; i < 50; i++ {
			select {
			case <-cancel:
				client.Shutdown()
				return nil, errors.New("cancelled by user")
			default:
				if client.TestReady(m) {
					ready = true
					break
				}
				time.Sleep(200 * time.Millisecond)
			}
		}

		if !ready {
			client.Shutdown()
			return nil, errors.New(config.Command + " is not responding")
		}
	}

	lspMutex.Lock()
	lspClients[key] = client
	lspMutex.Unlock()

	return client, nil
}

// TriggerLSPInitialization starts LSP initialization in the background if not already running
// This is called on first Tab or first Ctrl+G to warm up the LSP server
func TriggerLSPInitialization(m mode.Mode, workspaceRoot string) {
	// Quick check without locking - if already exists, don't bother
	lspMutex.Lock()
	key := lspClientKey(m, workspaceRoot)
	if client, exists := lspClients[key]; exists && client != nil {
		lspMutex.Unlock()
		return
	}
	lspMutex.Unlock()

	// Start LSP in background (fire and forget)
	go func() {
		neverCancel := make(chan bool)
		GetOrCreateLSPClient(m, workspaceRoot, neverCancel)
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
		// Already have a temp workspace for this file
		tempDir := rustTempDirs[filePath]
		lspMutex.Unlock()
		return tempDir, mappedPath
	}

	// Check if we already have a global temp workspace for standalone files
	// Use a special key for the shared workspace
	sharedKey := "standalone"
	if tempDir, exists := rustTempDirs[sharedKey]; exists {
		// Reuse existing temp workspace, just add this file to it
		lspMutex.Unlock()

		srcDir := filepath.Join(tempDir, "src")
		targetPath := filepath.Join(srcDir, filepath.Base(filePath))

		// Check if file already exists (might be from previous session)
		if _, err := os.Lstat(targetPath); err == nil {
			// File/symlink exists, remove it first to ensure clean state
			os.Remove(targetPath)
		}

		// Always copy the file instead of symlinking for better reliability
		// This ensures rust-analyzer can always read the file
		if content, readErr := os.ReadFile(filePath); readErr == nil {
			if writeErr := os.WriteFile(targetPath, content, 0644); writeErr != nil {
				// Failed to write, return original path as fallback
				return workspaceRoot, filePath
			}
		} else {
			// Failed to read source, return original path as fallback
			return workspaceRoot, filePath
		}

		// Verify the file was actually created
		if _, err := os.Stat(targetPath); err != nil {
			return workspaceRoot, filePath // File not created, fallback
		}

		// Update rust-project.json to include the new file
		updateRustProjectJSON(tempDir, filepath.Base(filePath))

		// Track the mapping
		lspMutex.Lock()
		rustFileMapping[filePath] = targetPath
		rustTempDirs[filePath] = tempDir
		lspMutex.Unlock()

		return tempDir, targetPath
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

	// Also create rust-project.json for better rust-analyzer support
	// This helps rust-analyzer understand standalone files without full Cargo setup
	rustProjectPath := filepath.Join(tempDir, "rust-project.json")

	// Try to find the sysroot automatically
	sysrootCmd := exec.Command("rustc", "--print", "sysroot")
	sysrootOutput, err := sysrootCmd.Output()
	sysroot := ""
	if err == nil {
		sysroot = strings.TrimSpace(string(sysrootOutput))
	}

	rustProject := map[string]interface{}{
		"crates": []map[string]interface{}{
			{
				"root_module": filepath.Join("src", filepath.Base(filePath)),
				"edition":     "2021",
				"deps":        []interface{}{},
			},
		},
	}

	// Add sysroot_src if we found it
	if sysroot != "" {
		sysrootSrc := filepath.Join(sysroot, "lib", "rustlib", "src", "rust", "library")
		if _, err := os.Stat(sysrootSrc); err == nil {
			rustProject["sysroot_src"] = sysrootSrc
		}
	}

	rustProjectJSON, _ := json.MarshalIndent(rustProject, "", "  ")
	os.WriteFile(rustProjectPath, rustProjectJSON, 0644)

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
	rustTempDirs["standalone"] = tempDir // Track the shared workspace
	rustTempDirs[filePath] = tempDir     // Track which workspace this file uses
	// Map original file path to temp workspace file path
	rustFileMapping[filePath] = targetPath
	lspMutex.Unlock()

	return tempDir, targetPath // Use temp workspace and mapped file path
}

// updateRustProjectJSON updates the rust-project.json to include a new file
func updateRustProjectJSON(tempDir, newFileName string) {
	rustProjectPath := filepath.Join(tempDir, "rust-project.json")

	// Read existing rust-project.json
	data, err := os.ReadFile(rustProjectPath)
	if err != nil {
		return // Can't update if file doesn't exist
	}

	var rustProject map[string]interface{}
	if err := json.Unmarshal(data, &rustProject); err != nil {
		return
	}

	// Get the crates array
	crates, ok := rustProject["crates"].([]interface{})
	if !ok {
		return
	}

	// Check if this file is already in the crates list
	newRootModule := filepath.Join("src", newFileName)
	for _, crateInterface := range crates {
		crate, ok := crateInterface.(map[string]interface{})
		if !ok {
			continue
		}
		if rootModule, ok := crate["root_module"].(string); ok && rootModule == newRootModule {
			return // Already exists
		}
	}

	// Add new crate entry
	newCrate := map[string]interface{}{
		"root_module": newRootModule,
		"edition":     "2021",
		"deps":        []interface{}{},
	}
	crates = append(crates, newCrate)
	rustProject["crates"] = crates

	// Write back
	updatedJSON, _ := json.MarshalIndent(rustProject, "", "  ")
	os.WriteFile(rustProjectPath, updatedJSON, 0644)
}

// ensureCWorkspace creates a temporary workspace for standalone C/C++ files
// Returns the workspace root and the file path within the workspace
func ensureCWorkspace(workspaceRoot, filePath string, languageID string) (string, string) {
	// Check if there's a compile_commands.json in the current directory or parent directories
	dir := filepath.Dir(filePath)
	for {
		compileCommandsPath := filepath.Join(dir, "compile_commands.json")
		if _, err := os.Stat(compileCommandsPath); err == nil {
			// Found compile_commands.json - use this as the workspace
			return dir, filePath
		}

		// Try parent directory
		parent := filepath.Dir(dir)
		if parent == dir || parent == "." || parent == "/" {
			// Reached root, no compile_commands.json found
			break
		}
		dir = parent
	}

	// For standalone files, create a temporary workspace with compile_commands.json
	lspMutex.Lock()
	if tempDir, exists := rustTempDirs[filePath]; exists {
		// Reuse existing temp workspace
		lspMutex.Unlock()

		targetPath := filepath.Join(tempDir, filepath.Base(filePath))

		// Update the file content
		if content, readErr := os.ReadFile(filePath); readErr == nil {
			os.WriteFile(targetPath, content, 0644)
		}

		return tempDir, targetPath
	}
	lspMutex.Unlock()

	// Create a temporary workspace
	tempDir, err := os.MkdirTemp("", "orbiton-clangd-*")
	if err != nil {
		return workspaceRoot, filePath
	}

	// Copy the source file to temp directory
	targetPath := filepath.Join(tempDir, filepath.Base(filePath))
	if content, readErr := os.ReadFile(filePath); readErr == nil {
		if writeErr := os.WriteFile(targetPath, content, 0644); writeErr != nil {
			os.RemoveAll(tempDir)
			return workspaceRoot, filePath
		}
	} else {
		os.RemoveAll(tempDir)
		return workspaceRoot, filePath
	}

	// Create compile_commands.json for clangd
	compileCommandsPath := filepath.Join(tempDir, "compile_commands.json")

	// Use clang/clang++ for clangd compatibility
	compiler := "clang"
	standard := "-std=c11"
	if languageID == "cpp" {
		compiler = "clang++"
		standard = "-std=c++17"
	}

	// Use absolute path for the file
	absTargetPath, _ := filepath.Abs(targetPath)

	compileCommands := []map[string]interface{}{
		{
			"directory": tempDir,
			"arguments": []string{compiler, standard, "-c", absTargetPath},
			"file":      absTargetPath,
		},
	}

	compileCommandsJSON, _ := json.MarshalIndent(compileCommands, "", "  ")
	if err := os.WriteFile(compileCommandsPath, compileCommandsJSON, 0644); err != nil {
		os.RemoveAll(tempDir)
		return workspaceRoot, filePath
	}

	// Track the mapping
	lspMutex.Lock()
	rustFileMapping[filePath] = targetPath
	rustTempDirs[filePath] = tempDir
	lspMutex.Unlock()

	return tempDir, targetPath
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

	// Determine if we're completing after a dot/colon (member access) or a standalone identifier
	var prefix string
	var isMemberAccess bool

	if strings.Contains(context, ".") {
		parts := strings.Split(context, ".")
		if len(parts) > 0 {
			prefix = strings.ToLower(parts[len(parts)-1])
		}
		isMemberAccess = true
	} else if strings.Contains(context, ":") && strings.Contains(context, "::") {
		// Namespace/module access like std::
		parts := strings.Split(context, "::")
		if len(parts) > 0 {
			prefix = strings.ToLower(parts[len(parts)-1])
		}
		isMemberAccess = true
	} else {
		// Completing a standalone identifier - extract the last word
		words := strings.FieldsFunc(context, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
		})
		if len(words) > 0 {
			prefix = strings.ToLower(words[len(words)-1])
		}
		isMemberAccess = false
	}

	// Common methods that should be prioritized (especially for Rust)
	commonMethods := map[string]int{
		// Very common String/Vec methods
		"len": 500, "is_empty": 450, "clone": 400, "to_string": 400,
		"as_str": 350, "push": 350, "push_str": 350, "pop": 300,
		"clear": 300, "trim": 300, "split": 300, "chars": 250,
		"bytes": 250, "lines": 250, "contains": 250, "starts_with": 250,
		"ends_with": 250, "replace": 200, "to_lowercase": 200, "to_uppercase": 200,
		"parse": 200, "into": 200, "iter": 200, "collect": 200,
		"map": 180, "filter": 180, "fold": 150, "find": 150,
		"get": 150, "insert": 150, "remove": 150, "extend": 150,
		// Common Option/Result methods
		"unwrap": 200, "expect": 180, "unwrap_or": 180, "unwrap_or_default": 150,
		"is_some": 150, "is_none": 150, "is_ok": 150, "is_err": 150,
		"ok": 150, "err": 150, "and_then": 130, "or_else": 130,
	}

	// Gather codebase statistics
	codebaseStats := gatherCodebaseStatistics(workspaceRoot, fileExtensions)
	hasDot := strings.HasSuffix(context, ".") || strings.HasSuffix(context, ":")
	var packageName string
	if hasDot {
		if parts := strings.Split(context, "."); len(parts) >= 2 {
			packageName = parts[len(parts)-2]
		} else if parts := strings.Split(context, ":"); len(parts) >= 2 {
			packageName = parts[len(parts)-2]
		}
	}

	scored := make([]scoredItem, 0, len(items))
	for _, item := range items {
		// Trim whitespace from label for comparison (clangd sometimes adds leading spaces)
		trimmedLabel := strings.TrimSpace(item.Label)

		// Filter by prefix if one exists
		if prefix != "" && !strings.HasPrefix(strings.ToLower(trimmedLabel), prefix) {
			continue
		}

		score := 0
		if hasDot && item.Label == packageName {
			continue
		}

		// Boost common methods (only for member access)
		if isMemberAccess {
			if boost, isCommon := commonMethods[strings.ToLower(item.Label)]; isCommon {
				score += boost
			}
		}

		// Use codebase statistics
		if count, found := codebaseStats[item.Label]; found && count > 0 {
			score += count * 50 // Reduced from 100 to balance with other factors
		}

		// Preselect hint from LSP
		if item.Preselect {
			score += 300
		}

		// LSP's sortText contains intelligence about relevance
		if item.SortText != "" {
			// Shorter sortText usually means higher priority
			score += (100 - len(item.SortText))
		}

		// Prioritize by item kind - different priorities for member access vs standalone
		if isMemberAccess {
			// Completing after dot - prioritize methods
			switch item.Kind {
			case 2: // Method
				score += 200
			case 5: // Field
				score += 150
			case 3: // Function
				score += 100
			case 6: // Variable
				score += 50
			case 21: // Constant
				score += 40
			case 22: // Struct
				score += 30
			case 8: // Interface/Trait
				score += 20
			case 9: // Module
				score += 10
			}
		} else {
			// Completing standalone identifier - prioritize variables and local items
			switch item.Kind {
			case 6: // Variable - highest priority for standalone completion
				score += 500
			case 5: // Field
				score += 400
			case 21: // Constant
				score += 300
			case 3: // Function
				score += 200
			case 22: // Struct
				score += 150
			case 8: // Interface/Trait
				score += 100
			case 2: // Method - lower priority for standalone
				score += 50
			case 9: // Module
				score += 30
			}
		}

		// Penalize complex signatures (more parameters = more complex)
		if item.Detail != "" {
			paramCount := strings.Count(item.Detail, ",") + 1
			if strings.Contains(item.Detail, "(") {
				// Has parameters
				if paramCount > 3 {
					score -= (paramCount - 3) * 20 // Penalize many parameters
				}
			}

			// Penalize complex generics
			genericCount := strings.Count(item.Detail, "<")
			if genericCount > 1 {
				score -= (genericCount - 1) * 30
			}

			// Penalize deprecated items heavily
			detailLower := strings.ToLower(item.Detail)
			if strings.Contains(detailLower, "deprecated") {
				score -= 500
			}

			// Boost simple return types
			if strings.Contains(item.Detail, "-> bool") ||
				strings.Contains(item.Detail, "-> usize") ||
				strings.Contains(item.Detail, "-> &str") {
				score += 30
			}
		}

		// Penalize very long names (usually less common)
		if len(item.Label) > 20 {
			score -= (len(item.Label) - 20) * 5
		}

		// Boost items that exactly match the prefix
		if prefix != "" && strings.HasPrefix(strings.ToLower(item.Label), prefix) {
			score += 2000 // High priority for prefix matches
			// Even higher boost for exact matches
			if strings.ToLower(item.Label) == prefix {
				score += 1000
			}
		}

		scored = append(scored, scoredItem{item, score})
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		// If scores are equal, sort alphabetically
		return scored[i].item.Label < scored[j].item.Label
	})

	// Deduplicate by label - keep only the first (highest scored) occurrence
	seen := make(map[string]bool)
	result := make([]LSPCompletionItem, 0, len(scored))
	for _, s := range scored {
		if !seen[s.item.Label] {
			seen[s.item.Label] = true
			result = append(result, s.item)
		}
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

	// For Rust and C/C++ standalone files, create a temporary workspace
	// and get the mapped file path that the LSP will understand
	lspFilePath := absPath
	if e.mode == mode.Rust {
		workspaceRoot, lspFilePath = ensureRustWorkspace(workspaceRoot, absPath)
	} else if e.mode == mode.C || e.mode == mode.Cpp {
		workspaceRoot, lspFilePath = ensureCWorkspace(workspaceRoot, absPath, config.LanguageID)
	}

	neverCancel := make(chan bool)
	client, err := GetOrCreateLSPClient(e.mode, workspaceRoot, neverCancel)
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

	fileContent := buf.String()

	// Use the LSP file path (which might be in temp workspace for Rust/C/C++)
	uri := "file://" + lspFilePath

	// For Rust and C/C++, also update the physical file in temp workspace
	// LSP servers might read from disk instead of relying only on DidChange
	if needsWorkspaceSetup(e.mode) && lspFilePath != absPath {
		os.WriteFile(lspFilePath, []byte(fileContent), 0644)
	}

	if lastOpenedURI != uri {
		if err := client.DidOpen(uri, config.LanguageID, fileContent); err != nil {
			return nil, err
		}
		lastOpenedURI = uri
		lastOpenedVersion = 1
	} else {
		lastOpenedVersion++
		if err := client.DidChange(uri, fileContent, lastOpenedVersion); err != nil {
			return nil, err
		}
		// Give LSP servers a moment to process the change
		// This is especially important for the first completion request
		if needsWorkspaceSetup(e.mode) {
			time.Sleep(50 * time.Millisecond)
		}
	}

	// Detect trigger character for better completion context
	// Check if the character before cursor position is a trigger character
	var triggerChar string
	if x > 0 && len(currentLine) > 0 {
		lastChar := currentLine[len(currentLine)-1:]
		if lastChar == "." || lastChar == ":" {
			triggerChar = lastChar
		}
	}

	// Debug: Log the request details
	items, err := client.GetCompletions(uri, line, x, triggerChar)
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

	for key, client := range lspClients {
		if client != nil {
			client.Shutdown()
		}
		delete(lspClients, key)
	}

	// Clean up temporary Rust workspaces
	seenDirs := make(map[string]bool)
	for key, tempDir := range rustTempDirs {
		if tempDir != "" && !seenDirs[tempDir] {
			os.RemoveAll(tempDir)
			seenDirs[tempDir] = true
		}
		delete(rustTempDirs, key)
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
// Timeline from Tab press to completion display
func (e *Editor) handleLSPCompletion(c *vt.Canvas, status *StatusBar, tty *vt.TTY, undo *Undo) bool {
	if !e.ShouldOfferLSPCompletion() {
		return false
	}

	config, ok := lspConfigs[e.mode]
	if !ok {
		return false
	}

	// STEP 1: Check if LSP executable exists
	lspCommand := config.Command
	if _, err := exec.LookPath(lspCommand); err != nil {
		// LSP executable not found, silently do nothing
		return false
	}

	// STEP 2: Check if we need to create a new LSP client
	absPath, err := filepath.Abs(e.filename)
	if err != nil {
		status.SetMessageAfterRedraw(fmt.Sprintf("%s: %v", lspCommand, err))
		return false
	}

	workspaceRoot := findWorkspaceRoot(absPath, config.RootMarkerFiles)

	// For Rust standalone files, create a temporary workspace
	lspFilePath := absPath
	if e.mode == mode.Rust {
		workspaceRoot, lspFilePath = ensureRustWorkspace(workspaceRoot, absPath)
	} else if e.mode == mode.C || e.mode == mode.Cpp {
		workspaceRoot, lspFilePath = ensureCWorkspace(workspaceRoot, absPath, config.LanguageID)
	}

	// Check if LSP client already exists
	existingClient := GetReadyLSPClient(e.mode, workspaceRoot)
	isNewClient := (existingClient == nil)

	// Only show "Starting..." message and spinner if creating new client
	var quitSpinner chan bool
	var spinnerActive bool
	var stopSpinner func()
	neverCancel := make(chan bool) // Never cancels - Esc shows message but doesn't abort LSP init

	if isNewClient {
		spinnerMsg := fmt.Sprintf("Starting %s", lspCommand)
		status.SetMessage(spinnerMsg)
		status.Show(c, e)
		const cursorAfterText = true
		quitSpinner = e.Spinner(c, tty, "", "Canceled", 750*time.Millisecond, e.MenuTextColor, cursorAfterText)
		spinnerActive = true
		stopSpinner = func() {
			if spinnerActive {
				quitSpinner <- true
				spinnerActive = false
				c.Draw() // Clear spinner
			}
		}
		defer stopSpinner() // Ensure spinner stops if we return early
	}

	// STEP 3-5: Get or create LSP client
	client, err := GetOrCreateLSPClient(e.mode, workspaceRoot, neverCancel)
	if err != nil {
		if isNewClient {
			stopSpinner()
		}
		status.SetMessageAfterRedraw(fmt.Sprintf("Could not launch %s: %v", lspCommand, err))
		return false
	}

	// Verify client is ready
	if !client.initialized {
		if isNewClient {
			stopSpinner()
		}
		status.SetMessageAfterRedraw(fmt.Sprintf("%s is not responding", lspCommand))
		return false
	}

	// Get current position
	line := int(e.DataY())
	x, err := e.DataX()
	if err != nil {
		x = 0
		if lineRunes, ok := e.lines[line]; ok {
			x = len(lineRunes)
		}
	}

	// Get current line for context
	var currentLine string
	if lineRunes, ok := e.lines[line]; ok {
		if x >= 0 && x <= len(lineRunes) {
			currentLine = string(lineRunes[:x])
		} else {
			currentLine = string(lineRunes)
		}
	}

	// Build file content
	var buf bytes.Buffer
	for i := 0; i < len(e.lines); i++ {
		if lineContent, ok := e.lines[i]; ok {
			buf.WriteString(string(lineContent))
		}
		buf.WriteRune('\n')
	}
	fileContent := buf.String()

	// For Rust and C/C++, update physical file in temp workspace
	if needsWorkspaceSetup(e.mode) && lspFilePath != absPath {
		os.WriteFile(lspFilePath, []byte(fileContent), 0644)
	}

	// Send file content to LSP
	uri := "file://" + lspFilePath
	if lastOpenedURI != uri {
		if err := client.DidOpen(uri, config.LanguageID, fileContent); err != nil {
			if isNewClient {
				stopSpinner()
			}
			status.SetMessageAfterRedraw(fmt.Sprintf("%s error: %v", lspCommand, err))
			return false
		}
		lastOpenedURI = uri
		lastOpenedVersion = 1

		// For C/C++, clangd needs time to index the standard library
		// Based on testing, indexing typically completes within 3-4 seconds
		if e.mode == mode.C || e.mode == mode.Cpp {
			time.Sleep(3 * time.Second)
		}
	} else {
		lastOpenedVersion++
		if err := client.DidChange(uri, fileContent, lastOpenedVersion); err != nil {
			if isNewClient {
				stopSpinner()
			}
			status.SetMessageAfterRedraw(fmt.Sprintf("%s error: %v", lspCommand, err))
			return false
		}
		// Give LSP servers a moment to process
		if needsWorkspaceSetup(e.mode) {
			time.Sleep(50 * time.Millisecond)
		}
	}

	// Detect trigger character
	var triggerChar string
	if x > 0 && len(currentLine) > 0 {
		lastChar := currentLine[len(currentLine)-1:]
		if lastChar == "." || lastChar == ":" {
			triggerChar = lastChar
		}
	}

	// Request completions from LSP
	// For Rust and C/C++, retry a few times if we get empty results (server might still be indexing)
	var items []LSPCompletionItem
	maxAttempts := 1
	if needsWorkspaceSetup(e.mode) {
		maxAttempts = 5 // Try up to 5 times with delays
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		items, err = client.GetCompletions(uri, line, x, triggerChar)
		if err != nil {
			if isNewClient {
				stopSpinner()
			}
			status.SetMessageAfterRedraw(fmt.Sprintf("%s error: %v", lspCommand, err))
			return false
		}

		// Filter and sort completions
		items = sortAndFilterCompletions(items, currentLine, workspaceRoot, config.FileExtensions)

		// If we got results, or this doesn't need workspace setup, we're done
		if len(items) > 0 || !needsWorkspaceSetup(e.mode) {
			break
		}

		// For Rust/C/C++, if empty results and not last attempt, wait and retry
		if attempt < maxAttempts-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	if len(items) > 10 {
		items = items[:10]
	}

	// Check if we have completions
	if len(items) == 0 {
		if isNewClient {
			stopSpinner()
		}
		status.SetMessageAfterRedraw("No completions found")
		return false
	}

	// Find the maximum label length for column alignment
	// For C/C++, use just the function name (before parenthesis)
	maxLabelLen := 0
	for _, item := range items {
		labelLen := len(item.Label)

		// For C/C++ functions, measure only the function name part
		if (e.mode == mode.C || e.mode == mode.Cpp) && strings.Contains(item.Label, "(") {
			trimmedLabel := strings.TrimSpace(item.Label)
			if parenIdx := strings.Index(trimmedLabel, "("); parenIdx > 0 {
				labelLen = len(trimmedLabel[:parenIdx])
			}
		}

		if labelLen > maxLabelLen {
			maxLabelLen = labelLen
		}
	}

	// Build menu choices with column alignment
	choices := make([]string, 0, len(items))
	for _, item := range items {
		label := item.Label

		// Special formatting for C/C++ functions
		if (e.mode == mode.C || e.mode == mode.Cpp) && strings.Contains(label, "(") {
			// Extract function name and parameters
			// Label format from clangd: " printf(const char *restrict format, ...)"
			trimmedLabel := strings.TrimSpace(label)
			if parenIdx := strings.Index(trimmedLabel, "("); parenIdx > 0 {
				funcName := trimmedLabel[:parenIdx]
				params := trimmedLabel[parenIdx:]

				// Format: "funcName    (params) -> returnType" with column alignment
				padding := maxLabelLen - len(funcName) + 2
				if padding < 2 {
					padding = 2
				}

				if item.Detail != "" && len(item.Detail) < 80 {
					label = funcName + strings.Repeat(" ", padding) + params + " -> " + item.Detail
				} else {
					label = funcName + strings.Repeat(" ", padding) + params
				}
			}
		} else if item.Detail != "" && len(item.Detail) < 80 {
			// Standard formatting for other languages
			// Pad label to align details in columns
			padding := maxLabelLen - len(item.Label) + 2 // At least 2 spaces
			label += strings.Repeat(" ", padding) + item.Detail
		}
		choices = append(choices, label)
	}

	// Stop spinner before showing menu
	if isNewClient {
		stopSpinner()
	}

	// Show menu and get user selection
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

	// Remove any existing parentheses from the insert text
	if parenIndex := strings.Index(insertText, "("); parenIndex > 0 {
		insertText = insertText[:parenIndex]
	}

	// Determine if we should add parentheses based on the Detail or Label field
	addParens := ""

	// For C/C++, check the Label for function signature
	if (e.mode == mode.C || e.mode == mode.Cpp) && strings.Contains(items[choice].Label, "(") {
		// Label format: " printf(const char *restrict format, ...)"
		trimmedLabel := strings.TrimSpace(items[choice].Label)
		if startIdx := strings.Index(trimmedLabel, "("); startIdx >= 0 {
			if endIdx := strings.LastIndex(trimmedLabel, ")"); endIdx > startIdx {
				params := strings.TrimSpace(trimmedLabel[startIdx+1 : endIdx])
				// Check if function takes parameters
				if params == "" || params == "void" {
					// No parameters - add closing paren
					addParens = "()"
				} else {
					// Has parameters - add opening paren only
					addParens = "("
				}
			}
		}
	} else if items[choice].Detail != "" {
		// For Rust and other languages, check Detail field
		detail := items[choice].Detail
		// Check if this is a function/method (has parentheses in detail)
		if strings.Contains(detail, "(") && strings.Contains(detail, ")") {
			// Extract the part between parentheses
			startIdx := strings.Index(detail, "(")
			endIdx := strings.Index(detail, ")")
			if startIdx >= 0 && endIdx > startIdx {
				params := strings.TrimSpace(detail[startIdx+1 : endIdx])
				// Check if function takes parameters
				if params == "" || params == "&self" || params == "self" || params == "&mut self" || params == "mut self" {
					// No parameters or only self - add closing paren
					addParens = "()"
				} else {
					// Has parameters - add opening paren only
					addParens = "("
				}
			}
		}
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

	e.InsertString(c, insertText+addParens)

	const drawLines = true
	e.FullResetRedraw(c, status, drawLines, false)
	e.redraw.Store(true)
	e.redrawCursor.Store(true)

	status.SetMessage("Completed: " + insertText)
	status.ShowNoTimeout(c, e)

	c.Draw()

	return true
}
