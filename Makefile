VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build dev sync clean

build: frontend-build
	cp -r frontend/build/* cmd/github-lens/frontend/ 2>/dev/null || cp -r frontend/build/. cmd/github-lens/frontend/
	go build -trimpath -ldflags "$(LDFLAGS)" -o github-lens ./cmd/github-lens

frontend-build:
	cd frontend && npm ci && npm run build

dev:
	@trap 'kill 0' EXIT; \
	go run -ldflags "$(LDFLAGS)" ./cmd/github-lens & \
	cd frontend && npm run dev & \
	wait

sync:
	go run -ldflags "$(LDFLAGS)" ./cmd/github-lens --sync-once

clean:
	rm -f github-lens
	rm -rf dist/
	rm -f *.db *.db-shm *.db-wal
	rm -rf frontend/build
	rm -rf frontend/.svelte-kit
	rm -rf frontend/node_modules
