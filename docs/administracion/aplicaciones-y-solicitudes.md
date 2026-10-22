# Aplicaciones y solicitudes

Una **aplicación** es un programa autorizado a imprimir; en la API y en la CLI se llama *cliente*. Cada una tiene un nivel de seguridad, orígenes, impresoras permitidas y operaciones sensibles.

Para elegir el nivel de una aplicación, ver [integradores/niveles-de-seguridad.md](../integradores/niveles-de-seguridad.md). El modelo completo está en [seguridad/](../seguridad/README.md).

## Niveles y llaves

| Nivel | Cómo se identifica | Qué muestra el panel |
|---|---|---|
| 1 — Origen | Solo el encabezado `Origin` del navegador. Necesita al menos un origen. | — |
| 2 — Llave (por defecto) | Origen (si viene de un navegador) y la llave `pk_…`. | La llave **una sola vez**, con botón para copiarla y el código de integración. |
| 3 — Firma | Documentos firmados con Ed25519 por el backend de la aplicación. | Valida la llave pública (PEM o base64) y muestra su huella. Opcionalmente exige también una llave `pk_`. |

- **Regenerar la llave:** la anterior deja de funcionar en ese momento.
- **Revocar la llave:** en el nivel 2 la aplicación no puede imprimir hasta que se genere otra; en el nivel 3 sigue imprimiendo con documentos firmados.
- `abrir_cajon` y `raw` están deshabilitadas hasta que las marques para esa aplicación.

## Solicitudes de origen

Cuando una página web desconocida intenta imprimir, el puente la rechaza (`ORIGEN_NO_AUTORIZADO`) y la anota aquí con el número de intentos. Al **aprobarla** eliges:

- **Aplicación nueva** (nivel 1 o 2): puede imprimir en todas las impresoras y sin operaciones sensibles; ajústala después en Aplicaciones. Con nivel 2 se muestra la llave una vez.
- **Aplicación existente:** el origen se agrega a esa aplicación con sus permisos.

**Rechazar** la oculta: sus intentos se siguen contando, pero ya no aparece como pendiente. **Olvidar** la borra; si el origen vuelve a intentarlo, aparecerá otra vez.

La lista se limpia sola, para que ningún equipo de la red pueda llenarla con orígenes inventados:

- cada hora se borran las pendientes sin intentos en 30 días y las rechazadas sin intentos en 90;
- caben 200; al llegar al tope, un origen nuevo reemplaza a la pendiente con la actividad más antigua;
- con más de 150, la sección muestra un aviso. Muchos orígenes desconocidos suelen indicar que alguien en la red está probando el puente.
