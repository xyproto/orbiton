package usermodel

import (
	"os/exec"
	"strings"

	"github.com/xyproto/files"
)

type Task string

const (
	llmManagerExecutable = "llm-manager"
	defaultModelFallback = "gemma2:2b"

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
		"code":            "deepseek-coder:1.3b",
		"code-completion": "deepseek-coder:1.3b",
		"test":            "tinyllama:1b",
		"text-generation": "gemma2:2b",
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
	llmManagerPath := files.WhichCached(llmManagerExecutable)
	if llmManagerPath == "" {
		return defaultModel(task)
	}
	cmd := exec.Command(llmManagerPath, "get", string(task))
	outputBytes, err := cmd.Output()
	if err != nil {
		return defaultModel(task)
	}
	output := strings.TrimSpace(string(outputBytes))
	if output == "" {
		return defaultModel(task)
	}
	return output
}
