# Flujo de impresión

¿Por qué pasos viaja un documento desde que llega hasta la impresora? Vista general para integradores y desarrolladores; el orden exacto de los rechazos está en [07-orden-autorizacion.md](07-orden-autorizacion.md).

```mermaid
flowchart LR
    doc["documento JSON"]:::externo --> ip["límite por IP"]:::seguridad
    ip --> ident["identificar cliente<br/>internal/seguridad"]:::seguridad
    ident --> sobre["leer sobre<br/>internal/contrato"]:::prisma
    sobre --> firma["verificar firma<br/>internal/seguridad"]:::seguridad
    firma --> rep["¿id repetido?<br/>internal/almacen"]:::datos
    rep --> perm["permisos<br/>internal/seguridad"]:::seguridad
    perm --> val["validar esquema y reglas<br/>internal/contrato"]:::prisma
    val --> render["renderizar ESC/POS<br/>internal/escpos"]:::prisma
    render --> reserva["reservar id y crear trabajo<br/>internal/almacen"]:::datos
    reserva --> cola["encolar<br/>internal/cola"]:::prisma
    cola --> con["enviar<br/>internal/conector"]:::prisma
    con --> imp["impresora"]:::externo
    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
```

- Lo costoso (validar el esquema completo, decodificar imágenes, renderizar) ocurre **después** de identificar al cliente y comprobar sus permisos.
- *Leer sobre* extrae solo lo necesario para autorizar: `id`, `impresora`, versión, campos de firma y el `tipo` de cada operación.
- El `id` se consulta antes de validar (para responder rápido un duplicado), pero se **reserva** recién después de renderizar: un documento inválido nunca ocupa un `id`.
- Si el envío falla, el trabajo queda `fallido` y su `id` se libera ([24-idempotencia.md](24-idempotencia.md)).
- La respuesta llega cuando la impresora terminó, o con 202 a los 60 s si tenía trabajos por delante.

**Fuentes en el código:**

- internal/servidor/api.go (`autorizarDocumento`, `imprimir`, `finalizar`)
- internal/servidor/servidor.go (`limitarIP`)
- internal/seguridad/autorizacion.go
- internal/contrato/validar.go
- internal/escpos/renderizar.go
- internal/almacen/ids_recientes.go
- internal/cola/cola.go
- internal/conector/conector.go

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
