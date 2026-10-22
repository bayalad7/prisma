# Instalar PRISMA en Raspberry Pi y Linux

Para Raspberry Pi OS **de 64 bits** (Pi 3, 4, 5 o Zero 2) y para Debian 11, Ubuntu 20.04 o posteriores, de 64 bits.

## Con una línea

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

## Con el paquete .deb

Descarga de la [página de versiones](https://github.com/bayalad7/prisma/releases/latest) el `.deb` de tu equipo: `arm64` para la Raspberry Pi, `amd64` para una PC. Instálalo con:

```sh
sudo apt install ./prisma_<versión>_arm64.deb
```

## Después de instalar

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

## Desinstalar

`sudo apt remove prisma` conserva los datos, la configuración y el usuario. `sudo apt purge prisma` borra todo.

Siguiente paso: [después de instalar](README.md#después-de-instalar).
