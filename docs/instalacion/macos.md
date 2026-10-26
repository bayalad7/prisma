# Instalar PRISMA en macOS

Para macOS 12 Monterey o posterior, en Mac con Intel o con Apple Silicon.

1. Descarga `PRISMA-<versión>-macos.pkg` de la [página de versiones](https://github.com/bayalad7/prisma/releases/latest). Sirve para Mac con Intel y con Apple Silicon.
2. Ábrelo y sigue el instalador. Si macOS dice que **no puede verificar el desarrollador**, abre *Ajustes del Sistema → Privacidad y seguridad* y pulsa *Abrir de todas formas* junto al aviso de PRISMA.
3. Abre `https://localhost:8443/admin` en Safari o Chrome y crea la contraseña del panel. No hay aviso de certificado: el instalador hizo que este Mac confíe en el del puente (`prisma tls confiar`), que está limitado a la red local. Si actualizaste desde 0.2.0 y aparece un aviso, revisa *Ajustes → HTTPS*.

El instalador crea el usuario oculto `_prisma` y un servicio de sistema (LaunchDaemon) que arranca con la Mac y se reinicia si falla. Configuración y datos: `/Library/Application Support/PRISMA`. Bitácoras: `/Library/Logs/PRISMA`, también visibles en la app **Consola**.

Para generar el código del panel desde la Terminal: `sudo -u _prisma prisma admin codigo-inicial`.

## Comandos útiles

| Para… | Comando |
|---|---|
| Ver si está en marcha | `prisma servicio estado` (no necesita `sudo`) |
| Reiniciarlo | `sudo prisma servicio reiniciar` |
| Ver la bitácora | `tail -f /Library/Logs/PRISMA/prisma.log` |

## Desinstalar

`sudo prisma tls confiar --quitar && sudo prisma servicio desinstalar && sudo rm /usr/local/bin/prisma`. Al quitar el certificado, macOS pide autorizar el cambio en una ventana. Se conservan `/Library/Application Support/PRISMA` y `/Library/Logs/PRISMA`.

Siguiente paso: [después de instalar](README.md#después-de-instalar).
