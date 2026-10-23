# 0006. Firma Ed25519 sobre la forma canónica JCS en el nivel 3

- **Estado:** aceptada, vigente en PRISMA 0.1.0
- **Fecha:** 2026-10-14

## Contexto

En el nivel 3, el backend del integrador debe poder garantizar que un documento no se alteró ni se repitió, desde cualquier lenguaje, y sin que el puente guarde secretos que puedan filtrarse.

## Decisión

El backend firma con Ed25519 los bytes UTF-8 de la forma canónica JCS (RFC 8785) del documento sin el campo `firma`. El puente solo guarda la llave **pública**. El documento lleva `cliente_id`, `emitido` y `expira` (vigencia de 0 a 120 s, 30 s de tolerancia de reloj).

## Consecuencias

- Una base filtrada no permite falsificar documentos.
- JCS evita las diferencias de serialización entre lenguajes; hay firmadores de referencia en PHP, Node.js y Python, probados contra los mismos casos que el puente, y un vector público.
- El reloj del backend debe estar sincronizado (NTP).

## Evidencia en el código

`internal/seguridad/firma.go` (`github.com/gowebpki/jcs`), `sdk/firmadores/`, `sdk/firmadores/casos-jcs.json`, `testdata/vector-firma/`.
