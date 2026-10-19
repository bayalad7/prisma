# API pública v1

La API que usan las aplicaciones para imprimir en un puente PRISMA. El documento que se envía está especificado en [contrato-documento-v1.md](contrato-documento-v1.md).

- **Base:** `https://<equipo-del-puente>:8443/v1/`. El puerto se configura en [configuracion.md](configuracion.md).
- **Formato:** JSON en UTF-8, salvo `/v1/previsualizar`, que devuelve HTML.
- **TLS:** obligatorio, salvo en el modo `solo_localhost_http`. Para que el navegador confíe en el certificado autofirmado, ver [tls.md](tls.md).

| Método y ruta | Autorización | Para qué |
|---|---|---|
| `GET /v1/salud` | ninguna | Comprobar que el puente responde. |
| `GET /v1/impresoras` | origen o llave | Listar las impresoras que la aplicación puede usar. |
| `POST /v1/imprimir` | origen, llave o firma | Imprimir un documento. |
| `POST /v1/previsualizar` | igual que imprimir | Ver cómo saldría el ticket, sin imprimir. |

---

## Autorización

Cada aplicación se registra en el puente como un **cliente**, con un nivel de seguridad:

| Nivel | Qué debe enviar la aplicación | Para qué tipo de aplicación |
|---|---|---|
| **1 — Origen** | Nada extra: el navegador envía `Origin`, que debe estar aprobado. | Páginas web sin backend en una red de confianza. No protege contra herramientas como curl. |
| **2 — Llave** (por defecto) | `Origin` aprobado (desde un navegador) **y** `Authorization: Bearer pk_…`. Desde un servidor, solo la llave. | La mayoría de las aplicaciones. |
| **3 — Firma** | Documentos firmados con Ed25519 por el backend de la aplicación ([contrato §7](contrato-documento-v1.md#7-firma-nivel-3)). La llave `pk_…` es opcional, salvo que el cliente la exija. | Aplicaciones con backend que no deben poder ser suplantadas. |

**Cómo se registra un cliente:** el dueño del puente lo crea en el [panel](panel.md) (*Aplicaciones*), o aprueba en *Solicitudes* el origen de tu página después de su primer intento. También se puede con la CLI:

```sh
prisma clientes agregar --nombre "Caja web" --nivel 2 --origen https://caja.example
prisma clientes agregar --nombre "Backend" --nivel 3 --llave-publica publica.pem
```

Cuando un origen desconocido intenta imprimir, queda registrado como **solicitud pendiente** para que el dueño lo apruebe.

### Orden de las comprobaciones

El puente aplica las comprobaciones en este orden y se detiene en la primera que falla. Nada costoso (validar el esquema completo, decodificar imágenes, renderizar) ocurre antes de autorizar.

1. **Origen:**
   - Si la petición trae `Origin`, debe pertenecer a un cliente activo. Si no pertenece a ninguno, la respuesta es `ORIGEN_NO_AUTORIZADO` y el origen queda registrado como solicitud pendiente.
   - Las peticiones sin `Origin` (curl, servidores) solo se aceptan para clientes de nivel 2 o 3.
2. **Llave:**
   - Se exige en el nivel 2, y en el 3 si el cliente la exige.
   - Si se envía, debe ser válida aunque el nivel no la exija.
   - Se compara con su hash argon2id en tiempo constante. El resultado queda en caché, así que solo la primera petición con una llave paga el cálculo.
3. **Firma (nivel 3):**
   - `cliente_id` debe coincidir con el cliente.
   - La firma Ed25519 sobre la forma canónica JCS debe ser válida.
   - La vigencia (`expira − emitido`) debe estar entre 0 y 120 s.
   - Se aceptan 30 s de diferencia de reloj.
   - Si la petición no trae ni origen ni llave, el cliente se identifica por el `cliente_id` del documento firmado.
4. **Idempotencia:** si el mismo `id` llegó hace menos de 10 minutos, se aplica lo descrito en [contrato §8](contrato-documento-v1.md#8-idempotencia).
5. **Permisos:** la impresora debe estar permitida para el cliente, y `abrir_cajon` y `raw` deben estar habilitadas para él.
6. **Validación completa del documento**, render, cola y respuesta.

---

## `GET /v1/salud`

Sin autorización ni datos sensibles.

```json
{"ok": true, "version": "0.4.0", "nombre_puente": "Café Prisma"}
```

El nombre sale del ajuste `puente.nombre` (por defecto, «PRISMA»).

## `GET /v1/impresoras`

Devuelve las impresoras activas que el cliente puede usar, con su estado.

```json
[
  {"alias": "caja",   "nombre": "Caja principal", "ancho_mm": 80, "estado": "en_linea"},
  {"alias": "cocina", "nombre": "Cocina",         "ancho_mm": 58, "estado": "fuera_de_linea"}
]
```

`estado` puede ser `en_linea`, `fuera_de_linea`, `ocupada` o `desconocido`. El puente prueba cada impresora cada 30 s.

## `POST /v1/imprimir`

El cuerpo es un documento v1. La respuesta llega cuando la impresora terminó.

**200 — impreso:**

```json
{
  "ok": true,
  "trabajo_id": "trb_cb1393f662e11be7",
  "impresora": "caja",
  "duracion_ms": 412,
  "advertencias": [
    {"codigo": "CARACTER_SUSTITUIDO", "mensaje": "La página de códigos CP437 no tiene algunos caracteres; …"}
  ]
}
```

**200 — repetido:** mismo `id` y mismo contenido en los últimos 10 minutos. No se vuelve a imprimir:

```json
{"ok": true, "duplicado": true, "trabajo_id": "trb_cb1393f662e11be7", "impresora": "caja", "estado": "completado", "duracion_ms": 412, "advertencias": []}
```

**202 — en cola:** la impresora tardó más de 60 s en atender (por ejemplo, porque tenía muchos trabajos por delante). El ticket se imprimirá, y el resultado queda en el historial del puente:

```json
{"ok": true, "trabajo_id": "trb_…", "impresora": "caja", "estado": "encolado", "advertencias": []}
```

**Error:**

```json
{
  "ok": false,
  "codigo": "IMPRESORA_INALCANZABLE",
  "mensaje": "La impresora en 192.168.1.50:9100 rechazó la conexión.",
  "sugerencia": "Revisa que la dirección y el puerto (normalmente 9100) sean correctos…",
  "trabajo_id": "trb_…"
}
```

Si la impresión falló, el `id` del documento queda libre: puedes reintentar el **mismo** documento cuando la impresora vuelva.

## `POST /v1/previsualizar`

- **Autorización:** la misma que `imprimir`, sin idempotencia.
- **Respuesta:** un fragmento HTML autocontenido (`text/html`) con la simulación exacta del ticket en el perfil de la impresora. No imprime nada.
- **Seguridad:** todo el texto del documento va escapado, y el fragmento solo usa estilos en línea e imágenes `data:`.

---

## CORS y red privada

- **`Access-Control-Allow-Origin`:** repite el origen **exacto**, y solo si el origen está aprobado. Nunca es `*`. Todas las respuestas llevan `Vary: Origin`.
- **Preflight (`OPTIONS`):** para un origen aprobado responde 204 con:
  - `Access-Control-Allow-Methods: GET, POST, OPTIONS`;
  - `Access-Control-Allow-Headers: Content-Type, Authorization`;
  - `Access-Control-Allow-Private-Network: true`, si el navegador la pide (Private Network Access, cuando una página pública llama a un equipo de la red local).
- **Origen desconocido:** recibe 403 sin cabeceras CORS y queda como solicitud pendiente.

Desde un navegador, una petición de un origen no aprobado falla como **error de red**: el navegador no deja leer el cuerpo. En la consola de desarrollo se ve el aviso de CORS.

## Límites

| Límite | Valor | Respuesta |
|---|---|---|
| Cuerpo de la petición | 300 KB leídos; el documento puede medir hasta 256 KB | 413 `DOCUMENTO_DEMASIADO_GRANDE` |
| Peticiones por IP | 120 por minuto por defecto (`[limites]` de `config.toml`, o el panel) | 429 `DEMASIADAS_PETICIONES` + `Retry-After` |
| Peticiones por cliente | 60 por minuto por defecto (`[limites]` de `config.toml`, o el panel) | 429 `DEMASIADAS_PETICIONES` + `Retry-After` |
| Trabajos pendientes por impresora | 20 | 503 `COLA_LLENA` + `Retry-After` |

Los límites admiten ráfagas: con 60 por minuto se pueden enviar 60 seguidas y luego una por segundo.

## Códigos de error de la API

Los códigos del documento (`DOCUMENTO_INVALIDO`, `FIRMA_INVALIDA`, `FILA_NO_CABE`, …) están en [contrato §9](contrato-documento-v1.md#9-validación-y-errores). Además:

| Código | HTTP | Cuándo |
|---|---|---|
| `ORIGEN_NO_AUTORIZADO` | 403 | El origen no pertenece a ningún cliente (queda como solicitud pendiente). |
| `ORIGEN_REQUERIDO` | 403 | Un cliente de nivel 1 llamó sin `Origin` (desde fuera de un navegador). |
| `CREDENCIALES_REQUERIDAS` | 401 | Sin origen, sin llave y sin documento firmado por un cliente de nivel 3. |
| `LLAVE_REQUERIDA` | 401 | Nivel 2 (o nivel 3 que la exige) sin `Authorization`. |
| `LLAVE_INVALIDA` | 401 | Llave mal formada, desconocida, incorrecta o de otro cliente. |
| `CLIENTE_DESACTIVADO` | 403 | El cliente está desactivado en el puente. |
| `IMPRESORA_NO_PERMITIDA` | 403 | El cliente no puede usar esa impresora. |
| `OPERACION_NO_PERMITIDA` | 403 | El documento usa `abrir_cajon` o `raw` y el cliente no las tiene habilitadas. |
| `IMPRESORA_NO_ENCONTRADA` | 404 | No hay ninguna impresora activa con ese alias. |
| `RUTA_NO_ENCONTRADA` · `METODO_NO_PERMITIDO` | 404 · 405 | Ruta o método inexistente. |
| `DEMASIADAS_PETICIONES` | 429 | Se superó el límite de peticiones. |
| `IMPRESORA_INALCANZABLE` · `PUERTO_OCUPADO` · `PERMISO_DENEGADO` · `COLA_LLENA` | 503 | La impresora no pudo imprimir; ver [conectores.md](conectores.md#errores-de-conexión). |
| `TIEMPO_AGOTADO` | 504 | La impresora dejó de recibir datos. |
| `PUENTE_NO_DISPONIBLE` | 503 | El puente se está deteniendo. |
| `ERROR_INTERNO` | 500 | Error inesperado. El detalle queda solo en la bitácora del puente. |

---

## Ejemplos

### Nivel 1 o 2 desde el navegador

```js
const respuesta = await fetch("https://192.168.1.10:8443/v1/imprimir", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "Authorization": "Bearer pk_7ax6nj4r4a_…",   // solo nivel 2
  },
  body: JSON.stringify(documento),
});
const resultado = await respuesta.json();
if (!resultado.ok) alert(`${resultado.mensaje}\n${resultado.sugerencia}`);
```

⏳ El SDK JavaScript (`sdk/js/prisma.js`) hará esto, con reintentos idempotentes.

### Nivel 2 con curl

```sh
curl -k https://192.168.1.10:8443/v1/imprimir \
  -H "Authorization: Bearer pk_7ax6nj4r4a_…" \
  -H "Content-Type: application/json" \
  --data-binary @ejemplos/validos/venta-simple.json
```

`-k` acepta el certificado autofirmado. Para no usarlo, confía en el certificado como explica [tls.md](tls.md) y usa `--cacert certificado.pem`.

### Nivel 3 con curl

```sh
prisma dev llaves --salida llaves/                       # par de prueba
prisma clientes agregar --nombre "Backend" --nivel 3 --llave-publica llaves/publica.pem
prisma dev firmar ejemplos/validos/comanda-cocina.json --llave llaves/privada.pem \
       --cliente cli_xxxxxxxx --nuevo-id -o firmado.json
curl -k https://192.168.1.10:8443/v1/imprimir -H "Content-Type: application/json" --data-binary @firmado.json
```

El documento firmado vence a los 90 s (`--vigencia`, como máximo 120 s). Para cada ticket nuevo, fírmalo de nuevo con `--nuevo-id`.

### Firmar desde tu backend (Node.js)

```js
import { sign, createPrivateKey } from "node:crypto";
import canonicalize from "canonicalize";            // JCS (RFC 8785)

function firmar(documento, clienteId, llavePrivadaPem) {
  const emitido = new Date();
  const doc = {
    ...documento,
    cliente_id: clienteId,
    emitido: emitido.toISOString().replace(/\.\d{3}Z$/, "Z"),
    expira: new Date(emitido.getTime() + 90_000).toISOString().replace(/\.\d{3}Z$/, "Z"),
  };
  delete doc.firma;
  const firma = sign(null, Buffer.from(canonicalize(doc), "utf8"), createPrivateKey(llavePrivadaPem));
  return { ...doc, firma: firma.toString("base64") };
}
```

Para validar tu firmador, usa el vector fijo de [testdata/vector-firma](../testdata/vector-firma/LEEME.md): con su llave y su documento debes obtener exactamente su `firma.txt`.
