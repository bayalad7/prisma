# Secuencia de `POST /v1/imprimir`

¿Qué pasa, en orden, desde que llega la petición hasta la respuesta?

```mermaid
sequenceDiagram
    autonumber
    participant App as Aplicación
    participant S as servidor
    participant Seg as seguridad
    participant C as contrato
    participant A as almacen
    participant E as escpos
    participant Q as cola
    participant K as conector
    participant I as Impresora

    App->>S: POST /v1/imprimir
    S->>S: límite por IP (429)
    S->>Seg: Identificar(Origin, Authorization)
    Seg-->>S: cliente, u ORIGEN_NO_AUTORIZADO, LLAVE_*
    S->>S: límite por cliente (429)
    S->>S: leer el cuerpo, 300 KB como máximo (413)
    S->>C: LeerSobre: JSON y versión
    S->>Seg: VerificarDocumento: firma en el nivel 3
    S->>A: ¿id ya visto? (ids_recientes)
    alt mismo id y mismo contenido, trabajo no fallido
        S-->>App: 200 duplicado, sin imprimir
    else mismo id, otro contenido
        S-->>App: 409 ID_REUTILIZADO
    end
    S->>Seg: VerificarPermisos: impresora, abrir_cajon y raw
    S->>C: Decodificar: esquema y reglas
    S->>A: impresora activa (404)
    S->>E: Renderizar(documento, perfil)
    E-->>S: bytes y advertencias
    S->>A: Reservar el id y crear el trabajo (encolado)
    S->>Q: Encolar (503 COLA_LLENA)
    Q->>K: Enviar(bytes)
    K->>I: ESC/POS y, con Confirmar impresión, DLE EOT 1
    I-->>K: datos recibidos o estado
    K-->>Q: ok, Aviso o error
    Q-->>S: Resultado
    S->>A: Finalizar el trabajo, si falló, liberar el id
    alt terminó antes de 60 s
        S-->>App: 200 impreso, o error con trabajo_id
    else la impresora tenía trabajos por delante
        S-->>App: 202 encolado, el resultado queda en el historial
    end
```

Nada costoso (esquema completo, imágenes, render) ocurre antes de identificar al cliente y comprobar sus permisos.

**Basado en:**

- internal/servidor/api.go (`autorizarDocumento`, `imprimir`, `responderSiRepetido`, `liberarSiFallo`, `finalizar`)
- internal/servidor/servidor.go (`limitarIP`)
- internal/seguridad/autorizacion.go
- internal/contrato/validar.go
- internal/escpos/renderizar.go
- internal/cola/cola.go
- internal/conector/*
