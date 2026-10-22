---
titulo: "Integración continua y publicación"
descripcion: "Qué se comprueba en cada cambio y al publicar una versión"
audiencia: [desarrollador]
pantallas: []
secciones_manual: [operacion/actualizacion]
---
# Integración continua y publicación

¿Qué se comprueba en cada cambio y qué pasa al publicar una versión? Para quien mantiene el proyecto.

```mermaid
flowchart LR
    subgraph ci["ci.yml: push a main o develop y pull requests"]
        go["Go en 3 sistemas: vet,<br/>staticcheck, test -race"]:::prisma
        sdk["SDK JS en 3 sistemas"]:::prisma
        fir["firmadores PHP, Node.js, Python"]:::seguridad
        srvc["servicio real en 3 sistemas"]:::prisma
        dia["diagramas Mermaid"]:::datos
        go --> srvc
    end
    subgraph rel["release.yml: etiqueta v*"]
        b["binarios, Go 1.26.x"]:::prisma
        p["deb, windows y macos,<br/>cada uno instalado y probado"]:::prisma
        pub["checksums.txt firmado con<br/>PRISMA_LLAVE_RELEASES y verificado"]:::seguridad
        gh["GitHub Release"]:::externo
        npm["npm, solo con NPM_TOKEN"]:::externo
        b --> p --> pub --> gh
        pub --> npm
    end
    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
```

- `go test ./...` incluye las pruebas de la documentación (`go test ./docs`): rutas de la API, códigos de error y enlaces.
- El trabajo *diagramas* corre `scripts/validar-diagramas.sh`: si un bloque Mermaid no compila, el CI falla.
- El CI y la publicación usan el último parche de Go 1.26 (`go-version: '1.26.x'`), no el mínimo de `go.mod`.
- La publicación verifica la firma de `checksums.txt` contra la llave pública del repositorio **antes** de subir nada.

**Fuentes en el código:**

- .github/workflows/ci.yml
- .github/workflows/release.yml
- scripts/validar-diagramas.sh
- Makefile
- build.ps1

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
