# Publicación de una versión

¿Qué hace el workflow al empujar una etiqueta `v*`?

```mermaid
flowchart LR
    T["git push de la etiqueta v*"] --> B["binarios<br/>6 plataformas, Go 1.26.x"]
    B --> DEB["deb<br/>amd64 y arm64<br/>instalación probada"]
    B --> WIN["windows<br/>instaladores Inno Setup<br/>instalación probada"]
    B --> MAC["macos<br/>.pkg universal<br/>instalación probada"]
    DEB --> P["publicar<br/>checksums.txt + firma Ed25519<br/>verificada antes de subir"]
    WIN --> P
    MAC --> P
    P --> R["GitHub Release"]
    P --> NPM["npm<br/>solo con NPM_TOKEN"]
```

El proceso para el responsable está en [desarrollo/publicar.md](../desarrollo/publicar.md).

**Basado en:**

- .github/workflows/release.yml
- Makefile
- build.ps1
- empaquetado/*
