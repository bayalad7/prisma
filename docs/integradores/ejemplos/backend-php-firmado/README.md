# Ejemplo: pedido firmado por un backend PHP (nivel 3)

Una página de pedidos en la que **el backend arma y firma el ticket**. El navegador solo elige productos y cantidades; si alguien altera el documento firmado antes de enviarlo, el puente lo rechaza con `FIRMA_INVALIDA`.

```
navegador ──(productos)──► firmar.php ──(documento firmado)──► navegador ──► puente PRISMA
                            arma el ticket con                  imprimirDocumento()
                            precios del servidor y firma
```

| Archivo | Qué hace |
|---|---|
| `index.php` | Página del pedido. Usa el SDK desde el CDN y solo conoce la dirección del puente. |
| `firmar.php` | Arma el ticket con el catálogo del servidor y lo firma con [`PrismaFirmador.php`](../../../../sdk/firmadores/php/PrismaFirmador.php). |
| `config.php` | Dirección del puente, impresora, id del cliente y ruta de la llave privada (también por variables de entorno). |

## Requisitos

- PHP 8.2 o superior con la extensión `sodium` (viene activada en casi todas las instalaciones).
- Un puente PRISMA en la red y acceso a su panel.

## Probarlo

1. **Genera el par de llaves** (en la carpeta del ejemplo). La carpeta `llaves/` está en `.gitignore`:

   ```sh
   prisma dev llaves --salida llaves
   ```

   Sin la CLI de PRISMA, con OpenSSL:

   ```sh
   mkdir llaves
   openssl genpkey -algorithm ed25519 -out llaves/privada.pem
   openssl pkey -in llaves/privada.pem -pubout -out llaves/publica.pem
   ```

2. **Registra la aplicación en el panel del puente** (*Aplicaciones → Agregar aplicación*):
   - Nivel: **3 — Firma**.
   - Llave pública: pega el contenido de `llaves/publica.pem`.
   - Origen: `http://localhost:8000`, la dirección de esta página. Cuando el navegador envía el documento, el puente identifica a la aplicación por su origen.
   - Impresoras: la que vayas a usar.

   Anota el id de la aplicación (`cli_…`).

3. **Arranca el servidor de PHP** con tus datos:

   ```sh
   PRISMA_URL=https://192.168.1.10:8443 PRISMA_CLIENTE_ID=cli_… PRISMA_IMPRESORA=caja php -S localhost:8000
   ```

   En PowerShell:

   ```powershell
   $env:PRISMA_URL = "https://192.168.1.10:8443"; $env:PRISMA_CLIENTE_ID = "cli_…"; php -S localhost:8000
   ```

4. Abre `http://localhost:8000` en un equipo de la red del negocio que confíe en el certificado del puente, y pulsa **Imprimir pedido**.

## Antes de llevarlo a producción

- Saca la llave privada de la carpeta pública del sitio y léela de un gestor de secretos o de una variable de entorno.
- Exige la sesión del usuario y un token CSRF en `firmar.php`: hoy cualquiera que lo llame obtiene un ticket firmado.
- Si tu backend está en la misma red que el puente, también puede enviar el documento directamente, sin pasar por el navegador. Ver [firmar-documentos.md](../../firmar-documentos.md#enviar-desde-el-backend).
