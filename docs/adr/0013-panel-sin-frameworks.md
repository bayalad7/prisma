# 0013. Panel sin frameworks ni CDN, con sesiones en memoria

- **Estado:** aceptada, vigente en PRISMA 0.1.0
- **Fecha:** 2026-10-19

## Contexto

El panel corre en el equipo del negocio, a menudo sin Internet, y controla todo el puente. Un framework o un CDN agregarían dependencias, tamaño y superficie de ataque.

## Decisión

El panel es HTML, CSS y JavaScript propios, embebidos en el binario, sin frameworks ni recursos externos, con una CSP `default-src 'self'`. Las sesiones viven en memoria (se pierden al reiniciar), con cookie `HttpOnly`, `Secure` y `SameSite=Strict`, CSRF por cabecera y acceso solo desde IPs privadas por defecto.

## Consecuencias

- Funciona sin Internet y no hay nada que actualizar aparte del binario.
- Reiniciar el puente cierra todas las sesiones.
- La API interna `/admin/api` no es pública y puede cambiar entre versiones.

## Evidencia en el código

`internal/admin/admin.go` (`cspPanel`, `redPermitida`), `internal/admin/sesiones.go`, `internal/admin/web/`.
