# Experiencia del panel

[DESIGN_SYSTEM.md](DESIGN_SYSTEM.md) es el sistema de diseño vigente: la marca (logo «Conversión»), la paleta Magenta de proceso con su reparto 60-30-10, la tipografía, los tokens, las utilidades y los componentes.

El trabajo de experiencia del panel de la versión 0.2.0:

- [ANALISIS-PANEL-V1.md](ANALISIS-PANEL-V1.md): hallazgos sobre el panel de 0.1.0, flujos, accesibilidad, plan acordado y decisiones tomadas.
- [RESULTADOS.md](RESULTADOS.md): el antes y el después (clics por flujo, hallazgos resueltos, accesibilidad, tamaño) y lo que queda.
- [PROPUESTAS-V2.md](PROPUESTAS-V2.md): lo que quedaba fuera del plan y cómo se cerró.

Las capturas que cita el análisis enlazan al commit de la versión 0.3.4 (`f4d29ac`).

## Lo que se quitó

Las pruebas de cada entrega se quitaron del árbol después de publicar 0.3.4: capturas, scripts de un solo uso e informes JSON. Todas las comprobaciones de esos informes habían pasado, y lo que verificaban lo cubren hoy las pruebas permanentes (`e2e/`, `go test ./...`). Se quitaron:

- las carpetas de entrega `fase2/` a `fase64/`, `fase7e4/`, `propuestas-e1/`, `propuestas-e3/`, `v030-e2/`, `v031/`, `v033/` y `v034/`;
- `despues/`: las capturas de 0.2.0. Se regeneran con `CAPTURAS=<carpeta>` en `e2e/pantallas.spec.mjs`;
- `antes/`: las capturas de 0.1.0, el informe de axe (`axe-antes.json`) y el método con el que se tomaron (`sembrar.sh`, `recorrido.mjs`, `a-webp.mjs`). Para repetirlo, recupera la carpeta y ejecútalo contra la etiqueta `v0.1.0`.

`fase5e1/csp-svg.mjs` reproduce el experimento del [ADR 0018](../adr/0018-diagramas-del-panel-en-svg-pregenerado.md): el `<style>` de un SVG mostrado con `<img>` bajo la CSP del panel, en Chromium, Firefox y WebKit.

Todo sigue en el historial, en el commit de la versión 0.3.4 (`f4d29ac`); lo de 0.2.0 también está en el de esa versión (`2d79ff1`):

```sh
git restore --source=f4d29ac -- docs/ux/despues     # una carpeta
git show f4d29ac:docs/ux/fase5e1/csp-svg.mjs        # un archivo
```

O en GitHub: <https://github.com/bayalad7/prisma/tree/f4d29ac1d85b79906182bfa12a53bce07f5a766d/docs/ux>.
