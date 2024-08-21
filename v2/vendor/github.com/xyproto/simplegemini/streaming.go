package simplegemini

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/vertexai/genai"
	"google.golang.org/api/iterator"
)

// SubmitToClientStreaming sends the current parts to Gemini, and streams the response back by calling the streamCallback function.
func (gc *GeminiClient) SubmitToClientStreaming(ctx context.Context, streamCallback func(string)) (result string, err error) {
	if streamCallback == nil {
		return "", errors.New("the given streamCallback function cannot be null")
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic occurred: %v", r)
		}
	}()

	// Configure the model
	model := gc.Client.GenerativeModel(gc.ModelName)
	model.SetTemperature(gc.Temperature)

	// Start streaming the response
	iter := model.GenerateContentStream(ctx, gc.Parts...)

	for {
		resp, err := iter.Next()
		if err == iterator.Done {
			// Ensure all remaining parts are processed
			break
		}
		if err != nil {
			return "", fmt.Errorf("streaming error: %v", err)
		}
		if len(resp.Candidates) == 0 {
			return "", errors.New("empty response when streaming")
		}

		// Process each candidate's parts
		for _, candidate := range resp.Candidates {
			for _, part := range candidate.Content.Parts {
				switch p := part.(type) {
				case genai.Text:
					partialResult := string(p)
					if gc.Trim {
						partialResult = strings.TrimSpace(partialResult)
					}
					streamCallback(partialResult)
					result += partialResult
				default:
					// Handle or skip other types like Blob, FileData, etc.
				}
			}
		}
	}

	// Final call to ensure all results are processed and returned
	if gc.Trim {
		result = strings.TrimSpace(result)
	}
	return result, nil
}
