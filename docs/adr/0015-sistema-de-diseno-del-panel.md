# 0015. Sistema de diseño propio para el panel

- **Estado:** aceptada, en desarrollo para PRISMA 0.2.0
- **Fecha:** 2026-10-22

## Contexto

El análisis del panel de 0.1.0 ([docs/ux/ANALISIS-PANEL-V1.md](../ux/ANALISIS-PANEL-V1.md)) encontró estilos y piezas repetidas a mano en cada pantalla, desbordes en celular, foco invisible en algunas opciones, errores de formulario sin asociar al campo y estados que dependían solo del color. La versión 0.2.0 rediseña todas las pantallas y agrega otras (ayuda, diagramas, área de pruebas), así que hace falta una base común. [ADR 0013](0013-panel-sin-frameworks.md) sigue vigente: nada de frameworks ni CDNs, y la CSP no admite `unsafe-inline`.

## Decisión

El panel tiene un sistema de diseño propio, embebido en el binario:

- **Tokens** en `panel.css`: colores (incluidos los de estado: correcto, advertencia, error, información, desconocido y virtual), tipografía del sistema, espaciado, radios y sombras. Hay tres bloques de color: claro, oscuro automático (`prefers-color-scheme`) y oscuro elegido a mano (`data-theme`). La elección manual se guarda en `localStorage`, como comodidad de cada navegador.
- **Componentes** en `sistema.js` (`window.PrismaUI`), con funciones que crean nodos DOM: el texto se inserta siempre como texto y los estilos dinámicos usan CSSOM, que la CSP permite. Cada componente resuelve su accesibilidad (errores con `aria-invalid` y `aria-describedby`, pestañas con flechas, foco visible, regiones vivas).
- **Textos** en un solo archivo, `textos.js`, para poder traducir el panel más adelante.
- **Iconos** propios en un solo sprite, `iconos.svg`, usados con `<use>`.
- **Galería** en `/admin/estilos`, solo con `modo_desarrollo`, con cada componente y los prototipos de las pantallas.

## Consecuencias

- Las pantallas se arman con los mismos componentes, así que el aspecto y la accesibilidad son uniformes, y lo que se corrige en un componente se corrige en todas.
- Pruebas de Go vigilan lo que suele romperse sin que nadie lo note: el contraste AA de cada par de tokens en los dos temas, que los dos bloques del tema oscuro sean iguales y que el sprite solo tenga formas básicas y todos los iconos que usa el panel.
- Mantener componentes propios cuesta más que usar una biblioteca, pero son pocos y no agregan dependencias ni superficie de ataque.
- El tema oscuro se repite en dos bloques de CSS (sin preprocesador no se puede compartir); una prueba exige que sean idénticos.

## Evidencia en el código

`internal/admin/web/` (`panel.css`, `sistema.js`, `textos.js`, `iconos.svg`, `estilos.html`, `estilos.js`), `internal/admin/admin.go` (`paginasDesarrollo`, `servirEstatico`), `internal/admin/estilos_test.go`.
