# PRISMA

**Puente Remoto de Impresión Sobre Múltiples Arquitecturas**

PRISMA es un servicio que se instala en una PC, Raspberry Pi o Mac dentro de la red local de un negocio. Cualquier aplicación web puede enviarle por HTTPS un documento JSON con las operaciones de un ticket. PRISMA lo valida, verifica que el cliente esté autorizado, lo convierte a ESC/POS según el perfil de la impresora y lo imprime por USB, red (TCP 9100) o Bluetooth.

```json
{
  "version": "1",
  "id": "3f2b8c1e-5a4d-4e7b-9c2a-1d0e6f8a7b90",
  "impresora": "caja",
  "operaciones": [
    {"tipo": "texto", "valor": "ABARROTES LA ESPERANZA", "alineacion": "centro", "negrita": true, "tamano": [2, 2]},
    {"tipo": "par", "izquierda": "TOTAL", "derecha": "$163.50", "negrita": true},
    {"tipo": "qr", "valor": "https://facturas.example.com/t/4127"},
    {"tipo": "cortar"}
  ]
}
```

> **Versión actual: [0.3.4](https://github.com/bayalad7/prisma/releases/tag/v0.3.4)** ([registro de cambios](CHANGELOG.md)). Funciona en Windows, Linux/Raspberry Pi y macOS.
>
> - Las aplicaciones imprimen por la [API v1](docs/referencia/api-v1.md), con HTTPS y autorización por niveles.
> - El dueño configura todo desde el [panel de administración](docs/administracion/README.md): primeros pasos guiados, impresoras con prueba de acentos, aplicaciones y solicitudes, trabajos con «qué pasó y qué hacer», y el manual y los diagramas dentro del panel, sin Internet.
> - Con la **impresora virtual** y *Probar impresión* se puede probar sin papel.
> - Los integradores tienen un [SDK JavaScript](docs/integradores/sdk.md) y firmadores de nivel 3 para PHP, Node.js y Python.
> - Se instala como servicio con los [instaladores](docs/instalacion/README.md) de cada sistema. En ese equipo, el panel abre sin advertencia de certificado.

## Principios

- **Independiente.** No depende de ninguna nube, API ni cuenta externa. Todo se configura en un panel web local.
- **Fácil para integradores.** Una aplicación sin backend puede imprimir con el SDK de JavaScript en minutos.
- **Seguro por defecto.** Ninguna página web ni equipo de la red puede imprimir sin la autorización explícita del dueño.
- **Multiplataforma.** Hay un binario sin dependencias para cada plataforma: Windows (amd64, 386), Linux (amd64, arm64) y macOS (amd64, arm64).

## Cómo se imprime

- Las aplicaciones envían **operaciones**, no bytes: texto, filas, pares, QR, códigos de barras, imágenes, corte, etc. PRISMA las adapta al ancho, la página de códigos y las capacidades de cada impresora.
- Las impresoras se identifican por un **alias** (`"caja"`, `"cocina"`). Las IPs y los puertos nunca salen del puente.
- Cada operación es **independiente** y trae su propio estilo.

La especificación completa está en [docs/referencia/contrato-documento-v1.md](docs/referencia/contrato-documento-v1.md), y el esquema formal en [schema/documento-v1.schema.json](schema/documento-v1.schema.json) (JSON Schema 2020-12). En [ejemplos/](ejemplos/) hay documentos válidos y un caso inválido por cada regla.

## Seguridad

El dueño del puente elige un nivel para cada aplicación cliente:

| Nivel | Qué exige | Protege contra |
|---|---|---|
| 1 — Origen | El encabezado `Origin` del navegador debe estar aprobado. | Páginas web maliciosas. |
| 2 — Llave (por defecto) | Origen + una llave `pk_...` en cada petición. | Equipos de la red sin llave. |
| 3 — Firma | Origen + documentos firmados con Ed25519 por el backend del cliente. | Documentos alterados, repetidos o falsificados. |

Además:

- Las operaciones sensibles (`abrir_cajon`, `raw`) están deshabilitadas por defecto y se habilitan por cliente.
- Cuando un origen desconocido intenta imprimir, queda como solicitud pendiente para que el dueño la apruebe o la rechace.

### HTTPS en la red local

Los navegadores solo dejan que una página HTTPS hable con otra dirección HTTPS. Por eso el puente genera una **autoridad local** y un certificado del servidor firmado por ella ([ADR 0021](docs/adr/0021-autoridad-local-y-certificado-del-servidor.md)).

- **La autoridad local** está limitada a la red local ([ADR 0020](docs/adr/0020-confiar-en-el-certificado-del-puente-al-instalar.md)): aunque un equipo confíe en ella, no sirve para hacerse pasar por ningún sitio de Internet.
- **En el equipo del puente**, el instalador hace que el sistema confíe en ella.
- **En los demás equipos que imprimen**, se instala una sola vez: `prisma.crt`, en *Ajustes → HTTPS* del panel.
- **Regenerar el certificado del servidor** (cambio de IP o vencimiento) no obliga a reinstalar nada.

Con un dominio propio también se puede usar Let's Encrypt ([guía](docs/instalacion/https-y-certificados.md)).

## Requisitos para compilar

- Go 1.26 o superior (1.26.4 o superior para empaquetar el `.deb`).
- No se necesita CGO: la base de datos es SQLite en Go puro.

## Compilar y probar

En Windows:

```powershell
.\build.ps1 test         # pruebas
.\build.ps1 build        # dist\prisma.exe
.\build.ps1 build-todo   # binarios para las 6 plataformas en dist\
.\build.ps1 lint         # go vet + staticcheck
.\build.ps1 test-sdk     # SDK JavaScript y firmadores (Node.js, PHP, Python)
.\build.ps1 deb          # paquetes .deb; también: instalador (Inno Setup), checksums
```

Pruebas de punta a punta del panel (Playwright; compilan y arrancan sus propios puentes):

```sh
cd e2e && npm ci && npx playwright install chromium && npm test
```

En Linux y macOS:

```sh
make test
make build
make build-todo
make lint
make test-sdk
make deb                 # también: pkg (solo en macOS), checksums
```

El CI ([`.github/workflows`](.github/workflows/)) corre en Windows, Ubuntu y macOS:

- `go test -race` y las pruebas del SDK y los firmadores;
- la instalación real del servicio, incluida la reinstalación con el servicio en marcha;
- la confianza en la autoridad local, validada por el propio sistema;
- las pruebas de punta a punta del panel: flujos, pantallas en claro y oscuro con axe, teclado, usuario nuevo y un Firefox real contra el certificado.

Las versiones se publican con una etiqueta `v*` ([docs/desarrollo/publicar.md](docs/desarrollo/publicar.md)).

## Documentación

| Si… | Empieza por |
|---|---|
| Integras PRISMA en tu aplicación | [docs/integradores/](docs/integradores/README.md): inicio rápido, SDK, niveles de seguridad y firmas |
| Instalas PRISMA en un negocio | [docs/instalacion/](docs/instalacion/README.md): Windows, Raspberry Pi, Linux y macOS |
| Administras un puente | [docs/administracion/](docs/administracion/README.md) (panel) y [docs/operacion/](docs/operacion/README.md) (respaldo, actualización, bitácoras, recuperación) |
| Desarrollas PRISMA | [docs/desarrollo/](docs/desarrollo/README.md), [CONTRIBUTING.md](CONTRIBUTING.md), [docs/arquitectura/](docs/arquitectura/README.md) y [docs/adr/](docs/adr/README.md) |
| Buscas una referencia | [API v1](docs/referencia/api-v1.md) · [OpenAPI](docs/referencia/openapi.yaml) · [contrato](docs/referencia/contrato-documento-v1.md) · [códigos de error](docs/referencia/codigos-de-error.md) · [CLI](docs/referencia/cli.md) · [configuración](docs/referencia/configuracion.md) · [glosario](docs/referencia/glosario.md) |
| Te preocupa la seguridad | [docs/seguridad/](docs/seguridad/README.md) y [SECURITY.md](SECURITY.md) |

El índice completo está en [docs/README.md](docs/README.md).

## Uso

```sh
prisma version                         # versión, plataforma y versiones del contrato
prisma config ruta                     # dónde busca la configuración y los datos
prisma config validar [--config ruta]  # valida config.toml y muestra la configuración efectiva
prisma ejecutar [--config ruta]        # ejecuta el puente (API HTTPS y panel /admin) en primer plano
prisma admin codigo-inicial            # código para configurar el panel desde otro equipo
prisma admin restablecer-contrasena    # genera una contraseña nueva para el panel
prisma tls info                        # certificado del servidor y autoridad local en uso
prisma tls regenerar [--nueva-autoridad]  # certificado del servidor nuevo (misma autoridad: los equipos no hacen nada)
prisma tls confiar [--quitar]          # como administrador: este equipo confía en la autoridad local
prisma servicio instalar | iniciar | estado | detener | reiniciar | desinstalar
prisma actualizar [--comprobar]        # instala la última versión (firma y SHA-256 verificados)
```

### Aplicaciones

```sh
prisma clientes agregar --nombre "Caja web" --nivel 2 --origen https://caja.example   # muestra su llave pk_ una vez
prisma clientes agregar --nombre "Backend" --nivel 3 --llave-publica publica.pem
prisma clientes listar | eliminar ID | regenerar-llave ID
```

Cómo integrar PRISMA en una aplicación (SDK, niveles, firmas y errores): [docs/integradores/](docs/integradores/README.md).

### Impresoras

```sh
prisma impresoras detectar [--red] [--json]   # impresoras, colas y puertos de este equipo (y de la red)
prisma impresoras agregar --alias caja --tipo red --destino 192.168.1.50 [--ancho 58|80] [--cajon]
prisma impresoras agregar --alias prueba --tipo virtual   # impresora virtual: el ticket se ve en el panel
prisma impresoras listar
prisma impresoras probar caja                 # comprueba la conexión sin imprimir
prisma impresoras eliminar caja
prisma imprimir --impresora caja documento.json   # solo con [dev] modo_desarrollo = true
```

Cómo obtener el destino de cada impresora (IP fija, nombre USB, cola de CUPS, puerto Bluetooth) y qué permisos necesita el servicio en cada sistema: [docs/instalacion/impresoras.md](docs/instalacion/impresoras.md).

### Herramientas de desarrollo

Todas usan un perfil de prueba de 58 u 80 mm con página de códigos CP858.

```sh
prisma dev impresora-virtual [--puerto 9100] [--ancho 80]  # impresora de red simulada: traza + vista de cada trabajo
prisma dev previsualizar documento.json [--ancho 58|80]   # simula el ticket en la terminal
prisma dev previsualizar documento.json --html vista.html # o en una página HTML
prisma dev renderizar documento.json -o ticket.bin        # genera los bytes ESC/POS
prisma dev decodificar ticket.bin                         # traza legible de los comandos
```

Si el documento usa imágenes guardadas (`"recurso"`), indica con `--recursos carpeta` dónde están los archivos `<nombre>.png`.

Las pruebas golden de `internal/dev` comparan la traza y la vista de cada ejemplo válido a 58 y 80 mm. Si un cambio en el renderizador es intencional, regéneralas con `go test ./internal/dev -actualizar` y revisa la diferencia antes de commitear.

### Configuración

`config.toml` solo contiene lo necesario para arrancar. Impresoras, clientes y ajustes se administran desde el panel y se guardan en la base de datos. En [empaquetado/config.toml](empaquetado/config.toml) hay un ejemplo comentado.

```toml
[servidor]
escuchar = "0.0.0.0"
puerto = 8443
modo_tls = "autofirmado"   # autofirmado | archivos | solo_localhost_http

[datos]
ruta = ""                  # vacío = ruta por defecto del sistema

[dev]
modo_desarrollo = false
```

Si `config.toml` no existe, PRISMA usa los valores por defecto.

### Rutas por defecto

| Sistema | Configuración | Datos (`prisma.db`) |
|---|---|---|
| Windows | `%ProgramData%\PRISMA\config.toml` | `%ProgramData%\PRISMA\datos` |
| Linux | `/etc/prisma/config.toml` | `/var/lib/prisma` |
| macOS | `/Library/Application Support/PRISMA/config.toml` | `/Library/Application Support/PRISMA/datos` |

Para desarrollar, usa un `config.toml` propio cuyo `[datos] ruta` apunte a una carpeta local. Así no tocas las rutas del sistema.

## Estructura

```
cmd/prisma/           CLI y punto de entrada
internal/config/      carga y validación de config.toml
internal/rutas/       rutas por sistema operativo
internal/almacen/     SQLite: esquema, migraciones y repositorios
internal/contrato/    tipos del documento v1 y validación (esquema + reglas)
internal/escpos/      renderizador a ESC/POS, páginas de códigos, QR, códigos de barras e imágenes
internal/conector/    red, USB (winspool / dispositivo / CUPS), Bluetooth (puerto serie), archivo y detección
internal/cola/        cola FIFO por impresora y estado (desconocido / en línea / fuera de línea / ocupada)
internal/dev/         decodificador ESC/POS, impresora virtual y previsualización
internal/seguridad/   autorización, llaves, límites, contraseñas y firmas Ed25519 (JCS)
internal/servidor/    HTTPS y API pública /v1
internal/admin/       panel de administración /admin (HTML/CSS/JS embebidos, manual y diagramas sin Internet)
internal/servicio/    servicio del sistema (Windows, systemd, launchd) y bitácora rotativa
internal/actualizacion/ versiones nuevas: firma, checksums y reemplazo del binario
internal/version/     versión del binario
sdk/js/               SDK JavaScript público
sdk/firmadores/       firmadores de nivel 3 (PHP, Node.js, Python)
e2e/                  pruebas de punta a punta del panel (Playwright, axe, Firefox real)
testdata/             vector de firma público
schema/               JSON Schema publicados (embebidos en el binario)
docs/                 documentación por tipo de lector (ver docs/README.md)
ejemplos/             documentos de ejemplo válidos e inválidos
empaquetado/          instaladores: Inno Setup (Windows), .deb (nfpm), .pkg (macOS) y script de Raspberry Pi
.github/workflows/    CI y publicación de versiones
```

## Contribuir

Crea tu rama desde `develop` y abre el pull request hacia `develop`; `main` solo recibe lo que se publica. Las reglas de commits y pull requests están en [CONTRIBUTING.md](CONTRIBUTING.md), y las vulnerabilidades se reportan en privado como explica [SECURITY.md](SECURITY.md).

## Hoja de ruta

- [x] Contrato «documento v1», esquema y ejemplos
- [x] Configuración, base de datos con migraciones y CLI básica
- [x] Validación del documento y firma Ed25519 (JCS)
- [x] Renderizador ESC/POS, páginas de códigos y previsualización
- [x] Conectores de red, USB, Bluetooth y archivo
- [x] Impresora virtual dentro del puente, con simulación de fallas (0.2.0)
- [x] Cola por impresora
- [x] Servidor HTTPS, API pública `/v1` y autorización por niveles
- [x] Panel de administración
- [x] SDK JavaScript, firmadores de referencia y guías para integradores
- [x] Instalación como servicio, instaladores por plataforma, actualizaciones y CI
- [x] Primera versión publicada (0.1.0)
- [x] Panel rediseñado, accesible y con el manual y los diagramas dentro, sin Internet (0.2.0)
- [x] Probar impresión, rollo en vivo y comprobador de permisos (0.2.0)
- [x] Panel sin advertencia de certificado en el equipo del puente: autoridad local limitada a la red local (0.3.0 y 0.3.1)
- [ ] SDK publicado en npm
- [ ] Firma de código de los instaladores de Windows y macOS
- [ ] Negociación de versiones del documento (contrato §10)

## Licencia

[Apache-2.0](LICENSE). Ver también [NOTICE](NOTICE).
