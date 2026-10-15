# PRISMA — tareas de desarrollo (Linux/macOS). En Windows usa build.ps1.

MODULO       := github.com/bayalad7/prisma
VERSION      ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.0.0-dev)
COMMIT       ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo desconocido)
FECHA        ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
STATICCHECK  := honnef.co/go/tools/cmd/staticcheck@v0.8.2
PLATAFORMAS  := windows/amd64 windows/386 linux/amd64 linux/arm64 darwin/amd64 darwin/arm64
LDFLAGS      := -s -w \
	-X $(MODULO)/internal/version.Version=$(VERSION) \
	-X $(MODULO)/internal/version.Commit=$(COMMIT) \
	-X $(MODULO)/internal/version.Fecha=$(FECHA)

export CGO_ENABLED := 0

.PHONY: test build build-todo lint limpiar

test:
	go test ./...

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o dist/prisma$(shell go env GOEXE) ./cmd/prisma

build-todo:
	@for p in $(PLATAFORMAS); do \
		os=$${p%/*}; arch=$${p#*/}; ext=; [ $$os = windows ] && ext=.exe; \
		echo "==> $$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" \
			-o dist/prisma-$$os-$$arch$$ext ./cmd/prisma || exit 1; \
	done

lint:
	go vet ./...
	go run $(STATICCHECK) ./...

limpiar:
	rm -rf dist
