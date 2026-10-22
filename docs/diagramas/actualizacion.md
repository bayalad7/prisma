# Actualización

¿Cómo se actualiza el puente sin riesgo?

```mermaid
sequenceDiagram
    participant D as Dueño
    participant C as prisma actualizar
    participant G as GitHub Releases
    participant S as Servicio

    D->>C: prisma actualizar (como administrador)
    C->>G: última versión estable
    G-->>C: v0.x.y
    C->>G: checksums.txt y checksums.txt.sig
    C->>C: verifica la firma Ed25519 con la llave pública embebida
    alt la firma no verifica
        C-->>D: error, no se toca nada
    end
    C->>G: binario de esta plataforma
    C->>C: verifica el SHA-256 contra checksums.txt
    C->>S: detener (si estaba en marcha)
    C->>C: reemplazar el binario
    C->>S: iniciar
    Note over S: al arrancar, si hay migraciones,<br/>respalda la base antes de migrarla
```

El panel solo avisa de la versión nueva; nunca instala nada solo. Ver [ADR 0012](../adr/0012-actualizaciones-firmadas.md).

**Basado en:**

- internal/actualizacion/actualizacion.go
- internal/actualizacion/reemplazar.go
- cmd/prisma/actualizar.go (`actualizarBinario`)
- internal/almacen/respaldo.go (`RespaldarSiHayMigraciones`)
- empaquetado/llave-releases.pub
