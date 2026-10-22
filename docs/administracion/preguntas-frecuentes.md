# Preguntas frecuentes

## ¿PRISMA necesita Internet?

No. Imprime, se administra y muestra este manual sin Internet. Solo lo usa, si lo dejas activado, para avisar una vez al día si hay una versión nueva (*Ajustes → General*), y los enlaces marcados «requiere Internet» llevan a GitHub.

## ¿PRISMA guarda lo que imprimo?

No. Guarda que llegó un ticket, de qué aplicación, a qué impresora y cómo terminó, pero nunca su contenido. La única excepción son las impresoras virtuales: muestran los últimos tickets en pantalla, solo en la memoria del puente, y los olvidan al reiniciar ([Privacidad](probar-sin-impresora.md#privacidad)).

## ¿Qué pasa si se apaga o se reinicia el equipo del puente?

PRISMA está instalado como servicio: arranca solo al encender el equipo. Lo que se estaba imprimiendo en ese momento queda como fallido, y la aplicación puede volver a enviarlo sin que salga dos veces.

## ¿Qué pasa si la impresora está apagada cuando alguien imprime?

El ticket no se imprime y la aplicación recibe un error (`IMPRESORA_INALCANZABLE`) para avisar o reintentar. En *Trabajos* aparece como fallido, con qué revisar.

## ¿Puedo usar el panel desde el celular?

Sí, desde cualquier equipo o celular conectado a la red del negocio. Por seguridad, de fábrica no se abre desde fuera de esa red ([Ajustes](ajustes.md)).

## ¿Cualquier página de mi red puede imprimir?

No. Solo las aplicaciones que autorizaste, en las impresoras que les permitiste. Si una página desconocida lo intenta, se rechaza y aparece en *Solicitudes*. Abrir el cajón y enviar comandos directos están apagados hasta que los permitas para una aplicación.

## ¿Cuántas impresoras y aplicaciones puedo tener?

Las que necesites: cada impresora tiene su propia fila de tickets, así que una impresora lenta no retrasa a las demás.

## ¿Qué hago si olvido la contraseña del panel?

En el equipo del puente, como administrador: `prisma admin restablecer-contrasena`. Te da una contraseña temporal y el panel te pide cambiarla al entrar ([detalle](problemas-comunes.md#olvidé-la-contraseña-del-panel)).

## ¿Cómo cambio de equipo el puente?

Descarga un respaldo en *Ajustes → Respaldo*, instala PRISMA en el equipo nuevo y restáuralo ahí. Para no reinstalar el certificado en cada caja, copia también la carpeta `tls/` ([Respaldo y restauración](../operacion/respaldo-y-restauracion.md)).

## ¿Cómo sé si hay una versión nueva?

El Inicio lo avisa. Se instala desde el equipo del puente, y solo si está firmada por el proyecto ([Actualización](../operacion/actualizacion.md)).

## Una aplicación imprimió dos veces el mismo ticket, ¿por qué?

PRISMA no imprime dos veces el mismo documento: si la aplicación lo reenvía con el mismo identificador, responde que ya lo imprimió. Si salió dos veces, la aplicación lo mandó como dos documentos distintos; quien la programa puede revisarlo en [Idempotencia](../diagramas/24-idempotencia.md).
