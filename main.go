package main

import (
	"bytes"
	"encoding/json"
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

type reactionReq struct {
	ChatID    int64       `json:"chat_id"`
	MessageID int         `json:"message_id"`
	Reaction  interface{} `json:"reaction"`
}

func sendReaction(botToken string, chatID int64, msgID int, emoji string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/setMessageReaction", botToken)
	body := reactionReq{
		ChatID:    chatID,
		MessageID: msgID,
		Reaction: []map[string]string{
			{"type": "emoji", "emoji": emoji},
		},
	}
	data, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
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

	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}
	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}

	bot, err := tgbotapi.NewBotAPIWithClient(token, tgbotapi.APIEndpoint, client)
	if err != nil {
		log.Fatal(err)
	}
	bot.Debug = false
	log.Println("Bot aktif:", bot.Self.UserName)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("OK")) })

	webhookURL := os.Getenv("RENDER_EXTERNAL_URL") + "/webhook"
	bot.Request(tgbotapi.DeleteWebhookConfig{})
	wh, _ := tgbotapi.NewWebhook(webhookURL)
	bot.Request(wh)

	updates := bot.ListenForWebhook("/webhook")
	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	go func() { log.Fatal(server.ListenAndServe()) }()

	for update := range updates {
		msg := update.Message
		if msg == nil || msg.From == nil {
			continue
		}
		if msg.From.ID != OWNER_ID {
			continue
		}

		text := msg.Text
		if text == "" {
			continue
		}

		chatType := msg.Chat.Type

		// DM reactions
		if chatType == "private" {
			var emoji, prefix string
			switch {
			case strings.HasPrefix(text, "/love"):
				emoji, prefix = "❤️", "/love"
			case strings.HasPrefix(text, "/like"):
				emoji, prefix = "👍", "/like"
			case strings.HasPrefix(text, "/dislike"):
				emoji, prefix = "👎", "/dislike"
			case strings.HasPrefix(text, "/poop"):
				emoji, prefix = "💩", "/poop"
			case strings.HasPrefix(text, "/lol"):
				emoji, prefix = "😁", "/lol"
			case strings.HasPrefix(text, "/mid"):
				emoji, prefix = "🖕", "/mid"
			case strings.HasPrefix(text, "/ang"):
				emoji, prefix = "😡", "/ang"
			}

			if emoji != "" {
				link := strings.TrimSpace(text[len(prefix):])
				if link == "" {
					bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Link boş"))
					continue
				}
				chatID, msgID, err := parseMessageLink(link)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Geçersiz link"))
					continue
				}
				go func() {
					if err := sendReaction(token, chatID, msgID, emoji); err != nil {
						bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Reaksiyon eklenemedi"))
						return
					}
					bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Reaksiyon eklendi"))
				}()
				continue
			}

			// /del
			if strings.HasPrefix(text, "/del") {
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
				go func() {
					_, err := bot.Request(tgbotapi.DeleteMessageConfig{ChatID: chatID, MessageID: msgID})
					if err != nil {
						bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Silinemedi"))
						return
					}
					bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Silindi"))
				}()
				continue
			}
		}

		// Group /txt
		if chatType == "group" || chatType == "supergroup" {
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
}
