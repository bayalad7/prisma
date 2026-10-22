# Guía de desarrollo

Esta guía cubre cómo compilar y probar PRISMA. Para contribuir, ver también [CONTRIBUTING.md](../../CONTRIBUTING.md). Las reglas del proyecto están en [CLAUDE.md](../../CLAUDE.md): decisiones técnicas que no se cambian sin preguntar, convenciones de nombres y estructura.

## Requisitos

| Herramienta | Versión | Para qué |
|---|---|---|
| Go | 1.26 o superior; **1.26.4 o superior** para `deb` (nfpm v2.48.0 lo exige) | compilar y probar |
| Git | cualquiera | control de versiones |
| `make` | opcional (Linux/macOS) | tareas con el `Makefile` |
| PowerShell | 5.1 o superior (Windows) | tareas con `build.ps1` |
| Node.js | 18 o superior | SDK JavaScript y firmador de Node (`test-sdk`) |
| PHP · Python | 8.2 con `sodium` · 3.9 con `cryptography` | firmadores de referencia (`test-sdk`) |

No se necesita CGO ni un compilador de C: todo es Go puro, incluida la base de datos SQLite (`modernc.org/sqlite`). staticcheck se descarga solo al correr `lint`.

## Tareas

| Tarea | Windows | Linux / macOS |
|---|---|---|
| Pruebas | `.\build.ps1 test` | `make test` |
| Pruebas del SDK y los firmadores | `.\build.ps1 test-sdk` | `make test-sdk` |
| Binario local | `.\build.ps1 build` → `dist\prisma.exe` | `make build` → `dist/prisma` |
| Las 6 plataformas | `.\build.ps1 build-todo` → `dist\prisma-<so>-<arq>[.exe]` | `make build-todo` |
| Análisis estático | `.\build.ps1 lint` (`go vet` + staticcheck v0.8.2) | `make lint` |
| Paquetes e instaladores | `.\build.ps1 deb` · `instalador` · `checksums` | `make deb` · `pkg` · `checksums` |
| Limpiar | `.\build.ps1 limpiar` | `make limpiar` |

La versión, el commit, la fecha y la llave pública de las versiones (`empaquetado/llave-releases.pub`) se inyectan al compilar con `-ldflags`. La versión sale de `git describe`, sin la «v». Un binario compilado con `go build` a secas muestra `0.0.0-dev` y no puede usar `prisma actualizar`.

**CI** ([`.github/workflows/ci.yml`](../../.github/workflows/ci.yml)) corre en cada push a `main` y `develop` y en cada pull request, con el último parche de Go 1.26 (`go-version: '1.26.x'`), no con el mínimo de `go.mod`:

- `go vet`, staticcheck y `go test -race` en Windows, Ubuntu y macOS;
- las pruebas del SDK en los tres sistemas, y las de los firmadores;
- un trabajo que instala el servicio de verdad en los tres sistemas, comprueba que el panel responda, lo reinicia y lo desinstala.

Las versiones se publican con una etiqueta: ver [publicar.md](publicar.md).

Para revisar el código de otra plataforma sin cambiar de equipo:

```sh
GOOS=linux GOARCH=arm64 go vet ./...          # bash
$env:GOOS="darwin"; go vet ./...              # PowerShell (luego: Remove-Item Env:GOOS)
```

## Entorno de desarrollo

1. Crea `config.dev.toml` con una carpeta de datos local y `modo_desarrollo = true` (ver [primeros-pasos.md](primeros-pasos.md#2-crear-una-configuración-de-desarrollo)). Tanto `config.dev.toml` como `datos-dev/` están en `.gitignore`.
2. Agrega impresoras a esa base con `prisma impresoras agregar --config config.dev.toml …`.
3. Para trabajar sin hardware, usa la impresora virtual (`prisma dev impresora-virtual`) o el conector de archivo (`--tipo archivo --destino ./tickets`).

### Herramientas de `prisma dev`

| Comando | Para qué |
|---|---|
| `dev previsualizar doc.json [--ancho 58\|80] [--html archivo]` | Ver cómo saldrá un ticket, en la terminal o en HTML. |
| `dev renderizar doc.json -o ticket.bin` | Obtener los bytes ESC/POS exactos. |
| `dev decodificar ticket.bin` | Leer un `.bin` comando por comando: sirve para depurar el renderizador o capturas de otros sistemas. |
| `dev impresora-virtual [--puerto 9100]` | Impresora de red simulada que muestra y guarda cada trabajo. |

La previsualización no es una maqueta aparte: ejecuta los mismos bytes que recibiría la impresora en una impresora virtual (`internal/dev/simulador.go`). Si la previsualización se ve bien, los bytes son correctos.

## Pruebas

Cómo correrlas, qué cubren y cómo regenerar los archivos generados: [pruebas.md](pruebas.md).

## Estructura del código

Cada paquete, su responsabilidad y sus dependencias reales están en [arquitectura/componentes.md](../arquitectura/componentes.md). El flujo de un documento, en [arquitectura/flujo-de-impresion.md](../arquitectura/flujo-de-impresion.md).

## Convenciones

- **Nombres:** los del dominio van en español (`impresora`, `trabajo`, `conector`); los términos técnicos de Go, en inglés (`ctx`, `err`, `Write`).
- **Errores visibles:** llevan un código estable en MAYÚSCULAS, un mensaje en español y una sugerencia (`contrato.Error`, `conector.Error`).
- **Logs:** se escriben con `log/slog` en JSON. **Nunca** registres el contenido de los tickets ni las llaves.
- **Cambios en la base de datos:** agrega una migración nueva (`0002_…sql`). No edites una migración ya publicada.
- **Cambios en el contrato:** actualiza a la vez la especificación (`docs/referencia/contrato-documento-v1.md`), el esquema (`schema/`), los ejemplos y el índice de inválidos.
- **Dependencias:** las de `go.mod` (ver [arquitectura/componentes.md](../arquitectura/componentes.md)). Cualquier dependencia nueva hay que justificarla en el pull request.

## Ramas, commits y pull requests

Están en [CONTRIBUTING.md](../../CONTRIBUTING.md). Para publicar una versión, ver [publicar.md](publicar.md).
