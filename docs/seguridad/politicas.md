# Políticas de seguridad

Cómo trata PRISMA 0.1.0 los datos, las credenciales y los archivos. Cada punto describe lo que hace el código; si cambia, esta página cambia con él.

## Qué se guarda y qué no

| Se guarda (en `prisma.db`) | No se guarda nunca |
|---|---|
| Impresoras y perfiles · aplicaciones, orígenes y permisos · hash argon2id y prefijo público de cada llave `pk_` · llaves **públicas** de nivel 3 · hash argon2id de la contraseña del panel · solicitudes de origen · historial de trabajos: estado, código de error, advertencias, duración, aplicación, impresora y referencia · `id` y hash del contenido de los documentos recientes (10 min, para la idempotencia) · imágenes guardadas · ajustes | El contenido de los tickets · las llaves `pk_` en claro · las llaves privadas de nivel 3 · la contraseña en claro · los códigos de configuración en claro |

**Bitácoras:** registran método, ruta (sin parámetros), estado, duración, IP y aplicación de cada petición, y los cambios hechos desde el panel. Nunca registran el contenido de los tickets, llaves, contraseñas, códigos ni firmas. Los errores que incluyen contenido del documento (por ejemplo, un código de barras inválido) se devuelven a quien los envió, pero no se registran. Ver [operacion/bitacoras.md](../operacion/bitacoras.md).

**Retención:** el historial de trabajos se borra a los 30 días (ajustable de 1 a 3650 en el panel); los `id` de idempotencia, a los 10 minutos; las solicitudes de origen pendientes, a los 30 días sin actividad, y las rechazadas, a los 90.

## Credenciales

| Credencial | Cómo se trata |
|---|---|
| Llave `pk_` (nivel 2) | Se genera en el puente y se muestra **una sola vez**. Se guarda su hash argon2id (19 MiB, 2 pasadas) y un prefijo público para encontrarla sin calcular nada. La comparación es en tiempo constante; el resultado, válido o no, queda en caché 10 minutos y se calculan como máximo 2 a la vez. Regenerarla invalida la anterior al momento. |
| Llave pública Ed25519 (nivel 3) | El dueño la pega en el panel (PEM o base64); el puente muestra su huella. El puente nunca conoce la privada. |
| Contraseña del panel | Al menos 10 caracteres, hash argon2id (64 MiB, 3 pasadas). 5 fallos desde una IP la bloquean 5 minutos. Restablecerla con la CLI obliga a cambiarla al entrar y cierra las sesiones. |
| Código de configuración inicial | De un solo uso, se guarda su hash, vence a los 30 minutos y se anula tras 10 intentos fallidos. Solo hace falta desde otro equipo: el propio equipo del puente entra sin código. |
| Sesión del panel | Token aleatorio de 32 bytes guardado como SHA-256, en memoria (se pierde al reiniciar). 30 min de inactividad, 12 h como máximo. |

## Autorización

- **Nivel por defecto:** 2 (origen y llave).
- **Operaciones sensibles:** `abrir_cajon` y `raw`, deshabilitadas para cada aplicación hasta que el dueño las habilita.
- **Impresoras:** una aplicación puede limitarse a algunas impresoras.
- **Orígenes:** exactos, normalizados y sin comodines. Un origen pertenece a una sola aplicación. `Origin: null` (iframes con sandbox, `file://`) se rechaza sin registrarse.
- **Orígenes desconocidos:** se rechazan y quedan como solicitud pendiente; el dueño decide.

## Red

- **TLS 1.2** o superior, siempre, salvo el modo `solo_localhost_http` (HTTP solo desde el propio equipo).
- **Certificado autofirmado:** ECDSA P-256, 825 días, con `localhost`, el nombre del equipo y sus IPs. Se avisa desde 60 días antes de vencer y se regenera solo si ya venció.
- **Panel:** por defecto solo responde a IPs privadas (RFC 1918 y equivalentes IPv6), de enlace local y al propio equipo.
- **CORS:** eco exacto del origen aprobado, nunca `*`, con `Vary: Origin` y sin credenciales; los orígenes desconocidos no reciben cabeceras.
- **Firewall de Windows:** el instalador abre el puerto solo para la subred local.
- **Nunca** se debe abrir el puerto 8443 hacia Internet.

## Cabeceras

| Cabecera | API `/v1` | Panel `/admin` |
|---|---|---|
| `Content-Security-Policy` | `default-src 'none'; frame-ancestors 'none'` | `default-src 'self'`, `img-src 'self' data: blob:`, `frame-ancestors 'none'`, `base-uri 'none'`, `form-action 'self'` |
| `X-Content-Type-Options` | `nosniff` | `nosniff` |
| `X-Frame-Options` | `DENY` | `DENY` |
| `Referrer-Policy` | `no-referrer` | `no-referrer` |
| `Cache-Control` | `no-store` | `no-store` |

## Archivos

| Archivo o carpeta | Permisos |
|---|---|
| Carpeta de la base | 0700 (al crearla el puente) o 0750 (al crearla el servicio), del usuario del servicio |
| `prisma.db` | 0600 |
| `tls/llave.pem` | 0600, en una carpeta 0700 |
| `respaldos/` | 0750 |
| Bitácoras | 0750, del usuario del servicio |
| Windows: `C:\ProgramData\PRISMA\datos` y `logs` | Solo SYSTEM y Administradores (`icacls`) |
| Tickets del conector de archivo (pruebas) | 0640 |

## Llave de las versiones

- **Uso:** firma `checksums.txt` de cada versión. `prisma actualizar` y el instalador de Linux rechazan cualquier versión cuya firma no verifique con la llave pública que llevan dentro.
- **Dónde está:** la pública, en `empaquetado/llave-releases.pub` y en `empaquetado/linux/instalar.sh` (una prueba comprueba que coincidan). La privada, **solo** en el secreto `PRISMA_LLAVE_RELEASES` del repositorio y en el gestor de contraseñas del dueño del proyecto.
- **Si se pierde:** los puentes instalados no podrán actualizarse con `prisma actualizar`; habrá que reinstalarlos con el instalador de una versión firmada con una llave nueva.
- **Si se filtra:** genera un par nuevo, reemplaza el secreto y la pública, publica una versión nueva lo antes posible y avísalo en [SECURITY.md](../../SECURITY.md). Los puentes que no actualicen seguirán aceptando la llave filtrada: indícales que reinstalen.
- **Cómo generarla:** [desarrollo/publicar.md](../desarrollo/publicar.md#una-sola-vez-la-llave-de-las-versiones).

## Modo desarrollo

`[dev] modo_desarrollo = true` solo habilita el comando local `prisma imprimir`, que imprime sin autorización. No abre nada en la red. Los instaladores lo dejan en `false`, y una prueba lo verifica (`TestConfiguracionDeLosInstaladores`). Nunca lo actives en un negocio.
