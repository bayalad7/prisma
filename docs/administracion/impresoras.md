# Impresoras en el panel

La sección *Impresoras* agrega, prueba y ajusta las impresoras del puente. Antes de agregar una impresora USB o Bluetooth, prepárala en el sistema como explica [instalacion/impresoras.md](../instalacion/impresoras.md).

## Agregar una impresora

Son 3 pasos:

1. **Conexión.** Elige una impresora detectada (USB, colas del sistema, puertos serie), busca en la red (puerto 9100) o escribe el destino a mano.
2. **Papel.** 58 u 80 mm. Las *opciones avanzadas* fijan la página de códigos, el comando `ESC t`, las columnas de las fuentes A y B y los puntos por línea.
3. **Capacidades y prueba.** Cuchilla, cajón, QR y códigos de barras por comando, imágenes y, en red y Bluetooth, *Confirmar impresión*. Luego pulsa *Probar conexión*.

El **alias** (minúsculas, dígitos, `-` y `_`, hasta 32 caracteres) es el nombre con el que las aplicaciones eligen la impresora. Si lo cambias, las aplicaciones que usan el alias anterior reciben `IMPRESORA_NO_ENCONTRADA`.

## Diagnóstico y páginas de códigos

- **Ticket de diagnóstico.** Imprime el alias, la conexión, el papel, la página de códigos, una regla que debe ocupar exactamente una línea, los acentos, las tres alineaciones y un QR (si la impresora puede imprimirlo). El panel muestra además cómo debería verse. Si la regla se parte en dos líneas, las columnas están mal configuradas.
- **Probar páginas de códigos.** Imprime la misma línea con acentos con `ESC t 0`, `2`, `16` y `19`. Elige en el panel la que se leyó bien y el perfil se actualiza solo.

Los dos quedan en el historial como trabajos de la aplicación «Panel».

## Confirmar impresión

Solo en impresoras de red y Bluetooth. Después de cada ticket, el puente pregunta a la impresora si lo recibió; si no responde, el trabajo falla en vez de darse por impreso. Si responde que está fuera de línea (tapa abierta, sin papel), el trabajo se completa con la advertencia `IMPRESORA_FUERA_DE_LINEA`, porque el ticket saldrá al resolverlo.

Actívala y usa *Ticket de diagnóstico* para comprobar que tu impresora responde. Si el diagnóstico falla con `TIEMPO_AGOTADO` aunque el ticket salió, la impresora no admite la consulta: desactívala. Detalles en [instalacion/impresoras.md](../instalacion/impresoras.md#confirmar-impresión).

## Estado

Cada tarjeta muestra el estado (*en línea*, *fuera de línea*, *ocupada* o *desconocido*) y el último error. El puente prueba cada impresora cada 30 s, salvo mientras imprime.

## Desactivar y eliminar

- **Desactivar** (desmarcar *Activa* al editarla) la oculta a las aplicaciones (`IMPRESORA_NO_ENCONTRADA`) sin perder su perfil.
- **Eliminar** la borra con sus permisos; el historial de trabajos se conserva.
