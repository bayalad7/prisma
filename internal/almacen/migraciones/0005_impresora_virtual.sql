-- prisma:sin-claves-foraneas
-- Impresora virtual: vive dentro del puente, no imprime en papel y guarda en memoria los últimos
-- tickets para mostrarlos en el panel (nunca en esta base). Puede simular fallas para que una
-- aplicación pruebe cómo las maneja. SQLite no permite cambiar el CHECK de tipo: se reconstruye la
-- tabla conservando los id (clientes_impresoras los referencia).

CREATE TABLE impresoras_nueva (
    id                           INTEGER PRIMARY KEY,
    alias                        TEXT    NOT NULL UNIQUE CHECK (length(alias) BETWEEN 1 AND 32),
    nombre                       TEXT    NOT NULL,
    tipo                         TEXT    NOT NULL CHECK (tipo IN ('red', 'usb', 'bluetooth', 'archivo', 'virtual')),
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
    actualizada                  INTEGER NOT NULL,
    confirmar_impresion          INTEGER NOT NULL DEFAULT 0 CHECK (confirmar_impresion IN (0, 1)),
    -- Solo impresoras virtuales: qué falla simulan y cuántos tickets recuerdan.
    simulacion                   TEXT    NOT NULL DEFAULT 'normal' CHECK (simulacion IN ('normal', 'fuera_de_linea', 'lenta', 'falla_aleatoria')),
    simulacion_retraso_ms        INTEGER NOT NULL DEFAULT 3000 CHECK (simulacion_retraso_ms BETWEEN 0 AND 30000),
    simulacion_falla_pct         INTEGER NOT NULL DEFAULT 30 CHECK (simulacion_falla_pct BETWEEN 0 AND 100),
    tickets_en_memoria           INTEGER NOT NULL DEFAULT 20 CHECK (tickets_en_memoria BETWEEN 1 AND 200)
) STRICT;

INSERT INTO impresoras_nueva (id, alias, nombre, tipo, destino, ancho_mm, columnas, columnas_fuente_b, ancho_px,
    pagina_codigos, comando_pagina_codigos, soporta_qr_nativo, soporta_corte, soporta_cajon, soporta_imagen,
    soporta_codigo_barras_nativo, baudios, activa, creada, actualizada, confirmar_impresion)
SELECT id, alias, nombre, tipo, destino, ancho_mm, columnas, columnas_fuente_b, ancho_px,
    pagina_codigos, comando_pagina_codigos, soporta_qr_nativo, soporta_corte, soporta_cajon, soporta_imagen,
    soporta_codigo_barras_nativo, baudios, activa, creada, actualizada, confirmar_impresion
FROM impresoras;

DROP TABLE impresoras;
ALTER TABLE impresoras_nueva RENAME TO impresoras;

-- Los trabajos recuerdan si fueron a una impresora virtual (la impresora puede borrarse después).
ALTER TABLE trabajos ADD COLUMN virtual INTEGER NOT NULL DEFAULT 0 CHECK (virtual IN (0, 1));
