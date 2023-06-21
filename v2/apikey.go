package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/xyproto/env/v2"
)

var (
	openAIKeyFilename = filepath.Join(userCacheDir, "o", "openai_key.txt") // just for caching the key, if it's entered via the menu
	openAIKey         = env.StrAlt("OPENAI_API_KEY", "OPENAI_KEY", env.Str("CHATGPT_API_KEY"))
)

func init() {
	// Delay checking if the OpenAI API Key file is there until after the main file has been read
	afterLoad = append(afterLoad, func() {
		if openAIKey == "" {
			openAIKey = ReadAPIKey()
		}
	})
}

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
	if noWriteToCache {
		return nil
	}
	return os.WriteFile(openAIKeyFilename, []byte(apiKey+"\n"), 0o600)
}
