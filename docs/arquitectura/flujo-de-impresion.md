# Del documento a los bytes

Qué hace el puente con un documento desde que lo acepta hasta que la impresora lo recibe. La secuencia completa de una petición está en [diagramas/secuencia-imprimir.md](../diagramas/secuencia-imprimir.md).

## 1. Validación (`internal/contrato`)

`Decodificar` aplica el orden de la especificación (§9 del contrato):

1. Tamaño del cuerpo: máximo 256 KB.
2. JSON estricto: UTF-8, sin BOM, sin claves repetidas, anidamiento limitado.
3. Versión, que se revisa antes que el esquema para dar `VERSION_NO_SOPORTADA` y no un error genérico.
4. JSON Schema 2020-12. El esquema está **embebido** desde `schema/`, así que se valida exactamente el mismo archivo que se publica.
5. Reglas que no dependen de la impresora: dígitos verificadores, PNG válido de hasta 4096 px, raw decodificable, suma de porcentajes de fila y vigencia de la firma.

El resultado son tipos Go (`*contrato.Texto`, `*contrato.Fila`, …) con los valores por defecto aplicados. Los errores son `*contrato.Error{codigo, mensaje, sugerencia, detalles[{ruta, regla, mensaje}]}`.

## 2. Renderizado (`internal/escpos`)

`Renderizar(doc, perfil, recursos)` genera los bytes y la lista de advertencias.

- **Texto:**
  - Se normaliza a NFC y se convierte a la página de códigos del perfil (CP437, CP850, CP858 o WPC1252).
  - Lo que no existe en la página se imprime sin acento o como `?`, con advertencia y nunca con error.
  - Se mide por runas, a un carácter por columna, y se parte por palabras.
- **Fila y par:** reparten las columnas del perfil con el algoritmo de §4.3 del contrato.
- **QR y códigos de barras:** se envían por comando (`GS ( k`, `GS k`). Si el perfil no los soporta, se dibujan como imagen: el QR con `skip2/go-qrcode` y los códigos de barras con un generador propio (`internal/escpos/barras.go`).
- **Imágenes:** se decodifica el PNG, se compone sobre blanco, se reduce sin agrandar, se pasa a 1 bit con Floyd–Steinberg y se envía con `GS v 0` en bandas de 255 líneas.
- **Copias:** repiten el ticket completo.

## 3. Cola (`internal/cola`)

- **Una goroutine por impresora:** atiende en orden FIFO y de a un trabajo a la vez; las impresoras distintas trabajan en paralelo.
- **Límite:** como máximo 20 trabajos por impresora, entre los que esperan y el que se imprime; el siguiente recibe `COLA_LLENA`.
- **Resultado:** llega por un canal: ok, duración, error y, si la impresora recibió el ticket pero informó un problema, un `Aviso` (el trabajo cuenta como completado y el aviso queda como advertencia).
- **Estado:** la misma goroutine llama a `Probar()` cada 30 s, así que nunca prueba mientras imprime. Estados posibles: `desconocido` (todavía sin probar), `en_linea`, `fuera_de_linea` y `ocupada`. Ver [diagramas/estados-impresora.md](../diagramas/estados-impresora.md).

## 4. Conectores (`internal/conector`)

Todos implementan `Enviar(ctx, datos)` y `Probar(ctx)`. Las variantes por sistema están en archivos con build tags (`usb_windows.go`, `usb_unix.go`).

| Tipo | Implementación |
|---|---|
| red | TCP: 3 s para conectar y 10 s para escribir. Un reintento, y solo si no se envió ningún byte (repetir un envío parcial imprimiría dos veces). Con *Confirmar impresión*, envía `DLE EOT 1` después del ticket y espera la respuesta hasta 10 s. |
| usb (Windows) | winspool.drv: documento RAW (`OpenPrinterW` … `WritePrinter` … `ClosePrinter`). Si vence el plazo, cancela con `AbortPrinter`. |
| usb (Linux/macOS) | `/dev/usb/lpN` directo, o `lp -d <cola> -o raw` (CUPS). |
| bluetooth | Puerto serie (`go.bug.st/serial`), con la misma confirmación opcional que la red. |
| archivo | `<fecha>-<id>.bin` más un `.txt` de previsualización. |

Los detalles por sistema operativo están en [instalacion/impresoras.md](../instalacion/impresoras.md).

## Errores

Cada error visible lleva un **código estable en MAYÚSCULAS**, un **mensaje** y una **sugerencia** en español:

- **Documento:** `contrato.Error`, con los códigos del contrato (§9), por ejemplo `FILA_NO_CABE`.
- **Impresión:** `conector.Error`, por ejemplo `IMPRESORA_INALCANZABLE` o `COLA_LLENA`.
- **Advertencias:** no impiden imprimir y quedan en el trabajo (`CARACTER_SUSTITUIDO`, `SIN_CUCHILLA`, `SIN_CAJON`, `QR_REDUCIDO`). El conector puede añadir `IMPRESORA_FUERA_DE_LINEA` (con Confirmar impresión, ver [instalacion/impresoras.md](../instalacion/impresoras.md#confirmar-impresión)).

## Herramientas de desarrollo

`internal/dev` incluye una **impresora virtual** que interpreta ESC/POS: alineación, estilos, tamaños, raster, QR, códigos de barras, corte y cajón. Sobre ella funcionan:

- la vista en terminal y el HTML de previsualización, que salen de los bytes reales y no de una maqueta;
- el decodificador (`prisma dev decodificar`);
- la impresora de red simulada (`prisma dev impresora-virtual`);
- las pruebas golden de cada ejemplo a 58 y 80 mm.
