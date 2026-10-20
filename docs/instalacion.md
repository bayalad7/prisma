# Instalar PRISMA

Esta guía instala PRISMA en el equipo del negocio: una PC con Windows, una Raspberry Pi o un equipo con Linux, o una Mac. Al terminar, PRISMA:

- arranca solo cada vez que se enciende el equipo, sin que nadie inicie sesión;
- se vuelve a levantar solo si llega a fallar;
- se administra desde el navegador, en el **panel**: `https://<ip-del-equipo>:8443/admin`.

> ⏳ Los instaladores se descargan de la página de [versiones de PRISMA](https://github.com/bayalad7/prisma/releases/latest). Estarán ahí a partir de la primera versión publicada; mientras tanto, usa la [instalación manual](#instalación-manual-avanzado).

## Antes de empezar

- **Un equipo que quede encendido** mientras el negocio imprime. Sirve uno modesto: PRISMA ocupa poco.
- **En la misma red** que las impresoras de red y que las cajas, tabletas o teléfonos desde donde se imprime.
- **Con IP fija.** Las aplicaciones buscan a PRISMA por su dirección, así que, si cambia, dejan de encontrarlo. Se fija en el router (*reserva de DHCP*) o en la configuración de red del equipo.
- **Permisos de administrador** para instalar.

| Sistema | Versión mínima | Qué descargar |
|---|---|---|
| Windows | 10 u 11, de 64 bits (también Server 2016 o posterior) | `PRISMA-<versión>-windows-amd64-setup.exe` |
| Windows de 32 bits | 10 | `PRISMA-<versión>-windows-386-setup.exe` |
| Raspberry Pi | Raspberry Pi OS **de 64 bits** (Pi 3, 4, 5 o Zero 2) | se instala con una línea (abajo) |
| Linux | Debian 11, Ubuntu 20.04 o posteriores, de 64 bits | `prisma_<versión>_amd64.deb` o `_arm64.deb` |
| macOS | 12 Monterey o posterior (Intel o Apple Silicon) | `PRISMA-<versión>-macos.pkg` |

---

## Windows

1. Descarga `PRISMA-<versión>-windows-amd64-setup.exe` de la [página de versiones](https://github.com/bayalad7/prisma/releases/latest).
2. Ábrelo. Si Windows muestra **«Windows protegió su PC»**, pulsa *Más información → Ejecutar de todas formas* (ver [SmartScreen](#windows-smartscreen-o-el-antivirus-bloquean-el-instalador)).
3. Acepta los permisos de administrador y sigue el asistente.
4. Al terminar, deja marcada **«Abrir el panel de administración»** y pulsa *Finalizar*.
5. El navegador dirá que **la conexión no es privada**. Es lo esperado: PRISMA usa su propio certificado. Pulsa *Configuración avanzada → Continuar a localhost*.
6. Crea la **contraseña del panel** (al menos 10 caracteres) y guárdala en un lugar seguro.

Qué hizo el instalador:

- copió PRISMA a `C:\Program Files\PRISMA` y lo agregó al `PATH`, así que el comando `prisma` funciona en PowerShell;
- creó el servicio de Windows **«PRISMA - Puente de Impresión Térmica»**, que arranca con el equipo y se reinicia solo si falla (se ve en *Servicios*, `services.msc`);
- abrió el puerto 8443 en el Firewall de Windows, solo para la red local;
- guardó configuración, datos y bitácoras en `C:\ProgramData\PRISMA`. Solo los administradores pueden leer los datos.

Para entrar al panel desde otra computadora de la red, abre `https://<ip-de-este-equipo>:8443/admin`. Si todavía no creaste la contraseña, el panel te pedirá un código: genéralo en este equipo, en una PowerShell **como administrador**, con `prisma admin codigo-inicial`.

---

## Raspberry Pi y Linux

### Con una línea

En la Raspberry Pi (o en el equipo con Debian o Ubuntu), abre una terminal y escribe:

```sh
curl -fsSL https://raw.githubusercontent.com/bayalad7/prisma/main/empaquetado/linux/instalar.sh | sh
```

El script pide tu contraseña de `sudo` y hace lo siguiente:

1. Comprueba que el sistema sea de 64 bits.
2. Descarga el paquete de la última versión.
3. Verifica su firma y su SHA-256: si algo no coincide, no instala nada.
4. Lo instala.

Si prefieres leerlo antes de ejecutarlo:

```sh
curl -fsSLO https://raw.githubusercontent.com/bayalad7/prisma/main/empaquetado/linux/instalar.sh
less instalar.sh
sh instalar.sh
```

### Con el paquete .deb

Descarga de la [página de versiones](https://github.com/bayalad7/prisma/releases/latest) el `.deb` de tu equipo: `arm64` para la Raspberry Pi, `amd64` para una PC. Instálalo con:

```sh
sudo apt install ./prisma_<versión>_arm64.deb
```

### Después de instalar

El paquete:

- crea el usuario de sistema `prisma`, con acceso a impresoras USB (grupo `lp`) y Bluetooth (grupo `dialout`);
- instala el servicio `prisma`, que arranca con el equipo y se reinicia si falla;
- muestra la dirección del panel.

Una Raspberry Pi suele no tener pantalla, así que el panel se abre desde otra computadora: `https://<ip-de-la-pi>:8443/admin`. La primera vez pide un código de configuración; genéralo en la Pi con:

```sh
sudo -u prisma prisma admin codigo-inicial
```

El código vence a los 30 minutos.

| Para… | Comando |
|---|---|
| Ver si está en marcha | `systemctl status prisma` |
| Ver la bitácora en vivo | `journalctl -u prisma -f` |
| Reiniciarlo | `sudo systemctl restart prisma` |
| Saber la IP de la Pi | `hostname -I` |

Configuración: `/etc/prisma/config.toml`. Datos: `/var/lib/prisma`. Bitácoras: `/var/log/prisma` y journald.

---

## macOS

1. Descarga `PRISMA-<versión>-macos.pkg` de la [página de versiones](https://github.com/bayalad7/prisma/releases/latest). Sirve para Mac con Intel y con Apple Silicon.
2. Ábrelo y sigue el instalador. Si macOS dice que **no puede verificar el desarrollador**, abre *Ajustes del Sistema → Privacidad y seguridad* y pulsa *Abrir de todas formas* junto al aviso de PRISMA.
3. Abre `https://localhost:8443/admin` en Safari o Chrome. Acepta el aviso del certificado (*Mostrar detalles → visitar este sitio web*) y crea la contraseña del panel.

El instalador crea el usuario oculto `_prisma` y un servicio de sistema (LaunchDaemon) que arranca con la Mac y se reinicia si falla. Configuración y datos: `/Library/Application Support/PRISMA`. Bitácoras: `/Library/Logs/PRISMA`, también visibles en la app **Consola**.

Para generar el código del panel desde la Terminal: `sudo -u _prisma prisma admin codigo-inicial`.

---

## Después de instalar, en cualquier sistema

1. **Confía en el certificado** en cada equipo, tableta o teléfono desde el que se imprimirá, para que el navegador deje de mostrar la advertencia. Se descarga en el panel (*Ajustes → HTTPS → Descargar certificado*) y se instala como explica [tls.md](tls.md#confiar-en-el-certificado).
2. **Agrega las impresoras** en el panel, en *Impresoras*, e imprime el **ticket de diagnóstico** de cada una. Si es USB o Bluetooth, revisa antes [conectores.md](conectores.md).
3. **Autoriza las aplicaciones** que van a imprimir en *Aplicaciones*, o aprueba su página en *Solicitudes* cuando intenten imprimir. Todo el panel está explicado en [panel.md](panel.md).

---

## Actualizar

Cuando hay una versión nueva, el panel lo avisa en *Inicio*. El puente pregunta a GitHub una vez al día; se puede desactivar en *Ajustes → Actualizaciones*.

| Sistema | Cómo |
|---|---|
| Windows | Ejecuta el instalador de la versión nueva, o en PowerShell **como administrador**: `prisma actualizar`. |
| Raspberry Pi y Linux | `sudo prisma actualizar`, o vuelve a ejecutar la línea de instalación. |
| macOS | Ejecuta el `.pkg` nuevo, o `sudo prisma actualizar`. |

`prisma actualizar`:

1. Verifica la firma de la versión y el SHA-256 del archivo antes de tocar nada.
2. Reemplaza el programa.
3. Reinicia el servicio.

Si la versión nueva cambia la base de datos, PRISMA guarda antes una copia en la carpeta `respaldos`, junto a los datos, y conserva las últimas 5. Tu configuración y tus datos se mantienen.

Para ver si hay versión nueva sin instalarla: `prisma actualizar --comprobar`.

## Respaldo

Todo lo que configuraste en el panel está en un solo archivo, `prisma.db`. Descarga una copia en *Ajustes → Respaldo* (se hace sin detener nada) y guárdala fuera del equipo. La copia contiene datos sensibles del panel, así que guárdala en un lugar privado.

## Desinstalar

| Sistema | Cómo | Qué se conserva |
|---|---|---|
| Windows | *Configuración → Aplicaciones → PRISMA → Desinstalar* | `C:\ProgramData\PRISMA` (bórrala si ya no la necesitas). |
| Raspberry Pi y Linux | `sudo apt remove prisma` | Datos, configuración y usuario. `sudo apt purge prisma` borra todo. |
| macOS | `sudo prisma servicio desinstalar && sudo rm /usr/local/bin/prisma` | `/Library/Application Support/PRISMA` y `/Library/Logs/PRISMA`. |

---

## Problemas comunes

### Windows: SmartScreen o el antivirus bloquean el instalador

PRISMA es un programa nuevo y Windows todavía no lo «conoce».

- **SmartScreen** («Windows protegió su PC»): pulsa *Más información → Ejecutar de todas formas*.
- **El antivirus lo borra o lo pone en cuarentena:** es un falso positivo habitual con programas recién publicados. Antes de permitirlo, comprueba que el archivo sea el original: en PowerShell, `Get-FileHash .\PRISMA-…-setup.exe` debe dar el mismo valor que su línea en `checksums.txt`, en la página de la versión. Luego restáuralo desde la cuarentena y agrega una exclusión para `C:\Program Files\PRISMA`.
- Si la versión está firmada digitalmente, en *Propiedades → Firmas digitales* del archivo aparece el editor.

### El panel o las aplicaciones no llegan desde otro equipo

1. En el equipo de PRISMA, `https://localhost:8443/admin` debe abrir. Si no abre, PRISMA no está en marcha: revisa [que el servicio arranque](#el-servicio-no-arranca).
2. Desde el otro equipo, prueba `https://<ip>:8443/v1/salud`.
   - **No carga:** es la red o el firewall.
   - **Windows:** el instalador abrió el puerto solo para la subred local. Si la caja está en otra subred (por ejemplo, otra red Wi-Fi), agrega una regla en *Firewall de Windows → Configuración avanzada → Reglas de entrada*.
   - **Linux con `ufw`:** `sudo ufw allow from 192.168.1.0/24 to any port 8443 proto tcp`, con tu red.
   - **Macs con el firewall activado:** permite las conexiones entrantes de `prisma` en *Ajustes del Sistema → Red → Firewall → Opciones*.
   - **Redes de invitados:** suelen aislar a los equipos entre sí. Conecta la caja a la red principal.
3. Si responde con una advertencia de certificado, instala el certificado en ese equipo ([tls.md](tls.md#confiar-en-el-certificado)).

Nunca abras el puerto 8443 hacia Internet en el router.

### El navegador dice que la conexión no es privada

El puente usa un certificado propio, que tu navegador todavía no conoce. Para entrar al panel basta con *Configuración avanzada → Continuar*. Para que las **aplicaciones** puedan imprimir desde ese equipo, en cambio, hay que instalar el certificado: el aviso aceptado solo vale para navegar ([tls.md](tls.md#confiar-en-el-certificado)).

Si el aviso aparece en equipos que antes funcionaban, cambió la IP del equipo de PRISMA: regenera el certificado en *Ajustes → HTTPS* y vuelve a instalarlo en los equipos. Para evitarlo, dale IP fija.

### Linux: la impresora USB no responde (permiso denegado)

El servicio corre con el usuario `prisma`, que pertenece al grupo `lp`. Si la impresora aparece como `/dev/usb/lp0`, comprueba:

```sh
ls -l /dev/usb/lp0        # debe ser del grupo lp
id prisma                 # debe incluir lp
```

Si CUPS ya usa la impresora, el puerto aparece ocupado. En ese caso agrégala como cola de CUPS en modo raw. Ambos casos están en [conectores.md](conectores.md#linux-y-raspberry-pi).

### Bluetooth: la impresora no conecta o aparece «ocupada»

- La impresora debe estar **emparejada** con el equipo de PRISMA y tener su puerto serie: `COM5` en Windows, `/dev/rfcomm0` en Linux.
- Casi todas las impresoras Bluetooth aceptan **un solo equipo conectado a la vez**. Si un teléfono o una tableta se conectó a ella, PRISMA no puede: desconéctalos.
- En Linux, el usuario `prisma` necesita el grupo `dialout` (el instalador lo agrega).

Detalles por sistema en [conectores.md](conectores.md#impresoras-bluetooth).

### El servicio no arranca

| Sistema | Ver el error |
|---|---|
| Windows | `C:\ProgramData\PRISMA\logs\prisma.log`. Si el servicio ni siquiera llega a escribir la bitácora, el *Visor de eventos* (Registros de Windows → Sistema) muestra el aviso del Administrador de servicios. |
| Linux | `journalctl -u prisma -n 50` |
| macOS | `/Library/Logs/PRISMA/prisma.log` y `/Library/Logs/PRISMA/launchd.err` |

La causa más común es que **otro programa ya usa el puerto 8443**. Cambia `puerto` en `config.toml` y reinicia: `prisma servicio reiniciar`, como administrador. La segunda es un error en `config.toml`: `prisma config validar` lo muestra con su número de línea.

### Olvidé la contraseña del panel

En el equipo de PRISMA, como administrador:

| Sistema | Comando |
|---|---|
| Windows | `prisma admin restablecer-contrasena` |
| Linux | `sudo -u prisma prisma admin restablecer-contrasena` |
| macOS | `sudo -u _prisma prisma admin restablecer-contrasena` |

Muestra una contraseña nueva, que el panel pide cambiar al entrar.

Más problemas, de impresión y de las aplicaciones, en [solucion-de-problemas.md](solucion-de-problemas.md).

---

## Instalación manual (avanzado)

Sin instalador, con el binario de la [página de versiones](https://github.com/bayalad7/prisma/releases/latest) o compilado desde el código (`make build-todo`, o `.\build.ps1 build-todo`).

```sh
# Linux (como root): el binario debe quedar fuera de /home.
sudo install -m 0755 prisma-linux-arm64 /usr/local/bin/prisma
sudo prisma servicio instalar     # usuario prisma, carpetas y unidad de systemd
sudo prisma servicio iniciar

# macOS
sudo install -m 0755 prisma-darwin-arm64 /usr/local/bin/prisma
sudo xattr -d com.apple.quarantine /usr/local/bin/prisma 2>/dev/null   # binario descargado sin firma
sudo prisma servicio instalar     # usuario _prisma, carpetas y LaunchDaemon
sudo prisma servicio iniciar
```

```powershell
# Windows, en PowerShell como administrador
New-Item -ItemType Directory -Force "C:\Program Files\PRISMA" | Out-Null
Copy-Item .\prisma-windows-amd64.exe "C:\Program Files\PRISMA\prisma.exe"
& "C:\Program Files\PRISMA\prisma.exe" servicio instalar   # servicio, carpetas y regla del firewall
& "C:\Program Files\PRISMA\prisma.exe" servicio iniciar
```

`prisma servicio estado | detener | reiniciar | desinstalar` controla el servicio en los tres sistemas. Los comandos se describen en [cli.md](cli.md#prisma-servicio).

Rutas por sistema:

| | Windows | Linux / Raspberry Pi | macOS |
|---|---|---|---|
| Programa | `C:\Program Files\PRISMA\prisma.exe` | `/usr/bin/prisma` (paquete) o `/usr/local/bin/prisma` | `/usr/local/bin/prisma` |
| Configuración | `C:\ProgramData\PRISMA\config.toml` | `/etc/prisma/config.toml` | `/Library/Application Support/PRISMA/config.toml` |
| Datos | `C:\ProgramData\PRISMA\datos\` | `/var/lib/prisma/` | `/Library/Application Support/PRISMA/datos/` |
| Bitácoras (10 MB × 5) | `C:\ProgramData\PRISMA\logs\` | `/var/log/prisma/` y journald | `/Library/Logs/PRISMA/` |
| Servicio corre como | LocalSystem | `prisma` | `_prisma` |

> Ejecuta los comandos de administración con el **mismo usuario que el servicio**: `sudo -u prisma prisma …` en Linux, `sudo -u _prisma prisma …` en macOS, y una PowerShell de administrador en Windows. Si otro usuario crea `prisma.db`, el servicio no podrá abrirla.

## Lista de verificación

- [ ] El equipo de PRISMA tiene IP fija, y las impresoras de red también.
- [ ] `prisma servicio estado` dice «en marcha», y sigue así después de **reiniciar el equipo**.
- [ ] El panel abre desde otra computadora de la red, y tiene contraseña guardada en un lugar seguro.
- [ ] Cada equipo que imprime abre `https://<ip>:8443/v1/salud` sin advertencia de certificado.
- [ ] El ticket de diagnóstico de cada impresora sale completo, con la regla de columnas en una sola línea.
- [ ] Cada aplicación está registrada con el nivel y las impresoras que necesita. Solo las que lo requieren tienen `abrir_cajon` o `raw`.
- [ ] No quedan solicitudes pendientes sin revisar.
- [ ] Hay un respaldo descargado del panel, guardado fuera del equipo.
