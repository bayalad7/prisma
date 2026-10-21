# Esquemas del contrato

| Archivo | Contenido |
|---|---|
| [`documento-v1.schema.json`](documento-v1.schema.json) | JSON Schema (draft 2020-12) del documento v1. Es público: los integradores pueden validar con él antes de enviar. |
| [`schema.go`](schema.go) | Embebe el esquema en el binario para que `internal/contrato` lo use sin leer archivos. |

El esquema cubre lo que se puede expresar en JSON Schema. Las reglas que dependen del documento completo o de la impresora (dígitos verificadores, porcentajes de las filas, PNG válidos, firma) las aplica el puente. La lista completa está en [docs/contrato-documento-v1.md §9](../docs/contrato-documento-v1.md#9-validación-y-errores).

Cada versión del contrato tiene su propio esquema, que no cambia una vez publicado.
