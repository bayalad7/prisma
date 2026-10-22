# Trabajos

Cada ticket que llega al puente es un **trabajo**: de qué aplicación vino, a qué impresora fue, cuándo, cuánto tardó y cómo terminó. Por privacidad, el puente **no guarda el contenido** de los tickets: solo su resultado.

## Buscar un trabajo

- **Filtros rápidos:** *Todos*, *Fallidos*, *Con aviso* y *En cola*.
- **Más filtros:** por aplicación, impresora y fechas.
- Se muestran los 100 más recientes; los filtros encuentran los demás. El historial se conserva los días indicados en *Ajustes → General* (30 por defecto).

Los trabajos que imprime el propio panel (tickets de prueba, *Probar impresión*) aparecen como «Panel (prueba)»; los de la terminal, como «Consola (CLI)».

## Qué pasó con un trabajo

Al tocar un trabajo, el panel explica:

- **Qué pasó**, en palabras, por ejemplo «El puente intentó conectarse con la impresora y no respondió».
- **Por qué suele pasar**: impresora apagada, sin papel, desconectada, en otra red…
- **Qué hacer**, paso a paso, con *Probar la impresora* ahí mismo.

Si el trabajo **falló**, tu aplicación puede reintentarlo: como no se imprimió, no saldrá dos veces. Si se imprimió **con aviso** (un acento que la impresora no tiene, un corte que no hizo porque no tiene cuchilla), el aviso también se explica, con lo que conviene ajustar en la impresora.

En una impresora virtual, el detalle muestra además cómo salió el ticket ([Probar sin impresora](probar-sin-impresora.md)).

## Estados

| Estado | Significado |
|---|---|
| Impreso | La impresora recibió el ticket. |
| Impreso con aviso | Salió, pero algo se resolvió de otra forma; el aviso lo explica. |
| Falló | No se imprimió; el detalle dice por qué. |
| En cola | Espera su turno: la impresora atiende un ticket a la vez. |
| Imprimiendo | Se está enviando a la impresora. |

El recorrido completo de un trabajo, en un dibujo: [Ciclo de vida de un trabajo](../diagramas/08-ciclo-vida-trabajo.md). Los códigos de error, uno por uno: [Códigos de error](../referencia/codigos-de-error.md).
