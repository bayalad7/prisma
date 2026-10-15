# Guía de desarrollo

Esta guía cubre cómo compilar, probar y contribuir a PRISMA. Las reglas del proyecto están en [CLAUDE.md](../CLAUDE.md): decisiones técnicas que no se cambian sin preguntar, convenciones de nombres y estructura.

## Requisitos

| Herramienta | Versión | Para qué |
|---|---|---|
| Go | 1.26 o superior | compilar y probar |
| Git | cualquiera | control de versiones |
| `make` | opcional (Linux/macOS) | tareas con el `Makefile` |
| PowerShell | 5.1 o superior (Windows) | tareas con `build.ps1` |

No se necesita CGO ni un compilador de C: todo es Go puro, incluida la base de datos SQLite (`modernc.org/sqlite`). staticcheck se descarga solo al correr `lint`.

## Tareas

| Tarea | Windows | Linux / macOS |
|---|---|---|
| Pruebas | `.\build.ps1 test` | `make test` |
| Binario local | `.\build.ps1 build` → `dist\prisma.exe` | `make build` → `dist/prisma` |
| Las 6 plataformas | `.\build.ps1 build-todo` → `dist\prisma-<so>-<arq>[.exe]` | `make build-todo` |
| Análisis estático | `.\build.ps1 lint` (`go vet` + staticcheck v0.8.2) | `make lint` |
| Limpiar | `.\build.ps1 limpiar` | `make limpiar` |

La versión, el commit y la fecha se inyectan al compilar con `-ldflags`. La versión sale de `git describe`. Un binario compilado con `go build` a secas muestra `0.0.0-dev`.

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

```sh
go test ./...                       # todo
go test ./internal/escpos/ -run Fila -v
go test ./internal/dev -actualizar  # regenera los archivos golden
```

- **SQLite en memoria:** las pruebas de `internal/almacen` usan una base en memoria y un reloj controlado.
- **Ejemplos del contrato:** las pruebas recorren `ejemplos/validos/*.json` y `ejemplos/invalidos/indice.json`. Si agregas una regla al contrato, agrega su ejemplo inválido y su entrada en el índice: `codigo`, `detecta` (`esquema` o `puente`), `ruta` y `regla`.
- **Golden:** `internal/dev/testdata/golden/` guarda la vista y la traza de cada ejemplo válido a 58 y 80 mm. Si cambias el renderizador a propósito, regenéralos con `-actualizar` y **revisa el diff** antes de commitear: es la forma de ver qué cambió en el papel.
- **Por plataforma:** `internal/conector` tiene pruebas con build tags (`usb_windows_test.go`, `usb_unix_test.go`). `go test ./...` debe pasar en Windows, Linux y macOS.
- **Race detector:** `-race` requiere CGO y el proyecto se compila sin él. Las pruebas concurrentes (cola, impresora virtual) protegen sus búferes con mutex.

## Estructura del código

| Paquete | Responsabilidad |
|---|---|
| `cmd/prisma` | CLI (cobra). Cada grupo de comandos está en su archivo: `config.go`, `impresoras.go`, `imprimir.go`, `dev.go`, etc. |
| `internal/config` | Lee y valida `config.toml`. Reporta todos los problemas juntos. |
| `internal/rutas` | Rutas por defecto por sistema (build tags). |
| `internal/almacen` | SQLite: migraciones embebidas (`migraciones/NNNN_nombre.sql`) y un repositorio por tabla. |
| `internal/contrato` | Tipos del documento v1, `Decodificar` (JSON estricto, versión, esquema embebido y reglas) y errores con código. |
| `internal/escpos` | `Renderizar(doc, perfil, recursos)`: operaciones → bytes. Codificación de caracteres, disposición de filas, QR, códigos de barras e imágenes. |
| `internal/conector` | `Conector{Enviar, Probar}` para red, USB, Bluetooth y archivo, más la detección de impresoras. |
| `internal/cola` | Una goroutine por impresora (FIFO), límite de 20 trabajos y estado. |
| `internal/seguridad` | Orígenes, contraseñas (argon2id) y firmas Ed25519 sobre JCS. |
| `internal/dev` | Decodificador, simulador, vistas (texto/HTML) e impresora virtual. |
| `schema` | JSON Schema publicado, embebido en el binario. |

El flujo completo de un documento está en [arquitectura.md](arquitectura.md).

## Convenciones

- **Nombres:** los del dominio van en español (`impresora`, `trabajo`, `conector`); los términos técnicos de Go, en inglés (`ctx`, `err`, `Write`).
- **Errores visibles:** llevan un código estable en MAYÚSCULAS, un mensaje en español y una sugerencia (`contrato.Error`, `conector.Error`).
- **Logs:** se escriben con `log/slog` en JSON. **Nunca** registres el contenido de los tickets ni las llaves.
- **Cambios en la base de datos:** agrega una migración nueva (`0002_…sql`). No edites una migración ya publicada.
- **Cambios en el contrato:** actualiza a la vez la especificación (`docs/contrato-documento-v1.md`), el esquema (`schema/`), los ejemplos y el índice de inválidos.
- **Dependencias:** solo las listadas en el prompt de cada parte o en CLAUDE.md; cualquier otra hay que justificarla.

## Ramas y commits

| Rama | Uso |
|---|---|
| `main` | Producción: solo versiones estables. |
| `develop` | Integración: aquí se trabaja. |

Crea tu rama desde `develop` y abre el pull request hacia `develop`.

Los mensajes de commit siguen la forma `tipo(ámbito): asunto`, en imperativo y sin punto final, con un cuerpo que explique el porqué. Tipos en uso: `caracteristica`, `correccion`, `documentacion`, `construccion`, `configuracion`, `refactorizacion`, `prueba`, `rendimiento`, `estilo`. Ámbitos en uso: `prisma`, `contrato`, `escpos`, `conector`, `cola`, `seguridad`, `dev`. Haz un commit por ámbito, y comprueba que cada uno compile y pase sus pruebas por sí solo.

## Antes de abrir un pull request

```sh
go test ./...
go vet ./...
.\build.ps1 lint        # o make lint
.\build.ps1 build-todo  # compila las 6 plataformas
```
