# Publicar una versión

Para quien mantiene PRISMA. Una etiqueta `v*` dispara [`.github/workflows/release.yml`](../.github/workflows/release.yml), que compila, empaqueta, prueba cada instalador en su sistema, firma los checksums y crea la versión en GitHub Releases.

## Una sola vez: la llave de las versiones

`prisma actualizar` y el instalador de Linux solo instalan lo que viene firmado con la llave Ed25519 del proyecto.

- **La llave privada** vive solo en un secreto de GitHub.
- **La pública** está en el repositorio y se embebe en cada binario.

1. Genera el par en un equipo de confianza, nunca en el CI:

   ```sh
   openssl genpkey -algorithm ed25519 -out llave-releases.pem
   openssl pkey -in llave-releases.pem -pubout -outform DER | tail -c 32 | base64 > empaquetado/llave-releases.pub
   ```

2. Pon la misma llave pública en el instalador de Linux, en `LLAVE_RELEASES="…"` de [`empaquetado/linux/instalar.sh`](../empaquetado/linux/instalar.sh). Una prueba de `internal/actualizacion` comprueba que coincidan.
3. En GitHub, *Settings → Secrets and variables → Actions*, crea el secreto `PRISMA_LLAVE_RELEASES` con el contenido completo de `llave-releases.pem`.
4. Guarda `llave-releases.pem` fuera del repositorio, en un gestor de contraseñas y con una copia de respaldo. **Si se pierde,** las versiones nuevas no se podrán instalar con `prisma actualizar` desde las ya instaladas: habrá que reinstalar con el instalador. **Si se filtra,** cualquiera podría publicar una actualización que los puentes aceptarían.
5. Haz commit de `empaquetado/llave-releases.pub` y `instalar.sh`.

Sin `empaquetado/llave-releases.pub`, el workflow se niega a publicar.

## Secretos opcionales

| Secreto | Para qué |
|---|---|
| `NPM_TOKEN` | Publica el SDK (`sdk/js`) en npm cuando su versión de `package.json` todavía no existe. Sin él, se omite. |
| `WINDOWS_PFX_BASE64`, `WINDOWS_PFX_CLAVE` | Firma de código de Windows: el certificado `.pfx` en base64 y su contraseña. Firma los `.exe` y los instaladores con `signtool`. |
| `APPLE_CERTIFICADOS_P12_BASE64`, `APPLE_CERTIFICADOS_CLAVE` | Certificados *Developer ID Application* y *Developer ID Installer* exportados en un `.p12`. |
| `APPLE_FIRMA_APP`, `APPLE_FIRMA_INSTALADOR` | Nombres de esas identidades, por ejemplo `Developer ID Application: Mi Empresa (ABCDE12345)`. |
| `APPLE_ID`, `APPLE_EQUIPO`, `APPLE_CLAVE_APP` | Notarización: Apple ID, Team ID y una contraseña de app. |

Sin firma de código, todo funciona igual, pero Windows muestra SmartScreen y macOS pide *Abrir de todas formas*. La guía de instalación lo explica.

## Cada versión

1. En `develop`, con el CI en verde, actualiza la documentación y, si cambió el SDK, la versión de `sdk/js/package.json` y `VERSION_SDK` en `sdk/js/src/prisma.mjs` (`npm run construir`).
2. Lleva `develop` a `main` con un pull request.
3. Etiqueta `main` y empújala:

   ```sh
   git switch main && git pull
   git tag -a v1.2.0 -m "PRISMA 1.2.0"
   git push origin v1.2.0
   ```

   Una etiqueta con guion (`v1.2.0-rc.1`) se publica como versión preliminar. `prisma actualizar` y el aviso del panel no la ofrecen, porque consultan solo la última versión estable.

El workflow publica:

| Archivo | Qué es |
|---|---|
| `prisma-<so>-<arq>[.exe]` | Binarios de las 6 plataformas, con versión, commit, fecha y llave embebidos. |
| `prisma_<versión>_amd64.deb`, `prisma_<versión>_arm64.deb` | Paquetes para Debian, Ubuntu y Raspberry Pi OS. |
| `PRISMA-<versión>-windows-amd64-setup.exe`, `…-386-setup.exe` | Instaladores de Windows. |
| `PRISMA-<versión>-macos.pkg` | Instalador universal de macOS. |
| `checksums.txt`, `checksums.txt.sig` | SHA-256 de todo lo anterior y su firma Ed25519, en base64. |

Antes de publicar, cada instalador se prueba en su sistema:

- el `.deb` se instala en Ubuntu y se purga;
- el instalador de Windows se ejecuta en modo silencioso, y se comprueba que el servicio arranque y que el panel responda; luego se desinstala;
- el `.pkg` se instala en macOS y se comprueba el LaunchDaemon y el panel.

## Construir localmente

| Paso | Linux / macOS | Windows |
|---|---|---|
| Binarios | `make build-todo VERSION=1.2.0` | `$env:VERSION="1.2.0"; .\build.ps1 build-todo` |
| Paquetes .deb | `make deb VERSION=1.2.0` | `.\build.ps1 deb` |
| Instaladores de Windows | — | `.\build.ps1 instalador` (requiere [Inno Setup 6](https://jrsoftware.org/isinfo.php)) |
| Paquete de macOS | `make pkg VERSION=1.2.0` (solo en macOS) | — |
| Checksums | `make checksums` | `.\build.ps1 checksums` |

La firma de `checksums.txt` se hace solo en el CI, con la llave privada:

```sh
openssl pkeyutl -sign -rawin -inkey llave-releases.pem -in checksums.txt | base64 -w0 > checksums.txt.sig
```
