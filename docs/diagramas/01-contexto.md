# Contexto del sistema

¿Quién habla con el puente y por dónde? Para cualquiera que llega a PRISMA por primera vez: dueños, instaladores, integradores y auditores.

```mermaid
flowchart LR
    nav["navegador<br/>página de la aplicación"]:::externo
    back["backend de la aplicación<br/>nivel 2 o 3"]:::externo
    duenio["dueño del negocio<br/>navegador"]:::externo
    subgraph equipo["equipo del puente (red local)"]
        servidor["internal/servidor<br/>API /v1 · HTTPS :8443"]:::prisma
        admin["internal/admin<br/>panel /admin"]:::prisma
        db[("prisma.db")]:::datos
        tls[("tls/ certificado")]:::seguridad
    end
    red["impresora de red<br/>TCP 9100"]:::externo
    usb["impresora USB"]:::externo
    bt["impresora Bluetooth"]:::externo
    gh["GitHub Releases"]:::externo

    nav -- "HTTPS + JSON, Origin, llave pk_" --> servidor
    back -- "HTTPS + JSON, llave pk_ o firma" --> servidor
    duenio -- "HTTPS, sesión y CSRF" --> admin
    servidor --- db
    servidor --- tls
    admin --- db
    servidor -- "ESC/POS" --> red & usb & bt
    servidor -. "aviso diario y prisma actualizar" .-> gh
    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
```

- **No hay servidor de certificados remoto.** El certificado lo genera el propio puente (`autofirmado`) o lo entrega el dueño en archivos (`archivos`); ver [12-modos-tls.md](12-modos-tls.md). PRISMA 0.1.0 no descarga certificados de ningún servidor.
- La única conexión saliente es a GitHub Releases: una consulta diaria desactivable y `prisma actualizar`, que el dueño ejecuta. Imprimir no depende de Internet.
- El panel y la API comparten puerto: `internal/servidor` reparte `/admin` a `internal/admin`.
- Las aplicaciones nunca hablan con las impresoras: solo conocen su alias.

**Fuentes en el código:**

- internal/servidor/servidor.go (`Handler`)
- internal/servidor/escuchar.go
- internal/admin/admin.go
- internal/conector/
- internal/actualizacion/actualizacion.go

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
