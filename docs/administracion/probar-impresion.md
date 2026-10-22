# Probar impresión

Arma un ticket, mira cómo saldrá y envíalo a cualquier impresora, física o virtual, sin programar nada. Lo que se imprime aquí queda en *Trabajos* a nombre de «Panel».

## Armar el ticket

Hay tres formas, y cambiar entre ellas conserva el ticket:

- **Plantillas:** el diagnóstico de la impresora elegida, una venta, una venta con logo y QR, una comanda o un ticket con todas las operaciones. Si una plantilla usa una imagen que no existe en este puente, lo avisa.
- **Constructor:** agrega texto, separadores, pares (concepto y valor), filas con columnas, renglones en blanco, códigos QR y de barras, imágenes guardadas, corte o apertura del cajón. Cada operación muestra solo sus opciones; se reordenan con *Subir* y *Bajar*.
- **JSON:** el documento tal como lo envía una aplicación ([Contrato «documento v1»](../referencia/contrato-documento-v1.md)). Se revisa mientras escribes, con las mismas reglas que la API, y el error dice el renglón o la operación.

## Vista previa y envío

La **vista previa** dibuja el ticket al ancho de la impresora elegida, con sus capacidades: si no corta o no tiene cajón, lo avisa antes de imprimir.

**Enviar a «impresora»** lo imprime por el mismo camino que la API. El resultado muestra la respuesta que recibiría una aplicación; si falla, explica qué pasó y ofrece *Probar conexión*.

**Copiar como código** da el mismo ticket con el SDK de JavaScript y con curl, para quien programa la aplicación.

## Rollo en vivo

En una impresora virtual, *Ver el rollo* (o *Ver tickets* en su tarjeta) abre su rollo: los tickets de cualquier aplicación aparecen solos. Está explicado en [Probar sin impresora](probar-sin-impresora.md#para-quien-programa-una-aplicación).

## ¿Por qué mi app no puede imprimir?

Describe cómo imprime una aplicación (desde su página, otra página o un servidor; con o sin llave; con o sin firma; en qué impresora; si usa el cajón) y el panel recorre, sin imprimir, los mismos pasos que sigue el puente al recibir un ticket:

1. La página web (origen).
2. La llave de la aplicación.
3. La firma del documento (nivel 3).
4. El permiso para la impresora.
5. Las operaciones sensibles (cajón, comandos directos).
6. Que la impresora exista y esté activa.

Dice en qué paso fallaría, qué respondería la API, cómo resolverlo y a qué pantalla ir. No pide la llave (supone que la aplicación envía la correcta) y no registra solicitudes. También se abre desde *Aplicaciones → Comprobar permisos*. El orden, en un dibujo: [Orden de autorización](../diagramas/07-orden-autorizacion.md).
