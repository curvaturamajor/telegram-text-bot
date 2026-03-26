package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const OWNER_ID int64 = 7350150331

var httpClient = &http.Client{}

func sendReaction(token string, chatID int64, msgID int, emoji string) {
	url := "https://api.telegram.org/bot" + token + "/setMessageReaction"
	payload := fmt.Sprintf(`{"chat_id":%d,"message_id":%d,"reaction":[{"type":"emoji","emoji":"%s"}]}`, chatID, msgID, emoji)
	
	req, _ := http.NewRequest("POST", url, bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	
	resp, _ := httpClient.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
}

func parseMessageLink(link string) (int64, int) {
	parts := strings.Split(strings.TrimSpace(link), "/")
	if len(parts) < 3 { return 0, 0 }
	msgID, _ := strconv.Atoi(parts[len(parts)-1])
	rawID := strings.TrimPrefix(parts[len(parts)-2], "c")
	chatID, _ := strconv.ParseInt("-100"+rawID, 10, 64)
	return chatID, msgID
}

func main() {
	token := os.Getenv("BOT_TOKEN")
	bot, _ := tgbotapi.NewBotAPI(token)
	bot.Client = httpClient

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	webhookURL := os.Getenv("RENDER_EXTERNAL_URL") + "/webhook"
	bot.Request(tgbotapi.DeleteWebhookConfig{})
	wh, _ := tgbotapi.NewWebhook(webhookURL)
	bot.Request(wh)

	updates := bot.ListenForWebhook("/webhook")
	go http.ListenAndServe(":"+os.Getenv("PORT"), nil)

	for update := range updates {
		if update.Message == nil || update.Message.From.ID != OWNER_ID {
			continue
		}

		text := update.Message.Text
		if len(text) < 3 || text[0] != '/' { continue }

		parts := strings.SplitN(text, " ", 2)
		cmd := parts[0]

		// 1️⃣ /tt - (Eski /txt) Ultra Hızlı Yanıt
		if cmd == "/tt" && len(parts) > 1 {
			go func(cID int64, content string, rID int) {
				m := tgbotapi.NewMessage(cID, content)
				m.ReplyToMessageID = rID
				bot.Send(m)
			}(update.Message.Chat.ID, parts[1], func() int {
				if update.Message.ReplyToMessage != nil {
					return update.Message.ReplyToMessage.MessageID
				}
				return 0
			}())
			continue
		}

		// 2️⃣ DM Operasyonları
		if update.Message.Chat.Type == "private" && len(parts) > 1 {
			link := parts[1]

			if cmd == "/del" {
				go func(l string) {
					cID, mID := parseMessageLink(l)
					bot.Request(tgbotapi.DeleteMessageConfig{ChatID: cID, MessageID: mID})
				}(link)
				continue
			}

			var emoji string
			switch cmd {
			case "/love": emoji = "❤️"
			case "/like": emoji = "👍"
			case "/dislike": emoji = "👎"
			case "/poop": emoji = "💩"
			case "/lol": emoji = "😁"
			case "/mid": emoji = "🖕"
			case "/ang": emoji = "😡"
			}

			if emoji != "" {
				go func(l, e string) {
					cID, mID := parseMessageLink(l)
					sendReaction(token, cID, mID, e)
				}(link, emoji)
			}
		}
	}
}
