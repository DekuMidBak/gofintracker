FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /out/transaction-service ./cmd/transaction-service

FROM alpine:3.22

RUN addgroup -S app && adduser -S app -G app

USER app
WORKDIR /app

COPY --from=build /out/transaction-service /app/transaction-service

EXPOSE 50052

ENTRYPOINT ["/app/transaction-service"]
