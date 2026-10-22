# Datos

SQLite (`prisma.db`) en modo WAL, con claves foráneas activas y tablas `STRICT`. Las migraciones están embebidas en el binario y se aplican al arrancar: cada una en su propia transacción y con su versión en `PRAGMA user_version`.

| Tabla | Contenido |
|---|---|
| `impresoras` | Alias, tipo, destino y perfil: ancho, columnas, puntos, página de códigos y capacidades. |
| `clientes` | Nivel, hash y prefijo de la llave, llave pública, operaciones sensibles y `todas_las_impresoras`. |
| `clientes_impresoras` | Impresoras permitidas, cuando un cliente no tiene acceso a todas. |
| `clientes_origenes` | Orígenes aprobados. Cada origen pertenece a un solo cliente. |
| `solicitudes_origen` | Orígenes desconocidos que intentaron imprimir (pendientes o rechazados). |
| `trabajos` | Historial: estado, código de error, advertencias y duración. **Sin** contenido del ticket. |
| `ids_recientes` | Idempotencia: id del documento, hash del contenido y trabajo original. Sobrevive a los reinicios. |
| `recursos` | Imágenes PNG guardadas en el puente. |
| `ajustes` | Contraseña del panel, retención, límites, nombre del puente, acceso al panel, etc. |

- **Idempotencia:** un `id` se reserva al aceptar el trabajo y se libera si el trabajo falla o lo interrumpe un reinicio, para que el reintento se imprima. Ver [diagramas/idempotencia.md](../diagramas/idempotencia.md).
- **Mantenimiento:** cada hora se purgan los trabajos más viejos que la retención, los `id` vencidos y las solicitudes de origen sin actividad (30 días las pendientes, 90 las rechazadas).
- **Respaldo antes de migrar:** si una versión nueva trae migraciones, la base se copia a `respaldos/` antes de migrarla y se conservan las últimas 5.

El diagrama de tablas está en [diagramas/modelo-datos.md](../diagramas/modelo-datos.md).
