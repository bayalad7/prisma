# 0007. Llaves `pk_` con argon2id, prefijo público y caché

- **Estado:** aceptada, vigente en PRISMA 0.1.0
- **Fecha:** 2026-10-15

## Contexto

Las llaves de nivel 2 son credenciales: si la base se filtra, no deben poder usarse. Pero calcular un hash lento en cada petición abre una vía de denegación de servicio.

## Decisión

Se guarda el hash argon2id de cada llave (19 MiB, 2 pasadas) y un prefijo público para encontrarla. Una llave con un prefijo desconocido se rechaza sin calcular nada; el resultado, válido o no, queda en caché 10 minutos, y se calculan como máximo 2 a la vez. El límite por IP se aplica antes que todo.

## Consecuencias

- La llave se muestra una sola vez; si se pierde, se regenera.
- Solo la primera petición con una llave paga el cálculo.
- La contraseña del panel usa argon2id con parámetros más altos (64 MiB, 3 pasadas), porque se verifica pocas veces.

## Evidencia en el código

`internal/seguridad/llaves.go`, `internal/seguridad/autorizacion.go` (`turnos`, `cache`), `internal/almacen/migraciones/0002_llaves_argon2id.sql`.
