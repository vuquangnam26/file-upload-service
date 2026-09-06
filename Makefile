include .env
export

# ─── Variables ────────────────────────────────────────────────────────────────
DB_URL=postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable
MIGRATION_DIR=scripts/migrations

# ─── App ──────────────────────────────────────────────────────────────────────
.PHONY: run
run:
	go run cmd/api/main.go

.PHONY: build
build:
	go build -o bin/api cmd/api/main.go

# ─── Migration ────────────────────────────────────────────────────────────────
.PHONY: migrate-up
migrate-up:
	migrate -path $(MIGRATION_DIR) -database "$(DB_URL)" up

.PHONY: migrate-down
migrate-down:
	migrate -path $(MIGRATION_DIR) -database "$(DB_URL)" down 1

.PHONY: migrate-drop
migrate-drop:
	migrate -path $(MIGRATION_DIR) -database "$(DB_URL)" drop -f

.PHONY: migrate-version
migrate-version:
	migrate -path $(MIGRATION_DIR) -database "$(DB_URL)" version

.PHONY: migrate-force
migrate-force:
	migrate -path $(MIGRATION_DIR) -database "$(DB_URL)" force $(v)

## Usage: make migrate-create name=add_users_table
.PHONY: migrate-create
migrate-create:
	migrate create -ext sql -dir $(MIGRATION_DIR) -seq $(name)

# ─── Docker ───────────────────────────────────────────────────────────────────
.PHONY: docker-up
docker-up:
	docker compose up -d

.PHONY: docker-down
docker-down:
	docker compose down

.PHONY: docker-reset
docker-reset:
	docker compose down -v && docker compose up -d

# ─── Go tooling ───────────────────────────────────────────────────────────────
.PHONY: tidy
tidy:
	go mod tidy

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: test
test:
	go test ./... -v
