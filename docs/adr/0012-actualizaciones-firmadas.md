# 0012. Actualizaciones firmadas con una llave Ed25519 del proyecto

- **Estado:** aceptada, vigente en PRISMA 0.1.0
- **Fecha:** 2026-10-20

## Contexto

`prisma actualizar` reemplaza el programa que controla las impresoras de un negocio. Descargar de GitHub por HTTPS no basta: una cuenta o un CI comprometidos podrían publicar un binario malicioso.

## Decisión

Cada versión publica `checksums.txt` con el SHA-256 de cada archivo y su firma Ed25519 (`checksums.txt.sig`). La llave pública va embebida en el binario (`-ldflags`) y en el instalador de Linux; la privada solo vive en el secreto `PRISMA_LLAVE_RELEASES` del CI. `prisma actualizar` verifica la firma y el SHA-256 antes de tocar nada, y nunca se ejecuta solo: el panel solo avisa.

## Consecuencias

- Una versión sin firma válida no se instala.
- Perder la llave privada impide actualizar los puentes instalados; filtrarla permite publicar actualizaciones falsas. Ver [seguridad/politicas.md](../seguridad/politicas.md#llave-de-las-versiones).

## Evidencia en el código

`internal/actualizacion/actualizacion.go`, `cmd/prisma/actualizar.go`, `empaquetado/llave-releases.pub`, `.github/workflows/release.yml` (job `publicar`).
