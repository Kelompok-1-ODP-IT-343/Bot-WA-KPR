# ===== Builder =====
FROM golang:1.24-alpine AS builder
WORKDIR /src

# timezone & certs & build deps
RUN apk add --no-cache ca-certificates tzdata git

# cache modules
COPY go.mod go.sum ./
RUN go mod download

# copy all source
COPY . .

# build metadata (opsional)
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

# build binary (CGO free)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${BUILD_DATE}" \
  -o /out/bot ./cmd/bot

# ===== Runtime =====
FROM debian:bookworm-slim

# user non-root + certs + tz + folder store
RUN useradd -u 10001 -m appuser \
 && apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates tzdata \
 && rm -rf /var/lib/apt/lists/* \
 && mkdir -p /data/wa-store \
 && chown -R appuser:appuser /data

WORKDIR /app
COPY --from=builder /out/bot /app/bot
COPY --from=builder /src/ddl.sql /app/ddl.sql
COPY --from=builder /src/sql_audit.jsonl /app/sql_audit.jsonl
COPY --from=builder /src/prompt.txt /app/prompt.txt

# ENV default (override via env di deploy/CI)
ENV HTTP_ADDR=":8080" \
    WHATSAPP_STORE_PATH="/data/wa-store/whatsmeow.db" \
    TZ="Asia/Jakarta"

EXPOSE 8080
USER appuser
ENTRYPOINT ["/app/bot"]
