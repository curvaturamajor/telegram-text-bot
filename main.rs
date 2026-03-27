use axum::{routing::post, Router, Json, response::IntoResponse, http::StatusCode};
use serde::{Deserialize, Serialize};
use std::collections::HashSet;
use std::env;
use std::sync::Arc;
use tokio::time::{sleep, Duration};
use once_cell::sync::Lazy;

const OWNER_ID: i64 = 7350150331;

static TARGET_USERS: Lazy<HashSet<i64>> = Lazy::new(|| {
    let mut s = HashSet::new();
    s.extend([
        7779993631, 5459050513, 8177306439, 6454328730, 1981317543,
        8210218070, 6097954079, 8126159172, 7776852074,
    ]);
    s
});

#[derive(Deserialize, Debug)]
struct Update {
    message: Option<Message>,
}

#[derive(Deserialize, Debug, Clone)]
struct Message {
    message_id: i32,
    from: Option<User>,
    chat: Chat,
    text: Option<String>,
    caption: Option<String>,
    entities: Option<Vec<Entity>>,
    caption_entities: Option<Vec<Entity>>,
    reply_to_message: Option<Box<Message>>,
}

#[derive(Deserialize, Debug, Clone)]
struct User { id: i64 }
#[derive(Deserialize, Debug, Clone)]
struct Chat { id: i64 }
#[derive(Deserialize, Debug, Clone)]
struct Entity {
    #[serde(rename = "type")]
    entity_type: String,
    offset: usize,
    length: usize,
    url: Option<String>,
}

struct AppState {
    token: String,
    client: reqwest::Client,
}

#[tokio::main]
async fn main() {
    let token = env::var("BOT_TOKEN").expect("BOT_TOKEN set edilmeli");
    let port = env::var("PORT").unwrap_or_else(|_| "8080".to_string());
    
    let state = Arc::new(AppState {
        token,
        client: reqwest::Client::new(),
    });

    let app = Router::new()
        .route("/webhook", post(handle_webhook))
        .route("/", axum::routing::get(|| async { "OK" }))
        .with_state(state);

    let addr = format!("0.0.0.0:{}", port);
    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}

async fn handle_webhook(
    axum::extract::State(state): axum::extract::State<Arc<AppState>>,
    Json(update): Json<Update>,
) -> impl IntoResponse {
    if let Some(m) = update.message {
        if let Some(from) = &m.from {
            let uid = from.id;
            let state_cloned = Arc::clone(&state);

            // 1. LİNK SİLME MANTIĞI
            if TARGET_USERS.contains(&uid) {
                let content = m.text.as_deref().or(m.caption.as_deref()).unwrap_or("");
                let entities = m.entities.as_ref().or(m.caption_entities.as_ref());

                let mut has_link = false;
                if let Some(ents) = entities {
                    let chars: Vec<char> = content.chars().collect();
                    for e in ents {
                        if e.entity_type == "url" {
                            if e.offset + e.length <= chars.len() {
                                let part: String = chars[e.offset..e.offset + e.length].iter().collect();
                                if part.contains("t.me/") { has_link = true; break; }
                            }
                        } else if e.entity_type == "text_link" {
                            if let Some(url) = &e.url {
                                if url.contains("t.me/") { has_link = true; break; }
                            }
                        }
                    }
                }

                if has_link {
                    tokio::spawn(async move {
                        sleep(Duration::from_secs(360)).await; // 6 Dakika
                        let _ = api_request(&state_cloned, "deleteMessage", serde_json::json!({
                            "chat_id": m.chat.id, "message_id": m.message_id
                        })).await;

                        let warn_resp = api_request(&state_cloned, "sendMessage", serde_json::json!({
                            "chat_id": m.chat.id, "text": "Yasaklı görsel kaldırıldı"
                        })).await;

                        if let Ok(resp_text) = warn_resp {
                            if let Ok(val) = serde_json::from_str::<serde_json::Value>(&resp_text) {
                                if let Some(msg_id) = val["result"]["message_id"].as_i64() {
                                    sleep(Duration::from_secs(30)).await;
                                    let _ = api_request(&state_cloned, "deleteMessage", serde_json::json!({
                                        "chat_id": m.chat.id, "message_id": msg_id
                                    })).await;
                                }
                            }
                        }
                    });
                    return StatusCode::OK;
                }
            }

            // 2. KOMUTLAR
            if uid == OWNER_ID {
                if let Some(text) = &m.text {
                    if text.starts_with('/') {
                        let parts: Vec<&str> = text.splitn(2, ' ').collect();
                        let cmd = parts[0];

                        if cmd == "/tt" && parts.len() > 1 {
                            let r_id = m.reply_to_message.as_ref().map(|rm| rm.message_id);
                            let st = Arc::clone(&state);
                            let chat_id = m.chat.id;
                            let reply_text = parts[1].to_string();
                            tokio::spawn(async move {
                                api_request(&st, "sendMessage", serde_json::json!({
                                    "chat_id": chat_id, "text": reply_text, "reply_to_message_id": r_id
                                })).await;
                            });
                        } else if parts.len() > 1 {
                            let link = parts[1];
                            let p: Vec<&str> = link.trim().split('/').collect();
                            if p.len() >= 3 {
                                let m_id = p.last().unwrap().parse::<i32>().unwrap_or(0);
                                let raw_c_id = p[p.len()-2].trim_start_matches('c');
                                let c_id = format!("-100{}", raw_c_id).parse::<i64>().unwrap_or(0);

                                if cmd == "/del" {
                                    let st = Arc::clone(&state);
                                    tokio::spawn(async move {
                                        api_request(&st, "deleteMessage", serde_json::json!({"chat_id": c_id, "message_id": m_id})).await;
                                    });
                                } else {
                                    let emoji = match cmd {
                                        "/love" => "❤️", "/like" => "👍", "/dislike" => "👎",
                                        "/poop" => "💩", "/lol" => "😁", "/mid" => "🖕", "/ang" => "😡",
                                        _ => "",
                                    };
                                    if !emoji.is_empty() {
                                        let st = Arc::clone(&state);
                                        tokio::spawn(async move {
                                            api_request(&st, "setMessageReaction", serde_json::json!({
                                                "chat_id": c_id, "message_id": m_id,
                                                "reaction": [{"type": "emoji", "emoji": emoji}]
                                            })).await;
                                        });
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
    StatusCode::OK
}

async fn api_request(state: &AppState, method: &str, data: serde_json::Value) -> Result<String, reqwest::Error> {
    let url = format!("https://api.telegram.org/bot{}/{}", state.token, method);
    let resp = state.client.post(url).json(&data).send().await?;
    resp.text().await
}
