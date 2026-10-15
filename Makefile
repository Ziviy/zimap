.PHONY: all test test-backend test-frontend build build-mtp release-local clean

all: test build

build-mtp:
	@mkdir -p tools/mtp
	@echo "==> Compiling zimap-mtp sidecar binary..."
	./scripts/build-mtp.sh
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

release-local: build
	@echo "==> Packaging local release archive..."
	@APP_PATH=""; \
	if [ -d "build/bin/ZiMap.app" ]; then APP_PATH="build/bin/ZiMap.app"; \
	elif [ -d "build/bin/zimap.app" ]; then APP_PATH="build/bin/zimap.app"; fi; \
	if [ -z "$$APP_PATH" ]; then \
		echo "Error: Application bundle not found in build/bin!"; \
		exit 1; \
	fi; \
	ARCH=$$(go env GOARCH); \
	OUT_ZIP="ZiMap-darwin-$${ARCH}.zip"; \
	ditto -c -k --keepParent "$$APP_PATH" "$$OUT_ZIP"; \
	echo "==> Successfully packaged $$OUT_ZIP"

clean:
	rm -rf build/bin tools/mtp/zimap-mtp frontend/dist *.zip

