# Contexto del sistema

¿Quién habla con el puente y por dónde? Nada más sale a Internet.

```mermaid
flowchart LR
    subgraph Red_local["Red local del negocio"]
        nav["Navegador<br/>página de la aplicación"]
        back["Backend de la aplicación<br/>nivel 2 o 3"]
        duenio["Dueño<br/>navegador"]
        subgraph Equipo["Equipo del puente"]
            prisma[["prisma<br/>API /v1 y panel /admin<br/>HTTPS :8443"]]
            db[("prisma.db")]
            prisma --- db
        end
        red_imp["Impresora de red<br/>TCP 9100"]
        usb_imp["Impresora USB<br/>winspool · CUPS · /dev/usb"]
        bt_imp["Impresora Bluetooth<br/>puerto serie"]
    end
    gh["GitHub Releases"]

    nav -- "HTTPS + JSON<br/>Origin, llave pk_" --> prisma
    back -- "HTTPS + JSON<br/>llave pk_ o firma Ed25519" --> prisma
    duenio -- "HTTPS<br/>sesión y CSRF" --> prisma
    prisma -- "ESC/POS" --> red_imp
    prisma -- "ESC/POS" --> usb_imp
    prisma -- "ESC/POS" --> bt_imp
    prisma -. "una consulta al día (desactivable)<br/>y prisma actualizar" .-> gh
```

**Basado en:**

- internal/servidor/servidor.go (`Handler`)
- internal/admin/admin.go
- internal/conector/*
- internal/actualizacion/*
