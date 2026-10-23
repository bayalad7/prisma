# 0009. Un trabajo que llegó a la impresora no falla

- **Estado:** aceptada, vigente en PRISMA 0.1.0
- **Fecha:** 2026-10-21

## Contexto

Con *Confirmar impresión*, la impresora responde a `DLE EOT 1` después de cada ticket. Puede responder «fuera de línea» (tapa abierta, sin papel): el ticket ya está en su búfer y saldrá al resolverlo. Si el trabajo fallara, la aplicación lo reintentaría y saldría dos veces.

## Decisión

Si la impresora responde «fuera de línea», el conector devuelve un `Aviso`, no un error: el trabajo se completa con la advertencia `IMPRESORA_FUERA_DE_LINEA`. Solo falla (`TIEMPO_AGOTADO`) cuando la impresora no responde, porque entonces no hay evidencia de que lo recibió.

## Consecuencias

- Es el complemento de [ADR 0008](0008-idempotencia-persistente.md): fallar solo lo que no salió, completar lo que sí.
- La aplicación debe mostrar la advertencia a la persona y **no** reenviar.
- Confirmar impresión está desactivada por defecto, porque algunas impresoras no responden a `DLE EOT`.

## Evidencia en el código

`internal/conector/conector.go` (`Aviso`, `interpretarEstado`), `internal/conector/red.go` (`confirmar`), `internal/cola/cola.go` (`Resultado.Aviso`), CLAUDE.md «Invariantes».
