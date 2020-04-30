// Language server protocol client

package main

import (
	"io"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
)

type LanguageServer struct {
	cmd        *exec.Cmd
	stdinPipe  io.WriteCloser
	stdoutPipe io.ReadCloser
	stderrPipe io.ReadCloser
}

func (ls *LanguageServer) Start() error {
	var err error
	cmd := exec.Command("gopls")
	ls.stdinPipe, err = cmd.StdinPipe()
	if err != nil {
		return err
	}
	ls.stdoutPipe, err = cmd.StdoutPipe()
	if err != nil {
		return err
	}
	ls.stderrPipe, err = cmd.StderrPipe()
	if err != nil {
		return err
	}
	ls.cmd = cmd
	return ls.cmd.Start()
}

func (ls *LanguageServer) Stop() error {
	ls.stdinPipe.Close()
	ls.stdoutPipe.Close()
	ls.stderrPipe.Close()
	return ls.cmd.Wait() // Wait for the process to quit, now that stdin is closed
}

func (ls *LanguageServer) Process(msg string) (string, error) {
	var req strings.Builder

	// LSP JSON header
	req.WriteString("Content-Length: ")
	req.WriteString(strconv.Itoa(len(msg)))
	req.WriteString("\r\n")
	req.WriteString("Content-Type: application/vscode-jsonrpc;charset=utf-8\r\n")
	req.WriteString("\r\n") // blank line signifies the end of the header

	// 	var request struct {
	// 		jsonrpc string `json:"jsonrpc"`
	// 		id int `json:"id"` // or string
	// 		method string `json:"method"`
	// 		params   []string `json:"params,omitempty"`
	// 	}
	// 	var response struct {
	// 	}
	// 	if err := json.NewDecoder(stdout).Decode(&request); err != nil {
	// 		log.Fatal(err)
	// 	}

	req.WriteString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "textDocument/didOpen",
		"params": {
			 
		}
	}`)

	_, err := io.WriteString(ls.stdinPipe, req.String())
	if err != nil {
		return "", err
	}

	bOut, err := ioutil.ReadAll(ls.stdoutPipe)
	if err != nil {
		return "", err
	}

	bErr, err := ioutil.ReadAll(ls.stderrPipe)
	if err != nil {
		return "", err
	}

	return string(bOut) + string(bErr), nil
}

func NewLanguageServer() *LanguageServer {
	return &LanguageServer{}
}

func query(JSON string) (string, error) {
	ls := NewLanguageServer()

	// Start the language server
	err := ls.Start()
	if err != nil {
		return "", err
	}

	// Pass in text to stdin
	result, err := ls.Process("ASDFASDF")
	if err != nil {
		return "", err
	}

	err = ls.Stop()
	if err != nil {
		return "", err
	}

	return "|" + result + "|", nil
}
