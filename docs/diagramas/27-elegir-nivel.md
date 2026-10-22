---
titulo: "Qué nivel de seguridad elegir"
descripcion: "Qué nivel de seguridad le conviene a una aplicación"
audiencia: [usuario, integrador]
pantallas: [aplicaciones]
secciones_manual: [integradores/niveles-de-seguridad]
---
# Qué nivel de seguridad elegir

¿Qué nivel le conviene a mi aplicación? Para integradores antes de pedirle al dueño que la registre.

```mermaid
flowchart TD
    a{"¿tu aplicación tiene backend propio?"}:::externo -- "no: solo una página o app web" --> b{"¿la red del negocio es de confianza<br/>y el ticket no mueve dinero ni cajón?"}:::externo
    a -- sí --> c{"¿el contenido del ticket debe ser<br/>imposible de alterar o repetir?<br/>pedidos pagados, folios, cajón"}:::externo
    b -- "sí: menú, comandas internas" --> n1["nivel 1: origen"]:::seguridad
    b -- "no, o no estás seguro" --> n2["nivel 2: llave"]:::seguridad
    c -- sí --> n3["nivel 3: firma"]:::seguridad
    c -- no --> n2b["nivel 2: llave,<br/>desde el backend, sin Origin"]:::seguridad
    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
```

- El nivel 2 es el de fábrica y sirve en la mayoría de los casos.
- El nivel 1 no protege contra herramientas fuera del navegador: úsalo solo en redes de confianza.
- La explicación de cada rama: [integradores/niveles-de-seguridad.md](../integradores/niveles-de-seguridad.md).

**Fuentes en el código:**

- internal/almacen/clientes.go (`NivelOrigen`, `NivelLlave`, `NivelFirma`)
- internal/seguridad/autorizacion.go
- docs/integradores/niveles-de-seguridad.md

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
