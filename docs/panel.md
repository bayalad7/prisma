# Panel de administración

El panel es la página web donde el dueño del negocio configura PRISMA: impresoras, aplicaciones autorizadas, solicitudes de origen, historial de trabajos, imágenes y ajustes. Lo sirve el mismo programa `prisma` que imprime, en el mismo puerto que la API:

```
https://<ip-del-equipo-del-puente>:8443/admin
```

No necesita Internet ni ninguna cuenta: todo vive en la base de datos del equipo donde está instalado el puente. Funciona en computadora y en celular.

El panel solo está disponible mientras el puente está en marcha:

- **En un negocio:** el puente corre como servicio y arranca solo ([instalacion.md](instalacion.md)).
- **En tu computadora, para probar:** ejecuta `prisma ejecutar --config config.dev.toml`, deja esa terminal abierta y entra a `https://localhost:8443/admin`. Los pasos están en [primeros-pasos.md](primeros-pasos.md#7-configurar-todo-desde-el-panel).

La primera vez el navegador avisará que el certificado es autofirmado. Continúa, o instálalo como explica [tls.md](tls.md).

## Primer acceso

La primera vez que se abre el panel hay que crear la contraseña de administrador (al menos 10 caracteres).

| Desde dónde | Qué hace falta |
|---|---|
| El propio equipo del puente (`https://localhost:8443/admin`) | Nada más: el panel pide la contraseña nueva. |
| Otro equipo de la red | Un **código de configuración** de un solo uso. En el equipo del puente ejecuta `prisma admin codigo-inicial` y escribe en el panel el código que muestra (por ejemplo `8527-HDH3`). Vence a los 30 minutos y se anula tras 10 intentos fallidos. |

Así, aunque el puente quede encendido sin configurar, nadie de la red puede adueñarse del panel sin acceso al equipo.

Si olvidas la contraseña, `prisma admin restablecer-contrasena` genera una nueva. El panel obliga a cambiarla al entrar y cierra las sesiones abiertas.

## Secciones

| Sección | Para qué sirve |
|---|---|
| **Inicio** | Estado de las impresoras, trabajos de hoy, solicitudes pendientes, los últimos trabajos y el aviso de una versión nueva. |
| **Impresoras** | Agregar una impresora en 3 pasos: elegir la conexión (detectadas, escaneo de red en el puerto 9100 o a mano), elegir el papel (58 u 80 mm, con opciones avanzadas) y probar. Por impresora: **Probar conexión** (sin imprimir), **Ticket de diagnóstico**, **Probar páginas de códigos**, editar, desactivar y eliminar. |
| **Aplicaciones** | Las aplicaciones autorizadas a imprimir (en la API se llaman *clientes*). Nivel de seguridad, orígenes, impresoras permitidas y operaciones sensibles. |
| **Solicitudes** | Orígenes desconocidos que intentaron imprimir. Se aprueban (con una aplicación nueva o existente), se rechazan o se olvidan. |
| **Trabajos** | Historial filtrable por aplicación, impresora, estado y fechas. Nunca muestra el contenido de los tickets: el puente no lo guarda. |
| **Imágenes** | PNG guardados en el puente para usarlos con `"recurso"` en la operación `imagen`. Muestra cómo quedarán en 1 bit antes de guardarlos. |
| **Ajustes** | Nombre del puente, límites de peticiones, retención del historial, acceso solo desde la red local, certificado HTTPS, contraseña, respaldo y restauración. |
| **Ayuda** | Cómo confiar en el certificado en cada sistema y enlaces a esta documentación. |

### Impresoras: diagnóstico y páginas de códigos

- **Ticket de diagnóstico.** Imprime el alias, la conexión, el papel, la página de códigos, una regla que debe ocupar exactamente una línea, los acentos, las tres alineaciones y un QR (si la impresora puede imprimirlo). El panel muestra además cómo debería verse. Si la regla se parte en dos líneas, las columnas están mal configuradas.
- **Probar páginas de códigos.** Imprime la misma línea con acentos con `ESC t 0`, `2`, `16` y `19`. Elige en el panel la que se leyó bien y el perfil se actualiza solo.

Los dos quedan en el historial como trabajos de la aplicación «Panel».

### Aplicaciones: niveles y llaves

| Nivel | Cómo se identifica | Qué muestra el panel |
|---|---|---|
| 1 — Origen | Solo el encabezado `Origin` del navegador. Necesita al menos un origen. | — |
| 2 — Llave (por defecto) | Origen (si viene de un navegador) y la llave `pk_…`. | La llave **una sola vez**, con botón para copiarla y el código de integración. |
| 3 — Firma | Documentos firmados con Ed25519 por el backend de la aplicación. | Valida la llave pública (PEM o base64) y muestra su huella. Opcionalmente exige también una llave `pk_`. |

- **Regenerar la llave:** la anterior deja de funcionar en ese momento.
- **Revocar la llave:** en el nivel 2 la aplicación no puede imprimir hasta que se genere otra; en el nivel 3 sigue imprimiendo con documentos firmados.
- `abrir_cajon` y `raw` están deshabilitadas hasta que las marques para esa aplicación.

### Solicitudes de origen

Cuando una página web desconocida intenta imprimir, el puente la rechaza (`ORIGEN_NO_AUTORIZADO`) y la anota aquí con el número de intentos. Al **aprobarla** eliges:

- **Aplicación nueva** (nivel 1 o 2): puede imprimir en todas las impresoras y sin operaciones sensibles; ajústala después en Aplicaciones. Con nivel 2 se muestra la llave una vez.
- **Aplicación existente:** el origen se agrega a esa aplicación con sus permisos.

**Rechazar** la oculta: sus intentos se siguen contando, pero ya no aparece como pendiente. **Olvidar** la borra; si el origen vuelve a intentarlo, aparecerá otra vez.

### Ajustes

| Ajuste | Efecto |
|---|---|
| Límites de peticiones | Por IP y por aplicación, por minuto (0 = sin límite). Se aplican al momento y mandan sobre `[limites]` de `config.toml`, que queda como valor inicial. |
| Retención | Días que se conserva el historial de trabajos (1 a 3650). |
| Solo red local | Activado por defecto: el panel solo responde a IPs privadas (192.168.x.x, 10.x.x.x, 172.16–31.x.x, IPv6 locales) y al propio equipo. El equipo del puente siempre puede entrar. |
| Certificado | Muestra nombres, IPs, vencimiento y huella. Con `modo_tls = "autofirmado"` se puede **regenerar** (por ejemplo, si cambió la IP del equipo) y **descargar** para instalarlo en los equipos que imprimen. El puente lo usa sin reiniciar. |
| Actualizaciones | Activado por defecto: una vez al día, el puente pregunta a GitHub si hay una versión nueva y la avisa en *Inicio*. No envía datos del negocio y nunca instala nada solo; se instala con el instalador o con `prisma actualizar`. Desactívalo si el equipo no debe hacer conexiones salientes. |
| Respaldo | Descarga una copia completa y consistente de la base de datos, sin detener el puente. |
| Restaurar | Sube un respaldo. El puente lo valida, reemplaza la base con una pausa breve, recarga impresoras y ajustes, y cierra todas las sesiones. La base anterior queda junto a la nueva como `prisma.db.antes-de-restaurar`. |

## Seguridad del panel

- **Contraseña:** guardada con argon2id. Tras 5 intentos fallidos desde una misma IP, esa IP queda bloqueada 5 minutos.
- **Sesión:** cookie `HttpOnly`, `Secure` (con HTTPS) y `SameSite=Strict`, limitada a `/admin`. Termina tras 30 minutos sin actividad, a las 12 horas, al reiniciar el puente o al cambiar la contraseña.
- **CSRF:** cada cambio lleva un token de la sesión en `X-CSRF-Token` y debe venir del mismo origen que el panel.
- **Contenido:** HTML, CSS y JavaScript van dentro del binario, sin CDNs ni recursos externos, con una política CSP que solo permite archivos del propio puente.
- **Bitácora:** registra las acciones (impresora agregada, aplicación creada, origen aprobado…), nunca contraseñas, llaves, códigos ni el contenido de los tickets.

## API interna

El panel usa rutas JSON bajo `/admin/api/` protegidas por la sesión. **No son una API pública**: pueden cambiar entre versiones sin aviso. Las aplicaciones deben usar la [API v1](api-v1.md).
