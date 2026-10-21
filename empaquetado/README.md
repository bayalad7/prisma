# Empaquetado e instaladores

Archivos para instalar PRISMA como servicio en cada sistema. Los genera el workflow de versiones ([`.github/workflows/release.yml`](../.github/workflows/release.yml)) y también se pueden construir a mano ([docs/publicar.md](../docs/publicar.md#construir-localmente)).

| Archivo | Qué es |
|---|---|
| [`config.toml`](config.toml) | Configuración de ejemplo comentada que copian los instaladores. Nunca activa `modo_desarrollo`: lo comprueba `TestConfiguracionDeLosInstaladores`. |
| [`llave-releases.pub`](llave-releases.pub) | Llave pública Ed25519 de las versiones, que se embebe en cada binario para `prisma actualizar`. Vacía hasta que se genere ([docs/publicar.md](../docs/publicar.md#una-sola-vez-la-llave-de-las-versiones)). |
| [`windows/prisma.iss`](windows/prisma.iss) | Instalador de Inno Setup: copia el programa, lo agrega al `PATH`, ejecuta `prisma servicio instalar` (servicio, firewall y carpetas), lo inicia y abre el panel. |
| [`linux/nfpm.yaml`](linux/nfpm.yaml) | Paquete `.deb` (amd64 y arm64) con nfpm. Sus [`scripts/`](linux/scripts/) crean el usuario `prisma`, habilitan el servicio y lo borran todo con `apt purge`. |
| [`linux/prisma.service`](linux/prisma.service) | Unidad de systemd del `.deb`. Es idéntica a la que genera `prisma servicio instalar`; una prueba de `internal/servicio` lo garantiza (`-actualizar` la regenera). |
| [`linux/instalar.sh`](linux/instalar.sh) | Instalador de una línea para Raspberry Pi, Debian y Ubuntu. Verifica la firma y el SHA-256 antes de instalar el `.deb`. |
| [`macos/construir-pkg.sh`](macos/construir-pkg.sh) | `.pkg` universal (Intel y Apple Silicon) con pkgbuild; firma y notarización opcionales. Sus [`scripts/`](macos/scripts/) ejecutan `prisma servicio instalar` e `iniciar`. |

La lógica del servicio (usuario, carpetas, permisos, firewall) está en el código, en [`internal/servicio`](../internal/servicio/). Los instaladores de Windows y macOS la reutilizan con `prisma servicio instalar`; el `.deb` la replica en sus scripts porque las unidades de un paquete deben ser del paquete.

Guía para quien instala: [docs/instalacion.md](../docs/instalacion.md).
