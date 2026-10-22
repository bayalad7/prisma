# Firmar documentos (nivel 3)

En el nivel 3 tu backend firma cada ticket con una llave privada Ed25519. El puente solo guarda la llave **pública**: puede comprobar que el ticket viene de ti y que nadie lo cambió, pero no puede firmar en tu nombre.

Hay firmadores de referencia, sin más dependencias que la criptografía estándar de cada lenguaje, en [`sdk/firmadores`](../../sdk/firmadores/):

| Lenguaje | Archivo | Requisitos |
|---|---|---|
| PHP | [`php/PrismaFirmador.php`](../../sdk/firmadores/php/PrismaFirmador.php) | PHP 8.2+ con la extensión `sodium` |
| Node.js | [`node/firmador.mjs`](../../sdk/firmadores/node/firmador.mjs) | Node.js 18+ |
| Python | [`python/prisma_firmador.py`](../../sdk/firmadores/python/prisma_firmador.py) | Python 3.9+ y `pip install cryptography` |

Cópialo a tu proyecto. Los tres pasan las mismas pruebas que el puente: con la llave del [vector fijo](../../testdata/vector-firma/LEEME.md) producen exactamente su firma.

## 1. Crea el par de llaves

Con la CLI de PRISMA:

```sh
prisma dev llaves --salida llaves
```

O con OpenSSL:

```sh
openssl genpkey -algorithm ed25519 -out privada.pem
openssl pkey -in privada.pem -pubout -out publica.pem
```

- `privada.pem` se queda **solo** en tu backend, como cualquier otro secreto: fuera del repositorio y de la carpeta pública del sitio.
- `publica.pem` se la das al dueño del puente.

## 2. Registra la aplicación

El dueño del puente, en el panel (*Aplicaciones → Agregar aplicación*):

1. Elige el nivel **3 — Firma** y pega `publica.pem`. El panel valida la llave y muestra su huella (SHA-256 de los 32 bytes de la llave). Para compararla con la tuya: `openssl pkey -pubin -in publica.pem -outform DER | tail -c 32 | sha256sum`; deben coincidir los primeros 16 caracteres (el panel los muestra en mayúsculas y separados por dos puntos).
2. Si el navegador entregará los documentos, agrega el **origen** de tu página.
3. Elige las impresoras y, si hacen falta, las operaciones sensibles.

El panel muestra el id de la aplicación (`cli_…`): lo necesita tu backend.

## 3. Firma en tu backend

Cada firmador recibe el documento sin firmar y devuelve uno nuevo con cuatro campos más:

- `cliente_id`;
- `emitido`, la hora actual en UTC;
- `expira`, 90 s después por defecto, 120 s como máximo;
- `firma`.

Si el documento no trae `id`, le asigna un UUID v4.

**PHP:**

```php
require 'PrismaFirmador.php';

$documento = [
    'version' => '1',
    'impresora' => 'caja',
    'operaciones' => [
        ['tipo' => 'texto', 'valor' => 'Pedido #812', 'negrita' => true],
        ['tipo' => 'par', 'izquierda' => 'TOTAL', 'derecha' => '$249.00'],
        ['tipo' => 'cortar'],
    ],
];
$firmado = PrismaFirmador::firmar($documento, 'cli_7Hq2MZp4XwR9', file_get_contents('/ruta/segura/privada.pem'));
echo json_encode($firmado, JSON_UNESCAPED_UNICODE);
```

**Node.js:**

```js
import { readFileSync } from "node:fs";
import { firmar } from "./firmador.mjs";

const firmado = firmar(documento, {
  clienteId: "cli_7Hq2MZp4XwR9",
  llavePrivada: readFileSync("/ruta/segura/privada.pem", "utf8"),
});
```

En Node puedes armar el documento con el SDK: `new Documento("caja").texto("Pedido #812").cortar().json()`.

**Python:**

```python
from prisma_firmador import firmar

with open("/ruta/segura/privada.pem", "rb") as f:
    firmado = firmar(documento, "cli_7Hq2MZp4XwR9", f.read())
```

### Reglas

- **Firma justo antes de enviar.** El documento vence a los `expira` segundos. El puente acepta 30 s de diferencia de reloj, así que mantén la hora del servidor sincronizada (NTP).
- **Un id por ticket.** Si reintentas, reenvía el **mismo** documento firmado: si ya había llegado, el puente responde `duplicado: true` y no imprime dos veces.
- **Si venció antes de enviarlo, fírmalo de nuevo con el mismo `id`.** Si el puente nunca lo recibió, o si su impresión falló, lo imprime. Si ya lo había aceptado, responde `409 ID_REUTILIZADO`: ese ticket ya salió o está en cola, así que no lo reenvíes con otro `id` (ver [contrato §8](../contrato-documento-v1.md#8-idempotencia)).
- **No modifiques el documento después de firmarlo.** Ni siquiera el orden o el formato importan, pero cualquier valor sí: el puente compara la forma canónica.
- **Solo enteros.** El contrato v1 no usa números con decimales, y los firmadores los rechazan (`"$249.00"` es texto).

## 4. Envía el documento

### Desde el navegador

Tu página pide el documento firmado a tu backend y lo entrega al puente con el SDK, sin tocarlo:

```js
const prisma = new Prisma({ url: "https://192.168.1.10:8443" }); // sin llave
const firmado = await fetch("/api/ticket-firmado", { method: "POST", body: JSON.stringify(pedido) }).then((r) => r.json());
await prisma.imprimirDocumento(firmado);
```

El origen de la página debe estar aprobado para tu aplicación de nivel 3. Este es el caso habitual cuando tu backend está en Internet y no puede ver la red del negocio. Ejemplo completo: [ejemplos/backend-php-firmado](ejemplos/backend-php-firmado/).

### Enviar desde el backend

Si tu backend está en la misma red que el puente, puede enviarlo directamente. Sin `Origin` ni llave, el puente identifica a tu aplicación por el `cliente_id` firmado:

```sh
curl --cacert prisma.crt https://192.168.1.10:8443/v1/imprimir \
  -H "Content-Type: application/json" --data-binary @firmado.json
```

```js
// Node.js, con NODE_EXTRA_CA_CERTS=prisma.crt
import { Prisma } from "@bayalad7/prisma";
await new Prisma({ url: "https://192.168.1.10:8443" }).imprimirDocumento(firmado);
```

## Cómo se firma (para otros lenguajes)

Si escribes tu propio firmador:

1. Arma el documento completo con `cliente_id`, `emitido` y `expira` (RFC 3339 en UTC, por ejemplo `2026-10-14T18:30:00Z`), **sin** `firma`.
2. Calcula su forma canónica JCS ([RFC 8785](https://www.rfc-editor.org/rfc/rfc8785)):
   - claves ordenadas por unidades de código UTF-16;
   - sin espacios;
   - cadenas con el escape mínimo de JSON (`\"`, `\\`, `\b`, `\f`, `\n`, `\r`, `\t` y `\u00XX` en minúsculas para el resto de controles);
   - todo lo demás en UTF-8 directo, incluidos `/`, `ñ`, `€` y U+2028.
3. Firma esos bytes UTF-8 con Ed25519 puro (RFC 8032, sin prehash ni contexto).
4. Agrega la firma en base64 estándar, con relleno: 88 caracteres.

Para probarlo:

- [`sdk/firmadores/casos-jcs.json`](../../sdk/firmadores/casos-jcs.json) trae entradas con su forma canónica exacta, incluidos los casos difíciles (escapes, orden UTF-16 de claves fuera del plano básico, separadores de línea). El puente comprueba ese mismo archivo en sus pruebas.
- [`testdata/vector-firma`](../../testdata/vector-firma/LEEME.md) trae una llave de prueba, un documento, su forma canónica y la firma que debes obtener.

Si tu firma no coincide, compara primero tu forma canónica con `canonico.txt`.

## Errores de firma

| Código | Causa habitual |
|---|---|
| `FIRMA_REQUERIDA` | La aplicación es de nivel 3 y el documento no trae `firma`. |
| `FIRMA_INVALIDA` | El documento cambió después de firmarlo, se firmó con otra llave, `cliente_id` no es el de la aplicación, o la vigencia supera 120 s. |
| `FIRMA_EXPIRADA` | `expira` ya pasó: se tardó en enviar, o el reloj del servidor está atrasado. |
| `ORIGEN_NO_AUTORIZADO` | El navegador entregó el documento desde una página cuyo origen no está aprobado para la aplicación. |

## Probar los firmadores

Desde la raíz del repositorio:

```sh
php sdk/firmadores/php/prueba.php
node --test "sdk/firmadores/node/*.test.mjs"
python -m unittest discover -s sdk/firmadores/python
```
