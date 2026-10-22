# Impresoras en el panel

La sección *Impresoras* agrega, prueba y ajusta las impresoras del puente. Antes de agregar una impresora USB o Bluetooth, prepárala en el sistema como explica [instalacion/impresoras.md](../instalacion/impresoras.md).

## Agregar una impresora

*Agregar impresora* guía en 3 pasos:

1. **Tipo.** De tickets de 80 mm (la más común), de 58 mm, o **virtual** para probar sin papel ([más abajo](#impresora-virtual)).
2. **Conexión y nombre.** Elige una de las impresoras que el puente encontró (conectadas a este equipo o, con *Buscar en la red*, en la red local). Las impresoras del sistema que no son de tickets (Microsoft Print to PDF, XPS, Fax…) aparecen marcadas *No parece de tickets*. Si la tuya no aparece, *¿No aparece?* permite escribir cómo está conectada (red, USB, Bluetooth o carpeta) y su dirección, con un ejemplo para cada caso. Escribe un **nombre** («Caja principal»): el **alias** se propone solo (`caja-principal`). *Probar conexión* comprueba que responda sin imprimir. *Opciones avanzadas* tiene la cuchilla, el cajón, QR, códigos de barras, imágenes y *Preguntar a la impresora si recibió cada ticket*.
3. **Prueba.** *Guardar y probar* guarda la impresora e imprime el ticket de diagnóstico; el panel muestra cómo debe verse y pregunta si salió y si se leen bien los acentos. Si no salió, sugiere qué revisar y permite reintentar o corregir la conexión. Si los acentos salen mal, abre la prueba de acentos.

El **alias** (minúsculas, dígitos, `-` y `_`, hasta 32 caracteres) es el nombre con el que las aplicaciones eligen la impresora; no se puede cambiar después. Los errores (un alias con espacios, una impresora sin elegir) se muestran junto al campo.

## Diagnóstico y páginas de códigos

- **Ticket de prueba** (ticket de diagnóstico). Imprime el alias, la conexión, el papel, la página de códigos, una regla que debe ocupar exactamente una línea, los acentos, las tres alineaciones y un QR (si la impresora puede imprimirlo). El panel muestra además cómo debería verse. Si la regla se parte en dos líneas, las columnas están mal configuradas.
- **Probar acentos.** Imprime 4 renglones con los mismos acentos, cada uno con una página de códigos distinta (empiezan con `ESC t 0`, `2`, `16` y `19`). En el panel eliges el renglón que se lee bien y el perfil se actualiza solo. Si ninguno se lee bien, el panel explica cómo buscar en el manual de la impresora el número correcto.

Los dos quedan en el historial como trabajos de la aplicación «Panel».

## Confirmar impresión

Solo en impresoras de red y Bluetooth. Después de cada ticket, el puente pregunta a la impresora si lo recibió; si no responde, el trabajo falla en vez de darse por impreso. Si responde que está fuera de línea (tapa abierta, sin papel), el trabajo se completa con la advertencia `IMPRESORA_FUERA_DE_LINEA`, porque el ticket saldrá al resolverlo.

Actívala y usa *Ticket de prueba* para comprobar que tu impresora responde. Si el diagnóstico falla con `TIEMPO_AGOTADO` aunque el ticket salió, la impresora no admite la consulta: desactívala. Detalles en [instalacion/impresoras.md](../instalacion/impresoras.md#confirmar-impresión).

## Estado

Cada tarjeta muestra el estado con una insignia (*En línea*, *No responde*, *Imprimiendo*, *Sin probar* o *Desactivada*), cómo está conectada, el papel, la página de códigos, lo que la impresora sabe hacer y, si lo hubo, el último problema explicado en palabras. El puente prueba cada impresora cada 30 s, salvo mientras imprime.

La acción principal cambia con el estado: si la impresora no responde, *Probar conexión* (comprueba que responda sin imprimir); si está lista, *Ticket de prueba*; si está desactivada, *Activar*. Los resultados aparecen dentro de la tarjeta con lo que conviene revisar.

## Impresora virtual

Una impresora virtual vive dentro del puente: recibe los tickets igual que una de verdad, pero en lugar de imprimirlos los muestra en el panel. Sirve para probar el puente antes de tener la impresora, y para que quien programa una aplicación la pruebe sin papel.

- **Crearla:** en *Agregar impresora*, pulsa *Crear impresora virtual de prueba* (alias `prueba`). También se puede elegir el tipo *Virtual* al editar una impresora, o desde la terminal: `prisma impresoras agregar --alias prueba --tipo virtual`.
- **Usarla:** las aplicaciones la eligen por su alias, como cualquier otra. En `GET /v1/impresoras` aparece con `"virtual": true`, y en *Trabajos* sus trabajos llevan la insignia *Virtual*.
- **Ver los tickets:** *Ver tickets* abre el **rollo en vivo** (`/admin/pruebas/rollo/{alias}`): cada ticket aparece en cuanto llega, de cualquier aplicación, como saldría en el papel (203 ppp), con la fecha, la aplicación, la referencia, el `trabajo_id` y las advertencias explicadas. Se puede ver en grande, descargar la imagen (PNG) y los bytes exactos (`.bin`, para `prisma dev decodificar`) y ver los comandos ESC/POS. Revisa cada 2 s mientras la pestaña se ve (se puede pausar) y no cuenta como actividad: la sesión se cierra igual a los 30 minutos sin usar el panel. Sirve para dejarlo abierto en otra ventana mientras programas tu aplicación.
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

- **Desactivar** (en la tarjeta, o desmarcando *Activa* al editarla) la oculta a las aplicaciones (`IMPRESORA_NO_ENCONTRADA`) sin perder su perfil. Pide confirmación y dice qué pasará; *Activar* la devuelve.
- **Eliminar** la borra con sus permisos; el historial de trabajos se conserva.
