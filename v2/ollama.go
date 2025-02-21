package main

import (
	"fmt"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/ollamaclient/v2"
	"github.com/xyproto/usermodel"
)

// How many lines of context above and below should the tab completion try to use?
const ollamaContextLines = 20

var (
	// The Ollama model that is used for code completion
	codeCompletionModel string

	// Ollama client, used for tab completion
	ollamaClient *ollamaclient.Config
)

func GetOllamaCodeModel() bool {
	codeCompletionModel = env.Str("OLLAMA_MODEL", usermodel.GetCodeModel())
	return codeCompletionModel != ""
}

func LoadOllama(shouldLoad bool) error {
	if shouldLoad {
		ollamaClient = ollamaclient.New(codeCompletionModel)
		ollamaClient.Verbose = false
		const verbosePull = true
		if err := ollamaClient.PullIfNeeded(verbosePull); err != nil {
			if ollamaHost := env.Str("OLLAMA_HOST"); ollamaHost != "" {
				return fmt.Errorf("could not fetch the %s model, is Ollama up and running at %s?\n", codeCompletionModel, ollamaHost)
			} else {
				return fmt.Errorf("could not fetch the %s model, is Ollama running locally? (or is OLLAMA_HOST set?)\n", codeCompletionModel)
			}
			return err
		}
		ollamaClient.SetReproducible()
	}
	return nil
}
