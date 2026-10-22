# Seguridad

PRISMA es **seguro por defecto**: ninguna página web ni equipo de la red imprime sin que el dueño lo autorice, y las operaciones que mueven dinero (`abrir_cajon`) o envían bytes arbitrarios (`raw`) están deshabilitadas hasta que se habilitan para cada aplicación.

| Documento | Para qué |
|---|---|
| [modelo-de-amenazas.md](modelo-de-amenazas.md) | Qué se protege, de quién, cómo lo resuelve el código y qué riesgo queda. |
| [politicas.md](politicas.md) | Qué guarda el puente y qué no, cómo trata llaves y contraseñas, permisos de archivos, cabeceras y la llave de las versiones. |
| [../../SECURITY.md](../../SECURITY.md) | Cómo reportar una vulnerabilidad. |

## Resumen del modelo

| Nivel | Cómo se identifica la aplicación | Protege contra | No protege contra |
|---|---|---|---|
| 1 — Origen | Encabezado `Origin` aprobado. | Páginas web maliciosas abiertas en un navegador de la red. | Herramientas fuera del navegador (curl puede inventar un `Origin`). |
| 2 — Llave (por defecto) | `Origin` aprobado y llave `pk_…`. | Además, equipos de la red sin la llave. | Quien tenga la llave, por ejemplo leyéndola del código de una página. |
| 3 — Firma | Documento firmado con Ed25519 por el backend; el puente solo guarda la llave pública. | Además, documentos alterados, repetidos o falsificados. | Quien controle el backend que firma. |

Cómo elegir el nivel de una aplicación: [integradores/niveles-de-seguridad.md](../integradores/niveles-de-seguridad.md). Cómo decide el puente cada petición: [diagramas/07-orden-autorizacion.md](../diagramas/07-orden-autorizacion.md).
