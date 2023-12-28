package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/line/line-bot-sdk-go/v7/linebot"
)

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	events, err := bot.ParseRequest(r)

	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	for _, event := range events {
		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			// Handle only on text message
			case *linebot.TextMessage:
				req := message.Text

				// Reply with Gemini result
				ret := GeminiChatComplete(req)

				if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(ret)).Do(); err != nil {
					log.Print(err)
				}
			// Handle only on Sticker message
			case *linebot.StickerMessage:
				var kw string
				for _, k := range message.Keywords {
					kw = kw + "," + k
				}

				outStickerResult := fmt.Sprintf("收到貼圖訊息: %s, pkg: %s kw: %s  text: %s", message.StickerID, message.PackageID, kw, message.Text)
				if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(outStickerResult)).Do(); err != nil {
					log.Print(err)
				}

			// Handle only image message
			case *linebot.ImageMessage:
				log.Println("Got img msg ID:", message.ID)

				//Get image binary from LINE server based on message ID.
				content, err := bot.GetMessageContent(message.ID).Do()
				if err != nil {
					log.Println("Got GetMessageContent err:", err)
				}
				defer content.Content.Close()
				data, err := io.ReadAll(content.Content)
				if err != nil {
					log.Fatal(err)
				}
				ret, err := GeminiImage(data, "Describe this image with scientific detail, reply in zh-TW:")
				if err != nil {
					ret = "無法辨識影片內容文字，請重新輸入:" + err.Error()
				}
				if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(ret)).Do(); err != nil {
					log.Print(err)
				}

			// Handle only video message
			case *linebot.VideoMessage:
				// log.Println("Got video msg ID:", message.ID)
				// ret := "影片上傳判斷中，請稍候"

				// // Determine the push msg target.
				// target := event.Source.UserID
				// if event.Source.GroupID != "" {
				// 	target = event.Source.GroupID
				// } else if event.Source.RoomID != "" {
				// 	target = event.Source.RoomID
				// }

				// go uploadAndDectect(target, message, bot)

				// if _, err = bot.ReplyMessage(event.ReplyToken,
				// 	linebot.NewTextMessage(ret)).Do(); err != nil {
				// 	log.Print(err)
				// }
			}
		}
	}
}

func uploadAndDectect(target string, msg *linebot.VideoMessage, bot *linebot.Client) {
	//Get video content from LINE server based on message ID.
	content, err := bot.GetMessageContent(msg.ID).Do()
	if err != nil {
		log.Println("Got GetMessageContent err:", err)
	}
	defer content.Content.Close()

	client, err := storage.NewClient(context.Background())
	var ret string
	if err != nil {
		ret = "storage.NewClient: " + err.Error()
	} else {
		ret = "storage.NewClient: OK"
	}

	if content.ContentLength > 0 {
		uploader := &ClientUploader{
			cl:         client,
			bucketName: bucketName,
			projectID:  projectID,
			uploadPath: "test-files/",
		}

		// Upload Audio to Google Cloud Storage
		err = uploader.UploadVideo(content.Content)
		if err != nil {
			ret = "uploader.UploadFile: " + err.Error()
		} else {
			ret = "uploader.UploadFile: OK, " + uploader.GetPulicAddress()
		}

		vdourl := uploader.GetPulicAddress()

		// Detect string from video
		if err, ret = uploader.SpeachToText(); err != nil {
			log.Print(err)
		}

		if len(ret) == 0 {
			ret = "無法辨識影片內容文字，請重新輸入。"
		}
		flx := newVideoFlexMsg(vdourl, ret)

		if _, err = bot.PushMessage(target, linebot.NewFlexMessage("flex", flx)).Do(); err != nil {
			log.Print(err)
		}
	}
}

func newVideoFlexMsg(video, text string) linebot.FlexContainer {
	flex4 := 4
	flex1 := 1
	return &linebot.BubbleContainer{
		Type: linebot.FlexContainerTypeBubble,
		Hero: &linebot.VideoComponent{
			Type:       linebot.FlexComponentTypeVideo,
			URL:        video,
			PreviewURL: "https://scdn.line-apps.com/n/channel_devcenter/img/fx/01_1_cafe.png",
			AltContent: &linebot.ImageComponent{
				Type:        linebot.FlexComponentTypeImage,
				URL:         "https://scdn.line-apps.com/n/channel_devcenter/img/fx/01_1_cafe.png",
				Size:        linebot.FlexImageSizeTypeFull,
				AspectRatio: linebot.FlexImageAspectRatioType20to13,
				AspectMode:  linebot.FlexImageAspectModeTypeCover,
			},
			Action: &linebot.URIAction{
				Label: "More information",
				URI:   "https://github.com/kkdai/linebot-video-gcp",
			},
			AspectRatio: linebot.FlexVideoAspectRatioType20to13,
		},
		Body: &linebot.BoxComponent{
			Type:    linebot.FlexComponentTypeBox,
			Layout:  linebot.FlexBoxLayoutTypeVertical,
			Spacing: linebot.FlexComponentSpacingTypeMd,
			Contents: []linebot.FlexComponent{
				&linebot.TextComponent{
					Type:    linebot.FlexComponentTypeText,
					Wrap:    true,
					Weight:  linebot.FlexTextWeightTypeBold,
					Gravity: linebot.FlexComponentGravityTypeCenter,
					Text:    "翻譯後的文字如下",
				},
				&linebot.BoxComponent{
					Type:    linebot.FlexComponentTypeBox,
					Layout:  linebot.FlexBoxLayoutTypeBaseline,
					Spacing: linebot.FlexComponentSpacingTypeSm,
					Contents: []linebot.FlexComponent{
						&linebot.TextComponent{
							Type:  linebot.FlexComponentTypeText,
							Wrap:  true,
							Size:  linebot.FlexTextSizeTypeSm,
							Color: "#AAAAAA",
							Text:  "內容",
							Flex:  &flex1,
						},
						&linebot.TextComponent{
							Type:  linebot.FlexComponentTypeText,
							Wrap:  true,
							Size:  linebot.FlexTextSizeTypeSm,
							Color: "#666666",
							Text:  text,
							Flex:  &flex4,
						}},
				},
			},
		},
	}
}
