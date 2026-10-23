# 0002. SQLite en Go puro y `config.toml` solo para el arranque

- **Estado:** aceptada, vigente en PRISMA 0.1.0
- **Fecha:** 2026-10-14

## Contexto

El puente guarda impresoras, aplicaciones, historial y ajustes, y el dueño los administra desde el panel. Hace falta una base local, transaccional, de un solo archivo y sin servidor, que funcione en las 6 plataformas sin CGO.

## Decisión

- Base de datos: SQLite con `modernc.org/sqlite` (Go puro), en modo WAL, con claves foráneas, tablas `STRICT`, `busy_timeout(5000)` y una sola conexión de escritura. Migraciones embebidas en el binario, numeradas, que nunca se editan una vez publicadas.
- `config.toml` contiene solo lo necesario para arrancar: puerto, dirección, modo TLS, rutas y límites iniciales. Todo lo demás vive en la base y se cambia en el panel.

## Consecuencias

- Respaldar es copiar un archivo; el panel lo hace en caliente.
- Antes de migrar a un esquema nuevo se guarda una copia automática.
- Otro programa que abra `prisma.db` puede bloquearla: el puente responde `PUENTE_OCUPADO` en lugar de un error genérico.
- Un cambio de `config.toml` requiere reiniciar; un cambio en el panel, no.

## Evidencia en el código

`internal/almacen/almacen.go` (`pragmas`, `SetMaxOpenConns(1)`), `internal/almacen/migraciones/*.sql`, `internal/almacen/respaldo.go`, `internal/config/config.go`.
