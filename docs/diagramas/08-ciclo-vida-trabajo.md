# Ciclo de vida de un trabajo

¿Por qué estados pasa un trabajo y cuándo se libera su `id`? Para quien soporta o integra y lee el historial de *Trabajos*.

```mermaid
stateDiagram-v2
    [*] --> encolado: documento aceptado, id reservado
    encolado --> imprimiendo: solo con prisma imprimir (CLI)
    encolado --> completado: la impresora lo recibió
    imprimiendo --> completado
    encolado --> fallido: error de la impresora o cola cerrada
    imprimiendo --> fallido: error de la impresora
    encolado --> fallido: el puente se reinició (PUENTE_REINICIADO)
    imprimiendo --> fallido: el puente se reinició (PUENTE_REINICIADO)
    completado --> [*]
    fallido --> [*]: el id se libera

    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
    class encolado,imprimiendo prisma
    class completado datos
    class fallido seguridad
```

- **En 0.1.0, los trabajos de la API no pasan por `imprimiendo`.** La cola del servicio no registra `AlIniciar`, así que van directo de `encolado` a `completado` o `fallido`. Solo `prisma imprimir` (modo desarrollo) marca `imprimiendo`. Es la discrepancia D-22 del [plan](../PLAN-DOCUMENTACION.md).
- `completado` puede llevar advertencias, por ejemplo `IMPRESORA_FUERA_DE_LINEA` con Confirmar impresión: el ticket llegó y no debe reenviarse.
- Un `fallido` libera su `id`, así que el reintento de la aplicación se imprime ([24-idempotencia.md](24-idempotencia.md)).
- `PUENTE_REINICIADO` lo pone `MarcarInterrumpidos` al arrancar, en la misma transacción que libera los `id`.
- El historial se borra pasados los días de retención (30 por defecto).

**Fuentes en el código:**

- internal/almacen/trabajos.go (`Crear`, `MarcarImprimiendo`, `Finalizar`, `MarcarInterrumpidos`)
- internal/servidor/api.go (`finalizar`)
- cmd/prisma/ejecutar.go
- cmd/prisma/imprimir.go

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
