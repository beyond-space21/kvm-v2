.PHONY: run migrate-up migrate-down test build clear-data

run:
	RUN_MIGRATIONS=true go run .

build:
	go build -o bin/lms .

test:
	go test ./...

migrate-up:
	migrate -path services/db/migrations -database "$$DATABASE_URL" up

migrate-down:
	migrate -path services/db/migrations -database "$$DATABASE_URL" down 1

clear-data:
	go run ./scripts/clear_data --yes
