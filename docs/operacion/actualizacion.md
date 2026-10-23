# Actualización

## Saber si hay una versión nueva

- **Panel:** *Inicio* la avisa. El puente pregunta a GitHub una vez al día; no envía datos del negocio y se desactiva en *Ajustes → Actualizaciones*.
- **CLI:** `prisma actualizar --comprobar`.
- **Qué cambió:** [CHANGELOG.md](../../CHANGELOG.md) y la [página de versiones](https://github.com/bayalad7/prisma/releases).

## Actualizar

| Sistema | Cómo |
|---|---|
| Windows | Ejecuta el instalador de la versión nueva, o en PowerShell **como administrador**: `prisma actualizar`. |
| Raspberry Pi y Linux | `sudo prisma actualizar`, o vuelve a ejecutar la [línea de instalación](../instalacion/linux-raspberry-pi.md#con-una-línea). |
| macOS | Ejecuta el `.pkg` nuevo, o `sudo prisma actualizar`. |

`prisma actualizar`:

1. Busca la última versión **estable** en GitHub Releases (las preliminares, `v1.2.0-rc.1`, no se ofrecen).
2. Descarga `checksums.txt` y verifica su firma Ed25519 con la llave pública del proyecto, que va dentro del programa. Si la firma no verifica, se detiene sin tocar nada.
3. Descarga el programa de tu sistema y verifica su SHA-256 contra `checksums.txt`.
4. Detiene el servicio, reemplaza el programa y lo vuelve a iniciar. Si no puede reemplazarlo, deja el anterior y reinicia el servicio.

Opciones: `--si` no pide confirmación; `--forzar` reinstala aunque no haya versión nueva o reemplaza una compilación de desarrollo (`0.0.0-dev`, que de otro modo no se actualiza).

Al arrancar, si la versión nueva cambia la base de datos, el puente guarda antes una copia en `respaldos/` (ver [respaldo-y-restauracion.md](respaldo-y-restauracion.md)). La configuración y los datos se mantienen.

El diagrama completo está en [diagramas/28-actualizacion.md](../diagramas/28-actualizacion.md).

## Volver a la versión anterior

No hay un comando para volver atrás. Si una versión nueva falla:

1. Descarga el instalador o el programa de la versión anterior de la [página de versiones](https://github.com/bayalad7/prisma/releases) e instálalo encima.
2. Si la versión nueva **migró la base**, la anterior no la abrirá («la base de datos usa el esquema N, pero esta versión de PRISMA solo conoce hasta el M»). Restaura la copia que quedó en `respaldos/` antes de la migración, siguiendo [Restaurar sin el panel](respaldo-y-restauracion.md#restaurar-sin-el-panel). Lo que se configuró después de actualizar se pierde.
3. Reporta el problema en [GitHub Issues](https://github.com/bayalad7/prisma/issues), con la parte de la bitácora donde falló.

En Windows, la actualización deja el programa anterior como `prisma.exe.old` hasta el siguiente arranque.
