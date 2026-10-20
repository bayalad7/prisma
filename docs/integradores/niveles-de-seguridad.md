# Niveles de seguridad

El dueño del puente registra tu aplicación con uno de tres niveles. Cada uno añade una protección sobre el anterior:

| Nivel | Tu aplicación envía | Protege contra | No protege contra |
|---|---|---|---|
| **1 — Origen** | Nada extra: el navegador envía `Origin` solo. | Páginas web ajenas que intenten imprimir. | Programas fuera del navegador (curl, scripts) en la red, que pueden falsificar `Origin`. |
| **2 — Llave** *(por defecto)* | `Origin` aprobado (desde un navegador) y `Authorization: Bearer pk_…`. | Además, equipos de la red que no tienen la llave. | Quien tenga la llave. Una llave dentro de una página la puede leer cualquiera que vea el código fuente. |
| **3 — Firma** | Documentos firmados con Ed25519 por tu backend. El puente solo guarda tu llave **pública**. | Además, documentos alterados, repetidos o falsificados. Ni el navegador ni la red pueden cambiar un ticket. | Quien robe la llave privada de tu backend. |

Además, en todos los niveles:

- solo se imprime en las impresoras que el dueño permitió a tu aplicación;
- `abrir_cajon` y `raw` están deshabilitadas hasta que el dueño las habilite para ti.

## Cuál elegir

```mermaid
flowchart TD
    A[¿Tu aplicación tiene backend propio?] -->|No: solo una página o app web| B[¿La red del negocio es de confianza<br/>y el ticket no mueve dinero ni cajón?]
    A -->|Sí| C[¿El contenido del ticket debe ser<br/>imposible de alterar o repetir?<br/>pedidos pagados, folios, cajón]
    B -->|Sí: menú, comandas internas| N1[Nivel 1 — Origen]
    B -->|No, o no estás seguro| N2[Nivel 2 — Llave]
    C -->|Sí| N3[Nivel 3 — Firma]
    C -->|No| N2b[Nivel 2 — Llave<br/>desde el backend, sin Origin]
```

- **Nivel 1:** una carta digital que manda comandas a cocina, en una red sin invitados.
- **Nivel 2:** la opción por defecto. Sirve tanto en una página como en un backend o en una app de escritorio. Desde un servidor (sin navegador) basta la llave.
- **Nivel 3:** tiendas en línea y pedidos pagados. El backend decide qué se imprime y el puente puede comprobarlo.

## Cómo decide el puente

Cada petición pasa por estas comprobaciones, en orden. Se detiene en la primera que falla:

```mermaid
flowchart TD
    P[Petición a /v1/imprimir] --> O{¿Trae Origin?}
    O -->|Sí| OA{¿El origen pertenece a<br/>una aplicación activa?}
    OA -->|No| R1[403 ORIGEN_NO_AUTORIZADO<br/>y queda como solicitud pendiente]
    OA -->|Sí| L
    O -->|No: curl o un servidor| L{¿Trae llave pk_?}
    L -->|Sí| LV{¿Es válida y de<br/>esa aplicación?}
    LV -->|No| R2[401 LLAVE_INVALIDA]
    LV -->|Sí| F
    L -->|No| LN{¿La aplicación la exige?<br/>nivel 2, o nivel 3 que la exige}
    LN -->|Sí| R3[401 LLAVE_REQUERIDA]
    LN -->|No| F{¿Nivel 3?}
    F -->|Sí| FV{¿Firma válida, vigente<br/>y del mismo cliente_id?}
    FV -->|No| R4[401 FIRMA_REQUERIDA · FIRMA_INVALIDA · FIRMA_EXPIRADA]
    FV -->|Sí| I
    F -->|No| I{¿Mismo id en<br/>los últimos 10 min?}
    I -->|Mismo contenido| D[200 duplicado: true<br/>no vuelve a imprimir]
    I -->|Otro contenido| R5[409 ID_REUTILIZADO]
    I -->|No| PE{¿Impresora y operaciones<br/>permitidas?}
    PE -->|No| R6[403 IMPRESORA_NO_PERMITIDA · OPERACION_NO_PERMITIDA]
    PE -->|Sí| OK[Valida el documento, lo encola e imprime]
```

Si una petición no trae ni origen ni llave, el puente solo la acepta si trae un documento firmado: identifica a la aplicación por el `cliente_id` de la firma.

## Nivel 1 — Origen

```js
const prisma = new Prisma({ url: "https://192.168.1.10:8443" });
await prisma.imprimir("cocina", (d) => d.texto("Mesa 4: 2 tacos").cortar());
```

- El origen es esquema + nombre + puerto, **exacto**: `https://menu.mitienda.mx` y `https://www.menu.mitienda.mx` son orígenes distintos.
- Solo funciona desde un navegador. Fuera de él, el puente responde `ORIGEN_REQUERIDO`.

## Nivel 2 — Llave

```js
const prisma = new Prisma({ url: "https://192.168.1.10:8443", llave: "pk_7ax6nj4r4a_…" });
```

- La llave se muestra **una sola vez** en el panel. Si se pierde o se filtra, el dueño la regenera, y la anterior deja de funcionar en ese momento.
- Desde un navegador hacen falta el origen aprobado **y** la llave. Desde un servidor basta la llave.
- No la subas a un repositorio público. En una página web es visible para quien abra el código fuente: si eso importa, usa el nivel 3.

## Nivel 3 — Firma

```mermaid
sequenceDiagram
    participant N as Navegador
    participant B as Tu backend
    participant P as Puente PRISMA
    N->>B: productos del pedido
    B->>B: arma el ticket con sus datos<br/>y lo firma con la llave privada
    B-->>N: documento firmado (válido 90 s)
    N->>P: imprimirDocumento(firmado)
    P->>P: verifica la firma con la llave pública
    P-->>N: impreso
```

- Tu backend arma el ticket y lo firma con [un firmador de referencia](firmar-documentos.md). El navegador lo entrega sin modificarlo. Si alguien lo altera en el camino, el puente lo rechaza con `FIRMA_INVALIDA`.
- Un documento firmado vence a los 120 s como máximo, y su `id` no se puede reutilizar durante 10 minutos. Así, una firma capturada no sirve para imprimir dos veces.
- Si el navegador hace de mensajero, el origen de tu página debe estar aprobado para la aplicación de nivel 3. Si tu backend está en la red del negocio, puede enviarlo directamente, sin `Origin` ni llave.
- El dueño puede exigir además la llave `pk_` (opción *Exigir también la llave pk_ en cada petición*).

Guía completa con PHP, Node.js y Python: [firmar-documentos.md](firmar-documentos.md).
