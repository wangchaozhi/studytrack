.PHONY: up down logs build test dev-api dev-web

up:
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs -f

build:
	cd backend && go build ./...
	cd frontend && npm run build

test:
	cd backend && go test ./...
	cd frontend && npm run lint

dev-api:
	cd backend && go run ./cmd/server

dev-web:
	cd frontend && npm run dev
