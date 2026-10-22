# Panel de administración

El panel es la página web donde el dueño del negocio configura PRISMA: impresoras, aplicaciones autorizadas, solicitudes de origen, historial de trabajos, imágenes y ajustes. Lo sirve el mismo programa `prisma` que imprime, en el mismo puerto que la API:

```
https://<ip-del-equipo-del-puente>:8443/admin
```

No necesita Internet ni ninguna cuenta: todo vive en la base de datos del equipo donde está instalado el puente. Funciona en computadora y en celular.

El panel solo está disponible mientras el puente está en marcha:

- **En un negocio:** el puente corre como servicio y arranca solo ([instalacion/](../instalacion/README.md)).
- **En tu computadora, para probar:** ejecuta `prisma ejecutar --config config.dev.toml`, deja esa terminal abierta y entra a `https://localhost:8443/admin`. Los pasos están en [primeros-pasos.md](../desarrollo/primeros-pasos.md#7-configurar-todo-desde-el-panel).

La primera vez el navegador avisará que el certificado es autofirmado. Continúa, o instálalo como explica [instalacion/https-y-certificados.md](../instalacion/https-y-certificados.md).

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
| **Inicio** | Estado de las impresoras, trabajos de hoy, solicitudes pendientes, los últimos trabajos, el aviso de una versión nueva y el aviso de vencimiento del certificado. |
| **[Impresoras](impresoras.md)** | Agregar una impresora en 3 pasos: elegir la conexión (detectadas, escaneo de red en el puerto 9100 o a mano), elegir el papel (58 u 80 mm, con opciones avanzadas) y probar. Por impresora: **Probar conexión** (sin imprimir), **Ticket de diagnóstico**, **Probar páginas de códigos**, editar, desactivar y eliminar. |
| **[Aplicaciones](aplicaciones-y-solicitudes.md)** | Las aplicaciones autorizadas a imprimir (en la API se llaman *clientes*). Nivel de seguridad, orígenes, impresoras permitidas y operaciones sensibles. |
| **[Solicitudes](aplicaciones-y-solicitudes.md#solicitudes-de-origen)** | Orígenes desconocidos que intentaron imprimir. Se aprueban (con una aplicación nueva o existente), se rechazan o se olvidan. |
| **Trabajos** | Historial filtrable por aplicación, impresora, estado y fechas. Nunca muestra el contenido de los tickets: el puente no lo guarda. |
| **Imágenes** | PNG guardados en el puente para usarlos con `"recurso"` en la operación `imagen`. Muestra cómo quedarán en 1 bit antes de guardarlos. |
| **[Ajustes](ajustes.md)** | Nombre del puente, límites de peticiones, retención del historial, acceso solo desde la red local, certificado HTTPS, contraseña, respaldo y restauración. |
| **Ayuda** | Cómo confiar en el certificado en cada sistema y enlaces a esta documentación. |

Si algo no funciona, ver [problemas-comunes.md](problemas-comunes.md).

## Seguridad del panel

- **Contraseña:** guardada con argon2id. Tras 5 intentos fallidos desde una misma IP, esa IP queda bloqueada 5 minutos.
- **Sesión:** cookie `HttpOnly`, `Secure` (con HTTPS) y `SameSite=Strict`, limitada a `/admin`. Termina tras 30 minutos sin actividad, a las 12 horas, al reiniciar el puente o al cambiar la contraseña.
- **CSRF:** cada cambio lleva un token de la sesión en `X-CSRF-Token` y debe venir del mismo origen que el panel.
- **Contenido:** HTML, CSS y JavaScript van dentro del binario, sin CDNs ni recursos externos, con una política CSP que solo permite archivos del propio puente.
- **Bitácora:** registra las acciones (impresora agregada, aplicación creada, origen aprobado…), nunca contraseñas, llaves, códigos ni el contenido de los tickets.

## API interna

El panel usa rutas JSON bajo `/admin/api/` protegidas por la sesión. **No son una API pública**: pueden cambiar entre versiones sin aviso. Las aplicaciones deben usar la [API v1](../referencia/api-v1.md).
