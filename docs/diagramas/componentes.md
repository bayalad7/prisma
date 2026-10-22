# Componentes

¿Qué paquete depende de cuál? Solo dependencias entre paquetes del proyecto, según `go list ./...`.

```mermaid
flowchart TB
    main["cmd/prisma"]
    admin["admin"]
    servidor["servidor"]
    seguridad["seguridad"]
    cola["cola"]
    conector["conector"]
    escpos["escpos"]
    dev["dev"]
    contrato["contrato"]
    almacen["almacen"]
    config["config"]
    rutas["rutas"]
    schema["schema"]
    servicio["servicio"]
    actualizacion["actualizacion"]
    version["version"]

    main --> admin & servidor & servicio & actualizacion & version & config
    admin --> servidor & actualizacion
    admin --> seguridad & cola & conector & escpos & dev & contrato & almacen & config
    servidor --> seguridad & cola & conector & escpos & dev & contrato & almacen & config
    seguridad --> almacen & contrato
    cola --> conector
    conector --> almacen & dev
    dev --> escpos & contrato & almacen
    escpos --> contrato & almacen
    contrato --> schema
    config --> rutas
```

`cmd/prisma` importa además, directamente, todos los paquetes de `internal/`; el diagrama muestra solo las aristas que explican la estructura. La tabla completa está en [arquitectura/componentes.md](../arquitectura/componentes.md).

**Basado en:**

- `go list -f '{{.ImportPath}} {{.Imports}}' ./...`
- cmd/prisma
- internal/*
- schema
