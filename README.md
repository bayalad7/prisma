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

> **Estado: en desarrollo.** Ya funcionan el contrato del documento con su validación, el renderizador ESC/POS con su previsualización, los conectores (red, USB, Bluetooth y archivo), la cola por impresora, la configuración, la base de datos y la CLI; las aplicaciones ya imprimen por la [API v1](docs/api-v1.md) con HTTPS y autorización por niveles, el dueño configura todo desde el [panel de administración](docs/panel.md), los integradores tienen un [SDK JavaScript](sdk/js/) con [guías](docs/integradores/README.md) y firmadores de nivel 3 para PHP, Node.js y Python, y se instala como servicio en Windows, Linux/Raspberry Pi y macOS ([instalación](docs/instalacion.md)). Todavía no hay una versión publicada. Consulta la [hoja de ruta](#hoja-de-ruta).

## Principios

- **Independiente.** No depende de ninguna nube, API ni cuenta externa. Todo se configura en un panel web local.
- **Fácil para integradores.** Una aplicación sin backend puede imprimir con el SDK de JavaScript en minutos.
- **Seguro por defecto.** Ninguna página web ni equipo de la red puede imprimir sin la autorización explícita del dueño.
- **Multiplataforma.** Hay un binario sin dependencias para cada plataforma: Windows (amd64, 386), Linux (amd64, arm64) y macOS (amd64, arm64).

## Cómo se imprime

- Las aplicaciones envían **operaciones**, no bytes: texto, filas, pares, QR, códigos de barras, imágenes, corte, etc. PRISMA las adapta al ancho, la página de códigos y las capacidades de cada impresora.
- Las impresoras se identifican por un **alias** (`"caja"`, `"cocina"`). Las IPs y los puertos nunca salen del puente.
- Cada operación es **independiente** y trae su propio estilo.

La especificación completa está en [docs/contrato-documento-v1.md](docs/contrato-documento-v1.md), y el esquema formal en [schema/documento-v1.schema.json](schema/documento-v1.schema.json) (JSON Schema 2020-12). En [ejemplos/](ejemplos/) hay documentos válidos y un caso inválido por cada regla.

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

## Requisitos para compilar

- Go 1.26 o superior.
- No se necesita CGO: la base de datos es SQLite en Go puro.

## Compilar y probar

En Windows:

```powershell
.\build.ps1 test         # pruebas
.\build.ps1 build        # dist\prisma.exe
.\build.ps1 build-todo   # binarios para las 6 plataformas en dist\
.\build.ps1 lint         # go vet + staticcheck
```

En Linux y macOS:

```sh
make test
make build
make build-todo
make lint
```

## Documentación

| Guía | Contenido |
|---|---|
| [Primeros pasos](docs/primeros-pasos.md) | Compilar e imprimir en una impresora virtual en 5 minutos. |
| [Instalación](docs/instalacion.md) | Producción en Windows, Linux, Raspberry Pi y macOS: servicio, respaldo y actualización. |
| [Conectores](docs/conectores.md) | Impresoras de red, USB y Bluetooth en cada sistema. |
| [CLI](docs/cli.md) · [Configuración](docs/configuracion.md) | Referencia de comandos y de `config.toml`. |
| [Arquitectura](docs/arquitectura.md) | Cómo viaja un documento hasta la impresora. |
| [Desarrollo](docs/desarrollo.md) | Pruebas, golden, convenciones y ramas. |
| [Solución de problemas](docs/solucion-de-problemas.md) | Errores frecuentes, acentos y columnas. |
| [Contrato documento v1](docs/contrato-documento-v1.md) | El JSON que envían las aplicaciones. |
| [API v1](docs/api-v1.md) · [HTTPS y certificados](docs/tls.md) | Endpoints, autorización, CORS, errores y cómo confiar en el certificado. |

El índice completo está en [docs/README.md](docs/README.md).

## Uso

```sh
prisma version                         # versión, plataforma y versiones del contrato
prisma config ruta                     # dónde busca la configuración y los datos
prisma config validar [--config ruta]  # valida config.toml y muestra la configuración efectiva
prisma ejecutar [--config ruta]        # ejecuta el puente (API HTTPS y panel /admin) en primer plano
prisma admin codigo-inicial            # código para configurar el panel desde otro equipo
prisma admin restablecer-contrasena    # genera una contraseña nueva para el panel
prisma tls info | regenerar            # certificado HTTPS en uso / uno nuevo con las IPs actuales
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
prisma impresoras listar
prisma impresoras probar caja                 # comprueba la conexión sin imprimir
prisma impresoras eliminar caja
prisma imprimir --impresora caja documento.json   # solo con [dev] modo_desarrollo = true
```

Cómo obtener el destino de cada impresora (IP fija, nombre USB, cola de CUPS, puerto Bluetooth) y qué permisos necesita el servicio en cada sistema: [docs/conectores.md](docs/conectores.md).

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
internal/cola/        cola FIFO por impresora y estado (en línea / fuera de línea / ocupada)
internal/dev/         decodificador ESC/POS, impresora virtual y previsualización
internal/seguridad/   autorización, llaves, límites, contraseñas y firmas Ed25519 (JCS)
internal/servidor/    HTTPS y API pública /v1
internal/admin/       panel de administración /admin (HTML/CSS/JS embebidos)
internal/servicio/    servicio del sistema (Windows, systemd, launchd) y bitácora rotativa
internal/actualizacion/ versiones nuevas: firma, checksums y reemplazo del binario
internal/version/     versión del binario
sdk/js/               SDK JavaScript público
sdk/firmadores/       firmadores de nivel 3 (PHP, Node.js, Python)
schema/               JSON Schema publicados (embebidos en el binario)
docs/                 especificaciones
ejemplos/             documentos de ejemplo válidos e inválidos
empaquetado/          instaladores: Inno Setup (Windows), .deb (nfpm), .pkg (macOS) y script de Raspberry Pi
.github/workflows/    CI y publicación de versiones
```

## Ramas

| Rama | Uso |
|---|---|
| `main` | Producción. Solo recibe versiones estables desde `develop`. |
| `develop` | Integración. Aquí se trabaja y se fusionan los cambios nuevos. |

Para contribuir, crea tu rama a partir de `develop` y abre el pull request hacia `develop`.

## Hoja de ruta

- [x] Contrato «documento v1», esquema y ejemplos
- [x] Configuración, base de datos con migraciones y CLI básica
- [x] Validación del documento y firma Ed25519 (JCS)
- [x] Renderizador ESC/POS, páginas de códigos y previsualización
- [x] Conectores de red, USB, Bluetooth y archivo
- [x] Cola por impresora
- [x] Servidor HTTPS, API pública `/v1` y autorización por niveles
- [x] Panel de administración
- [x] SDK JavaScript, firmadores de referencia y guías para integradores
- [x] Instalación como servicio, instaladores por plataforma, actualizaciones y CI
- [ ] Primera versión publicada
