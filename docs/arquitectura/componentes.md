# Componentes

Un solo binario, `prisma`, con estos paquetes. La columna *Importa* sale de `go list ./...` (solo paquetes del propio proyecto); el diagrama está en [diagramas/02-componentes.md](../diagramas/02-componentes.md).

| Paquete | Rol | Importa |
|---|---|---|
| `cmd/prisma` | CLI (cobra) y punto de entrada: `ejecutar`, `servicio`, `actualizar`, `admin`, `impresoras`, `clientes`, `tls`, `config`, `imprimir`, `dev`, `version`. | todos los de `internal/` |
| `internal/config` | `config.toml` de arranque: carga, valores por defecto y validación con todos los problemas juntos. | `rutas` |
| `internal/rutas` | Rutas por defecto de configuración, datos y bitácoras en cada sistema (build tags). | — |
| `internal/almacen` | SQLite (modernc): migraciones embebidas, un repositorio por tabla, respaldo y restauración en caliente, normalización de orígenes. | — |
| `internal/contrato` | Tipos del documento v1, `LeerSobre` (lo mínimo para autorizar) y `Decodificar` (JSON estricto, versión, esquema y reglas). | `schema` |
| `internal/escpos` | `Renderizar(doc, perfil, recursos)`: operaciones → bytes ESC/POS y advertencias; codificación de caracteres, filas, QR, códigos de barras e imágenes. | `almacen`, `contrato` |
| `internal/conector` | `Conector{Enviar, Probar}` por red, USB (winspool, CUPS, `/dev/usb`), Bluetooth/serie y archivo; Confirmar impresión (`DLE EOT`); detección de impresoras. | `almacen`, `dev` |
| `internal/cola` | Una goroutine por impresora (FIFO), límite de trabajos, pruebas periódicas y estado. | `conector` |
| `internal/seguridad` | Autorización (origen, llave, firma, permisos), firma Ed25519 sobre JCS, argon2id, límites (cubetas de fichas), código inicial. | `almacen`, `contrato` |
| `internal/servidor` | HTTPS (autofirmado, archivos, solo localhost), API pública `/v1`, CORS, cabeceras de seguridad y traducción de errores. Monta el panel. | `almacen`, `cola`, `conector`, `config`, `contrato`, `dev`, `escpos`, `seguridad` |
| `internal/admin` | Panel `/admin`: HTML/CSS/JS embebidos, sesiones, CSRF y la API interna `/admin/api`. | `actualizacion`, `almacen`, `cola`, `conector`, `config`, `contrato`, `dev`, `escpos`, `seguridad`, `servidor` |
| `internal/servicio` | Servicio del sistema (kardianos/service): plantillas de systemd y launchd, usuario, carpetas, firewall de Windows y bitácora rotativa. | — |
| `internal/actualizacion` | Versiones nuevas en GitHub Releases: firma de `checksums.txt`, SHA-256, reemplazo del binario y aviso del panel. | — |
| `internal/dev` | Decodificador, simulador, vistas en texto y HTML, impresora virtual. | `almacen`, `contrato`, `escpos` |
| `internal/version` | Versión, commit, fecha y versiones del contrato aceptadas (inyectadas con `-ldflags`). | — |
| `schema` | JSON Schema 2020-12 del documento, embebido en el binario. | — |
| `sdk/js` | SDK JavaScript: constructor de documentos, validación local, reintentos y diagnóstico. | (API `/v1` por HTTP) |
| `sdk/firmadores` | Firmadores de nivel 3 de referencia en PHP, Node.js y Python. | — |

Dependencias directas (`go.mod`): `modernc.org/sqlite` (SQLite en Go puro), `github.com/kardianos/service` (servicio del sistema), `github.com/spf13/cobra` (CLI), `go.bug.st/serial` (puerto serie), `golang.org/x/crypto` (argon2id), `golang.org/x/sys` (winspool y llamadas del sistema), `golang.org/x/text` (normalización y páginas de códigos), `github.com/BurntSushi/toml` (`config.toml`), `github.com/santhosh-tekuri/jsonschema/v6` (validación del esquema), `github.com/gowebpki/jcs` (forma canónica JCS) y `github.com/skip2/go-qrcode` (QR dibujado como imagen).
