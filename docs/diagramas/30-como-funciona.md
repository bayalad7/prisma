---
titulo: "Cómo funciona PRISMA"
descripcion: "Tus aplicaciones mandan los tickets a PRISMA y PRISMA los imprime en tus impresoras"
audiencia: [usuario]
pantallas: [inicio]
secciones_manual: [administracion/conceptos, administracion/primeros-pasos]
---
# Cómo funciona PRISMA

¿Qué hace PRISMA en el negocio, sin tecnicismos? Para el dueño. La versión técnica, con los componentes y los puertos, es [01-contexto.md](01-contexto.md).

```mermaid
flowchart LR
    app["tu punto de venta,<br/>menú o página web"]:::externo -- "manda el ticket" --> prisma
    subgraph equipo["un equipo de tu negocio"]
        prisma["PRISMA<br/>revisa que tenga permiso<br/>y lo prepara para la impresora"]:::prisma
    end
    prisma -- "imprime" --> caja["impresora de la caja"]:::externo
    prisma -- "imprime" --> cocina["impresora de la cocina"]:::externo
    prisma -. "o lo muestra en pantalla" .-> virtual["impresora virtual,<br/>para probar sin papel"]:::datos
    tu["tú, en el panel"]:::seguridad -- "agregas impresoras<br/>y das permiso a las aplicaciones" --> prisma
    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
```

- PRISMA vive en un equipo de tu negocio y funciona sin Internet. Tus aplicaciones no necesitan saber cómo está conectada cada impresora: la eligen por su nombre corto (`caja`, `cocina`).
- Ninguna aplicación imprime sin tu permiso: la autorizas en *Aplicaciones* o apruebas su solicitud en *Solicitudes*.
- PRISMA no guarda lo que imprimes; solo si salió bien o qué falló.

**Fuentes en el código:**

- internal/servidor/api.go (`imprimir`)
- internal/seguridad/autorizacion.go
- internal/conector/conector.go (`Nuevo`)
- internal/admin/admin.go (`Handler`)

**Última verificación:** PRISMA 0.2.0 en desarrollo (rama `develop`), 22 de octubre de 2026.
