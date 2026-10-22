# Registro de cambios

Todos los cambios notables de PRISMA. El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/) y las versiones, [SemVer](https://semver.org/lang/es/). Mientras la versión sea 0.x, una versión menor puede traer cambios incompatibles; se marcarán como tales.

Hay tres versiones independientes: la del **programa** (este registro), la de la **API** (`/v1/`) y la del **documento** (`"version": "1"`).

## [Sin publicar]

### Agregado

- Licencia Apache-2.0 (`LICENSE`, `NOTICE`). El SDK pasa de `UNLICENSED` a `Apache-2.0`.
- Documentación reorganizada por tipo de lector: `instalacion/`, `administracion/`, `operacion/`, `integradores/`, `seguridad/`, `arquitectura/`, `adr/`, `diagramas/`, `referencia/` y `desarrollo/`.
- Referencia formal de la API en OpenAPI 3.1 (`docs/referencia/openapi.yaml`) y una tabla con todos los códigos de error y advertencias.
- Modelo de amenazas, políticas de seguridad, runbook de operación, 14 registros de decisiones (ADR) y 19 diagramas.
- Pruebas que mantienen la documentación fiel al código (`go test ./docs`): rutas de la API, códigos de error y enlaces.
- `CONTRIBUTING.md` y `SECURITY.md`.

### Cambiado

- La *Ayuda* del panel enlaza la documentación de la versión instalada, no la de la rama `develop`.
- Los mensajes que citan la documentación usan las rutas nuevas. Las rutas viejas siguen existiendo como páginas que redirigen.

## [0.1.0] — 2026-10-22

Primera versión publicada.

### Agregado

- **Puente** para Windows (amd64, 386), Linux (amd64, arm64, incluida Raspberry Pi) y macOS (amd64, arm64): un binario en Go sin CGO, con SQLite en Go puro.
- **API pública `/v1`** por HTTPS: `salud`, `impresoras`, `imprimir` y `previsualizar`.
- **Contrato «documento v1»** con JSON Schema 2020-12 embebido y ejemplos válidos e inválidos: texto, separador, fila, par, salto, QR, código de barras, imagen, cortar, abrir cajón y `raw`.
- **Renderizador ESC/POS** por perfil de impresora: páginas de códigos CP437, CP850, CP858 y WPC1252; QR y códigos de barras por comando o como imagen; imágenes en 1 bit.
- **Conectores** de red (TCP 9100), USB (winspool, CUPS, `/dev/usb`), Bluetooth (puerto serie) y archivo, con detección de impresoras. Opción *Confirmar impresión* (`DLE EOT`) en red y Bluetooth.
- **Cola** FIFO por impresora, con estado y pruebas periódicas.
- **Seguridad** por niveles: origen, llave `pk_` (argon2id) y firma Ed25519 sobre JCS; idempotencia persistente; CORS con eco exacto; límites por IP y por aplicación; solicitudes de origen que el dueño aprueba; `abrir_cajon` y `raw` deshabilitadas por defecto.
- **Panel de administración** en `/admin`: impresoras, aplicaciones, solicitudes, historial, imágenes, ajustes, certificado, respaldo y restauración.
- **CLI** `prisma`: `ejecutar`, `servicio`, `actualizar`, `admin`, `impresoras`, `clientes`, `tls`, `config`, `imprimir` y `dev` (previsualizar, renderizar, decodificar, impresora virtual, llaves y firmar).
- **SDK JavaScript** para navegador y Node.js 18+, con validación local, reintentos idempotentes y diagnóstico de conexión; firmadores de nivel 3 en PHP, Node.js y Python.
- **Instalación como servicio** en los tres sistemas; instaladores para Windows (Inno Setup), Debian y Raspberry Pi (`.deb` e instalador de una línea) y macOS (`.pkg`).
- **Actualizaciones** con `prisma actualizar`, verificadas con la firma Ed25519 de `checksums.txt` y el SHA-256 de cada archivo; aviso de versión nueva en el panel.
- Certificado autofirmado de 825 días, con aviso desde 60 días antes de vencer y regeneración automática si venció.

### Problemas conocidos

- El SDK no está publicado en npm. Se carga desde `cdn.jsdelivr.net/gh/bayalad7/prisma@v0.1.0/sdk/js/dist/…` o copiando `sdk/js/dist/` al proyecto.
- Los instaladores no tienen firma de código: Windows muestra SmartScreen y macOS pide *Abrir de todas formas*. Verifica los archivos con `checksums.txt`.
- Los trabajos que llegan por la API pasan directo de `encolado` a `completado` o `fallido`; el estado `imprimiendo` solo lo usa `prisma imprimir`.
- La *Ayuda* del panel enlaza la documentación de la rama `develop`.

[Sin publicar]: https://github.com/bayalad7/prisma/compare/v0.1.0...develop
[0.1.0]: https://github.com/bayalad7/prisma/releases/tag/v0.1.0
