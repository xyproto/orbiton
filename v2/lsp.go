package main

import (
	"bufio"
	"context"
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

	"github.com/xyproto/files"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

// needsWorkspaceSetup checks if the mode needs special LSP handling
func needsWorkspaceSetup(m mode.Mode) bool {
	return m == mode.Rust || m == mode.C || m == mode.Cpp || m == mode.Gleam
}

// LSPClient manages communication with a language server
type LSPClient struct {
	stdin          io.WriteCloser
	stdout         io.ReadCloser
	stderr         io.ReadCloser
	cmd            *exec.Cmd
	reader         *bufio.Reader // persistent reader for stdout
	workspaceRoot  string
	openedURI      string
	linkedProjects []any // inline rust-project.json objects for standalone Rust files
	openedVersion  int
	requestID      int
	mutex          sync.Mutex
	readerMu       sync.Mutex // protects reader access
	running        bool
	initialized    bool
}

// LSPCompletionItem represents a single completion suggestion
type LSPCompletionItem struct {
	Documentation any `json:"documentation"` // Can be string or object
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

// lineUpToX returns the content of the given line up to position x
func (e *Editor) lineUpToX(line, x int) string {
	if lineRunes, ok := e.lines[line]; ok {
		if x >= 0 && x <= len(lineRunes) {
			return string(lineRunes[:x])
		}
		return string(lineRunes)
	}
	return ""
}

const (
	lspInitTimeout           = 30 * time.Second
	lspCompletionTimeout     = 10 * time.Second
	lspCompletionWaitTimeout = 3 * time.Second
	lspDefinitionTimeout     = 200 * time.Millisecond
	lspShutdownTimeout       = 2 * time.Second
)

// LSP CompletionItemKind constants (from the LSP specification)
const (
	lspKindText          = 1
	lspKindMethod        = 2
	lspKindFunction      = 3
	lspKindConstructor   = 4
	lspKindField         = 5
	lspKindVariable      = 6
	lspKindClass         = 7
	lspKindInterface     = 8
	lspKindModule        = 9
	lspKindProperty      = 10
	lspKindUnit          = 11
	lspKindValue         = 12
	lspKindEnum          = 13
	lspKindKeyword       = 14
	lspKindSnippet       = 15
	lspKindColor         = 16
	lspKindFile          = 17
	lspKindReference     = 18
	lspKindFolder        = 19
	lspKindEnumMember    = 20
	lspKindConstant      = 21
	lspKindStruct        = 22
	lspKindEvent         = 23
	lspKindOperator      = 24
	lspKindTypeParameter = 25
)

// lspClientKey generates a unique key for an LSP client based on mode and workspace
func lspClientKey(m mode.Mode, workspaceRoot string) string {
	return fmt.Sprintf("%d:%s", m, workspaceRoot)
}

var (
	lspClients     = make(map[string]*LSPClient) // key is "mode:workspaceRoot"
	lspMutex       sync.Mutex
	lspTempDirs    = make(map[string]string) // file path -> temp directory
	lspFileMapping = make(map[string]string) // original file path -> temp workspace file path
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
		Command:         "pyright-langserver",
		Args:            []string{"--stdio"},
		LanguageID:      "python",
		RootMarkerFiles: []string{"setup.py", "pyproject.toml", "requirements.txt", ".git"},
		FileExtensions:  []string{".py"},
	},
	mode.Zig: {
		Command:         "zls",
		Args:            []string{"--disable-lsp-logs"},
		LanguageID:      "zig",
		RootMarkerFiles: []string{"build.zig", "build.zig.zon", "zls.json", ".git"},
		FileExtensions:  []string{".zig"},
	},
	mode.Gleam: {
		Command:         "gleam",
		Args:            []string{"lsp"},
		LanguageID:      "gleam",
		RootMarkerFiles: []string{"gleam.toml"},
		FileExtensions:  []string{".gleam"},
	},
	mode.Haskell: {
		Command:         "haskell-language-server-wrapper",
		Args:            []string{"--lsp"},
		LanguageID:      "haskell",
		RootMarkerFiles: []string{"hie.yaml", "*.cabal", "cabal.project", "stack.yaml", ".git"},
		FileExtensions:  []string{".hs"},
	},
	mode.Lua: {
		Command:         "lua-language-server",
		Args:            []string{},
		LanguageID:      "lua",
		RootMarkerFiles: []string{".luarc.json", ".luarc.jsonc", ".luacheckrc", ".git"},
		FileExtensions:  []string{".lua"},
	},
	mode.Ruby: {
		Command:         "ruby-lsp",
		Args:            []string{},
		LanguageID:      "ruby",
		RootMarkerFiles: []string{"Gemfile", ".ruby-version", ".git"},
		FileExtensions:  []string{".rb"},
	},
	mode.Shell: {
		Command:         "bash-language-server",
		Args:            []string{"start"},
		LanguageID:      "shellscript",
		RootMarkerFiles: []string{".git"},
		FileExtensions:  []string{".sh", ".bash"},
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
	// drain stderr in the background
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
func (lsp *LSPClient) writeMessage(message map[string]any) error {
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
func (lsp *LSPClient) sendNotification(method string, params any) error {
	lsp.mutex.Lock()
	defer lsp.mutex.Unlock()

	if !lsp.running {
		return errors.New("LSP client not running")
	}
	notification := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	return lsp.writeMessage(notification)
}

// sendRequest sends a JSON-RPC request to the language server
func (lsp *LSPClient) sendRequest(method string, params any) (int, error) {
	lsp.mutex.Lock()
	defer lsp.mutex.Unlock()

	if !lsp.running {
		return 0, errors.New("LSP client not running")
	}
	lsp.requestID++
	id := lsp.requestID
	request := map[string]any{
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

// readResponse reads the JSON-RPC response matching expectedID,
// skipping notifications and acknowledging server-to-client requests
func (lsp *LSPClient) readResponse(expectedID int, timeout time.Duration) (map[string]any, error) {
	deadline := time.Now().Add(timeout)

	lsp.readerMu.Lock()
	defer lsp.readerMu.Unlock()

	if lsp.reader == nil {
		lsp.reader = bufio.NewReader(lsp.stdout)
	}

	for time.Now().Before(deadline) {
		done := make(chan struct{})
		var result map[string]any
		var readErr error

		go func() {
			defer close(done)
			headers := make(map[string]string)
			for {
				line, err := lsp.reader.ReadString('\n')
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
			if _, err := io.ReadFull(lsp.reader, body); err != nil {
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
				lsp.mutex.Lock()
				lsp.running = false
				lsp.openedURI = ""
				lsp.mutex.Unlock()
				return nil, readErr
			}
			// server-to-client request — acknowledge it
			if _, hasMethod := result["method"]; hasMethod {
				if reqID, hasID := result["id"]; hasID {
					lsp.mutex.Lock()
					lsp.writeMessage(map[string]any{
						"jsonrpc": "2.0",
						"id":      reqID,
						"result":  nil,
					})
					lsp.mutex.Unlock()
				}
				continue
			}
			if idVal, hasID := result["id"]; hasID {
				if id, ok := idVal.(float64); ok && int(id) == expectedID {
					return result, nil
				}
				continue // stale response
			}
			// no id or method, skip
		case <-time.After(time.Until(deadline)):
			return nil, errors.New("timeout reading LSP response")
		}
	}

	return nil, errors.New("timeout reading LSP response")
}

// Initialize sends the initialize request to the language server
func (lsp *LSPClient) Initialize() error {
	params := map[string]any{
		"processId": os.Getpid(),
		"rootUri":   "file://" + lsp.workspaceRoot,
		"rootPath":  lsp.workspaceRoot,
		"capabilities": map[string]any{
			"textDocument": map[string]any{
				"completion": map[string]any{
					"completionItem": map[string]any{
						"snippetSupport": false,
					},
				},
			},
		},
	}
	if len(lsp.linkedProjects) > 0 {
		params["initializationOptions"] = map[string]any{
			"linkedProjects": lsp.linkedProjects,
		}
	}
	if id, err := lsp.sendRequest("initialize", params); err != nil {
		return err
	} else if _, err := lsp.readResponse(id, lspInitTimeout); err != nil {
		return err
	}
	if err := lsp.sendNotification("initialized", map[string]any{}); err != nil {
		return err
	}
	lsp.initialized = true
	return nil
}

// TestReady checks if the language server is ready to serve requests
func (lsp *LSPClient) TestReady(m mode.Mode) bool {
	if !lsp.initialized {
		return false
	}

	// use workspace/symbol as a lightweight ping
	if needsWorkspaceSetup(m) {
		params := map[string]any{
			"query": "",
		}

		if id, err := lsp.sendRequest("workspace/symbol", params); err != nil {
			return false
		} else if _, err = lsp.readResponse(id, 500*time.Millisecond); err == nil {
			return true
		}
		return false
	}

	// For other languages, assume ready after initialization
	return true
}

// DidOpen notifies the language server that a document was opened
func (lsp *LSPClient) DidOpen(uri, languageID, text string) error {
	params := map[string]any{
		"textDocument": map[string]any{
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
	params := map[string]any{
		"textDocument": map[string]any{
			"uri":     uri,
			"version": version,
		},
		"contentChanges": []map[string]any{
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
	params := map[string]any{
		"textDocument": map[string]any{
			"uri": uri,
		},
		"position": map[string]any{
			"line":      line,
			"character": character,
		},
	}

	// add completion context
	if triggerCharacter != "" {
		params["context"] = map[string]any{
			"triggerKind":      2, // TriggerCharacter
			"triggerCharacter": triggerCharacter,
		}
	} else {
		params["context"] = map[string]any{
			"triggerKind": 1, // Invoked manually
		}
	}

	id, err := lsp.sendRequest("textDocument/completion", params)
	if err != nil {
		return nil, err
	}
	response, err := lsp.readResponse(id, lspCompletionTimeout)
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
	params := map[string]any{
		"textDocument": map[string]any{
			"uri": uri,
		},
		"position": map[string]any{
			"line":      line,
			"character": character,
		},
	}
	id, err := lsp.sendRequest("textDocument/definition", params)
	if err != nil {
		return nil, err
	}
	response, err := lsp.readResponse(id, timeout)
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
	request := map[string]any{
		"jsonrpc": "2.0",
		"id":      lsp.requestID,
		"method":  "shutdown",
		"params":  nil,
	}
	lsp.writeMessage(request)
	notification := map[string]any{
		"jsonrpc": "2.0",
		"method":  "exit",
	}
	lsp.writeMessage(notification)
	lsp.stdin.Close()
	lsp.stdout.Close()
	lsp.stderr.Close()
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

// isAlive checks whether the LSP process is still running by probing stdin
func (lsp *LSPClient) isAlive() bool {
	if lsp.cmd == nil || lsp.cmd.Process == nil {
		return false
	}
	// a harmless notification; fails if the process has died
	err := lsp.sendNotification("$/cancelRequest", map[string]any{"id": -1})
	return err == nil
}

// GetReadyLSPClient returns the LSP client if already running and initialized, or nil
func GetReadyLSPClient(m mode.Mode, workspaceRoot string) *LSPClient {
	lspMutex.Lock()
	defer lspMutex.Unlock()

	key := lspClientKey(m, workspaceRoot)
	if client, exists := lspClients[key]; exists && client != nil && client.running && client.initialized {
		if !client.isAlive() {
			client.running = false
			delete(lspClients, key)
			return nil
		}
		return client
	}
	return nil
}

// GetOrCreateLSPClient returns the LSP client for the given mode, creating it if needed
func GetOrCreateLSPClient(m mode.Mode, workspaceRoot string, ctx context.Context, linkedProjects ...any) (*LSPClient, error) {
	key := lspClientKey(m, workspaceRoot)

	lspMutex.Lock()

	if client, exists := lspClients[key]; exists && client != nil && client.running {
		if !client.isAlive() {
			client.running = false
			delete(lspClients, key)
		} else if client.initialized {
			lspMutex.Unlock()
			return client, nil
		} else {
			lspMutex.Unlock()
			for range 30 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
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
			return nil, errors.New("LSP client initialization timeout")
		}
	}

	// shut down any client for the same mode but different workspace
	for existingKey, client := range lspClients {
		if strings.HasPrefix(existingKey, fmt.Sprintf("%d:", m)) && existingKey != key {
			if client != nil && client.running {
				client.Shutdown()
			}
			delete(lspClients, existingKey)
		}
	}

	lspMutex.Unlock()

	config, ok := lspConfigs[m]
	if !ok {
		return nil, fmt.Errorf("no LSP configuration for mode %v", m)
	}

	// For Python, fall back to pylsp if pyright-langserver is not installed
	command := config.Command
	args := config.Args
	if m == mode.Python && files.WhichCached(config.Command) == "" {
		if files.WhichCached("pylsp") != "" {
			command = "pylsp"
			args = []string{}
		}
	}

	client, err := NewLSPClient(command, args, workspaceRoot)
	if err != nil {
		return nil, err
	}
	if len(linkedProjects) > 0 {
		client.linkedProjects = linkedProjects
	}
	if err := client.Initialize(); err != nil {
		client.Shutdown()
		return nil, err
	}

	// wait until the server is truly ready (up to 10 seconds)
	if m == mode.Rust || m == mode.C || m == mode.Cpp {
		ready := false
		for range 50 {
			select {
			case <-ctx.Done():
				client.Shutdown()
				return nil, ctx.Err()
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
func TriggerLSPInitialization(m mode.Mode, workspaceRoot string, linkedProjects ...any) {
	// quick check — if already exists, return
	lspMutex.Lock()
	key := lspClientKey(m, workspaceRoot)
	if client, exists := lspClients[key]; exists && client != nil {
		lspMutex.Unlock()
		return
	}
	lspMutex.Unlock()

	// start LSP in the background
	go func() {
		GetOrCreateLSPClient(m, workspaceRoot, context.Background(), linkedProjects...)
	}()
}

// findWorkspaceRoot finds the workspace root directory based on marker files
func findWorkspaceRoot(startPath string, markerFiles []string) string {
	dir := filepath.Dir(startPath)
	for {
		for _, marker := range markerFiles {
			if strings.ContainsAny(marker, "*?[") {
				if matches, _ := filepath.Glob(filepath.Join(dir, marker)); len(matches) > 0 {
					return dir
				}
			} else if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// reached filesystem root
			return filepath.Dir(startPath)
		}
		dir = parent
	}
}

// hasCargoToml checks if a Cargo.toml exists in the given directory
func hasCargoToml(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "Cargo.toml"))
	return err == nil
}

// rustProjectForFile builds an inline rust-project.json object for a standalone Rust file
func rustProjectForFile(filename string) map[string]any {
	project := map[string]any{
		"crates": []map[string]any{{
			"root_module": filename,
			"edition":     "2021",
			"deps":        []any{},
		}},
	}
	if out, err := exec.Command("rustc", "--print", "sysroot").Output(); err == nil {
		sysrootSrc := filepath.Join(strings.TrimSpace(string(out)), "lib", "rustlib", "src", "rust", "library")
		if _, err := os.Stat(sysrootSrc); err == nil {
			project["sysroot_src"] = sysrootSrc
		}
	}
	return project
}

// ensureCWorkspace creates a temporary workspace for standalone C/C++ files
func ensureCWorkspace(workspaceRoot, filePath string, languageID string) (string, string) {
	// look for an existing compile_commands.json
	dir := filepath.Dir(filePath)
	for {
		compileCommandsPath := filepath.Join(dir, "compile_commands.json")
		if _, err := os.Stat(compileCommandsPath); err == nil {
			// Found compile_commands.json - use this as the workspace
			return dir, filePath
		}

		parent := filepath.Dir(dir)
		if parent == dir || parent == "." || parent == "/" {
			break
		}
		dir = parent
	}

	// create a temporary workspace with compile_commands.json
	lspMutex.Lock()
	if tempDir, exists := lspTempDirs[filePath]; exists {
		lspMutex.Unlock()

		targetPath := filepath.Join(tempDir, filepath.Base(filePath))
		if content, readErr := os.ReadFile(filePath); readErr == nil {
			os.WriteFile(targetPath, content, 0644)
		}

		return tempDir, targetPath
	}
	lspMutex.Unlock()

	tempDir, err := os.MkdirTemp("", "orbiton-clangd-*")
	if err != nil {
		return workspaceRoot, filePath
	}

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

	compileCommandsPath := filepath.Join(tempDir, "compile_commands.json")
	compiler := "clang"
	standard := "-std=c11"
	if languageID == "cpp" {
		compiler = "clang++"
		standard = "-std=c++17"
	}

	absTargetPath, _ := filepath.Abs(targetPath)

	compileCommands := []map[string]any{
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

	// track the mapping
	lspMutex.Lock()
	lspFileMapping[filePath] = targetPath
	lspTempDirs[filePath] = tempDir
	lspMutex.Unlock()

	return tempDir, targetPath
}

// hasGleamToml checks if a gleam.toml exists in the given directory
func hasGleamToml(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "gleam.toml"))
	return err == nil
}

// ensureGleamWorkspace creates a temporary Gleam project for standalone files
// and returns the workspace root and the LSP file path within it
func ensureGleamWorkspace(absPath string) (string, string) {
	gleamTmpDir := filepath.Join(userCacheDir, "o", "gleam")
	os.MkdirAll(filepath.Join(gleamTmpDir, "src"), 0o755)
	gleamToml := "name = \"main\"\nversion = \"0.1.0\"\n\n[dependencies]\ngleam_stdlib = \">= 0.44.0 and < 2.0.0\"\n"
	os.WriteFile(filepath.Join(gleamTmpDir, "gleam.toml"), []byte(gleamToml), 0o644)
	targetPath := filepath.Join(gleamTmpDir, "src", "main.gleam")
	if data, err := os.ReadFile(absPath); err == nil {
		os.WriteFile(targetPath, data, 0o644)
	}
	return gleamTmpDir, targetPath
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
func sortAndFilterCompletions(items []LSPCompletionItem, context string, workspaceRoot string, fileExtensions []string, m mode.Mode) []LSPCompletionItem {
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
			// also match after a dot in the label (e.g. "io.println" matches prefix "pri")
			if dotIdx := strings.LastIndex(trimmedLabel, "."); dotIdx >= 0 {
				if !strings.HasPrefix(strings.ToLower(trimmedLabel[dotIdx+1:]), prefix) {
					continue
				}
			} else {
				continue
			}
		}

		score := 0
		if hasDot && item.Label == packageName {
			continue
		}

		// Boost items that match the prefix more closely
		// e.g., if prefix is "get", "getStdOut" should rank higher than "GenericReader"
		if prefix != "" {
			labelLower := strings.ToLower(trimmedLabel)
			// Give a boost proportional to how much of the prefix matches
			// This helps "getStdOut" rank higher than "GenericReader" when prefix is "ge" or "get"
			matchLen := 0
			for i := 0; i < len(prefix) && i < len(labelLower); i++ {
				if prefix[i] == labelLower[i] {
					matchLen++
				} else {
					break
				}
			}
			score += matchLen * 50 // Boost per matching character
		}

		// Boost common methods (only for member access)
		if isMemberAccess {
			labelLower := strings.ToLower(item.Label)
			if boost, isCommon := commonMethods[labelLower]; isCommon {
				score += boost
			}
			// Also boost items that START with common method prefixes
			for method := range commonMethods {
				if strings.HasPrefix(labelLower, method) && len(labelLower) > len(method) {
					// Partial boost for prefix match (e.g., "getStdOut" matches "get")
					score += commonMethods[method] / 2
					break
				}
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
			case lspKindMethod:
				score += 200
			case lspKindField:
				score += 150
			case lspKindFunction:
				score += 100
			case lspKindVariable:
				score += 50
			case lspKindConstant:
				score += 40
			case lspKindStruct:
				score += 30
			case lspKindInterface:
				score += 20
			case lspKindModule:
				score += 10
			}
		} else {
			// Completing standalone identifier - prioritize variables and local items
			switch item.Kind {
			case lspKindVariable:
				score += 500
			case lspKindField:
				score += 400
			case lspKindConstant:
				score += 300
			case lspKindFunction:
				score += 200
			case lspKindStruct:
				score += 150
			case lspKindInterface:
				score += 100
			case lspKindMethod:
				score += 50
			case lspKindModule:
				score += 30
			}
		}

		// For shell scripts, boost builtins/keywords over external commands
		if m == mode.Shell && item.Kind == lspKindKeyword {
			score += 600
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

// GetLSPCompletions gets LSP completions for the current cursor position
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

	workspaceRoot := findWorkspaceRoot(absPath, config.RootMarkerFiles)
	lspFilePath := absPath
	var linkedProjects []any
	if e.mode == mode.Rust && !hasCargoToml(workspaceRoot) {
		linkedProjects = []any{rustProjectForFile(filepath.Base(absPath))}
		workspaceRoot = filepath.Dir(absPath)
	} else if e.mode == mode.C || e.mode == mode.Cpp {
		workspaceRoot, lspFilePath = ensureCWorkspace(workspaceRoot, absPath, config.LanguageID)
	} else if e.mode == mode.Gleam && !hasGleamToml(workspaceRoot) {
		workspaceRoot, lspFilePath = ensureGleamWorkspace(absPath)
	}

	client, err := GetOrCreateLSPClient(e.mode, workspaceRoot, context.Background(), linkedProjects...)
	if err != nil {
		return nil, err
	}

	line := int(e.DataY()) // LSP uses 0-indexed lines and characters
	x, err := e.DataX()
	if err != nil {
		x = 0
		// If position is after data, use the line length
		if lineRunes, ok := e.lines[line]; ok {
			x = len(lineRunes)
		}
	}

	currentLine := e.lineUpToX(line, x)
	fileContent := e.String()

	uri := "file://" + lspFilePath
	if needsWorkspaceSetup(e.mode) && lspFilePath != absPath {
		os.WriteFile(lspFilePath, []byte(fileContent), 0644)
	}

	if client.openedURI != uri {
		if err := client.DidOpen(uri, config.LanguageID, fileContent); err != nil {
			client.openedURI = ""
			return nil, err
		}
		client.openedURI = uri
		client.openedVersion = 1
	} else {
		client.openedVersion++
		if err := client.DidChange(uri, fileContent, client.openedVersion); err != nil {
			client.openedURI = ""
			return nil, err
		}
		if needsWorkspaceSetup(e.mode) {
			time.Sleep(50 * time.Millisecond)
		}
	}

	// detect trigger character (only if cursor is immediately after "." or "::")
	var triggerChar string
	if x > 0 && len(currentLine) > 0 {
		lastChar := string(currentLine[len(currentLine)-1])
		if lastChar == "." {
			triggerChar = "."
		} else if len(currentLine) >= 2 && currentLine[len(currentLine)-2:] == "::" {
			triggerChar = ":"
		}
	}

	items, err := client.GetCompletions(uri, line, x, triggerChar)
	if err != nil {
		return nil, err
	}

	items = sortAndFilterCompletions(items, currentLine, workspaceRoot, config.FileExtensions, e.mode)
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

	// clean up temporary workspaces
	seenDirs := make(map[string]bool)
	for key, tempDir := range lspTempDirs {
		if tempDir != "" && !seenDirs[tempDir] {
			os.RemoveAll(tempDir)
			seenDirs[tempDir] = true
		}
		delete(lspTempDirs, key)
	}
}

// ShouldOfferLSPCompletion checks if LSP completion should be offered based on context
func (e *Editor) ShouldOfferLSPCompletion() bool {
	if _, ok := lspConfigs[e.mode]; !ok {
		return false
	}
	if !e.syntaxHighlight {
		return false
	}
	if e.pos.sx <= 0 {
		return false
	}
	leftRune := e.LeftRune()
	if !unicode.IsLetter(leftRune) && !unicode.IsDigit(leftRune) && leftRune != '_' && leftRune != '.' && leftRune != ':' {
		return false
	}

	return true
}

// handleLSPCompletion handles LSP-based code completion
func (e *Editor) handleLSPCompletion(c *vt.Canvas, status *StatusBar, tty *vt.TTY, undo *Undo) bool {
	if !e.ShouldOfferLSPCompletion() {
		return false
	}

	// Dismiss any active function description popup to avoid stale overlapping boxes
	e.DismissFunctionDescription()

	config, ok := lspConfigs[e.mode]
	if !ok {
		return false
	}

	// For Python, fall back to pylsp if pyright-langserver is not installed
	lspCommand := config.Command
	if e.mode == mode.Python && files.WhichCached(config.Command) == "" {
		if files.WhichCached("pylsp") != "" {
			lspCommand = "pylsp"
		}
	}

	if _, err := exec.LookPath(lspCommand); err != nil {
		status.SetMessageAfterRedraw(lspCommand + " is missing")
		return false
	}

	absPath, err := filepath.Abs(e.filename)
	if err != nil {
		status.SetMessageAfterRedraw(fmt.Sprintf("%s: %v", lspCommand, err))
		return false
	}

	workspaceRoot := findWorkspaceRoot(absPath, config.RootMarkerFiles)

	lspFilePath := absPath
	var linkedProjects []any
	if e.mode == mode.Rust && !hasCargoToml(workspaceRoot) {
		linkedProjects = []any{rustProjectForFile(filepath.Base(absPath))}
		workspaceRoot = filepath.Dir(absPath)
	} else if e.mode == mode.C || e.mode == mode.Cpp {
		workspaceRoot, lspFilePath = ensureCWorkspace(workspaceRoot, absPath, config.LanguageID)
	} else if e.mode == mode.Gleam && !hasGleamToml(workspaceRoot) {
		workspaceRoot, lspFilePath = ensureGleamWorkspace(absPath)
	}

	// check if the LSP client already exists
	existingClient := GetReadyLSPClient(e.mode, workspaceRoot)
	isNewClient := (existingClient == nil)

	var quitSpinner chan bool
	var spinnerActive bool

	startSpinner := func(msg string) {
		if spinnerActive {
			return
		}
		if msg != "" {
			status.SetMessage(msg)
			status.Show(c, e)
		}
		const cursorAfterText = true
		quitSpinner = e.Spinner(c, tty, "", "Canceled", 750*time.Millisecond, e.MenuTextColor, cursorAfterText)
		spinnerActive = true
	}

	stopSpinner := func() {
		if spinnerActive {
			quitSpinner <- true
			spinnerActive = false
			c.Draw()
		}
	}
	defer stopSpinner()

	if isNewClient {
		startSpinner(fmt.Sprintf("Starting %s", lspCommand))
	}

	client, err := GetOrCreateLSPClient(e.mode, workspaceRoot, context.Background(), linkedProjects...)
	if err != nil {
		stopSpinner()
		status.SetMessageAfterRedraw(fmt.Sprintf("Could not launch %s: %v", lspCommand, err))
		return false
	}

	// verify client is ready
	if !client.initialized {
		stopSpinner()
		status.SetMessageAfterRedraw(fmt.Sprintf("%s is not responding", lspCommand))
		return false
	}

	line := int(e.DataY())
	x, err := e.DataX()
	if err != nil {
		x = 0
		if lineRunes, ok := e.lines[line]; ok {
			x = len(lineRunes)
		}
	}

	currentLine := e.lineUpToX(line, x)
	fileContent := e.String()

	// check if completing right after a trigger character
	var needsPlaceholder bool
	if x > 0 && x <= len(currentLine) {
		lastChar := currentLine[len(currentLine)-1]
		if lastChar == '.' || (len(currentLine) >= 2 && lastChar == ':') {
			needsPlaceholder = true
		}
	}

	// for Zig, add a placeholder identifier to help the LSP with incomplete syntax
	if needsPlaceholder && e.mode == mode.Zig {
		// Insert placeholder "X" at the cursor position
		lines := strings.Split(fileContent, "\n")
		if line >= 0 && line < len(lines) {
			currentLineStr := lines[line]
			if x >= 0 && x <= len(currentLineStr) {
				lines[line] = currentLineStr[:x] + "X" + currentLineStr[x:]
				fileContent = strings.Join(lines, "\n")
			}
		}
	}

	// update the physical file in temp workspace, if any
	if needsWorkspaceSetup(e.mode) && lspFilePath != absPath {
		os.WriteFile(lspFilePath, []byte(fileContent), 0644)
	}

	startSpinner("")

	uri := "file://" + lspFilePath
	if client.openedURI != uri {
		if err := client.DidOpen(uri, config.LanguageID, fileContent); err != nil {
			stopSpinner()
			client.openedURI = ""
			status.SetMessageAfterRedraw(fmt.Sprintf("%s error: %v", lspCommand, err))
			return false
		}
		client.openedURI = uri
		client.openedVersion = 1

	} else {
		client.openedVersion++
		if err := client.DidChange(uri, fileContent, client.openedVersion); err != nil {
			stopSpinner()
			client.openedURI = ""
			status.SetMessageAfterRedraw(fmt.Sprintf("%s error: %v", lspCommand, err))
			return false
		}
		if needsWorkspaceSetup(e.mode) {
			time.Sleep(50 * time.Millisecond)
		}
	}

	// detect trigger character (only if cursor is immediately after "." or "::")
	var triggerChar string
	if x > 0 && len(currentLine) > 0 {
		lastChar := string(currentLine[len(currentLine)-1])
		if lastChar == "." {
			triggerChar = "."
		} else if len(currentLine) >= 2 && currentLine[len(currentLine)-2:] == "::" {
			triggerChar = ":"
		}
	}

	// request completions, retrying until timeout for Rust/C/C++
	var items []LSPCompletionItem
	completionDeadline := time.Now().Add(lspCompletionWaitTimeout)
	retryDelay := 500 * time.Millisecond

	for {
		items, err = client.GetCompletions(uri, line, x, triggerChar)
		if err != nil {
			stopSpinner()
			client.openedURI = ""
			status.SetMessageAfterRedraw(fmt.Sprintf("%s error: %v", lspCommand, err))
			return false
		}

		// filter out the placeholder added for Zig
		if needsPlaceholder && e.mode == mode.Zig {
			filtered := make([]LSPCompletionItem, 0, len(items))
			for _, item := range items {
				if item.Label != "X" {
					filtered = append(filtered, item)
				}
			}
			items = filtered
		}

		items = sortAndFilterCompletions(items, currentLine, workspaceRoot, config.FileExtensions, e.mode)

		if len(items) > 0 || !needsWorkspaceSetup(e.mode) {
			break
		}

		// retry until deadline
		if time.Now().Add(retryDelay).Before(completionDeadline) {
			time.Sleep(retryDelay)
		} else {
			stopSpinner()
			const drawLines = true
			e.FullResetRedraw(c, status, drawLines, false)
			c.Draw()
			status.SetMessage("No completions found")
			status.Show(c, e)
			return true
		}
	}

	if len(items) > 10 {
		items = items[:10]
	}

	if len(items) == 0 {
		stopSpinner()
		const drawLines = true
		e.FullResetRedraw(c, status, drawLines, false)
		c.Draw()
		status.SetMessage("No completions found")
		status.Show(c, e)
		return true
	}

	// find the maximum label length for column alignment
	maxLabelLen := 0
	for _, item := range items {
		labelLen := len(item.Label)

		// for C/C++/Python, measure only the function name part
		if (e.mode == mode.C || e.mode == mode.Cpp || e.mode == mode.Python) && strings.Contains(item.Label, "(") {
			trimmedLabel := strings.TrimSpace(item.Label)
			if parenIdx := strings.Index(trimmedLabel, "("); parenIdx > 0 {
				labelLen = len(trimmedLabel[:parenIdx])
			}
		}

		if labelLen > maxLabelLen {
			maxLabelLen = labelLen
		}
	}

	// build menu choices with column alignment
	choices := make([]string, 0, len(items))
	for _, item := range items {
		label := item.Label

		// special formatting for C/C++/Python functions
		if (e.mode == mode.C || e.mode == mode.Cpp || e.mode == mode.Python) && strings.Contains(label, "(") {
			// Extract function name and parameters
			// Label format from clangd/pylsp: " printf(const char *restrict format, ...)" or "print(values, sep, end, file, flush)"
			trimmedLabel := strings.TrimSpace(label)
			if parenIdx := strings.Index(trimmedLabel, "("); parenIdx > 0 {
				funcName := trimmedLabel[:parenIdx]
				params := trimmedLabel[parenIdx:]

				// Format: "funcName    (params) -> returnType" with column alignment
				padding := max(maxLabelLen-len(funcName)+2, 2)

				if item.Detail != "" && len(item.Detail) < 80 {
					label = funcName + strings.Repeat(" ", padding) + params + " -> " + item.Detail
				} else {
					label = funcName + strings.Repeat(" ", padding) + params
				}
			}
		} else if item.Detail != "" && len(item.Detail) < 80 {
			padding := maxLabelLen - len(item.Label) + 2
			label += strings.Repeat(" ", padding) + item.Detail
		}
		choices = append(choices, label)
	}

	stopSpinner()

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

	// remove any existing parentheses from the insert text
	if parenIndex := strings.Index(insertText, "("); parenIndex > 0 {
		insertText = insertText[:parenIndex]
	}

	addParens := ""
	if (e.mode == mode.C || e.mode == mode.Cpp || e.mode == mode.Python) && strings.Contains(items[choice].Label, "(") {
		trimmedLabel := strings.TrimSpace(items[choice].Label)
		if startIdx := strings.Index(trimmedLabel, "("); startIdx >= 0 {
			if endIdx := strings.LastIndex(trimmedLabel, ")"); endIdx > startIdx {
				params := strings.TrimSpace(trimmedLabel[startIdx+1 : endIdx])
				if params == "" || params == "void" {
					addParens = "()"
				} else {
					addParens = "("
				}
			}
		}
	} else if items[choice].Detail != "" {
		detail := items[choice].Detail
		if strings.Contains(detail, "(") && strings.Contains(detail, ")") {
			startIdx := strings.Index(detail, "(")
			endIdx := strings.Index(detail, ")")
			if startIdx >= 0 && endIdx > startIdx {
				params := strings.TrimSpace(detail[startIdx+1 : endIdx])
				if params == "" || params == "&self" || params == "self" || params == "&mut self" || params == "mut self" {
					addParens = "()"
				} else {
					addParens = "("
				}
			}
		}
	}

	// fallback for functions/methods/constructors without parameter info
	if addParens == "" && (items[choice].Kind == lspKindMethod || items[choice].Kind == lspKindFunction || items[choice].Kind == lspKindConstructor) {
		addParens = "("
	}

	// clangd may return Kind=Text with a leading space in the Label for C/C++ functions
	// but not for variables like std::cout, so skip qualified names (lines containing ::)
	if addParens == "" && (e.mode == mode.C || e.mode == mode.Cpp) && items[choice].Kind <= lspKindText && strings.HasPrefix(items[choice].Label, " ") && !strings.Contains(currentLine, "::") {
		addParens = "("
	}

	// Shell commands take arguments separated by spaces, not parentheses
	if e.mode == mode.Shell && addParens != "" {
		addParens = " "
	}

	// Haskell uses space-separated arguments, not parentheses
	if e.mode == mode.Haskell && addParens != "" {
		addParens = " "
	}

	// For C/C++, don't add parens for qualified completions (e.g. std::cout)
	// unless the Kind explicitly indicates a callable
	if addParens != "" && (e.mode == mode.C || e.mode == mode.Cpp) && strings.Contains(currentLine, "::") {
		kind := items[choice].Kind
		if kind != lspKindMethod && kind != lspKindFunction && kind != lspKindConstructor {
			addParens = ""
		}
	}

	var charsToDelete int
	if items[choice].TextEdit != nil {
		rangeStart := items[choice].TextEdit.Range.Start.Character
		rangeEnd := items[choice].TextEdit.Range.End.Character
		charsToDelete = rangeEnd - rangeStart
	} else if currentWord != "" {
		charsToDelete = len([]rune(currentWord))
	} else {
		// extract prefix from the current line
		trimmedLine := strings.TrimSpace(currentLine)
		if strings.Contains(trimmedLine, ".") {
			parts := strings.Split(trimmedLine, ".")
			if len(parts) > 0 {
				prefix := parts[len(parts)-1]
				charsToDelete = len([]rune(prefix))
			}
		} else {
			words := strings.FieldsFunc(trimmedLine, func(r rune) bool {
				return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
			})
			if len(words) > 0 {
				prefix := words[len(words)-1]
				charsToDelete = len([]rune(prefix))
			}
		}
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
