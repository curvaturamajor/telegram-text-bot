package main

import (
	"bytes"
	"net/http"
	"os"
	"strconv"
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

func sendReaction(token string, chatID int64, msgID int, emoji string) {
	url := "https://api.telegram.org/bot" + token + "/setMessageReaction"
	payload := `{"chat_id":` + strconv.FormatInt(chatID, 10) + `,"message_id":` + strconv.Itoa(msgID) + `,"reaction":[{"type":"emoji","emoji":"` + emoji + `"}]}`
	req, _ := http.NewRequest("POST", url, bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	if resp, err := httpClient.Do(req); err == nil {
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

// Sadece t.me kontrolü (En hızlı hali)
func isTG(url string) bool {
	return strings.Contains(url, "t.me/")
}

func main() {
	token := os.Getenv("BOT_TOKEN")
	bot, _ := tgbotapi.NewBotAPI(token)
	bot.Client = httpClient

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	webhookURL := os.Getenv("RENDER_EXTERNAL_URL") + "/webhook"
	bot.Request(tgbotapi.DeleteWebhookConfig{})
	wh, _ := tgbotapi.NewWebhook(webhookURL)
	bot.Request(wh)

	updates := bot.ListenForWebhook("/webhook")
	go http.ListenAndServe(":"+os.Getenv("PORT"), nil)

	for update := range updates {
		if update.Message == nil || update.Message.From == nil { continue }
		m := update.Message
		uid := m.From.ID

		// 1️⃣ OTOMATİK LİNK SİLME (Hyperlink Desteği Dahil)
		if targetUsers[uid] {
			hasLink := false
			entities := m.Entities
			content := m.Text
			// Eğer mesaj bir görsel/video ise açıklamaya (caption) bak
			if len(entities) == 0 && m.CaptionEntities != nil {
				entities = m.CaptionEntities
				content = m.Caption
			}

			for _, e := range entities {
				if e.Type == "url" {
					// Düz metin içindeki link
					if isTG(string([]rune(content)[e.Offset : e.Offset+e.Length])) {
						hasLink = true; break
					}
				} else if e.Type == "text_link" && isTG(e.URL) {
					// Yazı altına gizlenmiş link
					hasLink = true; break
				}
			}

			if hasLink {
				go func(cID int64, mID int) {
					time.Sleep(6 * time.Minute) // Süre 6 dakikaya düşürüldü
					bot.Request(tgbotapi.DeleteMessageConfig{ChatID: cID, MessageID: mID})
					
					warn := tgbotapi.NewMessage(cID, "Yasaklı görsel kaldırıldı")
					if sw, err := bot.Send(warn); err == nil {
						time.Sleep(30 * time.Second)
						bot.Request(tgbotapi.DeleteMessageConfig{ChatID: cID, MessageID: sw.MessageID})
					}
				}(m.Chat.ID, m.MessageID)
				continue
			}
		}

		// 2️⃣ SAHİBİ VE KOMUT KONTROLÜ
		if uid != OWNER_ID || len(m.Text) < 3 || m.Text[0] != '/' { continue }
		parts := strings.SplitN(m.Text, " ", 2)
		if len(parts) < 2 { continue }
		cmd := parts[0]

		if cmd == "/tt" {
			rID := 0
			if m.ReplyToMessage != nil { rID = m.ReplyToMessage.MessageID }
			go func(cID int64, txt string, reply int) {
				msg := tgbotapi.NewMessage(cID, txt)
				msg.ReplyToMessageID = reply
				bot.Send(msg)
			}(m.Chat.ID, parts[1], rID)
			continue
		}

		// 3️⃣ DM REAKSİYON VE SİLME
		if m.Chat.Type == "private" {
			link := parts[1]
			if cmd == "/del" {
				go func(l string) {
					cID, mID := parseMessageLink(l)
					if cID != 0 { bot.Request(tgbotapi.DeleteMessageConfig{ChatID: cID, MessageID: mID}) }
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
					if cID != 0 { sendReaction(token, cID, mID, e) }
				}(link, emoji)
			}
		}
	}
}
