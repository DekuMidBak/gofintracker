.PHONY: test tidy fmt

test:
	go test ./...

tidy:
	go mod tidy

fmt:
	go fmt ./...
