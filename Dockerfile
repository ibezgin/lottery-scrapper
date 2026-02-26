# Этап 1: Сборка приложения
FROM golang:1.26 AS builder

WORKDIR /app

# Копируем файлы зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем бинарный файл
RUN CGO_ENABLED=0 GOOS=linux go build -o lottery-scrapper main.go

# Этап 2: Финальный образ
FROM debian:bullseye-slim

WORKDIR /app

# Устанавливаем зависимости для Chromium и Playwright
# Нам нужны библиотеки для работы браузера и сам playwright для скачивания, если мы хотим сохранить логику скачивания в рантайме
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    libnss3 \
    libatk1.0-0 \
    libatk-bridge2.0-0 \
    libcups2 \
    libdrm2 \
    libxkbcommon0 \
    libxcomposite1 \
    libxdamage1 \
    libxext6 \
    libxfixes3 \
    libxrandr2 \
    libgbm1 \
    libpango-1.0-0 \
    libcairo2 \
    libasound2 \
    && rm -rf /var/lib/apt/lists/*

# Копируем собранный бинарник из первого этапа
COPY --from=builder /app/lottery-scrapper .

# Создаем директорию для браузера
RUN mkdir -p /app/browser

# Переменная окружения для Playwright
ENV PLAYWRIGHT_BROWSERS_PATH=/app/browser

# Запуск приложения
CMD ["./lottery-scrapper"]
