# 0017. Manual del panel desde docs/, con un renderizador propio en el navegador

- **Estado:** aceptada, en desarrollo para PRISMA 0.2.0
- **Fecha:** 2026-10-22

## Contexto

La Ayuda del panel 0.1.0 enlazaba a la documentación en GitHub: sin Internet no había ayuda, y la que se leía podía no corresponder a la versión instalada. La documentación ya existe en `docs/` y debe seguir siendo la única fuente.

Mostrar Markdown en el panel exige convertirlo. Las opciones evaluadas en [docs/ux/ANALISIS-PANEL-V1.md](../ux/ANALISIS-PANEL-V1.md#manual-dentro-del-panel) fueron generar HTML en el puente con goldmark (una dependencia nueva y HTML que sanear) o dibujarlo en el navegador. El dueño del proyecto eligió la segunda, aunque el plan de la Fase 4 pedía HTML generado en el servidor.

## Decisión

- El paquete `docs` embebe con `go:embed` los Markdown que forman el manual, sus capturas y `docs/manual.json`, que define los grupos y el orden. No hay copias ni pasos de generación: `go build` no necesita Node y lo que está en `docs/` es lo que se ve.
- El panel sirve esos archivos tal cual en `GET /admin/manual/{ruta}`, solo con sesión y solo los que están en el índice (y sus imágenes `*/img/*/*.webp`). `GET /admin/api/manual` da el índice con el título de cada documento.
- `internal/admin/web/manual.js` convierte el Markdown en nodos DOM con `createElement` y `textContent`, nunca con `innerHTML`: el contenido no puede inyectar HTML ni scripts, y no hay nada que sanear. Admite el subconjunto que usa `docs/`: encabezados, párrafos, listas anidadas, tablas, bloques de código, citas, líneas, enlaces, imágenes, negritas, cursivas y código en línea, y omite el front-matter.
- La búsqueda (`GET /admin/api/manual/buscar`) usa un índice por sección que el puente arma una vez desde lo embebido, sin servicios externos: no distingue mayúsculas, acentos ni la ñ, busca por prefijo e incluye la descripción del front-matter de los diagramas.
- El panel decide adónde lleva cada enlace: a otra página del manual (`/admin/ayuda/{ruta}#ancla`, sin recargar), a un ancla de la misma página, o a GitHub en la versión instalada para lo que no está en el manual (código, ADR), marcado como externo y «requiere Internet».

## Consecuencias

- La ayuda funciona sin Internet y corresponde siempre a la versión del puente.
- El binario crece con el manual: unos 300 KB de Markdown y 320 KB de capturas (WebP), más 10 KB de manual.js.
- El renderizador es código propio que mantener (unas 300 líneas). `TestSintaxisDelManual` falla si un documento del índice usa algo que no admite (HTML crudo, notas al pie, enlaces por referencia, cursivas con `_`, títulos subrayados, tablas sin `|` inicial), así que una página nunca se ve rota en silencio.
- `TestIndiceDelManual` exige que todo lo del índice exista y que las guías del panel y los diagramas estén en él.
- Los bloques `mermaid` se muestran como código plegado hasta que el panel dibuje los diagramas (Fase 5).

## Evidencia en el código

- docs/doc.go (`Manual`), docs/manual.json
- internal/admin/manual.go (`listarManual`, `servirManual`)
- internal/admin/web/manual.js (`renderizar`)
- internal/admin/web/panel.js (`vistaAyuda`)
- docs/manual_test.go (`TestIndiceDelManual`, `TestSintaxisDelManual`), internal/admin/manual_test.go
