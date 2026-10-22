# Idempotencia

¿Qué responde el puente si llega el mismo `id`? Para integradores que implementan reintentos.

```mermaid
flowchart TD
    l["llega un documento con id"]:::externo --> v{"¿el mismo cliente envió ese id<br/>en los últimos 10 min?"}:::datos
    v -- no --> n["procesar como nuevo:<br/>reservar el id"]:::prisma
    v -- sí --> h{"¿mismo contenido? (hash JCS)"}:::datos
    h -- no --> r409(["409 ID_REUTILIZADO"]):::externo
    h -- sí --> f{"¿el trabajo original falló?"}:::datos
    f -- sí --> lib["liberar el id"]:::datos
    lib --> n
    f -- no --> dup(["200 duplicado: true,<br/>sin volver a imprimir"]):::externo
    n --> i["imprimir"]:::prisma
    i -- "falla o reinicio del puente" --> lib2["liberar el id"]:::datos
    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
```

- Los `id` viven en `ids_recientes`, en la base: la ventana de 10 minutos sobrevive a los reinicios.
- Nunca se responde `ok: true` a un reintento de algo que no salió (invariante de CLAUDE.md).
- El SDK reintenta los errores de red con el **mismo** `id`; una aplicación propia debe hacer lo mismo.

**Fuentes en el código:**

- internal/almacen/ids_recientes.go (`Reservar`, `Liberar`, `VentanaIdempotencia`)
- internal/servidor/api.go (`responderSiRepetido`, `liberarSiFallo`, `finalizar`)
- internal/almacen/trabajos.go (`MarcarInterrumpidos`)

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
