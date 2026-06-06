# ЭТАП 1: Сборка (Build)
# Используем официальный образ Go с Alpine (легкий Linux)
FROM golang:1.25-alpine AS builder

# Устанавливаем рабочую директорию внутри контейнера
WORKDIR /app

# Копируем только файлы зависимостей сначала (это кеширует слой и ускоряет сборку)
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь исходный код
COPY . .

# Собираем статический бинарный файл.
# CGO_ENABLED=0 гарантирует, что бинарник не будет зависеть от системных библиотек C
# GOOS=linux указывает, что собираем под Linux (даже если вы на Windows/Mac)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o chat_server ./cmd/server

# ЭТАП 2: Запуск (Run)
FROM alpine:latest

# Устанавливаем сертификаты и часовые пояса
RUN apk --no-cache add ca-certificates tzdata

# Рабочая директория (лучше НЕ использовать /root/ для обычного юзера)
WORKDIR /app

# Создаём группу и пользователя (флаги -S и -D делают их системными и без пароля)
RUN addgroup -g 1001 -S appgroup && \
    adduser -S appuser -G appgroup -u 1001

# Копируем бинарник из этапа сборки
COPY --from=builder /app/chat_server .

# Передаём права на папку новому пользователю
RUN chown -R appuser:appgroup /app

EXPOSE 8080

# переключаемся на юзера ДО запуска CMD
USER appuser

CMD ["./chat_server"]
