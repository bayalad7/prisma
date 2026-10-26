# Instalar PRISMA en Windows

Para Windows 10 u 11 de 64 bits (también Windows Server 2016 o posterior). En Windows de 32 bits usa `PRISMA-<versión>-windows-386-setup.exe`.

1. Descarga `PRISMA-<versión>-windows-amd64-setup.exe` de la [página de versiones](https://github.com/bayalad7/prisma/releases/latest).
2. Ábrelo. Si Windows muestra **«Windows protegió su PC»**, pulsa *Más información → Ejecutar de todas formas* (ver [SmartScreen](problemas-comunes.md#windows-smartscreen-o-el-antivirus-bloquean-el-instalador)).
3. Acepta los permisos de administrador y sigue el asistente.
4. Al terminar, deja marcada **«Abrir el panel de administración»** y pulsa *Finalizar*.
5. El panel se abre sin advertencias: el instalador hizo que este equipo confíe en el certificado del puente. Si el navegador dice que **la conexión no es privada** (por ejemplo, porque actualizaste desde 0.2.0), revisa *Ajustes → HTTPS* después de entrar con *Configuración avanzada → Continuar a localhost*.
6. Crea la **contraseña del panel** (al menos 10 caracteres) y guárdala en un lugar seguro.

Qué hizo el instalador:

- copió PRISMA a `C:\Program Files\PRISMA` y lo agregó al `PATH`, así que el comando `prisma` funciona en PowerShell;
- creó el servicio de Windows **«PRISMA - Puente de Impresión Térmica»**, que arranca con el equipo y se reinicia solo si falla (se ve en *Servicios*, `services.msc`);
- abrió el puerto 8443 en el Firewall de Windows, solo para la red local;
- agregó el certificado del puente a las raíces de confianza de **este** equipo (`prisma tls confiar`). El certificado está limitado a la red local, así que no sirve para ningún sitio de Internet. Al desinstalar, se quita;
- guardó configuración, datos y bitácoras en `C:\ProgramData\PRISMA`. Solo los administradores pueden leer los datos.

Para entrar al panel desde otra computadora de la red, abre `https://<ip-de-este-equipo>:8443/admin`. Si todavía no creaste la contraseña, el panel te pedirá un código: genéralo en este equipo, en una PowerShell **como administrador**, con `prisma admin codigo-inicial`.

## Comandos útiles

En una PowerShell **como administrador**:

| Para… | Comando |
|---|---|
| Ver si está en marcha | `prisma servicio estado` (no necesita administrador) |
| Reiniciarlo | `prisma servicio reiniciar` |
| Ver la bitácora | `Get-Content C:\ProgramData\PRISMA\logs\prisma.log -Tail 50 -Wait` |
| Saber la IP del equipo | `ipconfig` |

## Desinstalar

*Configuración → Aplicaciones → PRISMA → Desinstalar*. Se conserva `C:\ProgramData\PRISMA`: bórrala si ya no la necesitas.

Siguiente paso: [después de instalar](README.md#después-de-instalar).
