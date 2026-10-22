# 0004. Alias de impresora en lugar de direcciones

- **Estado:** aceptada, vigente en PRISMA 0.1.0
- **Fecha:** 2026-10-14

## Contexto

Las aplicaciones de terceros no deberían conocer la red del negocio: si la impresora cambia de IP, de puerto o de modelo, ninguna aplicación debería cambiar.

## Decisión

Las aplicaciones solo nombran la impresora por su **alias** (`caja`, `cocina`). El destino real (IP y puerto, nombre de Windows, cola de CUPS, puerto serie) es configuración privada del puente.

## Consecuencias

- Cambiar una impresora no toca a las aplicaciones.
- `/v1/impresoras` nunca expone direcciones.
- Renombrar un alias rompe a las aplicaciones que lo usan (`IMPRESORA_NO_ENCONTRADA`).

## Evidencia en el código

`internal/servidor/api.go` (`impresoraVisible`), `internal/almacen/impresoras.go`.
