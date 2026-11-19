# Sistema de diseño de PRISMA

La marca y el sistema visual del panel de administración, de los instaladores y de cualquier pieza de
PRISMA (presentaciones, página del proyecto, SDK). Las decisiones están en el
[ADR 0015](../adr/0015-sistema-de-diseno-del-panel.md) (sistema propio del panel) y el
[ADR 0022](../adr/0022-marca-conversion-y-paleta-magenta.md) (logo y paleta).

**Fuente de verdad.** Si este documento y el código no coinciden, manda el código:

| Qué | Dónde |
|---|---|
| Geometría del logo y colores de la marca | [`internal/marca`](../../internal/marca/marca.go) |
| Archivos del logo (SVG, PNG, `.ico`, `.icns`) | [`empaquetado/marca/`](../../empaquetado/marca/), generados con `go test ./internal/marca -actualizar` |
| Tokens (color, tipografía, espacio, radios, sombras) | [`internal/admin/web/panel.css`](../../internal/admin/web/panel.css), sección 1 |
| Componentes | [`internal/admin/web/sistema.js`](../../internal/admin/web/sistema.js) (`window.PrismaUI`) |
| Galería viva | `/admin/estilos`, solo con `modo_desarrollo` |

## 1. Marca

### Logo «Conversión»

Una sola hoja que cambia al cruzar: arriba, el **documento** que manda la app (con la esquina
doblada); en medio, el **puente** en acento que la atraviesa de lado a lado, que es PRISMA; abajo, el
**ticket** dentado que sale por la impresora. Es exactamente lo que hace el producto: recibe un
documento JSON de una aplicación, en cualquier equipo, y lo entrega impreso.

| Pieza | Medidas en la rejilla de 32 | Color |
|---|---|---|
| Documento | 20 × 10, esquina doblada de 7, radio 2.5 | Tinta |
| Puente | 27 × 4, radio 2, de x 2.5 a 29.5 | Acento |
| Ticket | 20 × 9, borde dentado de tres muescas de 2.5 | Tinta |

El acento ocupa cerca del 19 % del isotipo: más que el 10 % habitual, porque el puente es la idea
central del logo.

### Versiones

| Versión | Archivo | Cuándo |
|---|---|---|
| Isotipo | `prisma.svg`, `prisma-oscuro.svg` | 24 px o más |
| Variante de bandeja | `prisma-bandeja.svg`, `prisma-bandeja-oscuro.svg` | 20 px o menos: puente más grueso, esquina más marcada y dos muescas |
| Horizontal | `prisma-horizontal.svg`, `prisma-horizontal-oscuro.svg` | Encabezados y documentos. «prisma» va **dibujado** (trazo 4.4, curvas de radio 6, altura x de 12.8 a 29.2), no escrito: se ve igual sin fuentes |
| Una tinta | `prisma-una-tinta.svg` | `currentColor`: impresión, franjas sólidas, iconos de plantilla de macOS |
| Icono de app | `prisma-icono.svg`, `prisma.ico`, `prisma.icns`, `png/` | Favicon, menú Inicio, menú de aplicaciones de Linux, Spotlight y Launchpad. El isotipo claro sobre la placa grafito (radio 7/32, margen 3); de 24 px para abajo usa la variante de bandeja |

En el panel, `PrismaUI.logo(tam)` dibuja la misma geometría con la tinta en `currentColor` y el
puente en `--marca-acento`; de 20 px para abajo cambia a la variante de bandeja.

### Colores de la marca

| | Fondo claro | Fondo oscuro | Sobre la placa |
|---|---|---|---|
| Tinta | `#171717` | `#ededed` | `#f2f2f2` |
| Acento | `#be185d` | `#f472b6` | `#f472b6` |
| Placa | | | `#1a1a1a` |

### Reglas

- Área de respeto: 2 unidades de la rejilla (1/16 del lado) libres alrededor.
- Mínimos: isotipo de 16 px (con la variante de bandeja) y horizontal de 24 px de alto.
- El nombre va en minúsculas en el logotipo («prisma») y en mayúsculas en el texto corrido (PRISMA).
- No hacer: rotarlo, cambiar el acento por un color de estado, agregarle sombras, degradados o
  contornos, separar sus tres piezas ni usarlo como icono de acción.

## 2. Color

### Reparto 60-30-10

| Papel | Tokens | Dónde |
|---|---|---|
| **60 % neutros:** el contenido | `--fondo`, `--superficie`, `--superficie-2`, `--superficie-3` | Fondo, tarjetas, tablas, diálogos |
| **30 % secundario:** la estructura | `--secundario` y su familia | La barra lateral (grafito) y la placa del icono de app |
| **10 % acento:** la acción | `--acento` y su familia | La acción primaria de cada zona, la selección, los enlaces, el foco y el puente del logo |

Los colores de estado no cuentan en el reparto. El magenta funciona por escasez: si decora, deja de
señalar.

La barra lateral no tiene reglas propias de color. Dentro de `.barra`, los tokens base toman los
valores `--secundario-*` (`--superficie` pasa a ser `--secundario`, `--tinta` pasa a ser
`--secundario-texto`, etc.), así que sus enlaces, su contador, el selector de tema y el botón de
menú se dibujan con las mismas reglas que en el resto del panel.

### Paleta semántica

Neutros puros, sin tinte, para que el magenta sea lo único con color.

| Token | Claro | Oscuro | Uso |
|---|---|---|---|
| `--fondo` | `#f5f5f5` | `#0f0f0f` | Fondo de la página |
| `--superficie` | `#ffffff` | `#171717` | Tarjetas, paneles, diálogos |
| `--superficie-2` | `#efefef` | `#212121` | Bloques internos, código, hover |
| `--superficie-3` | `#e3e3e3` | `#2c2c2c` | Tercer nivel (pistas, rellenos) |
| `--tinta` | `#171717` | `#ededed` | Texto principal; tinta del logo |
| `--tenue` | `#575757` | `#a3a3a3` | Ayudas y metadatos |
| `--borde` | `#dcdcdc` | `#2e2e2e` | Separadores |
| `--borde-control` | `#808080` | `#737373` | Bordes de controles (3:1) |
| `--acento` | `#be185d` | `#f472b6` | Botón primario, enlaces, foco |
| `--acento-fuerte` | `#9d174d` | `#f9a8d4` | Primario al pasar el puntero, texto sobre acento suave |
| `--acento-suave` | `#fce7f1` | `#3b1228` | Elegido, página actual |
| `--sobre-acento` | `#ffffff` | `#1a0510` | Texto sobre el acento |
| `--marca-acento` | `#be185d` | `#f472b6` | Puente del logo |

### Secundario

| Token | Claro | Oscuro | Uso |
|---|---|---|---|
| `--secundario` | `#1a1a1a` | `#0a0a0a` | Fondo de la barra lateral; placa del icono |
| `--secundario-2` | `#2a2a2a` | `#1c1c1c` | Hover dentro de la barra |
| `--secundario-borde` | `#333333` | `#262626` | Separadores de la barra |
| `--secundario-control` | `#8a8a8a` | `#7a7a7a` | Bordes de controles en la barra (3:1) |
| `--secundario-texto` | `#f2f2f2` | `#ededed` | Texto e iconos de la barra |
| `--secundario-suave` | `#a8a8a8` | `#9e9e9e` | Títulos de grupo y pie de la barra |
| `--secundario-elegido` | `#4a1830` | `#3b1228` | La página abierta |
| `--secundario-texto-elegido` | `#ffffff` | `#ffffff` | Texto de la página abierta |
| `--secundario-acento` | `#f472b6` | `#f472b6` | Puente del logo y foco dentro de la barra |
| `--secundario-sobre-acento` | `#1a1a1a` | `#1a0510` | Texto sobre el acento en la barra |
| `--secundario-aviso` | `#e6b04c` | `#e6b04c` | Contador de solicitudes pendientes |

### Estados

Siempre con icono y texto, nunca solo con color.

| Estado | Claro (texto / fondo) | Oscuro (texto / fondo) |
|---|---|---|
| `--ok` | `#146c43` / `#e1f3e8` | `#5cc993` / `#12301f` |
| `--aviso` | `#8a5700` / `#fcefd4` | `#e6b04c` / `#34280e` |
| `--error` | `#b42318` / `#fdebe9` | `#f48b82` / `#3d1714` |
| `--info` | `#1f5aa6` / `#e4edfa` | `#7fb0f0` / `#142640` |
| `--desconocido` | `#5c5c5c` / `#ededed` | `#a3a3a3` / `#262626` |
| `--virtual` | `#6a3db0` / `#efe8fb` | `#bb9af2` / `#2a1f40` |

El error (rojo) y el acento (magenta) se separan por tono. Con daltonismo rojo-verde se acercan; por
eso el error lleva siempre su icono y su texto.

Además: `--papel` y `--papel-tinta` para la vista previa de los tickets, `--velo` detrás de los
diálogos y `--sombra-1` / `--sombra-2`.

## 3. Tipografía

El panel no carga fuentes: el puente funciona sin Internet y la CSP no admite recursos externos.

| Token | Pila | Uso |
|---|---|---|
| `--letra` | `ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, …` | Todo el texto |
| `--letra-titulo` | `"Segoe UI Variable Display", "SF Pro Display", ui-sans-serif, system-ui, …` | El nombre «prisma» de la barra (800, −2 %, minúsculas) |
| `--letra-mono` | `ui-monospace, "Cascadia Mono", "SF Mono", Menlo, Consolas, …` | Código, llaves, alias, chips |

Escala (raíz de 15 px):

| Token | rem | px | Uso |
|---|---|---|---|
| `--fs-xs` | .78 | 11.7 | Títulos de grupo, insignias de la barra inferior |
| `--fs-sm` | .875 | 13.1 | Ayudas, metadatos |
| `--fs-base` | 1 | 15 | Texto |
| `--fs-lg` | 1.125 | 16.9 | `h2` |
| `--fs-xl` | 1.375 | 20.6 | `h1` en celular |
| `--fs-2xl` | 1.6 | 24 | `h1` |

Pesos: 400 (texto), 500 y 550 (enlaces y botones), 600 (insignias, etiquetas), 650 (títulos), 700 y
800 (contadores y el nombre de la marca). Para presentaciones y la página del proyecto, los títulos
van en Plus Jakarta Sans 700/800 y el texto en Inter, con la misma escala relativa.

## 4. Espacio, forma y profundidad

| Token | Valor | |
|---|---|---|
| `--e-1` a `--e-6` y `--e-8` | 4, 8, 12, 16, 20, 24 y 32 px | Rejilla de 4 px |
| `--radio-chico` | 6 px | Botones, campos, enlaces de la barra |
| `--radio` | 8 px | Tarjetas, notas |
| `--radio-grande` | 12 px | Diálogos |
| `--ancho-barra` | 240 px | Barra lateral |
| `--alto-nav-inferior` | 64 px | Barra inferior en celular |
| `--objetivo` | 44 px | Tamaño mínimo de lo que se toca en celular |

La jerarquía se marca con superficies y bordes. `--sombra-1` es para lo que está en reposo y
`--sombra-2`, solo para lo que flota (diálogos, avisos).

## 5. Utilidades

Una lista cerrada de clases que solo acomodan: dirección, hueco y rejilla. Nunca llevan color,
sombra, radio, tamaño de letra ni movimiento; eso es de los componentes.
`TestUtilidadesSoloAcomodan` lo revisa.

| Clase | Qué hace |
|---|---|
| `.fila` | Flex que envuelve, centrado, hueco de 8 × 12 |
| `.fila-entre` | Igual, con los extremos separados |
| `.pila` | Columna con hueco de 6 |
| `.pila-amplia` | Columna con hueco de 16 |
| `.lista` | Columna con hueco de 12 |
| `.rejilla` | Rejilla que se llena sola, celdas de 280 px como mínimo |
| `.rejilla.de-cuatro` | Igual, con celdas de 230 px |

De texto, en la base: `.tenue`, `.pequeno`, `.mono`, `.numeros` (cifras tabulares) y `.solo-lector`.

Reglas:

- Una utilidad nueva se agrega aquí y en `panel.css`, como un token.
- Sin valores arbitrarios ni variantes por tamaño de pantalla. Lo adaptable se resuelve en el
  componente.
- Las clases semánticas no se renombran: la utilidad se suma al lado.

## 6. Componentes

`window.PrismaUI` (en `sistema.js`) crea cada pieza como nodos DOM: el texto se inserta siempre como
texto y los estilos dinámicos usan CSSOM, que la CSP permite.

- **Acciones:** `boton` (primario, secundario, peligro, peligro lleno, texto, chico, solo icono), `conEspera`, `copiar`, `botonCopiar`.
- **Formularios:** `entrada`, `selector`, `campo`, `interruptor`, `casilla`, `revisarCampos`.
- **Contenido:** `tarjeta`, `insignia`, `nota`, `vacio`, `esqueleto`, `pasos`, `pestanas`, `codigo`, `tabla`, `ticketPapel`, `visorDiagrama`.
- **Avisos y diálogos:** `aviso`, `avisoError`, `dialogo`, `confirmar`, `configurarAyuda`.
- **Estructura:** `marco` (barra lateral, menú y barra inferior), `selectorTema`, `icono`, `logo`.

## 7. Reglas

- Color por función, nunca por tono; un estado siempre con icono y texto.
- Una sola acción primaria (magenta) por zona; el magenta no decora.
- Contraste WCAG AA: 4.5:1 para texto, 3:1 para bordes de controles, foco e iconos.
- Solo tokens: ningún color suelto en las reglas.
- Tres bloques de color: claro, oscuro automático y oscuro elegido a mano; los dos oscuros son idénticos.
- La barra lateral cambia los tokens, no las reglas.
- Sin frameworks, CDNs ni fuentes externas (ADR 0013).

## 8. Pruebas que lo vigilan

| Prueba | Qué revisa |
|---|---|
| `TestContrasteDeLosTokens` | AA de cada par de tokens en claro y oscuro |
| `TestContrasteDelSecundario` | AA de la barra lateral y del puente del logo |
| `TestTemaOscuroIgual` | Los dos bloques oscuros son iguales |
| `TestColoresDelPanelCoincidenConLaMarca` | La tinta, el acento y el secundario del panel son los de `internal/marca` |
| `TestLogoDelPanelCoincideConLaMarca` | `sistema.js` dibuja la geometría oficial |
| `TestUtilidadesSoloAcomodan` | Las utilidades no llevan color ni forma |
| `TestArchivosDeLaMarca` | Los SVG, PNG, `.ico` y `.icns` salen de `internal/marca` |
| `TestLosInstaladoresUsanLaMarca` | Windows, `.deb` y `.pkg` instalan el logo donde se busca el programa |
| `e2e/` (Playwright con axe) | Contraste y accesibilidad en el navegador, en claro y oscuro |

## 9. Cómo cambiar algo

- **Un color:** cambia el token en los tres bloques de `panel.css` y corre `go test ./internal/admin`.
  Si es de la marca, cámbialo también en `internal/marca` y regenera los archivos.
- **El logo:** cambia la geometría en `internal/marca/marca.go` y en `PrismaUI.logo` (`sistema.js`) y
  corre `go test ./internal/marca -actualizar`.
- **Una utilidad o un componente:** agrégalo aquí, en `panel.css` o en `sistema.js` y en la galería
  `/admin/estilos`.
