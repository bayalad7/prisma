# Ejemplos del contrato «documento v1»

Documentos JSON de ejemplo. Los usan las pruebas del puente (`internal/contrato`, `internal/dev`) y del SDK (`sdk/js`), así que también funcionan como casos de prueba.

| Carpeta | Contenido |
|---|---|
| [`validos/`](validos/) | Documentos que el puente acepta: venta simple, venta con logo y QR, comanda de cocina, todas las operaciones, operaciones sensibles y un documento firmado (nivel 3). |
| [`invalidos/`](invalidos/) | Un documento por cada regla del contrato que viola. [`indice.json`](invalidos/indice.json) indica, para cada archivo, el código de error esperado, quién lo detecta (`esquema` o `puente`) y la ruta y la regla que fallan. |

Para ver cómo sale un ejemplo sin impresora:

```sh
prisma dev previsualizar ejemplos/validos/venta-logo-qr.json --ancho 58
```

Si agregas una regla al contrato, agrega aquí su ejemplo inválido y su entrada en `indice.json`: el puente **y** el SDK deben detectarla. La especificación está en [docs/referencia/contrato-documento-v1.md](../docs/referencia/contrato-documento-v1.md#11-ejemplos).
