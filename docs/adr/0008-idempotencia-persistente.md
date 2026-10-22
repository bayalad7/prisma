# 0008. Idempotencia persistente, con liberación del `id` de lo que no se imprimió

- **Estado:** aceptada, vigente en PRISMA 0.1.0
- **Fecha:** 2026-10-15

## Contexto

La red de un negocio falla: una aplicación que no recibe respuesta reintenta, y el ticket no debe salir dos veces. Pero tampoco debe perderse uno que nunca salió (por ejemplo, una comanda de cocina).

## Decisión

Cada documento lleva un `id` (UUID v4). El puente recuerda cada `id`, con el hash de su contenido, durante 10 minutos, en la base (sobrevive a los reinicios):
- mismo `id` y mismo contenido: responde el trabajo original con `duplicado: true`, sin imprimir;
- mismo `id` con otro contenido: `ID_REUTILIZADO`;
- si el trabajo **no se imprimió** (falló o lo interrumpió un reinicio), el `id` se libera y el reintento se imprime.

## Consecuencias

- Los clientes pueden reintentar sin miedo; el SDK lo hace solo con el mismo `id`.
- Nunca se responde `ok: true` a un reintento de algo que no salió (invariante de CLAUDE.md; hallazgos C2 y M5 de la verificación).

## Evidencia en el código

`internal/almacen/ids_recientes.go`, `internal/almacen/trabajos.go` (`MarcarInterrumpidos`), `internal/servidor/api.go` (`responderSiRepetido`, `liberarSiFallo`, `finalizar`).
