package usermodel

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/files"
)

type Task string

const (
	llmManagerExecutable = "llm-manager"
	defaultModelFallback = "gemma3:4b"

	// Tasks
	ChatTask           = "chat"
	CodeTask           = "code"
	CodeCompletionTask = "code-completion"
	TestTask           = "test"
	TextGenerationTask = "text-generation"
	ToolUseTask        = "tool-use"
	TranslationTask    = "translation"
	VisionTask         = "vision"
)

var (
	DefaultModels = map[Task]string{
		"chat":            "llama3.2:3b",
		"code":            "deepseek-coder-v2:16b",
		"code-completion": "deepseek-coder-v2:16b",
		"test":            "tinyllama:1b",
		"text-generation": "gemma3:4b",
		"tool-use":        "llama3.2:3b",
		"translation":     "mixtral:8x7b",
		"vision":          "llava:7b",
	}
)

func GetChatModel() string           { return Get(ChatTask) }
func GetCodeModel() string           { return Get(CodeTask) }
func GetCodeCompletionModel() string { return Get(CodeCompletionTask) }
func GetTestModel() string           { return Get(TestTask) }
func GetTextGenerationModel() string { return Get(TextGenerationTask) }
func GetToolUseModel() string        { return Get(ToolUseTask) }
func GetTranslationModel() string    { return Get(TranslationTask) }
func GetVisionModel() string         { return Get(VisionTask) }

func AvailableTasks() []Task {
	return []Task{ChatTask, CodeTask, CodeCompletionTask, TestTask, TextGenerationTask, ToolUseTask, TranslationTask, VisionTask}
}

func defaultModel(task Task) string {
	if model, ok := DefaultModels[task]; ok {
		return model
	}
	return defaultModelFallback
}

// Get attempts to retrieve the model name using llm-manager.
// If llm-manager is not available or the command fails, it falls back to the Default*Model variables.
func Get(task Task) string {
	var (
		data             []byte
		err              error
		found            bool
		userConfFilename = env.ExpandUser("~/.config/llm-manager/llm.conf")
		rootConfFilename = "/etc/llm.conf"
	)
	if !files.Exists(userConfFilename) {
		userConfFilename = ""
	}
	if userConfFilename != "" {
		data, err = os.ReadFile(userConfFilename)
		if err == nil && len(bytes.TrimSpace(data)) > 0 { // success
			found = true
		}
	}
	if !files.Exists(rootConfFilename) {
		rootConfFilename = ""
	}
	if !found && rootConfFilename != "" {
		data, err = os.ReadFile(rootConfFilename)
		if err == nil && len(bytes.TrimSpace(data)) > 0 { // success
			found = true
		}
	}
	if found { // found a configuration file with data, and was able to read the file
		for _, line := range strings.Split(string(data), " ") {
			trimmedLine := strings.TrimSpace(line)
			if strings.HasPrefix(trimmedLine, string(task)) && strings.Count(trimmedLine, "=") == 1 {
				fields := strings.SplitN(trimmedLine, "=", 2)
				value := strings.TrimSpace(strings.ReplaceAll(strings.TrimSpace(fields[1]), "#", ""))
				if value != "" {
					return value
				}
			}
		}
	}
	if llmManagerPath := files.WhichCached(llmManagerExecutable); llmManagerPath != "" {
		cmd := exec.Command(llmManagerPath, "get", string(task))
		if outputBytes, err := cmd.Output(); err == nil { // success
			if output := strings.TrimSpace(string(outputBytes)); output != "" {
				return output
			}
		}
	}
	return defaultModel(task)
}
