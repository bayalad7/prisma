# Qué nivel de seguridad elegir

¿Qué nivel le conviene a mi aplicación?

```mermaid
flowchart TD
    A["¿Tu aplicación tiene backend propio?"] -->|No: solo una página o app web| B["¿La red del negocio es de confianza<br/>y el ticket no mueve dinero ni cajón?"]
    A -->|Sí| C["¿El contenido del ticket debe ser<br/>imposible de alterar o repetir?<br/>pedidos pagados, folios, cajón"]
    B -->|Sí: menú, comandas internas| N1[Nivel 1 — Origen]
    B -->|No, o no estás seguro| N2[Nivel 2 — Llave]
    C -->|Sí| N3[Nivel 3 — Firma]
    C -->|No| N2b[Nivel 2 — Llave<br/>desde el backend, sin Origin]
```

La explicación de cada rama está en [integradores/niveles-de-seguridad.md](../integradores/niveles-de-seguridad.md).

**Basado en:**

- docs/integradores/niveles-de-seguridad.md
- internal/almacen/clientes.go (`NivelOrigen`, `NivelLlave`, `NivelFirma`)
