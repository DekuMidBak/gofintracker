FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /out/telegram-bot ./cmd/telegram-bot

FROM alpine:3.22

RUN addgroup -S app && adduser -S app -G app

USER app
WORKDIR /app

COPY --from=build /out/telegram-bot /app/telegram-bot

ENTRYPOINT ["/app/telegram-bot"]
