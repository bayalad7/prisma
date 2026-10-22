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

Las pantallas de acceso permiten mostrar la contraseña mientras se escribe, marcan junto al campo lo que falta o no coincide (también un código de configuración equivocado) y, si el puente no responde, ofrecen *Volver a intentar*.

Si olvidas la contraseña, `prisma admin restablecer-contrasena` genera una nueva (el panel lo recuerda en *¿Olvidaste la contraseña?*). El panel obliga a cambiarla al entrar y cierra las sesiones abiertas.

## Secciones

Están agrupadas en *Operación*, *Integración* y *Sistema*, y cada una tiene su dirección (por ejemplo `/admin/impresoras`), que se puede guardar en favoritos. Con el teclado, `g` y la letra de la sección cambia de pantalla (`?` muestra la lista y permite desactivarlos). El tema claro u oscuro se elige al pie del menú.

| Sección | Para qué sirve |
|---|---|
| **Inicio** | El estado de un vistazo: impresoras listas o que no responden, solicitudes pendientes, tickets de hoy de tus aplicaciones (las pruebas del panel se cuentan aparte), días para que venza el certificado y los últimos trabajos, cada tarjeta con su acción. Mientras falte algo, **Primeros pasos** lista lo que queda (agregar una impresora, imprimir una prueba, revisar los acentos en una impresora física, autorizar una aplicación) con un botón para hacerlo; se puede ocultar y retomar. Avisa además de una versión nueva o de un certificado por vencer. |
| **[Impresoras](impresoras.md)** | Agregar una impresora en 3 pasos: el tipo (80 mm, 58 mm o virtual), la conexión (detectadas, búsqueda en la red o a mano) con su nombre, y un ticket de prueba con la revisión de los acentos. Por impresora: una insignia con su estado y la acción que toca según él: **Probar conexión** (sin imprimir), **Ticket de prueba**, **Probar acentos**, editar, activar o desactivar (con confirmación) y eliminar; las virtuales, además, **Ver tickets**. |
| **[Aplicaciones](aplicaciones-y-solicitudes.md)** | Las aplicaciones autorizadas a imprimir (en la API se llaman *clientes*). Nivel de seguridad, orígenes, impresoras permitidas y operaciones sensibles. |
| **[Solicitudes](aplicaciones-y-solicitudes.md#solicitudes-de-origen)** | Orígenes desconocidos que intentaron imprimir. Se aprueban (con una aplicación nueva o existente), se rechazan o se olvidan. |
| **Trabajos** | Historial con filtros (todos, fallidos, con aviso, en cola; y por aplicación, impresora y fechas). Al tocar un trabajo se ve **qué pasó, por qué suele pasar y qué hacer**, con *Probar la impresora* ahí mismo; los avisos (un acento que no existe, sin cuchilla…) también se explican, y el de una impresora virtual muestra su ticket. Las pruebas del panel aparecen como «Panel (prueba)». Nunca muestra el contenido de los tickets: el puente no lo guarda. |
| **Probar impresión** | Arma un ticket con una plantilla (diagnóstico, venta, venta con logo y QR, comanda, todas las operaciones), con el **constructor** (agregar texto, separador, par, fila, QR, código de barras, imagen guardada, corte…, cambiar sus opciones, reordenar y eliminar) o en JSON; cambiar de modo conserva el documento. Se valida mientras escribes (las mismas reglas que `/v1/imprimir`, con el renglón o la operación del error) y se dibuja al ancho de la impresora elegida. **Enviar a…** lo imprime en cualquier impresora activa, física o virtual, por el mismo camino que la API; el resultado muestra la respuesta tal como la recibiría una aplicación. **Copiar como código** da el equivalente con el SDK de JavaScript y con curl. Lo impreso queda en *Trabajos* a nombre de «Panel». |
| **Imágenes** | PNG guardados en el puente para usarlos con `"recurso"` en la operación `imagen`. Muestra cómo quedarán en 1 bit antes de guardarlos (en 58 u 80 mm) y explica junto al campo un nombre o un archivo que no sirven. |
| **[Ajustes](ajustes.md)** | En cuatro pestañas: General (nombre, límite de tickets, historial, versiones nuevas), Seguridad y acceso (contraseña, acceso solo desde la red del negocio), HTTPS (certificado) y Respaldo (descargar y restaurar). |
| **Ayuda** | Cómo confiar en el certificado en cada sistema y enlaces a esta documentación. |

Si algo no funciona, ver [problemas-comunes.md](problemas-comunes.md).

## Seguridad del panel

- **Contraseña:** guardada con argon2id. Tras 5 intentos fallidos desde una misma IP, esa IP queda bloqueada 5 minutos.
- **Sesión:** cookie `HttpOnly`, `Secure` (con HTTPS) y `SameSite=Strict`, limitada a `/admin`. Termina tras 30 minutos sin actividad, a las 12 horas, al reiniciar el puente o al cambiar la contraseña. Dos minutos antes del cierre por inactividad, el panel avisa con una cuenta regresiva; solo *Seguir conectado* (que pulsa la persona) renueva la sesión: el panel nunca la mantiene viva por su cuenta.
- **CSRF:** cada cambio lleva un token de la sesión en `X-CSRF-Token` y debe venir del mismo origen que el panel.
- **Contenido:** HTML, CSS y JavaScript van dentro del binario, sin CDNs ni recursos externos, con una política CSP que solo permite archivos del propio puente.
- **Bitácora:** registra las acciones (impresora agregada, aplicación creada, origen aprobado…), nunca contraseñas, llaves, códigos ni el contenido de los tickets.

## API interna

El panel usa rutas JSON bajo `/admin/api/` protegidas por la sesión. **No son una API pública**: pueden cambiar entre versiones sin aviso. Las aplicaciones deben usar la [API v1](../referencia/api-v1.md).
