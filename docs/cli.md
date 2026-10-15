# Referencia de la línea de comandos

```
prisma [comando] [opciones]
```

**Opciones globales** (valen para cualquier comando):

| Opción | Descripción |
|---|---|
| `--config ruta` | Usa ese `config.toml` en lugar del de la ruta del sistema. Si se indica, el archivo debe existir. |
| `-h`, `--help` | Muestra la ayuda del comando. |

**Salida y errores:**
- Si un comando termina bien, devuelve el código 0. Si falla, devuelve 1 y escribe en stderr `Error: …` y, cuando la hay, una línea `Sugerencia: …`.
- Los errores de documentos y de impresión llevan un código estable, por ejemplo `IMPRESORA_INALCANZABLE` o `FILA_NO_CABE`.

---

## `prisma version`

Muestra la versión, el commit, la fecha de compilación, la plataforma y las versiones del contrato de documento que acepta.

## `prisma config`

| Comando | Descripción |
|---|---|
| `config validar` | Valida `config.toml` y muestra la configuración efectiva. Si el archivo por defecto no existe, avisa que se usan los valores por defecto. Termina con código 1 si hay problemas y los lista todos juntos. |
| `config ruta` | Muestra dónde busca PRISMA la configuración, los datos, la base de datos y las bitácoras. |

## `prisma ejecutar`

Ejecuta el puente en primer plano hasta que recibe Ctrl+C o SIGTERM. Escribe las bitácoras en JSON por stderr.

- **Al arrancar:**
  - crea la base de datos si no existe y aplica las migraciones;
  - marca como `PUENTE_REINICIADO` los trabajos que quedaron a medias;
  - purga el historial vencido;
  - avisa si el panel no tiene contraseña.
- **Cada hora:** purga los trabajos más antiguos que la retención (30 días por defecto) y los ids de idempotencia vencidos.
- ⏳ Cuando exista el servidor HTTPS, este comando también lo levantará.

## `prisma admin restablecer-contrasena`

Genera una contraseña aleatoria para el panel (por ejemplo, `k7Rmq-Xa3Vt-9pLwe-Hc2Nf`) y la muestra **una sola vez**. Guarda solo su hash argon2id y marca que hay que cambiarla en el próximo inicio de sesión.

Funciona solo en el equipo del puente, porque escribe directamente en su base de datos. Se puede ejecutar aunque el puente esté corriendo.

## `prisma impresoras`

### `impresoras detectar [--red] [--subred CIDR] [--json]`

Lista lo que encuentra el equipo:

| Sistema | Qué detecta |
|---|---|
| Windows | Impresoras instaladas (EnumPrinters): el `DESTINO` es el nombre que va en `--destino`. |
| Linux / macOS | Colas de CUPS (`lpstat`) y dispositivos `/dev/usb/lp*`. |
| Todos | Puertos serie (`COMn`, `/dev/rfcommN`, `/dev/cu.*`). |

| Opción | Descripción |
|---|---|
| `--red` | Busca además equipos con el puerto 9100 abierto en las subredes privadas de este equipo. Tarda hasta 15 s y revisa como máximo un /24 por interfaz. |
| `--subred 192.168.1.0/24` | Escanea esa subred en lugar de las locales. Se puede repetir e implica `--red`. |
| `--json` | Salida en JSON: `[{tipo, destino, detalle}]`. |

### `impresoras agregar`

```sh
prisma impresoras agregar --alias caja --tipo red --destino 192.168.1.50 --cajon
```

| Opción | Por defecto | Descripción |
|---|---|---|
| `--alias` | **obligatorio** | Nombre con el que las aplicaciones eligen la impresora: `^[a-z0-9][a-z0-9_-]{0,31}$`. |
| `--tipo` | **obligatorio** | `red`, `usb`, `bluetooth` o `archivo`. |
| `--destino` | **obligatorio** | `IP[:puerto]` (9100 por defecto) · nombre en Windows · `/dev/usb/lpN` o cola de CUPS · `COMn`, `/dev/rfcommN` o `/dev/cu.*` · carpeta. |
| `--nombre` | el alias | Nombre descriptivo. |
| `--ancho` | `80` | Ancho del papel en mm: `58` u `80`. Fija las columnas y los puntos por defecto. |
| `--columnas` | 48 (80 mm) / 32 (58 mm) | Caracteres por línea con la fuente A. |
| `--columnas-b` | 64 / 42 | Caracteres por línea con la fuente B. |
| `--ancho-px` | 576 / 384 | Puntos imprimibles por línea, para imágenes y códigos. |
| `--pagina` | `CP858` | Página de códigos: `CP437`, `CP850`, `CP858` o `WPC1252`. |
| `--comando-pagina` | `19` | El `n` de `ESC t n` para esa página en tu impresora (en Epson: CP437=0, CP850=2, WPC1252=16, CP858=19). |
| `--baudios` | 9600 | Velocidad del puerto serie (Bluetooth). |
| `--cajon` | no | Hay un cajón de dinero conectado. |
| `--sin-corte` | — | No tiene cuchilla: el corte se reemplaza por un avance de papel. |
| `--sin-qr-nativo` | — | No imprime QR por comando: se envían como imagen. |
| `--sin-barras-nativo` | — | No imprime códigos de barras por comando: se envían como imagen. |
| `--sin-imagen` | — | No imprime imágenes. |
| `--inactiva` | — | La agrega desactivada. |

### `impresoras listar`

Muestra las impresoras configuradas: alias, tipo, destino, ancho, columnas, página de códigos y si está activa.

### `impresoras probar ALIAS`

Comprueba que la impresora esté alcanzable **sin imprimir**: abre y cierra la conexión, o consulta su estado en Windows o CUPS. Si falla, muestra el código y la sugerencia.

### `impresoras eliminar ALIAS`

Elimina la impresora. Los permisos de los clientes sobre ella se borran con ella; el historial de trabajos se conserva.

## `prisma imprimir DOCUMENTO.json [--impresora ALIAS]`

Valida, renderiza e imprime un documento pasando por la misma cola que usará el servidor, y deja el trabajo en el historial con el cliente `cli_consola`.

- Si no se indica `--impresora`, usa el campo `impresora` del documento.
- **Solo funciona con `[dev] modo_desarrollo = true`.** En un negocio, las aplicaciones imprimen por la API con su autorización.

## `prisma dev`

Herramientas para desarrollar sin impresora. Usan un perfil de prueba de 58 u 80 mm con CP858 y todas las capacidades.

### `dev previsualizar DOCUMENTO.json`

Muestra en la terminal cómo se imprimirá el documento.

| Opción | Descripción |
|---|---|
| `--ancho 58\|80` | Ancho del papel (80 por defecto). |
| `--html archivo.html` | Guarda una página HTML con la simulación exacta, en lugar de mostrarla en la terminal. |
| `--recursos carpeta` | Carpeta con las imágenes `<nombre>.png` que el documento usa con `"recurso"`. |
| `--sin-color` | Sin secuencias ANSI. También se desactivan con la variable de entorno `NO_COLOR`. |

La vista en terminal es aproximada: las imágenes salen como bloques y la fuente B se dibuja con el ancho de la A, con `↳` donde una línea continúa. El HTML sí es exacto.

### `dev renderizar DOCUMENTO.json -o salida.bin [--ancho] [--recursos]`

Escribe los bytes ESC/POS del documento en un archivo.

### `dev decodificar archivo.bin`

Traza legible: una línea por comando, con su posición, sus bytes y su significado. Ejemplo:

```
000005  1B 61 01     alineación centro
000017  41 42 41 …   texto "ABARROTES LA ESPERANZA"
```

### `dev impresora-virtual`

Escucha por TCP como una impresora de red. Por cada trabajo muestra la traza y la previsualización, y guarda el `.bin`. Atiende una conexión a la vez, como una impresora real.

| Opción | Por defecto | Descripción |
|---|---|---|
| `--puerto` | `9100` | Puerto TCP. |
| `--escuchar` | `127.0.0.1` | Usa `0.0.0.0` para recibir trabajos desde otros equipos. |
| `--ancho` | `80` | Ancho del papel de la simulación. |
| `--salida` | carpeta temporal | Dónde guardar los `.bin`. |
| `--sin-traza` | — | Muestra solo la previsualización. |
| `--sin-color` | — | Sin secuencias ANSI. |
