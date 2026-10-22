# Integración continua

¿Qué se comprueba en cada push a `main` o `develop` y en cada pull request?

```mermaid
flowchart LR
    E["push o pull request"] --> GO["Go: Windows, Ubuntu, macOS<br/>go vet · staticcheck · go test -race"]
    E --> SDK["SDK JS: Windows, Ubuntu, macOS<br/>npm test"]
    E --> FIR["Firmadores<br/>PHP · Node.js · Python"]
    GO --> SRV["Servicio real: Windows, Ubuntu, macOS<br/>instalar · panel responde · estado ·<br/>reiniciar · desinstalar"]
    GO --> B6["Ubuntu: compila las 6 plataformas"]
```

Las pruebas de `go test` incluyen las de la documentación (`go test ./docs`): rutas de la API, códigos de error y enlaces.

**Basado en:**

- .github/workflows/ci.yml
