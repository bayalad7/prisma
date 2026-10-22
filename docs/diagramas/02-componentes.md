---
titulo: "Componentes y dependencias"
descripcion: "Qué paquete del código depende de cuál"
audiencia: [desarrollador]
pantallas: []
secciones_manual: [referencia/glosario]
---
# Componentes y dependencias

¿Qué paquete depende de cuál? Para quien desarrolla o revisa el código. Solo se dibujan dependencias entre paquetes del proyecto, según `go list ./...`.

```mermaid
flowchart TB
    main["cmd/prisma"]:::prisma
    subgraph entrada["API y panel"]
        servidor["internal/servidor"]:::prisma
        admin["internal/admin"]:::prisma
    end
    seguridad["internal/seguridad"]:::seguridad
    subgraph impresion["impresión"]
        contrato["internal/contrato"]:::prisma
        escpos["internal/escpos"]:::prisma
        cola["internal/cola"]:::prisma
        conector["internal/conector"]:::prisma
        dev["internal/dev"]:::prisma
    end
    subgraph datos["datos y configuración"]
        almacen["internal/almacen"]:::datos
        config["internal/config"]:::datos
        schema["schema"]:::datos
    end
    subgraph sistema["sistema"]
        servicio["internal/servicio"]:::prisma
        actualizacion["internal/actualizacion"]:::seguridad
    end

    main --> admin & servidor & servicio & actualizacion
    admin --> servidor & actualizacion
    servidor --> seguridad & cola & escpos & contrato & almacen & config
    seguridad --> almacen & contrato
    cola --> conector
    conector --> almacen & dev
    dev --> escpos
    escpos --> contrato & almacen
    contrato --> schema
    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
```

- Para no pasar de 15 nodos se omiten `internal/rutas` (lo usa solo `internal/config`) e `internal/version` (sin dependencias; lo usa `cmd/prisma`).
- `cmd/prisma` también importa directamente casi todos los paquetes; se dibujan solo las aristas que explican la estructura.
- `internal/admin` importa además `cola`, `conector`, `config`, `contrato`, `dev`, `escpos`, `seguridad` y `almacen`, igual que `internal/servidor`; se omiten para no duplicar flechas.
- `internal/conector` depende de `internal/dev` solo por la vista previa del conector de archivo.
- La tabla completa está en [arquitectura/componentes.md](../arquitectura/componentes.md).

**Fuentes en el código:**

- `go list -f '{{.ImportPath}} {{.Imports}}' ./...`
- cmd/prisma/
- internal/
- schema/

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
