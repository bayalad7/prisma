# CORS y red local

¿Por qué el navegador a veces no ve la respuesta del puente?

```mermaid
sequenceDiagram
    participant Pg as Página https://caja.example
    participant Nav as Navegador
    participant P as Puente

    Pg->>Nav: fetch POST /v1/imprimir
    Nav->>P: OPTIONS (preflight), Origin y, si la pide,<br/>Access-Control-Request-Private-Network
    alt origen aprobado
        P-->>Nav: 204 con Allow-Origin exacto, métodos,<br/>cabeceras, Max-Age 600 y Allow-Private-Network
        Nav->>P: POST con Origin y Authorization
        P-->>Nav: respuesta con Allow-Origin exacto y Vary: Origin
        Nav-->>Pg: la página lee la respuesta
    else origen desconocido
        P-->>Nav: 403 sin cabeceras CORS
        Note over P: el origen queda como solicitud pendiente
        Nav-->>Pg: TypeError: Failed to fetch
        Pg->>Nav: el SDK prueba /v1/salud sin CORS
        Nav->>P: GET /v1/salud (no-cors)
        P-->>Nav: 200 (opaca)
        Nav-->>Pg: el puente responde: ORIGEN_NO_AUTORIZADO
    end
```

Un 429 del límite por IP también lleva las cabeceras CORS, para que la página vea `DEMASIADAS_PETICIONES` y no un error de red.

**Basado en:**

- internal/servidor/servidor.go (`preflight`, `permitirCORS`, `limitarIP`)
- internal/servidor/api.go (`salud`)
- sdk/js/src/prisma.mjs (`_diagnosticarNavegador`)
