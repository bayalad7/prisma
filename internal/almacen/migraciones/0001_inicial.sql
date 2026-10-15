-- Esquema inicial de PRISMA.
-- Fechas: INTEGER en milisegundos Unix UTC. Booleanos: INTEGER 0/1.

CREATE TABLE impresoras (
    id                           INTEGER PRIMARY KEY,
    alias                        TEXT    NOT NULL UNIQUE CHECK (length(alias) BETWEEN 1 AND 32),
    nombre                       TEXT    NOT NULL,
    tipo                         TEXT    NOT NULL CHECK (tipo IN ('red', 'usb', 'bluetooth', 'archivo')),
    destino                      TEXT    NOT NULL CHECK (destino <> ''),
    ancho_mm                     INTEGER NOT NULL CHECK (ancho_mm > 0),
    columnas                     INTEGER NOT NULL CHECK (columnas > 0),
    columnas_fuente_b            INTEGER NOT NULL CHECK (columnas_fuente_b > 0),
    ancho_px                     INTEGER NOT NULL CHECK (ancho_px > 0),
    pagina_codigos               TEXT    NOT NULL,
    comando_pagina_codigos       INTEGER NOT NULL CHECK (comando_pagina_codigos BETWEEN 0 AND 255),
    soporta_qr_nativo            INTEGER NOT NULL DEFAULT 1 CHECK (soporta_qr_nativo IN (0, 1)),
    soporta_corte                INTEGER NOT NULL DEFAULT 1 CHECK (soporta_corte IN (0, 1)),
    soporta_cajon                INTEGER NOT NULL DEFAULT 0 CHECK (soporta_cajon IN (0, 1)),
    soporta_imagen               INTEGER NOT NULL DEFAULT 1 CHECK (soporta_imagen IN (0, 1)),
    soporta_codigo_barras_nativo INTEGER NOT NULL DEFAULT 1 CHECK (soporta_codigo_barras_nativo IN (0, 1)),
    baudios                      INTEGER CHECK (baudios IS NULL OR baudios > 0),
    activa                       INTEGER NOT NULL DEFAULT 1 CHECK (activa IN (0, 1)),
    creada                       INTEGER NOT NULL,
    actualizada                  INTEGER NOT NULL
) STRICT;

CREATE TABLE clientes (
    id                    TEXT    PRIMARY KEY CHECK (id LIKE 'cli\_%' ESCAPE '\'),
    nombre                TEXT    NOT NULL CHECK (nombre <> ''),
    nivel                 INTEGER NOT NULL DEFAULT 2 CHECK (nivel IN (1, 2, 3)),
    -- Nivel 2: SHA-256 de la llave pk_ (32 bytes). La llave completa nunca se guarda.
    llave_hash            BLOB    CHECK (llave_hash IS NULL OR length(llave_hash) = 32),
    llave_prefijo         TEXT    UNIQUE,
    -- Nivel 3: llave pública Ed25519 (32 bytes).
    llave_publica         BLOB    CHECK (llave_publica IS NULL OR length(llave_publica) = 32),
    operaciones_sensibles TEXT    NOT NULL DEFAULT '[]' CHECK (json_valid(operaciones_sensibles) AND json_type(operaciones_sensibles) = 'array'),
    todas_las_impresoras  INTEGER NOT NULL DEFAULT 1 CHECK (todas_las_impresoras IN (0, 1)),
    activo                INTEGER NOT NULL DEFAULT 1 CHECK (activo IN (0, 1)),
    creado                INTEGER NOT NULL,
    actualizado           INTEGER NOT NULL,
    CHECK (nivel <> 2 OR (llave_hash IS NOT NULL AND llave_prefijo IS NOT NULL)),
    CHECK (nivel <> 3 OR llave_publica IS NOT NULL)
) STRICT;

-- Impresoras permitidas cuando clientes.todas_las_impresoras = 0.
CREATE TABLE clientes_impresoras (
    cliente_id   TEXT    NOT NULL REFERENCES clientes (id) ON DELETE CASCADE,
    impresora_id INTEGER NOT NULL REFERENCES impresoras (id) ON DELETE CASCADE,
    PRIMARY KEY (cliente_id, impresora_id)
) STRICT;
CREATE INDEX clientes_impresoras_impresora ON clientes_impresoras (impresora_id);

-- Un origen pertenece a un solo cliente: en el nivel 1 es lo único que lo identifica.
CREATE TABLE clientes_origenes (
    cliente_id TEXT NOT NULL REFERENCES clientes (id) ON DELETE CASCADE,
    origen     TEXT NOT NULL UNIQUE,
    PRIMARY KEY (cliente_id, origen)
) STRICT;

CREATE TABLE solicitudes_origen (
    origen      TEXT    PRIMARY KEY,
    primera_vez INTEGER NOT NULL,
    ultima_vez  INTEGER NOT NULL,
    intentos    INTEGER NOT NULL DEFAULT 1 CHECK (intentos > 0),
    estado      TEXT    NOT NULL DEFAULT 'pendiente' CHECK (estado IN ('pendiente', 'rechazada'))
) STRICT;

-- Sin contenido del ticket. cliente_id sin FK: el historial sobrevive al borrar un cliente.
CREATE TABLE trabajos (
    id              TEXT    PRIMARY KEY,
    documento_id    TEXT    NOT NULL,
    cliente_id      TEXT    NOT NULL,
    impresora_alias TEXT    NOT NULL,
    referencia      TEXT,
    estado          TEXT    NOT NULL CHECK (estado IN ('encolado', 'imprimiendo', 'completado', 'fallido')),
    codigo_error    TEXT,
    advertencias    TEXT    NOT NULL DEFAULT '[]' CHECK (json_valid(advertencias) AND json_type(advertencias) = 'array'),
    duracion_ms     INTEGER CHECK (duracion_ms IS NULL OR duracion_ms >= 0),
    creado          INTEGER NOT NULL,
    finalizado      INTEGER
) STRICT;
CREATE INDEX trabajos_creado ON trabajos (creado);
CREATE INDEX trabajos_cliente ON trabajos (cliente_id, creado);

-- Idempotencia y anti-repetición (sobrevive a reinicios).
CREATE TABLE ids_recientes (
    cliente_id     TEXT    NOT NULL,
    documento_id   TEXT    NOT NULL,
    hash_contenido BLOB    NOT NULL CHECK (length(hash_contenido) = 32),
    trabajo_id     TEXT    NOT NULL,
    expira         INTEGER NOT NULL,
    PRIMARY KEY (cliente_id, documento_id)
) STRICT;
CREATE INDEX ids_recientes_expira ON ids_recientes (expira);

CREATE TABLE recursos (
    nombre   TEXT    PRIMARY KEY CHECK (length(nombre) BETWEEN 1 AND 64),
    png      BLOB    NOT NULL CHECK (length(png) BETWEEN 1 AND 524288),
    ancho_px INTEGER NOT NULL CHECK (ancho_px > 0),
    alto_px  INTEGER NOT NULL CHECK (alto_px > 0),
    creado   INTEGER NOT NULL
) STRICT;

CREATE TABLE ajustes (
    clave       TEXT    PRIMARY KEY,
    valor       TEXT    NOT NULL,
    actualizado INTEGER NOT NULL
) STRICT;
