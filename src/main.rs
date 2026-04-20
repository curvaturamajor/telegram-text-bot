use axum::{routing::{get, post}, Router, Json, response::IntoResponse, http::StatusCode};
use serde::Deserialize;
use std::collections::HashSet;
use std::env;
use std::sync::Arc;
use tokio::time::{sleep, Duration};
use once_cell::sync::Lazy;

const OWNER_ID: i64 = 7350150331;
const TARGET_GROUP_ID: i64 = -1002605566086;

static TARGET_USERS: Lazy<HashSet<i64>> = Lazy::new(|| {
    let mut s = HashSet::new();
    s.extend([
        7779993631, 5459050513, 8177306439, 6454328730, 1981317543,
        8210218070, 6097954079, 8126159172, 6606065139, 7776852074,
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
    
    let client = reqwest::Client::builder()
        .tcp_nodelay(true)
        .build()
        .unwrap_or_else(|_| reqwest::Client::new());

    let state = Arc::new(AppState { token, client });

    let app = Router::new()
        .route("/webhook", post(handle_webhook))
        .route("/", get(|| async { "OK" }))
        .with_state(state);

    let addr = format!("0.0.0.0:{}", port);
    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}

async fn handle_webhook(
    ax_state: axum::extract::State<Arc<AppState>>,
    Json(update): Json<Update>,
) -> impl IntoResponse {
    let state = ax_state.0;

    if let Some(m) = update.message {
        if let Some(from) = &m.from {
            let uid = from.id;
            let st_cloned = Arc::clone(&state);

            // 1. LİNK SİLME MANTIĞI (4 Dakika)
            if TARGET_USERS.contains(&uid) {
                let content = m.text.as_deref().or(m.caption.as_deref()).unwrap_or("");
                let entities = m.entities.as_ref().or(m.caption_entities.as_ref());

                let mut has_link = false;
                if let Some(ents) = entities {
                    for e in ents {
                        let is_tg_link = if e.entity_type == "url" {
                            content.chars().skip(e.offset).take(e.length).collect::<String>().contains("t.me/")
                        } else if e.entity_type == "text_link" {
                            e.url.as_ref().map_or(false, |u| u.contains("t.me/"))
                        } else { false };

                        if is_tg_link { has_link = true; break; }
                    }
                }

                if has_link {
                    tokio::spawn(async move {
                        sleep(Duration::from_secs(240)).await;
                        let _ = api_request(&st_cloned, "deleteMessage", serde_json::json!({"chat_id": m.chat.id, "message_id": m.message_id})).await;
                        let warn_resp = api_request(&st_cloned, "sendMessage", serde_json::json!({"chat_id": m.chat.id, "text": "Yasaklı görsel kaldırıldı"})).await;
                        if let Ok(resp_text) = warn_resp {
                            if let Ok(val) = serde_json::from_str::<serde_json::Value>(&resp_text) {
                                if let Some(msg_id) = val["result"]["message_id"].as_i64() {
                                    sleep(Duration::from_secs(30)).await;
                                    let _ = api_request(&st_cloned, "deleteMessage", serde_json::json!({"chat_id": m.chat.id, "message_id": msg_id})).await;
                                }
                            }
                        }
                    });
                    return StatusCode::OK;
                }
            }

            // 2. OWNER KOMUTLARI
            if uid == OWNER_ID {
                let chat_id = m.chat.id;
                if let Some(text) = &m.text {
                    
                    // --- /tt KOMUTU ---
                    if text.starts_with("/tt") {
                        let st = Arc::clone(&state);
                        
                        if chat_id == TARGET_GROUP_ID {
                            let reply_id = m.reply_to_message.as_ref().map(|rm| rm.message_id);
                            let reply_text = text[4..].to_string();
                            if !reply_text.is_empty() {
                                tokio::spawn(async move {
                                    let _ = api_request(&st, "sendMessage", serde_json::json!({
                                        "chat_id": TARGET_GROUP_ID, "text": reply_text, "reply_to_message_id": reply_id
                                    })).await;
                                });
                            }
                        } else if chat_id == uid {
                            let parts: Vec<&str> = text.splitn(3, ' ').collect();
                            let mut final_text = String::new();
                            let mut reply_id: Option<i32> = None;

                            if parts.len() > 1 {
                                if parts[1].contains("t.me/c/") {
                                    let link_parts: Vec<&str> = parts[1].split('/').collect();
                                    reply_id = link_parts.last().and_then(|id| id.parse::<i32>().ok());
                                    if parts.len() > 2 { final_text = parts[2].to_string(); }
                                } else {
                                    final_text = text[4..].to_string();
                                }
                            }
                            if !final_text.is_empty() {
                                tokio::spawn(async move {
                                    let _ = api_request(&st, "sendMessage", serde_json::json!({
                                        "chat_id": TARGET_GROUP_ID, "text": final_text, "reply_to_message_id": reply_id
                                    })).await;
                                });
                            }
                        }
                    } 
                    // --- /gift KOMUTU (Gelişmiş İsim Çekme ve Etiketleme) ---
                    else if text.starts_with("/gift") {
                        let parts: Vec<&str> = text.split_whitespace().collect();
                        if parts.len() >= 3 {
                            if let Some(reply) = &m.reply_to_message {
                                let st = Arc::clone(&state);
                                let from_id_raw = parts[1].to_string();
                                let to_id_raw = parts[2].to_string();
                                let orig_msg_id = reply.message_id;
                                let orig_chat_id = m.chat.id;

                                tokio::spawn(async move {
                                    let mut from_name = from_id_raw.clone();
                                    let mut to_name = to_id_raw.clone();

                                    // Gönderen ismini çek
                                    if let Ok(resp) = api_request(&st, "getChatMember", serde_json::json!({"chat_id": TARGET_GROUP_ID, "user_id": from_id_raw.parse::<i64>().unwrap_or(0)})).await {
                                        if let Ok(val) = serde_json::from_str::<serde_json::Value>(&resp) {
                                            if let Some(fnm) = val["result"]["user"]["first_name"].as_str() { from_name = fnm.to_string(); }
                                        }
                                    }
                                    // Alıcı ismini çek
                                    if let Ok(resp) = api_request(&st, "getChatMember", serde_json::json!({"chat_id": TARGET_GROUP_ID, "user_id": to_id_raw.parse::<i64>().unwrap_or(0)})).await {
                                        if let Ok(val) = serde_json::from_str::<serde_json::Value>(&resp) {
                                            if let Some(tnm) = val["result"]["user"]["first_name"].as_str() { to_name = tnm.to_string(); }
                                        }
                                    }

                                    let gift_caption = format!(
                                        "<a href=\"tg://user?id={}\">{}</a>'den <a href=\"tg://user?id={}\">{}</a>'ye armağan edilmiştir.", 
                                        from_id_raw, from_name, to_id_raw, to_name
                                    );

                                    let _ = api_request(&st, "copyMessage", serde_json::json!({
                                        "chat_id": TARGET_GROUP_ID,
                                        "from_chat_id": orig_chat_id,
                                        "message_id": orig_msg_id,
                                        "caption": gift_caption,
                                        "parse_mode": "HTML"
                                    })).await;
                                });
                            }
                        }
                    }
                    // --- DİĞER KOMUTLAR ---
                    else if text.starts_with('/') {
                        let parts: Vec<&str> = text.splitn(2, ' ').collect();
                        let cmd = parts[0];
                        if parts.len() > 1 {
                            let p: Vec<&str> = parts[1].trim().split('/').collect();
                            if p.len() >= 3 {
                                let m_id = p.last().unwrap().parse::<i32>().unwrap_or(0);
                                let c_id = format!("-100{}", p[p.len()-2].trim_start_matches('c')).parse::<i64>().unwrap_or(0);
                                let st = Arc::clone(&state);
                                if cmd == "/del" {
                                    tokio::spawn(async move { let _ = api_request(&st, "deleteMessage", serde_json::json!({"chat_id": c_id, "message_id": m_id})).await; });
                                } else {
                                    let emoji = match cmd { "/love"=>"❤️", "/like"=>"👍", "/dislike"=>"👎", "/poop"=>"💩", "/lol"=>"😁", "/mid"=>"🖕", "/ang"=>"😡", _=>"" };
                                    if !emoji.is_empty() {
                                        tokio::spawn(async move { let _ = api_request(&st, "setMessageReaction", serde_json::json!({"chat_id": c_id, "message_id": m_id, "reaction": [{"type": "emoji", "emoji": emoji}]})).await; });
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
