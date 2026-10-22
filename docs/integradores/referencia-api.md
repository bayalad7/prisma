# Referencia de la API HTTP v1

Para llamar al puente sin el SDK: desde otro lenguaje, desde curl o para entender qué hace el SDK por dentro. El cuerpo de `imprimir` y `previsualizar` es un documento v1, especificado en [contrato-documento-v1.md](../contrato-documento-v1.md).

- **Base:** `https://<ip-del-puente>:8443/v1/`. El puerto lo elige el negocio; el del ejemplo es el de fábrica.
- **Formato:** JSON en UTF-8. Solo `previsualizar` responde HTML.
- **TLS:** siempre HTTPS. La única excepción es un puente configurado para aceptar HTTP solo desde su propio equipo (`http://localhost`).

| Método y ruta | Autorización | Respuesta |
|---|---|---|
| `GET /v1/salud` | ninguna | `{"ok": true, "version": "0.6.0", "nombre_puente": "Café Prisma"}` |
| `GET /v1/impresoras` | origen o llave | Las impresoras activas que tu aplicación puede usar. |
| `POST /v1/imprimir` | origen, llave o firma | El resultado del trabajo. |
| `POST /v1/previsualizar` | igual que imprimir | HTML con la simulación exacta del ticket. No imprime. |

## Autorización

| Encabezado | Cuándo |
|---|---|
| `Origin` | Lo envía el navegador solo. Debe pertenecer a tu aplicación (niveles 1 y 2, y nivel 3 si el navegador entrega el documento). |
| `Authorization: Bearer pk_…` | Nivel 2, y nivel 3 si la aplicación exige llave. Si se envía, debe ser válida aunque el nivel no la exija. |
| Campos `cliente_id`, `emitido`, `expira` y `firma` del documento | Nivel 3. |

El detalle de cada nivel y el orden de las comprobaciones están en [niveles-de-seguridad.md](niveles-de-seguridad.md#cómo-decide-el-puente).

## `GET /v1/impresoras`

```json
[
  {"alias": "caja",   "nombre": "Caja principal", "ancho_mm": 80, "estado": "en_linea"},
  {"alias": "cocina", "nombre": "Cocina",         "ancho_mm": 58, "estado": "fuera_de_linea"}
]
```

`estado` es `en_linea`, `fuera_de_linea`, `ocupada` o `desconocido`. El puente prueba cada impresora cada 30 s.

## `POST /v1/imprimir`

La respuesta llega cuando la impresora terminó.

| Estado | Cuerpo | Significado |
|---|---|---|
| 200 | `{"ok": true, "trabajo_id": "trb_…", "impresora": "caja", "duracion_ms": 412, "advertencias": […]}` | Impreso. |
| 200 | `{… "duplicado": true, "estado": "completado"}` | El mismo `id` con el mismo contenido llegó hace menos de 10 min. **No se volvió a imprimir.** |
| 202 | `{"ok": true, "trabajo_id": "trb_…", "estado": "encolado", …}` | La impresora tardó más de 60 s en atenderlo (tenía trabajos por delante). Se imprimirá. |
| 4xx/5xx | `{"ok": false, "codigo": "…", "mensaje": "…", "sugerencia": "…", "trabajo_id"?: "…"}` | Error. Ver la tabla de códigos. |

Las `advertencias` no impiden imprimir. Por ejemplo, `CARACTER_SUSTITUIDO` indica que un carácter no existía en la página de códigos de la impresora, y `SIN_CUCHILLA` que la impresora no corta. `IMPRESORA_FUERA_DE_LINEA` (solo en impresoras con *Confirmar impresión*) indica que la impresora recibió el ticket pero tiene la tapa abierta o no tiene papel: saldrá al resolverlo, así que **no lo reenvíes**.

### Reintentos e idempotencia

Genera un `id` (UUID v4) nuevo para cada ticket y **reutilízalo** en los reintentos de ese mismo ticket. El puente recuerda cada `id` durante 10 minutos:

- mismo `id` y mismo contenido: responde el trabajo original con `duplicado: true`, sin imprimir;
- mismo `id` con otro contenido: `409 ID_REUTILIZADO`;
- si el trabajo original **falló** (impresora apagada, sin papel…), el `id` queda libre y puedes reenviar el mismo documento.

Reintenta solo cuando no hubo respuesta (error de red o tiempo agotado). Un error con código es una respuesta del puente: no cambiará por insistir.

## `POST /v1/previsualizar`

Misma autorización que `imprimir`, sin idempotencia. Responde un fragmento HTML (`text/html`) autocontenido: todo el texto va escapado, con estilos en línea e imágenes `data:`. Insértalo en un contenedor o en un `<iframe srcdoc>`.

## CORS y acceso a la red local

| Situación | Qué hace el puente |
|---|---|
| Preflight (`OPTIONS`) de un origen aprobado | 204 con `Access-Control-Allow-Origin: <tu origen exacto>`, `Allow-Methods: GET, POST, OPTIONS`, `Allow-Headers: Content-Type, Authorization` y `Max-Age: 600`. |
| Origen aprobado | Cada respuesta repite tu origen exacto en `Access-Control-Allow-Origin` (nunca `*`), con `Vary: Origin`. Expone `Retry-After`. |
| Origen desconocido | 403 **sin** cabeceras CORS, y el origen queda como solicitud pendiente en el panel. |
| Private Network Access (`Access-Control-Request-Private-Network: true`) | Responde `Access-Control-Allow-Private-Network: true` a los orígenes aprobados. |

Consecuencias en el navegador:

- **Un origen no aprobado se ve como un error de red** (`TypeError: Failed to fetch`), no como un 403: el navegador no deja leer la respuesta. El SDK lo distingue haciendo una petición sin CORS a `/v1/salud`. Si esa responde, el problema es la autorización del origen.
- **Página pública → puente en la red local:** Chrome y Edge aplican *Private Network Access* / *Local Network Access*. El puente responde al preflight que piden, y las versiones recientes además piden permiso a la persona. Ver [problemas-comunes.md](problemas-comunes.md#chrome-o-edge-permiso-de-red-local).
- **Página `https://` → puente `http://`:** el navegador lo bloquea por contenido mixto, salvo con `localhost`.

## Límites

| Límite | Valor | Respuesta |
|---|---|---|
| Documento | 256 KB (262 144 bytes) | 413 `DOCUMENTO_DEMASIADO_GRANDE` |
| Operaciones por documento | 500 | 400 `DEMASIADAS_OPERACIONES` |
| Peticiones por IP | 120 por minuto (ajustable por el dueño) | 429 `DEMASIADAS_PETICIONES` + `Retry-After` |
| Peticiones por aplicación | 60 por minuto (ajustable por el dueño) | 429 `DEMASIADAS_PETICIONES` + `Retry-After` |
| Trabajos en espera por impresora | 20 | 503 `COLA_LLENA` + `Retry-After` |
| Vigencia de un documento firmado | 120 s | 401 `FIRMA_INVALIDA` |
| Ventana de idempotencia | 10 min | ver arriba |

Los límites de peticiones admiten ráfagas: con 60 por minuto puedes enviar 60 seguidas y luego una por segundo.

## Códigos de error

Programa contra `codigo`, que es estable. `mensaje` y `sugerencia` están en español y pensados para mostrarlos a la persona; pueden cambiar.

### Del documento

| Código | HTTP | Cuándo |
|---|---|---|
| `JSON_INVALIDO` | 400 | El cuerpo no es JSON, o repite una clave. |
| `VERSION_NO_SOPORTADA` | 400 | `version` falta o no es `"1"`. Incluye `versiones_soportadas`. |
| `OPERACION_DESCONOCIDA` | 400 | Un `tipo` no existe. |
| `DEMASIADAS_OPERACIONES` | 400 | Más de 500 operaciones. |
| `DOCUMENTO_INVALIDO` | 400 | Cualquier otra regla del contrato. `detalles` trae `{ruta, regla, mensaje}` por cada falla; `ruta` es un JSON Pointer. |
| `CODIGO_BARRAS_INVALIDO` | 400 | El dígito verificador no coincide. |
| `IMAGEN_INVALIDA` | 400 | El PNG está dañado o supera 100 KB o 4096 px. |
| `FIRMA_REQUERIDA` · `FIRMA_INVALIDA` · `FIRMA_EXPIRADA` | 401 | Nivel 3: falta la firma, no verifica o venció. |
| `RECURSO_NO_ENCONTRADO` | 404 | La imagen guardada no existe en el puente. |
| `ID_REUTILIZADO` | 409 | El mismo `id` llegó con otro contenido. |
| `FILA_NO_CABE` | 422 | Las columnas de una `fila` no caben en esa impresora. |
| `OPERACION_NO_SOPORTADA` | 422 | La impresora no puede ejecutar una operación (por ejemplo, una imagen en una impresora sin imágenes). |
| `DOCUMENTO_DEMASIADO_GRANDE` | 413 | El cuerpo supera 256 KB. |

### De autorización, permisos e impresora

| Código | HTTP | Cuándo |
|---|---|---|
| `ORIGEN_NO_AUTORIZADO` | 403 | El origen no pertenece a ninguna aplicación (queda pendiente en el panel). |
| `ORIGEN_REQUERIDO` | 403 | Una aplicación de nivel 1 llamó desde fuera de un navegador. |
| `CREDENCIALES_REQUERIDAS` | 401 | Sin origen, sin llave y sin documento firmado. |
| `LLAVE_REQUERIDA` · `LLAVE_INVALIDA` | 401 | Falta la llave, o es incorrecta o de otra aplicación. |
| `CLIENTE_DESACTIVADO` | 403 | El dueño desactivó tu aplicación. |
| `IMPRESORA_NO_PERMITIDA` · `OPERACION_NO_PERMITIDA` | 403 | Tu aplicación no puede usar esa impresora, o `abrir_cajon`/`raw`. |
| `IMPRESORA_NO_ENCONTRADA` | 404 | No hay ninguna impresora activa con ese alias. |
| `DEMASIADAS_PETICIONES` | 429 | Límite de peticiones; espera `Retry-After` segundos. |
| `IMPRESORA_INALCANZABLE` · `PUERTO_OCUPADO` · `PERMISO_DENEGADO` · `COLA_LLENA` | 503 | La impresora no pudo imprimir. Incluye `trabajo_id`. |
| `TIEMPO_AGOTADO` | 504 | La impresora dejó de recibir datos a mitad del ticket o no confirmó que lo recibió. Incluye `trabajo_id`. |
| `PUENTE_NO_DISPONIBLE` | 503 | El puente se está deteniendo. |
| `PUENTE_OCUPADO` | 503 | Otro programa tiene bloqueada la base de datos del puente. No se imprimió nada; reintenta tras `Retry-After` (2 s). |
| `RUTA_NO_ENCONTRADA` · `METODO_NO_PERMITIDO` | 404 · 405 | Ruta o método inexistente. |
| `ERROR_INTERNO` | 500 | Error inesperado; el detalle queda en la bitácora del puente. |

Los códigos que genera el propio SDK, cuando no hay respuesta del puente, están en [referencia-sdk.md](referencia-sdk.md#errores).

## Ejemplos sin SDK

**Nivel 2 con curl** (`--cacert` con el certificado que descargas del panel, en *Ajustes → HTTPS*):

```sh
curl --cacert prisma.crt https://192.168.1.10:8443/v1/imprimir \
  -H "Authorization: Bearer pk_7ax6nj4r4a_…" \
  -H "Content-Type: application/json" \
  --data-binary @ticket.json
```

**Nivel 2 con `fetch`:**

```js
const r = await fetch("https://192.168.1.10:8443/v1/imprimir", {
  method: "POST",
  headers: { "Content-Type": "application/json", Authorization: "Bearer pk_7ax6nj4r4a_…" },
  body: JSON.stringify(documento),
});
const resultado = await r.json();
if (!resultado.ok) console.error(resultado.codigo, resultado.mensaje, resultado.sugerencia);
```
