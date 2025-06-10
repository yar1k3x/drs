# Этап сборки
FROM golang:1.24.3 AS builder

WORKDIR /app

# Копируем go.mod и go.sum из текущей директории
COPY go.mod go.sum ./
RUN go mod download

# Копируем всё содержимое текущей директории (это и есть весь проект)
COPY . .

# Сборка бинарника из main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o drs main.go

# Финальный образ — минимальный scratch
FROM scratch

WORKDIR /app

COPY --from=builder /app/drs .

COPY --from=builder /app/service ./service
COPY --from=builder /app/proto ./proto
COPY --from=builder /app/db ./db
COPY --from=builder /app/notification ./notification
COPY --from=builder /app/server ./server
COPY --from=builder /app/validate ./validate

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENV DB_HOST=mysql \
    DB_PORT=3306 \
    DB_USER=root \
    DB_PASSWORD=root \
    DB_NAME=drs_db

ENTRYPOINT ["./drs"]
