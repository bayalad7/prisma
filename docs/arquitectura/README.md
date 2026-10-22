# Arquitectura

Cómo está construido PRISMA por dentro. Para usarlo no hace falta leer esto; está pensado para quien desarrolla o audita el puente.

| Documento | Qué explica |
|---|---|
| [componentes.md](componentes.md) | Los paquetes, qué hace cada uno y de cuáles depende. |
| [flujo-de-impresion.md](flujo-de-impresion.md) | Validación, renderizado, cola, conectores y errores. |
| [datos.md](datos.md) | La base SQLite, sus tablas, la idempotencia y el mantenimiento. |
| [../seguridad/](../seguridad/README.md) | Modelo de seguridad, amenazas y políticas. |
| [../adr/](../adr/README.md) | Las decisiones de arquitectura y por qué se tomaron. |
| [../diagramas/](../diagramas/README.md) | Todos los diagramas. |

## Visión general

```
 Aplicación web / backend                       Equipo del negocio
 ────────────────────────                       ──────────────────────────────────────────────────────────────
                                HTTPS + JSON    ┌──────────────────────────── prisma ─────────────────────────────┐
  SDK JS / fetch / curl  ─────────────────────▶ │ servidor /v1 ── TLS, límite por IP, CORS                          │
                                                │   │                                                              │
                                                │   ▼                                                              │
                                                │ seguridad.Identificar ── 1. origen  2. llave pk_ (argon2id+caché) │
                                                │   │                                                              │
                                                │   ▼                                                              │
                                                │ contrato.LeerSobre ── tamaño, JSON, versión (sin validar todo)    │
                                                │   │                                                              │
                                                │   ▼                                                              │
                                                │ seguridad.VerificarDocumento ── 3. firma Ed25519 y vigencia       │
                                                │   │                                                              │
                                                │   ▼                                                              │
                                                │ almacen.IDsRecientes ── 4. idempotencia (10 min)                  │
                                                │   │                                                              │
                                                │   ▼                                                              │
                                                │ seguridad.VerificarPermisos ── 5. impresora y operaciones         │
                                                │   │                                                              │
                                                │   ▼                                                              │
                                                │ contrato.Decodificar ── 6. esquema y reglas                        │
                                                │   │                                                              │
                                                │   ▼                                                              │
                                                │ escpos.Renderizar(doc, perfil) ── bytes + advertencias            │
                                                │   │                                                              │
                                                │   ▼                                                              │
                                                │ cola.Encolar(alias) ── una goroutine por impresora (FIFO)         │
                                                │   │                                                              │
                                                │   ▼                                                              │
                                                │ conector.Enviar ── red · USB · Bluetooth · archivo ───────────────┼──▶ 🖨️
                                                │   │                                                              │
                                                │   ▼                                                              │
                                                │ almacen.Trabajos ── historial (sin contenido del ticket)          │
                                                └──────────────────────────────────────────────────────────────────┘
```

Lo costoso (validar el esquema completo, decodificar imágenes, renderizar) ocurre **después** de autorizar. `prisma imprimir` (modo desarrollo) recorre el mismo camino desde `contrato.Decodificar`, sin la autorización.

## Principios que guían el diseño

- **Operaciones, no bytes.** Las aplicaciones describen el ticket y el puente genera el ESC/POS para cada impresora. Así, el mismo documento sirve en 58 y 80 mm, en otra página de códigos o en una impresora sin QR nativo.
- **Sin estado entre operaciones.** Cada operación establece **todos** sus estilos y los restablece al terminar, de modo que ninguna depende de la anterior.
- **Alias en vez de direcciones.** Los clientes solo conocen `"caja"`; las IPs, puertos y nombres del sistema son configuración privada del puente.
- **Validar antes de imprimir.** Todo lo que puede fallar por el documento o por el perfil se detecta antes de encolar, así que **nunca sale un ticket a medias** por un error del documento.
- **Independiente y local.** Es un solo binario Go, sin CGO, con SQLite en Go puro y sin servicios externos.
