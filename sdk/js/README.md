# @bayalad7/prisma

SDK para imprimir tickets en un puente [PRISMA](https://github.com/bayalad7/prisma) desde el navegador o desde Node.js 18+. Sin dependencias, con tipos TypeScript.

```html
<script src="https://cdn.jsdelivr.net/gh/bayalad7/prisma@v0.1.0/sdk/js/dist/prisma.umd.js"></script>
<script>
  const prisma = new Prisma({ url: "https://192.168.1.10:8443", llave: "pk_..." });
  prisma.imprimir("caja", (doc) => doc
    .texto("MI NEGOCIO", { alineacion: "centro", negrita: true, tamano: [2, 2] })
    .separador()
    .par("2 x Taco", "$60.00")
    .par("TOTAL", "$60.00", { negrita: true })
    .qr("https://minegocio.com/p/812")
    .cortar());
</script>
```

> ⏳ El paquete todavía no está en npm. Mientras tanto, usa la dirección de arriba o copia
> `dist/prisma.mjs` a tu proyecto. Todas las formas de instalarlo están en
> [docs/integradores/sdk.md](../../docs/integradores/sdk.md#instalación).

```sh
npm install @bayalad7/prisma
```

```js
import { Prisma } from "@bayalad7/prisma";
```

## Qué hace por ti

- **Valida localmente** cada operación con las mismas reglas que el puente. Los errores llegan con el campo exacto, antes de salir de tu aplicación.
- **Genera el `id`** (UUID v4) de cada ticket y **reintenta solo los errores de red** con ese mismo `id`. El puente nunca imprime dos veces el mismo ticket.
- **Explica los errores** con un `PrismaError { codigo, mensaje, sugerencia, status }`, y distingue los problemas típicos: certificado no confiable, permiso de red local denegado, puente inalcanzable u origen pendiente de aprobar en el panel.
- **Nivel 3:** `imprimirDocumento(firmado)` entrega sin modificarlo un documento firmado por tu backend.

| Método | Para qué |
|---|---|
| `salud()` | Comprobar que el puente responde. |
| `impresoras()` | Las impresoras que tu aplicación puede usar, con su estado. |
| `imprimir(alias, doc => …)` | Armar e imprimir un ticket. |
| `imprimirDocumento(json)` | Imprimir un documento ya armado o firmado. |
| `previsualizar(alias, doc => …)` | El HTML de cómo saldrá el ticket, sin imprimir. |
| `diagnosticar()` | Explicar por qué no hay conexión, sin lanzar. |

## Documentación

- [Inicio rápido](https://github.com/bayalad7/prisma/blob/main/docs/integradores/inicio-rapido.md): tu primer ticket en 5 minutos.
- [Referencia del SDK](https://github.com/bayalad7/prisma/blob/main/docs/integradores/referencia-sdk.md): todos los métodos, opciones y códigos de error.
- [Niveles de seguridad](https://github.com/bayalad7/prisma/blob/main/docs/integradores/niveles-de-seguridad.md) y [firmar documentos](https://github.com/bayalad7/prisma/blob/main/docs/integradores/firmar-documentos.md).
- [Problemas comunes](https://github.com/bayalad7/prisma/blob/main/docs/integradores/problemas-comunes.md).

## Desarrollo

`src/prisma.mjs` es la fuente. `dist/` se genera y se versiona, para que jsDelivr también pueda servirlo desde GitHub:

```sh
npm run construir   # genera dist/: prisma.mjs, prisma.umd.js y los tipos
npm test            # comprueba que dist/ esté al día y corre las pruebas (node:test, sin instalar nada)
npm run tipos       # comprueba los tipos con TypeScript (requiere npm install)
```

Las pruebas validan contra los mismos ejemplos que el puente (`ejemplos/validos` y `ejemplos/invalidos`), así que se ejecutan dentro del repositorio.
