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

## Impresora virtual

Una impresora virtual vive dentro del puente: recibe los tickets igual que una de verdad, pero en lugar de imprimirlos los muestra en el panel. Sirve para probar el puente antes de tener la impresora, y para que quien programa una aplicación la pruebe sin papel.

- **Crearla:** en *Agregar impresora*, pulsa *Crear impresora virtual de prueba* (alias `prueba`). También se puede elegir el tipo *Virtual* al editar una impresora, o desde la terminal: `prisma impresoras agregar --alias prueba --tipo virtual`.
- **Usarla:** las aplicaciones la eligen por su alias, como cualquier otra. En `GET /v1/impresoras` aparece con `"virtual": true`, y en *Trabajos* sus trabajos llevan la insignia *Virtual*.
- **Ver los tickets:** *Ver tickets* muestra cada ticket como saldría en el papel (203 ppp), con la aplicación y la referencia, y permite descargar la imagen (PNG), los bytes exactos (`.bin`, para `prisma dev decodificar`) y ver los comandos ESC/POS.
- **Privacidad:** los tickets se guardan **solo en la memoria del puente**: nunca en la base de datos, la bitácora ni los respaldos. Se conservan los últimos 20 (se puede cambiar de 1 a 200) y se borran al reiniciar el puente o con *Vaciar rollo*.
- **Simular fallas:** al editarla, *Simulación* permite probar cómo responde una aplicación cuando algo sale mal, con los mismos errores que una impresora física:

| Simulación | Qué responde |
|---|---|
| Normal | Recibe todos los tickets. |
| Fuera de línea | Rechaza todo con `IMPRESORA_INALCANZABLE` (también *Probar conexión*). |
| Lenta | Tarda el retraso indicado (hasta 30 s) en recibir cada ticket. |
| Falla aleatoria | Rechaza con `IMPRESORA_INALCANZABLE` el porcentaje indicado de tickets. |

Un ticket rechazado no se guarda y su `id` queda libre: la aplicación puede reintentarlo sin que salga dos veces.

## Desactivar y eliminar

- **Desactivar** (desmarcar *Activa* al editarla) la oculta a las aplicaciones (`IMPRESORA_NO_ENCONTRADA`) sin perder su perfil.
- **Eliminar** la borra con sus permisos; el historial de trabajos se conserva.
