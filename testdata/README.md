# Datos de prueba públicos

| Carpeta | Contenido |
|---|---|
| [`vector-firma/`](vector-firma/LEEME.md) | Vector fijo de firma del nivel 3: llave de prueba (pública, de RFC 8032), documento, forma canónica JCS, firma y documento firmado. Con él, cualquier firmador debe obtener exactamente la misma firma. |

Lo usan las pruebas del puente (`internal/seguridad`), los firmadores de referencia (`sdk/firmadores`) y el SDK. Las llaves de esta carpeta son públicas: nunca las uses fuera de pruebas.
