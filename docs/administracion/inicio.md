# Inicio

La primera pantalla del panel: el estado del puente de un vistazo. Cada tarjeta tiene su acción, para ir directo a lo que hace falta.

## Primeros pasos

Mientras falte algo para que el puente quede listo, el Inicio lista lo que queda, con un botón para cada paso:

1. Crear la contraseña del panel.
2. Agregar una impresora (o una virtual, para probar sin papel).
3. Imprimir un ticket de prueba.
4. Revisar que salgan bien los acentos y la ñ (solo en impresoras de papel).
5. Autorizar la primera aplicación.

Los pasos salen de los datos del puente, así que se ven igual desde cualquier equipo. *Ocultar* la lista solo afecta a ese navegador. La guía completa está en [Primeros pasos](primeros-pasos.md).

## Las tarjetas

| Tarjeta | Qué muestra | Qué puedes hacer |
|---|---|---|
| **Impresoras** | Cuántas están listas y el estado de cada una. | *Revisar* la que no responde, o ir a Impresoras. |
| **Solicitudes** | Páginas que intentaron imprimir sin permiso. | Revisarlas en Solicitudes. |
| **Hoy** | Los tickets de tus aplicaciones de hoy y cuántos fallaron. Las pruebas del panel se cuentan aparte. | *Ver qué pasó* abre los fallidos en Trabajos. |
| **Certificado HTTPS** | Cuántos días faltan para que venza. | Instalarlo en otros equipos. |

Abajo están los **últimos trabajos**; *Ver todos* lleva a [Trabajos](trabajos.md).

## Avisos

El Inicio avisa arriba cuando hay algo que atender:

- **Una versión nueva de PRISMA.** Se instala desde el equipo del puente ([Actualización](../operacion/actualizacion.md)).
- **El certificado está por vencer o venció.** Con el certificado autofirmado, si vence el puente genera uno nuevo al arrancar, y hay que volver a confiar en él en cada equipo ([HTTPS y certificados](../instalacion/https-y-certificados.md#vencimiento)).
