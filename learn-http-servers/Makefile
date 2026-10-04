ENV_FILE=.env

ifneq (,$(wildcard $(ENV_FILE)))
include $(ENV_FILE)
endif

dev:
	air
run:
	go run .
test:
	go test -json ./... | gotestfmt
fmt:
	golangci-lint fmt
lint:
	golangci-lint run
fix:
	golangci-lint run --fix
clean:
	go clean

db-migrate-up:
	goose postgres "$(DB_URL)" -dir sql/schema up

db-migrate-down:
	goose postgres "$(DB_URL)" -dir sql/schema down

sqlc:
	sqlc generate
