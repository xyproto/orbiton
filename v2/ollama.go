package main

import (
	"fmt"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/ollamaclient/v2"
	"github.com/xyproto/usermodel"
)

// CodeCompleter holds a model name, a boolean for if the model name was found and an Ollama client struct
type CodeCompleter struct {
	// The Ollama model that is used for code completion
	ModelName string

	// found a model name?
	foundModel bool

	// Ollama client, used for tab completion
	ollamaClient *ollamaclient.Config
}

// NewCodeCompleter returns the pointer to an empty CodeCompleter struct
func NewCodeCompleter() *CodeCompleter {
	return &CodeCompleter{}
}

// FindModel checks if a code completion model was specified either in $OLLAMA_MODEL,
// ~/.config/llm-manager/llm.conf or /etc/llm.conf.
// See https://github.com/xyproto/usermodel for more info.
// Returns true if an Ollama model name[:tag] was found.
func (cc *CodeCompleter) FindModel() bool {
	cc.ModelName = env.Str("OLLAMA_MODEL", usermodel.GetCodeModel())
	cc.foundModel = cc.ModelName != ""
	return cc.foundModel
}

// LoadModel tries to load the cc.ModelName by using the Ollama client
func (cc *CodeCompleter) LoadModel() error {
	if !cc.foundModel {
		return fmt.Errorf("could not find a code completion model name to use")
	}
	cc.ollamaClient = ollamaclient.New(cc.ModelName)
	cc.ollamaClient.Verbose = false
	const verbosePull = true
	if err := cc.ollamaClient.PullIfNeeded(verbosePull); err != nil {
		if ollamaHost := env.Str("OLLAMA_HOST"); ollamaHost != "" {
			return fmt.Errorf("could not fetch the %s model, check if Ollama is up and running at %s", cc.ModelName, ollamaHost)
		}
		return fmt.Errorf("could not fetch the %s model, check if Ollama is up and running locally or at $OLLAMA_HOST", cc.ModelName)
	}
	cc.ollamaClient.SetReproducible()
	return nil
}

// FoundModel returns true if the name of the model to use for code completion was found
func (cc *CodeCompleter) FoundModel() bool {
	return cc.foundModel
}

// Loaded returns true if the ollama client could be used and the code completion model could be loaded
func (cc *CodeCompleter) Loaded() bool {
	return cc.ollamaClient != nil
}

// CompleteBetween tries to return generated code that fits between the given codeStart and codeEnd strings
func (cc *CodeCompleter) CompleteBetween(codeStart, codeEnd string) (string, error) {
	response, err := cc.ollamaClient.GetBetweenResponse(codeStart, codeEnd)
	if err != nil {
		return "", err
	}
	return response.Response, nil
}
