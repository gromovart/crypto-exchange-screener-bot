# Makefile для Crypto Exchange Screener Bot

.PHONY: all bot signals test clean

all: bot

bot:
	go run cmd/bot/main.go

growth:
	go run cmd/signals/main.go

build-bot:
	go build -o bin/bot cmd/bot/main.go

build-growth:
	go build -o bin/growth cmd/signals/main.go

build: build-bot build-growth

test:
	go test ./...

clean:
	rm -rf bin/ logs/*.log

run-debug:
	./debug_run.sh

run-test:
	./test_signal.sh

install:
	go mod download