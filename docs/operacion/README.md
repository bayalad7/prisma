# Operación

Cómo mantener un puente en marcha: dónde está cada cosa, cómo leer la bitácora, respaldar, actualizar, vigilar y recuperarse de una falla. Para instalarlo, ver [instalacion/](../instalacion/README.md).

| Documento | Para qué |
|---|---|
| [bitacoras.md](bitacoras.md) | Leer la bitácora: dónde está, formato y qué eventos registra. |
| [respaldo-y-restauracion.md](respaldo-y-restauracion.md) | Respaldar la base y restaurarla. |
| [actualizacion.md](actualizacion.md) | Instalar una versión nueva y volver atrás. |
| [monitoreo.md](monitoreo.md) | Saber si el puente y las impresoras funcionan. |
| [recuperacion.md](recuperacion.md) | Qué hacer ante cada falla grave. |

## Dónde está cada cosa

| | Windows | Linux / Raspberry Pi | macOS |
|---|---|---|---|
| Programa | `C:\Program Files\PRISMA\prisma.exe` | `/usr/bin/prisma` (paquete) o `/usr/local/bin/prisma` | `/usr/local/bin/prisma` |
| Configuración | `C:\ProgramData\PRISMA\config.toml` | `/etc/prisma/config.toml` | `/Library/Application Support/PRISMA/config.toml` |
| Datos (`prisma.db`, `tls/`, `respaldos/`) | `C:\ProgramData\PRISMA\datos\` | `/var/lib/prisma/` | `/Library/Application Support/PRISMA/datos/` |
| Bitácoras (10 MB × 5) | `C:\ProgramData\PRISMA\logs\` | `/var/log/prisma/` y journald | `/Library/Logs/PRISMA/` |
| Servicio | «PRISMA - Puente de Impresión Térmica» (`PRISMA`) | `prisma.service` (systemd) | `com.github.bayalad7.prisma` (LaunchDaemon) |
| Corre como | LocalSystem | usuario `prisma` (grupos `lp`, `dialout`) | usuario oculto `_prisma` |
| Puerto | 8443/TCP (HTTPS: API y panel) | igual | igual |

`prisma config ruta` muestra qué `config.toml` usa el programa, y `prisma config validar` lo revisa.

> Ejecuta los comandos de administración con el **mismo usuario que el servicio**: `sudo -u prisma prisma …` en Linux, `sudo -u _prisma prisma …` en macOS, y una PowerShell de administrador en Windows. Si otro usuario crea o modifica `prisma.db`, el servicio no podrá abrirla.

## Controlar el servicio

| Acción | Comando (como administrador) |
|---|---|
| Ver el estado | `prisma servicio estado` (no requiere administrador) |
| Iniciar · detener · reiniciar | `prisma servicio iniciar` · `detener` · `reiniciar` |
| Volver a registrarlo con la configuración actual | `prisma servicio instalar` |

En Linux también funcionan `systemctl status|restart prisma`; en Windows, `services.msc`. Detalles en [referencia/cli.md](../referencia/cli.md#prisma-servicio).

## Qué hace el puente solo

- **Al arrancar:** respalda la base si la versión trae migraciones y la migra, marca como `PUENTE_REINICIADO` los trabajos que quedaron en cola (y libera sus `id`), regenera el certificado autofirmado si venció y avisa si está por vencer.
- **Cada 30 s:** prueba cada impresora que no esté imprimiendo.
- **Cada hora:** purga el historial más viejo que la retención (30 días por defecto), los `id` de idempotencia vencidos y las solicitudes de origen sin actividad.
- **Una vez al día** (desactivable en *Ajustes*): pregunta a GitHub si hay una versión nueva. Nunca instala nada solo.
- **Ante un fallo:** el sistema reinicia el servicio a los 5 s.
