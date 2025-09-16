# usermodel

Given a task string (like `code-completion`), return the user-configured Ollama model for this task.

The user-configured Ollama model is found by examining `~/.config/llm-manager/llm.conf` and `/etc/llm.conf` and if that does not work out, by executing [`llm-manager`](https://github.com/xyproto/llm-manager).

If no user-configured model is available, return a default model.

### Example use

```go
package main

import (
    "fmt"

    "github.com/xyproto/usermodel"
)

func main() {
    fmt.Println(usermodel.GetVisionModel())
}
```

### Exported types

```go
type Task string
```

### Exported constants

```go
ChatTask           = "chat"
CodeTask           = "code"
CodeCompletionTask = "code-completion"
TestTask           = "test"
TextGenerationTask = "text-generation"
ToolUseTask        = "tool-use"
TranslationTask    = "translation"
VisionTask         = "vision"
```

### Exported functions

```go
func AvailableTasks() []Task
func GetChatModel() string
func GetCodeModel() string
func GetCodeCompletionModel() string
func GetTestModel() string
func GetTextGenerationModel() string
func GetToolUseModel() string
func GetTranslationModel() string
func GetVisionModel() string
func Get(task Task) string
```

### General info

* Version: 1.2.3
* License: BSD-3
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
