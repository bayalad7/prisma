# Modelo de amenazas

Para PRISMA 0.1.0. Cada mitigación cita el código que la implementa; cada riesgo residual es una consecuencia aceptada del diseño o queda fuera del alcance del puente.

## Contexto

El puente corre en un equipo de la red local de un negocio. Recibe documentos de aplicaciones (navegadores, backends) por HTTPS y los imprime. El dueño lo administra desde un panel web en el mismo puerto. Ver [diagramas/01-contexto.md](../diagramas/01-contexto.md).

## Activos

| Activo | Por qué importa |
|---|---|
| La capacidad de imprimir | Tickets falsos (comandas, comprobantes) y gasto de papel. |
| `abrir_cajon` | Abre el cajón de dinero. |
| `raw` | Envía bytes ESC/POS arbitrarios a la impresora. |
| Llaves `pk_` de las aplicaciones | Permiten imprimir como esa aplicación. |
| Llave privada de nivel 3 | La guarda el backend del integrador; el puente nunca la tiene. |
| Contraseña y sesiones del panel | Control total del puente. |
| `prisma.db` | Configuración, hashes de llaves y de la contraseña, historial. |
| Llave privada del certificado TLS | Permite suplantar al puente ante las cajas. |
| Llave privada de las versiones | Permite publicar una actualización que todos los puentes aceptarían. Solo está en un secreto del CI. |
| Contenido de los tickets | Datos de clientes y ventas. **El puente no lo guarda.** |

## Actores

- **Página web maliciosa** abierta en un navegador de la red del negocio.
- **Equipo de la red** con herramientas propias (curl, un script): un empleado, un visitante en el Wi-Fi o un equipo comprometido.
- **Aplicación autorizada** que se comporta mal (errores, reintentos en bucle).
- **Usuario local** del equipo del puente sin permisos de administrador.
- **Atacante en la cadena de publicación** (cuenta de GitHub, CI, CDN).

El **administrador del equipo** del puente está fuera del modelo: controla el sistema operativo y, por lo tanto, todo.

## Límites de confianza y amenazas (STRIDE)

### 1. Aplicación → API `/v1` (red local)

| Amenaza | Mitigación | Código |
|---|---|---|
| **S**uplantar una aplicación desde una página web | El `Origin` debe pertenecer a una aplicación activa; el navegador no deja inventarlo. CORS con eco exacto, nunca `*`. | `seguridad/autorizacion.go`, `servidor/servidor.go` (`permitirCORS`, `preflight`) |
| Suplantar una aplicación desde la red | Nivel 2: llave `pk_` comparada con su hash argon2id en tiempo constante. Nivel 3: firma Ed25519. | `seguridad/llaves.go`, `seguridad/firma.go` |
| **T**ampering: alterar un documento en tránsito | TLS 1.2 o superior. Nivel 3: la firma cubre todo el documento (JCS). | `servidor/tls.go`, `seguridad/firma.go` |
| Repetir un documento capturado | Idempotencia por `id` durante 10 min, persistente; nivel 3: vigencia de 0 a 120 s y 30 s de tolerancia de reloj. | `almacen/ids_recientes.go`, `contrato/reglas.go` |
| **I**nformation disclosure: saber qué impresoras e IPs hay | Las aplicaciones solo conocen alias; `/v1/impresoras` filtra por permisos; `/v1/salud` no expone nada sensible. Los errores internos no dan detalles. | `servidor/api.go` |
| **D**enial of service | Límite por IP antes de todo y por aplicación después; cubetas limpiadas cada minuto. Cuerpo de 300 KB como máximo. Cálculo argon2id limitado a 2 a la vez y con caché, buscado por prefijo público. Cola de 20 por impresora. | `seguridad/limitador.go`, `seguridad/autorizacion.go`, `cola/cola.go` |
| Llenar la base con solicitudes de origen inventadas | Tope de 200, reemplazo de la pendiente más antigua y purga a los 30/90 días. | `almacen/solicitudes.go` |
| **E**levation: usar el cajón o `raw` | Deshabilitadas por aplicación hasta que el dueño las habilita. | `seguridad/autorizacion.go` (`VerificarPermisos`) |
| DNS rebinding | El certificado no es válido para un dominio ajeno, el `Origin` del atacante no está aprobado y la cookie del panel es `SameSite=Strict`. | `servidor/tls.go`, `admin/sesiones.go` |

**Riesgos residuales:**

- **Nivel 1 sin llave:** cualquier equipo de la red puede imprimir enviando un `Origin` aprobado con curl. Es la definición del nivel 1; por eso el nivel por defecto es el 2 (matriz manual, caso N-N8b).
- **Llave dentro de una página web:** una llave `pk_` en el JavaScript de una página la puede leer quien abra el código. El nivel 2 protege contra equipos sin la llave, no contra quien ya tiene la página. Para contenido que no debe alterarse, nivel 3.
- **Un ticket corto puede darse por impreso** si la impresora de red se apaga mientras lo recibe. Se mitiga con *Confirmar impresión*, desactivada por defecto.

### 2. Dueño → panel `/admin`

| Amenaza | Mitigación | Código |
|---|---|---|
| Adueñarse de un puente sin configurar | Desde otro equipo hace falta un código de un solo uso generado en el equipo del puente (30 min, anulado tras 10 fallos). | `admin/sesiones.go` |
| Adivinar la contraseña | argon2id (64 MiB, 3 pasadas), al menos 10 caracteres, bloqueo de 5 min por IP tras 5 fallos, un cálculo a la vez. | `seguridad/contrasena.go`, `admin/sesiones.go` |
| Robar o fijar la sesión | Token de 32 bytes aleatorios guardado como SHA-256; cookie `HttpOnly`, `Secure`, `SameSite=Strict` y limitada a `/admin`; 30 min de inactividad, 12 h máximo; un token nuevo en cada inicio de sesión; se invalidan al cambiar la contraseña. | `admin/sesiones.go` |
| CSRF | Token en `X-CSRF-Token`, comparado en tiempo constante, y `Origin` igual al del puente. | `admin/admin.go` |
| XSS o carga de código ajeno | HTML/CSS/JS embebidos, sin CDN; CSP `default-src 'self'` y `frame-ancestors 'none'`. | `admin/admin.go` (`cspPanel`) |
| Acceso desde fuera de la red local | *Solo red local* activado por defecto: solo IPs privadas y el propio equipo. | `admin/admin.go` (`redPermitida`) |

**Riesgos residuales:**

- La primera vez, el navegador muestra la advertencia del certificado autofirmado y el dueño la acepta. Hasta que instale el certificado, un atacante en la red podría interceptar esa sesión.
- Cualquier equipo de la red puede intentar entrar al panel: lo protegen la contraseña y el bloqueo, no la red.

### 3. Puente → impresoras

| Amenaza | Mitigación |
|---|---|
| Inyectar ESC/POS a través del texto | Se prohíben los caracteres de control en los textos del documento; los bytes arbitrarios solo existen en `raw`, deshabilitada por defecto (`contrato`, esquema). |

**Riesgo residual (fuera del alcance):** el puerto 9100 de una impresora de red **no tiene autenticación**. Cualquier equipo de la red puede imprimir directo en ella, sin pasar por PRISMA. Mitigación del negocio: poner las impresoras en una red o VLAN a la que solo llegue el equipo del puente.

### 4. Puente → GitHub (actualizaciones)

| Amenaza | Mitigación | Código |
|---|---|---|
| Instalar un binario falso | `checksums.txt` firmado con la llave Ed25519 del proyecto (la pública va dentro del binario) y SHA-256 de cada archivo. Nada se instala solo: lo pide el dueño. | `actualizacion/*`, `cmd/prisma/actualizar.go` |
| Filtrar datos del negocio | La consulta diaria solo pide la última versión; se desactiva en *Ajustes*. | `actualizacion/aviso.go` |

**Riesgos residuales:** si se filtra la llave privada de las versiones, un atacante podría publicar una actualización aceptada (ver la rotación en [politicas.md](politicas.md#llave-de-las-versiones)). Los instaladores de 0.1.0 no tienen firma de código de Windows ni de Apple: se verifican con `checksums.txt`.

### 5. Usuarios locales del equipo

| Amenaza | Mitigación |
|---|---|
| Leer la base o la llave del certificado | Base y llave en 0600, carpeta de datos 0700 o 0750 del usuario del servicio; en Windows, datos y bitácoras solo para SYSTEM y Administradores. |
| Leer el contenido de los tickets | No se guarda en la base ni en las bitácoras. |

### 6. Cadena de publicación y SDK

| Amenaza | Mitigación |
|---|---|
| Un SDK alterado en el CDN | El SDK se sirve desde una etiqueta fija del repositorio (`@v0.1.0`), no desde una rama. |
| Dependencias comprometidas | Pocas dependencias directas, versiones fijadas en `go.mod`/`go.sum`; el SDK no tiene dependencias. |
| Secretos del CI | La llave de las versiones solo está en el secreto `PRISMA_LLAVE_RELEASES`; el workflow verifica la firma contra la llave pública antes de publicar. |

**Riesgo residual:** el SDK no publica un hash de integridad (SRI) en la documentación. Un integrador que quiera fijarlo puede calcular el `integrity` del archivo de la etiqueta.
