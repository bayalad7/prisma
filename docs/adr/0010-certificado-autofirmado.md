# 0010. Certificado autofirmado de 825 días, con aviso y regeneración

- **Estado:** aceptada, vigente en PRISMA 0.1.0; complementada por [0020](0020-confiar-en-el-certificado-del-puente-al-instalar.md) (límite a la red local y confianza al instalar)
- **Fecha:** 2026-10-15

## Contexto

Los navegadores exigen HTTPS para que una página pública hable con un equipo de la red local, y el puente no tiene dominio público ni puede depender de una autoridad externa ([principio de independencia](../../CLAUDE.md)).

## Decisión

Por defecto, el puente genera su propio certificado: ECDSA P-256, 825 días (el máximo que aceptan macOS e iOS), con `localhost`, el nombre del equipo y sus IPs. Lo recarga en caliente al regenerarse. Avisa desde 60 días antes de vencer (bitácora y panel) y, si ya venció, genera uno nuevo al arrancar. También admite un certificado propio (`archivos`) y HTTP solo para el propio equipo (`solo_localhost_http`).

## Consecuencias

- Cada equipo que imprime debe confiar en el certificado (instalarlo una vez).
- Si cambia la IP del equipo, hay que regenerarlo y volver a instalarlo: conviene IP fija.
- La interfaz `ProveedorCertificado` deja lugar para un proveedor automático (ACME) más adelante.

## Evidencia en el código

`internal/servidor/tls.go` (`vigenciaAutofirmado`, `AvisoVencimiento`, `CargarAutofirmado`, `ProveedorCertificado`), `internal/servidor/escuchar.go`.
