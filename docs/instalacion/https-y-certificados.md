# HTTPS y certificados

Los navegadores solo permiten que una página HTTPS llame a otra dirección HTTPS. Por eso PRISMA sirve su API por HTTPS en la red local. El modo se elige con `servidor.modo_tls` en `config.toml` ([configuracion.md](../referencia/configuracion.md)).

| Modo | Certificado | Cuándo usarlo |
|---|---|---|
| `autofirmado` (por defecto) | PRISMA lo genera al primer arranque. | Lo normal en un negocio. El instalador confía en él en el equipo del puente; en los demás equipos que imprimen hay que hacerlo una vez. |
| `archivos` | Uno tuyo: de una CA interna, de mkcert, de tu dominio, etc. | Cuando ya tienes una CA o un certificado válido para el equipo. |
| `solo_localhost_http` | Ninguno: HTTP plano, **solo** en `127.0.0.1`. | Cuando la página que imprime corre en el mismo equipo que el puente. |

PRISMA exige **TLS 1.2 o superior**. Con tu propio dominio también puedes usar Let's Encrypt ([abajo](#con-lets-encrypt-y-tu-propio-dominio)).

## Certificado autofirmado

Al primer arranque, PRISMA crea en la carpeta de datos:

- `tls/certificado.pem` (público);
- `tls/llave.pem` (privada, con permisos solo para el usuario del servicio).

El certificado:

- incluye `localhost`, el nombre del equipo (y `<nombre>.local`) y **todas sus IPs actuales**;
- usa ECDSA P-256 y dura 825 días, el máximo que aceptan macOS e iOS;
- desde 0.3.0 está **limitado a la red local**: solo vale para `localhost`, nombres `.local`, el nombre del equipo, redes privadas y sus propias IPs. Aunque un equipo confíe en él, su llave no sirve para hacerse pasar por ningún sitio de Internet ([ADR 0020](../adr/0020-confiar-en-el-certificado-del-puente-al-instalar.md)).

```sh
prisma tls info        # archivo, nombres, IPs, vencimiento, huella SHA-256 y si está limitado
prisma tls regenerar   # uno nuevo con el nombre y las IPs actuales
prisma tls confiar     # como administrador: este equipo confía en él (lo hace el instalador)
```

- **Cuándo regenerarlo:** si cambia la IP del equipo, o cuando esté por vencer. Al arrancar, el puente avisa en la bitácora si alguna IP actual no está en el certificado.
- **Sin reiniciar:** el puente en marcha toma el certificado nuevo en unos segundos.
- **Después de regenerar:** hay que volver a confiar en él en cada equipo.

### Vencimiento

- **Desde 60 días antes:** la bitácora lo avisa en cada arranque, *Inicio* del panel muestra un aviso con el botón *Ir a Ajustes*, y la fila *Vence* lo marca como «Vence en N días». Regenéralo con tiempo y vuelve a instalarlo en cada caja.
- **Si ya venció:** con `modo_tls = "autofirmado"`, el puente genera uno nuevo al arrancar, porque con el vencido ningún cliente podría conectarse. Hay que volver a confiar en él en cada equipo. Con `modo_tls = "archivos"` el puente no puede renovarlo: lo avisa en la bitácora y en el panel, y hay que reemplazar los archivos.

> Conviene que el equipo del puente tenga **IP fija**: el certificado nombra sus IPs, y las aplicaciones lo buscan por su dirección.

### En el equipo del puente: lo hace el instalador

Los instaladores de Windows, macOS y Linux (`.deb`) ejecutan `prisma tls confiar` al terminar: agregan el certificado al almacén de confianza de ese equipo, así que el panel se abre sin advertencia de seguridad en Edge, Chrome, Safari y Firefox. Al desinstalar, lo quitan (`prisma tls confiar --quitar`).

- **Si actualizaste desde 0.2.0:** tu certificado no está limitado a la red local y el instalador no confía en él. *Ajustes → HTTPS* lo avisa. Regenéralo y ejecuta `prisma tls confiar` como administrador; después, vuelve a confiar en él en los demás equipos.
- **Si lo regeneras** (panel o `prisma tls regenerar`): ejecuta otra vez `prisma tls confiar` como administrador en el equipo del puente. En Windows y Linux reemplaza al anterior. En macOS solo agrega el nuevo: para quitar los anteriores, `sudo prisma tls confiar --quitar` (macOS pide autorizarlo en una ventana) y después `sudo prisma tls confiar`.
- **Chrome y Chromium en Linux** usan además su propia base (NSS): ahí sigue haciendo falta el paso de abajo.

### Confiar en el certificado

Se hace una vez en cada equipo (o teléfono) desde el que se imprimirá; en el del puente ya lo hizo el instalador:

1. Copia `certificado.pem` del puente a ese equipo. El archivo es público, así que no hace falta protegerlo.
2. Antes de instalarlo, compara su huella con la que muestra `prisma tls info`. Así te aseguras de instalar el certificado correcto.

**Windows** (Chrome y Edge usan el almacén de Windows). En PowerShell como administrador:

```powershell
Import-Certificate -FilePath .\certificado.pem -CertStoreLocation Cert:\LocalMachine\Root
```

También puedes hacerlo con doble clic en el archivo → *Instalar certificado* → *Equipo local* → *Colocar todos los certificados en el siguiente almacén* → **Entidades de certificación raíz de confianza**.

**macOS** (Safari y Chrome usan el llavero):

```sh
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain certificado.pem
```

**Linux (Debian, Ubuntu, Raspberry Pi OS):**

```sh
sudo cp certificado.pem /usr/local/share/ca-certificates/prisma.crt
sudo update-ca-certificates
# Chrome/Chromium en Linux usa su propia base (NSS):
certutil -d sql:$HOME/.pki/nssdb -A -t "C,," -n "PRISMA" -i certificado.pem
```

**Firefox** usa en Windows y macOS las raíces del sistema, así que basta con el paso de ese sistema. Si aun así avisa (en Linux, o si se desactivó esa opción), importa el certificado: Ajustes → Privacidad y seguridad → Certificados → *Ver certificados* → *Autoridades* → *Importar*, y marca «Confiar en esta CA para identificar sitios web».

**Android:**
1. Ajustes → Seguridad → *Cifrado y credenciales* → *Instalar un certificado* → *Certificado de CA*.
2. Chrome para Android confía en los certificados de CA instalados por el usuario. Las aplicaciones nativas (no el navegador) pueden no hacerlo.

**iOS / iPadOS:**
1. Abre el archivo, que se instala como perfil, en *Ajustes → General → VPN y gestión de dispositivos*.
2. Actívalo en *Ajustes → General → Información → Ajustes de confianza de certificados*.

Para comprobarlo, abre `https://<ip-del-puente>:8443/v1/salud` en ese equipo: debe responder sin advertencia de seguridad.

## Certificado propio (`archivos`)

```toml
[servidor]
modo_tls = "archivos"
certificado = "tls/puente.pem"   # relativo a la carpeta de config.toml, o absoluto
llave = "tls/puente-llave.pem"
```

- **Validación:** `prisma config validar` comprueba que el certificado y la llave existan y formen un par válido.
- **Recarga en caliente:** el puente revisa los archivos cada pocos segundos y usa el certificado nuevo **sin reiniciar**. Sirve para renovaciones automáticas. Si los archivos nuevos son inválidos (por ejemplo, a mitad de una copia), sigue usando el anterior.
- **Con mkcert:** `mkcert -install` en cada equipo cliente, y `mkcert 192.168.1.10 caja.local localhost` en el equipo del puente.

### Con Let's Encrypt y tu propio dominio

Let's Encrypt solo emite certificados para dominios públicos: no para `localhost`, IPs privadas ni nombres `.local`. Si el negocio tiene su dominio, puede darle al puente un nombre como `caja.mitienda.mx` que apunte a la IP privada del equipo y sacar el certificado con validación por DNS:

1. En el DNS del dominio, crea un registro `A` `caja.mitienda.mx` → `192.168.1.10` (la IP fija del puente).
2. En el equipo del puente, obtén el certificado con un cliente ACME que valide por DNS (DNS-01), por ejemplo `certbot certonly --manual --preferred-challenges dns -d caja.mitienda.mx`, o `acme.sh` con el plugin de tu proveedor de DNS para que la renovación sea automática.
3. Apunta `certificado` y `llave` de `config.toml` a los archivos que genera (con `modo_tls = "archivos"`). El puente los recarga solo al renovarse.
4. Las aplicaciones usan `https://caja.mitienda.mx:8443`. Ningún equipo tiene que instalar nada, pero:
   - hace falta Internet para renovarlo, cada pocos meses;
   - algunos routers bloquean los nombres públicos que apuntan a IPs privadas (protección contra *DNS rebinding*). Si `caja.mitienda.mx` no resuelve dentro de la red, revisa esa opción del router.

## Solo localhost (`solo_localhost_http`)

```toml
[servidor]
escuchar = "127.0.0.1"           # obligatorio en este modo
modo_tls = "solo_localhost_http"
```

El puente sirve HTTP sin cifrar y solo acepta conexiones del mismo equipo. Los navegadores consideran seguro `http://localhost` y `http://127.0.0.1`, así que Chrome, Edge y Firefox permiten que una página HTTPS lo llame. Safari puede bloquearlo como contenido mixto; para Safari usa `autofirmado`.

Úsalo en un punto de venta donde la aplicación web y la impresora están en la misma PC y nadie más imprime.
