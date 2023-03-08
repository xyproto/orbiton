package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/xyproto/env/v2"
)

var (
	openAIKeyFilename = filepath.Join(userCacheDir, "o", "openai_key.txt")
	openAIKey         = env.StrAlt("CHATGPT_API_KEY", "OPENAI_API_KEY", env.Str("OPENAI_KEY", ReadAPIKey()))
)

// ReadAPIKey tries to read the Open AI API Key from file.
// An empty string is returned if the file could not be read.
func ReadAPIKey() string {
	data, err := os.ReadFile(openAIKeyFilename)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// WriteAPIKey writes the given OpenAI API key to file
func WriteAPIKey(apiKey string) error {
	return os.WriteFile(openAIKeyFilename, []byte(openAIKey+"\n"), 0o600)
}
