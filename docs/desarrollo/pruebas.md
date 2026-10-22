# Pruebas

```sh
go test ./...                       # todo
go test ./internal/escpos/ -run Fila -v
go test ./internal/dev -actualizar  # regenera los archivos golden
```

- **SQLite en memoria:** las pruebas de `internal/almacen` usan una base en memoria y un reloj controlado.
- **Ejemplos del contrato:** las pruebas recorren `ejemplos/validos/*.json` y `ejemplos/invalidos/indice.json`. Si agregas una regla al contrato, agrega su ejemplo inválido y su entrada en el índice: `codigo`, `detecta` (`esquema` o `puente`), `ruta` y `regla`.
- **Matriz de autorización:** `internal/servidor` prueba {nivel 1, 2, 3} × {origen válido, inválido, ausente} × {llave válida, inválida, ausente} × {firma válida, alterada, expirada, repetida} contra un oráculo escrito a partir de la especificación. Si cambias una regla de autorización, actualiza el oráculo (`esperado`) y [referencia/api-v1.md](../referencia/api-v1.md).
- **Vector de firma:** `testdata/vector-firma/` es público para los integradores; `internal/seguridad` comprueba que siga siendo coherente con el ejemplo del contrato.
- **SDK JavaScript (`sdk/js`):** `npm test` corre con `node:test`, sin instalar nada. Valida contra los mismos `ejemplos/` que el puente: si agregas una regla al contrato, el SDK debe detectarla o la prueba falla. `src/prisma.mjs` es la fuente; después de cambiarla, `npm run construir` regenera `dist/` (que se versiona para el CDN) y `npm test` falla si `dist/` quedó desactualizado.
- **Firmadores (`sdk/firmadores`):** cada uno se prueba contra `casos-jcs.json` y el vector de firma. `internal/seguridad` comprueba ese mismo `casos-jcs.json` con la canonicalización del puente. Si agregas un caso, genera su `canonico` con el firmador de Node y deja que la prueba de Go lo confirme.
- **Golden:** `internal/dev/testdata/golden/` guarda la vista y la traza de cada ejemplo válido a 58 y 80 mm. Si cambias el renderizador a propósito, regenéralos con `-actualizar` y **revisa el diff** antes de commitear: es la forma de ver qué cambió en el papel.
- **Por plataforma:** `internal/conector` tiene pruebas con build tags (`usb_windows_test.go`, `usb_unix_test.go`). `go test ./...` debe pasar en Windows, Linux y macOS.
- **Race detector:** el CI corre `go test -race ./...` en los tres sistemas, con `CGO_ENABLED=1` solo para las pruebas (los binarios publicados se compilan sin CGO). En local, en Windows sin compilador de C, córrelo en un contenedor Linux: `docker run --rm -v "$PWD:/src" -w /src golang:1.26 go test -race ./...`.
- **Documentación fiel al código:** `docs/` tiene pruebas propias (`go test ./docs`). Comprueban que las rutas de la API coincidan con [referencia/openapi.yaml](../referencia/openapi.yaml), que cada código de error del puente y del SDK esté en [referencia/codigos-de-error.md](../referencia/codigos-de-error.md), y que los enlaces y anclas de todos los Markdown del repositorio existan. Si agregas una ruta o un código, documéntalo o la prueba falla.

## Pruebas manuales

Antes de publicar una versión, la [matriz manual](pruebas/matriz.md) cubre lo que el CI no puede: impresoras reales, navegadores y teléfonos. Los resultados de cada verificación quedan en [pruebas/](pruebas/README.md).
