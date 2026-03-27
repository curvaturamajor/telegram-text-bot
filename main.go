package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const OWNER_ID = 7350150331

var targetUsers = map[int64]bool{
	7779993631: true, 5459050513: true, 8177306439: true,
	6454328730: true, 1981317543: true, 8210218070: true,
	6097954079: true, 8126159172: true, 7776852074: true,
}

// JSON Yapıları (Büyük harfle başlamalı!)
type Update struct {
	Message *Message `json:"message"`
}

type Message struct {
	MessageID int      `json:"message_id"`
	From      *User    `json:"from"`
	Chat      Chat     `json:"chat"`
	Text      string   `json:"text"`
	Caption   string   `json:"caption"`
	Entities  []Entity `json:"entities"`
	CEntities []Entity `json:"caption_entities"`
	ReplyTo   *Message `json:"reply_to_message"`
}

type User struct {
	ID int64 `json:"id"`
}

type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type Entity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	URL    string `json:"url"`
}

var httpClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:    100,
		IdleConnTimeout: 90 * time.Second,
	},
}

func apiRequest(token, method string, data interface{}) {
	url := "https://api.telegram.org/bot" + token + "/" + method
	payload, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if resp, err := httpClient.Do(req); err == nil {
		resp.Body.Close()
	}
}

func main() {
	token := os.Getenv("BOT_TOKEN")
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }

	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		var u Update
		if err := json.NewDecoder(r.Body).Decode(&u); err != nil || u.Message == nil || u.Message.From == nil {
			return
		}

		m := u.Message
		uid := m.From.ID

		// 1️⃣ LİNK SİLME
		if targetUsers[uid] {
			entities := m.Entities
			content := m.Text
			if len(entities) == 0 && m.CEntities != nil {
				entities = m.CEntities
				content = m.Caption
			}

			hasLink := false
			runes := []rune(content)
			for _, e := range entities {
				if e.Type == "url" {
					// Index out of range hatasını önlemek için güvenli slice
					if e.Offset+e.Length <= len(runes) {
						if strings.Contains(string(runes[e.Offset:e.Offset+e.Length]), "t.me/") {
							hasLink = true; break
						}
					}
				} else if e.Type == "text_link" && strings.Contains(e.URL, "t.me/") {
					hasLink = true; break
				}
			}

			if hasLink {
				go func(cID int64, mID int) {
					time.Sleep(6 * time.Minute)
					apiRequest(token, "deleteMessage", map[string]interface{}{"chat_id": cID, "message_id": mID})

					// Uyarı gönder
					url := "https://api.telegram.org/bot" + token + "/sendMessage"
					warnData := map[string]interface{}{"chat_id": cID, "text": "Yasaklı görsel kaldırıldı"}
					p, _ := json.Marshal(warnData)
					if resp, err := httpClient.Post(url, "application/json", bytes.NewReader(p)); err == nil {
						var res struct {
							Result struct {
								MessageID int `json:"message_id"`
							} `json:"result"`
						}
						json.NewDecoder(resp.Body).Decode(&res)
						resp.Body.Close()
						if res.Result.MessageID != 0 {
							time.Sleep(30 * time.Second)
							apiRequest(token, "deleteMessage", map[string]interface{}{"chat_id": cID, "message_id": res.Result.MessageID})
						}
					}
				}(m.Chat.ID, m.MessageID)
				return
			}
		}

		// 2️⃣ KOMUTLAR
		if uid == OWNER_ID && len(m.Text) > 2 && m.Text[0] == '/' {
			parts := strings.SplitN(m.Text, " ", 2)
			cmd := parts[0]

			if cmd == "/tt" && len(parts) > 1 {
				rID := 0
				if m.ReplyTo != nil { rID = m.ReplyTo.MessageID }
				go apiRequest(token, "sendMessage", map[string]interface{}{
					"chat_id": m.Chat.ID, "text": parts[1], "reply_to_message_id": rID,
				})
			} else if len(parts) > 1 {
				link := parts[1]
				p := strings.Split(strings.TrimSpace(link), "/")
				if len(p) >= 3 {
					mID, _ := strconv.Atoi(p[len(p)-1])
					cID, _ := strconv.ParseInt("-100"+strings.TrimPrefix(p[len(p)-2], "c"), 10, 64)

					if cmd == "/del" {
						go apiRequest(token, "deleteMessage", map[string]interface{}{"chat_id": cID, "message_id": mID})
					} else {
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
							go func(e string) {
								url := "https://api.telegram.org/bot" + token + "/setMessageReaction"
								payload := `{"chat_id":` + strconv.FormatInt(cID, 10) + `,"message_id":` + strconv.Itoa(mID) + `,"reaction":[{"type":"emoji","emoji":"` + e + `"}]}`
								res, _ := httpClient.Post(url, "application/json", strings.NewReader(payload))
								if res != nil { res.Body.Close() }
							}(emoji)
						}
					}
				}
			}
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	http.ListenAndServe(":"+port, nil)
}
