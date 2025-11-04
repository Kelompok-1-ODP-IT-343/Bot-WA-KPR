# ===== Builder (compile Go) =====
FROM golang:1.22-alpine AS builder
WORKDIR /src

# deps dasar utk build & zona waktu
RUN apk add --no-cache ca-certificates tzdata

# pakai cache maksimal
COPY go.mod go.sum ./
RUN go mod download

# bawa semua source
COPY . .

# metadata build (opsional, bisa diisi dari CI)
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

# build binary dari cmd/bot
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${BUILD_DATE}" \
    -o /out/bot ./cmd/bot

# ===== Runtime (ringan & punya CA cert) =====
FROM debian:bookworm-slim
# user non-root + sertifikat TLS + timezone + folder store WA
RUN useradd -u 10001 -m appuser \
 && apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates tzdata \
 && rm -rf /var/lib/apt/lists/* \
 && mkdir -p /data/wa-store \
 && chown -R appuser:appuser /data

WORKDIR /app
COPY --from=builder /out/bot /app/bot

# default env (bisa dioverride via .env / kubernetes)
ENV HTTP_ADDR=":8080" \
    WHATSAPP_STORE_PATH="/data/wa-store" \
    TZ="Asia/Jakarta"

EXPOSE 8080
USER appuser
ENTRYPOINT ["/app/bot"]
