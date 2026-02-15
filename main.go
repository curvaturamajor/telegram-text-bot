package main

import (
	"log"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN environment variable yok")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	bot.Debug = false

	log.Printf("Bot başlatıldı: %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {

		if update.Message == nil {
			continue
		}

		text := update.Message.Text

		if !strings.HasPrefix(text, "/txt") {
			continue
		}

		// Admin kontrolü
		member, err := bot.GetChatMember(tgbotapi.ChatConfigWithUser{
			ChatID: update.Message.Chat.ID,
			UserID: update.Message.From.ID,
		})
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
