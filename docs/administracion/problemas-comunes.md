# Problemas comunes: impresión y panel

Cuando algo falla, PRISMA muestra un código y una **sugerencia**. Todos los códigos están en [referencia/codigos-de-error.md](../referencia/codigos-de-error.md).

- Los problemas al instalar o al arrancar el servicio están en [instalacion/problemas-comunes.md](../instalacion/problemas-comunes.md).
- Los que ve una aplicación (certificado, CORS, permiso de red local, Safari) están en [integradores/problemas-comunes.md](../integradores/problemas-comunes.md).
- Recuperarse de una falla grave (base dañada, certificado vencido, restauración fallida) está en [operacion/recuperacion.md](../operacion/recuperacion.md).

## La aplicación no puede imprimir (API)

### El navegador dice «NET::ERR_CERT_AUTHORITY_INVALID» o «la conexión no es privada»

El equipo no confía en el certificado autofirmado del puente. Instálalo como confiable siguiendo [instalacion/https-y-certificados.md](../instalacion/https-y-certificados.md). Una excepción temporal del navegador no sirve: las llamadas `fetch` de la aplicación siguen fallando.

### Error de certificado con la IP correcta

Si el equipo del puente cambió de IP, el certificado no la incluye: el puente lo avisa al arrancar, y `prisma tls info` y el panel (*Ajustes → HTTPS*) lo muestran. Reinicia el servicio (desde 0.3.1, el puente lo renueva solo al arrancar) o regenéralo en el panel o con `prisma tls regenerar`. Lo firma la misma autoridad local, así que los equipos no tienen que hacer nada; con un certificado de antes de 0.3.1, hay que volver a confiar en él. Para que no se repita, dale una IP fija al equipo.

### `fetch` falla con «TypeError: Failed to fetch» y la consola menciona CORS

El origen de la página no está aprobado. El puente no le envía cabeceras CORS y el navegador no deja leer la respuesta. El origen quedó como **solicitud pendiente**. Apruébalo:
- en el [panel](README.md), sección *Solicitudes*;
- o con la CLI: `prisma clientes agregar … --origen https://tu-app.example`.

El origen debe coincidir **exactamente**: esquema (`https`), nombre y puerto.

### 401 o 403 al imprimir

| Código | Qué falta |
|---|---|
| `LLAVE_REQUERIDA` | La aplicación es de nivel 2 y no envió `Authorization: Bearer pk_…`. |
| `LLAVE_INVALIDA` | La llave es incorrecta o se regeneró. Pide la nueva (`prisma clientes regenerar-llave`). |
| `CREDENCIALES_REQUERIDAS` | La petición no trae origen, llave ni documento firmado. Desde un servidor, envía la llave. |
| `ORIGEN_REQUERIDO` | La aplicación es de nivel 1: solo puede imprimir desde un navegador. |
| `FIRMA_INVALIDA` | El documento cambió después de firmarlo, se firmó con otra llave o `cliente_id` no corresponde. Prueba tu firmador con [testdata/vector-firma](../../testdata/vector-firma/LEEME.md). |
| `FIRMA_EXPIRADA` | Pasaron más de `expira` + 30 s. Firma justo antes de enviar y revisa la hora del servidor que firma. |
| `IMPRESORA_NO_PERMITIDA` · `OPERACION_NO_PERMITIDA` | La aplicación no tiene permiso para esa impresora o para `abrir_cajon` / `raw`. |

La referencia completa de errores está en [referencia/codigos-de-error.md](../referencia/codigos-de-error.md).

### La respuesta dice `"duplicado": true` y no imprimió

El documento llegó con el mismo `id` que otro de los últimos 10 minutos. Es la protección contra los tickets dobles. Genera un `id` (UUID v4) nuevo para cada ticket y reutilízalo solo al reintentar ese mismo ticket. Si el trabajo original **falló**, el `id` queda libre y el reintento sí imprime.

### 429 `DEMASIADAS_PETICIONES`

Se superó el límite de peticiones por minuto. Respeta `Retry-After`, o ajusta `[limites]` en [configuracion.md](../referencia/configuracion.md).

### 503 `PUENTE_OCUPADO`

Otro programa tiene bloqueada la base de datos del puente, por ejemplo un visor de SQLite abierto sobre `prisma.db` o una copia de respaldo hecha con otra herramienta. No se imprimió nada. Cierra ese programa; el puente vuelve a responder solo. Para copiar la base usa *Ajustes → Respaldo*, que no la bloquea.

### Todas las aplicaciones dejaron de conectarse el mismo día

Lo más probable es que **venciera el certificado** del puente. *Inicio* lo avisa desde 60 días antes. Con el certificado autofirmado, el puente genera uno nuevo al arrancar: reinicia el servicio. Desde 0.3.1, lo firma la misma autoridad local y los equipos no tienen que hacer nada; con un certificado anterior, vuelve a instalarlo en cada equipo que imprime ([instalacion/https-y-certificados.md](../instalacion/https-y-certificados.md#vencimiento)).

## La impresora no imprime

| Código | Causa típica | Qué hacer |
|---|---|---|
| `IMPRESORA_NO_ENCONTRADA` | El alias no existe en el puente, o el destino (nombre, cola, puerto) no existe en el sistema. | `prisma impresoras listar` y `prisma impresoras detectar`; el destino debe coincidir exactamente. |
| `IMPRESORA_INALCANZABLE` | Está apagada, desconectada, en otra red o marcada «sin conexión» en Windows. | Revisa encendido, cable o red, y su IP. Ver [instalacion/impresoras.md](../instalacion/impresoras.md). |
| `TIEMPO_AGOTADO` | Dejó de aceptar datos: sin papel, tapa abierta, atasco o Bluetooth fuera de alcance. | Revisa el papel y la tapa; acércate si es Bluetooth. |
| `PUERTO_OCUPADO` | Otro programa tiene abierto el puerto serie o el dispositivo USB. | Cierra el otro programa; en Linux, usa la cola de CUPS. |
| `PERMISO_DENEGADO` | El usuario del servicio no tiene acceso. | Linux: grupos `lp` y `dialout`. Windows: permiso de impresión. |
| `COLA_LLENA` | Hay 20 trabajos pendientes en esa impresora. | La impresora no avanza: revisa el estado con `prisma impresoras probar`. |

### Dice «impreso», pero no salió (impresora de red)

Con impresoras de red, el puente da un ticket por impreso cuando lo entregó al sistema, no cuando la impresora lo recibió. Si la impresora se apaga justo mientras recibe un ticket corto, la aplicación recibe `ok`. Activa **Confirmar impresión** en el perfil de la impresora ([impresoras.md](impresoras.md#confirmar-impresión)).

### Advertencia `IMPRESORA_FUERA_DE_LINEA`

Solo con Confirmar impresión. La impresora recibió el ticket, pero tiene la tapa abierta, no tiene papel o está en error. Lo imprimirá en cuanto se resuelva: **no** lo reenvíes.

### Con Confirmar impresión, todos los tickets fallan con `TIEMPO_AGOTADO`

La impresora no responde a la consulta de estado `DLE EOT`. Desactiva *Confirmar impresión* en su perfil.

### Windows dice que imprimió, pero no salió nada

En USB, Windows acepta el trabajo en su cola de impresión aunque la impresora esté apagada. Abre la cola en *Configuración > Impresoras* y revisa si hay trabajos atascados o si está «sin conexión». `prisma impresoras probar` también lo detecta, si el controlador lo informa.

### Sale texto raro o símbolos en lugar del ticket

La impresora está interpretando los bytes con un controlador que no es RAW, o no habla ESC/POS:

- **Windows:** usa el controlador del fabricante o «Generic / Text Only».
- **Linux y macOS:** crea la cola con `-m raw`.
- **Impresoras de etiquetas:** las que usan ZPL o TSPL no son ESC/POS y no son compatibles.

## El ticket sale, pero no como se espera

### Acentos o ñ salen mal

La página de códigos del perfil no coincide con la que tiene activa la impresora.

1. Imprime la hoja de autoprueba (enciende la impresora con FEED presionado) y busca su tabla de páginas de códigos.
2. Ajusta `--pagina` y `--comando-pagina`, el `n` de `ESC t n` en tu modelo. En Epson y la mayoría de compatibles:

| Página | `--pagina` | `--comando-pagina` |
|---|---|---|
| PC437 (EE. UU.) | `CP437` | 0 |
| PC850 (Multilingüe) | `CP850` | 2 |
| WPC1252 | `WPC1252` | 16 |
| PC858 (con €) | `CP858` | 19 |

En muchos clones chinos los números cambian; por ejemplo, CP858 puede ser 13. Prueba con `ejemplos/validos/todas-las-operaciones.json` hasta que salgan bien `á é í ó ú ñ ¿ ¡ €`.

Si en las advertencias del trabajo aparece **`CARACTER_SUSTITUIDO`**, el carácter no existe en esa página y se imprimió sin acento o como `?`. Elige una página que lo tenga: CP858 o WPC1252 cubren el español y el €.

### Las columnas no cuadran o el texto se parte antes de tiempo

El número de columnas del perfil no es el de la impresora. Muchas de 80 mm usan 48 columnas con la fuente A, pero otras usan 42. Corrígelo así:

```sh
prisma impresoras eliminar caja
prisma impresoras agregar --alias caja … --columnas 42 --columnas-b 56
```

### No corta o no abre el cajón

- **Sin cuchilla:** registra la impresora con `--sin-corte`. El corte se reemplaza por un avance de papel y queda la advertencia `SIN_CUCHILLA`.
- **No abre el cajón:**
  - registra la impresora con `--cajon`;
  - revisa el pin: la operación usa `pin: 2` por defecto, y algunos cajones usan el 5;
  - la aplicación debe tener habilitada la operación (`prisma clientes agregar … --operaciones abrir_cajon`); si no, la API responde `OPERACION_NO_PERMITIDA`.

### El QR o el código de barras no aparece

La impresora no tiene QR o códigos de barras por comando, algo común en modelos económicos. Regístrala con `--sin-qr-nativo` o `--sin-barras-nativo`: PRISMA los enviará como imagen.

Si el código de barras no cabe en el papel, el documento se rechaza con `OPERACION_NO_SOPORTADA`. Acorta el valor o usa una simbología más compacta.

### Errores del documento

`FILA_NO_CABE`, `DOCUMENTO_INVALIDO`, `CODIGO_BARRAS_INVALIDO`, etc. son problemas del JSON, no de la impresora. Cada error indica el campo (`/operaciones/3/columnas`) y la regla que falló. Las reglas están en el [contrato](../referencia/contrato-documento-v1.md), y [ejemplos/invalidos](../../ejemplos/invalidos/) tiene un ejemplo de cada error.

Para ver el resultado antes de imprimir:

```sh
prisma dev previsualizar documento.json --ancho 58
```

## Panel de administración

### El panel responde `ACCESO_SOLO_RED_LOCAL`

Por defecto el panel solo se abre desde IPs privadas de la red local y desde el propio equipo del puente. Si llegas a través de una VPN o de una red con IPs públicas, entra desde el equipo del puente y desactiva *Ajustes → Solo red local*.

### El panel pide un código de configuración

El panel aún no tiene contraseña y lo abriste desde otro equipo. Ejecuta `prisma admin codigo-inicial` en el equipo del puente, o abre `https://localhost:8443/admin` en ese mismo equipo.

### Olvidé la contraseña del panel

En el equipo del puente, ejecuta `prisma admin restablecer-contrasena` con el mismo usuario que el servicio. El panel pedirá cambiarla al entrar.

### `ACCESO_BLOQUEADO` al entrar al panel

Hubo 5 contraseñas incorrectas desde ese equipo. Espera 5 minutos; los demás equipos no quedan bloqueados.

## Diagnóstico avanzado

- **Ver los bytes exactos:** `prisma dev renderizar doc.json -o t.bin` y luego `prisma dev decodificar t.bin`.
- **Imprimir sin hardware:** usa `prisma dev impresora-virtual`, o una impresora `--tipo archivo`, que guarda el `.bin` y un `.txt` con la vista.
- **Bitácoras:**
  - Linux: `journalctl -u prisma -f`.
  - Windows: `C:\ProgramData\PRISMA\logs\prisma.log`.
  - macOS: `/Library/Logs/PRISMA/prisma.log`.

  Son JSON, una línea por evento, y nunca incluyen el contenido de los tickets.
