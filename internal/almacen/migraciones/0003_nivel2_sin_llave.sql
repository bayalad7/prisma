-- prisma:sin-claves-foraneas
-- Un cliente de nivel 2 puede quedarse sin llave: así se revoca desde el panel. Sin llave no
-- puede autenticarse (la API responde LLAVE_REQUERIDA) hasta que se le genere una nueva.

CREATE TABLE clientes_nueva (
    id                    TEXT    PRIMARY KEY CHECK (id LIKE 'cli\_%' ESCAPE '\'),
    nombre                TEXT    NOT NULL CHECK (nombre <> ''),
    nivel                 INTEGER NOT NULL DEFAULT 2 CHECK (nivel IN (1, 2, 3)),
    llave_hash            TEXT    CHECK (llave_hash IS NULL OR llave_hash LIKE '$argon2id$%'),
    llave_prefijo         TEXT    UNIQUE,
    llave_publica         BLOB    CHECK (llave_publica IS NULL OR length(llave_publica) = 32),
    exigir_llave          INTEGER NOT NULL DEFAULT 0 CHECK (exigir_llave IN (0, 1)),
    operaciones_sensibles TEXT    NOT NULL DEFAULT '[]' CHECK (json_valid(operaciones_sensibles) AND json_type(operaciones_sensibles) = 'array'),
    todas_las_impresoras  INTEGER NOT NULL DEFAULT 1 CHECK (todas_las_impresoras IN (0, 1)),
    activo                INTEGER NOT NULL DEFAULT 1 CHECK (activo IN (0, 1)),
    creado                INTEGER NOT NULL,
    actualizado           INTEGER NOT NULL,
    CHECK ((llave_hash IS NULL) = (llave_prefijo IS NULL)),
    CHECK (nivel <> 3 OR llave_publica IS NOT NULL),
    CHECK (exigir_llave = 0 OR llave_hash IS NOT NULL)
) STRICT;

INSERT INTO clientes_nueva SELECT id, nombre, nivel, llave_hash, llave_prefijo, llave_publica, exigir_llave,
    operaciones_sensibles, todas_las_impresoras, activo, creado, actualizado FROM clientes;

DROP TABLE clientes;
ALTER TABLE clientes_nueva RENAME TO clientes;
