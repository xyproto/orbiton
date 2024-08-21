package simplegemini

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"cloud.google.com/go/vertexai/genai"
)

var ErrEmptyPrompt = errors.New("empty prompt")

// AddFunctionTool allows users to add a custom Go function as a tool for the model
func (gc *GeminiClient) AddFunctionTool(name, description string, fn interface{}) error {
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	// Validate that the function is of the correct type
	if fnType.Kind() != reflect.Func {
		return fmt.Errorf("provided argument is not a function")
	}

	// Create a function declaration based on the function's signature
	parameters := make(map[string]*genai.Schema)
	var required []string

	for i := 0; i < fnType.NumIn(); i++ {
		paramType := fnType.In(i)
		paramName := fmt.Sprintf("param%d", i+1) // param names are generated as "param1", "param2", etc.

		parameters[paramName] = &genai.Schema{
			Type: mapGoTypeToGenaiType(paramType),
		}
		required = append(required, paramName)
	}

	// Register the function in the internal map with the name as the key
	gc.Functions[name] = fnValue

	// Create the function declaration and tool
	functionDecl := &genai.FunctionDeclaration{
		Name:        name,
		Description: description,
		Parameters: &genai.Schema{
			Type:       genai.TypeObject,
			Properties: parameters,
			Required:   required,
		},
	}

	tool := &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{functionDecl},
	}
	gc.Tools = append(gc.Tools, tool)

	return nil
}

// MultiQuery processes a prompt with optional base64-encoded data and MIME type for the data,
// and supports function tools (ftools) by parsing the response and calling the user-supplied functions
func (gc *GeminiClient) MultiQuery(prompt string, base64Data, dataMimeType *string, temperature *float32) (string, error) {
	if strings.TrimSpace(prompt) == "" {
		return "", ErrEmptyPrompt
	}

	gc.ClearParts()
	gc.AddText(prompt)

	// If base64Data and dataMimeType are provided, decode the data and add it to the multimodal instance
	if base64Data != nil && dataMimeType != nil {
		data, err := base64.StdEncoding.DecodeString(*base64Data)
		if err != nil {
			return "", fmt.Errorf("failed to decode base64 data: %v", err)
		}
		gc.AddData(*dataMimeType, data)
	}

	ctx, cancel := context.WithTimeout(context.Background(), gc.Timeout)
	defer cancel()

	// Set up the model with tools and start a chat session
	model := gc.Client.GenerativeModel(gc.ModelName)
	if temperature != nil {
		model.SetTemperature(*temperature)
	}
	model.Tools = gc.Tools
	session := model.StartChat()

	// Submit the multimodal query and process the result
	res, err := session.SendMessage(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to send message: %v", err)
	}

	// Handle function calls if present
	for _, part := range res.Candidates[0].Content.Parts {
		if funcall, ok := part.(genai.FunctionCall); ok {
			// Invoke the user-defined function using reflection
			responseData, err := gc.invokeFunction(funcall.Name, funcall.Args)
			if err != nil {
				return "", fmt.Errorf("failed to handle function call: %v", err)
			}
			// Send the function response back to the model
			res, err = session.SendMessage(ctx, genai.FunctionResponse{
				Name:     funcall.Name,
				Response: responseData,
			})
			if err != nil {
				return "", fmt.Errorf("failed to send function response: %v", err)
			}
			var (
				finalResult string
				stringPart  string
			)
			// Process the final response from the LLM
			for _, part := range res.Candidates[0].Content.Parts {
				stringPart = fmt.Sprintf("%v", part)
				if stringPart != "" {
					finalResult += fmt.Sprintf("%s\n", stringPart)
				}
			}
			return strings.TrimSpace(finalResult), nil
		}
	}

	// Handle the usual case where no function call is made
	result, err := gc.SubmitToClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to process response: %v", err)
	}

	return strings.TrimSpace(result), nil
}

// invokeFunction uses reflection to call the appropriate user-defined function based on the AI's request
func (gc *GeminiClient) invokeFunction(name string, args map[string]any) (map[string]any, error) {
	fn, exists := gc.Functions[name]
	if !exists {
		return nil, fmt.Errorf("function %s not found", name)
	}

	fnType := fn.Type()

	// Prepare the input arguments
	in := make([]reflect.Value, fnType.NumIn())
	for i := 0; i < fnType.NumIn(); i++ {
		paramName := fmt.Sprintf("param%d", i+1)
		argValue, exists := args[paramName]
		if !exists {
			return nil, fmt.Errorf("missing argument: %s", paramName)
		}
		in[i] = reflect.ValueOf(argValue)
	}

	// Call the function
	out := fn.Call(in)

	// Prepare the return values as a map
	result := make(map[string]any)
	for i := 0; i < len(out); i++ {
		result[fmt.Sprintf("return%d", i+1)] = out[i].Interface()
	}

	return result, nil
}

// mapGoTypeToGenaiType maps Go types to the corresponding genai.Schema Type values
func mapGoTypeToGenaiType(goType reflect.Type) genai.Type {
	switch goType.Kind() {
	case reflect.String:
		return genai.TypeString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return genai.TypeInteger
	case reflect.Float32, reflect.Float64:
		return genai.TypeNumber
	case reflect.Bool:
		return genai.TypeBoolean
	default: // default to a string type
		return genai.TypeString
	}
}

func (gc *GeminiClient) ClearToolsAndFunctions() {
	gc.Functions = make(map[string]reflect.Value)
	gc.Tools = []*genai.Tool{}
}
