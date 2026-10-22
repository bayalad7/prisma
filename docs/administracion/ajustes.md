# Ajustes del panel

*Ajustes* se divide en cuatro pestañas: **General** (nombre del puente, límite de tickets, historial y versiones nuevas), **Seguridad y acceso** (contraseña y acceso solo desde la red del negocio), **HTTPS** (el certificado) y **Respaldo** (descargar y restaurar). La pestaña abierta queda en la dirección (`/admin/ajustes?pestana=https`), así que se puede enlazar. Los valores fuera de rango, una contraseña actual equivocada o una restauración sin archivo se marcan junto al campo.

| Ajuste | Efecto |
|---|---|
| Límites de peticiones | Por IP y por aplicación, por minuto (0 = sin límite). Se aplican al momento y mandan sobre `[limites]` de `config.toml`, que queda como valor inicial. |
| Retención | Días que se conserva el historial de trabajos (1 a 3650). |
| Solo red local («Abrir el panel solo desde la red del negocio») | Activado por defecto; al desactivarlo, el panel pide confirmación y explica el riesgo: el panel solo responde a IPs privadas (192.168.x.x, 10.x.x.x, 172.16–31.x.x, IPv6 locales) y al propio equipo. El equipo del puente siempre puede entrar. |
| Certificado | Muestra nombres, IPs, vencimiento y huella. Con `modo_tls = "autofirmado"` se puede **regenerar** (por ejemplo, si cambió la IP del equipo) y **descargar** para instalarlo en los equipos que imprimen. El puente lo usa sin reiniciar. Desde 60 días antes de vencer, *Inicio* muestra un aviso y la fila *Vence* lo marca; ver [instalacion/https-y-certificados.md](../instalacion/https-y-certificados.md#vencimiento). |
| Actualizaciones | Activado por defecto: una vez al día, el puente pregunta a GitHub si hay una versión nueva y la avisa en *Inicio*. No envía datos del negocio y nunca instala nada solo; se instala con el instalador o con `prisma actualizar`. Desactívalo si el equipo no debe hacer conexiones salientes. |
| Respaldo | Descarga una copia completa y consistente de la base de datos, sin detener el puente. |
| Restaurar | Hay que elegir el archivo y escribir RESTAURAR; el panel sugiere descargar antes un respaldo de lo actual. Sube un respaldo. El puente lo valida, reemplaza la base con una pausa breve, recarga impresoras y ajustes, y cierra todas las sesiones. La base anterior queda junto a la nueva como `prisma.db.antes-de-restaurar`. |


Cómo restaurar un respaldo paso a paso, y qué hacer si algo falla, está en [operacion/respaldo-y-restauracion.md](../operacion/respaldo-y-restauracion.md).
