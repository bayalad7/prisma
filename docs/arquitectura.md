# Arquitectura

## Visión general

```
 Aplicación web / backend                        Equipo del negocio
 ─────────────────────────                       ─────────────────────────────────────────────────────────────
                                 HTTPS + JSON    ┌──────────────────────────── prisma ────────────────────────────┐
  SDK JS / fetch  ─────────────────────────────▶ │ servidor /v1 ⏳                                                 │
                                                 │   │                                                            │
                                                 │   ▼                                                            │
                                                 │ contrato.Decodificar ── tamaño, JSON, versión, esquema, reglas │
                                                 │   │                                                            │
                                                 │   ▼                                                            │
                                                 │ seguridad ── origen, llave pk_ ⏳, firma Ed25519, permisos ⏳   │
                                                 │   │                                                            │
                                                 │   ▼                                                            │
                                                 │ almacen.IDsRecientes.Reservar ── idempotencia (10 min)          │
                                                 │   │                                                            │
                                                 │   ▼                                                            │
                                                 │ escpos.Renderizar(doc, perfil) ── bytes + advertencias          │
                                                 │   │                                                            │
                                                 │   ▼                                                            │
                                                 │ cola.Encolar(alias) ── una goroutine por impresora (FIFO)       │
                                                 │   │                                                            │
                                                 │   ▼                                                            │
                                                 │ conector.Enviar ── red · USB · Bluetooth · archivo ─────────────┼──▶ 🖨️
                                                 │   │                                                            │
                                                 │   ▼                                                            │
                                                 │ almacen.Trabajos ── historial (sin contenido del ticket)        │
                                                 └────────────────────────────────────────────────────────────────┘
```

Hoy `prisma imprimir` (modo desarrollo) recorre el mismo camino desde `contrato.Decodificar` en adelante. No pasa por la seguridad ni por la idempotencia, que llegarán con el servidor.

## Principios que guían el diseño

- **Operaciones, no bytes.** Las aplicaciones describen el ticket y el puente genera el ESC/POS para cada impresora. Así, el mismo documento sirve en 58 y 80 mm, en otra página de códigos o en una impresora sin QR nativo.
- **Sin estado entre operaciones.** Cada operación establece **todos** sus estilos y los restablece al terminar, de modo que ninguna depende de la anterior.
- **Alias en vez de direcciones.** Los clientes solo conocen `"caja"`; las IPs, puertos y nombres del sistema son configuración privada del puente.
- **Validar antes de imprimir.** Todo lo que puede fallar por el documento o por el perfil se detecta antes de encolar, así que **nunca sale un ticket a medias** por un error del documento.
- **Independiente y local.** Es un solo binario Go, sin CGO, con SQLite en Go puro y sin servicios externos.

## Paquetes

| Paquete | Rol | Depende de |
|---|---|---|
| `cmd/prisma` | CLI y punto de entrada | todos |
| `internal/config` | `config.toml` (arranque) | `rutas` |
| `internal/rutas` | rutas por sistema operativo | — |
| `internal/almacen` | SQLite y repositorios | `seguridad` (normalizar orígenes) |
| `internal/contrato` | tipos del documento v1 y validación | `schema` |
| `internal/escpos` | renderizador | `contrato`, `almacen` (perfil) |
| `internal/conector` | envío a la impresora y detección | `almacen` (perfil), `dev` (vista del conector de archivo) |
| `internal/cola` | filas por impresora y estado | `conector` |
| `internal/seguridad` | orígenes, contraseñas, firmas | — |
| `internal/dev` | decodificador, simulador, vistas, impresora virtual | `escpos`, `contrato`, `almacen` |
| `schema` | JSON Schema embebido | — |
| ⏳ `internal/servidor`, `internal/admin`, `internal/servicio`, `sdk/js` | API, panel, servicio, SDK | — |

## Del documento a los bytes

### 1. Validación (`internal/contrato`)

`Decodificar` aplica el orden de la especificación (§9 del contrato):

1. Tamaño del cuerpo: máximo 256 KB.
2. JSON estricto: UTF-8, sin BOM, sin claves repetidas, anidamiento limitado.
3. Versión, que se revisa antes que el esquema para dar `VERSION_NO_SOPORTADA` y no un error genérico.
4. JSON Schema 2020-12. El esquema está **embebido** desde `schema/`, así que se valida exactamente el mismo archivo que se publica.
5. Reglas que no dependen de la impresora: dígitos verificadores, PNG válido de hasta 4096 px, raw decodificable, suma de porcentajes de fila y vigencia de la firma.

El resultado son tipos Go (`*contrato.Texto`, `*contrato.Fila`, …) con los valores por defecto aplicados. Los errores son `*contrato.Error{codigo, mensaje, sugerencia, detalles[{ruta, regla, mensaje}]}`.

### 2. Renderizado (`internal/escpos`)

`Renderizar(doc, perfil, recursos)` genera los bytes y la lista de advertencias.

- **Texto:**
  - Se normaliza a NFC y se convierte a la página de códigos del perfil (CP437, CP850, CP858 o WPC1252).
  - Lo que no existe en la página se imprime sin acento o como `?`, con advertencia y nunca con error.
  - Se mide por runas, a un carácter por columna, y se parte por palabras.
- **Fila y par:** reparten las columnas del perfil con el algoritmo de §4.3 del contrato.
- **QR y códigos de barras:** se envían por comando (`GS ( k`, `GS k`). Si el perfil no los soporta, se dibujan como imagen con generadores propios.
- **Imágenes:** se decodifica el PNG, se compone sobre blanco, se reduce sin agrandar, se pasa a 1 bit con Floyd–Steinberg y se envía con `GS v 0` en bandas de 255 líneas.
- **Copias:** repiten el ticket completo.

### 3. Cola (`internal/cola`)

- **Una goroutine por impresora:** atiende en orden FIFO y de a un trabajo a la vez; las impresoras distintas trabajan en paralelo.
- **Límite:** como máximo 20 trabajos por impresora, entre los que esperan y el que se imprime; el siguiente recibe `COLA_LLENA`.
- **Resultado:** llega por un canal: ok, duración y error.
- **Estado:** la misma goroutine llama a `Probar()` cada 30 s, así que nunca prueba mientras imprime. Estados posibles: `en_linea`, `fuera_de_linea`, `ocupada`.

### 4. Conectores (`internal/conector`)

Todos implementan `Enviar(ctx, datos)` y `Probar(ctx)`. Las variantes por sistema están en archivos con build tags (`usb_windows.go`, `usb_unix.go`).

| Tipo | Implementación |
|---|---|
| red | TCP: 3 s para conectar y 10 s para escribir. Un reintento, y solo si no se envió ningún byte (repetir un envío parcial imprimiría dos veces). |
| usb (Windows) | winspool.drv: documento RAW (`OpenPrinterW` … `WritePrinter` … `ClosePrinter`). |
| usb (Linux/macOS) | `/dev/usb/lpN` directo, o `lp -d <cola> -o raw` (CUPS). |
| bluetooth | Puerto serie (`go.bug.st/serial`). |
| archivo | `<fecha>-<id>.bin` más un `.txt` de previsualización. |

Los detalles por sistema operativo están en [conectores.md](conectores.md).

## Seguridad

| Nivel | Cómo se identifica el cliente | Estado |
|---|---|---|
| 1 — Origen | Encabezado `Origin`, exacto, normalizado y sin comodines (`seguridad.NormalizarOrigen`). | Las tablas existen ⏳ falta la API |
| 2 — Llave (por defecto) | Origen + `Authorization: Bearer pk_…`. Solo se guarda el SHA-256 de la llave y un prefijo para buscarla. | Las tablas existen ⏳ falta la API |
| 3 — Firma | Origen + Ed25519 sobre la forma canónica JCS (RFC 8785) del documento sin `firma`. Se guarda solo la llave **pública**. | ✅ Verificación (`seguridad.VerificarFirma`) ⏳ falta la API |

Además:

- **Contraseña del panel:** argon2id (64 MiB, 3 pasadas). Las llaves `pk_` usan SHA-256 porque son aleatorias de 256 bits: argon2 por cada impresión sería caro y permitiría agotar la memoria del puente.
- **Operaciones sensibles:** `abrir_cajon` y `raw` están deshabilitadas por cliente hasta que el dueño las habilite.
- **Orígenes desconocidos:** quedan como solicitudes pendientes, con un tope de 200 para que nadie llene la base.
- **Bitácoras:** nunca registran el contenido de los tickets ni las llaves.

## Datos

SQLite (`prisma.db`) en modo WAL, con claves foráneas activas y tablas `STRICT`. Las migraciones están embebidas en el binario y se aplican al arrancar: cada una en su propia transacción y con su versión en `PRAGMA user_version`.

| Tabla | Contenido |
|---|---|
| `impresoras` | Alias, tipo, destino y perfil: ancho, columnas, puntos, página de códigos y capacidades. |
| `clientes` | Nivel, hash y prefijo de la llave, llave pública, operaciones sensibles y `todas_las_impresoras`. |
| `clientes_impresoras` | Impresoras permitidas, cuando un cliente no tiene acceso a todas. |
| `clientes_origenes` | Orígenes aprobados. Cada origen pertenece a un solo cliente. |
| `solicitudes_origen` | Orígenes desconocidos que intentaron imprimir (pendientes o rechazados). |
| `trabajos` | Historial: estado, código de error, advertencias y duración. **Sin** contenido del ticket. |
| `ids_recientes` | Idempotencia: id del documento, hash del contenido y trabajo original. Sobrevive a los reinicios. |
| `recursos` | Imágenes PNG guardadas en el puente. |
| `ajustes` | Contraseña del panel, retención, etc. |

## Errores

Cada error visible lleva un **código estable en MAYÚSCULAS**, un **mensaje** y una **sugerencia** en español:

- **Documento:** `contrato.Error`, con los códigos del contrato (§9), por ejemplo `FILA_NO_CABE`.
- **Impresión:** `conector.Error`, por ejemplo `IMPRESORA_INALCANZABLE` o `COLA_LLENA`.
- **Advertencias:** no impiden imprimir y quedan en el trabajo (`CARACTER_SUSTITUIDO`, `SIN_CUCHILLA`, `SIN_CAJON`, `QR_REDUCIDO`).

## Herramientas de desarrollo

`internal/dev` incluye una **impresora virtual** que interpreta ESC/POS: alineación, estilos, tamaños, raster, QR, códigos de barras, corte y cajón. Sobre ella funcionan:

- la vista en terminal y el HTML de previsualización, que salen de los bytes reales y no de una maqueta;
- el decodificador (`prisma dev decodificar`);
- la impresora de red simulada (`prisma dev impresora-virtual`);
- las pruebas golden de cada ejemplo a 58 y 80 mm.
