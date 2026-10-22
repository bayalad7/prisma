-- prisma:sin-claves-foraneas
-- Las llaves pk_ pasan de SHA-256 (32 bytes) a argon2id en formato PHC, y el nivel 3 puede exigir
-- además la llave de nivel 2 (exigir_llave). SQLite no permite cambiar el tipo ni los CHECK de una
-- columna: se reconstruye la tabla con el procedimiento oficial (claves foráneas desactivadas
-- durante la migración y comprobadas al final).

CREATE TABLE clientes_nueva (
    id                    TEXT    PRIMARY KEY CHECK (id LIKE 'cli\_%' ESCAPE '\'),
    nombre                TEXT    NOT NULL CHECK (nombre <> ''),
    nivel                 INTEGER NOT NULL DEFAULT 2 CHECK (nivel IN (1, 2, 3)),
    -- Hash argon2id (PHC) de la llave pk_. La llave completa nunca se guarda.
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
    CHECK (nivel <> 2 OR llave_hash IS NOT NULL),
    CHECK (nivel <> 3 OR llave_publica IS NOT NULL),
    CHECK (exigir_llave = 0 OR llave_hash IS NOT NULL)
) STRICT;

-- Un hash SHA-256 no se puede convertir a argon2id. Los clientes que tenían llave la pierden;
-- los de nivel 2 quedan desactivados (y en nivel 1 para cumplir las restricciones) hasta que se
-- les genere una llave nueva. Al momento de esta migración aún no existía forma de crear clientes.
INSERT INTO clientes_nueva (id, nombre, nivel, llave_hash, llave_prefijo, llave_publica, exigir_llave,
                            operaciones_sensibles, todas_las_impresoras, activo, creado, actualizado)
SELECT id, nombre,
       CASE WHEN nivel = 2 THEN 1 ELSE nivel END,
       NULL, NULL, llave_publica, 0,
       operaciones_sensibles, todas_las_impresoras,
       CASE WHEN nivel = 2 THEN 0 ELSE activo END,
       creado, actualizado
FROM clientes;

DROP TABLE clientes;
ALTER TABLE clientes_nueva RENAME TO clientes;
