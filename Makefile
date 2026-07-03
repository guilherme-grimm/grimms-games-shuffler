SQLC := $(shell go env GOPATH)/bin/sqlc

.PHONY: dev build web gen lint test run check

check: lint test ## pre-push gate — no CI, every push deploys
	$(SQLC) diff
	cd web && pnpm build

build: web ## production binary with embedded frontend
	go build -o bin/ggs ./cmd/ggs

web:
	cd web && pnpm install --ignore-scripts && pnpm build

gen:
	$(SQLC) generate

lint:
	golangci-lint run

test:
	go test -race ./...

run: ## backend only; pair with `cd web && pnpm dev` for the frontend
	DATA_DIR=./.data PORT=8080 go run ./cmd/ggs
