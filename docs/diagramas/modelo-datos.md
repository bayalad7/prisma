# Modelo de datos

¿Qué guarda la base y cómo se relacionan las tablas?

```mermaid
erDiagram
    impresoras {
        INTEGER id PK
        TEXT alias UK
        TEXT tipo
        TEXT destino
        INTEGER ancho_mm
        TEXT pagina_codigos
        INTEGER confirmar_impresion
        INTEGER activa
    }
    clientes {
        TEXT id PK
        TEXT nombre
        INTEGER nivel
        TEXT llave_hash
        TEXT llave_prefijo UK
        BLOB llave_publica
        INTEGER exigir_llave
        TEXT operaciones_sensibles
        INTEGER todas_las_impresoras
        INTEGER activo
    }
    clientes_impresoras {
        TEXT cliente_id PK,FK
        INTEGER impresora_id PK,FK
    }
    clientes_origenes {
        TEXT cliente_id PK,FK
        TEXT origen PK
    }
    solicitudes_origen {
        TEXT origen PK
        INTEGER intentos
        TEXT estado
        INTEGER ultima_vez
    }
    trabajos {
        TEXT id PK
        TEXT documento_id
        TEXT cliente_id
        TEXT impresora_alias
        TEXT estado
        TEXT codigo_error
        TEXT advertencias
        INTEGER duracion_ms
    }
    ids_recientes {
        TEXT cliente_id PK
        TEXT documento_id PK
        BLOB hash_contenido
        TEXT trabajo_id
        INTEGER expira
    }
    recursos {
        TEXT nombre PK
        BLOB png
    }
    ajustes {
        TEXT clave PK
        TEXT valor
    }
    clientes ||--o{ clientes_origenes : "orígenes aprobados"
    clientes ||--o{ clientes_impresoras : "impresoras permitidas"
    impresoras ||--o{ clientes_impresoras : ""
    clientes ||..o{ trabajos : "por id, sin clave foránea"
    trabajos ||..o| ids_recientes : "por trabajo_id" 
```

Solo se muestran las columnas principales. `trabajos` e `ids_recientes` no tienen claves foráneas a propósito: el historial se conserva aunque se borre la aplicación o la impresora. `trabajos` **nunca** guarda el contenido del ticket.

**Basado en:**

- internal/almacen/migraciones/0001_inicial.sql
- internal/almacen/migraciones/0002_llaves_argon2id.sql
- internal/almacen/migraciones/0003_nivel2_sin_llave.sql
- internal/almacen/migraciones/0004_confirmar_impresion.sql
