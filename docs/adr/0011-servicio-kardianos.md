# 0011. Servicio del sistema con kardianos/service y plantillas propias

- **Estado:** aceptada, vigente en PRISMA 0.1.0
- **Fecha:** 2026-10-20

## Contexto

El puente debe arrancar con el equipo, sin sesión iniciada, reiniciarse si falla y correr con los menores privilegios posibles en tres sistemas con gestores distintos.

## Decisión

Se usa `github.com/kardianos/service` con plantillas propias de systemd y launchd: Windows corre como LocalSystem (inicio automático, reinicio a los 5 s); Linux, como el usuario `prisma` (grupos `lp` y `dialout`, `ProtectSystem`, `NoNewPrivileges`); macOS, como el usuario oculto `_prisma`. La unidad del paquete `.deb` es idéntica a la de la plantilla, y una prueba lo verifica.

## Consecuencias

- `prisma servicio instalar|iniciar|detener|reiniciar|estado|desinstalar` funciona igual en los tres sistemas.
- En macOS, kardianos consulta el estado en el dominio del usuario; el puente lo corrige consultando el dominio del sistema (hallazgo B9).

## Evidencia en el código

`internal/servicio/servicio.go`, `internal/servicio/plantillas.go`, `internal/servicio/sistema_*.go`, `empaquetado/linux/`.
