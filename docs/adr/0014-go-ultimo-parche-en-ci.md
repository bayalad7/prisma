# 0014. CI y publicación con el último parche de Go 1.26

- **Estado:** aceptada, vigente en PRISMA 0.1.0
- **Fecha:** 2026-10-22

## Contexto

`go.mod` declara `go 1.26.0` como mínimo para compilar. El CI usaba `go-version-file: go.mod`, que instala exactamente esa versión: la publicación de 0.1.0 falló porque nfpm v2.48.0 exige Go 1.26.4, y los binarios se habrían compilado sin los parches de seguridad de la biblioteca estándar.

## Decisión

El CI y el workflow de publicación instalan `go-version: '1.26.x'` con `check-latest: true`: el parche más reciente de la serie 1.26. `go.mod` sigue declarando 1.26.0 como mínimo.

## Consecuencias

- Los binarios publicados llevan los parches de seguridad vigentes al publicar.
- Compilar localmente sigue funcionando con 1.26.0; empaquetar el `.deb` requiere 1.26.4 o superior.
- Pasar a Go 1.27 es una decisión aparte, con su propio ADR.

## Evidencia en el código

`.github/workflows/ci.yml`, `.github/workflows/release.yml`, `go.mod`, `Makefile` (`NFPM`).
