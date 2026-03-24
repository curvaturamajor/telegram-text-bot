package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const OWNER_ID int64 = 7350150331

// 🔗 Link parse
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
	defer func() {
		if r := recover(); r != nil {
			log.Println("panic recovered:", r)
		}
	}()

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN yok")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	// ⚡ Optimized HTTP client
	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   5 * time.Second,
	}

	// 🔹 Render v5.5.1 uyumlu çağrı
	bot, err := tgbotapi.NewBotAPIWithClient(token, tgbotapi.APIEndpoint, client)
	if err != nil {
		log.Fatal(err)
	}

	bot.Debug = false
	log.SetFlags(0)

	log.Println("Bot aktif:", bot.Self.UserName)

	// ✅ Health endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	webhookURL := os.Getenv("RENDER_EXTERNAL_URL") + "/webhook"

	// 🔄 Webhook reset
	_, _ = bot.Request(tgbotapi.DeleteWebhookConfig{})

	wh, err := tgbotapi.NewWebhook(webhookURL)
	if err != nil {
		log.Fatal(err)
	}

	_, err = bot.Request(wh)
	if err != nil {
		log.Fatal(err)
	}

	updates := bot.ListenForWebhook("/webhook")

	// ⚡ Optimized HTTP server
	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Fatal(server.ListenAndServe())
	}()

	// 🔁 Update loop
	for update := range updates {

		msg := update.Message
		if msg == nil || msg.From == nil {
			continue
		}

		// 🔐 sadece owner
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

			// ⚡ async delete
			go func(chatID int64, msgID int, userChatID int64) {
				_, err := bot.Request(tgbotapi.DeleteMessageConfig{
					ChatID:    chatID,
					MessageID: msgID,
				})

				if err != nil {
					bot.Send(tgbotapi.NewMessage(userChatID, "❌ "+err.Error()))
					return
				}

				bot.Send(tgbotapi.NewMessage(userChatID, "✅ Silindi"))
			}(chatID, msgID, msg.Chat.ID)

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

		_, _ = bot.Send(out)
	}
}
