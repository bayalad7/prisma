# Modelo de datos

¿Qué tablas tiene `prisma.db`, con qué llaves y cómo se relacionan? Para desarrolladores y para quien revisa qué datos guarda el puente.

```mermaid
erDiagram
    impresoras {
        INTEGER id PK
        TEXT alias UK
        TEXT nombre
        TEXT tipo
        TEXT destino
        INTEGER ancho_mm
        TEXT pagina_codigos
        INTEGER confirmar_impresion
        INTEGER activa
        TEXT simulacion
        INTEGER tickets_en_memoria
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
        TEXT cliente_id PK, FK
        INTEGER impresora_id PK, FK
    }
    clientes_origenes {
        TEXT cliente_id PK, FK
        TEXT origen PK
    }
    solicitudes_origen {
        TEXT origen PK
        INTEGER primera_vez
        INTEGER ultima_vez
        INTEGER intentos
        TEXT estado
    }
    trabajos {
        TEXT id PK
        TEXT documento_id
        TEXT cliente_id
        TEXT impresora_alias
        INTEGER virtual
        TEXT referencia
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
        INTEGER ancho_px
        INTEGER alto_px
    }
    ajustes {
        TEXT clave PK
        TEXT valor
    }
    clientes ||--o{ clientes_origenes : "orígenes aprobados"
    clientes ||--o{ clientes_impresoras : "impresoras permitidas"
    impresoras ||--o{ clientes_impresoras : "permitida a"
    clientes ||..o{ trabajos : "por cliente_id, sin FK"
    trabajos ||..o| ids_recientes : "por trabajo_id, sin FK"

```

- Las líneas continuas son claves foráneas reales (con borrado en cascada); las punteadas, referencias por valor **sin** clave foránea, a propósito: el historial se conserva aunque se borre la aplicación o la impresora, y `trabajos` guarda el alias, no el id de la impresora.
- `trabajos` **nunca** guarda el contenido del ticket. `ids_recientes` guarda solo el hash del contenido, 10 minutos.
- `ajustes` es clave-valor: contraseña del panel (hash), retención, límites, nombre del puente, acceso, código inicial (hash).
- Se omiten columnas secundarias (`creado`, `actualizado`, columnas y puntos del perfil). Los diagramas ER no admiten `classDef` en el render de GitHub, así que este no lleva colores.

**Fuentes en el código:**

- internal/almacen/migraciones/0001_inicial.sql
- internal/almacen/migraciones/0002_llaves_argon2id.sql
- internal/almacen/migraciones/0003_nivel2_sin_llave.sql
- internal/almacen/migraciones/0004_confirmar_impresion.sql
- internal/almacen/migraciones/0005_impresora_virtual.sql
- internal/almacen/ajustes.go

**Última verificación:** PRISMA 0.2.0 en desarrollo (rama `develop`), 22 de octubre de 2026.
