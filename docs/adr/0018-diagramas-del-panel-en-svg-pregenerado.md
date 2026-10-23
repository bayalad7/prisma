# 0018. Diagramas del panel en SVG pregenerado, servidos como imágenes

- **Estado:** aceptada, en desarrollo para PRISMA 0.2.0
- **Fecha:** 2026-10-22

## Contexto

Los diagramas de `docs/diagramas/` son bloques Mermaid: GitHub los dibuja, pero el panel debe mostrarlos sin Internet, con la CSP del panel (sin `unsafe-inline` ni `unsafe-eval`), en claro y oscuro, legibles en celular y siempre al día con `docs/`. Y `go build` no debe necesitar Node.

Se evaluaron tres opciones, con medidas de octubre de 2026 (mermaid-cli 11.4.2, que usa Mermaid 11.17.2):

| Opción | Tamaño en el binario | CSP | Sin Node al compilar | Fidelidad |
|---|---|---|---|---|
| **a) SVG pregenerado** con `mmdc`, claro y oscuro | 27 diagramas × 2 variantes: 2,6 MB en SVG; **580 KB con gzip** | Se muestran con `<img>`: no hace falta ninguna excepción (ver abajo) | Sí: los SVG se versionan | La de Mermaid en Chromium |
| b) Mermaid en el navegador | `mermaid.min.js`: 3,5 MB (975 KB con gzip), más su carga diferida | Mermaid crea elementos `<style>` en la página: exige `style-src 'unsafe-inline'` | Sí | Depende del navegador del usuario |
| c) Híbrido | La suma de a) y b) | Como b) en modo desarrollo | Sí | — |

Pruebas hechas:

- Con `htmlLabels` desactivado, Mermaid no genera `foreignObject` ni scripts. El `<style>` interno pesa de 4 a 9 KB por SVG.
- Un SVG mostrado con `<img>` es un documento aislado, sin scripts. Se probó el mismo SVG con su `<style>` servido bajo la CSP del panel (`style-src 'self'`) y bajo una CSP mínima propia (`style-src 'unsafe-inline'`), en **Chromium, Firefox 142 y WebKit 26** (Playwright): en los tres motores se dibuja igual en los dos casos. La CSP de la respuesta no afecta a los estilos de una imagen.
- El diagrama más pesado (modelo de datos) ocupa 190 KB en SVG y 28 KB con gzip.

## Decisión

- **Opción a).** `scripts/docs/generar-diagramas.{sh,ps1}` (con `generar-diagramas.mjs`, Node 18+) dibuja cada bloque `mermaid` en claro y en oscuro con el tema de los tokens del panel. En la variante oscura, los cuatro colores comunes (`externo`, `prisma`, `datos`, `seguridad`) y las cajas de las secuencias pasan a su par oscuro. El fondo es transparente: lo pone el visor.
- Cada SVG se **sanea** (sin `<script>`, manejadores `on…`, `foreignObject` ni enlaces o recursos externos) y se guarda **con gzip** en `internal/admin/ayuda/diagramas/`, junto a `manifiesto.json`: id, metadatos del front-matter, texto alternativo, SHA-256 del bloque y archivos.
- **Se conserva el `<style>` interno y no hay CSP especial:** el panel los muestra con `<img>`, donde la CSP no afecta, así que se sirven con la misma CSP del panel. Se descarta convertir los estilos en atributos, que exigía un motor de CSS en la generación y no aportaba nada en los tres motores probados. Esto simplifica la decisión 4 de [docs/ux/ANALISIS-PANEL-V1.md](../ux/ANALISIS-PANEL-V1.md#decisiones-tomadas).
- El panel los sirve en `GET /admin/diagramas/{id}.{claro|oscuro}.svg`, solo con sesión, comprimidos (o sin comprimir si el navegador no acepta gzip); `GET /admin/api/diagramas` da el manifiesto. El visor (`visorDiagrama`, en `sistema.js`) cambia de variante con el tema, acerca, aleja y mueve (botones, teclado, rueda con Ctrl, pellizcar y arrastrar), tiene «Ver en grande» sin la API de pantalla completa, descarga el SVG y copia el código Mermaid; si la imagen no carga, muestra el código.
- En el manual, el bloque de cada página de diagrama se ve dibujado, y `{{diagrama:id}}` solo en su renglón pone el visor de ese diagrama en cualquier página (en GitHub se lee como texto; `TestInclusionDeDiagramas` exige que el diagrama exista).
- Cada `docs/diagramas/NN-*.md` lleva un **front-matter** obligatorio: `titulo`, `descripcion`, `audiencia` (`usuario`, `instalador`, `integrador`, `desarrollador`), `pantallas` (claves de `ayuda.pantallas` del panel) y `secciones_manual` (documentos del manual, sin `.md`).

## Consecuencias

- `go build` no necesita Node; solo quien cambia un diagrama corre el generador (minuto y medio para los 27).
- El binario crece unos 580 KB.
- `TestDiagramasVigentes` falla si un bloque o un front-matter cambió y no se volvió a generar, con el mensaje «Diagramas desactualizados: ejecuta scripts/docs/generar-diagramas»; el CI lo corre con `go test ./...`. `TestDiagramasSaneados` revisa cada SVG. `go test ./docs` valida el front-matter.
- Los SVG vienen de Chromium (el de `mmdc`): las letras pueden verse un poco distintas en otros sistemas.
- Abrir un SVG directamente en el navegador (no como imagen) lo mostraría sin estilos por la CSP del panel; el panel siempre los muestra como imagen y ofrece descargarlos.

## Evidencia en el código

- scripts/docs/generar-diagramas.mjs, scripts/docs/generar-diagramas.sh, scripts/docs/generar-diagramas.ps1
- internal/admin/ayuda/diagramas/manifiesto.json
- internal/admin/admin.go (`archivosDiagramas`)
- internal/admin/diagramas_test.go (`TestDiagramasVigentes`, `TestDiagramasSaneados`)
- docs/docs_test.go (`revisarFrontMatter`)
