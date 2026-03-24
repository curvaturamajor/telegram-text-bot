package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const OWNER_ID int64 = 7350150331

func parseMessageLink(link string) (int64, int, error) {
	parts := strings.Split(link, "/")
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("geçersiz link")
	}

	msgID, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0, 0, err
	}

	chatID, err := strconv.ParseInt("-100"+parts[len(parts)-2], 10, 64)
	if err != nil {
		return 0, 0, err
	}

	return chatID, msgID, nil
}

func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN yok")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	// Debug kapalı → performans için
	bot.Debug = false

	log.Printf("Bot aktif: %s", bot.Self.UserName)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	webhookURL := os.Getenv("RENDER_EXTERNAL_URL") + "/webhook"

	// Webhook reset
	bot.Request(tgbotapi.DeleteWebhookConfig{})

	wh, err := tgbotapi.NewWebhook(webhookURL)
	if err != nil {
		log.Fatal(err)
	}

	_, err = bot.Request(wh)
	if err != nil {
		log.Fatal(err)
	}

	updates := bot.ListenForWebhook("/webhook")

	go func() {
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}()

	for update := range updates {

		msg := update.Message
		if msg == nil {
			continue
		}

		// 🔒 sadece owner
		if msg.From.ID != OWNER_ID {
			continue
		}

		text := msg.Text
		if text == "" {
			continue
		}

		chatType := msg.Chat.Type

		// 🗑️ DELETE (PRIVATE)
		if chatType == "private" && strings.HasPrefix(text, "/del") {

			link := strings.TrimSpace(text[4:])
			if link == "" {
				bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Link boş"))
				continue
			}

			chatID, msgID, err := parseMessageLink(link)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Geçersiz link"))
				continue
			}

			_, err = bot.Request(tgbotapi.DeleteMessageConfig{
				ChatID:    chatID,
				MessageID: msgID,
			})

			if err != nil {
				bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ "+err.Error()))
				continue
			}

			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Silindi"))
			continue
		}

		// 📝 TXT (GROUP)
		if chatType != "group" && chatType != "supergroup" {
			continue
		}

		if !strings.HasPrefix(text, "/txt") {
			continue
		}

		content := strings.TrimSpace(text[4:])
		if content == "" {
			continue
		}

		out := tgbotapi.NewMessage(msg.Chat.ID, content)

		if msg.ReplyToMessage != nil {
			out.ReplyToMessageID = msg.ReplyToMessage.MessageID
		}

		bot.Send(out)
	}
}
