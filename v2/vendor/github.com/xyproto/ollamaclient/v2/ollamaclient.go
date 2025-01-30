// Package ollamaclient can be used for communicating with the Ollama service
package ollamaclient

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/xyproto/env/v2"
)

const (
	defaultModel       = "gemma2:2b"
	defaultHTTPTimeout = 10 * time.Minute // per HTTP request to Ollama
	defaultFixedSeed   = 256              // for when generated output should not be random, but have temperature 0 and a specific seed
	defaultPullTimeout = 48 * time.Hour   // pretty generous, in case someone has a poor connection
	mimeJSON           = "application/json"
)

// RequestOptions holds the seed and temperature
type RequestOptions struct {
	Seed          int     `json:"seed"`
	Temperature   float64 `json:"temperature"`
	ContextLength int64   `json:"num_ctx,omitempty"`
}

// Model represents a downloaded model
type Model struct {
	Modified time.Time `json:"modified_at"`
	Name     string    `json:"name"`
	Digest   string    `json:"digest"`
	Size     int64     `json:"size"`
}

// ListResponse represents the response data from the tag API call
type ListResponse struct {
	Models []Model `json:"models"`
}

// VersionResponse represents the response data containing the Ollama version
type VersionResponse struct {
	Version string `json:"version"`
}

// Config represents configuration details for communicating with the Ollama API
type Config struct {
	ServerAddr                string
	ModelName                 string
	SeedOrNegative            int
	TemperatureIfNegativeSeed float64
	PullTimeout               time.Duration
	HTTPTimeout               time.Duration
	TrimSpace                 bool
	Verbose                   bool
	ContextLength             int64
	SystemPrompt              string
	Tools                     []Tool
}

// Cache is used for caching reproducible results from Ollama (seed -1, temperature 0)
var Cache *bigcache.BigCache

// InitCache initializes the BigCache cache
func InitCache() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	config := bigcache.DefaultConfig(24 * time.Hour)
	config.HardMaxCacheSize = 256 // MB
	config.StatsEnabled = false
	config.Verbose = false
	c, err := bigcache.New(ctx, config)
	if err != nil {
		return err
	}
	Cache = c
	return nil
}

// New initializes a new Config using environment variables
func New(optionalModel ...string) *Config {
	model := defaultModel
	if len(optionalModel) > 0 {
		model = optionalModel[0]
	}
	oc := Config{
		ServerAddr:                env.Str("OLLAMA_HOST", "http://localhost:11434"),
		ModelName:                 env.Str("OLLAMA_MODEL", model),
		SeedOrNegative:            defaultFixedSeed,
		TemperatureIfNegativeSeed: 0.8,
		PullTimeout:               defaultPullTimeout,
		HTTPTimeout:               defaultHTTPTimeout,
		TrimSpace:                 true,
		Verbose:                   env.Bool("OLLAMA_VERBOSE"),
	}
	oc.modelSpecificAdjustments()
	return &oc
}

// NewConfig initializes a new Config using a specified model, address (like http://localhost:11434) and a verbose bool
func NewConfig(serverAddr, modelName string, seedOrNegative int, temperatureIfNegativeSeed float64, pTimeout, hTimeout time.Duration, trimSpace, verbose bool) *Config {
	oc := Config{
		ServerAddr:                serverAddr,
		ModelName:                 modelName,
		SeedOrNegative:            seedOrNegative,
		TemperatureIfNegativeSeed: temperatureIfNegativeSeed,
		PullTimeout:               pTimeout,
		HTTPTimeout:               hTimeout,
		TrimSpace:                 trimSpace,
		Verbose:                   verbose,
	}
	oc.modelSpecificAdjustments()
	return &oc
}

// modelSpecificAdjustments will make adjustments for some specific model names
func (oc *Config) modelSpecificAdjustments() {
	switch oc.ModelName {
	case "llama3-gradient":
		oc.ContextLength = 256000 // can be set as high as 1M+, but this will requre 100GB+ memory
	}
}

// SetReproducible configures the generated output to be reproducible, with temperature 0 and a specific seed.
// It takes an optional random seed.
func (oc *Config) SetReproducible(optionalSeed ...int) {
	if len(optionalSeed) > 0 {
		oc.SeedOrNegative = optionalSeed[0]
		return
	}
	oc.SeedOrNegative = defaultFixedSeed
}

// SetSystemPrompt sets the system prompt for this Ollama config
func (oc *Config) SetSystemPrompt(prompt string) {
	oc.SystemPrompt = prompt
}

// SetRandom configures the generated output to not be reproducible
func (oc *Config) SetRandom() {
	oc.SeedOrNegative = -1
}

// SetContextLength sets the context lenght for this Ollama config
func (oc *Config) SetContextLength(contextLength int64) {
	oc.ContextLength = contextLength
}

// SetTool sets the tools for this Ollama config
func (oc *Config) SetTool(tool Tool) {
	oc.Tools = append(oc.Tools, tool)
}

// GetChatResponse sends a request to the Ollama API and returns the generated response
func (oc *Config) GetChatResponse(promptAndOptionalImages ...string) (OutputResponse, error) {
	var (
		temperature float64
		seed        = oc.SeedOrNegative
	)
	if len(promptAndOptionalImages) == 0 {
		return OutputResponse{}, errors.New("at least one prompt must be given (and then optionally, base64 encoded JPG or PNG image strings)")
	}
	prompt := promptAndOptionalImages[0]
	var images []string
	if len(promptAndOptionalImages) > 1 {
		images = promptAndOptionalImages[1:]
	}
	if seed < 0 {
		temperature = oc.TemperatureIfNegativeSeed
	}
	messages := []Message{}
	if oc.SystemPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: oc.SystemPrompt,
		})
	}
	messages = append(messages, Message{
		Role:    "user",
		Content: prompt,
	})
	var reqBody GenerateChatRequest
	if len(images) > 0 {
		reqBody = GenerateChatRequest{
			Model:    oc.ModelName,
			Messages: messages,
			Images:   images,
			Tools:    oc.Tools,
			Options: RequestOptions{
				Seed:        seed,        // set to -1 to make it random
				Temperature: temperature, // set to 0 together with a specific seed to make output reproducible
			},
		}
	} else {
		reqBody = GenerateChatRequest{
			Model:    oc.ModelName,
			Messages: messages,
			Tools:    oc.Tools,
			Options: RequestOptions{
				Seed:        seed,        // set to -1 to make it random
				Temperature: temperature, // set to 0 together with a specific seed to make output reproducible
			},
		}
	}
	if oc.ContextLength != 0 {
		reqBody.Options.ContextLength = oc.ContextLength
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return OutputResponse{}, err
	}
	if oc.Verbose {
		fmt.Printf("Sending request to %s/api/chat: %s\n", oc.ServerAddr, string(reqBytes))
	}
	HTTPClient := &http.Client{
		Timeout: oc.HTTPTimeout,
	}
	resp, err := HTTPClient.Post(oc.ServerAddr+"/api/chat", mimeJSON, bytes.NewBuffer(reqBytes))
	if err != nil {
		return OutputResponse{}, err
	}
	defer resp.Body.Close()
	var res = OutputResponse{}
	var sb strings.Builder
	decoder := json.NewDecoder(resp.Body)
	for {
		var genResp GenerateChatResponse
		if err := decoder.Decode(&genResp); err != nil {
			break
		}
		sb.WriteString(genResp.Message.Content)
		if genResp.Done {
			res.Role = genResp.Message.Role
			res.ToolCalls = genResp.Message.ToolCalls
			res.PromptTokens = genResp.PromptEvalCount
			res.ResponseTokens = genResp.EvalCount
			break
		}
	}
	res.Response = strings.TrimPrefix(sb.String(), "\n")
	if oc.TrimSpace {
		res.Response = strings.TrimSpace(res.Response)
	}
	return res, nil
}

// GetOutputChatVision sends a request to the Ollama API and returns the generated response.
// It is similar to GetChatResponse, but it adds the images into the Message struct before sending them.
func (oc *Config) GetOutputChatVision(promptAndOptionalImages ...string) (string, error) {
	var (
		temperature float64
		seed        = oc.SeedOrNegative
	)
	if len(promptAndOptionalImages) == 0 {
		return "", errors.New("at least one prompt must be given (and then optionally, base64 encoded JPG or PNG image strings)")
	}
	prompt := promptAndOptionalImages[0]
	var images []string
	if len(promptAndOptionalImages) > 1 {
		images = promptAndOptionalImages[1:]
	}
	if seed < 0 {
		temperature = oc.TemperatureIfNegativeSeed
	}
	messages := []Message{}
	if oc.SystemPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: oc.SystemPrompt,
		})
	}
	messages = append(messages, Message{
		Role:    "user",
		Content: prompt,
		Images:  images,
	})

	reqBody := GenerateChatRequest{
		Model:    oc.ModelName,
		Messages: messages,
		Tools:    oc.Tools,
		Options: RequestOptions{
			Seed:        seed,        // set to -1 to make it random
			Temperature: temperature, // set to 0 together with a specific seed to make output reproducible
		},
	}

	if oc.ContextLength != 0 {
		reqBody.Options.ContextLength = oc.ContextLength
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	if oc.Verbose {
		fmt.Printf("Sending request to %s/api/chat: %s\n", oc.ServerAddr, string(reqBytes))
	}
	HTTPClient := &http.Client{
		Timeout: oc.HTTPTimeout,
	}
	resp, err := HTTPClient.Post(oc.ServerAddr+"/api/chat", mimeJSON, bytes.NewBuffer(reqBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var res = ""
	var sb strings.Builder
	decoder := json.NewDecoder(resp.Body)
	for {
		var genResp GenerateChatResponse
		if err := decoder.Decode(&genResp); err != nil {
			break
		}
		sb.WriteString(genResp.Message.Content)
		if genResp.Done {
			d, _ := json.Marshal(genResp.Message.ToolCalls)
			res = string(d)
			break
		}
	}
	res = strings.TrimPrefix(sb.String(), "\n")
	if oc.TrimSpace {
		res = strings.TrimSpace(res)
	}
	return res, nil
}

// GetResponse sends a request to the Ollama API and returns the generated response
func (oc *Config) GetResponse(promptAndOptionalImages ...string) (OutputResponse, error) {
	var (
		temperature float64
		cacheKey    string
		seed        = oc.SeedOrNegative
	)
	if len(promptAndOptionalImages) == 0 {
		return OutputResponse{}, errors.New("at least one prompt must be given (and then optionally, base64 encoded JPG or PNG image strings)")
	}
	prompt := promptAndOptionalImages[0]
	var images []string
	if len(promptAndOptionalImages) > 1 {
		images = promptAndOptionalImages[1:]
	}
	if seed < 0 {
		temperature = oc.TemperatureIfNegativeSeed
	} else {
		temperature = 0 // Since temperature is set to 0 when seed >=0
		// The cache is only used for fixed seeds and a temperature of 0
		keyData := struct {
			Prompts     []string
			ModelName   string
			Seed        int
			Temperature float64
		}{
			Prompts:     promptAndOptionalImages,
			ModelName:   oc.ModelName,
			Seed:        seed,
			Temperature: temperature,
		}
		keyDataBytes, err := json.Marshal(keyData)
		if err != nil {
			return OutputResponse{}, err
		}
		hash := sha256.Sum256(keyDataBytes)
		cacheKey = hex.EncodeToString(hash[:])
		if Cache == nil {
			if err := InitCache(); err != nil {
				return OutputResponse{}, err
			}
		}
		if entry, err := Cache.Get(cacheKey); err == nil {
			var res OutputResponse
			err = json.Unmarshal(entry, &res)
			if err != nil {
				return OutputResponse{}, err
			}
			return res, nil
		}
	}
	var reqBody GenerateRequest
	if len(images) > 0 {
		reqBody = GenerateRequest{
			Model:  oc.ModelName,
			Prompt: prompt,
			Images: images,
			Options: RequestOptions{
				Seed:        seed,        // set to -1 to make it random
				Temperature: temperature, // set to 0 together with a specific seed to make output reproducible
			},
		}
	} else {
		reqBody = GenerateRequest{
			Model:  oc.ModelName,
			Prompt: prompt,
			Options: RequestOptions{
				Seed:        seed,        // set to -1 to make it random
				Temperature: temperature, // set to 0 together with a specific seed to make output reproducible
			},
		}
	}
	if oc.ContextLength != 0 {
		reqBody.Options.ContextLength = oc.ContextLength
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return OutputResponse{}, err
	}
	if oc.Verbose {
		fmt.Printf("Sending request to %s/api/generate: %s\n", oc.ServerAddr, string(reqBytes))
	}
	HTTPClient := &http.Client{
		Timeout: oc.HTTPTimeout,
	}
	resp, err := HTTPClient.Post(oc.ServerAddr+"/api/generate", mimeJSON, bytes.NewBuffer(reqBytes))
	if err != nil {
		return OutputResponse{}, err
	}
	defer resp.Body.Close()
	response := OutputResponse{
		Role: "assistant",
	}
	var sb strings.Builder
	decoder := json.NewDecoder(resp.Body)
	for {
		var genResp GenerateResponse
		if err := decoder.Decode(&genResp); err != nil {
			break
		}
		sb.WriteString(genResp.Response)
		if genResp.Done {
			response.PromptTokens = genResp.PromptEvalCount
			response.ResponseTokens = genResp.EvalCount
			break
		}
	}
	outputString := strings.TrimPrefix(sb.String(), "\n")
	if oc.TrimSpace {
		outputString = strings.TrimSpace(outputString)
	}
	response.Response = outputString
	if cacheKey != "" {
		data, err := json.Marshal(response)
		if err != nil {
			return OutputResponse{}, err
		}
		Cache.Set(cacheKey, data)
	}
	return response, nil
}

// GetOutput sends a request to the Ollama API and returns the generated output string
func (oc *Config) GetOutput(promptAndOptionalImages ...string) (string, error) {
	resp, err := oc.GetResponse(promptAndOptionalImages...)
	if err != nil {
		return "", err
	}
	return resp.Response, nil
}

// MustOutput returns the generated output string from Ollama, or the error as a string if not
func (oc *Config) MustOutput(promptAndOptionalImages ...string) string {
	output, err := oc.GetOutput(promptAndOptionalImages...)
	if err != nil {
		return err.Error()
	}
	return output
}

// MustGetResponse returns the response from Ollama, or an error if not
func (oc *Config) MustGetResponse(promptAndOptionalImages ...string) OutputResponse {
	resp, err := oc.GetResponse(promptAndOptionalImages...)
	if err != nil {
		return OutputResponse{Error: err.Error()}
	}
	return resp
}

// MustGetChatResponse returns the response from Ollama, or a response with an error if not
func (oc *Config) MustGetChatResponse(promptAndOptionalImages ...string) OutputResponse {
	output, err := oc.GetChatResponse(promptAndOptionalImages...)
	if err != nil {
		return OutputResponse{Error: err.Error()}
	}
	return output
}

// List collects info about the currently downloaded models
func (oc *Config) List() ([]string, map[string]time.Time, map[string]int64, error) {
	if oc.Verbose {
		fmt.Printf("Sending request to %s/api/tags\n", oc.ServerAddr)
	}
	resp, err := http.Get(oc.ServerAddr + "/api/tags")
	if err != nil {
		return nil, nil, nil, err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	var listResp ListResponse
	if err := decoder.Decode(&listResp); err != nil {
		return nil, nil, nil, err
	}
	var names []string
	modifiedMap := make(map[string]time.Time)
	sizeMap := make(map[string]int64)
	for _, model := range listResp.Models {
		names = append(names, model.Name)
		modifiedMap[model.Name] = model.Modified
		sizeMap[model.Name] = model.Size
	}
	return names, modifiedMap, sizeMap, nil
}

// SizeOf returns the current size of the given model in bytes,
// or returns (-1, err) if it the model can't  be found.
func (oc *Config) SizeOf(model string) (int64, error) {
	model = strings.TrimSpace(model)
	if !strings.Contains(model, ":") {
		model += ":latest"
	}
	names, _, sizeMap, err := oc.List()
	if err != nil {
		return 0, err
	}
	for _, name := range names {
		if name == model {
			return sizeMap[name], nil
		}
	}
	return -1, fmt.Errorf("could not find model: %s", model)
}

// Has returns true if the given model exists
func (oc *Config) Has(model string) (bool, error) {
	model = strings.TrimSpace(model)
	if !strings.Contains(model, ":") {
		model += ":latest"
	}
	if names, _, _, err := oc.List(); err == nil { // success
		for _, name := range names {
			if name == model {
				return true, nil
			}
		}
	} else {
		return false, err
	}
	return false, nil // could list models, but could not find the given model name
}

// HasModel returns true if the configured model exists
func (oc *Config) HasModel() (bool, error) {
	return oc.Has(oc.ModelName)
}

// PullIfNeeded pulls a model, but only if it's not already there.
// While Pull downloads/updates the model regardless.
// Also takes an optional bool for if progress bars should be used when models are being downloaded.
func (oc *Config) PullIfNeeded(optionalVerbose ...bool) error {
	if found, err := oc.HasModel(); err != nil {
		return err
	} else if !found {
		if _, err := oc.Pull(optionalVerbose...); err != nil {
			return err
		}
	}
	return nil
}

// CloseCache signals the shutdown of the cache
func CloseCache() {
	if Cache != nil {
		Cache.Close()
	}
}

// ClearCache removes the current cache entries
func ClearCache() {
	if Cache != nil {
		Cache.Reset()
	}
}

// DescribeImages can load a slice of image filenames into base64 encoded strings
// and build a prompt that starts with "Describe this/these image(s):" followed
// by the encoded images, and return a result. Typically used together with the "llava" model.
func (oc *Config) DescribeImages(imageFilenames []string, desiredWordCount int) (string, error) {
	var errNoImages = errors.New("must be given at least one image file to describe")

	if len(imageFilenames) == 0 {
		return "", errNoImages
	}

	var images []string
	for _, imageFilename := range imageFilenames {
		base64image, err := Base64EncodeFile(imageFilename)
		if err != nil {
			return "", fmt.Errorf("could not base64 encode %s: %v", imageFilename, err)
		}
		// append the base64 encoded image to the "images" string slice
		images = append(images, base64image)
	}

	var prompt string
	switch len(images) {
	case 0:
		return "", errNoImages
	case 1:
		if desiredWordCount > 0 {
			prompt = fmt.Sprintf("Describe this image using a maximum of %d words:", desiredWordCount)
		} else {
			prompt = "Describe this image:"
		}
	default:
		if desiredWordCount > 0 {
			prompt = fmt.Sprintf("Describe these images using a maximum of %d words:", desiredWordCount)
		} else {
			prompt = "Describe these images:"
		}
	}

	promptAndImages := append([]string{prompt}, images...)

	return oc.GetOutput(promptAndImages...)
}

// CreateModel creates a new model based on a Modelfile
func (oc *Config) CreateModel(name, modelfile string) error {
	reqBody := map[string]string{"name": name, "modelfile": modelfile}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	resp, err := http.Post(oc.ServerAddr+"/api/create", mimeJSON, bytes.NewBuffer(reqBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// CopyModel duplicates an existing model under a new name
func (oc *Config) CopyModel(source, destination string) error {
	reqBody := map[string]string{"source": source, "destination": destination}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	resp, err := http.Post(oc.ServerAddr+"/api/copy", mimeJSON, bytes.NewBuffer(reqBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// DeleteModel removes a model from the server
func (oc *Config) DeleteModel(name string) error {
	reqBody := map[string]string{"name": name}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("DELETE", oc.ServerAddr+"/api/delete", bytes.NewBuffer(reqBytes))
	if err != nil {
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
