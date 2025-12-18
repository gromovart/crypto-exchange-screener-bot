# Makefile для Crypto Exchange Screener Bot

.PHONY: all bot signals test clean

all: bot

bot:
	go run cmd/bot/main.go

build-bot:
	go build -o bin/bot cmd/bot/main.go

build: build-bot

clean:
	rm -rf bin/ logs/*.log

run-debug:
	./debug_run.sh

install:
	go mod download