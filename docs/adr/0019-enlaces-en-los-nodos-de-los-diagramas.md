# 0019. Enlaces en los nodos de los diagramas, superpuestos a la imagen

- **Estado:** aceptada, en desarrollo para PRISMA 0.2.0
- **Fecha:** 2026-10-23

## Contexto

El panel muestra los diagramas como imágenes SVG ([ADR 0018](0018-diagramas-del-panel-en-svg-pregenerado.md)), así que sus nodos no reciben clics. En los diagramas para el dueño conviene que un nodo lleve a su pantalla («impresora de la caja» → *Impresoras*) o a la sección del manual que lo explica. Meter el SVG en la página permitiría enlaces dentro del dibujo, pero su `<style>` interno exigiría relajar la CSP del panel (`style-src 'unsafe-inline'`).

## Decisión

- El front-matter de un diagrama puede llevar `enlaces: [nodo=destino, …]`. `nodo` es el id del nodo en el bloque `mermaid`. `destino` es un documento del manual, sin `.md` y con `#ancla` opcional (`administracion/trabajos#qué-pasó-con-un-trabajo`), o una pantalla del panel (`/admin/impresoras`).
- `scripts/docs/generar-diagramas` calcula la caja de cada nodo en el SVG claro y la guarda en `manifiesto.json`, en proporción de la imagen (0 a 1), junto con el texto del nodo. Para calcularla suma los `translate` de los grupos que lo contienen y mide la primera figura del nodo (`rect`, `polygon` o `circle`). Solo hay enlaces en los diagramas de flujo (`flowchart`), y el generador falla si no encuentra un nodo.
- El visor pone sobre la imagen una capa con la misma caja y la misma transformación (zoom y desplazamiento), con un `<a>` transparente por nodo. Las posiciones se asignan con CSSOM, así que no hace falta ninguna excepción en la CSP.
  - Cada enlace se enfoca con Tab y tiene su nombre accesible: el texto del nodo y adónde lleva.
  - Se marca al pasar el puntero o al enfocarlo.
  - Tocar un enlace no inicia el arrastre.
  - Desde «Ver en grande», seguir un enlace vuelve primero al tamaño normal.
- La imagen sigue siendo la misma; un nodo sin enlace se ve y se comporta como antes.

## Consecuencias

- **Pruebas que lo cubren:**
  - `go test ./docs` revisa que cada nodo exista en el bloque y que cada destino exista: el documento en `docs/manual.json` y su ancla, o la sección en `nav.secciones` de `textos.js`.
  - `TestDiagramasVigentes` exige que el manifiesto tenga los mismos enlaces que el front-matter, con cajas dentro de la imagen.
  - Una prueba de punta a punta sigue dos enlaces del diagrama 30, en escritorio y en celular.
- Cambiar `enlaces` exige volver a generar los diagramas, como cualquier cambio del front-matter.
- Si una versión nueva de Mermaid cambia cómo anida los nodos, el generador puede dejar de encontrarlos y falla.
- Por ahora llevan enlaces los diagramas 29, 30 y 31.
