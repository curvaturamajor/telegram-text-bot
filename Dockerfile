FROM rust:1.84 as builder

WORKDIR /app
COPY . .

RUN cargo build --release

FROM debian:bookworm-slim

WORKDIR /app
COPY --from=builder /app/target/release/telegram_bot .

ENV PORT=8080

CMD ["./telegram_bot"]
