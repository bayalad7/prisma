---
titulo: "Confirmar impresión"
descripcion: "Cómo confirma el puente que el ticket llegó a la impresora"
audiencia: [instalador, integrador]
pantallas: [impresoras]
secciones_manual: [administracion/impresoras]
---
# Confirmar impresión

¿Cómo confirma el puente que el ticket llegó a una impresora de red o Bluetooth? Para instaladores que deciden si activarla e integradores que reciben su advertencia.

```mermaid
sequenceDiagram
    box rgb(230,244,234) PRISMA
        participant cola as internal/cola
        participant con as internal/conector
    end
    box rgb(232,240,254) impresora
        participant imp as impresora de red o serie
    end

    cola->>con: Enviar(bytes)
    con->>imp: ticket completo
    con->>imp: DLE EOT 1 (10 04 01)
    Note over imp: comando de tiempo real: responde al recibirlo, sin imprimir
    alt responde en línea (0x12)
        imp-->>con: estado
        con-->>cola: ok, trabajo completado
    else responde fuera de línea (bit 3)
        imp-->>con: estado
        con-->>cola: Aviso IMPRESORA_FUERA_DE_LINEA, trabajo completado
    else no responde en 10 s o cierra la conexión
        con-->>cola: TIEMPO_AGOTADO, trabajo fallido e id liberado
    end
```

- Como la consulta viaja detrás del ticket, su respuesta confirma que el ticket completo llegó.
- «Fuera de línea» no es un fallo: la impresora ya tiene el ticket y lo imprimirá al resolverse; un reintento lo imprimiría dos veces ([ADR 0009](../adr/0009-trabajo-recibido-no-falla.md)).
- Está desactivada por defecto: algunas impresoras no responden a `DLE EOT`.
- La impresora virtual (`prisma dev impresora-virtual`) responde como una impresora en línea.

**Fuentes en el código:**

- internal/conector/conector.go (`consultaEstado`, `interpretarEstado`, `Aviso`, `TiempoConfirmacion`)
- internal/conector/red.go (`confirmar`)
- internal/conector/serie.go (`confirmar`)
- internal/cola/cola.go (`Resultado.Aviso`)
- internal/dev/virtual.go (`EstadoEnLinea`)

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
