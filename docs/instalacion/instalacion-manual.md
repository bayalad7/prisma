# Instalación manual (avanzado)

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

`prisma servicio estado | detener | reiniciar | desinstalar` controla el servicio en los tres sistemas. Los comandos se describen en [cli.md](../referencia/cli.md#prisma-servicio).


Las rutas de configuración, datos y bitácoras de cada sistema están en [operacion/](../operacion/README.md#dónde-está-cada-cosa).

> Ejecuta los comandos de administración con el **mismo usuario que el servicio**: `sudo -u prisma prisma …` en Linux, `sudo -u _prisma prisma …` en macOS, y una PowerShell de administrador en Windows. Si otro usuario crea `prisma.db`, el servicio no podrá abrirla.
