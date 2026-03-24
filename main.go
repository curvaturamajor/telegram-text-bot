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

	msgIDStr := parts[len(parts)-1]
	chatIDStr := parts[len(parts)-2]

	msgID, err := strconv.Atoi(msgIDStr)
	if err != nil {
		return 0, 0, err
	}

	chatID, err := strconv.ParseInt("-100"+chatIDStr, 10, 64)
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

	bot.Debug = true

	log.Printf("Bot başlatıldı: %s", bot.Self.UserName)

	// Health endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	webhookURL := os.Getenv("RENDER_EXTERNAL_URL") + "/webhook"

	// Webhook reset (Render için daha stabil)
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

	// HTTP server
	go func() {
		log.Printf("HTTP server %s portunda başlatıldı", port)
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}()

	for update := range updates {

		if update.Message == nil {
			continue
		}

		userID := update.Message.From.ID
		chatType := update.Message.Chat.Type
		text := update.Message.Text

		// 🔐 sadece sen kullan
		if userID != OWNER_ID {
			continue
		}

		// 🗑️ /del → sadece private chat
		if chatType == "private" && strings.HasPrefix(text, "/del") {

			link := strings.TrimSpace(strings.TrimPrefix(text, "/del"))
			if link == "" {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ Link boş")
				bot.Send(msg)
				continue
			}

			chatID, msgID, err := parseMessageLink(link)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ Geçersiz link")
				bot.Send(msg)
				continue
			}

			del := tgbotapi.DeleteMessageConfig{
				ChatID:    chatID,
				MessageID: msgID,
			}

			_, err = bot.Request(del)
			if err != nil {
				log.Println("Mesaj silinemedi:", err)

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ Mesaj silinemedi")
				bot.Send(msg)
				continue
			}

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "✅ Mesaj silindi")
			bot.Send(msg)

			continue
		}

		// 📝 /txt → sadece grup
		if chatType != "group" && chatType != "supergroup" {
			continue
		}

		if !strings.HasPrefix(text, "/txt") {
			continue
		}

		config := tgbotapi.GetChatMemberConfig{
			ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
				ChatID: update.Message.Chat.ID,
				UserID: userID,
			},
		}

		member, err := bot.GetChatMember(config)
		if err != nil {
			continue
		}

		if member.Status != "administrator" && member.Status != "creator" {
			continue
		}

		content := strings.TrimSpace(text[4:])
		if content == "" {
			continue
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, content)

		if update.Message.ReplyToMessage != nil {
			msg.ReplyToMessageID = update.Message.ReplyToMessage.MessageID
		}

		_, err = bot.Send(msg)
		if err != nil {
			log.Println("Mesaj gönderilemedi:", err)
		}
	}
}
