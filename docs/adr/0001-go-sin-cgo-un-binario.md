# 0001. Go sin CGO, un binario por plataforma

- **Estado:** aceptada, vigente en PRISMA 0.1.0
- **Fecha:** 2026-10-14

## Contexto

El puente se instala en equipos de negocios: PC con Windows (64 y 32 bits), Raspberry Pi y Mac, a menudo sin nadie técnico cerca. Cualquier dependencia del sistema (un runtime, una biblioteca de C, un compilador) es un punto de falla en la instalación.

## Decisión

Se escribe en Go 1.26+, sin CGO, y se publica un solo binario por plataforma: Windows amd64 y 386, Linux amd64 y arm64, macOS amd64 y arm64. Toda la funcionalidad (HTTPS, SQLite, panel, ESC/POS) va dentro del binario.

## Consecuencias

- La instalación es copiar un archivo; los instaladores solo agregan el servicio y las carpetas.
- La compilación cruzada sale de un solo equipo (`make build-todo`).
- Se descartan bibliotecas que requieren CGO, como el SQLite oficial ([ADR 0002](0002-sqlite-en-go-puro.md)).
- `-race` necesita CGO: solo se habilita en las pruebas del CI, nunca en los binarios publicados.

## Evidencia en el código

`go.mod` (`go 1.26.0`); `Makefile` y `build.ps1` (`CGO_ENABLED=0`, `build-todo`); `.github/workflows/release.yml`.
