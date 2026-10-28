# Códigos de error y advertencias

Todos los códigos que pueden aparecer en PRISMA 0.1.0, con el estado HTTP con que llegan y dónde se originan. Programa contra `codigo`, que es estable; `mensaje` y `sugerencia` están en español, pensados para mostrarlos a la persona, y pueden cambiar.

Esta página está verificada por una prueba (`go test ./docs`): cada código del puente y del SDK debe aparecer aquí.

## Forma del error

```json
{
  "ok": false,
  "codigo": "DOCUMENTO_INVALIDO",
  "mensaje": "El documento no cumple el contrato v1.",
  "sugerencia": "Revisa los detalles; cada uno indica el campo y la regla que falló.",
  "detalles": [{"ruta": "/operaciones/3/alineacion", "regla": "enum", "mensaje": "debe ser uno de: izquierda, centro, derecha"}]
}
```

| Campo | Cuándo aparece |
|---|---|
| `codigo`, `mensaje`, `sugerencia` | Siempre. |
| `detalles` | Errores del documento: un `{ruta, regla, mensaje}` por falla; `ruta` es un JSON Pointer. |
| `trabajo_id` | Errores de la impresora, al imprimir. |
| `versiones_soportadas` | `VERSION_NO_SOPORTADA`. |

Los 429 y los 503 de cola llena o base ocupada llevan además la cabecera `Retry-After`.

## API pública `/v1`

### Del documento

Los genera `internal/contrato` (validación) o `internal/escpos` (perfil de la impresora). Ver el [contrato §9](contrato-documento-v1.md#9-validación-y-errores) para el orden en que se comprueban.

| Código | HTTP | Cuándo |
|---|---|---|
| `DOCUMENTO_DEMASIADO_GRANDE` | 413 | El cuerpo supera 256 KB. |
| `JSON_INVALIDO` | 400 | El cuerpo no es JSON, no es UTF-8 o repite una clave. |
| `VERSION_NO_SOPORTADA` | 400 | `version` falta o no es `"1"`. Incluye `versiones_soportadas`. |
| `OPERACION_DESCONOCIDA` | 400 | Un `tipo` de operación no existe. |
| `DEMASIADAS_OPERACIONES` | 400 | Más de 500 operaciones. |
| `DOCUMENTO_INVALIDO` | 400 | Cualquier otra regla del esquema. Trae `detalles`. |
| `CODIGO_BARRAS_INVALIDO` | 400 | El dígito verificador no coincide. |
| `IMAGEN_INVALIDA` | 400 | El PNG está dañado o supera 100 KB o 4096 px por lado. |
| `FIRMA_REQUERIDA` | 401 | Nivel 3 sin firma. |
| `FIRMA_INVALIDA` | 401 | La firma no verifica, el `cliente_id` no coincide o la vigencia no está entre 0 y 120 s. |
| `FIRMA_EXPIRADA` | 401 | Pasó `expira` (con 30 s de tolerancia de reloj). |
| `RECURSO_NO_ENCONTRADO` | 404 | La imagen guardada (`recurso`) no existe en el puente. |
| `ID_REUTILIZADO` | 409 | El mismo `id` llegó con otro contenido en los últimos 10 minutos. |
| `FILA_NO_CABE` | 422 | Las columnas de una `fila` no caben en esa impresora, o los porcentajes suman más de 100. |
| `OPERACION_NO_SOPORTADA` | 422 | La impresora no puede ejecutar una operación (por ejemplo, una imagen en una impresora sin imágenes). |

### De autorización y permisos

Los genera `internal/seguridad`. El orden de las comprobaciones está en [api-v1.md](api-v1.md#orden-de-las-comprobaciones).

| Código | HTTP | Cuándo |
|---|---|---|
| `ORIGEN_NO_AUTORIZADO` | 403 | El `Origin` no pertenece a ninguna aplicación activa. Queda como solicitud pendiente en el panel. |
| `ORIGEN_REQUERIDO` | 403 | Una aplicación de nivel 1 llamó sin `Origin` (desde fuera de un navegador). |
| `CREDENCIALES_REQUERIDAS` | 401 | Sin origen, sin llave y sin documento firmado. |
| `LLAVE_REQUERIDA` | 401 | Nivel 2 (o nivel 3 que la exige) sin `Authorization`, o aplicación de nivel 2 cuya llave se revocó. |
| `LLAVE_INVALIDA` | 401 | Llave mal formada, desconocida, incorrecta o de otra aplicación. |
| `CLIENTE_DESACTIVADO` | 403 | El dueño desactivó la aplicación. |
| `IMPRESORA_NO_PERMITIDA` | 403 | La aplicación no puede usar esa impresora. |
| `OPERACION_NO_PERMITIDA` | 403 | El documento usa `abrir_cajon` o `raw` y la aplicación no las tiene habilitadas. |
| `DEMASIADAS_PETICIONES` | 429 | Se superó el límite por IP o por aplicación. Lleva `Retry-After`. |

### De la impresora

Los genera `internal/conector` y `internal/cola`. Incluyen `trabajo_id`, y el `id` del documento queda libre para reintentar.

| Código | HTTP | Cuándo |
|---|---|---|
| `IMPRESORA_NO_ENCONTRADA` | 404 | No hay ninguna impresora activa con ese alias, o su destino (nombre, cola, puerto) no existe en el sistema. |
| `IMPRESORA_INALCANZABLE` | 503 | Está apagada, desconectada, en otra red, rechazó la conexión o está marcada sin conexión. |
| `PUERTO_OCUPADO` | 503 | Otro programa tiene abierto el puerto serie o el dispositivo. |
| `PERMISO_DENEGADO` | 503 | El usuario del servicio no puede abrir el destino. |
| `COLA_LLENA` | 503 | La impresora ya tiene 20 trabajos pendientes. Lleva `Retry-After`. |
| `TIEMPO_AGOTADO` | 504 | La impresora dejó de recibir datos a mitad del ticket o, con Confirmar impresión, no confirmó que lo recibió. |

### Del puente

| Código | HTTP | Cuándo |
|---|---|---|
| `PUENTE_NO_DISPONIBLE` | 503 | El puente se está deteniendo. |
| `PUENTE_OCUPADO` | 503 | Otro programa tiene bloqueada la base de datos (`SQLITE_BUSY` o `SQLITE_LOCKED` tras 5 s). No se imprimió ni se reservó nada. Lleva `Retry-After: 2`. |
| `RUTA_NO_ENCONTRADA` | 404 | La ruta no existe. |
| `METODO_NO_PERMITIDO` | 405 | La ruta existe pero no acepta ese método. Lleva `Allow`. |
| `ERROR_INTERNO` | 500 | Error inesperado. El detalle queda solo en la bitácora del puente. |

## Advertencias

No impiden imprimir. Llegan en `advertencias` de una respuesta `ok` y quedan guardadas con el trabajo.

| Código | Origen | Significado |
|---|---|---|
| `CARACTER_SUSTITUIDO` | `internal/escpos` | Un carácter no existía en la página de códigos de la impresora; se imprimió sin acento o como `?`. |
| `SIN_CUCHILLA` | `internal/escpos` | La impresora no tiene cuchilla: `cortar` se reemplazó por un avance de papel. |
| `SIN_CAJON` | `internal/escpos` | La impresora no tiene cajón: `abrir_cajon` se omitió. |
| `QR_REDUCIDO` | `internal/escpos` | El QR no cabía con el `tamano` pedido y se redujo. |
| `IMPRESORA_FUERA_DE_LINEA` | `internal/conector` | Solo con Confirmar impresión: la impresora recibió el ticket, pero tiene la tapa abierta, no tiene papel o está en error. Saldrá al resolverlo: **no lo reenvíes**. |

## Estado de un trabajo

Códigos que solo aparecen en el historial de trabajos (panel, *Trabajos*), no en una respuesta:

| Código | Significado |
|---|---|
| `PUENTE_REINICIADO` | El puente se detuvo (corte de luz, reinicio) con el trabajo en cola o imprimiéndose. El trabajo quedó como fallido y su `id` se liberó: el reintento de la aplicación lo imprime. |

## SDK JavaScript

Los genera el propio SDK cuando no hay una respuesta útil del puente. Llegan como `PrismaError`, con `status` 0. Detalles y causas en [integradores/sdk.md](../integradores/sdk.md#errores).

| Código | Cuándo |
|---|---|
| `CONFIGURACION_INVALIDA` | URL o llave mal formadas, o contenido de `imprimir` que no es función, `Documento` ni arreglo. |
| `ORIGEN_NO_AUTORIZADO` | Desde el navegador: el puente responde, pero sin CORS para tu página. |
| `PERMISO_RED_LOCAL_DENEGADO` | El navegador bloqueó el acceso de la página a la red local. |
| `CERTIFICADO_NO_CONFIABLE` | Node.js no confía en el certificado del puente. |
| `PUENTE_INALCANZABLE` | No hubo conexión. Desde el navegador trae `causas_posibles`: `CERTIFICADO_NO_CONFIABLE` o `PUENTE_APAGADO_O_FUERA_DE_RED`. |
| `PUENTE_SIN_RESPUESTA` | Conectó, pero no respondió a tiempo. |
| `CONTENIDO_MIXTO` | Página `https://` y puente `http://` fuera de localhost. |
| `RESPUESTA_INVALIDA` | Respondió algo que no es un puente PRISMA. |

El SDK también valida el documento antes de enviarlo y lanza los mismos códigos del documento que el puente (`DOCUMENTO_INVALIDO`, `DOCUMENTO_DEMASIADO_GRANDE`, `FIRMA_INVALIDA`…).

## Panel de administración

Los genera `internal/admin` en la API interna `/admin/api`, que **no es pública**. Se documentan porque el dueño los ve en el panel.

| Código | HTTP | Cuándo |
|---|---|---|
| `ACCESO_SOLO_RED_LOCAL` | 403 | El panel se abrió desde una IP que no es de la red local, con *Solo red local* activado. |
| `ACCESO_BLOQUEADO` | 429 | 5 contraseñas incorrectas desde esa IP: queda bloqueada 5 minutos. |
| `CONTRASENA_INCORRECTA` | 401 | La contraseña no coincide. |
| `SESION_REQUERIDA` | 401 | No hay sesión, o venció (30 min de inactividad, 12 h, reinicio del puente o cambio de contraseña). |
| `CSRF_INVALIDO` | 403 | Falta el token CSRF o la petición no viene del mismo origen que el panel. |
| `CAMBIO_DE_CONTRASENA_REQUERIDO` | 403 | La contraseña se restableció con la CLI y hay que cambiarla antes de seguir. |
| `SIN_CONFIGURAR` | 409 | El panel aún no tiene contraseña. |
| `YA_CONFIGURADO` | 409 | El panel ya tiene contraseña: la configuración inicial ya se hizo. |
| `CODIGO_REQUERIDO` | 403 | Desde otro equipo hace falta el código de configuración, y no hay uno vigente: nunca se generó o venció a los 30 minutos. Genéralo con `prisma admin codigo-inicial`. |
| `CODIGO_INVALIDO` | 403 | El código de configuración no coincide. |
| `CODIGO_ANULADO` | 403 | Hubo 10 intentos con códigos incorrectos y el código se anuló: genera otro. |
| `DATOS_INVALIDOS` | 400 | Un campo del formulario no es válido; el mensaje dice cuál. |
| `IMAGEN_INVALIDA` | 400 | El archivo subido en *Imágenes* no es un PNG válido. |
| `LLAVE_PUBLICA_INVALIDA` | 400 | La llave pública de nivel 3 no es Ed25519 en PEM ni en base64. |
| `RESPALDO_INVALIDO` | 400 | El archivo subido en *Restaurar* no es un respaldo válido de PRISMA. |
| `ALIAS_DUPLICADO` | 409 | Ya existe una impresora con ese alias. |
| `ORIGEN_DUPLICADO` | 409 | El origen ya pertenece a otra aplicación. |
| `DUPLICADO` | 409 | Ya existe un registro con ese nombre (por ejemplo, una imagen). |
| `IMPRESORA_INACTIVA` | 409 | Se pidió imprimir desde el panel (diagnóstico o páginas de códigos) en una impresora desactivada. |
| `IMPRESORA_NO_VIRTUAL` | 409 | Se pidió el rollo de una impresora que no es virtual: sus tickets salen en papel y el puente no los guarda. |
| `TICKET_NO_ENCONTRADO` | 404 | El ticket ya no está en el rollo de la impresora virtual: solo recuerda los últimos y los olvida al reiniciar el puente. |
| `TLS_NO_AUTOFIRMADO` | 409 | Solo se puede regenerar el certificado autofirmado. |
| `SIN_CERTIFICADO` | 404 | No hay archivo de certificado que descargar (por ejemplo, en el modo `solo_localhost_http`). |
| `SIN_COMPROBADOR` | 409 | *Comprobar ahora* en una compilación de desarrollo, que no busca versiones nuevas. |
| `CLIENTE_NO_ENCONTRADO` | 404 | La aplicación ya no existe. |
| `SOLICITUD_NO_ENCONTRADA` | 404 | La solicitud de origen ya no existe. |
| `NO_ENCONTRADO` | 404 | Lo que se buscaba ya no existe. |
| `RUTA_NO_ENCONTRADA` | 404 | La ruta no existe en la API del panel. |
