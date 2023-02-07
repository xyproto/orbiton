# go-gpt3

An OpenAI GPT-3 API client enabling Go/Golang programs to interact with the gpt3 APIs.

Supports using the completion APIs with or without streaming.

[![PkgGoDev](https://pkg.go.dev/badge/github.com/PullRequestInc/go-gpt3)](https://pkg.go.dev/github.com/PullRequestInc/go-gpt3)

## Usage

Simple usage to call the main gpt-3 API, completion:

```go
client := gpt3.NewClient(apiKey)
resp, err := client.Completion(ctx, gpt3.CompletionRequest{
    Prompt: []string{"2, 3, 5, 7, 11,"},
})

fmt.Print(resp.Choices[0].Text)
// prints " 13, 17, 19, 23, 29, 31", etc
```

## Documentation

Check out the go docs for more detailed documentation on the types and methods provided: https://pkg.go.dev/github.com/PullRequestInc/go-gpt3

### Full Examples

Try out any of these examples with putting the contents in a `main.go` and running `go run main.go`.
I would recommend using go modules in which case you will also need to run `go mod init` within your
test repo. Alternatively you can clone this repo and run the test script with `go run cmd/test/main.go`.

You will also need to have a `.env` file that looks like this to use these examples:

```
API_KEY=<openAI API Key>
```

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/PullRequestInc/go-gpt3"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Fatalln("Missing API KEY")
	}

	ctx := context.Background()
	client := gpt3.NewClient(apiKey)

	resp, err := client.Completion(ctx, gpt3.CompletionRequest{
		Prompt:    []string{"The first thing you should know about javascript is"},
		MaxTokens: gpt3.IntPtr(30),
		Stop:      []string{"."},
		Echo:      true,
	})
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(resp.Choices[0].Text)
}
```

## Support

- [x] List Engines API
- [x] Get Engine API
- [x] Completion API (this is the main gpt-3 API)
- [x] Streaming support for the Completion API
- [x] Document Search API
- [x] Overriding default url, user-agent, timeout, and other options

## Powered by

[<img src="https://www.pullrequest.com/images/pullrequest-logo.svg" width="200">](https://www.pullrequest.com)
