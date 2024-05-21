// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

var bucketName string
var geminiKey string
var channelToken string

var bot *messaging_api.MessagingApiAPI
var blob *messaging_api.MessagingApiBlobAPI
var gemini *GeminiApp

func main() {
	var err error
	geminiKey = os.Getenv("GOOGLE_GEMINI_API_KEY")
	channelToken = os.Getenv("ChannelAccessToken")
	gaeKey := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	firebaseURL := os.Getenv("FIREBASE_URL")

	//Init firebase
	initFirebase(gaeKey, firebaseURL, context.Background())

	// initialize the messaging API
	bot, err = messaging_api.NewMessagingApiAPI(channelToken)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize the Gemini API
	gemini = InitGemini(geminiKey)
	defer gemini.client.Close()

	blob, err = messaging_api.NewMessagingApiBlobAPI(channelToken)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/callback", callbackHandler)
	port := os.Getenv("PORT")
	addr := fmt.Sprintf(":%s", port)
	http.ListenAndServe(addr, nil)
}
