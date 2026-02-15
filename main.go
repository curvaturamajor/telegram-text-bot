package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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

	log.Printf("Bot başlatıldı: %s", bot.Self.UserName)

	// ✅ Health endpoint (UptimeRobot için)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Webhook URL oluştur
	webhookURL := os.Getenv("RENDER_EXTERNAL_URL") + "/webhook"

	wh, err := tgbotapi.NewWebhook(webhookURL)
	if err != nil {
		log.Fatal(err)
	}

	_, err = bot.Request(wh)
	if err != nil {
		log.Fatal(err)
	}

	// Webhook endpoint
	updates := bot.ListenForWebhook("/webhook")

	// HTTP server başlat
	go func() {
		log.Printf("HTTP server %s portunda başlatıldı", port)
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}()

	// Update loop
	for update := range updates {

		if update.Message == nil {
			continue
		}

		text := update.Message.Text
		if !strings.HasPrefix(text, "/txt") {
			continue
		}

		config := tgbotapi.GetChatMemberConfig{
			ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
				ChatID: update.Message.Chat.ID,
				UserID: update.Message.From.ID,
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
