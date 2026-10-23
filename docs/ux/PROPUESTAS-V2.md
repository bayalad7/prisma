# Propuestas para después de 0.2.0

Cambios al puente que salieron del trabajo de experiencia del panel pero quedan fuera de lo acordado para 0.2.0 (la única excepción de backend fue la impresora virtual y el área de pruebas). Ninguno está implementado. Al final, los hallazgos del [análisis](ANALISIS-PANEL-V1.md) que quedaron pendientes (ver [RESULTADOS.md](RESULTADOS.md#hallazgos)); esos son solo del panel.

## El panel comparte el límite de peticiones por IP con la API

- **Qué pasa:** `peticiones_ip_por_minuto` (120 por defecto) cuenta todas las peticiones de una IP al puente, también las del panel. Abrir el panel cuesta unas 11 (la página, sus archivos, la sesión, el resumen); navegar dentro de él, una o dos por pantalla. En uso normal no se nota, pero si el equipo del dueño es también el punto de venta que imprime por `/v1`, las dos cosas se restan del mismo límite: muchas impresiones seguidas pueden dejar el panel sin cargar (los archivos llegan como error `DEMASIADAS_PETICIONES`) y al revés.
- **Cómo se encontró:** el recorrido de la Fase 4 abría las 63 páginas del manual recargando el panel en cada una y superó el límite en menos de un minuto.
- **Propuesta:** no contar los archivos estáticos del panel (son públicos e iguales para todos) y dar al panel con sesión un límite propio, separado del de `/v1`. Requiere decidir los valores y una prueba en `internal/servidor`.

## Nodos de los diagramas que enlacen al manual

- **Qué falta:** el plan de la Fase 5 lo proponía como opcional: que ciertos nodos de un diagrama lleven a una sección del manual o a una pantalla del panel, con un mapa nodo → enlace en el front-matter.
- **Por qué no está:** el panel muestra los diagramas como imágenes (ADR 0018), y una imagen no recibe clics en sus nodos. Insertar el SVG en la página exigiría relajar la CSP del panel.
- **Propuesta:** que `generar-diagramas` guarde en el manifiesto la caja de cada nodo con enlace (posición y tamaño, en proporción) y que el visor superponga enlaces transparentes sobre la imagen, que sigan el zoom. Sin cambios en la CSP.

## Inicio que se actualiza solo

- **Hallazgo:** M2. El estado de las impresoras, la cola y el contador de solicitudes cambian en el puente, pero el panel no lo muestra hasta cambiar de pantalla.
- **Por qué no está:** cuando se revisó, consultar `/admin/api/resumen` cada pocos segundos renovaba la sesión y anulaba el cierre por inactividad; la cabecera que lo evita llegó después, con el rollo en vivo.
- **Propuesta:** consultar el resumen cada 15 s solo con la pestaña visible y con la cabecera `X-Prisma-Sondeo: 1`, que el rollo en vivo ya usa para no renovar la sesión. Sin cambios en el puente: la cabecera ya vale para cualquier ruta del panel.

## Nombre del puente repetido

- **Hallazgo:** M6. Con el nombre predeterminado, la barra muestra «PRISMA / PRISMA» y el título de Inicio dice «PRISMA».
- **Propuesta:** si el nombre es el predeterminado, mostrar «Puente de impresión» e invitar a ponerle nombre en *Ajustes → General*.

## Deshacer al rechazar un origen

- **Hallazgo:** B5. Rechazar una solicitud no confirma, y está bien porque es reversible desde la pestaña «Rechazadas», pero no ofrece deshacer en el momento.
- **Propuesta:** el aviso de «rechazado» con la acción «Deshacer» (`aviso(..., { accion })` ya lo permite) durante unos segundos.

## Cerrar el menú del celular al tocar fuera

- **Hallazgo:** B4, en parte. El menú se cierra con Escape y al elegir una sección, pero no al tocar fuera de él.
- **Propuesta:** cerrarlo con un `pointerdown` fuera de la barra mientras esté abierto, y devolver el foco al botón *Menú*.
