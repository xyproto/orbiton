package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/xyproto/env/v2"
)

// KeyHolder holds an API key and a cache filename
type KeyHolder struct {
	Key      string
	Filename string
}

var openAIKeyHolder = NewKeyHolder()

// NewKeyHolder creates a new struct for storing the OpenAI API Key + a key cache filename
// Can return nil if the key ends up being empty!
// Use NewKeyHolderWithKey instead to allow empty keys and never return nil.
func NewKeyHolder() *KeyHolder {
	key := env.StrAlt("OPENAI_API_KEY", "OPENAI_KEY", env.Str("CHATGPT_API_KEY"))
	kh := NewKeyHolderWithKey(key)
	if kh.Key == "" {
		if !kh.ReadAPIKey() {
			return nil // !
		}
	}
	return kh
}

// NewKeyHolderWithKey creates a new struct for storing the OpenAI API Key + a key cache filename,
// and takes an initial key string. Will always return a struct, never nil.
func NewKeyHolderWithKey(key string) *KeyHolder {
	var kh KeyHolder
	kh.Filename = filepath.Join(userCacheDir, "o", "openai_key.txt") // just for caching the key, if it's entered via the menu
	kh.Key = key
	return &kh
}

// ReadAPIKey tries to read the Open AI API Key from file.
// An empty string is returned if the file could not be read.
// Return true if a key is exists, or false if the key is empty.
func (kh *KeyHolder) ReadAPIKey() bool {
	if kh.Filename == "" {
		return kh.Key != ""
	}
	data, err := os.ReadFile(kh.Filename)
	if err != nil {
		return kh.Key != ""
	}
	kh.Key = strings.TrimSpace(string(data))
	return kh.Key != ""
}

// WriteAPIKey writes the given OpenAI API key to file
func (kh *KeyHolder) WriteAPIKey() error {
	if noWriteToCache {
		return nil
	}
	if kh.Key == "" {
		return errors.New("no API Key to write")
	}
	if kh.Filename == "" {
		return errors.New("no API filename to write to")
	}
	return os.WriteFile(kh.Filename, []byte(kh.Key+"\n"), 0o600)
}
