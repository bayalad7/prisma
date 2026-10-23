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

Ejecuta el puente en primer plano hasta que recibe Ctrl+C o SIGTERM. Escribe las bitácoras en JSON por stderr. En un negocio, el puente corre como servicio ([`prisma servicio`](#prisma-servicio)), que hace lo mismo con la bitácora en un archivo rotativo.

- **Al arrancar:**
  - si la versión nueva cambia el esquema de la base, guarda antes una copia en `<datos>/respaldos` (conserva las últimas 5);
  - crea la base de datos si no existe y aplica las migraciones;
  - marca como `PUENTE_REINICIADO` los trabajos que quedaron a medias;
  - purga el historial vencido;
  - aplica los límites guardados desde el panel;
  - avisa si el panel no tiene contraseña.
- **Cada hora:** purga los trabajos más antiguos que la retención (30 días por defecto) y los ids de idempotencia vencidos.
- **Impresoras:** registra en la cola todas las impresoras activas y las prueba cada 30 s. Las que se agregan o modifican con la CLI mientras corre se toman en el siguiente trabajo.
- **Versiones nuevas:** una hora después de arrancar, y luego una vez al día, pregunta a GitHub Releases si hay una versión nueva y la avisa en el panel. Se desactiva en *Ajustes → Actualizaciones*. Las compilaciones de desarrollo no consultan.
- **API y panel:** levanta la API pública ([api-v1.md](api-v1.md)) y el panel en `/admin` ([administracion/](../administracion/README.md)) en `servidor.escuchar:servidor.puerto` según `modo_tls` ([instalacion/https-y-certificados.md](../instalacion/https-y-certificados.md)). Con `autofirmado`, genera el certificado la primera vez y registra su huella en la bitácora.

## `prisma servicio`

Instala PRISMA como servicio del sistema, para que arranque con el equipo y se reinicie si falla.

| Comando | Qué hace |
|---|---|
| `servicio instalar` | Registra el servicio con inicio automático, sin iniciarlo. Si ya existía, lo vuelve a registrar con la configuración actual. |
| `servicio iniciar` · `detener` · `reiniciar` | Controlan el servicio. |
| `servicio estado` | Dice si está instalado y en marcha, y dónde está su bitácora. No requiere permisos de administrador. |
| `servicio desinstalar` | Lo detiene y lo quita. Conserva datos, configuración y bitácoras. |

Todos, salvo `estado`, requieren permisos de administrador: PowerShell como administrador en Windows, `sudo` en Linux y macOS. `--config ruta` instala el servicio con esa configuración.

| | Windows | Linux | macOS |
|---|---|---|---|
| Gestor | Servicio «PRISMA - Puente de Impresión Térmica» (`PRISMA`) | Unidad de systemd `prisma.service` | LaunchDaemon `com.github.bayalad7.prisma` |
| Corre como | LocalSystem | usuario de sistema `prisma` (grupos `lp` y `dialout`), que el comando crea si no existe | usuario oculto `_prisma`, que el comando crea con `dscl` |
| Ante un fallo | se reinicia a los 5 s | `Restart=on-failure`, a los 5 s | se reinicia si termina con error |
| Además | regla del firewall para el puerto, solo subred local; las carpetas de datos y de bitácoras quedan solo para SYSTEM y Administradores | `ProtectSystem`, `ProtectHome` y `NoNewPrivileges`; el binario debe estar fuera de `/home` | — |

- **Bitácora:** `prisma.log` en la carpeta de bitácoras, que rota al llegar a 10 MB y conserva 5 archivos anteriores. En Linux también va a journald (`journalctl -u prisma`).
- **Linux con el paquete .deb:** la unidad la instala el paquete en `/lib/systemd/system`. `servicio instalar` lo detecta y no la duplica; los demás comandos funcionan igual.

## `prisma actualizar [--comprobar] [--si] [--forzar]`

Instala la última versión publicada en GitHub Releases:

1. Descarga `checksums.txt` y verifica su firma Ed25519 con la llave embebida en el binario. Si no verifica, no instala nada.
2. Descarga el archivo de esta plataforma y verifica su SHA-256.
3. Detiene el servicio, reemplaza el programa y lo vuelve a iniciar. En Windows, el `.exe` anterior queda como `prisma.exe.old` y se borra en el siguiente arranque.
4. Si PRISMA se instaló con el paquete `.deb`, descarga el `.deb` de la versión e instala con `dpkg`, para que el sistema de paquetes siga sabiendo qué versión hay.

| Opción | Efecto |
|---|---|
| `--comprobar` | Solo dice si hay una versión nueva. |
| `--si` / `-y` | No pide confirmación. |
| `--forzar` | Reinstala aunque no haya versión nueva, o reemplaza una compilación de desarrollo. |

Requiere permisos de administrador. La base de datos se respalda sola antes de migrarla, al siguiente arranque. Una compilación sin la llave de las versiones (por ejemplo, una compilada con `go build` a secas) se niega a instalar.

## `prisma admin restablecer-contrasena`

Genera una contraseña aleatoria para el panel (por ejemplo, `k7Rmq-Xa3Vt-9pLwe-Hc2Nf`) y la muestra **una sola vez**. Guarda solo su hash argon2id y marca que hay que cambiarla en el próximo inicio de sesión.

Funciona solo en el equipo del puente, porque escribe directamente en su base de datos. Se puede ejecutar aunque el puente esté corriendo: las sesiones abiertas en el panel terminan.

## `prisma admin codigo-inicial`

Genera el código de un solo uso (por ejemplo, `8527-HDH3`) para crear la contraseña del panel **desde otro equipo** de la red. Vence a los 30 minutos y se anula tras 10 intentos fallidos. Desde el propio equipo del puente no hace falta: basta con abrir `https://localhost:8443/admin`.

Falla si el panel ya tiene contraseña; en ese caso usa `prisma admin restablecer-contrasena`.

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
| `--tipo` | **obligatorio** | `red`, `usb`, `bluetooth`, `archivo` o `virtual`. |
| `--destino` | **obligatorio** (salvo `virtual`) | `IP[:puerto]` (9100 por defecto) · nombre en Windows · `/dev/usb/lpN` o cola de CUPS · `COMn`, `/dev/rfcommN` o `/dev/cu.*` · carpeta. La impresora virtual no tiene destino. |
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

## `prisma clientes`

Aplicaciones autorizadas a imprimir. El [panel](../administracion/README.md) hace lo mismo en *Aplicaciones*.

### `clientes agregar`

```sh
prisma clientes agregar --nombre "Caja web" --nivel 2 --origen https://caja.example
```

| Opción | Por defecto | Descripción |
|---|---|---|
| `--nombre` | **obligatorio** | Nombre de la aplicación. |
| `--nivel` | `2` | `1` (origen), `2` (llave) o `3` (firma). Ver [api-v1.md](api-v1.md#autorización). |
| `--origen` | — | Origen exacto desde el que imprime el navegador, por ejemplo `https://miapp.com`. Se puede repetir. Obligatorio en el nivel 1. Si estaba como solicitud pendiente, deja de estarlo. |
| `--impresoras` | todas | Alias permitidos, separados por coma. |
| `--operaciones` | ninguna | Operaciones sensibles habilitadas: `abrir_cajon`, `raw`. |
| `--llave-publica` | — | Nivel 3: archivo PEM o base64 de la llave pública Ed25519. |
| `--exigir-llave` | — | Nivel 3: exige además la llave de cliente. |
| `--con-llave` | — | Genera una llave aunque el nivel no la exija. |
| `--inactivo` | — | Lo registra desactivado. |

Si el cliente tiene llave (nivel 2, `--exigir-llave` o `--con-llave`), el comando la muestra **una sola vez**. El puente guarda solo su hash argon2id.

### `clientes listar`

Muestra el id, el nombre, el nivel, el prefijo de la llave, los orígenes, las impresoras, las operaciones sensibles y si está activo.

### `clientes regenerar-llave ID`

Genera una llave nueva. La anterior deja de funcionar de inmediato.

### `clientes eliminar ID`

Elimina la aplicación con sus orígenes y permisos. Su historial de trabajos se conserva.

## `prisma tls`

| Comando | Descripción |
|---|---|
| `tls info` | Muestra el certificado en uso: archivo, nombres, IPs, vencimiento y huella SHA-256. Avisa si alguna IP actual del equipo no está incluida. |
| `tls regenerar` | Solo en modo `autofirmado`. Genera un certificado nuevo con el nombre y las IPs actuales. El puente en marcha lo toma sin reiniciar. |

Cómo confiar en el certificado en cada sistema: [instalacion/https-y-certificados.md](../instalacion/https-y-certificados.md).

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

### `dev llaves --salida carpeta`

Genera un par Ed25519 de prueba para un cliente de nivel 3: `privada.pem` (permisos 0600) y `publica.pem`. También muestra la pública en base64. No sobrescribe archivos existentes.

### `dev firmar DOCUMENTO.json --llave privada.pem --cliente cli_…`

Firma el documento como lo haría el backend de un cliente de nivel 3: agrega `cliente_id`, `emitido` (ahora), `expira` y `firma`.

| Opción | Por defecto | Descripción |
|---|---|---|
| `--llave` | **obligatoria** | Llave privada Ed25519 en PEM. |
| `--cliente` | **obligatorio** | Id del cliente de nivel 3. |
| `--nuevo-id` | — | Reemplaza el `id` por un UUID v4 nuevo, para que el puente no lo tome como repetido. |
| `--vigencia` | `90s` | Tiempo hasta `expira` (máximo `120s`). |
| `-o`, `--salida` | salida estándar | Archivo de salida. |

```sh
prisma dev firmar pedido.json --llave privada.pem --cliente cli_xxx --nuevo-id -o firmado.json
curl -k https://localhost:8443/v1/imprimir -H "Content-Type: application/json" --data-binary @firmado.json
```

Para comprobar un firmador propio, usa el vector fijo de [testdata/vector-firma](../../testdata/vector-firma/LEEME.md).
