# Conectar impresoras a PRISMA

Esta guía explica cómo dejar cada tipo de impresora lista para PRISMA en Windows, Linux (incluida Raspberry Pi) y macOS. En cada caso hay que obtener el **destino** que se registra en el puente: una IP, el nombre de la impresora, una cola de CUPS, un puerto serie o una carpeta.

```sh
prisma impresoras detectar          # lo que ve este equipo (USB, colas, puertos serie)
prisma impresoras detectar --red    # además busca impresoras en la red local (puerto 9100)
prisma impresoras agregar --alias caja --tipo red --destino 192.168.1.50 --cajon
prisma impresoras probar caja       # comprueba la conexión sin imprimir
```

En el panel de administración se hace lo mismo con un formulario.

| Tipo | `--tipo` | Destino | Funciona en |
|---|---|---|---|
| Red (Ethernet o Wi-Fi) | `red` | `IP` o `IP:puerto` (9100 por defecto) | todos |
| USB | `usb` | Windows: nombre de la impresora · Linux: `/dev/usb/lpN` o cola de CUPS · macOS: cola de CUPS | todos |
| Bluetooth | `bluetooth` | Windows: `COMn` · Linux: `/dev/rfcommN` · macOS: `/dev/cu.<nombre>` | todos |
| Archivo | `archivo` | una carpeta | todos (desarrollo y auditoría) |

---

## Impresoras de red

PRISMA envía los bytes por TCP al puerto **9100** (modo «raw» o JetDirect), que es el que usan casi todas las impresoras de tickets.

- **Tiempos:** 3 s para conectar y 10 s para enviar.
- **Reintento:** si la conexión falla, PRISMA reintenta una vez. Si ya se habían enviado datos, no reintenta, para no imprimir el ticket dos veces.

### Confirmar impresión

En modo «raw», enviar termina cuando el ticket está en el búfer de red del equipo, **no cuando la impresora lo recibió**. Un ticket normal (5 a 20 KB) siempre cabe en ese búfer. Si la impresora se apaga o se desconecta justo mientras lo recibe, el puente lo da por impreso y la aplicación no se entera.

La opción **Confirmar impresión** del perfil (casilla en el panel, o `--confirmar` en `prisma impresoras agregar`) lo resuelve:

1. Después de cada ticket, el puente envía la consulta de estado `DLE EOT 1` (`10 04 01`). Es un comando de tiempo real: no imprime nada y la impresora lo contesta al recibirlo.
2. Como la consulta viaja detrás del ticket, la respuesta confirma que el ticket completo llegó.

| Respuesta | Resultado |
|---|---|
| En línea | Trabajo completado. |
| Fuera de línea: tapa abierta, sin papel o en error | Trabajo **completado** con la advertencia `IMPRESORA_FUERA_DE_LINEA`. La impresora ya tiene el ticket y lo imprimirá al resolverlo. Si el trabajo fallara, el reintento de la aplicación saldría dos veces. |
| Ninguna en 10 s, o la impresora cerró la conexión | `TIEMPO_AGOTADO` («no confirmó que recibió el ticket»). El trabajo falla y su `id` queda libre para reintentar. |

- **Funciona con:** red (puerto 9100) y Bluetooth o serie. Casi todas las impresoras ESC/POS responden a `DLE EOT`.
- **No funciona con:** USB por winspool (Windows), CUPS ni `/dev/usb/lp*`, porque esos caminos no devuelven la respuesta de la impresora. Para ellas la casilla no aparece.
- **Está desactivada por defecto.** Algunas impresoras económicas no responden a la consulta, y con ella activada todos sus trabajos fallarían.
- **Cómo probarlo:** actívala y usa *Ticket de prueba* en el panel. Si falla con `TIEMPO_AGOTADO` aunque el ticket salió, la impresora no admite la consulta: desactívala.
- La impresora virtual (`prisma dev impresora-virtual`) responde como una impresora en línea.

### Darle una IP fija (imprescindible)

Si la impresora recibe una IP distinta cada vez que se enciende, el puente deja de encontrarla. Hay dos formas de fijarla; elige una:

1. **Reserva DHCP en el router (recomendada):**
   1. Busca la MAC de la impresora en la hoja de autoprueba. La mayoría la imprime si la enciendes manteniendo presionado el botón FEED.
   2. En el router, reserva una IP para esa MAC.
2. **IP estática en la impresora:** se configura con la utilidad del fabricante o con su página web (escribe la IP actual en el navegador). Elige una IP fuera del rango que reparte el router.

Después, registra esa IP como destino: `--destino 192.168.1.50`.

### Encontrarla

`prisma impresoras detectar --red` prueba el puerto 9100 en la subred de este equipo:

- tiene un límite de conexiones simultáneas y de tiempo (15 s);
- si la subred es mayor que /24, solo revisa el /24 de este equipo;
- para otra subred, usa `--subred 192.168.2.0/24`.

El escaneo nunca se hace automáticamente: solo cuando lo pides desde el panel o la CLI.

### Si no conecta

- El equipo de PRISMA y la impresora deben estar en la **misma red**. Las redes de invitados suelen aislar a los equipos entre sí.
- La impresora acepta **una conexión a la vez**. Si otro programa le está enviando un trabajo, rechazará la conexión hasta que termine.

---

## Impresoras USB

### Windows

1. Instala el controlador del fabricante. Si no hay controlador, agrega la impresora con el controlador **«Generic / Text Only»**. PRISMA envía los datos en modo RAW, así que el controlador no los interpreta.
2. Copia el nombre **exacto** que aparece en *Configuración > Bluetooth y dispositivos > Impresoras y escáneres*, o el que muestra `prisma impresoras detectar`.
3. Agrega la impresora: `--tipo usb --destino "EPSON TM-T20II Receipt"`.

PRISMA usa la cola de Windows (winspool) con un documento RAW. Si `probar` indica que está **sin conexión**:

- revisa el cable y que esté encendida;
- abre la cola de la impresora en Windows y desmarca *Impresora > Usar impresora sin conexión*;
- elimina los trabajos atascados.

Si el spooler no acepta el trabajo en 60 s, PRISMA lo cancela en la cola de Windows (`AbortPrinter`) y responde `TIEMPO_AGOTADO`. Así una cola bloqueada no detiene los demás trabajos ni impide detener el servicio.

El usuario con el que corre el servicio PRISMA necesita permiso de *Imprimir* sobre la impresora. Lo tiene por defecto.

### Linux y Raspberry Pi

Hay dos opciones.

**A. Dispositivo directo, `/dev/usb/lpN` (lo más simple).** El módulo `usblp` crea `/dev/usb/lp0` al conectar la impresora.

```sh
ls -l /dev/usb/lp*                 # crw-rw---- 1 root lp … /dev/usb/lp0
sudo usermod -aG lp prisma         # el usuario del servicio debe estar en el grupo lp
```

Registra la impresora con `--tipo usb --destino /dev/usb/lp0`.

Con varias impresoras USB, el número `lpN` puede cambiar entre reinicios. Para evitarlo, crea un nombre fijo con una regla de udev, usando los identificadores que muestra `lsusb`:

```
# /etc/udev/rules.d/90-prisma.rules
SUBSYSTEM=="usbmisc", ATTRS{idVendor}=="04b8", ATTRS{idProduct}=="0202", SYMLINK+="impresora-caja", GROUP="lp", MODE="0660"
```

Después usa `--destino /dev/impresora-caja`.

Si CUPS ya tiene abierta la impresora, el dispositivo responde `PUERTO_OCUPADO`. En ese caso, usa la opción B.

**B. Cola de CUPS en modo raw.** Úsala si la impresora también la usan otros programas.

```sh
lpinfo -v | grep usb               # ej.: usb://EPSON/TM-T20II?serial=…
sudo lpadmin -p termica -E -v "usb://EPSON/TM-T20II?serial=XXXX" -m raw
```

Registra la impresora con `--tipo usb --destino termica`. PRISMA envía los trabajos con `lp -d termica -o raw`. Si `lpadmin` advierte que las colas raw están obsoletas, puedes ignorarlo: siguen funcionando.

### macOS

macOS siempre pasa por CUPS:

1. Conecta la impresora y búscala con `lpinfo -v`.
2. Crea una cola raw, igual que en Linux: `sudo lpadmin -p termica -E -v "usb://…" -m raw`.
3. Registra la impresora con `--tipo usb --destino termica`. `lpstat -e` lista las colas existentes.

---

## Impresoras Bluetooth

Las impresoras Bluetooth de tickets usan el perfil SPP (puerto serie). PRISMA las maneja como un puerto serie:

- la velocidad por defecto es de 9600 baudios (se cambia con `--baudios`);
- al probar o imprimir, el sistema establece la conexión Bluetooth al abrir el puerto.

### Windows: puerto COM saliente

1. Empareja la impresora en *Configuración > Bluetooth y dispositivos > Agregar dispositivo*. El PIN suele ser `0000` o `1234`.
2. Abre *Más opciones de Bluetooth > Puertos COM*. Cada impresora tiene dos puertos; usa el que dice **Saliente** (por ejemplo, `COM5`). Si no aparece, pulsa *Agregar… > Saliente* y elige la impresora.
3. Regístrala con `--tipo bluetooth --destino COM5`.

`PUERTO_OCUPADO` significa que otro programa, como otra aplicación de impresión, tiene el puerto abierto.

### Linux: rfcomm persistente

```sh
bluetoothctl
  power on
  scan on                          # anota la MAC, ej. 66:22:B3:01:02:03
  pair 66:22:B3:01:02:03
  trust 66:22:B3:01:02:03
  quit
sudo rfcomm bind 0 66:22:B3:01:02:03 1   # crea /dev/rfcomm0 (canal 1)
sudo usermod -aG dialout prisma          # el usuario del servicio debe estar en el grupo dialout
```

`rfcomm bind` se pierde al reiniciar. Para que sea persistente, crea un servicio de systemd:

```ini
# /etc/systemd/system/rfcomm-caja.service
[Unit]
Description=Puerto serie Bluetooth de la impresora caja
After=bluetooth.service
Requires=bluetooth.service

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/bin/rfcomm bind 0 66:22:B3:01:02:03 1
ExecStop=/usr/bin/rfcomm release 0

[Install]
WantedBy=multi-user.target
```

```sh
sudo systemctl enable --now rfcomm-caja
```

Registra la impresora con `--tipo bluetooth --destino /dev/rfcomm0`.

### macOS: `/dev/cu.*`

1. Empareja la impresora en *Ajustes del Sistema > Bluetooth*.
2. Busca su puerto con `ls /dev/cu.*`, por ejemplo `/dev/cu.PT-210`. Usa siempre `cu.` y no `tty.`: los puertos `tty.` esperan una señal del módem y se bloquean al abrirlos.
3. Regístrala con `--tipo bluetooth --destino /dev/cu.PT-210`.

---

## Permisos del servicio

| Sistema | Qué necesita el usuario del servicio PRISMA |
|---|---|
| Windows | El servicio corre como LocalSystem: imprime en las impresoras instaladas para todos los usuarios y abre los puertos COM sin permisos especiales. Una impresora instalada solo para un usuario no la ve: instálala para todos. |
| Linux | El usuario `prisma` necesita el grupo `lp` para `/dev/usb/lpN` y `dialout` para `/dev/rfcommN` y `/dev/ttyUSBN`; el instalador y `prisma servicio instalar` los agregan. CUPS lo puede usar cualquier usuario local. |
| macOS | El usuario `_prisma` imprime a través de CUPS, que lo permite por defecto, y abre los puertos `/dev/cu.*`, que son de lectura y escritura para todos. |

Después de agregar un usuario a un grupo, reinicia el servicio para que tome el cambio.

---

## Desarrollar sin impresora

**Impresora virtual.** Escucha por TCP como una impresora de red. Muestra la traza ESC/POS y la previsualización de cada trabajo, y guarda cada uno como `.bin`.

```sh
prisma dev impresora-virtual --puerto 9100 --ancho 80
prisma impresoras agregar --alias virtual --tipo red --destino 127.0.0.1:9100
prisma imprimir --impresora virtual ejemplos/validos/venta-simple.json   # requiere [dev] modo_desarrollo = true
```

**Conector de archivo.** `--tipo archivo --destino ./tickets` guarda cada trabajo como `<fecha>-<id>.bin`, junto con un `.txt` que muestra cómo se vería impreso.

---

## Errores de conexión

| Código | Qué significa | Qué revisar |
|---|---|---|
| `IMPRESORA_NO_ENCONTRADA` | El alias, el nombre, la cola o el puerto no existe. | El destino exacto en `prisma impresoras detectar`. |
| `IMPRESORA_INALCANZABLE` | Existe, pero no respondió o está marcada sin conexión. | Encendido, cable o red, papel y tapa. |
| `PUERTO_OCUPADO` | Otro programa tiene abierto el puerto o el dispositivo. | Cierra ese programa, o en Linux usa la cola de CUPS. |
| `TIEMPO_AGOTADO` | La impresora dejó de recibir datos a mitad del trabajo o, con [Confirmar impresión](#confirmar-impresión), no confirmó que lo recibió. | Papel atascado, tapa abierta o Bluetooth fuera de alcance. Si pasa con todos los tickets, la impresora no admite la confirmación. |
| `PERMISO_DENEGADO` | El usuario del servicio no puede abrir el destino. | Los grupos `lp` y `dialout` (Linux) o los permisos de la impresora (Windows). |
| `COLA_LLENA` | La impresora ya tiene 20 trabajos pendientes. | Que la impresora esté avanzando; si está detenida, el primer trabajo dirá por qué. |

**Advertencia `IMPRESORA_FUERA_DE_LINEA`:** solo con Confirmar impresión. El ticket llegó, pero la impresora informó tapa abierta, falta de papel o un error; saldrá al resolverlo. No reenvíes el ticket.

El puente prueba cada impresora cada 30 s y el panel muestra su estado: *en línea*, *fuera de línea* u *ocupada*. Si la impresora está imprimiendo, el puente no la prueba.
