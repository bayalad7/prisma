# Plan de documentación de PRISMA

- **Fecha:** 22 de octubre de 2026
- **Base revisada:** `develop` en `f4a2778`, igual a `main` y a la versión publicada `v0.1.0` (`c6def40`).
- **Estado:** aprobado y **ejecutado** el 22 de octubre de 2026 (ver [Ejecución](#ejecución)). Se conserva como registro de la reestructuración.
- **Criterio:** cuando la documentación y el código no coinciden, manda el código. Cada discrepancia indica el archivo y la línea, y de qué parte del código sale lo correcto.

## Resumen

La documentación está **mayormente al día con el código**. Las cifras (límites, tiempos, tamaños), las rutas de la API, las claves de `config.toml`, los comandos y opciones de la CLI, las opciones del SDK, el orden de validación y las secciones del panel coinciden con el código. Hay tres tipos de problemas:

1. **El estado de publicación quedó viejo.** Varios documentos dicen que no hay ninguna versión publicada y cargan el SDK desde la rama `develop`.
2. **Hay una funcionalidad documentada que no existe:** la negociación de versiones por `GET /v1/estado`.
3. **La organización es plana y está duplicada.** Hay dos referencias de la API, tres guías de problemas y ningún documento de seguridad, operación, decisiones (ADR) ni referencia de errores.

Dos condiciones limitan la reestructuración:

- **Enlaces dentro de lo ya publicado.** Los binarios, el panel, el SDK y los instaladores de `v0.1.0` llevan grabadas 7 rutas de `docs/` (ver [§3.3](#33-rutas-que-no-se-pueden-mover)). Moverlas sin dejar nada en su lugar rompe enlaces que ya no se pueden corregir en esa versión.
- **Licencia.** El repositorio es público y no tiene licencia, y el SDK se declara `UNLICENSED`. Antes de escribir `CONTRIBUTING.md` y de invitar a integradores, hace falta una decisión del dueño ([D-17](#2-discrepancias-entre-código-y-documentación)).

> **Nota sobre «v1».** La versión publicada es **0.1.0** (etiqueta `v0.1.0`). «v1» es la versión de la **API** (`/v1/`) y del **contrato del documento** (`"version": "1"`), y las dos son independientes de la versión del programa. La documentación nueva debe escribir «PRISMA 0.1.0», «API v1» y «documento v1» para no confundirlas.

---

## 1. Inventario de la documentación existente

Públicos:

- **Int:** integrador, quien conecta su aplicación.
- **Inst:** instalador, quien deja el puente en el equipo del negocio.
- **Adm:** administrador, el dueño que usa el panel.
- **Dev:** desarrollador de PRISMA.

Estados:

- ✅ vigente;
- ⚠️ desactualizado, porque contradice el código o el estado actual;
- 🟡 incompleto, porque es correcto pero le falta algo que existe en el código.

### Raíz y carpetas del repositorio

| Archivo | Propósito | Público | Estado | Evidencia |
|---|---|---|---|---|
| `README.md` | Presentación, inicio, comandos, estructura y hoja de ruta. | Todos | ⚠️ | Línea 21: «Estado: en desarrollo… Todavía no hay una versión publicada». Línea 234: «[ ] Primera versión publicada». Ya existe `v0.1.0`. |
| `CLAUDE.md` | Reglas del proyecto: decisiones, invariantes, convenciones y estructura. | Dev | ✅ / 🟡 | La estructura coincide con `go list ./...` y las invariantes, con las pruebas que cita. Le falta la decisión de compilar con el último parche de Go 1.26 (`ci.yml`, `release.yml`). Ver D-12. |
| `ejemplos/README.md` | Documentos válidos e inválidos (`indice.json`). | Int, Dev | ✅ | Los usan `internal/contrato` y el SDK en sus pruebas. |
| `empaquetado/README.md` | Instaladores, llave pública y `config.toml` de fábrica. | Dev | ✅ | Coincide con `empaquetado/*`. |
| `internal/README.md` | Mapa de paquetes. | Dev | ✅ | Coincide con `go list ./...`. |
| `schema/README.md` | JSON Schema embebido. | Int, Dev | ✅ | `schema/documento-v1.schema.json`. |
| `sdk/README.md`, `sdk/firmadores/README.md` | Índice del SDK y de los firmadores. | Int | ✅ | |
| `sdk/js/README.md` | README del paquete npm. | Int | ⚠️ | Líneas 23 y 27: `npm install @bayalad7/prisma`. El paquete no existe en npm (404), porque falta el secreto `NPM_TOKEN`. Ver D-7. |
| `testdata/README.md`, `testdata/vector-firma/LEEME.md` | Vector de firma público. | Int, Dev | ✅ | Lo usan las pruebas de `internal/seguridad` y de los firmadores. |

### `docs/`

| Archivo | Propósito | Público | Estado | Evidencia |
|---|---|---|---|---|
| `README.md` | Índice por tarea, estado y glosario. | Todos | ⚠️ | Línea 27: «PRISMA está en desarrollo». Línea 43: «el resto pendiente de decisión», cuando todo está corregido. Línea 44: «Primera versión… ⏳ Pendiente». |
| `primeros-pasos.md` | Ver PRISMA funcionando sin impresora, desde el código. | Dev, Int | ✅ | Los comandos coinciden con `cmd/prisma/*`. |
| `instalacion.md` | Instalación por sistema, actualización, respaldo y problemas. | Inst, Adm | ⚠️ | Línea 9: «⏳ … Estarán ahí a partir de la primera versión publicada». Los nombres de archivo de las líneas 20–24 coinciden con los del Release `v0.1.0`. |
| `panel.md` | Manual del panel. | Adm | 🟡 | Las secciones coinciden con `SECCIONES` en `internal/admin/web/panel.js:207`. En la línea 35, *Inicio* no menciona el aviso de vencimiento del certificado (`avisoCertificado` en `internal/admin/ajustes.go`). |
| `conectores.md` | Preparar impresoras de red, USB y Bluetooth por sistema. | Inst, Adm | ✅ | Coincide con `internal/conector/*`, incluidos Confirmar impresión y `AbortPrinter`. |
| `cli.md` | Referencia de la CLI. | Adm, Dev | 🟡 | Los comandos y opciones coinciden con `cmd/prisma/*.go`. En la línea 65, Windows solo restringe la carpeta de datos, pero el código también restringe la de bitácoras (`internal/servicio/sistema_windows.go`). |
| `configuracion.md` | `config.toml`. | Inst, Dev | ✅ | Las claves coinciden con las etiquetas `toml:` de `internal/config/config.go`, y los valores, con `PorDefecto()` (120/60 por minuto). |
| `tls.md` | Modos de HTTPS, confianza y vencimiento. | Adm, Int | ✅ | ECDSA P-256, 825 días y aviso a los 60 días (`internal/servidor/tls.go:93,151,247`). El ⏳ de ACME de la línea 11 es una intención futura, con la interfaz `ProveedorCertificado` ya en el código. |
| `arquitectura.md` | Flujo, paquetes, seguridad, datos y errores. | Dev | 🟡 | Ver D-10. Las dependencias de las líneas 66–67 están incompletas y falta `internal/version`. La línea 106 no menciona `Resultado.Aviso`, la 107 omite el estado `desconocido`, la 115 omite Confirmar impresión y la 144 omite la limpieza de solicitudes. |
| `desarrollo.md` | Compilar, probar y contribuir. | Dev | 🟡 | Ver D-11 y D-12. La tabla de la línea 82 no incluye `internal/admin`, `servicio`, `actualizacion`, `version` ni `empaquetado/`. La línea 9 dice «Go 1.26 o superior», pero `make deb` necesita 1.26.4. |
| `publicar.md` | Llave de las versiones, secretos y cómo publicar. | Dev | 🟡 | La línea 61 supone que el SDK llegará a npm. Puede usar `cdn.jsdelivr.net/gh/…@v0.1.0`, que ya responde 200. Ver D-6. |
| `contrato-documento-v1.md` | Especificación del documento v1. | Int, Dev | ⚠️ | La línea 574 describe `GET /v1/estado` con `versiones_documento`, que no existe. Ver D-9. El resto coincide con `schema/` e `internal/contrato`: 256 KB, 500 operaciones y 120 s de vigencia. |
| `api-v1.md` | Referencia técnica de la API pública. | Int, Dev | ✅ / 🟡 | Las 4 rutas coinciden con `internal/servidor/servidor.go` (`rutas`), y los códigos, con el código. En la línea 67, la versión del ejemplo es `0.4.0`. Duplica `integradores/referencia-api.md` (D-19). |
| `solucion-de-problemas.md` | Problemas frecuentes de todo tipo. | Adm, Inst | 🟡 | No trata `PUENTE_OCUPADO`, el certificado vencido, Confirmar impresión ni `servicio estado` en macOS. Se superpone con `integradores/problemas-comunes.md` y con `instalacion.md#problemas-comunes`. |
| `pruebas/README.md`, `pruebas/matriz.md`, `pruebas/reporte-verificacion.md` | Matriz manual y reporte previo a la publicación. | Dev | ✅ | El reporte es histórico, sobre `9872a39` y sus correcciones, y quedó cerrado. |
| `integradores/README.md` | Índice para integradores. | Int | 🟡 | La línea 27 ofrece «npm `@bayalad7/prisma`», que no existe. |
| `integradores/inicio-rapido.md` | Primer ticket en 5 minutos. | Int | ⚠️ | La línea 19 carga el SDK desde `@develop`: funciona, pero cambia con cada commit. La nota ⏳ de la línea 34 quedó vieja. Ver D-6. |
| `integradores/niveles-de-seguridad.md` | Elegir el nivel 1, 2 o 3. | Int | ✅ | Ya trae dos diagramas Mermaid, en las líneas 18 y 36. |
| `integradores/firmar-documentos.md` | Firmar en el nivel 3. | Int | 🟡 | La línea 128 importa de npm. |
| `integradores/referencia-api.md` | La API, explicada para integradores. | Int | ✅ / 🟡 | En la línea 11, la versión del ejemplo es `0.6.0`. Duplica `api-v1.md`. |
| `integradores/referencia-sdk.md` | Referencia del SDK. | Int | ⚠️ | Las opciones y métodos coinciden con `sdk/js/src/prisma.mjs:523–581`. Las líneas 10, 17 y 24–32 ofrecen npm y unpkg, que responden 404, y la nota de la línea 34 apunta a `@develop`. |
| `integradores/problemas-comunes.md` | Problemas desde el lado del integrador. | Int | ✅ | |
| `integradores/ejemplos/pagina-simple.html` | Ejemplo sin backend. | Int | ⚠️ | La línea 14 carga el SDK desde `@develop`. |
| `integradores/ejemplos/backend-php-firmado/*` | Ejemplo del nivel 3 en PHP. | Int | ⚠️ | `index.php:29` carga el SDK desde `@develop`. |

**Lo que no existe:** `CHANGELOG.md`, `CONTRIBUTING.md`, `SECURITY.md`, `LICENSE`, la especificación OpenAPI, una referencia completa de códigos de error, el modelo de amenazas, el runbook de operación, los ADR y `docs/diagramas/`. Las notas del Release `v0.1.0` las generó GitHub con la lista de PR; no hay un registro de cambios.

---

## 2. Discrepancias entre código y documentación

Ordenadas por impacto. «Código» indica de dónde sale el valor correcto.

| # | Documento | Qué dice | Qué hace el código | Código |
|---|---|---|---|---|
| D-1 | `README.md:21` | «Estado: en desarrollo… Todavía no hay una versión publicada». | Está publicada `v0.1.0`, con instaladores firmados. | Release `v0.1.0`; `git tag` |
| D-2 | `README.md:234` | «[ ] Primera versión publicada». | Ya se publicó. | Ídem |
| D-3 | `docs/README.md:27`, `:43`, `:44`, `:46` | «En desarrollo», «pendiente de decisión», «⏳ Pendiente» y «La documentación marca con ⏳…». | Todo está corregido y publicado. Solo falta npm. | `docs/pruebas/reporte-verificacion.md`; Release |
| D-4 | `docs/instalacion.md:9` | «⏳ … Estarán ahí a partir de la primera versión publicada». | Ya están. `releases/latest/download/…` redirige a `v0.1.0`. | `release.yml`; Release |
| D-5 | `integradores/inicio-rapido.md:19`, `ejemplos/pagina-simple.html:14`, `ejemplos/backend-php-firmado/index.php:29` | El SDK se carga desde `…/prisma@develop/…`. | Funciona, pero la rama cambia con cada commit. `…/prisma@v0.1.0/sdk/js/dist/prisma.umd.js` ya responde 200 y no cambia. | `sdk/js/dist/prisma.umd.js` en la etiqueta `v0.1.0` |
| D-6 | `inicio-rapido.md:34`, `referencia-sdk.md:34`, `sdk/js/README.md:19`, `publicar.md:61` | Notas ⏳: usar `@develop` y cambiar a npm cuando exista. | Conviene fijar `@v0.1.0` ya y dejar npm como opción. El paso 4 de `publicar.md` debe cubrir las dos URLs. | Ídem |
| D-7 | `sdk/js/README.md:23,27`, `referencia-sdk.md:10,17,24–32,254`, `firmar-documentos.md:128`, `integradores/README.md:27` | `npm install @bayalad7/prisma`, unpkg y jsDelivr `/npm/`. | El paquete no está en npm: el job `npm` de `release.yml` (líneas 241–245) lo omitió por falta de `NPM_TOKEN`, y el registro responde 404. | `release.yml:241–245`; `registry.npmjs.org/@bayalad7%2fprisma` |
| D-8 | `docs/README.md:44`, notas ⏳ | «SDK en npm» aparece como parte de la publicación. | Es un paso aparte, que depende de crear `NPM_TOKEN` y volver a ejecutar el job. | `release.yml` |
| D-9 | `contrato-documento-v1.md:574` | «`GET /v1/estado` anuncia `versiones_documento`… El SDK la consulta y genera la versión más alta…». | **No existe.** Las únicas rutas son `/v1/salud`, `/v1/impresoras`, `/v1/imprimir` y `/v1/previsualizar`. `/v1/salud` devuelve solo `ok`, `version` y `nombre_puente`, y el SDK no negocia versiones. Las versiones aceptadas están en `contrato.VersionesSoportadas` = `["1"]` (y duplicadas en `version.VersionesDocumento`). | `internal/servidor/servidor.go` (`rutas`); `internal/servidor/api.go:36`; `internal/contrato/validar.go:29`; `internal/version/version.go:16` |
| D-10 | `arquitectura.md:66–67`, `:106–107`, `:115`, `:144` | Dependencias de `servidor` y `admin`; resultado de la cola; estados; red; solicitudes. | `admin` también importa `actualizacion`, `cola`, `conector`, `config` y `contrato`, y `servidor` importa `almacen`, `conector`, `config` y `contrato`. Además: `internal/version` existe, `cola.Resultado` lleva `Aviso`, existe el estado `desconocido`, la red tiene Confirmar impresión (`DLE EOT`) y las solicitudes se purgan a los 30/90 días y se reemplaza la más antigua. | `go list ./...`; `internal/cola/cola.go:30–36`; `internal/conector/red.go`; `internal/almacen/solicitudes.go:17,153–154` |
| D-11 | `desarrollo.md:82–99` | Tabla «Estructura del código». | Faltan `internal/admin`, `internal/servicio`, `internal/actualizacion`, `internal/version` y `empaquetado/`. | `go list ./...`; CLAUDE.md «Estructura» |
| D-12 | `desarrollo.md:9`, `README.md:55` | «Go 1.26 o superior». | Compila con 1.26.0 (`go.mod:3`), pero `make deb` / `build.ps1 deb` usan nfpm v2.48.0, que exige **1.26.4** o superior (`Makefile:11`, `build.ps1:13`). El CI y la publicación usan el último 1.26.x (`ci.yml`, `release.yml`). | `go.mod:3`; `Makefile:11`; `.github/workflows/*.yml` |
| D-13 | `cli.md:65` | Windows: «la carpeta de datos queda solo para SYSTEM y Administradores». | También la de bitácoras. | `internal/servicio/sistema_windows.go` (`icacls` sobre `op.Datos` y `op.Logs`) |
| D-14 | `panel.md:35` | *Inicio*: impresoras, trabajos, solicitudes y aviso de versión. | También muestra el aviso de vencimiento del certificado. | `internal/admin/ajustes.go` (`avisoCertificado`, clave `certificado` de `/resumen`); `panel.js` |
| D-15 | `solucion-de-problemas.md` | No trata `PUENTE_OCUPADO`, el certificado vencido, Confirmar impresión ni `servicio estado` en macOS. | Los cuatro existen y tienen mensajes propios. | `servidor.ResponderError`; `servidor/tls.go`; `conector/conector.go`; `servicio/sistema_darwin.go` |
| D-16 | `internal/admin/web/panel.js:1182` (enlaces de *Ayuda*) y `:907` | `DOCS = 'https://github.com/bayalad7/prisma/blob/develop/docs/'`. La línea 907 remite a `api-v1.md` para firmar. | El panel de 0.1.0 muestra la documentación de `develop`, que puede describir funciones que ese puente no tiene. Para firmar, el documento adecuado es `integradores/firmar-documentos.md`. **Se corrige en el código, no en los docs:** hay que preguntar. | `panel.js:1182,1202,1216–1220,907` |
| D-17 | (falta `LICENSE`); `sdk/js/package.json:9` | El repositorio es público y los docs invitan a usar el SDK. | Sin licencia rigen todos los derechos reservados, y `UNLICENSED` lo confirma: legalmente, nadie puede usar ni contribuir. **Es una decisión del dueño.** | `package.json:9`; repositorio público (`gh api`) |
| D-18 | — | Los códigos del panel y de la CLI no están documentados. | Existen y la persona los ve: `CAMBIO_DE_CONTRASENA_REQUERIDO`, `CODIGO_INVALIDO`, `CODIGO_ANULADO`, `CODIGO_REQUERIDO`, `CONTRASENA_INCORRECTA`, `CSRF_INVALIDO`, `SESION_REQUERIDA`, `SIN_CONFIGURAR`, `YA_CONFIGURADO`, `RESPALDO_INVALIDO`, `TLS_NO_AUTOFIRMADO`, `SIN_CERTIFICADO`, `ALIAS_DUPLICADO`, `ORIGEN_DUPLICADO`, `LLAVE_PUBLICA_INVALIDA`, `IMPRESORA_INACTIVA`, `CLIENTE_NO_ENCONTRADO`, `SOLICITUD_NO_ENCONTRADA`, `NO_ENCONTRADO` y `DATOS_INVALIDOS`. Todos los de la API pública sí están documentados. | `internal/admin/*.go` |
| D-19 | `api-v1.md` ↔ `integradores/referencia-api.md`; `solucion-de-problemas.md` ↔ `integradores/problemas-comunes.md` ↔ `instalacion.md#problemas-comunes` | Dos referencias de la API y tres listas de problemas que se repiten. | Una sola fuente por tema evita que diverjan, como ya pasó con las versiones de ejemplo `0.4.0` y `0.6.0` de `api-v1.md:67` y `referencia-api.md:11`, cuando la real es `0.1.0`. | — |
| D-20 | `docs/README.md` (glosario) | El glosario está dentro del índice. | Debe ser un documento de referencia propio, con los términos «cliente» (API) / «aplicación» (panel), «perfil», «destino», «trabajo» y «advertencia». | — |

**Verificado y sin discrepancias** (para que no haga falta revisarlo de nuevo):

- **Rutas de la API:** las 4 de `/v1`.
- **API interna del panel:** las 38 rutas `/admin/api`, que `panel.md` declara internas a propósito.
- **CLI:** comandos y opciones.
- **`config.toml`** y el `config.toml` de los instaladores.
- **Cifras:**
  - 256 KB de documento y 300 KB de cuerpo; 500 operaciones;
  - imagen de 100 KB, `raw` de 64 KB y recurso de 512 KB;
  - vigencia de 120 s y tolerancia de 30 s; idempotencia de 10 min;
  - cola de 20 trabajos; 60 s por trabajo y de respuesta (202);
  - pruebas de impresora cada 30 s; 3 s para conectar y 10 s para escribir y confirmar;
  - certificado de 825 días con aviso a los 60;
  - sesión de 30 min de inactividad y 12 h como máximo; bloqueo de 5 min;
  - argon2id: 64 MiB y 3 pasadas para la contraseña, 19 MiB y 2 pasadas para las llaves;
  - retención de 30 días; 200 solicitudes y vida de 30/90 días.
- **Orden de autorización:** coincide con `servidor/api.go` (`autorizarDocumento`, `imprimir`).
- **Opciones y métodos del SDK.**
- **Tablas de la base:** coinciden con `migraciones/*.sql`.
- **Nombres de los archivos del Release y la URL de `instalar.sh`:** responde 200.
- **CLAUDE.md:** decisiones, invariantes y estructura.

---

## 3. Estructura propuesta de `docs/`

### 3.1 Árbol

Parte de tu base, con cuatro ajustes justificados por el código, que se explican en [§3.2](#32-ajustes-a-la-base-y-por-qué):

```
CHANGELOG.md                      nuevo · Keep a Changelog; 0.1.0 reconstruido de los PR #1–#4
CONTRIBUTING.md                   nuevo · ramas, commits, pruebas por commit, PR (depende de D-17)
SECURITY.md                       nuevo · cómo reportar, versiones soportadas, alcance
LICENSE                           nuevo · solo si el dueño decide la licencia (D-17)

docs/
  README.md                       índice por tipo de lector (integrador, instalador, administrador, desarrollador)
  PLAN-DOCUMENTACION.md           este plan (se archiva al terminar)

  integradores/                   ya existe: se actualiza, no se reescribe
    README.md
    inicio-rapido.md              SDK fijado a @v0.1.0 (D-5)
    niveles-de-seguridad.md       sus diagramas pasan a diagramas/ y se enlazan
    firmar-documentos.md
    sdk.md                        ← referencia-sdk.md (instalación: CDN fijado; npm cuando exista)
    problemas-comunes.md          solo problemas vistos desde la aplicación
    ejemplos/                     igual, con la URL fijada

  instalacion/
    README.md                     requisitos y elección del sistema
    windows.md                    ← instalacion.md § Windows
    linux-raspberry-pi.md         ← instalacion.md § Raspberry Pi y Linux
    macos.md                      ← instalacion.md § macOS
    instalacion-manual.md         ← instalacion.md § Instalación manual
    impresoras.md                 ← conectores.md (preparar el hardware y el sistema por tipo)
    https-y-certificados.md       ← tls.md (instalar y confiar en el certificado)
    problemas-comunes.md          ← instalacion.md § Problemas + parte de solucion-de-problemas.md

  administracion/
    README.md                     manual del panel: primer acceso y recorrido (← panel.md)
    impresoras.md                 alta, diagnóstico, páginas de códigos, Confirmar impresión
    aplicaciones-y-solicitudes.md niveles desde el lado del dueño, llaves, orígenes, limpieza
    ajustes.md                    límites, retención, red local, certificado, actualizaciones
    problemas-comunes.md          ← solucion-de-problemas.md § panel, impresión, acentos y columnas

  operacion/                      runbook
    README.md                     dónde está cada cosa por sistema (rutas, servicio, puertos)
    bitacoras.md                  formato JSON, campos, qué nunca se registra, rotación, journald y Consola
    respaldo-y-restauracion.md    panel, CLI, respaldo automático antes de migrar
    actualizacion.md              prisma actualizar, firma y SHA-256, volver atrás
    recuperacion.md               contraseña olvidada, base bloqueada, certificado vencido,
                                  falla al restaurar, trabajos interrumpidos por un corte
    monitoreo.md                  /v1/salud, estado de impresoras, señales en la bitácora

  seguridad/
    README.md                     resumen del modelo y enlaces
    modelo-de-amenazas.md         activos, actores y límites de confianza, tabla STRIDE por límite,
                                  mitigación en el código y riesgo residual (p. ej. el nivel 1 y la
                                  falsificación de Origin, ya en la matriz)
    politicas.md                  qué se guarda y qué no (sin tickets), llaves y argon2id, permisos
                                  de archivos, CORS, CSP, límites, llave de las versiones y su rotación

  arquitectura/
    README.md                     visión general y principios (← arquitectura.md § 1–2)
    componentes.md                paquetes y dependencias reales (corrige D-10)
    flujo-de-impresion.md         del documento a los bytes (← arquitectura.md § 3–4)
    datos.md                      SQLite, tablas, migraciones e idempotencia

  adr/                            una decisión por archivo: NNNN-titulo.md (formato MADR breve)
    README.md                     índice y plantilla

  diagramas/                      un diagrama Mermaid por archivo (§4)

  referencia/
    openapi.yaml                  OpenAPI 3.1 de las 4 rutas /v1; el documento hace $ref a
                                  schema/documento-v1.schema.json (3.1 admite JSON Schema 2020-12)
    api-v1.md                     ← api-v1.md + integradores/referencia-api.md (una sola fuente)
    contrato-documento-v1.md      ← contrato-documento-v1.md (sin /v1/estado: D-9)
    codigos-de-error.md           todos los códigos: API, advertencias, conectores, panel (D-18)
    cli.md                        ← cli.md
    configuracion.md              ← configuracion.md
    glosario.md                   ← glosario de docs/README.md (D-20)

  desarrollo/
    README.md                     requisitos (Go ≥ 1.26.4 para paquetes: D-12), tareas, entorno
                                  (← desarrollo.md + primeros-pasos.md)
    pruebas.md                    tipos de prueba, golden, -race, actualizar archivos generados
    publicar.md                   ← publicar.md
    pruebas/                      ← pruebas/ (matriz manual y reportes de verificación)
```

### 3.2 Ajustes a la base, y por qué

1. **Un solo lugar para los ADR (`docs/adr/`).** La base los pone en `arquitectura/` y en `adr/` a la vez. Dos lugares terminan con decisiones repartidas. `arquitectura/` enlaza a `adr/`.
2. **`instalacion/impresoras.md` y `https-y-certificados.md`.** Preparar el hardware (udev, rfcomm, colas de Windows) y confiar en el certificado se hace al instalar, en el equipo y fuera del panel. El código lo separa igual: `internal/conector/usb_*.go`, `serie.go` y los modos de `servidor/tls.go`.
3. **`desarrollo/pruebas/` conserva la matriz y los reportes.** CLAUDE.md los cita en «Comandos» como paso previo a publicar, y el reporte de `v0.1.0` es un registro histórico. Se mueve con un aviso en la ruta vieja y se actualiza la referencia en CLAUDE.md.
4. **Problemas comunes por público, no un archivo único.** Hoy los mismos síntomas están en tres archivos (D-19). Cada guía conserva solo lo que su lector puede resolver y enlaza `referencia/codigos-de-error.md` para el detalle.

### 3.3 Rutas que no se pueden mover

Estas rutas están dentro de artefactos ya publicados en `v0.1.0`:

| Ruta | Dónde aparece | Desde |
|---|---|---|
| `docs/instalacion.md` | `Documentation=` de la unidad de systemd y `AppSupportURL` del instalador de Windows | `servicio/plantillas.go:41`, `empaquetado/windows/prisma.iss:29` |
| `docs/tls.md` | Sugerencia del SDK y de `prisma tls` | `sdk/js/src/prisma.mjs:721`, `cmd/prisma/tls.go:69` |
| `docs/conectores.md` | Mensajes de error del puerto serie | `internal/conector/serie.go:109,119` |
| `docs/contrato-documento-v1.md` | Esquema, firmadores, SDK y comentarios | `schema/…json:5`, `sdk/firmadores/*`, `sdk/js` |
| `docs/api-v1.md`, `tls.md`, `conectores.md`, `contrato-documento-v1.md`, `solucion-de-problemas.md` | Enlaces de *Ayuda* del panel (rama `develop`) | `panel.js:1182,1216–1220` |
| `docs/publicar.md` | Comentario y error del workflow | `release.yml:1,35` |

**Propuesta:** cada una se queda como una página corta: «Esta guía se movió a …», con el enlace, y un resumen de tres líneas para quien llegue desde un mensaje de error. Los mensajes del código se actualizan en una versión posterior, de forma incremental y con permiso, como pide CLAUDE.md. Las páginas cortas se mantienen al menos hasta que ninguna versión soportada las cite. La alternativa es no mover esas seis rutas y reorganizar solo las demás.

### 3.4 Cómo mantener la documentación fiel al código

Lo que ya divergió una vez (D-9, D-10, D-18) se puede comprobar con pruebas. Se proponen tres; cada una es un cambio de código, así que va con aprobación aparte:

- **Rutas:** una prueba en `internal/servidor` que compare las rutas de `Handler()` con los `paths` de `docs/referencia/openapi.yaml`.
- **Códigos de error:** una prueba que recorra las constantes de error y exija que cada código aparezca en `docs/referencia/codigos-de-error.md`.
- **Enlaces:** la comprobación de enlaces y anclas de Markdown en el CI. Hoy es un script local, `enlaces.py`.

### 3.5 Orden de trabajo propuesto

Cada paso es un commit por ámbito, por `develop` y con un PR a `main`, como el resto del proyecto:

1. **Correcciones sin reestructurar:** D-1 a D-15 y D-19 (versiones de ejemplo). Es lo urgente: hoy un integrador lee que no hay versión y recibe 404 de npm.
2. **Raíz:** `CHANGELOG.md` y `SECURITY.md`. `CONTRIBUTING.md` y `LICENSE`, después de D-17.
3. **`referencia/`:** OpenAPI, códigos de error y glosario. Es la base que citan los demás documentos.
4. **`diagramas/`** (§4), antes de la arquitectura y los manuales que los incluyen.
5. **Reorganización** en las carpetas de §3.1, con las páginas cortas de §3.3 y el índice nuevo.
6. **Contenido nuevo:** `operacion/`, `seguridad/` y `adr/`.
7. **Pruebas de §3.4**, si se aprueban.

---

## 4. Diagramas para `docs/diagramas/`

Un diagrama por archivo, con un título, una línea sobre qué responde, el bloque `mermaid` y la lista de archivos de código en que se basa, para poder revisarlo cuando ese código cambie. Los textos van en español. No se usan los tipos experimentales de Mermaid (C4, architecture-beta), porque GitHub no siempre los muestra.

| # | Archivo | Tipo Mermaid | Qué responde | Basado en |
|---|---|---|---|---|
| 1 | `contexto.md` | `flowchart LR` | ¿Quién habla con el puente y por dónde? Aplicaciones (navegador o backend), el dueño (panel), impresoras (red, USB, Bluetooth), GitHub Releases (aviso y actualización). Nada más sale a Internet. | `internal/servidor/servidor.go` (`Handler`), `internal/admin/admin.go`, `internal/conector/*`, `internal/actualizacion/*` |
| 2 | `componentes.md` | `flowchart TB` | ¿Qué paquete depende de cuál? Generado desde `go list`, para corregir D-10. | `go list -f '{{.Imports}}' ./...`; `cmd/prisma`, `internal/*`, `schema` |
| 3 | `secuencia-imprimir.md` | `sequenceDiagram` | ¿Qué pasa, en orden, en `POST /v1/imprimir`? Límite por IP, identificación, cuerpo, sobre, firma, repetido, permisos, esquema, perfil, render, reserva del id, cola, conector, resultado; la respuesta 200 o 202 y la liberación del id si falla. | `internal/servidor/api.go` (`autorizarDocumento`, `imprimir`, `finalizar`, `liberarSiFallo`), `internal/seguridad/autorizacion.go`, `internal/contrato/validar.go`, `internal/escpos/renderizar.go`, `internal/cola/cola.go` |
| 4 | `autorizacion.md` | `flowchart TD` | ¿Se acepta esta petición y con qué código se rechaza? Origin, llave, firma y nivel. Reemplaza al segundo diagrama de `niveles-de-seguridad.md`, que pasará a enlazar este. | `internal/seguridad/autorizacion.go`, `internal/seguridad/firma.go`, `internal/servidor/api.go` (`identificar`) |
| 5 | `elegir-nivel.md` | `flowchart TD` | ¿Qué nivel le conviene a mi aplicación? Ya existe en `niveles-de-seguridad.md:18`; se mueve. | `docs/integradores/niveles-de-seguridad.md`; `internal/almacen/clientes.go` (niveles) |
| 6 | `estados-trabajo.md` | `stateDiagram-v2` | ¿Por qué estados pasa un trabajo? Encolado, imprimiendo, completado (con o sin advertencias) y fallido, más el caso interrumpido por un reinicio (`PUENTE_REINICIADO`). | `internal/almacen/trabajos.go` (`Finalizar`, `MarcarInterrumpidos`), `internal/servidor/api.go`, `cmd/prisma/ejecutar.go` |
| 7 | `idempotencia.md` | `flowchart TD` | ¿Qué responde el puente si llega el mismo `id`? Nuevo, duplicado con el mismo contenido, `ID_REUTILIZADO`, o un duplicado de un fallido, que se reimprime. | `internal/almacen/ids_recientes.go`, `internal/servidor/api.go` (`responderSiRepetido`, `liberarSiFallo`) |
| 8 | `estados-impresora.md` | `stateDiagram-v2` | ¿Cuándo una impresora está en línea, fuera de línea, ocupada o en estado desconocido? Incluye las pruebas cada 30 s y el efecto de un aviso. | `internal/cola/cola.go` (`cambiar`, `probar`, `imprimir`) |
| 9 | `confirmar-impresion.md` | `sequenceDiagram` | ¿Cómo confirma el puente que el ticket llegó? Envío, `DLE EOT 1` y respuesta: en línea, fuera de línea (aviso) o sin respuesta (`TIEMPO_AGOTADO`). | `internal/conector/red.go` (`confirmar`), `internal/conector/serie.go`, `internal/conector/conector.go` (`interpretarEstado`, `Aviso`), `internal/dev/virtual.go` |
| 10 | `firma-nivel3.md` | `sequenceDiagram` | ¿Quién firma qué en el nivel 3? Backend, JCS, Ed25519, navegador o servidor, puente, verificación, vigencia y anti-repetición. | `internal/seguridad/firma.go` (incluye la forma canónica JCS), `sdk/firmadores/*`, `docs/integradores/firmar-documentos.md` |
| 11 | `modelo-datos.md` | `erDiagram` | ¿Qué guarda la base y cómo se relacionan las tablas? `impresoras`, `clientes`, `clientes_impresoras`, `clientes_origenes`, `solicitudes_origen`, `trabajos`, `ids_recientes`, `recursos` y `ajustes`. | `internal/almacen/migraciones/0001–0004*.sql` |
| 12 | `solicitudes-origen.md` | `stateDiagram-v2` | ¿Qué pasa con un origen desconocido? Pendiente, aprobado (aplicación nueva o existente), rechazado u olvidado, la purga a los 30/90 días y el reemplazo al llegar al tope. | `internal/almacen/solicitudes.go`, `internal/admin/clientes.go` (solicitudes), `cmd/prisma/ejecutar.go` (`mantenimiento`) |
| 13 | `sesion-panel.md` | `sequenceDiagram` | ¿Cómo se entra al panel? Primer acceso local o con código inicial, creación de la contraseña, sesión, CSRF, bloqueo tras 5 fallos y vencimiento. | `internal/admin/sesiones.go`, `internal/admin/admin.go` (`entrar`, `configuracionInicial`, `conSesion`), `cmd/prisma/admin.go` |
| 14 | `certificado.md` | `flowchart TD` | ¿Qué hace el puente con su certificado al arrancar y mientras corre? Generar, usar el existente, avisar a los 60 días, regenerar si venció en modo autofirmado y recargar en caliente. | `internal/servidor/tls.go` (`CargarAutofirmado`, `Vencimiento`), `internal/servidor/escuchar.go`, `internal/admin/ajustes.go` |
| 15 | `actualizacion.md` | `sequenceDiagram` | ¿Cómo se actualiza el puente sin riesgo? Consulta a GitHub, descarga, firma de `checksums.txt`, SHA-256, reemplazo del binario y reinicio del servicio. | `internal/actualizacion/*`, `cmd/prisma/actualizar.go`, `empaquetado/llave-releases.pub` |
| 16 | `despliegue.md` | `flowchart LR` | ¿Cómo queda instalado en cada sistema? Gestor, usuario, rutas de configuración, datos y bitácoras, puerto y firewall. | `internal/servicio/*` (`plantillas.go`, `sistema_*.go`), `internal/rutas/*`, `empaquetado/*` |
| 17 | `publicacion.md` | `flowchart LR` | ¿Qué hace el workflow al empujar una etiqueta? Binarios, `.deb`, Windows y macOS (cada uno probado), publicación firmada y npm. | `.github/workflows/release.yml`, `Makefile`, `build.ps1` |
| 18 | `ci.md` | `flowchart LR` | ¿Qué se comprueba en cada push y PR? Go (vet, staticcheck, `-race`) en los 3 sistemas, SDK, firmadores y servicio real. | `.github/workflows/ci.yml` |
| 19 | `cors.md` | `sequenceDiagram` | ¿Por qué el navegador a veces no ve la respuesta? Preflight, eco exacto del origen, Private Network Access y 429 con CORS. Sirve a los integradores que diagnostican «Failed to fetch». | `internal/servidor/servidor.go` (`preflight`, `limitarIP`, `permitirCORS`), `sdk/js/src/prisma.mjs` (`_diagnosticarNavegador`) |

### ADR a escribir

Cada ADR documenta una decisión que **ya está tomada** en CLAUDE.md o en el código. No se inventa ninguna. Formato breve: contexto, decisión, consecuencias y evidencia en el código.

1. Go sin CGO, un binario por plataforma.
2. SQLite en Go puro (modernc) y `config.toml` solo para el arranque.
3. Operaciones JSON sin estado en lugar de bytes ESC/POS.
4. Alias de impresora en lugar de direcciones.
5. Tres niveles de seguridad por cliente, con el nivel 2 por defecto.
6. Firma Ed25519 sobre JCS (RFC 8785) para el nivel 3.
7. Llaves `pk_` con argon2id, prefijo público y caché.
8. Idempotencia persistente y liberación del `id` de lo que no se imprimió.
9. Un trabajo cuyos datos llegaron a la impresora no falla (Confirmar impresión, «fuera de línea»).
10. Certificado autofirmado de 825 días con aviso y regeneración.
11. Servicio con kardianos/service y plantillas propias.
12. Actualizaciones firmadas con una llave Ed25519 del proyecto.
13. Panel sin frameworks ni CDN, con sesiones en memoria.
14. CI y publicación con el último parche de Go 1.26.

---

## Decisiones tomadas

| # | Decisión | Opciones |
|---|---|---|
| 1 | Licencia (D-17) | Elegir una licencia abierta, por ejemplo MIT o Apache-2.0, y cambiar `package.json` · mantener «todos los derechos reservados» y decirlo en el README · decidirlo más adelante y no escribir `CONTRIBUTING.md` todavía. |
| 2 | Rutas publicadas (§3.3) | Moverlas y dejar páginas cortas · no mover esas seis. |
| 3 | URL del SDK (D-5, D-6) | Fijar `gh/…@v0.1.0` ya · esperar a npm (requiere crear `NPM_TOKEN`). |
| 4 | Enlaces del panel a `develop` (D-16) | Cambio de código en una versión futura: enlazar la documentación de la versión instalada (`blob/v<versión>/docs/`). |
| 5 | Pruebas que mantienen la documentación fiel (§3.4) | Las tres · solo enlaces · ninguna. |
| 6 | Negociación de versiones (D-9) | Quitarla del contrato · marcarla como ⏳ futura · implementarla (cambio de API, fuera de este plan). |

**Respuestas del dueño (22 de octubre de 2026):**

1. **Licencia:** Apache-2.0.
2. **Rutas publicadas:** se mueven y cada ruta vieja queda como página corta.
3. **URL del SDK:** se fija `cdn.jsdelivr.net/gh/bayalad7/prisma@v0.1.0/…`; npm queda como opción cuando exista.
4. **Enlaces del panel:** se cambian en este trabajo para que apunten a la documentación de la versión instalada, junto con las rutas que citan los mensajes de error.
5. **Pruebas de fidelidad:** las tres.
6. **Negociación de versiones:** se marca como ⏳ política futura.

---

## Ejecución

Todo lo de §3.5 quedó hecho, con estas precisiones:

- **D-1 a D-20:** corregidas. D-16 (enlaces del panel) se corrigió en el código: la *Ayuda* enlaza `blob/v<versión>/docs/` si la versión es una publicada, o `main` si no.
- **Estructura:** la de §3.1, con dos diferencias. `integradores/referencia-api.md` se fusionó en `referencia/api-v1.md` en lugar de quedar como copia, y `integradores/referencia-sdk.md` pasó a `integradores/sdk.md`. Las 7 rutas citadas por 0.1.0 quedaron como páginas cortas que redirigen.
- **Pruebas de §3.4:** las tres están en `docs/docs_test.go` (`go test ./docs`). Para la de rutas, `internal/servidor` expone `RutasPublicas()`, que sale de la misma tabla que usa `Handler()`. Se comprobó que cada prueba falla al romper lo que vigila.
- **Diagramas:** los 19, validados con `@mermaid-js/mermaid-cli` 11.4.2: los 22 bloques Mermaid del repositorio se renderizan sin errores.
- **OpenAPI:** validado con `@redocly/cli` 1.34.0, sin errores ni advertencias.

- **Diagramas, segunda versión (22 de octubre):** los 19 se reorganizaron en 27 archivos numerados (`NN-nombre.md`), con un formato común (qué responde y para quién, explicación, fuentes en el código, última verificación), colores comunes por `classDef` y un máximo de unos 15 nodos. De la lista pedida no se crearon 13 (certificado remoto), 21 (motor WASM) y 22 (integración TEDYC): no existen en el código de 0.1.0. Ver [diagramas/README.md](diagramas/README.md). `scripts/validar-diagramas.sh` (y `.ps1`) los compila y corre en el CI.

### Discrepancias encontradas al ejecutar

| # | Documento | Qué decía | Qué hace el código | Estado |
|---|---|---|---|---|
| D-21 | `arquitectura.md` (§2 del flujo) | El QR y los códigos de barras sin soporte nativo «se dibujan como imagen con generadores propios». | El QR usa la biblioteca `github.com/skip2/go-qrcode` (`internal/escpos/mapa.go`); solo los códigos de barras son propios (`internal/escpos/barras.go`). | Corregida en `arquitectura/flujo-de-impresion.md`. |
| D-22 | `openapi.yaml`, guías | Un trabajo pasa por `encolado → imprimiendo → completado`. | La cola del servicio (`cmd/prisma/ejecutar.go`) no registra `AlIniciar`, así que los trabajos de la API nunca pasan a `imprimiendo`: van de `encolado` a `completado` o `fallido`. Solo `prisma imprimir` usa `MarcarImprimiendo`. | **Pendiente de decisión** (es código): documentado como problema conocido en `CHANGELOG.md` y en `diagramas/estados-trabajo.md`. |
| D-23 | Proceso | Las comprobaciones de enlaces de esta sesión reportaron «0 fallas». | El script local revisa solo los archivos que recibe como argumentos, y se ejecutó sin argumentos: no revisó nada. La prueba nueva (`go test ./docs`), que recorre todos los Markdown, encontró 2 anclas rotas, ya corregidas. | Corregida; la prueba de Go reemplaza al script. |

### Pendiente del dueño

1. **Activar el reporte privado de vulnerabilidades** en GitHub (*Settings → Code security → Private vulnerability reporting*). `SECURITY.md` lo indica como canal, y hoy está desactivado.
2. **Revisar los plazos de respuesta** de `SECURITY.md` (3, 10 y 30 días): son una propuesta, no un compromiso que se haya acordado.
3. **D-22:** decidir si la cola del servicio debe marcar `imprimiendo`.
4. **npm:** crear `NPM_TOKEN` cuando se quiera publicar el SDK ([desarrollo/publicar.md](desarrollo/publicar.md)).

