all: test build

test:
	go test -race -v ./...

build:
	go build -o bin/AvWxTelegramBot .

run:
	go run .
