FROM golang:1.21-alpine AS builder

WORKDIR /app

# Копируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build \
  -ldflags="-s -w -X main.version=1.0.0" \
  -o /growth-monitor ./cmd/bot

# Финальный образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Копируем бинарник
COPY --from=builder /growth-monitor /growth-monitor

# Создаем директории
RUN mkdir -p /logs

# Указываем переменные окружения по умолчанию
ENV LOG_LEVEL=info
ENV LOG_TO_CONSOLE=true

# Запуск приложения
ENTRYPOINT ["/growth-monitor"]
CMD ["--log-level=info"]