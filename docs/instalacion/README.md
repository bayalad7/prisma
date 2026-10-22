# Instalar PRISMA

Esta guía instala PRISMA en el equipo del negocio: una PC con Windows, una Raspberry Pi o un equipo con Linux, o una Mac. Al terminar, PRISMA:

- arranca solo cada vez que se enciende el equipo, sin que nadie inicie sesión;
- se vuelve a levantar solo si llega a fallar;
- se administra desde el navegador, en el **panel**: `https://<ip-del-equipo>:8443/admin`.

Los instaladores se descargan de la [página de versiones](https://github.com/bayalad7/prisma/releases/latest). La versión actual es **0.1.0**.

## Antes de empezar

- **Un equipo que quede encendido** mientras el negocio imprime. Sirve uno modesto: PRISMA ocupa poco.
- **En la misma red** que las impresoras de red y que las cajas, tabletas o teléfonos desde donde se imprime.
- **Con IP fija.** Las aplicaciones buscan a PRISMA por su dirección, así que, si cambia, dejan de encontrarlo. Se fija en el router (*reserva de DHCP*) o en la configuración de red del equipo.
- **Permisos de administrador** para instalar.

| Sistema | Versión mínima | Qué descargar |
|---|---|---|
| Windows | 10 u 11, de 64 bits (también Server 2016 o posterior) | `PRISMA-<versión>-windows-amd64-setup.exe` |
| Windows de 32 bits | 10 | `PRISMA-<versión>-windows-386-setup.exe` |
| Raspberry Pi | Raspberry Pi OS **de 64 bits** (Pi 3, 4, 5 o Zero 2) | se instala con una línea ([guía](linux-raspberry-pi.md)) |
| Linux | Debian 11, Ubuntu 20.04 o posteriores, de 64 bits | `prisma_<versión>_amd64.deb` o `_arm64.deb` |
| macOS | 12 Monterey o posterior (Intel o Apple Silicon) | `PRISMA-<versión>-macos.pkg` |

## Instalar

| Sistema | Guía |
|---|---|
| Windows | [windows.md](windows.md) |
| Raspberry Pi y Linux | [linux-raspberry-pi.md](linux-raspberry-pi.md) |
| macOS | [macos.md](macos.md) |
| Sin instalador, o compilado desde el código | [instalacion-manual.md](instalacion-manual.md) |

Si algo falla, ver [problemas-comunes.md](problemas-comunes.md).

## Después de instalar

1. **Confía en el certificado** en cada equipo, tableta o teléfono desde el que se imprimirá, para que el navegador deje de mostrar la advertencia. Se descarga en el panel (*Ajustes → HTTPS → Descargar certificado*) y se instala como explica [https-y-certificados.md](https-y-certificados.md#confiar-en-el-certificado).
2. **Agrega las impresoras** en el panel, en *Impresoras*, e imprime el **ticket de diagnóstico** de cada una. Si es USB o Bluetooth, revisa antes [impresoras.md](impresoras.md).
3. **Autoriza las aplicaciones** que van a imprimir en *Aplicaciones*, o aprueba su página en *Solicitudes* cuando intenten imprimir. Todo el panel está explicado en [administracion/](../administracion/README.md).

Para mantenerlo después (bitácoras, respaldos, actualizaciones y recuperación), ver [operacion/](../operacion/README.md).

## Lista de verificación

- [ ] El equipo de PRISMA tiene IP fija, y las impresoras de red también.
- [ ] `prisma servicio estado` dice «en marcha», y sigue así después de **reiniciar el equipo**.
- [ ] El panel abre desde otra computadora de la red, y tiene contraseña guardada en un lugar seguro.
- [ ] Cada equipo que imprime abre `https://<ip>:8443/v1/salud` sin advertencia de certificado.
- [ ] El ticket de diagnóstico de cada impresora sale completo, con la regla de columnas en una sola línea.
- [ ] Cada aplicación está registrada con el nivel y las impresoras que necesita. Solo las que lo requieren tienen `abrir_cajon` o `raw`.
- [ ] No quedan solicitudes pendientes sin revisar.
- [ ] Hay un respaldo descargado del panel, guardado fuera del equipo.
