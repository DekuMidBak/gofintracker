FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /out/analytics-service ./cmd/analytics-service

FROM alpine:3.22

RUN addgroup -S app && adduser -S app -G app

USER app
WORKDIR /app

COPY --from=build /out/analytics-service /app/analytics-service

EXPOSE 50053

ENTRYPOINT ["/app/analytics-service"]
