# Problemas comunes

Cada sección empieza con el código que verás en `e.codigo` (SDK) o en la respuesta del puente.

## Certificado no confiable

**Síntomas:** `PUENTE_INALCANZABLE` con `causas_posibles` que incluye `CERTIFICADO_NO_CONFIABLE` (navegador), `CERTIFICADO_NO_CONFIABLE` (Node.js), o «La conexión no es privada» al abrir el puente.

El puente usa HTTPS con un certificado autofirmado: cada equipo que imprime debe confiar en él **una vez**. El navegador no deja que una página distinga «certificado no confiable» de «equipo apagado», por eso el SDK da las dos causas.

1. En el equipo que imprime, abre `https://<ip-del-puente>:8443/v1/salud` en una pestaña:
   - si aparece una advertencia de seguridad, es el certificado;
   - si la página no carga, ve a [Puente inalcanzable](#puente-inalcanzable).
2. Descarga el certificado del panel (*Ajustes → HTTPS → Descargar certificado*) e instálalo como explica [tls.md](../tls.md#confiar-en-el-certificado): Windows, macOS, Linux, Firefox, Android o iOS.
3. Recarga tu página.

> Aceptar la advertencia con *Continuar de todos modos* **no basta**: el navegador lo recuerda solo para esa pestaña y solo para navegar. Las peticiones `fetch` de tu página siguen fallando.

**Si el certificado dejó de servir** en equipos que ya lo tenían, es porque cambió la IP del puente y el dueño lo regeneró: hay que volver a instalarlo. Por eso conviene que el equipo del puente tenga IP fija.

**En Node.js:** `NODE_EXTRA_CA_CERTS=prisma.crt node app.js`. **En curl:** `--cacert prisma.crt`. Evita `-k` y `NODE_TLS_REJECT_UNAUTHORIZED=0` fuera de pruebas.

## Origen no autorizado

**Síntomas:** `ORIGEN_NO_AUTORIZADO`. En la consola del navegador: *blocked by CORS policy: No 'Access-Control-Allow-Origin' header*.

El puente solo responde a las páginas que el dueño aprobó. Tu origen ya quedó registrado como solicitud pendiente: pide al dueño que lo apruebe en el panel, en *Solicitudes*.

El origen debe coincidir **exactamente** en esquema, nombre y puerto. Estos son orígenes distintos:

- `http://localhost:3000` y `http://127.0.0.1:3000`;
- `https://pos.mitienda.mx` y `https://www.pos.mitienda.mx`;
- `http://…` y `https://…` del mismo sitio.

Si tu aplicación ya estaba aprobada y de pronto aparece este error, puede que el dueño la haya desactivado.

## Chrome o Edge: permiso de red local

**Síntomas:** `PERMISO_RED_LOCAL_DENEGADO`, o un cuadro del navegador que pregunta si el sitio puede *buscar y conectarse a dispositivos de la red local*.

Chrome y Edge protegen la red local de las páginas de Internet (*Private Network Access*, y en versiones recientes *Local Network Access*). Cuando una página pública llama a una IP privada como `192.168.x.x`:

- el navegador envía un preflight especial, que el puente responde para los orígenes aprobados;
- las versiones recientes además **piden permiso** a la persona la primera vez.

Qué hacer:

- **Si la persona eligió «Bloquear»:** haz clic en el candado (o en el icono de ajustes) junto a la dirección de la página, busca *Red local* o *Dispositivos de la red local*, elige *Permitir* y recarga.
- **Sirve tu página por `https://`.** Las páginas `http://` de Internet pueden quedar bloqueadas sin que se pregunte nada.
- En equipos administrados por la empresa, la política puede venir del administrador de TI.

Si la página está en la misma red local que el puente (por ejemplo `http://192.168.1.20`), estas restricciones no se aplican.

## iPhone, iPad y Safari

- **Certificado:** en iOS no basta con descargarlo. Hay que instalarlo como perfil (*Ajustes → General → VPN y gestión de dispositivos*) **y** activarlo en *Ajustes → General → Información → Ajustes de confianza de certificados*. Sin el segundo paso, Safari sigue sin confiar. Ver [tls.md](../tls.md#confiar-en-el-certificado).
- **Contenido mixto:** Safari bloquea sin avisar las llamadas de una página `https://` a un puente `http://`. El SDK lo detecta como `CONTENIDO_MIXTO`. Usa la dirección `https://` del puente.
- **Misma red Wi-Fi:** el teléfono debe estar en la red del negocio, no en datos móviles ni en una red de invitados aislada. Prueba abriendo `https://<ip-del-puente>:8443/v1/salud` en Safari.
- **Todos los navegadores de iOS usan el motor de Safari.** Chrome o Firefox en iPhone se comportan igual.

## Puente inalcanzable

**Síntomas:** `PUENTE_INALCANZABLE` o `PUENTE_SIN_RESPUESTA`.

1. ¿El equipo del puente está encendido y PRISMA en marcha? El dueño lo ve en el panel.
2. ¿La IP y el puerto son los correctos? Usa la dirección que muestra el panel en *Inicio* o en *Ajustes → HTTPS*.
3. ¿Tu equipo está en la misma red? Las redes de invitados suelen aislar a los equipos entre sí.
4. ¿El firewall del equipo del puente permite el puerto 8443?

El SDK reintenta solo estos errores, con el mismo `id`: si el puente vuelve, el ticket se imprime **una sola vez**.

## Bluetooth ocupado

**Síntomas:** `PUERTO_OCUPADO` (503) al imprimir en una impresora Bluetooth, con `trabajo_id`.

La impresora o su puerto serie ya están en uso:

- **Otro dispositivo está conectado a la impresora.** Casi todas las impresoras Bluetooth aceptan una sola conexión a la vez; si un celular o una tablet se conectó (por ejemplo, con la aplicación del fabricante), el equipo del puente no puede.
- **Otro programa del equipo del puente tiene abierto el puerto COM:** la utilidad del fabricante, otro sistema de punto de venta o un monitor de puerto serie.

El dueño debe desconectar el otro dispositivo o cerrar el otro programa (ver [conectores.md](../conectores.md#impresoras-bluetooth)). Como el trabajo falló, su `id` quedó libre: puedes **reenviar el mismo documento** sin riesgo de que salga dos veces.

Los otros errores de impresora llegan igual, con `trabajo_id`, y también liberan el `id`:

| Código | Significado |
|---|---|
| `IMPRESORA_INALCANZABLE` | Apagada, sin red o fuera del alcance del Bluetooth. |
| `TIEMPO_AGOTADO` | Dejó de recibir datos a mitad del ticket (papel atascado, tapa abierta o Bluetooth fuera de alcance) o, si el perfil tiene *Confirmar impresión*, no confirmó que lo recibió. |
| `COLA_LLENA` | Tiene 20 trabajos esperando; espera `Retry-After` segundos. |
| `PUENTE_OCUPADO` | Otro programa tiene bloqueada la base de datos del puente. Nada se imprimió ni quedó reservado; espera `Retry-After` segundos y reenvía. |

## «Impreso», pero no salió

Con impresoras de red, el puente da un ticket por impreso cuando lo entregó al sistema, no cuando la impresora lo recibió (así funciona el puerto 9100). Si la impresora se apaga justo mientras recibe un ticket corto, la respuesta es `ok` y el papel no sale.

- **Solución:** el dueño activa *Confirmar impresión* en el perfil de la impresora (ver [conectores.md](../conectores.md#confirmar-impresión)). Con ella, el puente pregunta a la impresora si recibió el ticket y, si no responde, falla con `TIEMPO_AGOTADO` para que reintentes.
- **Advertencia `IMPRESORA_FUERA_DE_LINEA`:** solo aparece con esa opción. Llega en `advertencias` de una respuesta `ok`: la impresora tiene el ticket, pero está con la tapa abierta o sin papel, y lo imprimirá al resolverlo. Muéstraselo a la persona y **no reenvíes** el ticket.
- **USB y CUPS:** no admiten la confirmación. Con USB por winspool, Windows informa el error si el spooler no puede entregar el trabajo.

## Llaves y firmas

| Código | Causa habitual |
|---|---|
| `LLAVE_REQUERIDA` | La aplicación es de nivel 2 y no enviaste `llave` (o se perdió el encabezado `Authorization` en un proxy). |
| `LLAVE_INVALIDA` | Llave incompleta al copiarla, regenerada en el panel (la anterior deja de valer) o de otra aplicación. |
| `FIRMA_EXPIRADA` | El documento firmado tardó más de su vigencia en llegar, o el reloj de tu servidor está atrasado. Sincronízalo por NTP. |
| `FIRMA_INVALIDA` | El documento se modificó después de firmarlo. Prueba tu firmador con el [vector fijo](../../testdata/vector-firma/LEEME.md). |

## El ticket no se imprimió dos veces

**Síntoma:** la respuesta trae `"duplicado": true` y no salió nada nuevo.

Enviaste el mismo `id` con el mismo contenido en los últimos 10 minutos, y el puente lo trató como un reintento. Cada ticket nuevo necesita un `id` nuevo. El SDK lo genera solo en cada `imprimir()`; esto pasa si reutilizas un mismo `Documento` u objeto JSON para tickets distintos.

## El documento no pasa la validación

**Síntomas:** `DOCUMENTO_INVALIDO`, `FILA_NO_CABE`, `CODIGO_BARRAS_INVALIDO`… con `status: 0` (lo detectó el SDK) o 4xx (lo detectó el puente).

`e.detalles` dice qué campo falló y por qué:

```js
catch (e) {
  for (const d of e.detalles ?? []) console.log(d.ruta, d.mensaje);
  // /operaciones/2/columnas  Los porcentajes de las columnas suman 120; el máximo es 100.
}
```

`FILA_NO_CABE` y `OPERACION_NO_SOPORTADA` también pueden venir del puente aunque el SDK no los vea: dependen de la impresora (sus columnas, si tiene imágenes). Prueba con `previsualizar()` en esa impresora.
