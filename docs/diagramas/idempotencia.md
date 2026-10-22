# Idempotencia

¿Qué responde el puente si llega el mismo `id`?

```mermaid
flowchart TD
    L["Llega un documento con id"] --> V{"¿El mismo cliente envió<br/>ese id en los últimos 10 min?"}
    V -->|No| N["Se procesa como nuevo:<br/>se reserva el id"]
    V -->|Sí| H{"¿Mismo contenido?<br/>hash JCS"}
    H -->|No| R409["409 ID_REUTILIZADO"]
    H -->|Sí| F{"¿El trabajo original falló?"}
    F -->|Sí| LIB["Se libera el id"] --> N
    F -->|No| DUP["200 con duplicado: true<br/>y el trabajo original;<br/>no se vuelve a imprimir"]
    N --> I["Se imprime"]
    I -->|"falla o el puente se reinicia"| LIB2["El id se libera"]
```

Los `id` viven en la base: la ventana de 10 minutos sobrevive a los reinicios.

**Basado en:**

- internal/almacen/ids_recientes.go (`Reservar`, `Liberar`, `VentanaIdempotencia`)
- internal/servidor/api.go (`responderSiRepetido`, `liberarSiFallo`, `finalizar`)
- internal/almacen/trabajos.go (`MarcarInterrumpidos`)
