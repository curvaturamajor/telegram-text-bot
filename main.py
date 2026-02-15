import os
from telegram import Update
from telegram.ext import (
    ApplicationBuilder,
    ContextTypes,
    MessageHandler,
    filters,
)

BOT_TOKEN = os.environ.get("BOT_TOKEN")
RENDER_EXTERNAL_URL = os.environ.get("RENDER_EXTERNAL_URL")

print("DEBUG BOT_TOKEN:", "VAR" if BOT_TOKEN else "YOK")

async def handle_txt(update: Update, context: ContextTypes.DEFAULT_TYPE):
    msg = update.message
    text = msg.text

    if not text.startswith("/txt"):
        return

    member = await context.bot.get_chat_member(
        chat_id=msg.chat.id,
        user_id=msg.from_user.id
    )

    if member.status not in ("administrator", "creator"):
        return

    content = text[4:].strip()
    if not content:
        return

    if msg.reply_to_message:
        await context.bot.send_message(
            chat_id=msg.chat.id,
            text=content,
            reply_to_message_id=msg.reply_to_message.message_id
        )
    else:
        await context.bot.send_message(
            chat_id=msg.chat.id,
            text=content
        )

def main():
    if not BOT_TOKEN:
        raise RuntimeError("BOT_TOKEN ortam değişkeni YOK")

    if not RENDER_EXTERNAL_URL:
        raise RuntimeError("RENDER_EXTERNAL_URL yok")

    app = ApplicationBuilder().token(BOT_TOKEN).build()

    app.add_handler(
        MessageHandler(filters.TEXT & filters.ChatType.GROUPS, handle_txt)
    )

    port = int(os.environ.get("PORT", 10000))

    print("🚀 Bot Render üzerinde webhook ile başlatılıyor")

    app.run_webhook(
        listen="0.0.0.0",
        port=port,
        webhook_url=f"{RENDER_EXTERNAL_URL}/{BOT_TOKEN}",
    )

if __name__ == "__main__":
    main()
