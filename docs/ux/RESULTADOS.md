# Resultados del rediseño del panel (0.2.0)

El antes y el después del trabajo que planteó el [análisis del panel](ANALISIS-PANEL-V1.md). «Antes» es la versión 0.1.0. «Después» es `develop` al cerrar la Fase 7. Las capturas están en [antes/](antes/) y [despues/](despues/), con los mismos números de pantalla cuando la pantalla existe en las dos.

## Resumen

- **De recién instalado a imprimir:** antes no se podía imprimir sin una impresora física, salvo con la terminal. Ahora:
  - el primer ticket sale en una impresora virtual con **2 clics** después de crear la contraseña, desde «Primeros pasos»;
  - una aplicación autorizada imprime con **10 clics y 4 campos** en total.
  - La prueba automática tarda 4 s. Para una persona, la estimación es de 2 a 3 minutos; la meta era menos de 5.
- **Hallazgos:**
  - Resueltos: los 5 críticos, los 9 altos, 10 de los 12 medios y 5 de los 7 bajos.
  - En parte: B4.
  - Pendientes: M2, M6 y B5 (ver [Lo que queda](#lo-que-queda)).
- **Accesibilidad:**
  - Antes: 1 regla de axe incumplida en 5 de 24 pantallas, revisada solo en tema claro.
  - Ahora: 0 violaciones críticas o serias en 34 pantallas, en claro y oscuro, en escritorio y celular.
  - Ninguna pantalla se desborda a lo ancho.
  - El panel se usa completo con el teclado.
- **Sin Internet:** el manual completo (74 documentos) y los 29 diagramas viven dentro del binario. La ayuda ya no abre GitHub.
- **Tamaño del binario:** **+2.6 MB** (+15 %), casi todo manual, capturas y diagramas.

## Flujos

Se cuentan los clics, toques y campos desde que la persona decide hacerlo, sin contar la lectura, igual que en la [comparación de los flujos](ANALISIS-PANEL-V1.md#comparación-de-los-flujos). Los flujos «después» son los que recorre la prueba de punta a punta ([e2e/flujos.spec.mjs](../../e2e/flujos.spec.mjs)) en escritorio y en celular.

| Flujo | Antes | Después | Qué cambió además del número |
|---|---|---|---|
| **a) Primer arranque → contraseña → primer acceso** | 3 clics + 2 campos | 3 clics + 2 campos | Dice qué es PRISMA y qué sigue: «Primeros pasos» con el avance. Se puede mostrar la contraseña. Los errores aparecen junto al campo. |
| **b) Primera impresora → prueba → acentos** | ~13 clics + 3 campos, más interpretar «ESC t» | 7 clics + 1 campo (detectada) · virtual: **1 clic** desde Inicio | El alias se propone solo. «Guardar y probar» imprime. «¿Salió?» y «¿Se leen bien los acentos?» sin hablar de páginas de códigos. Al terminar, el paso de acentos queda hecho. |
| **c) Aplicación nivel 2 → llave y código → probar** | ~9 clics + 2 campos; probar: **imposible** | 5 clics + 2 campos (+1 si se copia la llave) | El código usa el SDK real y la impresora elegida. La llave se copia o descarga. El primer ticket se ve en el **rollo en vivo**, y el **comprobador de permisos** dice por qué no imprimiría. |
| **d) Aprobar una solicitud de origen** | 4 clics | 4 clics | Dice si viene de este equipo, de la red o de Internet, y si usa `http`. Propone el nombre y el nivel recomendado, y explica qué hacer con la llave. |
| **e) Por qué falló un trabajo y qué hacer** | 4 clics, **con Internet**, solo el código | **2 clics, sin Internet** («Ver qué pasó» en Inicio y el trabajo) | Qué pasó, por qué suele pasar y qué hacer, con *Probar la impresora* ahí mismo. Enlaza la sección del manual y el diagrama del error. |
| **f) Contraseña / certificado / respaldo** | 2 clics + 3 campos / 3 / 2, en una página larga | 3 clics + 3 campos / 4 / 3, en pestañas con dirección propia | Un clic más por la pestaña, a cambio de no buscar en una página con 7 formularios. Las confirmaciones dicen qué pasará. |
| **g) Integrador prueba su app sin impresora** | **Imposible** en el panel (terminal y ~8 clics) | 1 clic (virtual) + su aplicación | Probar impresión: plantillas, constructor, JSON, vista previa y envío. Rollo en vivo, simulación de fallas y comprobador de permisos. |

### Prueba de usuario nuevo

[e2e/usuario-nuevo.spec.mjs](../../e2e/usuario-nuevo.spec.mjs) arranca un puente recién instalado y sigue el camino que sugiere «Primeros pasos»:

1. crear la contraseña;
2. *Crear impresora virtual de prueba* y su *Ticket de prueba*;
3. volver a Inicio, *Autorizar*, crear la aplicación y guardar la llave;
4. la aplicación imprime y su ticket aparece en el rollo en vivo.

Corre en el CI con las demás pruebas de punta a punta y falla si pasa de 5 minutos.

| Medida | Resultado |
|---|---|
| Acciones | 10 clics y 4 campos (2 de contraseña, nombre y dirección de la aplicación) |
| Tiempo de la prueba automática | primer ticket en 2.3 s; aplicación imprimiendo en 4.3 s |
| Estimación para una persona | Modelo de pulsaciones (KLM): unos 2.6 s por clic con la decisión, 0.28 s por carácter, unos 110 caracteres. Da poco más de 1 minuto de acciones. Con la lectura de cada pantalla, **2 a 3 minutos**. |

**Con una persona real** (pendiente; hace falta alguien que no conozca PRISMA):

1. **Antes de empezar:**
   - Instala el puente en un equipo limpio con el instalador de la versión.
   - Siéntala frente al navegador con la dirección del panel abierta.
   - Dale solo esta tarea: «Haz que esta página de prueba imprima un ticket. No hay impresora conectada». La página de prueba es el ejemplo del SDK, con la dirección del puente ya puesta.
2. **Durante la prueba:**
   - No ayudes.
   - Pídele que piense en voz alta.
   - Toma el tiempo hasta que el ticket aparezca en el rollo.
   - Anota cada duda, cada vuelta atrás y cada vez que abra la Ayuda.
3. **Para aprobar:**
   - Menos de 5 minutos.
   - Ninguna pregunta al observador.
   - Ninguna visita a la terminal.
4. **Después:** cada duda es un hallazgo nuevo para la siguiente versión.

## Hallazgos

| # | Estado | Cómo quedó |
|---|---|---|
| C1 | Resuelto | Ninguna pantalla se sale a lo ancho en 390 px. Lo comprueba `sinViolaciones` en cada pantalla de las pruebas de punta a punta. |
| C2 | Resuelto | «Primeros pasos» en Inicio, calculado con los datos del puente, hasta que está listo. |
| C3 | Resuelto | Impresora virtual dentro del puente, Probar impresión, rollo en vivo y comprobador de permisos. |
| C4 | Resuelto | Manual y diagramas embebidos, con búsqueda sin Internet (ADR [0017](../adr/0017-manual-del-panel-con-renderizador-propio.md) y [0018](../adr/0018-diagramas-del-panel-en-svg-pregenerado.md)). |
| C5 | Resuelto | Detalle del trabajo con qué pasó, por qué y qué hacer. Una prueba exige la explicación de cada código. |
| A1 | Resuelto | Foco visible en las tarjetas de opción. |
| A2 | Resuelto | `novalidate`, errores junto al campo con `aria-invalid` y el foco al primer error. |
| A3 | Resuelto | Alias propuesto a partir del nombre; el ejemplo va en el texto de ayuda. |
| A4 | Resuelto | Vista previa raster al ancho del papel, compatible con la CSP. |
| A5 | Resuelto | «¿Cuál renglón se lee bien?» en lugar de «páginas de códigos». |
| A6 | Resuelto | El asistente tiene 3 pasos reales; guardar imprime la prueba. |
| A7 | Resuelto | El código para integrar usa el SDK publicado y la impresora elegida. |
| A8 | Resuelto | Desactivar una aplicación o una impresora pide confirmación y dice qué pasará. |
| A9 | Resuelto | Pestañas ARIA con flechas; los resultados se anuncian; los `pre` reciben el foco. |
| M1 | Resuelto | Textos sin jerga y «Saber más» en los campos técnicos. |
| M2 | **Pendiente** | El panel no se actualiza solo. Solo se actualizan el rollo en vivo y el aviso de sesión. Ya se puede hacer sin cambiar el puente, con la cabecera `X-Prisma-Sondeo`, que no renueva la sesión. Queda en las [propuestas](PROPUESTAS-V2.md#inicio-que-se-actualiza-solo). |
| M3 | Resuelto | Los tickets de hoy ya no cuentan las pruebas del panel. |
| M4 | Resuelto | Ajustes en cuatro pestañas. |
| M5 | Resuelto | Las solicitudes dicen de dónde vienen y si usan `http`. |
| M6 | **Pendiente** | La barra todavía muestra «PRISMA / PRISMA» cuando el puente conserva el nombre predeterminado ([despues/12](despues/12-inicio-escritorio.webp)). Queda en las [propuestas](PROPUESTAS-V2.md#nombre-del-puente-repetido). |
| M7 | Resuelto | «Desactivada», con su color. |
| M8 | Resuelto | Una acción principal según el estado y el error sin el código delante. |
| M9 | Resuelto | Aviso 2 minutos antes de cerrar la sesión, con «Seguir conectado». |
| M10 | Resuelto | Las confirmaciones importantes se quedan como nota; los avisos de error ofrecen «Cómo resolverlo». |
| M11 | Resuelto | Activar y desactivar desde la tarjeta. |
| M12 | Resuelto | «Confirmar impresión» en una línea sin jerga. |
| B1 | Resuelto | «Saltar al contenido» (lo prueba [e2e/teclado.spec.mjs](../../e2e/teclado.spec.mjs)). |
| B2 | Resuelto | Fechas y números con `es-MX`, desde `textos.js`. |
| B3 | Resuelto | Al cambiar de pantalla el foco va al título, sin contorno. |
| B4 | En parte | El menú del celular se cierra con Escape, devuelve el foco al botón y se cierra al elegir una sección. Todavía no se cierra al tocar fuera. |
| B5 | **Pendiente** | Rechazar no ofrece «Deshacer». Sí se puede revertir desde la pestaña «Rechazadas». Queda en las [propuestas](PROPUESTAS-V2.md#deshacer-al-rechazar-un-origen). |
| B6 | Resuelto | Trabajos dice cuántos hay y cuántos días se conservan. |
| B7 | Resuelto | Inicio sin trabajos tiene su propio mensaje. |

## Accesibilidad

| | Antes | Después |
|---|---|---|
| axe (WCAG 2.0/2.1 A y AA) | 24 pantallas, solo tema claro: `scrollable-region-focusable` en 5 | 34 pantallas × claro y oscuro × escritorio y celular, más cada paso de los flujos: **0 críticas o serias** |
| Reflujo (1.4.10) | No cumple (C1) | Cumple: ninguna pantalla se desplaza a lo ancho a 390 px |
| Foco visible (2.4.7) | No cumple (A1) | Cumple |
| Teclado (2.1.1) | Parcial | Cumple: saltar al contenido, menú con Tab, atajos, asistente sin ratón, diálogos con Escape que devuelven el foco, visor de diagramas con teclas |
| Evitar bloques (2.4.1) | Parcial | Cumple |
| Errores (3.3.1, 3.3.3) | No cumple (A2) | Cumple |
| Mensajes de estado (4.1.3) | Parcial | Cumple |
| Contraste en tema oscuro | Sin revisar | Lo revisa axe en cada pantalla. Una prueba de Go verifica los tokens de color (AA). |

Lo comprueban en cada cambio [e2e/pantallas.spec.mjs](../../e2e/pantallas.spec.mjs) y [e2e/teclado.spec.mjs](../../e2e/teclado.spec.mjs), en el trabajo `e2e` del CI.

## Impresora virtual

Dos verificaciones que el plan dejó para esta fase:

- **Coincide con los golden de ESC/POS.**
  - Prueba: `TestImpresoraVirtualCoincideConLosGolden` (internal/servidor).
  - Imprime cada ejemplo válido del repositorio por `/v1/imprimir` en una impresora virtual de 58 y de 80 mm.
  - Exige que el rollo tenga la misma vista y la misma traza que los golden de `internal/dev`.
  - Un golden alterado a propósito hace fallar la prueba.
- **Sus tickets no sobreviven a un reinicio.**
  - Prueba: la segunda de [e2e/usuario-nuevo.spec.mjs](../../e2e/usuario-nuevo.spec.mjs).
  - Detiene el proceso del puente y lo vuelve a arrancar.
  - Al volver, el rollo está vacío y el trabajo sigue en el historial.
  - El texto del ticket no aparece en ningún archivo de la carpeta de datos: ni en la base ni en la bitácora.

## Tamaño

Compilado con `go build -trimpath -ldflags "-s -w"`, como en la publicación:

| | 0.1.0 | 0.2.0 | Diferencia |
|---|---|---|---|
| Windows amd64 | 17.0 MB | 19.6 MB | +2.6 MB (+15 %) |
| Linux arm64 | 15.9 MB | 18.4 MB | +2.5 MB (+16 %) |

De dónde salen, aproximadamente (lo embebido no se comprime):

| Origen | Tamaño |
|---|---|
| Capturas del manual (WebP) | 0.8 MB |
| Diagramas SVG claro y oscuro (gzip) | 0.6 MB |
| Markdown del manual | 0.35 MB |
| HTML, CSS y JS del panel | 0.1 → 0.39 MB |
| Código nuevo (impresora virtual, raster, pruebas del panel) y `golang.org/x/image` | el resto |

La memoria del puente solo crece con la impresora virtual: guarda los últimos 20 tickets por impresora (se puede cambiar), con un tope de 16 MB por impresora.

## Lo que queda

- **M2, M6, B5 y el resto de B4:** anotados en [PROPUESTAS-V2.md](PROPUESTAS-V2.md).
- **Prueba con una persona real:** con el protocolo de [Prueba de usuario nuevo](#prueba-de-usuario-nuevo).
- **Matriz manual antes de publicar:** [docs/desarrollo/pruebas/matriz.md](../desarrollo/pruebas/matriz.md).
