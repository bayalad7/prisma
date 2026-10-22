---
titulo: "Impresora virtual"
descripcion: "La impresora virtual recibe los mismos bytes que una de papel y los muestra en el panel"
audiencia: [usuario, instalador, integrador]
pantallas: [impresoras, pruebas, rollo]
secciones_manual: [administracion/probar-sin-impresora]
---
# Impresora virtual

¿Qué recibe una impresora virtual y dónde queda lo que imprime? Para dueños, instaladores e integradores que prueban sin papel.

```mermaid
flowchart LR
    app["tu aplicación<br/>POST /v1/imprimir"]:::externo --> aut["autorización<br/>origen, llave o firma, permisos"]:::seguridad
    panel["panel: Probar impresión<br/>(sesión del dueño)"]:::externo --> val
    aut --> val["contrato v1: validar"]:::prisma
    val --> ren["escpos.Renderizar<br/>con el perfil de la impresora"]:::prisma
    ren --> bytes["bytes ESC/POS"]:::prisma
    bytes --> cola["cola de la impresora"]:::prisma
    cola -- "red, USB, Bluetooth" --> fisica["impresora de papel"]:::externo
    cola -- virtual --> sim{"simulación"}:::prisma
    sim -- "fuera de línea o falla:<br/>IMPRESORA_INALCANZABLE" --> falla["trabajo fallido:<br/>el id queda libre"]:::prisma
    sim -- "normal o lenta" --> rollo["rollo en memoria<br/>últimos 20, máx. 16 MB"]:::datos
    rollo --> ver["panel: rollo en vivo<br/>imagen a 203 ppp, .bin, comandos"]:::externo
    rollo -. "nunca" .-> nada["base, bitácora, respaldos"]:::seguridad
    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
```

- Hasta la cola, una impresora virtual y una de papel siguen el mismo camino: la misma autorización, la misma validación y el mismo renderizador con su perfil (ancho, columnas, página de códigos, capacidades). Por eso los bytes del rollo son exactamente los que recibiría una impresora real, y el `.bin` se puede revisar con `prisma dev decodificar`.
- Lo que prueba: el documento, la autorización de la aplicación, cómo queda el ticket y el manejo de errores de la aplicación. Lo que no prueba: el cable, la red, el papel o la página de códigos que la impresora física entiende de verdad.
- «Probar impresión» entra al mismo camino después de la autorización: la sesión del panel reemplaza al origen y la llave, y el trabajo queda a nombre de «Panel».
- La simulación responde con códigos que la API v1 ya informa. Un ticket que falla no llega al rollo y su `id` queda libre, como en cualquier trabajo fallido ([24-idempotencia.md](24-idempotencia.md)).
- El rollo vive solo en la memoria del puente: se borra al reiniciar o con *Vaciar rollo*, y nunca pasa por la base, la bitácora ni los respaldos (lo verifican `TestImpresoraVirtualDePrincipioAFin` y `TestProbarImpresionNoGuardaElContenido`).
- Cómo usarla, paso a paso: [administracion/probar-sin-impresora.md](../administracion/probar-sin-impresora.md).

**Fuentes en el código:**

- internal/conector/virtual.go (`Virtual`, `Rollos`, `RollosVirtuales`)
- internal/servidor/api.go (`imprimir`)
- internal/admin/pruebas.go (`imprimirDocumentoPrueba`)
- internal/admin/virtual.go (`listarRollo`, `imagenTicket`, `binTicket`, `trazaTicket`)
- internal/dev/raster.go (`Rasterizar`)
- internal/servidor/virtual_test.go (`TestImpresoraVirtualDePrincipioAFin`)

**Última verificación:** PRISMA 0.2.0 en desarrollo (rama `develop`), 22 de octubre de 2026.
