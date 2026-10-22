# Aplicaciones y solicitudes

Una **aplicación** es un programa autorizado a imprimir; en la API y en la CLI se llama *cliente*. Cada una tiene un nivel de seguridad, orígenes, impresoras permitidas y operaciones sensibles.

Para elegir el nivel de una aplicación, ver [integradores/niveles-de-seguridad.md](../integradores/niveles-de-seguridad.md). El modelo completo está en [seguridad/](../seguridad/README.md).

## Niveles y llaves

El panel explica los niveles en lenguaje llano; entre paréntesis, el nombre técnico. *¿Cuál elijo?*, en el formulario, resume cuándo conviene cada uno.

| En el panel | Nivel | Cómo se identifica | Qué muestra el panel |
|---|---|---|---|
| Solo la dirección de la página | 1 — Origen | Solo el encabezado `Origin` del navegador. Necesita al menos una página web. | — |
| Dirección y llave secreta (recomendado) | 2 — Llave (por defecto) | Origen (si viene de un navegador) y la llave `pk_…`. | La llave **una sola vez**: copiarla, descargarla como archivo de texto, qué hacer con ella y el código para integrar. |
| Firma desde su servidor | 3 — Firma | Documentos firmados con Ed25519 por el backend de la aplicación. | Valida la llave pública (PEM o base64) y muestra su huella. Opcionalmente pide también la llave secreta. |

- **Cambiar la llave:** la anterior deja de funcionar en ese momento; el panel lo advierte antes.
- **Quitar la llave:** en el nivel 2 la aplicación no puede imprimir hasta que se le dé otra; en el nivel 3 sigue imprimiendo con documentos firmados.
- **Desactivar** y **Eliminar** piden confirmación y dicen qué deja de funcionar.
- **Permisos especiales** (`abrir_cajon`: abrir el cajón de dinero; `raw`: enviar comandos directos a la impresora) están apagados hasta que los marques para esa aplicación.
- **Código para integrar** muestra el ejemplo con el SDK (etiqueta `<script>`), con `fetch` y con `curl`, para la impresora que elijas entre las permitidas.
- Los errores del formulario (sin nombre, una página web mal escrita, el nivel 1 sin páginas, una página que ya pertenece a otra aplicación) aparecen junto al campo.

## Solicitudes de origen

Cuando una página web desconocida intenta imprimir, el puente la rechaza (`ORIGEN_NO_AUTORIZADO`) y la anota aquí con el número de intentos. Cada solicitud indica si viene de **este equipo**, de un **equipo de tu red** o de un **sitio de Internet**, y si usa `http` **sin cifrar**; al aprobar un sitio de Internet o una página sin cifrar, el panel advierte el riesgo. Al **aprobarla** eliges:

- **Aplicación nueva** («Dirección y llave secreta», recomendado, o «Solo la dirección»): puede imprimir en todas las impresoras y sin permisos especiales; ajústala después en Aplicaciones. Con llave, se muestra una vez.
- **Aplicación existente:** el origen se agrega a esa aplicación con sus permisos.

**Rechazar** la pasa a *Rechazadas* (plegada al final): sus intentos se siguen contando, pero ya no aparece como pendiente y se puede aprobar después. **Olvidar** la borra; si el origen vuelve a intentarlo, aparecerá otra vez.

La lista se limpia sola, para que ningún equipo de la red pueda llenarla con orígenes inventados:

- cada hora se borran las pendientes sin intentos en 30 días y las rechazadas sin intentos en 90;
- caben 200; al llegar al tope, un origen nuevo reemplaza a la pendiente con la actividad más antigua;
- con más de 150, la sección muestra un aviso. Muchos orígenes desconocidos suelen indicar que alguien en la red está probando el puente.
