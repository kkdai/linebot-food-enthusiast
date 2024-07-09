package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/line/line-bot-sdk-go/v7/linebot"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

// Const variables of Prompts.
const ImagePrompt = "你是一個美食烹飪專家，根據這張圖片給予相關的食物敘述，越詳細越好。"
const CalcPrompt = "根據這張圖片，試著估算圖片食物的卡路里。 根據以下格式給我 food(name, calories), 只要給我 JSON 就好。"
const CookPrompt = "根據這張圖片，幫我找到相關的食譜。盡可能詳細列出烹煮步驟跟所需要材料，謝謝。"

// Image statics link.
const CalcImg = "https://raw.githubusercontent.com/kkdai/linebot-food-enthusiast/main/img/calc.jpg"
const CookImg = "https://raw.githubusercontent.com/kkdai/linebot-food-enthusiast/main/img/cooking.png"

// pushMsg: Push message to LINE server.
func pushMsg(target, text string) error {
	if _, err := bot.PushMessage(
		&messaging_api.PushMessageRequest{
			To: target,
			Messages: []messaging_api.MessageInterface{
				&messaging_api.TextMessage{
					Text: text,
				},
			},
		},
		"",
	); err != nil {
		return err
	}
	return nil
}

// replyText: Reply text message to LINE server.
func replyText(replyToken, text string) error {
	if _, err := bot.ReplyMessage(
		&messaging_api.ReplyMessageRequest{
			ReplyToken: replyToken,
			Messages: []messaging_api.MessageInterface{
				&messaging_api.TextMessage{
					Text: text,
				},
			},
		},
	); err != nil {
		return err
	}
	return nil
}

// handleCameraQuickReply: Handle camera quick reply.
func handleCameraQuickReply(replyToken string) error {
	msg := &messaging_api.TextMessage{
		Text: "請上傳一張美食照片，開始相關功能吧！",
		QuickReply: &messaging_api.QuickReply{
			Items: []messaging_api.QuickReplyItem{
				{
					ImageUrl: "",
					Action: &messaging_api.CameraAction{
						Label: "Camera",
					},
				},
			},
		},
	}
	if _, err := bot.ReplyMessage(
		&messaging_api.ReplyMessageRequest{
			ReplyToken: replyToken,
			Messages:   []messaging_api.MessageInterface{msg},
		},
	); err != nil {
		return err
	}
	return nil
}

// callbackHandler: Handle callback from LINE server.
func callbackHandler(w http.ResponseWriter, r *http.Request) {
	cb, err := webhook.ParseRequest(os.Getenv("ChannelSecret"), r)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	for _, event := range cb.Events {
		log.Printf("Got event %v", event)
		switch e := event.(type) {
		case webhook.MessageEvent:
			// 取得用戶 ID
			var uID string
			switch source := e.Source.(type) {
			case webhook.UserSource:
				uID = source.UserId
			case webhook.GroupSource:
				uID = source.UserId
			case webhook.RoomSource:
				uID = source.UserId
			}
			log.Println("User ID:", uID)
			fireDB.SetPath(fmt.Sprintf("%s/%s", DBFoodPath, uID))

			switch message := e.Message.(type) {
			// Handle only on text message
			case webhook.TextMessageContent:
				// Get all food from firebase
				var foods map[string]Food
				if err := fireDB.GetFromDB(&foods); err != nil {
					log.Print(err)
				}
				// Marshall to json
				jsonData, err := json.Marshal(foods)
				if err != nil {
					log.Print(err)
				}

				// Prepare QuickReply buttons.

				promt := fmt.Sprintf("目前您的卡路里資料如下: %s  \n\n 幫我回答我的問題: %s\n", jsonData, message.Text)
				answer := gemini.GeminiChatComplete(promt)
				if err := replyText(e.ReplyToken, answer); err != nil {
					log.Print(err)
				}

			// Handle only on Sticker message
			case webhook.StickerMessageContent:
				var kw string
				for _, k := range message.Keywords {
					kw = kw + "," + k
				}

				outStickerResult := fmt.Sprintf("收到貼圖訊息: %s, pkg: %s kw: %s  text: %s", message.StickerId, message.PackageId, kw, message.Text)
				if err := replyText(e.ReplyToken, outStickerResult); err != nil {
					log.Print(err)
				}

			// Handle only image message
			case webhook.ImageMessageContent:
				log.Println("Got img msg ID:", message.Id)

				//Get image binary from LINE server based on message ID.
				data, err := GetImageBinary(blob, message.Id)
				if err != nil {
					log.Println("Got GetMessageContent err:", err)
					continue
				}

				ret, err := gemini.GeminiImage(data, ImagePrompt)
				if err != nil {
					ret = "無法辨識影片內容文字，請重新輸入:" + err.Error()
				}

				// Prepare QuickReply buttons.
				qReply := &messaging_api.QuickReply{
					Items: []messaging_api.QuickReplyItem{
						{
							ImageUrl: CalcImg,
							Action: &messaging_api.PostbackAction{
								Label:       "calc",
								Data:        "action=calc&m_id=" + message.Id,
								DisplayText: "計算卡路里",
								Text:        "",
							},
						}, {
							ImageUrl: CookImg,
							Action: &messaging_api.PostbackAction{
								Label:       "cook",
								Data:        "action=cook&m_id=" + message.Id,
								DisplayText: "建議食譜",
								Text:        "",
							},
						},
					},
				}

				// Determine the push msg target.
				if _, err := bot.ReplyMessage(
					&messaging_api.ReplyMessageRequest{
						ReplyToken: e.ReplyToken,
						Messages: []messaging_api.MessageInterface{
							&messaging_api.TextMessage{
								Text:       ret,
								QuickReply: qReply,
							},
						},
					},
				); err != nil {
					log.Print(err)
				}

			// Handle only video message
			case webhook.VideoMessageContent:
				log.Println("Got video msg ID:", message.Id)

			default:
				log.Printf("Unknown message: %v", message)
			}
		case webhook.PostbackEvent:
			// Using urls value to parse event.Postback.Data strings.
			ret, err := url.ParseQuery(e.Postback.Data)
			if err != nil {
				log.Print("action parse err:", err, " dat=", e.Postback.Data)
				continue
			}

			log.Println("Action:", ret["action"])
			log.Println("Calc calories m_id:", ret["m_id"])

			// 取得用戶 ID
			var target string
			switch source := e.Source.(type) {
			case webhook.UserSource:
				target = source.UserId
			case webhook.GroupSource:
				target = source.UserId
			case webhook.RoomSource:
				target = source.UserId
			}
			log.Println("Target ID:", target)

			// Handle only on Postback message
			if ret["action"][0] == "calc" {
				// Determine the push msg target.
				processImage(e.ReplyToken, ret["m_id"][0], CalcPrompt, ret["action"][0], blob) // for calcCalories
			} else if ret["action"][0] == "cook" {
				// Determine the push msg target.
				processImage(e.ReplyToken, ret["m_id"][0], CookImg, ret["action"][0], blob) // for searchCooking
			}
		case webhook.FollowEvent:
			log.Printf("message: Got followed event")
		case webhook.BeaconEvent:
			log.Printf("Got beacon: " + e.Beacon.Hwid)
		}
	}
}

// ProcessImage: Process an image and reply with a text.
func processImage(target, m_id, prompt, proType string, blob *messaging_api.MessagingApiBlobAPI) {
	// Get image data
	data, err := GetImageBinary(blob, m_id)
	if err != nil {
		log.Printf("Got GetMessageContent err: %v", err)
		return
	}

	// Chat with Image
	responseMsg, err := gemini.GeminiImage(data, prompt)
	if err != nil {
		log.Printf("Got %s err: %v", proType, err)
		return
	}

	if proType == "calc" {
		jsonData := removeFirstAndLastLine(responseMsg)
		log.Println("Got JSON:", jsonData)
		// unmarshal json
		var food Food
		if err := json.Unmarshal([]byte(jsonData), &food); err != nil {
			log.Print(err)
		}

		// Add time
		food.Time = GetLocalTimeString()

		// Insert data to firebase
		if err := fireDB.InsertDB(food); err != nil {
			log.Print(err)
		}
		prompt := fmt.Sprintf("總共吃了以下食物 %s, 請幫我總結並且計算總卡路里數。 ", jsonData)
		responseMsg = gemini.GeminiChatComplete(prompt)
	}

	// Determine the push msg target.
	if _, err := bot.ReplyMessage(
		&messaging_api.ReplyMessageRequest{
			ReplyToken: target,
			Messages: []messaging_api.MessageInterface{
				&messaging_api.TextMessage{
					Text: responseMsg,
				},
			},
		},
	); err != nil {
		log.Print(err)
	}
}

// GetImageBinary: Get image binary from LINE server based on message ID.
func GetImageBinary(blob *messaging_api.MessagingApiBlobAPI, messageID string) ([]byte, error) {
	// Get image binary from LINE server based on message ID.
	content, err := blob.GetMessageContent(messageID)
	if err != nil {
		log.Println("Got GetMessageContent err:", err)
	}
	defer content.Body.Close()
	data, err := io.ReadAll(content.Body)
	if err != nil {
		log.Fatal(err)
	}

	return data, nil
}
