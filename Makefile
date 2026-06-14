.PHONY: run test check build docker-build docker-up docker-down

run:
	go run ./cmd/server

test:
	go test ./...

check:
	go test ./...
	node --check web/static/app.js

build:
	go build -o bin/imagecut ./cmd/server

docker-build:
	docker build -t imagecut:local .

docker-up:
	docker compose up --build

docker-down:
	docker compose down
