# API pública v1

La API que usan las aplicaciones para imprimir en un puente PRISMA. Sirve para llamar al puente sin el SDK, desde cualquier lenguaje o desde curl, y para entender qué hace el SDK por dentro.

- **Especificación formal:** [openapi.yaml](openapi.yaml) (OpenAPI 3.1).
- **Documento que se envía:** [contrato-documento-v1.md](contrato-documento-v1.md).
- **Todos los códigos de error:** [codigos-de-error.md](codigos-de-error.md).

> Si integras PRISMA en tu aplicación, empieza por la [guía para integradores](../integradores/README.md): el SDK JavaScript, los niveles de seguridad y los firmadores de nivel 3.

- **Base:** `https://<equipo-del-puente>:8443/v1/`. El puerto lo elige el negocio en [configuracion.md](configuracion.md); 8443 es el de fábrica.
- **Formato:** JSON en UTF-8, salvo `/v1/previsualizar`, que devuelve HTML.
- **TLS:** obligatorio, salvo en el modo `solo_localhost_http`, que acepta HTTP solo desde el propio equipo. Para que el navegador confíe en el certificado autofirmado, ver [instalacion/https-y-certificados.md](../instalacion/https-y-certificados.md).
- **Versiones:** «v1» es la versión de la API, independiente de la versión del programa (hoy 0.1.0) y de la del documento (`"version": "1"`).

| Método y ruta | Autorización | Para qué |
|---|---|---|
| `GET /v1/salud` | ninguna | Comprobar que el puente responde. |
| `GET /v1/impresoras` | origen o llave | Listar las impresoras que la aplicación puede usar. |
| `POST /v1/imprimir` | origen, llave o firma | Imprimir un documento. |
| `POST /v1/previsualizar` | igual que imprimir | Ver cómo saldría el ticket, sin imprimir. |

Cualquier otra ruta responde `404 RUTA_NO_ENCONTRADA`, y otro método, `405 METODO_NO_PERMITIDO` con la cabecera `Allow`.

---

## Autorización

Cada aplicación se registra en el puente como un **cliente** (en el panel se llama *aplicación*), con un nivel de seguridad:

| Nivel | Qué debe enviar la aplicación | Para qué tipo de aplicación |
|---|---|---|
| **1 — Origen** | Nada extra: el navegador envía `Origin`, que debe estar aprobado. | Páginas web sin backend en una red de confianza. No protege contra herramientas como curl. |
| **2 — Llave** (por defecto) | `Origin` aprobado (desde un navegador) **y** `Authorization: Bearer pk_…`. Desde un servidor, solo la llave. | La mayoría de las aplicaciones. |
| **3 — Firma** | Documentos firmados con Ed25519 por el backend de la aplicación ([contrato §7](contrato-documento-v1.md#7-firma-nivel-3)). La llave `pk_…` es opcional, salvo que el cliente la exija. | Aplicaciones con backend que no deben poder ser suplantadas. |

| Encabezado o campo | Cuándo |
|---|---|
| `Origin` | Lo envía el navegador solo. Debe pertenecer al cliente (niveles 1 y 2, y nivel 3 si el navegador entrega el documento). |
| `Authorization: Bearer pk_…` | Nivel 2, y nivel 3 si el cliente exige llave. Si se envía, debe ser válida aunque el nivel no la exija. |
| `cliente_id`, `emitido`, `expira` y `firma` del documento | Nivel 3. |

Para elegir el nivel, ver [integradores/niveles-de-seguridad.md](../integradores/niveles-de-seguridad.md). El árbol de decisión completo está en [diagramas/07-orden-autorizacion.md](../diagramas/07-orden-autorizacion.md).

**Cómo se registra un cliente:** el dueño lo crea en el panel (*Aplicaciones*) o aprueba en *Solicitudes* el origen de tu página después de su primer intento. También se puede con la CLI:

```sh
prisma clientes agregar --nombre "Caja web" --nivel 2 --origen https://caja.example
prisma clientes agregar --nombre "Backend" --nivel 3 --llave-publica publica.pem
```

### Orden de las comprobaciones

El puente aplica las comprobaciones en este orden y se detiene en la primera que falla. Nada costoso (validar el esquema completo, decodificar imágenes, renderizar) ocurre antes de autorizar.

1. **Límite por IP.**
2. **Origen:**
   - Si la petición trae `Origin`, debe pertenecer a un cliente activo. Si no pertenece a ninguno, la respuesta es `ORIGEN_NO_AUTORIZADO` y el origen queda registrado como solicitud pendiente.
   - Las peticiones sin `Origin` (curl, servidores) solo se aceptan para clientes de nivel 2 o 3.
3. **Llave:**
   - Se exige en el nivel 2, y en el 3 si el cliente la exige.
   - Si se envía, debe ser válida aunque el nivel no la exija.
   - Se compara con su hash argon2id en tiempo constante. El resultado queda en caché, así que solo la primera petición con una llave paga el cálculo.
4. **Límite por cliente.**
5. **Tamaño, JSON y versión** del documento (sin validarlo todo todavía).
6. **Firma (nivel 3):**
   - `cliente_id` debe coincidir con el cliente.
   - La firma Ed25519 sobre la forma canónica JCS debe ser válida.
   - La vigencia (`expira − emitido`) debe estar entre 0 y 120 s.
   - Se aceptan 30 s de diferencia de reloj.
   - Si la petición no trae ni origen ni llave, el cliente se identifica por el `cliente_id` del documento firmado.
7. **Idempotencia:** si el mismo `id` llegó hace menos de 10 minutos, se aplica lo descrito en [Reintentos e idempotencia](#reintentos-e-idempotencia).
8. **Permisos:** la impresora debe estar permitida para el cliente, y `abrir_cajon` y `raw` deben estar habilitadas para él.
9. **Validación completa del documento**, perfil de la impresora, render, cola y respuesta.

La secuencia completa está en [diagramas/03-flujo-impresion.md](../diagramas/03-flujo-impresion.md).

---

## `GET /v1/salud`

Sin autorización ni datos sensibles. Si la petición trae un `Origin` aprobado, la respuesta lleva sus cabeceras CORS.

```json
{"ok": true, "version": "0.1.0", "nombre_puente": "Café Prisma"}
```

`version` es la versión del programa. El nombre sale del ajuste `puente.nombre` (por defecto, «PRISMA»).

## `GET /v1/impresoras`

Devuelve las impresoras **activas** que el cliente puede usar, con su estado.

```json
[
  {"alias": "caja",   "nombre": "Caja principal", "ancho_mm": 80, "estado": "en_linea",       "virtual": false},
  {"alias": "cocina", "nombre": "Cocina",         "ancho_mm": 58, "estado": "fuera_de_linea", "virtual": false},
  {"alias": "prueba", "nombre": "Para probar",    "ancho_mm": 80, "estado": "en_linea",       "virtual": true}
]
```

`estado` es `en_linea`, `fuera_de_linea`, `ocupada` o `desconocido` (todavía sin probar). El puente prueba cada impresora cada 30 s, salvo mientras imprime. Nunca se exponen IPs, puertos ni nombres del sistema: solo el alias.

`virtual` (desde 0.2.0) indica una **impresora virtual**: vive dentro del puente, recibe exactamente los mismos bytes que una física y el ticket se ve en el panel en lugar de salir en papel. Sirve para probar una aplicación sin impresora. Se usa igual que cualquier otra; si el dueño la configura para simular fallas, responde con los mismos códigos que una física (por ejemplo `IMPRESORA_INALCANZABLE`).

## `POST /v1/imprimir`

El cuerpo es un documento v1. La respuesta llega cuando la impresora terminó.

| Estado | Significado |
|---|---|
| 200 | Impreso. |
| 200 con `"duplicado": true` | El mismo `id` con el mismo contenido llegó hace menos de 10 min. **No se volvió a imprimir.** |
| 202 con `"estado": "encolado"` | La impresora tardó más de 60 s en atenderlo (tenía trabajos por delante). Se imprimirá, y el resultado queda en el historial del puente. |
| 4xx / 5xx | Error. Los errores de la impresora incluyen `trabajo_id`. |

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

**200 — repetido:**

```json
{"ok": true, "duplicado": true, "trabajo_id": "trb_cb1393f662e11be7", "impresora": "caja", "estado": "completado", "duracion_ms": 412, "advertencias": []}
```

**202 — en cola:**

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

Las `advertencias` no impiden imprimir: `CARACTER_SUSTITUIDO` indica que un carácter no existía en la página de códigos de la impresora, y `SIN_CUCHILLA`, que la impresora no corta. `IMPRESORA_FUERA_DE_LINEA` (solo en impresoras con *Confirmar impresión*) indica que la impresora recibió el ticket, pero tiene la tapa abierta o no tiene papel: saldrá al resolverlo, así que **no lo reenvíes**. La lista completa está en [codigos-de-error.md](codigos-de-error.md#advertencias).

### Reintentos e idempotencia

Genera un `id` (UUID v4) nuevo para cada ticket y **reutilízalo** en los reintentos de ese mismo ticket. El puente recuerda cada `id` durante 10 minutos, también después de reiniciarse:

- **Mismo `id` y mismo contenido:** responde el trabajo original con `duplicado: true`, sin imprimir.
- **Mismo `id` con otro contenido:** `409 ID_REUTILIZADO`.
- **El trabajo original no se imprimió** (impresora apagada, sin papel, puente reiniciado con el ticket en cola): el `id` queda libre y el mismo documento se imprime al reenviarlo.

Reintenta solo cuando no hubo respuesta (error de red o tiempo agotado), o cuando la respuesta es un 503 con `Retry-After`. Cualquier otro error con código es una respuesta del puente: no cambiará por insistir. Ver [diagramas/24-idempotencia.md](../diagramas/24-idempotencia.md).

## `POST /v1/previsualizar`

- **Autorización:** la misma que `imprimir`, sin idempotencia.
- **Respuesta:** un fragmento HTML autocontenido (`text/html`) con la simulación exacta del ticket en el perfil de la impresora. No imprime nada.
- **Seguridad:** todo el texto del documento va escapado, y el fragmento solo usa estilos en línea e imágenes `data:`. Insértalo en un contenedor o en un `<iframe srcdoc>`.

---

## CORS y acceso a la red local

| Situación | Qué hace el puente |
|---|---|
| Preflight (`OPTIONS`) de un origen aprobado | 204 con `Access-Control-Allow-Origin: <origen exacto>`, `Allow-Methods: GET, POST, OPTIONS`, `Allow-Headers: Content-Type, Authorization` y `Max-Age: 600`. |
| Petición de un origen aprobado | Repite el origen exacto en `Access-Control-Allow-Origin` (nunca `*`), con `Vary: Origin`, y expone `Retry-After`. |
| Origen desconocido | 403 **sin** cabeceras CORS. El origen queda como solicitud pendiente en el panel. |
| `Access-Control-Request-Private-Network: true` | Responde `Access-Control-Allow-Private-Network: true` a los orígenes aprobados. |
| 429 del límite por IP | Lleva las cabeceras CORS del origen de la petición, para que el navegador pueda leer `DEMASIADAS_PETICIONES`. |

Consecuencias en el navegador:

- **Un origen no aprobado se ve como un error de red** (`TypeError: Failed to fetch`), no como un 403: el navegador no deja leer la respuesta. El SDK lo distingue con una petición sin CORS a `/v1/salud`: si esa responde, el problema es la autorización del origen.
- **Página pública → puente en la red local:** Chrome y Edge aplican *Private Network Access* / *Local Network Access*. El puente responde al preflight que piden, y las versiones recientes además piden permiso a la persona. Ver [integradores/problemas-comunes.md](../integradores/problemas-comunes.md#chrome-o-edge-permiso-de-red-local).
- **Página `https://` → puente `http://`:** el navegador lo bloquea por contenido mixto, salvo con `localhost`.

El recorrido completo está en [diagramas/04-secuencia-nivel-1.md](../diagramas/04-secuencia-nivel-1.md).

## Límites

| Límite | Valor | Respuesta |
|---|---|---|
| Cuerpo de la petición | Se leen hasta 300 KB; el documento puede medir hasta 256 KB (262 144 bytes) | 413 `DOCUMENTO_DEMASIADO_GRANDE` |
| Operaciones por documento | 500 | 400 `DEMASIADAS_OPERACIONES` |
| Peticiones por IP | 120 por minuto por defecto (`[limites]` de `config.toml`, o el panel). El panel (`/admin`) cuenta aparte, 600 por minuto | 429 `DEMASIADAS_PETICIONES` + `Retry-After` |
| Peticiones por cliente | 60 por minuto por defecto (`[limites]` de `config.toml`, o el panel) | 429 `DEMASIADAS_PETICIONES` + `Retry-After` |
| Trabajos pendientes por impresora | 20 | 503 `COLA_LLENA` + `Retry-After` |
| Vigencia de un documento firmado | 120 s, con 30 s de tolerancia de reloj | 401 `FIRMA_INVALIDA` o `FIRMA_EXPIRADA` |
| Ventana de idempotencia | 10 min | ver [Reintentos e idempotencia](#reintentos-e-idempotencia) |
| Espera del resultado | 60 s | 202 `encolado` |

Los límites de peticiones admiten ráfagas: con 60 por minuto se pueden enviar 60 seguidas y luego una por segundo.

## Errores

Todas las respuestas llevan `ok`. Los errores agregan, en el mismo nivel, `codigo`, `mensaje` y `sugerencia`, y según el caso `detalles`, `trabajo_id` o `versiones_soportadas`. Programa contra `codigo`, que es estable; `mensaje` y `sugerencia` están en español, pensados para mostrarlos a la persona, y pueden cambiar.

La tabla completa, con el estado HTTP de cada código, está en [codigos-de-error.md](codigos-de-error.md).

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

El [SDK JavaScript](../integradores/sdk.md) hace esto, valida el documento antes de enviarlo y reintenta los errores de red con el mismo `id`.

### Nivel 2 con curl

```sh
curl --cacert prisma.crt https://192.168.1.10:8443/v1/imprimir \
  -H "Authorization: Bearer pk_7ax6nj4r4a_…" \
  -H "Content-Type: application/json" \
  --data-binary @ejemplos/validos/venta-simple.json
```

`prisma.crt` es el certificado que se descarga del panel, en *Ajustes → HTTPS*. Para una prueba rápida, `-k` acepta el certificado autofirmado sin verificarlo.

### Nivel 3 con curl

```sh
prisma dev llaves --salida llaves/                       # par de prueba
prisma clientes agregar --nombre "Backend" --nivel 3 --llave-publica llaves/publica.pem
prisma dev firmar ejemplos/validos/comanda-cocina.json --llave llaves/privada.pem \
       --cliente cli_xxxxxxxx --nuevo-id -o firmado.json
curl --cacert prisma.crt https://192.168.1.10:8443/v1/imprimir -H "Content-Type: application/json" --data-binary @firmado.json
```

El documento firmado vence a los 90 s (`--vigencia`, como máximo 120 s). Para cada ticket nuevo, fírmalo de nuevo con `--nuevo-id`.

### Firmar desde tu backend

Hay firmadores de referencia para PHP, Node.js y Python en [`sdk/firmadores`](../../sdk/firmadores/README.md). La guía está en [integradores/firmar-documentos.md](../integradores/firmar-documentos.md).

```js
import { firmar } from "./firmador.mjs";   // sdk/firmadores/node
const firmado = firmar(documento, { clienteId: "cli_…", llavePrivada: pem });
```

Para validar tu firmador, usa el vector fijo de [testdata/vector-firma](../../testdata/vector-firma/LEEME.md): con su llave y su documento debes obtener exactamente su `firma.txt`.
