# Confirmar impresión

¿Cómo confirma el puente que el ticket llegó a la impresora?

```mermaid
sequenceDiagram
    participant Q as cola
    participant K as conector red o serie
    participant I as Impresora

    Q->>K: Enviar(bytes)
    K->>I: ticket completo
    K->>I: DLE EOT 1 (10 04 01)
    Note over I: Comando de tiempo real:<br/>responde al recibirlo, sin imprimir
    alt responde en línea (0x12)
        I-->>K: estado
        K-->>Q: ok
        Note over Q: trabajo completado
    else responde fuera de línea (bit 3)
        I-->>K: estado
        K-->>Q: Aviso IMPRESORA_FUERA_DE_LINEA
        Note over Q: trabajo completado con advertencia:<br/>el ticket saldrá al resolverlo
    else no responde en 10 s, o cierra la conexión
        K-->>Q: TIEMPO_AGOTADO
        Note over Q: trabajo fallido, id liberado
    end
```

Solo en impresoras de red y Bluetooth con la opción activada. Ver [instalacion/impresoras.md](../instalacion/impresoras.md#confirmar-impresión) y [ADR 0009](../adr/0009-trabajo-recibido-no-falla.md).

**Basado en:**

- internal/conector/red.go (`confirmar`)
- internal/conector/serie.go (`confirmar`)
- internal/conector/conector.go (`consultaEstado`, `interpretarEstado`, `Aviso`, `TiempoConfirmacion`)
- internal/cola/cola.go (`Resultado.Aviso`)
- internal/dev/virtual.go (`EstadoEnLinea`)
