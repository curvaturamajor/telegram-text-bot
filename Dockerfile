# 1. Build stage
FROM rust:1.84 as builder

RUN rustup target add x86_64-unknown-linux-musl \
 && apt-get update && apt-get install -y musl-tools pkg-config build-essential

WORKDIR /app
COPY . .

RUN cargo build --release --target x86_64-unknown-linux-musl

# 2. Minimal final stage
FROM scratch
COPY --from=builder /app/target/x86_64-unknown-linux-musl/release/telegram_bot /

EXPOSE 8080
CMD ["/telegram_bot"]
