# Inicio rápido: tu primer ticket en 5 minutos

Al terminar tendrás una página HTML, sin backend ni compilación, que imprime un ticket en la impresora del negocio con el SDK cargado desde un CDN.

## Lo que necesitas

1. **Un puente PRISMA en marcha** en la red, con una impresora configurada.
   - Si el negocio ya tiene uno, pide al dueño su IP y el alias de la impresora.
   - Si no, instálalo en tu propia PC ([instalacion/](../instalacion/README.md)) o compílalo y ejecútalo desde el código ([primeros-pasos.md](../desarrollo/primeros-pasos.md)).
   - Para probar sin hardware, usa la impresora virtual: ejecuta `prisma dev impresora-virtual --puerto 9100` en el mismo equipo del puente y, en el panel, *Impresoras → Agregar* con alias `caja`, tipo **Red** y destino `127.0.0.1:9100`. Cada ticket aparece en esa terminal. También sirve una impresora de tipo *Archivo*, que guarda cada ticket en una carpeta.
2. **Un equipo de esa misma red que confíe en el certificado del puente.** Para comprobarlo, abre `https://<ip-del-puente>:8443/v1/salud` en ese equipo: debe responder `{"ok":true,…}` sin advertencia de seguridad. Si aparece una advertencia, instala el certificado como explica [instalacion/https-y-certificados.md](../instalacion/https-y-certificados.md#confiar-en-el-certificado).
3. **Node.js**, solo para servir la página con `npx serve`. Sirve cualquier otro servidor estático.

## 1. Escribe la página

Guarda esto como `index.html` en una carpeta vacía:

```html
<!doctype html>
<meta charset="utf-8">
<button id="imprimir">Imprimir</button>
<script src="https://cdn.jsdelivr.net/gh/bayalad7/prisma@v0.1.0/sdk/js/dist/prisma.umd.js"></script>
<script>
  const prisma = new Prisma({ url: "https://192.168.1.10:8443", llave: "pk_..." });
  document.getElementById("imprimir").onclick = () => prisma
    .imprimir("caja", (d) => d.texto("¡Hola!", { negrita: true }).qr("https://example.com").cortar())
    .then((r) => alert(`Impreso: ${r.trabajo_id}`))
    .catch((e) => alert(`${e.mensaje}\n${e.sugerencia}`));
</script>
```

Cambia `192.168.1.10` por la IP del puente y `"caja"` por el alias de tu impresora: es el que aparece en el panel, en *Impresoras* (también lo lista `await prisma.impresoras()` una vez que tengas la llave). Por ahora deja `pk_...` tal cual: la llave real llega en el paso 3.

> La dirección fija la versión del SDK (`@v0.1.0`), así que tu página no cambia sin aviso. Para actualizar, cambia el número por el de una versión más nueva. Todas las formas de instalarlo están en [sdk.md](sdk.md#instalación).

## 2. Sírvela

```sh
npx serve -l 3000
```

Abre `http://localhost:3000` y pulsa **Imprimir**. La primera vez verás:

> El puente responde, pero no ha autorizado el origen http://localhost:3000.
> El dueño del puente debe aprobar este origen en el panel (Solicitudes): ya quedó registrado como pendiente. Si la aplicación ya estaba aprobada, puede estar desactivada.

Es lo esperado: ninguna página puede imprimir sin que el dueño del puente la autorice.

Si en cambio ves **«No se pudo conectar con el puente en https://…»**, el navegador todavía no confía en el certificado del puente (punto 2 de [Lo que necesitas](#lo-que-necesitas)) o el puente no está encendido o en la misma red.

## 3. Pide la autorización

> Si eres tú quien abre el panel por primera vez desde otro equipo, te pedirá un código de configuración que se obtiene en el equipo del puente. Ver [panel.md, Primer acceso](../administracion/README.md#primer-acceso).

En el panel del puente (`https://<ip-del-puente>:8443/admin`), el dueño:

1. Va a **Solicitudes**, donde aparece `http://localhost:3000`.
2. Pulsa **Aprobar → Crear una aplicación nueva**, con nivel **2 — Llave**.
3. Copia la llave `pk_…` que aparece **una sola vez** y te la entrega.

Pega la llave en lugar de `pk_...`, recarga la página y pulsa **Imprimir**. El ticket sale y el aviso muestra el id del trabajo.

## 4. Arma tu ticket

El documento se arma encadenando operaciones; cada una lleva su propio estilo:

```js
await prisma.imprimir("caja", (doc) => doc
  .texto("MI NEGOCIO", { alineacion: "centro", negrita: true, tamano: [2, 2] })
  .separador()
  .par("2 x Taco", "$60.00")
  .par("TOTAL", "$60.00", { negrita: true })
  .qr("https://minegocio.com/p/812")
  .cortar());
```

- El SDK valida cada operación antes de enviarla. Un error, por ejemplo `alineacion: "center"`, se lanza en la línea que lo causó, con el campo exacto en `e.detalles`.
- Cada ticket lleva un id nuevo. Si falla la red, el SDK reintenta con el **mismo** id, y el puente no imprime dos veces.
- Para ver cómo saldrá sin gastar papel: `const html = await prisma.previsualizar("caja", (doc) => …)`.

Todas las operaciones (filas, códigos de barras, imágenes, cajón…) están en [integradores/sdk.md](sdk.md).

## Siguientes pasos

- **Antes de producción**, elige el nivel de seguridad adecuado. Una llave dentro de una página la puede leer quien abra el código fuente, así que el nivel 2 protege contra equipos de la red, no contra quien ya tiene la página. Ver [niveles-de-seguridad.md](niveles-de-seguridad.md).
- **Maneja los errores por `codigo`**, que es estable; `mensaje` y `sugerencia` están pensados para mostrarlos a la persona. Ver [problemas-comunes.md](problemas-comunes.md).
- **Ejemplo completo:** [ejemplos/pagina-simple.html](ejemplos/pagina-simple.html).
