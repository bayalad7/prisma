# Reporte de verificación previa a la publicación

- **Fecha:** 21 de octubre de 2026
- **Versión revisada:** `develop` en `9872a39`, más las correcciones de este reporte.
- **Alcance:** código, seguridad, resiliencia y la experiencia del integrador. La matriz manual con hardware real está en [matriz.md](matriz.md).

## Resumen

| Severidad | Hallazgos | Estado |
|---|---|---|
| 🔴 Crítico | 2 | **Corregidos**, con pruebas nuevas |
| 🟠 Alto | 2 | **Corregidos** |
| 🟡 Medio | 6 | **Corregidos**; M3 con una opción nueva del perfil y documentado |
| 🔵 Bajo | 9 | **Corregidos** |

Lo que se ejecutó:

| Verificación | Resultado |
|---|---|
| `go test -race -count=1 ./...` (Linux, Docker) | ✅ Sin carreras de datos, en los 13 paquetes. |
| `go test ./...` en Windows y Linux | ✅ |
| `go vet` y staticcheck | ✅ |
| `errcheck` (errores ignorados) | ✅ Ninguno crítico; ver B1, B4 y B8. |
| CI en GitHub | ✅ Go y SDK en los 3 sistemas, y firmadores. ✅ **Servicio de Windows instalado, iniciado, reiniciado y desinstalado en el runner real.** ✅ Servicio en Ubuntu. Servicio en macOS: ❌ en la primera ejecución (C1). ✅ En la segunda (`39671479591`), con C1 corregido, arranca como `_prisma` y el panel responde; esa ejecución reveló B9. |
| Fugas de goroutines en la cola | ✅ Prueba nueva `TestSinFugasDeGoroutines`. |
| Contenido de tickets, llaves o contraseñas en los logs | ✅ Ninguno (revisadas las 44 llamadas a `slog`). |
| Resiliencia (6 escenarios contra el binario real) | ✅ con C2, M6 y A2 corregidos; M3 cubierto con *Confirmar impresión*. |
| Inicio rápido del integrador, al pie de la letra | ✅ Con A1 y B6 corregidos. |

---

## 🔴 Críticos (corregidos)

### C1. En macOS el servicio nunca arranca

- **Evidencia:** CI *Servicio (macos-latest)*. El LaunchDaemon queda en `spawn scheduled` y el panel no responde. La bitácora repite: `stat /Library/Application Support/PRISMA/config.toml: permission denied`.
- **Causa:** `servicio instalar` creaba la carpeta de datos con `MkdirAll(…, 0750)`, y eso creaba también su carpeta padre, `/Library/Application Support/PRISMA`, como root con 0750. El usuario `_prisma` no podía entrar ni a leer `config.toml`. En Linux no pasa porque `/var/lib` ya existe con 0755.
- **Corrección:** [`internal/servicio/sistema_darwin.go`](../../internal/servicio/sistema_darwin.go) crea la carpeta padre con 0755 (de root; solo contiene `config.toml`) y la repara si una instalación anterior la dejó en 0750. La carpeta de datos sigue en 0750 para `_prisma`.
- **✅ Confirmado en el CI** (ejecución `39671479591`): el LaunchDaemon queda `state = running` con el usuario `_prisma` y el panel responde 200.

### C2. Tickets perdidos en silencio después de un reinicio con trabajos en cola

- **Escenario:** un corte de luz (o un reinicio) con tickets esperando en la cola. Al volver, la aplicación reintenta el **mismo** documento, como indica el contrato.
- **Qué pasaba:** al arrancar, el puente marca esos trabajos como fallidos (`PUENTE_REINICIADO`) pero **no liberaba su `id`**. El reintento recibía:

  ```json
  {"ok": true, "duplicado": true, "estado": "fallido", …}
  ```

  El SDK lo trata como éxito, la aplicación cree que imprimió y el ticket nunca sale. Por ejemplo, una comanda de cocina.
- **Evidencia, de punta a punta en Linux** (5 tickets, el primero imprimiéndose al cortar):

  | Binario | Reintentos tras reiniciar | Tickets impresos |
  |---|---|---|
  | Antes (`9872a39`) | 4 × `200 ok, duplicado, estado=fallido` | **1 de 5** |
  | Corregido | 4 × `200` impresos | **5 de 5** |

- **Corrección:** [`MarcarInterrumpidos`](../../internal/almacen/trabajos.go) libera, en la misma transacción, los `id` de los trabajos que marca, como ya ocurría con cualquier trabajo fallido.
- **Prueba nueva:** `TestReintentoDespuesDeReiniciarConTrabajosEnCola` (`internal/servidor`).

---

## 🟠 Altos

### A1. El primer paso del inicio rápido falla hoy

Las dos URLs del SDK que da la guía responden **404**:

| URL | Estado | Por qué |
|---|---|---|
| `cdn.jsdelivr.net/npm/@bayalad7/prisma@0.1/…` | 404 | El paquete no está publicado en npm. |
| `cdn.jsdelivr.net/gh/bayalad7/prisma@main/…` | 404 | El SDK todavía no llegó a `main`. |
| `cdn.jsdelivr.net/gh/bayalad7/prisma@develop/…` | 200 | No aparece en la documentación. |

Quien sigue la guía obtiene `Prisma is not defined` y el botón no hace nada, sin ningún mensaje.

**Propuesta:** se resuelve al publicar la primera versión (`main` y npm). Mientras tanto, cambiar la nota ⏳ de las guías y de los ejemplos a `@develop`.

**✅ Corregido.**
- El inicio rápido y los dos ejemplos cargan el SDK desde `cdn.jsdelivr.net/gh/bayalad7/prisma@develop/…` (responde 200).
- Las notas ⏳ de `inicio-rapido.md`, `referencia-sdk.md` y `sdk/js/README.md` explican que la dirección de npm llegará con la primera versión.
- [publicar.md](../publicar.md) tiene el paso para volver a la URL de npm cuando exista.

### A2. El certificado autofirmado vence sin ningún aviso

- **Evidencia:** escenario 6, con un certificado vencido. El puente arranca normalmente, la bitácora no menciona el vencimiento y el panel solo muestra la fecha. Todos los clientes fallan con `certificate has expired`. Desde el navegador, el SDK lo muestra como «no se pudo conectar…».
- **Impacto:** el certificado dura 825 días. Pasado ese plazo, todas las cajas dejan de imprimir el mismo día y sin explicación.
- **Propuesta:**
  - avisar en la bitácora y en *Inicio* del panel desde 60 días antes;
  - al vencer, avisar en rojo con el botón *Regenerar*;
  - opcional: regenerarlo automáticamente si ya venció. Habría que volver a instalarlo en las cajas, pero el panel lo diría.

**✅ Corregido.**
- La bitácora avisa al arrancar: `Warn` desde 60 días antes y `Error` si ya venció.
- *Inicio* del panel muestra un aviso con el botón *Ir a Ajustes*, y en *Ajustes* la fila *Vence* lo marca.
- Con `modo_tls = "autofirmado"`, un certificado vencido se regenera al arrancar.
- Pruebas: `TestVencimientoDelCertificado` y `TestAvisoDeCertificadoVencido`. Ver [tls.md](../tls.md#vencimiento).

---

## 🟡 Medios

### M1. Una impresora USB de Windows puede quedar bloqueada para siempre

- **Causa:** el conector winspool pasa `cancelar = nil` a `conPlazo`. Si `WritePrinter` se cuelga (por ejemplo, con el spooler bloqueado), el plazo de 60 s vence, pero `conPlazo` sigue esperando.
- **Consecuencias:**
  - la goroutine de esa impresora no termina, y sus trabajos se acumulan hasta `COLA_LLENA`;
  - `prisma servicio detener` no completa: la cola espera a esa goroutine.
- **Propuesta:** cancelar con `AbortPrinter(h)`, que interrumpe el trabajo en la cola de Windows.
- **✅ Corregido:**
  - al vencer el plazo, el conector llama a `AbortPrinter`;
  - además, `conPlazo` abandona tras 5 s cualquier llamada que no se desbloquee, para que la cola y la detención del servicio nunca queden colgadas;
  - prueba: `TestConPlazoAbandonaLoQueNoSeDesbloquea`.

### M2. Las solicitudes pendientes se pueden llenar y nunca se limpian

- **Causa:** el tope de 200 cuenta pendientes **y rechazadas**, y nada las borra.
- **Cómo se llena:** cualquier equipo de la red puede ocuparlas en 2 minutos con `curl -H "Origin: https://x<n>.example"`. Dentro del límite de 120 por minuto por IP, o desde varias IPs.
- **Consecuencia:** después, los orígenes nuevos ya no aparecen en *Solicitudes*, aunque el SDK le diga al integrador «ya quedó registrado como pendiente». El dueño solo puede agregarlos a mano.
- **Propuesta:**
  - en el mantenimiento horario, borrar las pendientes sin actividad en 30 días y las rechazadas sin intentos en 90;
  - al llegar al tope, reemplazar la pendiente con la última actividad más antigua;
  - avisar en el panel cuando haya más de 150.
- **✅ Corregido**, las tres propuestas. Pruebas: `TestSolicitudesLimite` y `TestSolicitudesPurgar`.

### M3. Una impresora de red que se apaga a mitad de un ticket corto se reporta como impresa

- **Evidencia:** escenario 1.
  - Si el ticket cabe en los búferes TCP del sistema (en el loopback de Windows cupieron incluso 13 MB), el puente responde `200 ok` aunque la impresora solo haya recibido 100 KB y se haya apagado.
  - Con un ticket que no cabe (en Linux), responde bien: `503 IMPRESORA_INALCANZABLE`, y el reintento imprime una vez.
- **Causa:** es propia de imprimir en «raw» por el puerto 9100. `Write` termina cuando los datos están en el búfer, no cuando la impresora los procesó. Un ticket real mide de 5 a 20 KB, así que siempre cabe.
- **Propuesta:** para las impresoras que lo soportan (casi todas las ESC/POS), consultar el estado después de enviar (`DLE EOT 4`, papel; `DLE EOT 1`, en línea) con un plazo corto. Si no hay respuesta, `IMPRESORA_INALCANZABLE` y se libera el `id`. Sería una capacidad nueva del perfil, *Confirmar impresión*. Mientras tanto, documentarlo.
- **✅ Corregido con una opción nueva del perfil, *Confirmar impresión*** (migración `0004`; casilla en el panel; `--confirmar` en la CLI):
  - Después del ticket, los conectores de red y serie envían `DLE EOT 1` y esperan un byte, hasta 10 s.
  - Sin respuesta: `TIEMPO_AGOTADO`; el trabajo falla y su `id` se libera.
  - «Fuera de línea» (tapa abierta, sin papel): **no** es un fallo, porque la impresora ya tiene el ticket y lo imprimirá al resolverlo, y un reintento lo imprimiría dos veces. El trabajo se completa con la advertencia `IMPRESORA_FUERA_DE_LINEA`.
  - Desactivada por defecto, porque algunas impresoras no responden a `DLE EOT`.
  - No aplica a USB (winspool, CUPS, `/dev/usb/lp*`), que no devuelven la respuesta.
  - La impresora virtual responde a la consulta.
- **Documentado:** [conectores.md](../conectores.md#confirmar-impresión), [problemas-comunes.md](../integradores/problemas-comunes.md#impreso-pero-no-salió) y [panel.md](../panel.md).
- **Pruebas:** `TestRedConfirmarImpresion` (responde, fuera de línea, sin respuesta, cierra), `TestConfirmarConImpresoraVirtual`, `TestAvisoDelConectorEsExito` y `TestConfirmarImpresionFueraDeLinea`.
- **Pendiente:** probar con impresoras reales (matriz R1 y R1b).

### M4. Al superar el límite por IP, el SDK diagnostica mal el problema

- **Causa:** el `429` del límite por IP sale **antes** de identificar el origen, así que no lleva cabeceras CORS. El navegador lo ve como un error de red, la petición de diagnóstico del SDK a `/v1/salud` sí responde, y el SDK concluye `ORIGEN_NO_AUTORIZADO`.
- **Consecuencia:** la persona ve «el dueño debe aprobar este origen» cuando en realidad solo envió demasiadas peticiones.
- **Propuesta:** en las respuestas `429` del límite por IP, repetir el `Origin` de la petición en `Access-Control-Allow-Origin` (y exponer `Retry-After`). Esa respuesta no contiene datos, así que no expone nada.
- **✅ Corregido.** El `429` del límite por IP lleva las cabeceras CORS del origen. Así el SDK recibe el cuerpo y muestra `DEMASIADAS_PETICIONES` en lugar de `ORIGEN_NO_AUTORIZADO`. Prueba en `TestLimitePorIPYPorCliente`.

### M5. El duplicado de un trabajo fallido responde `ok: true`

- **Situación:** con C2 corregido ya no ocurre por reinicios. Pero si `IDsRecientes.Liberar` falla (su error se ignora, ver B4), un reintento de un trabajo **fallido** recibe `ok: true, duplicado: true, estado: "fallido"`.
- **Propuesta (defensa en profundidad):** si el trabajo original falló, liberar el `id` y procesar el documento como nuevo.
- **✅ Corregido.** Prueba: `TestDuplicadoDeUnTrabajoFallidoSeImprime`.

### M6. Con la base de datos bloqueada, la respuesta es un error genérico

- **Evidencia:** escenario 4, con otro proceso que mantiene un `BEGIN EXCLUSIVE` 8 s. `/v1/imprimir` responde a los 5 s `500 ERROR_INTERNO` («Ocurrió un error interno en el puente»). No imprime nada ni queda ningún `id` reservado, y al liberarse todo vuelve a funcionar. `/v1/salud` sigue respondiendo.
- **Problema:** el mensaje no ayuda, y el SDK no reintenta un 500.
- **Propuesta:** traducir `SQLITE_BUSY` a `503 PUENTE_OCUPADO` con `Retry-After: 2` y una sugerencia («Otro programa está usando la base de datos de PRISMA; reintenta en unos segundos»).
- **✅ Corregido.**
  - `SQLITE_BUSY` y `SQLITE_LOCKED` responden `503 PUENTE_OCUPADO`, con `Retry-After: 2` y un aviso en la bitácora.
  - Documentado en `api-v1.md` y `referencia-api.md`.
  - Pruebas: `TestBaseOcupadaResponde503` y `TestEsOcupada`.

---

## 🔵 Bajos

Todos ✅ **corregidos**.

| ID | Hallazgo | Propuesta |
|---|---|---|
| B1 | `Almacen.Restaurar`: si falla el renombrado de la base y luego también la reapertura, la conexión queda en `nil` y la siguiente consulta provoca un pánico. Requiere dos fallas seguidas del sistema de archivos. | Si la reapertura falla, devolver un error que obligue a reiniciar el servicio (el puente sale con código 1 y el sistema lo reinicia). |
| B2 | Windows: `C:\ProgramData\PRISMA\logs` es legible por todos los usuarios del equipo. Contiene IPs, ids de aplicaciones y alias; no contiene secretos. | Aplicarle el mismo `icacls` que a `datos`. |
| B3 | La impresora de tipo *archivo* escribe los tickets con permisos 0644. Es para pruebas. | Usar 0640. |
| B4 | `finalizar` ignora el error de `IDsRecientes.Liberar`. | Registrarlo en la bitácora (y ver M5). |
| B5 | El CI avisa que `actions/checkout@v4` y `actions/setup-go@v5` usan Node 20, ya obsoleto en los runners. | Pasar a las versiones con Node 24. |
| B6 | Inicio rápido, pasos confusos (además de A1): ver la [sección siguiente](#experiencia-del-integrador-inicio-rápido-al-pie-de-la-letra). | Ajustar la guía. |
| B7 | En la primera conexión fallida desde el navegador (por ejemplo, un certificado no confiable), el SDK reintenta 2 veces antes de explicar: unos 4 s sin respuesta. | No reintentar cuando `causas_posibles` incluye `CERTIFICADO_NO_CONFIABLE`, o mostrar el diagnóstico desde el primer fallo. |
| B8 | `debeCambiar` falla abierto si la base da error al leer el ajuste. Requiere una sesión ya válida. | Tratar el error como «debe cambiar». |
| B9 | macOS: `prisma servicio estado` sin `sudo` decía «detenido» con el servicio en marcha. kardianos usa `launchctl list`, que sin root consulta el dominio del usuario y no ve el LaunchDaemon del sistema. Lo detectó el CI. | Sin root, consultar `launchctl print system/<label>`. ✅ Corregido (`TestLaunchdEnMarcha`); se confirma en el próximo CI. |

---

## Seguridad: revisión punto por punto

| Punto | Resultado |
|---|---|
| Orden de autorización | ✅ Coincide con el contrato (§9) y con `api-v1.md`: límite por IP → origen → llave → tamaño → JSON/versión → firma → idempotencia → permisos → esquema → perfil. Nada costoso antes de identificar al cliente. |
| CORS | ✅ Eco exacto del origen (nunca `*`), `Vary: Origin`, sin credenciales; los orígenes desconocidos no reciben cabeceras; el preflight solo responde a orígenes aprobados; la cabecera Private Network Access solo cuando se pide. ⚠️ Ver M4. `Origin: null` (iframes sandbox, `file://`) se rechaza sin registrarse. |
| Solicitudes pendientes | ⚠️ Tope de 200 filas: la base no crece sin límite. Ver M2 (limpieza). |
| Sesión del panel | ✅ Token de 32 bytes aleatorios guardado como SHA-256; cookie `HttpOnly`, `SameSite=Strict`, `Secure` con TLS y limitada a `/admin`; 30 min de inactividad y 12 h máximo; se invalida al cambiar o restablecer la contraseña; un token nuevo en cada inicio de sesión. |
| CSRF del panel | ✅ Token en `X-CSRF-Token`, comparado en tiempo constante, más `Origin` igual al del puente. CSP `default-src 'self'` y `frame-ancestors 'none'`. |
| DNS rebinding | ✅ Mitigado: el certificado no es válido para un dominio ajeno (el navegador corta la conexión), el `Origin` del atacante no está aprobado, y la cookie del panel es `SameSite=Strict` y de otro host. |
| Límites de peticiones | ✅ Cubetas por IP y por aplicación; las inactivas se borran cada minuto, así que el mapa no crece sin control. Panel: bloqueo de 5 min tras 5 fallos por IP y un solo cálculo argon2id a la vez. |
| Idempotencia tras reinicio | ✅ Los `id` viven en la base y sobreviven al reinicio. 🔴 C2, corregido. |
| Permisos de la base y del certificado | ✅ Base 0600 en carpeta 0700. Llave del certificado 0600 en carpeta 0700. Respaldos en carpeta 0750. Windows: `datos` solo para SYSTEM y Administradores. Linux: `/var/lib/prisma` 0750 del usuario `prisma`. macOS: `datos` 0750 de `_prisma`. ⚠️ B2 y B3. |
| `modo_desarrollo` en los instaladores | ✅ El `config.toml` que copian Inno Setup y el `.deb` lo tiene en `false`; el `.pkg` no instala configuración (valor de fábrica `false`). Prueba nueva `TestConfiguracionDeLosInstaladores`. Además, solo habilita el comando local `prisma imprimir`: no abre nada en la red. |
| Contenido de tickets y llaves en los logs | ✅ La bitácora de peticiones registra método, ruta (sin query), estado, duración, IP y cliente. Los errores con contenido del ticket (códigos de barras) se devuelven al cliente y no se registran. Llaves, contraseñas, códigos y firmas nunca se registran. |

---

## Resiliencia

Contra el binario real, con impresoras de red simuladas que leen a la velocidad de una impresora. Los escenarios de red se ejecutaron en Linux, porque el loopback de Windows absorbe varios MB y oculta la contrapresión.

| # | Escenario | Resultado |
|---|---|---|
| 1 | Impresora de red apagada a mitad de un ticket | Ticket que no cabe en los búferes: `503 IMPRESORA_INALCANZABLE` y el reintento imprime **una** vez. Ticket que cabe: ⚠️ se reportaba impreso (M3). Ahora, con *Confirmar impresión*, responde `TIEMPO_AGOTADO`. |
| 2 | Bluetooth fuera de alcance (la conexión sigue, pero no se reciben datos) | `504 TIEMPO_AGOTADO` a los 12,6 s (10 s de escritura más el render); el `id` se libera. El puerto serie real se cubre en la matriz (R2). |
| 3 | 10 trabajos simultáneos a una impresora | 10 × `200`, enviados de uno en uno en orden de llegada, sin repetirse. |
| 3b | 30 simultáneos (la cola admite 20) | 25 × `200` y 5 × `503 COLA_LLENA` (algunos terminaron mientras llegaban los demás). |
| 4 | Base bloqueada 8 s por otro proceso | `500 ERROR_INTERNO` a los 5 s (M6, ya corregido: ahora `503 PUENTE_OCUPADO` con `Retry-After`); `/v1/salud` sigue respondiendo; al liberarse, todo vuelve a funcionar. Nada se imprimió a medias. |
| 5 | Reinicio con trabajos en cola | 🔴 C2: corregido; 5 de 5 impresos al reintentar. |
| 6 | Certificado vencido | Arrancaba sin avisar y los clientes fallaban (A2). Ya corregido: lo regenera y lo avisa en la bitácora y el panel. |

---

## Experiencia del integrador: inicio rápido al pie de la letra

Se siguió [inicio-rapido.md](../integradores/inicio-rapido.md) con un puente real, `npx serve -l 3000` y Edge. Las acciones del dueño se hicieron con la misma API del panel. Usando `@develop` en lugar de las URLs de la guía (A1), el resultado es el esperado:

1. el primer clic muestra «El puente responde, pero no ha autorizado el origen http://localhost:3000…»;
2. la solicitud aparece en *Solicitudes*;
3. se aprueba con *Crear una aplicación nueva*, con nivel 2 preseleccionado, y el panel entrega la llave;
4. con la llave pegada, sale «Impreso: trb_…» y el ticket se imprime.

Pasos confusos (todos ✅ corregidos en la guía):

| # | Paso | Qué confunde | Propuesta |
|---|---|---|---|
| 1 | Página, `<script src>` | Las dos URLs dan 404 y no se ve ningún error en la página (A1). | Ver A1. |
| 2 | Lo que necesitas, punto 1 | Supone un puente ya instalado. Un desarrollador sin el equipo del negocio no sabe cómo conseguir uno para probar. | Enlazar [primeros-pasos.md](../primeros-pasos.md) (puente en la propia PC) e [instalacion.md](../instalacion.md). |
| 3 | Lo que necesitas, punto 1 | Menciona la impresora virtual, pero no cómo agregarla (*Red* con destino `127.0.0.1:9100`). | Agregar el comando y el paso del panel. |
| 4 | Página | El alias `"caja"` está fijo y no se explica dónde verlo. | «Usa el alias que aparece en el panel, en *Impresoras*, o lista los disponibles con `await prisma.impresoras()`». |
| 5 | Paso 2, primer clic | Si no se cumplió el punto 2 (certificado), el mensaje es otro («No se pudo conectar…») y la guía no lo anticipa. | Una línea: «Si ves *No se pudo conectar…*, falta confiar en el certificado (punto 2)». |
| 6 | Paso 3 | Si el desarrollador es también el dueño y abre el panel desde su laptop, la primera vez el panel pide un código de configuración que la guía no menciona. | Enlazar [panel.md, Primer acceso](../panel.md#primer-acceso). |
| 7 | Paso 2 | La cita del mensaje está abreviada: el real añade «Si la aplicación ya estaba aprobada, puede estar desactivada». | Copiar el mensaje completo. |

---

## Cambios hechos en esta verificación

| Archivo | Cambio |
|---|---|
| `internal/servicio/sistema_darwin.go` | C1: carpeta padre de datos con 0755. |
| `internal/almacen/trabajos.go` | C2: `MarcarInterrumpidos` libera los `id`. |
| `internal/servidor/servidor_test.go` | Prueba de C2. |
| `internal/cola/cola_test.go` | Prueba de fugas de goroutines. |
| `internal/config/config_test.go` | Prueba de la configuración de los instaladores (`modo_desarrollo = false`, HTTPS en 8443). |
| `internal/almacen/almacen_test.go`, `solicitudes_test.go` | Pruebas de M2 y M6. |
| `internal/conector/conector_test.go` | Pruebas de M1 y M3. |
| `internal/admin/admin_test.go` | Prueba de A2 en el panel. |
| A1, B6 | `docs/integradores/inicio-rapido.md`, los ejemplos, `referencia-sdk.md`, `sdk/js/README.md` y `docs/publicar.md`. |
| A2 | `internal/servidor/tls.go` y `escuchar.go`, `internal/admin/ajustes.go`, el panel (`panel.js`, `demo.js`) y `docs/tls.md`. |
| M1 | `internal/conector/usb_windows.go` (`AbortPrinter`) y `conector.go` (`conPlazo` abandona tras 5 s). |
| M2 | `internal/almacen/solicitudes.go` (reemplazo, `Purgar`, `Contar`), el mantenimiento en `cmd/prisma/ejecutar.go` y el aviso en el panel. |
| M3 | Migración `0004_confirmar_impresion.sql`; `internal/conector` (red, serie, `Aviso`); `internal/cola` (`Resultado.Aviso`); `internal/servidor/api.go`; el panel y la CLI (`--confirmar`); la impresora virtual y el decodificador (`DLE EOT`); `conectores.md`, `problemas-comunes.md`, `panel.md`, `api-v1.md`, `referencia-api.md` y `arquitectura.md`. |
| M4 | `internal/servidor/servidor.go` (`limitarIP`). |
| M5, B4 | `internal/servidor/api.go` (`liberarSiFallo`; el error de `Liberar` va a la bitácora). |
| M6 | `internal/almacen/almacen.go` (`EsOcupada`) y `servidor.ResponderError` (`PUENTE_OCUPADO`). |
| B1 | `internal/almacen/respaldo.go`. |
| B2 | `internal/servicio/sistema_windows.go` (`icacls` también en `logs`). |
| B3 | `internal/conector/archivo.go` (0640). |
| B5 | `.github/workflows/*.yml`: acciones con Node 24. |
| B7 | `sdk/js/src/prisma.mjs` y `dist/`: sin reintentos en la primera conexión fallida. |
| B8 | `internal/admin/sesiones.go`. |
| `docs/pruebas/matriz.md` | Matriz manual; R1 y R1b cubren *Confirmar impresión*. |
| `CLAUDE.md` y los README | Actualizados a la estructura y al estado actual. |
