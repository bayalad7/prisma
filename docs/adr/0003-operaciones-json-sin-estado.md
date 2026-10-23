# 0003. Operaciones JSON sin estado en lugar de bytes ESC/POS

- **Estado:** aceptada, vigente en PRISMA 0.1.0
- **Fecha:** 2026-10-14

## Contexto

Cada impresora térmica difiere: ancho (58 u 80 mm), columnas, página de códigos, QR y códigos de barras nativos o no, cuchilla, cajón. Si las aplicaciones enviaran bytes ESC/POS, cada una tendría que conocer cada impresora, y un documento no serviría en otra.

## Decisión

Las aplicaciones envían un documento JSON (contrato «documento v1») con operaciones: `texto`, `fila`, `par`, `qr`, `imagen`, `cortar`… El puente genera el ESC/POS con un renderizador propio según el perfil de la impresora. Cada operación trae todo su estilo y lo restablece al terminar: ninguna depende de la anterior.

## Consecuencias

- El mismo documento sirve en cualquier impresora; lo que el perfil no soporta se resuelve de otra forma (QR como imagen) o se omite con una advertencia.
- Todo se valida antes de imprimir: nunca sale un ticket a medias por un error del documento.
- La previsualización sale de los mismos bytes que recibe la impresora.
- Para lo que no cubren las operaciones existe `raw`, deshabilitada por defecto ([ADR 0005](0005-tres-niveles-de-seguridad.md)).

## Evidencia en el código

`docs/referencia/contrato-documento-v1.md`, `schema/documento-v1.schema.json`, `internal/contrato/`, `internal/escpos/renderizar.go`.
