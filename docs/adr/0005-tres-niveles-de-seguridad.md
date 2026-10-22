# 0005. Tres niveles de seguridad por cliente, con el 2 por defecto

- **Estado:** aceptada, vigente en PRISMA 0.1.0
- **Fecha:** 2026-10-15

## Contexto

El puente atiende aplicaciones muy distintas: una página sin backend en una tienda pequeña y un sistema con backend que imprime comprobantes y abre el cajón. Una sola forma de autorizar sería demasiado débil para unas o demasiado complicada para otras.

## Decisión

Cada cliente tiene un nivel: **1 — origen** (el `Origin` del navegador debe estar aprobado), **2 — llave** (además, una llave `pk_`; es el nivel por defecto) o **3 — firma** (el backend firma cada documento con Ed25519, [ADR 0006](0006-firma-ed25519-jcs.md)). `abrir_cajon` y `raw` están deshabilitadas por cliente hasta que el dueño las habilita. Los orígenes desconocidos quedan como solicitudes pendientes.

## Consecuencias

- Una página sin backend imprime en minutos con el SDK (nivel 2).
- El nivel 1 no protege contra herramientas fuera del navegador; está documentado como riesgo aceptado.
- El puente nunca acepta una petición sin identificar a una aplicación autorizada por el dueño.

## Evidencia en el código

`internal/seguridad/autorizacion.go`, `internal/almacen/clientes.go` (`NivelOrigen`, `NivelLlave`, `NivelFirma`), `internal/almacen/solicitudes.go`.
