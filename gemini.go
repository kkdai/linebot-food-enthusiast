package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiApp struct {
	geminiKey string
	ctx       context.Context
	client    *genai.Client
}

var calorieTrackingTool *genai.Tool

func InitGemini(key string) *GeminiApp {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(key))
	if err != nil {
		log.Fatal(err)
	}

	calorieTrackingTool = &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{{
			Name:        "recordCalorie",
			Description: "Record a calorie intake with date, amount, and food item",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"foodItem": {
						Type:        genai.TypeString,
						Description: "The name of the food item",
					},
					"date": {
						Type:        genai.TypeString,
						Description: "The date of the intake in YYYY-MM-DD format",
					},
					"calories": {
						Type:        genai.TypeNumber,
						Description: "The amount of calories",
					},
				},
				Required: []string{"foodItem", "date", "calories"},
			},
		}, {
			Name:        "recordFood",
			Description: "Record a eating with date and food item",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"foodItem": {
						Type:        genai.TypeString,
						Description: "The name of the food item",
					},
					"date": {
						Type:        genai.TypeString,
						Description: "The date of the intake in YYYY-MM-DD format",
					},
				},
				Required: []string{"foodItem", "date"},
			},
		}},
	}

	return &GeminiApp{key, ctx, client}
}

func (app *GeminiApp) GeminiImage(imgData []byte, prompt string) (string, error) {
	model := app.client.GenerativeModel("gemini-1.5-flash")
	// Set the temperature to 0.8 for a balance between creativity and coherence.
	value := float32(0.8)
	model.Temperature = &value
	data := []genai.Part{
		genai.ImageData("png", imgData),
		genai.Text(prompt),
	}
	fmt.Println("Begin processing image...")
	resp, err := model.GenerateContent(app.ctx, data...)
	fmt.Println("Finished processing image...", resp)
	if err != nil {
		fmt.Println("err:", err)
		return "", err
	}

	return printResponse(resp), nil
}

// Gemini Chat Complete: Iput a prompt and get the response string.
func (app *GeminiApp) GeminiChatComplete(req string) string {
	model := app.client.GenerativeModel("gemini-1.5-flash")
	value := float32(0.8)
	model.Temperature = &value
	cs := model.StartChat()

	send := func(msg string) *genai.GenerateContentResponse {
		fmt.Printf("== Me: %s\n== Model:\n", msg)
		res, err := cs.SendMessage(app.ctx, genai.Text(msg))
		if err != nil {
			fmt.Println("err:", err)
		}
		return res
	}

	res := send(req)
	return printResponse(res)
}

// Gemini Function Call: Input a prompt and get the response string.
func (app *GeminiApp) GeminiFunctionCall(prompt string) string {
	// Add timestamp for this prompt.
	timelocal, _ := time.LoadLocation("Asia/Taipei")
	time.Local = timelocal
	curNow := time.Now().Local().String()
	prompt = prompt + " 本地時間: " + curNow
	// Use a model that supports function calling, like Gemini 1.0 Pro.
	model := app.client.GenerativeModel("gemini-1.5-flash-latest")

	// Specify the function declaration.
	model.Tools = []*genai.Tool{calorieTrackingTool}
	// Start new chat session.
	session := model.StartChat()
	// Send the message to the generative model.
	resp, err := session.SendMessage(app.ctx, genai.Text(prompt))
	if err != nil {
		fmt.Println("err:", err)
	}

	// Check that you got the expected function call back.
	part := resp.Candidates[0].Content.Parts[0]
	_, ok := part.(genai.FunctionCall)
	if ok {
		fmt.Printf("Received function call response:\n %s \n %s \n", part.(genai.FunctionCall).Name, part.(genai.FunctionCall).Args)

		// According to function call Name and Args we can call the function
		switch part.(genai.FunctionCall).Name {
		case "recordCalorie":
			fmt.Println("Calling recordCalorie function...")
			args := part.(genai.FunctionCall).Args
			foodItem := args["foodItem"]
			date := args["date"]
			calories := args["calories"]

			//convert float64 to int
			caloriesInt, err := strconv.Atoi(fmt.Sprintf("%v", calories))
			if err != nil {
				fmt.Println("err:", err)
				return fmt.Sprintf("err: %v", err)
			}

			fmt.Println("date: ", date, "calories: ", calories, "foodItem: ", foodItem)
			// Call the hypothetical API to record the calorie intake.
			apiResult := recordCalorie(foodItem.(string), date.(string), caloriesInt)
			// Send the hypothetical API result back to the generative model.
			fmt.Printf("Sending API result:\n%q\n\n", apiResult)
			resp, err = session.SendMessage(app.ctx, genai.FunctionResponse{
				Name:     calorieTrackingTool.FunctionDeclarations[0].Name,
				Response: apiResult,
			})
			if err != nil {
				fmt.Println("msg err:", err)
				return fmt.Sprintf("msg err: %v", err)
			}
			// Show the model's response, which is expected to be text.
			return printResponse(resp)
		case "recordFood":
			fmt.Println("Calling recordFood function...")
			args := part.(genai.FunctionCall).Args
			foodItem := args["foodItem"]
			date := args["date"]

			fmt.Println("Asking Gemini to guess the calories...")
			// using default prompt to ask user.
			prompt := fmt.Sprintf("我剛剛吃了 %s, 請幫我猜測卡路里，大概就好，只要回覆我數字。", foodItem)
			caloriesString := app.GeminiChatComplete(prompt)
			// Convert string to integer, e.g. "800 \n"
			caloriesString = removeFirstAndLastLine(caloriesString)
			calories, err := strconv.Atoi(caloriesString)
			fmt.Println("gemini guess calories: ", calories)
			if err != nil {
				fmt.Println("err:", err)
				return fmt.Sprintf("err: %v", err)
			}
			fmt.Println("date: ", date, "calories: ", calories, "foodItem: ", foodItem)
			// Call the hypothetical API to record the calorie intake.
			apiResult := recordCalorie(foodItem.(string), date.(string), calories)
			// Send the hypothetical API result back to the generative model.
			fmt.Printf("Sending API result:\n%q\n\n", apiResult)
			resp, err = session.SendMessage(app.ctx, genai.FunctionResponse{
				Name:     calorieTrackingTool.FunctionDeclarations[0].Name,
				Response: apiResult,
			})
			if err != nil {
				fmt.Println("msg err:", err)
				return fmt.Sprintf("msg err: %v", err)
			}
			// Show the model's response, which is expected to be text.
			return printResponse(resp)
		}
	}
	// Other cases, return the response as text.
	fmt.Printf("Expected type FunctionCall, got %T\n", part)

	// If no function call was made, return the response as text.
	var foods map[string]Food
	if err := fireDB.GetFromDB(&foods); err != nil {
		fmt.Println(err)
	}
	// Marshall to json
	jsonData, err := json.Marshal(foods)
	if err != nil {
		fmt.Println(err)
	}

	// using default prompt to ask user.
	prompt = fmt.Sprintf("目前您的卡路里資料如下: %s  \n\n 幫我回答我的問題: %s\n", jsonData, prompt)
	return app.GeminiChatComplete(prompt)
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
