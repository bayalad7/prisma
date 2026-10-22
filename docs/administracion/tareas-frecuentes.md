# Tareas frecuentes

Lo que se hace de vez en cuando en el panel, paso a paso.

## Agregar otra impresora

*Impresoras → Agregar impresora* y sigue los tres pasos (tipo, conexión y prueba), como en [Primeros pasos](primeros-pasos.md#3-agrega-la-impresora). Después, dale permiso a las aplicaciones que la usarán (siguiente tarea).

## Elegir en qué impresoras imprime una aplicación

1. *Aplicaciones* → en su tarjeta, *Editar*.
2. En *Impresoras*, marca *Puede imprimir en todas las impresoras* o elige las que puede usar.
3. Guarda los cambios.

Si una aplicación intenta imprimir en otra, recibe `IMPRESORA_NO_PERMITIDA`.

## Aprobar una página que quiere imprimir

1. *Solicitudes* muestra la página con sus intentos y de dónde viene (este equipo, tu red o Internet).
2. *Aprobar* → elige crear una aplicación nueva o agregarla a una existente.
3. Si la nueva lleva llave, cópiala: se muestra una sola vez.

Apruébala solo si sabes de quién es. Los detalles están en [Solicitudes de origen](aplicaciones-y-solicitudes.md#solicitudes-de-origen).

## Permitir abrir el cajón de dinero

1. La impresora debe tenerlo: *Impresoras → Editar → Tiene un cajón de dinero conectado*.
2. La aplicación debe tener permiso: *Aplicaciones → Editar → Permisos especiales → Abrir el cajón de dinero*.

## Cambiar la llave de una aplicación

Si la llave se perdió o la vio alguien que no debía: *Aplicaciones* → en su tarjeta, *Cambiar la llave*. La anterior deja de funcionar en ese momento; entrega la nueva a quien programa la aplicación.

## Saber qué pasó con un ticket

*Trabajos* lista lo que llegó, con filtros (fallidos, con aviso, por aplicación, impresora o fecha). Al tocar un trabajo, el panel explica qué pasó, por qué suele pasar y qué hacer, con un botón para probar la impresora. El contenido del ticket no se guarda.

## Saber por qué una aplicación no puede imprimir

*Aplicaciones* → en su tarjeta, *Comprobar permisos*: el panel revisa paso a paso, sin imprimir, si la aplicación pasaría y en qué paso se detiene ([detalle](aplicaciones-y-solicitudes.md)).

## Respaldar y restaurar

- **Respaldar:** *Ajustes → Respaldo → Descargar*. Guarda el archivo fuera del equipo del puente.
- **Restaurar:** *Ajustes → Respaldo*, elige el archivo y escribe RESTAURAR. Reemplaza toda la configuración del puente.

Más en [Respaldo y restauración](../operacion/respaldo-y-restauracion.md).

## Cambiar la contraseña del panel

*Ajustes → Seguridad y acceso → Contraseña*. ¿La olvidaste? En el equipo del puente, como administrador: `prisma admin restablecer-contrasena` ([Olvidé la contraseña del panel](problemas-comunes.md#olvidé-la-contraseña-del-panel)).

## Imprimir desde un equipo o celular nuevo

Ese equipo debe confiar en el certificado del puente: descárgalo en *Ajustes → HTTPS* e instálalo como explica [Confiar en el certificado](../instalacion/https-y-certificados.md#confiar-en-el-certificado).

## Actualizar PRISMA

El Inicio avisa cuando hay una versión nueva. Para instalarla, en el equipo del puente: el instalador nuevo, o `prisma actualizar` como administrador. Solo instala versiones firmadas por el proyecto ([Actualización](../operacion/actualizacion.md)).
