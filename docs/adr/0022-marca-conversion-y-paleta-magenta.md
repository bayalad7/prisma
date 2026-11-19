# 0022. Logo «Conversión», paleta Magenta de proceso y reparto 60-30-10

- **Estado:** aceptada
- **Fecha:** 2026-11-19
- **Complementa a:** [0015](0015-sistema-de-diseno-del-panel.md)

## Contexto

El logo del panel era un prisma que separaba la luz en tres franjas pintadas con los colores de
acento, correcto y advertencia: mezclaba la marca con los colores de estado. A 16 px se leía como una
señal de advertencia. Los instaladores no traían ningún icono, así que PRISMA no se reconocía al
buscarlo en el menú Inicio, en el menú de aplicaciones ni en Spotlight. El panel tampoco tenía favicon.

La paleta (turquesa con grises azul pizarra) era casi la de Aula, otra aplicación del mismo autor, y
las dos marcas se veían como una sola. PRISMA debe ser una marca independiente. El panel tampoco tenía
un color de estructura: la barra lateral era blanca como el contenido y la jerarquía dependía solo de
los bordes.

Se compararon diez logos y cuatro paletas, cada logo con su variante a 16 px probada en la bandeja de
Windows y macOS.

## Decisión

- **Logo «Conversión».** Una sola hoja: arriba, el documento de la app con la esquina doblada; en medio,
  el puente en acento que la atraviesa (PRISMA); abajo, el ticket dentado. Cuenta lo que hace el
  producto: recibe un documento JSON y entrega un ticket. Tiene una variante simplificada para 20 px o
  menos y un logotipo «prisma» dibujado en minúsculas.
- **Una sola fuente para la marca.** [`internal/marca`](../../internal/marca/marca.go) guarda la geometría
  y los colores, y genera desde ellos los SVG, PNG, `.ico` y `.icns` de
  [`empaquetado/marca/`](../../empaquetado/marca/) y el favicon del panel. Usa un rasterizador propio en Go
  puro: no agrega dependencias ni herramientas de dibujo al CI.
- **Paleta «Magenta de proceso».** Es la M de CMYK, una tinta propia del mundo de la impresión. Los
  neutros son puros, sin tinte, para que el magenta sea lo único con color y nada recuerde a Aula.
- **Reparto 60-30-10.** Neutros para el contenido, el grafito secundario para la estructura (la barra
  lateral y la placa del icono) y el magenta solo para la acción. La barra no tiene reglas propias:
  redefine los tokens base con `--secundario-*`.
- **Los instaladores usan el logo** donde se busca el programa: el acceso «PRISMA» del menú Inicio (Inno
  Setup), la entrada `prisma.desktop` con iconos `hicolor` del `.deb` y `PRISMA.app` en `/Applications`
  del `.pkg`. Los tres abren el panel en `https://localhost:8443/admin`.

## Consecuencias

- El panel cambia de aspecto (paleta y barra oscura) sin tocar su lógica: solo cambian los tokens, el
  logo y el favicon.
- Las otras paletas candidatas (naranja térmico, amarillo) chocaban con el color de «aviso». Con el
  magenta, los estados no cambian; solo el error (rojo) queda cerca en tono y ya lleva siempre icono y
  texto.
- El acento ocupa cerca del 19 % del isotipo, más que el 10 % habitual, porque el puente es la idea
  del logo.
- Cambiar la marca exige regenerar los archivos (`go test ./internal/marca -actualizar`). Las pruebas
  fallan si el panel, el favicon o los instaladores quedan con otra geometría o con otros colores.
- `PRISMA.app` es un acceso, no el programa: desinstalar en macOS también la borra a mano
  ([docs/instalacion/macos.md](../instalacion/macos.md)).
- Los diagramas del manual conservan por ahora sus colores (`classDef`).

## Evidencia en el código

`internal/marca/` (geometría, rasterizador, `.ico`, `.icns` y sus pruebas), `empaquetado/marca/`,
`internal/admin/web/panel.css` (tokens y `.barra`), `internal/admin/web/sistema.js` (`logo`),
`internal/admin/web/favicon.svg`, `internal/admin/web/favicon.ico`, `internal/admin/estilos_test.go`,
`empaquetado/windows/prisma.iss`, `empaquetado/linux/nfpm.yaml`, `empaquetado/linux/prisma.desktop`,
`empaquetado/macos/construir-pkg.sh`, `empaquetado/macos/PRISMA.app/` y
[docs/ux/DESIGN_SYSTEM.md](../ux/DESIGN_SYSTEM.md).
