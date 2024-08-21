package simplegemini

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/vertexai/genai"
)

// AddImage reads an image from a file, prepares it for processing,
// and adds it to the list of parts to be used by the model.
// It supports verbose logging of operations if enabled.
func (gc *GeminiClient) AddImage(filename string) error {
	imageBytes, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	if gc.Verbose {
		fmt.Printf("Read %d bytes from %s.\n", len(imageBytes), filename)
	}
	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	if ext == "jpg" {
		ext = "jpeg"
	}
	if gc.Verbose {
		fmt.Printf("Using ext type: %s\n", ext)
	}
	img := genai.ImageData(ext, imageBytes)
	if gc.Verbose {
		fmt.Printf("Prepared an image blob: %T\n", img)
	}
	gc.Parts = append(gc.Parts, img)
	return nil
}

// MustAddImage is a convenience function that adds an image to the MultiModal instance,
// terminating the program if adding the image fails.
func (gc *GeminiClient) MustAddImage(filename string) {
	if err := gc.AddImage(filename); err != nil {
		panic(err)
	}
}

// AddURI adds a file part to the MultiModal instance from a Google Cloud URI,
// allowing for integration with cloud resources directly.
// Example URI: "gs://generativeai-downloads/images/scones.jpg"
func (gc *GeminiClient) AddURI(URI string) {
	gc.Parts = append(gc.Parts, genai.FileData{
		MIMEType: mime.TypeByExtension(filepath.Ext(URI)),
		FileURI:  URI,
	})
}

// AddURL downloads the file from the given URL, identifies the MIME type,
// and adds it as a genai.Part.
func (gc *GeminiClient) AddURL(URL string) error {
	resp, err := http.Get(URL)
	if err != nil {
		return fmt.Errorf("failed to download the file from URL: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read the response body: %v", err)
	}
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		return fmt.Errorf("could see a Content-Type header for the given URL: %s", URL)
	}
	if gc.Verbose {
		fmt.Printf("Downloaded %d bytes with MIME type %s from %s.\n", len(data), mimeType, URL)
	}
	fileData := genai.Blob{
		MIMEType: mimeType,
		Data:     data,
	}
	gc.Parts = append(gc.Parts, fileData)
	return nil
}

// AddData adds arbitrary data with a specified MIME type to the parts of the MultiModal instance.
func (gc *GeminiClient) AddData(mimeType string, data []byte) {
	fileData := genai.Blob{
		MIMEType: mimeType,
		Data:     data,
	}
	gc.Parts = append(gc.Parts, fileData)
}

// AddText adds a textual part to the MultiModal instance.
func (gc *GeminiClient) AddText(prompt string) {
	gc.Parts = append(gc.Parts, genai.Text(prompt))
}

func (gc *GeminiClient) ClearParts() {
	gc.Parts = make([]genai.Part, 0)
}
