# Instalación en producción

Esta guía instala PRISMA en el equipo de un negocio (PC con Windows, Raspberry Pi o Linux, o Mac) para que arranque solo y siga funcionando después de un reinicio.

> **⏳ Estado actual.** Todavía no existen el servidor HTTPS, el panel de administración ni el comando `prisma servicio instalar`. Con lo que hay hoy puedes:
> - dejar instalados el binario, la configuración y la base de datos;
> - configurar y probar las impresoras;
> - hacer que `prisma ejecutar` arranque con el sistema.
>
> `prisma ejecutar` todavía solo mantiene la base de datos: las aplicaciones podrán imprimir por la red cuando llegue el servidor. Esta guía se actualizará entonces; los pasos marcados con ⏳ cambiarán.

## 1. Requisitos del equipo

| Sistema | Versión mínima | Binario |
|---|---|---|
| Windows | 10 / Server 2016 (64 o 32 bits) | `prisma-windows-amd64.exe` / `prisma-windows-386.exe` |
| Linux | kernel 3.2+, glibc no requerida | `prisma-linux-amd64` / `prisma-linux-arm64` |
| Raspberry Pi | Raspberry Pi OS **de 64 bits** (Pi 3, 4, 5, Zero 2) | `prisma-linux-arm64` |
| macOS | 12 Monterey (Intel o Apple Silicon) | `prisma-darwin-amd64` / `prisma-darwin-arm64` |

- **Raspberry Pi OS de 32 bits:** no hay binario oficial. Se puede compilar con `GOOS=linux GOARCH=arm GOARM=7 go build ./cmd/prisma`, pero no está probado.
- **Recursos:** PRISMA es un solo ejecutable sin dependencias. Una Raspberry Pi o una PC modesta alcanzan; el disco solo crece con el historial de trabajos, que se purga según la retención.
- **Red:** el equipo debe estar en la misma red que las impresoras de red y que los equipos desde donde se imprime. Conviene que tenga **IP fija**, porque las aplicaciones lo buscarán por su dirección.

## 2. Obtener el binario

Compílalo desde el código. En una máquina de desarrollo:

```sh
git switch main                 # versión de producción
.\build.ps1 build-todo          # Windows
make build-todo                 # Linux / macOS
```

Los binarios quedan en `dist/`. Copia al equipo del negocio el que corresponde. ⏳ Cuando haya versiones publicadas, se descargarán desde las *Releases* de GitHub.

## 3. Rutas

| | Windows | Linux / Raspberry Pi | macOS |
|---|---|---|---|
| Ejecutable (recomendado) | `C:\Program Files\PRISMA\prisma.exe` | `/usr/local/bin/prisma` | `/usr/local/bin/prisma` |
| Configuración | `C:\ProgramData\PRISMA\config.toml` | `/etc/prisma/config.toml` | `/Library/Application Support/PRISMA/config.toml` |
| Datos (`prisma.db`) | `C:\ProgramData\PRISMA\datos\` | `/var/lib/prisma/` | `/Library/Application Support/PRISMA/datos/` |
| Bitácoras | `C:\ProgramData\PRISMA\logs\` (al redirigirlas) | journald (`journalctl -u prisma`) | `/Library/Logs/PRISMA/` |

`prisma config ruta` muestra las rutas efectivas. Si `config.toml` no existe, PRISMA usa los valores por defecto, que son válidos.

> **Importante:** ejecuta los comandos de administración (`impresoras agregar`, `admin restablecer-contrasena`, …) con **el mismo usuario que el servicio**. Si root crea `prisma.db` y el servicio corre con otro usuario, el servicio no podrá abrirla.

---

## 4. Linux y Raspberry Pi

### Instalar

```sh
# Usuario de sistema sin acceso de inicio de sesión; los grupos lp y dialout dan acceso a USB y Bluetooth.
sudo useradd --system --home-dir /var/lib/prisma --shell /usr/sbin/nologin prisma
sudo usermod -aG lp,dialout prisma

sudo install -m 0755 prisma-linux-arm64 /usr/local/bin/prisma
sudo install -d -o prisma -g prisma -m 0700 /var/lib/prisma
sudo install -d -m 0755 /etc/prisma
sudo install -m 0644 empaquetado/config.toml /etc/prisma/config.toml   # ejemplo comentado

sudo -u prisma prisma config validar
sudo -u prisma prisma admin restablecer-contrasena      # anota la contraseña (⏳ se usará en el panel)
```

### Impresoras

Lee [conectores.md](conectores.md) y luego:

```sh
sudo -u prisma prisma impresoras detectar
sudo -u prisma prisma impresoras agregar --alias caja --tipo red --destino 192.168.1.50 --cajon
sudo -u prisma prisma impresoras probar caja
```

### Servicio con systemd

`/etc/systemd/system/prisma.service`:

```ini
[Unit]
Description=PRISMA, puente de impresión
After=network-online.target bluetooth.service cups.service
Wants=network-online.target

[Service]
User=prisma
Group=prisma
SupplementaryGroups=lp dialout
ExecStart=/usr/local/bin/prisma ejecutar
Restart=on-failure
RestartSec=5

# Endurecimiento: solo puede escribir en su carpeta de datos.
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/var/lib/prisma

[Install]
WantedBy=multi-user.target
```

Si usas una impresora `--tipo archivo`, agrega su carpeta a `ReadWritePaths`.

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now prisma
systemctl status prisma
journalctl -u prisma -f            # bitácoras en JSON
```

---

## 5. Windows

Abre **PowerShell como administrador**.

### Instalar

```powershell
New-Item -ItemType Directory -Force "C:\Program Files\PRISMA", "C:\ProgramData\PRISMA\logs" | Out-Null
Copy-Item .\prisma-windows-amd64.exe "C:\Program Files\PRISMA\prisma.exe"
Copy-Item .\empaquetado\config.toml "C:\ProgramData\PRISMA\config.toml"   # opcional

& "C:\Program Files\PRISMA\prisma.exe" config validar
& "C:\Program Files\PRISMA\prisma.exe" admin restablecer-contrasena
```

El binario no está firmado. Si SmartScreen lo bloquea, en las propiedades del archivo marca **Desbloquear**, o elige *Más información > Ejecutar de todas formas*.

Para no escribir la ruta completa, agrega `C:\Program Files\PRISMA` al `PATH` del sistema.

### Impresoras

Lee [conectores.md](conectores.md). Para USB, el destino es el nombre exacto de la impresora en Windows.

```powershell
prisma impresoras detectar
prisma impresoras agregar --alias caja --tipo usb --destino "POS-80" --cajon
prisma impresoras probar caja
```

### Arranque automático

⏳ `prisma servicio instalar` registrará PRISMA como servicio de Windows. Mientras tanto, una tarea programada lo inicia al encender el equipo, con la cuenta SYSTEM, y guarda las bitácoras en un archivo:

```powershell
$accion = New-ScheduledTaskAction -Execute "cmd.exe" `
  -Argument '/c ""C:\Program Files\PRISMA\prisma.exe" ejecutar >> "C:\ProgramData\PRISMA\logs\prisma.log" 2>&1"'
$disparador = New-ScheduledTaskTrigger -AtStartup
$cuenta = New-ScheduledTaskPrincipal -UserId "SYSTEM" -LogonType ServiceAccount -RunLevel Highest
$ajustes = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 3 `
  -RestartInterval (New-TimeSpan -Minutes 1) -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
Register-ScheduledTask -TaskName "PRISMA" -Action $accion -Trigger $disparador -Principal $cuenta -Settings $ajustes
Start-ScheduledTask -TaskName "PRISMA"
Get-Content "C:\ProgramData\PRISMA\logs\prisma.log" -Tail 20 -Wait
```

Para detenerlo usa `Stop-ScheduledTask -TaskName PRISMA`, y para quitarlo, `Unregister-ScheduledTask -TaskName PRISMA`.

Como la tarea corre con SYSTEM, ejecuta los comandos de administración desde una consola de administrador: escriben en la misma `C:\ProgramData\PRISMA`.

---

## 6. macOS

### Instalar

```sh
sudo install -m 0755 prisma-darwin-arm64 /usr/local/bin/prisma
sudo xattr -d com.apple.quarantine /usr/local/bin/prisma 2>/dev/null   # binario no firmado
sudo mkdir -p "/Library/Application Support/PRISMA" /Library/Logs/PRISMA
sudo prisma config validar
sudo prisma admin restablecer-contrasena
```

### Servicio con launchd

`/Library/LaunchDaemons/com.github.bayalad7.prisma.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>com.github.bayalad7.prisma</string>
  <key>ProgramArguments</key>
  <array>
    <string>/usr/local/bin/prisma</string>
    <string>ejecutar</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>/Library/Logs/PRISMA/prisma.log</string>
  <key>StandardErrorPath</key><string>/Library/Logs/PRISMA/prisma.log</string>
</dict>
</plist>
```

```sh
sudo chown root:wheel /Library/LaunchDaemons/com.github.bayalad7.prisma.plist
sudo launchctl bootstrap system /Library/LaunchDaemons/com.github.bayalad7.prisma.plist
sudo launchctl print system/com.github.bayalad7.prisma | head
tail -f /Library/Logs/PRISMA/prisma.log
```

Para detenerlo y quitarlo: `sudo launchctl bootout system/com.github.bayalad7.prisma`.

El servicio corre como root, así que ejecuta los comandos de administración con `sudo`.

---

## 7. Red y cortafuegos ⏳

Cuando exista el servidor HTTPS, PRISMA escuchará en el puerto **8443**, configurable en `config.toml`. Habrá que permitirlo **solo en la red local**:

```powershell
# Windows
New-NetFirewallRule -DisplayName "PRISMA" -Direction Inbound -Protocol TCP -LocalPort 8443 -Profile Private -Action Allow
```

```sh
# Linux con ufw
sudo ufw allow from 192.168.1.0/24 to any port 8443 proto tcp
```

Nunca expongas el puerto a Internet. PRISMA está pensado para la red local, y la seguridad de cada aplicación se configura por cliente.

## 8. Actualizar

1. Respalda la base de datos (sección 9).
2. Detén el servicio: `systemctl stop prisma`, `Stop-ScheduledTask PRISMA` o `launchctl bootout …`.
3. Reemplaza el binario.
4. Inicia el servicio. Las migraciones de la base de datos se aplican solas al arrancar.

Una versión **anterior** no abre una base ya migrada por una versión más nueva: se niega a arrancar con un error claro. Para volver atrás, restaura el respaldo hecho antes de actualizar.

## 9. Respaldo y restauración

Todo el estado está en `prisma.db`: impresoras, clientes, orígenes, historial y ajustes. `config.toml` se respalda aparte.

- **Respaldar:** detén el servicio y copia `prisma.db`. Si existen `prisma.db-wal` y `prisma.db-shm`, cópialos también. ⏳ Habrá un comando para respaldar en caliente.
- **Restaurar:** detén el servicio, reemplaza los archivos e inícialo.

La base contiene hashes de llaves y de la contraseña del panel. Guarda los respaldos en un lugar con acceso restringido.

## 10. Desinstalar

| Sistema | Pasos |
|---|---|
| Linux | `sudo systemctl disable --now prisma`; borra `/etc/systemd/system/prisma.service`, `/usr/local/bin/prisma`, `/etc/prisma` y `/var/lib/prisma`; `sudo userdel prisma`. |
| Windows | `Unregister-ScheduledTask PRISMA`; borra `C:\Program Files\PRISMA` y `C:\ProgramData\PRISMA`. |
| macOS | `sudo launchctl bootout system/com.github.bayalad7.prisma`; borra el plist, `/usr/local/bin/prisma`, `/Library/Application Support/PRISMA` y `/Library/Logs/PRISMA`. |

Las carpetas de datos contienen el historial y la configuración: bórralas solo si ya no las necesitas.

## Lista de verificación

- [ ] `prisma version` muestra la versión esperada.
- [ ] `prisma config validar` da «La configuración es válida».
- [ ] El equipo tiene IP fija, y las impresoras de red también.
- [ ] `prisma impresoras probar <alias>` da «en línea» para cada impresora.
- [ ] Hay contraseña de administrador (`prisma admin restablecer-contrasena`) y está guardada en un lugar seguro.
- [ ] El servicio arranca solo tras reiniciar el equipo (revisa las bitácoras).
- [ ] `modo_desarrollo` está en `false`.
