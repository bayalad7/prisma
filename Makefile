# PRISMA — tareas de desarrollo (Linux/macOS). En Windows usa build.ps1.

MODULO       := github.com/bayalad7/prisma
# Sin la «v» de la etiqueta: 1.2.3. Fuera de una etiqueta, git describe da algo como 1.2.3-4-gabc123.
VERSION      ?= $(patsubst v%,%,$(shell git describe --tags --always --dirty 2>/dev/null || echo 0.0.0-dev))
COMMIT       ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo desconocido)
FECHA        ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
# Llave pública Ed25519 con la que «prisma actualizar» verifica las versiones (ver docs/publicar.md).
LLAVE        ?= $(shell cat empaquetado/llave-releases.pub 2>/dev/null)
STATICCHECK  := honnef.co/go/tools/cmd/staticcheck@v0.8.2
NFPM         := github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.48.0
PLATAFORMAS  := windows/amd64 windows/386 linux/amd64 linux/arm64 darwin/amd64 darwin/arm64
LDFLAGS      := -s -w \
	-X $(MODULO)/internal/version.Version=$(VERSION) \
	-X $(MODULO)/internal/version.Commit=$(COMMIT) \
	-X $(MODULO)/internal/version.Fecha=$(FECHA) \
	-X $(MODULO)/internal/actualizacion.LlavePublica=$(LLAVE)

export CGO_ENABLED := 0

.PHONY: test test-sdk build build-todo deb pkg checksums lint limpiar

test:
	go test ./...

# SDK JavaScript y firmadores de referencia: requieren Node.js, PHP 8.2 (sodium) y Python con cryptography.
test-sdk:
	cd sdk/js && npm test
	node --test "sdk/firmadores/node/*.test.mjs"
	php sdk/firmadores/php/prueba.php
	python3 -m unittest discover -s sdk/firmadores/python

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o dist/prisma$(shell go env GOEXE) ./cmd/prisma

build-todo:
	@for p in $(PLATAFORMAS); do \
		os=$${p%/*}; arch=$${p#*/}; ext=; [ $$os = windows ] && ext=.exe; \
		echo "==> $$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" \
			-o dist/prisma-$$os-$$arch$$ext ./cmd/prisma || exit 1; \
	done

# Paquetes .deb (amd64 y arm64) a partir de los binarios de build-todo. VERSION debe empezar con
# un dígito: make deb VERSION=1.2.3
deb:
	@for arq in amd64 arm64; do \
		sed -e "s/\$${ARQ}/$$arq/g" -e "s/\$${VERSION}/$(VERSION)/g" empaquetado/linux/nfpm.yaml > dist/nfpm-$$arq.yaml; \
		go run $(NFPM) package -f dist/nfpm-$$arq.yaml -p deb -t dist/ || exit 1; \
	done

# Paquete .pkg de macOS (solo en macOS). Firma y notarización opcionales: ver empaquetado/macos.
pkg:
	empaquetado/macos/construir-pkg.sh $(VERSION)

# checksums.txt de todo lo que hay en dist/ (la firma la agrega el CI, ver docs/publicar.md).
checksums:
	cd dist && sha256sum prisma-* prisma_*.deb PRISMA-* 2>/dev/null > checksums.txt; cat checksums.txt

lint:
	go vet ./...
	go run $(STATICCHECK) ./...

limpiar:
	rm -rf dist
