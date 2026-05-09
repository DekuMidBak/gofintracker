FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /out/user-service ./cmd/user-service

FROM alpine:3.22

RUN addgroup -S app && adduser -S app -G app

USER app
WORKDIR /app

COPY --from=build /out/user-service /app/user-service

EXPOSE 50051

ENTRYPOINT ["/app/user-service"]
