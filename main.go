package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const OWNER_ID int64 = 7350150331

type reactionReq struct {
	ChatID    int64       `json:"chat_id"`
	MessageID int         `json:"message_id"`
	Reaction  interface{} `json:"reaction"`
}

func sendReaction(token string, chatID int64, msgID int, emoji string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/setMessageReaction", token)
	body := reactionReq{
		ChatID:    chatID,
		MessageID: msgID,
		Reaction: []map[string]string{
			{"type": "emoji", "emoji": emoji},
		},
	}
	data, _ := json.Marshal(body)
	http.Post(url, "application/json", bytes.NewReader(data))
}

func parseMessageLink(link string) (int64, int) {
	parts := strings.Split(link, "/")
	msgID, _ := strconv.Atoi(parts[len(parts)-1])
	chatID, _ := strconv.ParseInt("-100"+parts[len(parts)-2], 10, 64)
	return chatID, msgID
}

func main() {
	token := os.Getenv("BOT_TOKEN")
	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	bot, _ := tgbotapi.NewBotAPI(token)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	webhookURL := os.Getenv("RENDER_EXTERNAL_URL") + "/webhook"
	bot.Request(tgbotapi.DeleteWebhookConfig{})
	wh, _ := tgbotapi.NewWebhook(webhookURL)
	bot.Request(wh)

	updates := bot.ListenForWebhook("/webhook")
	go http.ListenAndServe(":"+port, nil)

	for update := range updates {
		if update.Message == nil || update.Message.From == nil || update.Message.From.ID != OWNER_ID {
			continue
		}

		text := update.Message.Text

		// 1️⃣ /txt komutu grup içinde reply destekli
		if strings.HasPrefix(text, "/txt") {
			content := strings.TrimSpace(text[4:])
			out := tgbotapi.NewMessage(update.Message.Chat.ID, content)
			if update.Message.ReplyToMessage != nil {
				out.ReplyToMessageID = update.Message.ReplyToMessage.MessageID
			}
			bot.Send(out)
			continue
		}

		// 2️⃣ DM reaction komutları ve DM /del
		if update.Message.Chat.Type == "private" {
			// /del
			if strings.HasPrefix(text, "/del") {
				chatID, msgID := parseMessageLink(strings.TrimSpace(text[4:]))
				bot.Request(tgbotapi.DeleteMessageConfig{
					ChatID:    chatID,
					MessageID: msgID,
				})
				continue
			}

			// Reaction komutları
			var emoji string
			switch text {
			case "/love":
				emoji = "❤️"
			case "/like":
				emoji = "👍"
			case "/dislike":
				emoji = "👎"
			case "/poop":
				emoji = "💩"
			case "/lol":
				emoji = "😁"
			case "/mid":
				emoji = "🖕"
			case "/ang":
				emoji = "😡"
			}

			if emoji != "" {
				link := strings.TrimSpace(strings.SplitN(text, " ", 2)[1])
				if link != "" {
					chatID, msgID := parseMessageLink(link)
					sendReaction(token, chatID, msgID, emoji)
				}
			}
		}
	}
}
