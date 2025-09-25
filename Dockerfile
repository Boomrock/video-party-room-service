# Stage 1: Сборка приложения
FROM golang:1.22-alpine AS builder

# Устанавливаем рабочую директорию внутри контейнера
WORKDIR /app

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.* ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем бинарник
# Используем флаги для уменьшения размера бинарника и статической линковки
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a \
    -installsuffix cgo \
    -ldflags="-s -w" \
    -o main .

# Stage 2: Запуск приложения
FROM alpine:latest AS final

# Устанавливаем необходимые зависимости (например, для работы с сертификатами)
RUN apk --no-cache add ca-certificates

# Устанавливаем рабочую директорию
WORKDIR /root/

# Копируем бинарник из первого этапа
COPY --from=builder /app/main .

# Делаем бинарник исполняемым
RUN chmod +x ./main

# Открываем порт (если нужно)
EXPOSE 8080

# Запускаем приложение
CMD ["./main"]