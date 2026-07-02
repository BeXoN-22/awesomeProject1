# ── Stage 1: build ──────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

# SSL-сертификаты нужны для HTTPS-запросов к RSS-источникам
RUN apk add --no-cache ca-certificates

# Создаём пользователя здесь, потому что scratch не имеет adduser
RUN adduser -D -u 1001 appuser

WORKDIR /app

# Сначала зависимости — слой кэшируется, пока go.mod/go.sum не изменятся
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 → статический бинарник (нет зависимости от libc)
# -ldflags="-w -s" → убираем debug-символы, ~30% меньше размер
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server .

# ── Stage 2: final ───────────────────────────────────────────────────────────
FROM scratch

# Копируем только три вещи из builder:
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /app/server /server

USER appuser

EXPOSE 8080
ENTRYPOINT ["/server"]