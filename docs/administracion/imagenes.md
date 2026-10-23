# Imágenes

Logos e imágenes guardados en el puente, para que cualquier aplicación los imprima por su nombre sin enviarlos en cada ticket. En la API se llaman *recursos*.

## Subir una imagen

1. Elige un PNG (o arrástralo a la zona punteada). Hasta 512 KB y 4096 píxeles por lado.
2. Revisa la vista previa: las impresoras de tickets imprimen en blanco y negro, así que el panel muestra cómo quedará en el papel, en 58 o en 80 mm.
3. Escribe el **nombre** con el que la imprimirán las aplicaciones: minúsculas, números, guion y guion bajo (por ejemplo `logo-tienda`).
4. *Guardar imagen*.

Si el archivo no es un PNG o el nombre no sirve, el panel lo dice junto al campo.

## Usarla en un ticket

*Copiar uso* copia la operación lista para el documento:

```json
{"tipo": "imagen", "recurso": "logo-tienda"}
```

En *Probar impresión*, el constructor ofrece las imágenes guardadas en la operación *Imagen guardada*.

## Eliminar

*Eliminar* pide confirmación: los tickets que la usen darán un error (`RECURSO_NO_ENCONTRADO`) hasta que vuelvas a subir una imagen con el mismo nombre.

Las imágenes van en los respaldos del puente ([Respaldo y restauración](../operacion/respaldo-y-restauracion.md)).
