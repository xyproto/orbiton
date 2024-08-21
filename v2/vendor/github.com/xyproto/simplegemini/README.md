[![Build](https://github.com/xyproto/simplegemini/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/xyproto/simplegemini/actions/workflows/build.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/simplegemini)](https://goreportcard.com/report/github.com/xyproto/simplegemini) [![License](https://img.shields.io/badge/license-Apache2-green.svg?style=flat)](https://raw.githubusercontent.com/xyproto/simplegemini/main/LICENSE)

# Simple Gemini

A simple and fun way to use Gemini.

## Features and limitations

* It is possible to submit a prompt and receive a response.
* The package can run both locally (calling the Gemini API) and in Google Cloud (for example as a Google Cloud Run instance).
* Supports multi-modal prompts (prompts where you can add text, images or data to the prompt).
* Supports tool / function calling where you can supply custom Go functions to the Gemini client, and Gemini can call the functions as needed (but only for 1 tool/function, for now).
* This package is a work in progress!
* The only currently known issue is that when adding more than 1 tool/function, it appears to not work, ref. the `multicall` branch.

## Example use

1. Run `gcloud auth application-default login`, if needed.
2. Get the Google Project ID at https://console.cloud.google.com/.
3. `export GCP_PROJECT=123`, where "123" is your own Google Project ID.
4. (optionally) `export GCP_LOCATION=us-west1`, if "us-west1" is the location you prefer.
5. Create a directory for this experiment, for instance: `mkdir -p ~/geminitest && cd ~/geminitest`.
6. Create a `main.go` file that looks like this (0.4 is the temperature, 0.0 is less creative, 1.0 is more creative):


```go
package main

import (
    "fmt"

    "github.com/xyproto/simplegemini"
)

func main() {
    fmt.Println(simplegemini.MustAsk("Write a haiku about cows.", 0.4))
}
```

7. Prepare a simple `go.mod` project file with ie. `go mod init cows`
8. Fetch the dependencies (this simplegemini package) with `go mod tidy`
9. Build and run the executable: `go build && ./cows`
10. Observe the output, that should look a bit like this:

```go
Black and white patches,
Chewing grass in sunlit fields,
Mooing gentle song.
```

## A note about Google Cloud

If an application that uses `simplegemini` is deployed to ie. Google Cloud Run, then creating a new service account with "Vertex AI User" permissions is probably needed. This can be created in the "IAM & Admin" section. The service account can then be selected when deploying to Cloud Run.

## Function calling / tool use

```go
package main

import (
    "fmt"
    "log"
    "strings"

    "github.com/xyproto/simplegemini"
)

func main() {
    gc := simplegemini.MustNew()

    // Define a custom function for getting the weather, that Gemini can choose to call
    getWeatherRightNow := func(location string) string {
        fmt.Println("getWeatherRightNow was called")
        switch location {
        case "NY":
            return "It's sunny in New York."
        case "London":
            return "It's rainy in London."
        default:
            return "Weather data not available."
        }
    }

    // Add the weather function as a tool
    err := gc.AddFunctionTool("get_weather_right_now", "Get the current weather for a specific location", getWeatherRightNow)
    if err != nil {
        log.Fatalf("Failed to add function tool: %v", err)
    }

    // Query Gemini with a prompt that requires using the custom weather tool
    result, err := gc.Query("What is the weather in NY?")
    if err != nil {
        log.Fatalf("Failed to query Gemini: %v", err)
    }

    // Check and print the weather response
    if !strings.Contains(result, "sunny") {
        log.Fatalf("Expected 'sunny' to be in the response, but got: %v", result)
    }
    fmt.Println("Weather AI Response:", result)

    gc.Clear() // Clear the current prompt parts, tools and functions

    // Define a custom function for reversing a string
    reverseString := func(input string) string {
        fmt.Println("reverseString was called")
        runes := []rune(input)
        for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
            runes[i], runes[j] = runes[j], runes[i]
        }
        return string(runes)
    }

    // Add the string reversal function as a tool
    err = gc.AddFunctionTool("reverse_string", "Reverse the given string", reverseString)
    if err != nil {
        log.Fatalf("Failed to add function tool: %v", err)
    }

    // Query Gemini with a prompt that requires using the string reversal tool
    result, err = gc.Query("Reverse the string 'hello'. Reply with a single word.")
    if err != nil {
        log.Fatalf("Failed to query Gemini: %v", err)
    }

    // Check and print the string reversal response
    expected := "olleh"
    if !strings.Contains(result, expected) {
        log.Fatalf("Expected '%s' to be in the response, but got: %v", expected, result)
    }
    fmt.Println("Response:", result)
}
```


## Multimodal prompts / analyzing images

```go
package main

import (
    "fmt"
    "log"

    "github.com/xyproto/simplegemini"
    "github.com/xyproto/wordwrap"
)

func main() {
    const (
        multiModalModelName = "gemini-1.0-pro-vision" // "gemini-1.5-pro" also works, if only text is sent
        temperature         = 0.4
        descriptionPrompt   = "Describe what is common for these two images."
    )

    gc, err := simplegemini.NewMultiModal(multiModalModelName, temperature)
    if err != nil {
        log.Fatalf("Could not initialize the Gemini client with the %s model: %v\n", multiModalModelName, err)
    }

    // Build a prompt
    if err := gc.AddImage("frog.png"); err != nil {
        log.Fatalf("Could not add frog.png: %v\n", err)
    }
    gc.AddURI("gs://generativeai-downloads/images/scones.jpg")
    gc.AddText(descriptionPrompt)

    // Count the tokens that are about to be sent
    tokenCount, err := gc.CountTokens()
    if err != nil {
        log.Fatalln(err)
    }
    fmt.Printf("Sending %d tokens.\n\n", tokenCount)

    // Submit the images and the text prompt
    response, err := gc.Submit()
    if err != nil {
        log.Fatalln(err)
    }

    // Format and print out the response
    if lines, err := wordwrap.WordWrap(response, 79); err == nil { // success
        for _, line := range lines {
            fmt.Println(line)
        }
        return
    }

    fmt.Println(response)
}
```

## Producing JSON

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/xyproto/simplegemini"
)

func main() {
    const (
        prompt      = `What color is the sky? Answer with a JSON struct where the only key is "color" and the value is a lowercase string.`
        modelName   = "gemini-1.5-pro"
        temperature = 0.0
        timeout     = 10 * time.Second
    )

    gc, err := simplegemini.NewWithTimeout(modelName, temperature, timeout)
    if err != nil {
        log.Fatalln(err)
    }

    fmt.Println(prompt)

    result, err := gc.Query(prompt)
    if err != nil {
        log.Fatalln(err)
    }

    fmt.Println(result)
}
```

* `gemini-1.5-flash` is the default model.
* `gemini-1.5-pro` is smarter, but slower and more expensive.

## Environment variables

These environment variables are supported:

* `GCP_PROJECT` or `PROJECT_ID` for the Google Cloud Project ID
* `GCP_LOCATION` or `PROJECT_LOCATION` for the Google Cloud Project location (like `us-west1`)
* `MODEL_NAME` for the Gemini model name (like `gemini-1.5-flash` or `gemini-1.5-pro`)
* `MULTI_MODAL_MODEL_NAME` for the Gemini multi-modal name (like `gemini-1.0-pro-vision`)

## General info

* Version: 1.4.1
* License: Apache 2
* Author: Alexander F. Rødseth
