all: test build

test:
	go test -v ./...

race:
	go test -race -v ./...

build:
	go build -o bin/AvWxTelegramBot .

clean:
	rm -f bin/AvWxTelegramBot

run:
	go run .
