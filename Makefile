.PHONY: all test test-backend test-frontend build build-mtp clean

all: test build

build-mtp:
	@mkdir -p tools/mtp
	@echo "==> Compiling zimap-mtp sidecar binary..."
	go build -o tools/mtp/zimap-mtp ./cmd/zimap-mtp
	@echo "==> zimap-mtp compiled successfully into tools/mtp/zimap-mtp"

test-backend: build-mtp
	@echo "==> Running backend Go tests..."
	go test -count=1 ./...

test-frontend:
	@echo "==> Running frontend Vitest tests..."
	cd frontend && npm test

test: test-backend test-frontend

GOPATH ?= $(shell go env GOPATH)
WAILS ?= $(shell which wails 2>/dev/null || echo $(GOPATH)/bin/wails)

build: build-mtp
	@echo "==> Building frontend production assets..."
	cd frontend && npm run build
	@echo "==> Building main ZiMap application..."
	$(WAILS) build
	@for app in build/bin/ZiMap.app build/bin/zimap.app; do \
		if [ -d "$$app/Contents/MacOS" ]; then \
			echo "==> Embedding zimap-mtp into $$app/Contents/MacOS/..."; \
			cp tools/mtp/zimap-mtp "$$app/Contents/MacOS/zimap-mtp"; \
		fi \
	done

clean:
	rm -rf build/bin tools/mtp/zimap-mtp frontend/dist
