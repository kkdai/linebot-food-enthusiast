package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiApp struct {
	geminiKey string
	ctx       context.Context
	client    *genai.Client
}

func InitGemini(key string) *GeminiApp {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(key))
	if err != nil {
		log.Fatal(err)
	}
	return &GeminiApp{key, ctx, client}
}

// Initialize the Gemini API
func (app *GeminiApp) GeminiFunctionCall(prompt string) (string, error) {

	currencyExchangeTool := &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{{
			Name:        "exchangeRate",
			Description: "Lookup currency exchange rates by date",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"currencyDate": {
						Type: genai.TypeString,
						Description: "A date that must always be in YYYY-MM-DD format" +
							" or the value 'latest' if a time period is not specified",
					},
					"currencyFrom": {
						Type:        genai.TypeString,
						Description: "Currency to convert from",
					},
					"currencyTo": {
						Type:        genai.TypeString,
						Description: "Currency to convert to",
					},
				},
				Required: []string{"currencyDate", "currencyFrom"},
			},
		}},
	}
	// Use a model that supports function calling, like Gemini 1.0 Pro.
	// See "Supported models" in the "Introduction to function calling" page.
	model := app.client.GenerativeModel("gemini-1.5-flash-latest")

	// Specify the function declaration.
	model.Tools = []*genai.Tool{currencyExchangeTool}
	value := float32(0.8)
	model.Temperature = &value
	cs := model.StartChat()

	// Send the message to the generative model.
	resp, err := cs.SendMessage(app.ctx, genai.Text(prompt))
	if err != nil {
		log.Fatalf("Error sending message: %v\n", err)
	}

	// Check that you got the expected function call back.
	part := resp.Candidates[0].Content.Parts[0]
	funcall, ok := part.(genai.FunctionCall)
	if !ok {
		log.Fatalf("Expected type FunctionCall, got %T", part)
	}

	// Check that the function call name is correct.
	if g, e := funcall.Name, currencyExchangeTool.FunctionDeclarations[0].Name; g != e {
		log.Fatalf("Expected FunctionCall.Name %q, got %q", e, g)
	}
	fmt.Printf("Received function call response:\n%q\n\n", part)

	apiResult := map[string]any{
		"base":  "USD",
		"date":  "2024-04-17",
		"rates": map[string]any{"SEK": 0.091}}

	// Send the hypothetical API result back to the generative model.
	fmt.Printf("Sending API result:\n%q\n\n", apiResult)
	resp, err = cs.SendMessage(app.ctx, genai.FunctionResponse{
		Name:     currencyExchangeTool.FunctionDeclarations[0].Name,
		Response: apiResult,
	})
	if err != nil {
		log.Fatalf("Error sending message: %v\n", err)
	}

	return printResponse(resp), nil
}

func (app *GeminiApp) GeminiImage(imgData []byte, prompt string) (string, error) {
	model := app.client.GenerativeModel("gemini-1.5-flash-latest")
	// Set the temperature to 0.8 for a balance between creativity and coherence.
	value := float32(0.8)
	model.Temperature = &value
	data := []genai.Part{
		genai.ImageData("png", imgData),
		genai.Text(prompt),
	}
	log.Println("Begin processing image...")
	resp, err := model.GenerateContent(app.ctx, data...)
	log.Println("Finished processing image...", resp)
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	return printResponse(resp), nil
}

// Gemini Chat Complete: Iput a prompt and get the response string.
func (app *GeminiApp) GeminiChatComplete(req string) string {
	model := app.client.GenerativeModel("gemini-1.5-flash-latest")
	value := float32(0.8)
	model.Temperature = &value
	cs := model.StartChat()

	send := func(msg string) *genai.GenerateContentResponse {
		fmt.Printf("== Me: %s\n== Model:\n", msg)
		res, err := cs.SendMessage(app.ctx, genai.Text(msg))
		if err != nil {
			log.Fatal(err)
		}
		return res
	}

	res := send(req)
	return printResponse(res)
}

// Print the response
func printResponse(resp *genai.GenerateContentResponse) string {
	var ret string
	for _, cand := range resp.Candidates {
		for _, part := range cand.Content.Parts {
			ret = ret + fmt.Sprintf("%v", part)
			fmt.Println(part)
		}
	}
	return ret
}

// removeFirstAndLastLine takes a string and removes the first and last lines.
func removeFirstAndLastLine(s string) string {
	// Split the string into lines.
	lines := strings.Split(s, "\n")

	// If there are less than 3 lines, return an empty string because removing the first and last would leave nothing.
	if len(lines) < 3 {
		return ""
	}

	// Join the lines back together, skipping the first and last lines.
	return strings.Join(lines[1:len(lines)-1], "\n")
}
