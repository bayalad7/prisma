# Problemas comunes de instalación

Problemas al instalar, al arrancar el servicio o al llegar al puente por la red. Los de impresión y del panel están en [administracion/problemas-comunes.md](../administracion/problemas-comunes.md); los que ve una aplicación, en [integradores/problemas-comunes.md](../integradores/problemas-comunes.md).

## Windows: SmartScreen o el antivirus bloquean el instalador

PRISMA es un programa nuevo y Windows todavía no lo «conoce».

- **SmartScreen** («Windows protegió su PC»): pulsa *Más información → Ejecutar de todas formas*.
- **El antivirus lo borra o lo pone en cuarentena:** es un falso positivo habitual con programas recién publicados. Antes de permitirlo, comprueba que el archivo sea el original: en PowerShell, `Get-FileHash .\PRISMA-…-setup.exe` debe dar el mismo valor que su línea en `checksums.txt`, en la página de la versión. Luego restáuralo desde la cuarentena y agrega una exclusión para `C:\Program Files\PRISMA`.
- Si la versión está firmada digitalmente, en *Propiedades → Firmas digitales* del archivo aparece el editor.

## El panel o las aplicaciones no llegan desde otro equipo

1. En el equipo de PRISMA, `https://localhost:8443/admin` debe abrir. Si no abre, PRISMA no está en marcha: revisa [que el servicio arranque](#el-servicio-no-arranca).
2. Desde el otro equipo, prueba `https://<ip>:8443/v1/salud`.
   - **No carga:** es la red o el firewall.
   - **Windows:** el instalador abrió el puerto solo para la subred local. Si la caja está en otra subred (por ejemplo, otra red Wi-Fi), agrega una regla en *Firewall de Windows → Configuración avanzada → Reglas de entrada*.
   - **Linux con `ufw`:** `sudo ufw allow from 192.168.1.0/24 to any port 8443 proto tcp`, con tu red.
   - **Macs con el firewall activado:** permite las conexiones entrantes de `prisma` en *Ajustes del Sistema → Red → Firewall → Opciones*.
   - **Redes de invitados:** suelen aislar a los equipos entre sí. Conecta la caja a la red principal.
3. Si responde con una advertencia de certificado, instala el certificado en ese equipo ([instalacion/https-y-certificados.md](https-y-certificados.md#confiar-en-el-certificado)).

Nunca abras el puerto 8443 hacia Internet en el router.

## El navegador dice que la conexión no es privada

El puente usa un certificado propio. En el equipo donde lo instalaste, el instalador ya confía en él; si aun así avisa, el certificado es de antes de 0.3.0: *Ajustes → HTTPS* explica cómo regenerarlo y ejecutar `prisma tls confiar`. En los demás equipos, el navegador todavía no lo conoce. Para entrar al panel basta con *Configuración avanzada → Continuar*. Para que las **aplicaciones** puedan imprimir desde ese equipo, en cambio, hay que instalar el certificado: el aviso aceptado solo vale para navegar ([instalacion/https-y-certificados.md](https-y-certificados.md#confiar-en-el-certificado)).

Si el aviso aparece en equipos que antes funcionaban:

- **Puente 0.3.1 o posterior:** se creó una autoridad local nueva, porque cambió el nombre del equipo o se pidió una; *Ajustes → HTTPS* lo dijo al regenerar. Descarga `prisma.crt` e instálalo de nuevo en esos equipos. Un cambio de IP ya no obliga a nada: reinicia el servicio o regenera el certificado.
- **Puente anterior:** cambió la IP del equipo de PRISMA. Regenera el certificado en *Ajustes → HTTPS* y vuelve a instalarlo en los equipos.

Para evitarlo, dale IP fija al equipo del puente.

## Linux: la impresora USB no responde (permiso denegado)

El servicio corre con el usuario `prisma`, que pertenece al grupo `lp`. Si la impresora aparece como `/dev/usb/lp0`, comprueba:

```sh
ls -l /dev/usb/lp0        # debe ser del grupo lp
id prisma                 # debe incluir lp
```

Si CUPS ya usa la impresora, el puerto aparece ocupado. En ese caso agrégala como cola de CUPS en modo raw. Ambos casos están en [instalacion/impresoras.md](impresoras.md#linux-y-raspberry-pi).

## Bluetooth: la impresora no conecta o aparece «ocupada»

- La impresora debe estar **emparejada** con el equipo de PRISMA y tener su puerto serie: `COM5` en Windows, `/dev/rfcomm0` en Linux.
- Casi todas las impresoras Bluetooth aceptan **un solo equipo conectado a la vez**. Si un teléfono o una tableta se conectó a ella, PRISMA no puede: desconéctalos.
- En Linux, el usuario `prisma` necesita el grupo `dialout` (el instalador lo agrega).

Detalles por sistema en [instalacion/impresoras.md](impresoras.md#impresoras-bluetooth).

## El servicio no arranca

| Sistema | Ver el error |
|---|---|
| Windows | `C:\ProgramData\PRISMA\logs\prisma.log`. Si el servicio ni siquiera llega a escribir la bitácora, el *Visor de eventos* (Registros de Windows → Sistema) muestra el aviso del Administrador de servicios. |
| Linux | `journalctl -u prisma -n 50` |
| macOS | `/Library/Logs/PRISMA/prisma.log` y `/Library/Logs/PRISMA/launchd.err` |

La causa más común es que **otro programa ya usa el puerto 8443**. Cambia `puerto` en `config.toml` y reinicia: `prisma servicio reiniciar`, como administrador. La segunda es un error en `config.toml`: `prisma config validar` lo muestra con su número de línea.

## «la configuración tiene N problema(s)»

`config.toml` tiene claves mal escritas o valores inválidos. El mensaje lista cada clave con su problema. Corrígelas y comprueba el resultado con `prisma config validar`. La referencia completa está en [configuracion.md](../referencia/configuracion.md).

## «datos.ruta: debe ser una ruta absoluta»

Usa una ruta completa. En Windows, escríbela entre comillas simples: `ruta = 'D:\PRISMA\datos'`.

## «la base de datos usa el esquema N, pero esta versión de PRISMA solo conoce hasta el M»

Una versión más nueva de PRISMA ya migró la base. Instala esa versión o una posterior, o restaura el respaldo hecho antes de actualizar.

## El servicio no puede abrir `prisma.db` (permiso denegado)

La base la creó otro usuario, normalmente root al ejecutar un comando con `sudo`. Hay que devolvérsela al usuario del servicio:

```sh
sudo chown -R prisma:prisma /var/lib/prisma                                   # Linux
sudo chown -R _prisma:_prisma "/Library/Application Support/PRISMA/datos"    # macOS
```

A partir de entonces, ejecuta los comandos de administración como ese usuario: `sudo -u prisma prisma …` en Linux, `sudo -u _prisma prisma …` en macOS.

## Windows bloquea el ejecutable (SmartScreen) o macOS dice que no se puede abrir

El binario no está firmado:

- **Windows:** en las *Propiedades* del archivo, marca **Desbloquear**.
- **macOS:** ejecuta `sudo xattr -d com.apple.quarantine /usr/local/bin/prisma`.

## «prisma imprimir solo está disponible con [dev] modo_desarrollo = true»

Es intencional: en producción, las aplicaciones imprimen por la API con su autorización. Para pruebas, usa una configuración de desarrollo aparte (`--config config.dev.toml`).
