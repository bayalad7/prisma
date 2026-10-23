# Referencia del SDK JavaScript

`@bayalad7/prisma` funciona en el navegador y en Node.js 18+. No tiene dependencias e incluye tipos TypeScript.

## Instalación

El SDK se publica con cada versión de PRISMA, en la etiqueta `v<versión>` del repositorio. Fija siempre la versión en la dirección, para que una versión nueva no cambie tu página sin aviso.

**Desde un CDN, con `<script>`** (define la variable global `Prisma`):

```html
<script src="https://cdn.jsdelivr.net/gh/bayalad7/prisma@v0.2.0/sdk/js/dist/prisma.umd.js"></script>
```

**Desde un CDN, como módulo ES:**

```html
<script type="module">
  import { Prisma } from "https://cdn.jsdelivr.net/gh/bayalad7/prisma@v0.2.0/sdk/js/dist/prisma.mjs";
</script>
```

**En Node.js 18+ o con un empaquetador:** copia el archivo al proyecto y impórtalo desde ahí.

```sh
curl -fsSLo prisma.mjs https://cdn.jsdelivr.net/gh/bayalad7/prisma@v0.2.0/sdk/js/dist/prisma.mjs
```

```js
import { Prisma } from "./prisma.mjs";           // módulo ES
const { Prisma } = require("./prisma.umd.js");   // CommonJS, con el archivo .umd.js
```

- **npm:** ⏳ el paquete `@bayalad7/prisma` todavía no está publicado en npm: `npm install @bayalad7/prisma`, `cdn.jsdelivr.net/npm/…` y unpkg responden 404. Cuando exista, se anunciará en el [registro de cambios](../../CHANGELOG.md).
- **Tipos TypeScript:** `prisma.d.mts` y `prisma.umd.d.ts`, en la misma carpeta `sdk/js/dist/`.
- **Versión del SDK:** `Prisma.VERSION_SDK`. La del puente al que te conectas la da `await prisma.salud()`.

## `new Prisma(opciones)`

```js
const prisma = new Prisma({ url: "https://192.168.1.10:8443", llave: "pk_..." });
```

| Opción | Por defecto | Descripción |
|---|---|---|
| `url` | obligatoria | Dirección del puente. Se usa solo el origen: `https://192.168.1.10:8443/v1/` es lo mismo que `https://192.168.1.10:8443`. |
| `llave` | — | Llave de cliente `pk_…`, para el nivel 2 (o el 3 que la exige). |
| `reintentos` | `2` | Reintentos ante errores **de red**. Los errores del puente nunca se reintentan. |
| `espera_reintento_ms` | `1000` | Espera antes del primer reintento. Se triplica en cada uno: 1 s, 3 s… |
| `tiempo_espera_ms` | `75000` | Tiempo máximo por petición. El puente responde 202 a los 60 s si la impresora tiene trabajos por delante. |
| `fetch` | `globalThis.fetch` | Una implementación propia de `fetch`, por ejemplo con un agente que confíe en el certificado del puente. |

Una URL o una llave mal formadas lanzan `CONFIGURACION_INVALIDA` en el momento.

## Métodos

Todos devuelven una promesa. Si algo falla, la promesa se rechaza con un [`PrismaError`](#errores).

### `salud()`

```js
const { version, nombre_puente } = await prisma.salud();
```

No necesita autorización. Sirve para comprobar que el puente responde.

### `impresoras()`

```js
const lista = await prisma.impresoras();
// [{ alias: "caja", nombre: "Caja principal", ancho_mm: 80, estado: "en_linea" }, …]
```

Devuelve solo las impresoras activas que tu aplicación puede usar.

### `imprimir(impresora, contenido, opciones?)`

```js
const r = await prisma.imprimir("caja", (doc) => doc
  .texto("MI NEGOCIO", { alineacion: "centro", negrita: true, tamano: [2, 2] })
  .separador()
  .par("2 x Taco", "$60.00")
  .par("TOTAL", "$60.00", { negrita: true })
  .qr("https://minegocio.com/p/812")
  .cortar(), { referencia: "VTA-4127" });
// { ok: true, trabajo_id: "trb_…", impresora: "caja", duracion_ms: 412, advertencias: [] }
```

`contenido` puede ser:

- una función que recibe un [`Documento`](#documento) y le agrega operaciones;
- un `Documento` ya armado;
- un arreglo de operaciones del contrato: `[{ tipo: "texto", valor: "Hola" }, { tipo: "cortar" }]`.

`opciones` admite `copias` (1 a 5), `referencia` (aparece en el historial del puente; no se imprime) e `id` (si quieres controlar el UUID v4 tú mismo).

| Campo del resultado | Cuándo |
|---|---|
| `trabajo_id`, `impresora`, `advertencias` | Siempre. |
| `duracion_ms` | Cuando la impresora terminó. |
| `duplicado: true` | El puente ya había recibido este mismo documento: no lo volvió a imprimir. |
| `estado: "encolado"` | El puente respondió 202: se imprimirá en cuanto la impresora lo atienda. |

### `imprimirDocumento(documento)`

```js
const firmado = await fetch("/firmar", { method: "POST", body: … }).then((r) => r.json());
await prisma.imprimirDocumento(firmado);
```

Envía un documento completo **sin modificarlo**. Se usa en el nivel 3, cuando tu backend firma el ticket, o cuando ya tienes el JSON armado. Se valida antes de enviarlo.

### `previsualizar(impresora, contenido, opciones?)` · `previsualizar(documento)`

```js
contenedor.innerHTML = await prisma.previsualizar("caja", (doc) => doc.texto("Hola").cortar());
```

Devuelve el HTML de la simulación exacta del ticket en el perfil de esa impresora (ancho, columnas, página de códigos), sin imprimir. El puente escapa todo el texto del documento.

### `documento(impresora, opciones?)`

Crea un `Documento` sin enviarlo, para armarlo en varios pasos:

```js
const doc = prisma.documento("cocina", { referencia: "MESA-4" });
for (const p of pedido) doc.par(`${p.cantidad} x ${p.nombre}`, p.notas ?? "");
doc.cortar();
await prisma.imprimir("cocina", doc);
```

### `diagnosticar()`

```js
const d = await prisma.diagnosticar();
if (!d.ok) mostrar(d.error.mensaje, d.error.sugerencia);
```

Nunca lanza. Devuelve `{ ok: true, salud }` o `{ ok: false, error }` con la causa más probable. Útil para un botón «Probar conexión».

## `Documento`

Cada método agrega una operación, la **valida en ese momento** y devuelve el mismo documento, para encadenar. Los nombres de las opciones son los del contrato.

| Método | Operación | Opciones |
|---|---|---|
| `texto(valor, opciones?)` | `texto` | `alineacion` (`izquierda`·`centro`·`derecha`), `negrita`, `subrayado`, `invertido`, `tamano: [ancho, alto]` (1–8), `fuente` (`A`·`B`). `\n` fuerza un salto de línea. |
| `separador(caracter?)` | `separador` | Un carácter; `"-"` por defecto. |
| `fila(columnas, opciones?)` | `fila` | `columnas`: textos o `{ texto, ancho, alineacion }`, donde `ancho` es un porcentaje entero o `"auto"`. Opción: `negrita`. |
| `par(izquierda, derecha, opciones?)` | `par` | `negrita`. |
| `salto(lineas?)` | `salto` | 1 a 10 líneas. |
| `qr(valor, opciones?)` | `qr` | `tamano` (1–16), `correccion` (`L`·`M`·`Q`·`H`), `alineacion`. |
| `codigoBarras(valor, simbologia, opciones?)` | `codigo_barras` | `simbologia`: `CODE128`, `CODE39`, `EAN13`, `EAN8`, `UPCA`, `ITF`. Opciones: `texto` (`arriba`·`abajo`·`ninguno`), `alto` (1–255), `alineacion`. |
| `imagen(fuente, opciones?)` | `imagen` | `fuente`: el nombre de una imagen guardada en el puente (`"logo-tienda"`), un PNG en base64 (con o sin `data:image/png;base64,`) o sus bytes (`Uint8Array`, `ArrayBuffer`). Opciones: `alineacion`, `ancho_max` (porcentaje). |
| `cortar(opciones?)` | `cortar` | `parcial`. |
| `abrirCajon(pin?)` | `abrir_cajon` | `2` o `5`. **Sensible:** el dueño debe habilitarla para tu aplicación. |
| `raw(datos)` | `raw` | Bytes ESC/POS en base64 o `Uint8Array`. **Sensible.** |
| `operacion(op)` | cualquiera | Una operación tal como la define el contrato: `{ tipo: "salto", lineas: 2 }`. |
| `copias(n)` · `referencia(texto)` | sobre | Cambian el sobre del documento. |
| `json()` | — | El documento validado completo, listo para enviar o guardar. `JSON.stringify(doc)` da lo mismo. |
| `id` | — | El UUID v4 del documento. |

```js
doc
  .fila(["1", { texto: "Pan de caja integral", ancho: "auto" }, { texto: "$62.50", ancho: 25, alineacion: "derecha" }])
  .codigoBarras("4006381333931", "EAN13", { alto: 60 })
  .imagen("logo-tienda", { ancho_max: 60 })
  .imagen(await Prisma.pngABase64(archivoDelInput))
  .abrirCajon();
```

Las reglas de cada operación (cómo se reparten las columnas, qué hace la impresora si no tiene QR nativo…) están en el [contrato](../referencia/contrato-documento-v1.md#4-operaciones).

## Validación local

Antes de enviar, el SDK aplica las mismas reglas que el puente: tipos, valores permitidos, largos, caracteres de control, dígitos verificadores, porcentajes de las filas, tamaño de imágenes y del documento. Lo que no pasa se rechaza con el **mismo código** que daría el puente y con `status: 0`. Nada sale de tu aplicación.

```js
try {
  prisma.documento("caja").texto("Total", { alineacion: "center" });
} catch (e) {
  e.codigo;              // "DOCUMENTO_INVALIDO"
  e.detalles[0].ruta;    // "/operaciones/0/alineacion"
  e.detalles[0].mensaje; // "debe ser uno de: izquierda, centro, derecha"
}
```

`Prisma.validarDocumento(json)` valida un documento ya armado y devuelve `null` o el `PrismaError`.

Lo que depende de la impresora (que una fila quepa en sus columnas, que tenga cuchilla o imágenes, que exista un recurso) y la verificación de firmas solo las puede comprobar el puente.

## Reintentos

- **Solo errores de red:** no hubo respuesta, se cortó la conexión o se agotó el tiempo. Un error con código del puente (`IMPRESORA_INALCANZABLE`, `DEMASIADAS_PETICIONES`…) nunca se reintenta solo.
- **Mismo documento:** cada reintento envía exactamente el mismo cuerpo, con el **mismo `id`**. Si el primer envío sí llegó, el puente responde `duplicado: true` en lugar de imprimir otra vez.
- **Sin reintento cuando no sirve:** `CERTIFICADO_NO_CONFIABLE`, `ORIGEN_NO_AUTORIZADO`, `PERMISO_RED_LOCAL_DENEGADO` y `CONTENIDO_MIXTO` se reportan al primer intento. En el navegador, tampoco se reintenta la primera conexión fallida con un puente que nunca respondió (paso 4 de [la sección siguiente](#por-qué-el-navegador-no-siempre-distingue-el-certificado)): lo más probable es el certificado, que no se arregla esperando.

Si reintentas tú, por ejemplo tras un botón «Reintentar», reutiliza el mismo documento: `prisma.imprimir("caja", doc)`, con el `Documento` original.

## Errores

Todos los errores son instancias de `PrismaError`:

| Campo | Descripción |
|---|---|
| `codigo` | Código estable en MAYÚSCULAS. Programa contra él. |
| `mensaje` | Qué pasó, en español, para mostrarlo a la persona. También está en `message`. |
| `sugerencia` | Qué hacer. |
| `status` | Estado HTTP de la respuesta del puente; `0` si no hubo respuesta o lo detectó el SDK. |
| `detalles` | En los errores del documento: `[{ ruta, regla, mensaje }]`. |
| `trabajo_id` | Si el error ocurrió al imprimir. |
| `causas_posibles` | Solo en `PUENTE_INALCANZABLE` desde el navegador (ver abajo). |
| `respuesta` | El cuerpo JSON tal como lo respondió el puente. |
| `reintentable` | `true` si es un error de red transitorio. |

```js
try {
  await prisma.imprimir("caja", (d) => d.texto("Hola").cortar());
} catch (e) {
  if (!(e instanceof Prisma.PrismaError)) throw e;
  switch (e.codigo) {
    case "IMPRESORA_INALCANZABLE": avisar("La impresora no responde: revisa que esté encendida."); break;
    case "ORIGEN_NO_AUTORIZADO":   avisar("Pide al encargado que apruebe esta página en el panel de PRISMA."); break;
    default:                       avisar(`${e.mensaje} ${e.sugerencia}`);
  }
}
```

Además de los [códigos del puente](../referencia/codigos-de-error.md), el SDK genera estos cuando no hay respuesta útil (`status: 0`):

| Código | Cuándo | Qué hacer |
|---|---|---|
| `CONFIGURACION_INVALIDA` | URL o llave mal formadas, o contenido de `imprimir` que no es función, `Documento` ni arreglo. | Corregir el código. |
| `ORIGEN_NO_AUTORIZADO` | Desde el navegador: el puente responde, pero sin CORS para tu página. | El dueño aprueba el origen en el panel (*Solicitudes*). |
| `PERMISO_RED_LOCAL_DENEGADO` | El navegador bloqueó el acceso de la página a la red local. | Permitirlo en el candado de la barra de direcciones. Ver [problemas-comunes.md](problemas-comunes.md#chrome-o-edge-permiso-de-red-local). |
| `CERTIFICADO_NO_CONFIABLE` | Node.js no confía en el certificado del puente. | `NODE_EXTRA_CA_CERTS=prisma.crt`. |
| `PUENTE_INALCANZABLE` | No hubo conexión: puente apagado, IP incorrecta, otra red o (en el navegador) certificado no confiable. | Ver `causas_posibles` y `sugerencia`. |
| `PUENTE_SIN_RESPUESTA` | Conectó, pero no respondió a tiempo. | Revisar la red; se reintenta solo. |
| `CONTENIDO_MIXTO` | Página `https://` y puente `http://` fuera de localhost. | Usar la dirección `https://` del puente. |
| `RESPUESTA_INVALIDA` | Respondió algo que no es un puente PRISMA. | Revisar la URL y el puerto. |

### Por qué el navegador no siempre distingue el certificado

Ante un certificado no confiable, un equipo apagado o una IP equivocada, el navegador da el mismo error (`TypeError: Failed to fetch`), sin detalles, por diseño. El SDK hace lo que puede:

1. Si la red local está bloqueada por permiso, responde `PERMISO_RED_LOCAL_DENEGADO`.
2. Si una petición sin CORS a `/v1/salud` sí llega, el puente está bien y lo que falta es autorizar el origen: `ORIGEN_NO_AUTORIZADO`.
3. Si ese puente ya respondió antes desde la página, ahora está apagado o fuera de la red: `PUENTE_INALCANZABLE` con `causas_posibles: ["PUENTE_APAGADO_O_FUERA_DE_RED"]`.
4. Si no, es la primera conexión, y la causa más común es el certificado: `PUENTE_INALCANZABLE` con `causas_posibles: ["CERTIFICADO_NO_CONFIABLE", "PUENTE_APAGADO_O_FUERA_DE_RED"]`. La sugerencia indica abrir `/v1/salud` en una pestaña: si aparece una advertencia de certificado, es eso. Este error llega de inmediato, sin reintentos (`reintentable: false`).

En Node.js el error sí trae la causa, y el SDK responde `CERTIFICADO_NO_CONFIABLE` o `PUENTE_INALCANZABLE` con certeza.

## Node.js

```js
import { Prisma } from "./prisma.mjs";   // copiado de sdk/js/dist (ver Instalación)

const prisma = new Prisma({ url: "https://192.168.1.10:8443", llave: process.env.PRISMA_LLAVE });
await prisma.imprimir("caja", (d) => d.texto("Desde el servidor").cortar());
```

- Para que Node confíe en el certificado autofirmado del puente, descárgalo del panel (*Ajustes → HTTPS → Descargar certificado*) y arranca con `NODE_EXTRA_CA_CERTS=prisma.crt node app.js`.
- Desde un servidor no hay `Origin`: el nivel 2 se identifica solo con la llave, y el nivel 3 con la firma.

## Utilidades

| Función | Descripción |
|---|---|
| `Prisma.validarDocumento(doc)` | Valida un documento v1; devuelve `null` o el `PrismaError`. |
| `Prisma.generarId()` | UUID v4 en minúsculas. Funciona también en páginas `http://`, donde no existe `crypto.randomUUID`. |
| `Prisma.pngABase64(archivo)` | Lee un `File`, `Blob` o bytes PNG y devuelve su base64 para `imagen`. |
| `Prisma.aBase64(bytes)` | Codifica bytes en base64 estándar. |
| `Prisma.VERSION_SDK` | Versión del SDK. |

Con módulos ES también se importan por nombre: `import { Prisma, Documento, PrismaError, validarDocumento } from "./prisma.mjs"`.
