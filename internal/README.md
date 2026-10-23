# Paquetes internos

Cada paquete tiene su comentario de paquete (`go doc ./internal/<paquete>`). El flujo completo de un documento y las dependencias entre paquetes están en [docs/arquitectura/](../docs/arquitectura/README.md).

| Paquete | Responsabilidad |
|---|---|
| [`config`](config/) | Lee y valida `config.toml` (solo lo necesario para arrancar). |
| [`rutas`](rutas/) | Rutas de configuración, datos y bitácoras por sistema. |
| [`almacen`](almacen/) | SQLite: migraciones, repositorios, respaldo y restauración. |
| [`contrato`](contrato/) | Documento v1: decodificación estricta, esquema y reglas. |
| [`escpos`](escpos/) | Renderizador de operaciones a ESC/POS. |
| [`conector`](conector/) | Envío a la impresora (red, USB, Bluetooth, archivo) y detección. |
| [`cola`](cola/) | Una fila FIFO por impresora y su estado. |
| [`seguridad`](seguridad/) | Autorización (origen, llave, firma, permisos), límites, contraseñas y firmas. |
| [`servidor`](servidor/) | HTTPS, API pública `/v1` y CORS. |
| [`admin`](admin/) | Panel de administración `/admin` y su API interna. |
| [`servicio`](servicio/) | Servicio del sistema (Windows, systemd, launchd) y bitácora rotativa. |
| [`actualizacion`](actualizacion/) | Versiones nuevas: firma de checksums, SHA-256, reemplazo del binario y aviso. |
| [`dev`](dev/) | Herramientas de desarrollo: decodificador, simulador, previsualización e impresora virtual. |
| [`version`](version/) | Versión, commit y fecha del binario (se inyectan al compilar). |

El punto de entrada y la CLI están en [`cmd/prisma`](../cmd/prisma/); los comandos se describen en [docs/referencia/cli.md](../docs/referencia/cli.md).
