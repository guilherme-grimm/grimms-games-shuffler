SQLC := $(shell go env GOPATH)/bin/sqlc

.PHONY: dev seed build web gen lint test run check

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

dev: seed ## fresh fake session, backend on :8080 + Vite on :5173, ctrl-c stops both
	@cd web && pnpm install --ignore-scripts
	@echo ">> open http://localhost:5173 and set: document.cookie = 'ggs_session=testtoken'"
	@trap 'kill 0' EXIT; \
	DATA_DIR=./.data PORT=8080 BASE_URL=http://localhost:8080 STEAM_API_KEY=devseed \
		go run ./cmd/ggs & \
	cd web && pnpm dev

seed: ## reset ./.data with a fake player, session and library (no Steam login)
	mkdir -p .data && rm -f .data/ggs.db .data/ggs.db-wal .data/ggs.db-shm
	go run ./cmd/devseed ./.data
