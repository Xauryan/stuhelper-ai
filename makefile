FRONTEND_ROOT_DIR = ./web
FRONTEND_CLASSIC_DIR = ./web/classic
BACKEND_DIR = .
DEV_FRONTEND_CLASSIC_PORT ?= 3001

.PHONY: all build-frontend build-frontend-classic build-all-frontends start-backend dev dev-api dev-web dev-web-classic

all: build-all-frontends start-backend

build-frontend:
	@$(MAKE) build-frontend-classic

build-frontend-classic:
	@echo "Building classic frontend..."
	@cd $(FRONTEND_ROOT_DIR) && bun install --frozen-lockfile
	@cd $(FRONTEND_CLASSIC_DIR) && VITE_REACT_APP_VERSION=$(cat ../../VERSION) bun run build

build-all-frontends: build-frontend-classic

start-backend:
	@echo "Starting backend dev server..."
	@cd $(BACKEND_DIR) && go run main.go &

dev-api:
	@echo "Starting backend services (docker)..."
	@docker compose -f docker-compose.dev.yml up -d

dev-web:
	@echo "Starting frontend dev server..."
	@cd $(FRONTEND_ROOT_DIR) && bun install
	@cd $(FRONTEND_CLASSIC_DIR) && bun run dev -- --host 0.0.0.0 --port $(DEV_FRONTEND_CLASSIC_PORT)

dev-web-classic:
	@echo "Starting classic frontend dev server..."
	@cd $(FRONTEND_ROOT_DIR) && bun install
	@cd $(FRONTEND_CLASSIC_DIR) && bun run dev -- --host 0.0.0.0 --port $(DEV_FRONTEND_CLASSIC_PORT)

dev: dev-api dev-web
