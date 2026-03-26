package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const OWNER_ID int64 = 7350150331

var targetUsers = map[int64]bool{
	7779993631: true, 5459050513: true, 8177306439: true,
	6454328730: true, 1981317543: true, 8210218070: true,
	6097954079: true, 8126159172: true, 7776852074: true,
}

var httpClient = &http.Client{}

func isTelegramLink(url string) bool {
	return strings.Contains(url, "t.me") || strings.Contains(url, "telegram.me")
}

func main() {
	token := os.Getenv("BOT_TOKEN")
	bot, _ := tgbotapi.NewBotAPI(token)
	bot.Client = httpClient

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	webhookURL := os.Getenv("RENDER_EXTERNAL_URL") + "/webhook"
	bot.Request(tgbotapi.DeleteWebhookConfig{})
	wh, _ := tgbotapi.NewWebhook(webhookURL)
	bot.Request(wh)

	updates := bot.ListenForWebhook("/webhook")
	go http.ListenAndServe(":"+os.Getenv("PORT"), nil)

	for update := range updates {
		if update.Message == nil || update.Message.From == nil {
			continue
		}

		m := update.Message
		uid := m.From.ID

		// 1️⃣ OTOMATİK TELEGRAM LİNKİ SİLME VE UYARI
		if targetUsers[uid] {
			hasTGLink := false
			for _, entity := range m.Entities {
				if entity.Type == "url" {
					link := string([]rune(m.Text)[entity.Offset : entity.Offset+entity.Length])
					if isTelegramLink(link) {
						hasTGLink = true
						break
					}
				}
				if entity.Type == "text_link" && isTelegramLink(entity.URL) {
					hasTGLink = true
					break
				}
			}

			if hasTGLink {
				go func(cID int64, mID int) {
					time.Sleep(7 * time.Minute)
					bot.Request(tgbotapi.DeleteMessageConfig{ChatID: cID, MessageID: mID})
					
					warn := tgbotapi.NewMessage(cID, "Yasaklı görsel kaldırıldı")
					sentWarn, err := bot.Send(warn)
					
					if err == nil {
						time.Sleep(30 * time.Second)
						bot.Request(tgbotapi.DeleteMessageConfig{ChatID: cID, MessageID: sentWarn.MessageID})
					}
				}(m.Chat.ID, m.MessageID)
				continue
			}
		}

		// 2️⃣ SAHİBİ VE KOMUT KONTROLÜ
		if uid != OWNER_ID || len(m.Text) < 3 || m.Text[0] != '/' {
			continue
		}

		parts := strings.SplitN(m.Text, " ", 2)
		if len(parts) < 2 { continue }
		cmd := parts[0]

		if cmd == "/tt" {
			rID := 0
			if m.ReplyToMessage != nil {
				rID = m.ReplyToMessage.MessageID
			}
			go func(cID int64, txt string, reply int) {
				msg := tgbotapi.NewMessage(cID, txt)
				msg.ReplyToMessageID = reply
				bot.Send(msg)
			}(m.Chat.ID, parts[1], rID)
		}
	}
}
