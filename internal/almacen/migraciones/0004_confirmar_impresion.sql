-- «Confirmar impresión»: después de cada ticket, el conector de red o Bluetooth consulta el estado
-- de la impresora (DLE EOT 1) y solo da el trabajo por impreso si responde. Desactivado por
-- defecto, porque algunas impresoras no responden a la consulta.
ALTER TABLE impresoras ADD COLUMN confirmar_impresion INTEGER NOT NULL DEFAULT 0 CHECK (confirmar_impresion IN (0, 1));
