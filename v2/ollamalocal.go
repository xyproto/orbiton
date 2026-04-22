package main

// Minimal local Ollama client. Implements only the API calls that Orbiton
// needs (list models, pull model, generate a response for a prompt), so the
// heavier github.com/xyproto/ollamaclient/v2 dependency (which in turn pulls
// in bigcache and net/http) is not required.

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/xyproto/env/v2"
)

const (
	ollamaDefaultAddr = "http://localhost:11434"
	ollamaDefaultSeed = 256
	ollamaGenTimeout  = 10 * time.Minute
	ollamaListTimeout = 10 * time.Second
	ollamaPullTimeout = 48 * time.Hour
)

// ollamaConfig is the subset of ollamaclient.Config that Orbiton uses.
type ollamaConfig struct {
	ServerAddr  string
	ModelName   string
	Seed        int
	Temperature float64
	Verbose     bool
}

// newOllamaConfig mirrors ollamaclient.New.
func newOllamaConfig(model string) *ollamaConfig {
	return &ollamaConfig{
		ServerAddr:  env.Str("OLLAMA_HOST", ollamaDefaultAddr),
		ModelName:   env.Str("OLLAMA_MODEL", model),
		Seed:        ollamaDefaultSeed,
		Temperature: 0,
		Verbose:     env.Bool("OLLAMA_VERBOSE"),
	}
}

// SetReproducible forces deterministic output with a fixed seed.
func (oc *ollamaConfig) SetReproducible() {
	oc.Seed = ollamaDefaultSeed
	oc.Temperature = 0
}

// hasModel calls /api/tags and returns true if oc.ModelName is available.
func (oc *ollamaConfig) hasModel() (bool, error) {
	resp, err := httpGet(oc.ServerAddr+"/api/tags", nil, ollamaListTimeout)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return false, fmt.Errorf("ollama /api/tags: HTTP %d", resp.StatusCode)
	}
	var list struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return false, err
	}
	want := oc.ModelName
	if !strings.Contains(want, ":") {
		want += ":latest"
	}
	for _, m := range list.Models {
		if m.Name == want {
			return true, nil
		}
	}
	return false, nil
}

// PullIfNeeded pulls the configured model if not already present. The verbose
// flag is accepted for API parity but progress output is minimal.
func (oc *ollamaConfig) PullIfNeeded(verbose bool) error {
	found, err := oc.hasModel()
	if err != nil {
		return err
	}
	if found {
		return nil
	}
	reqBytes, err := json.Marshal(map[string]any{
		"name":   oc.ModelName,
		"stream": true,
	})
	if err != nil {
		return err
	}
	resp, err := httpPostJSON(oc.ServerAddr+"/api/pull", reqBytes, ollamaPullTimeout)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("ollama /api/pull: HTTP %d", resp.StatusCode)
	}
	// Drain the stream; abort on first error. Each line is a JSON object
	// with a "status" field; "success" marks completion.
	dec := json.NewDecoder(resp.Body)
	for {
		var ev struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		}
		if err := dec.Decode(&ev); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if ev.Error != "" {
			return fmt.Errorf("ollama pull: %s", ev.Error)
		}
		if ev.Status == "success" {
			break
		}
	}
	return nil
}

// GetSimpleResponse posts a prompt to /api/generate and returns the
// concatenated response text.
func (oc *ollamaConfig) GetSimpleResponse(prompt string) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("prompt is empty")
	}
	reqBytes, err := json.Marshal(map[string]any{
		"model":  oc.ModelName,
		"prompt": prompt,
		"options": map[string]any{
			"seed":        oc.Seed,
			"temperature": oc.Temperature,
		},
	})
	if err != nil {
		return "", err
	}
	resp, err := httpPostJSON(oc.ServerAddr+"/api/generate", reqBytes, ollamaGenTimeout)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("ollama /api/generate: HTTP %d", resp.StatusCode)
	}
	// /api/generate streams newline-separated JSON objects with a "response"
	// field until "done" is true.
	dec := json.NewDecoder(resp.Body)
	var sb strings.Builder
	for {
		var ev struct {
			Response string `json:"response"`
			Done     bool   `json:"done"`
			Error    string `json:"error"`
		}
		if err := dec.Decode(&ev); err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		if ev.Error != "" {
			return "", fmt.Errorf("ollama generate: %s", ev.Error)
		}
		sb.WriteString(ev.Response)
		if ev.Done {
			break
		}
	}
	return strings.TrimPrefix(sb.String(), "\n"), nil
}
