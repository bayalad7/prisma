# Integrar PRISMA en tu aplicación

PRISMA es un puente de impresión que el negocio instala en un equipo de su red local. Tu aplicación (una página web, un punto de venta, un backend) le envía el ticket como JSON, y el puente lo imprime en la impresora térmica que el negocio configuró: red, USB o Bluetooth.

Tu aplicación no conoce IPs ni modelos de impresora. Solo usa:

- la **dirección del puente**, por ejemplo `https://192.168.1.10:8443`;
- el **alias de la impresora**, por ejemplo `"caja"` o `"cocina"`;
- según el nivel de seguridad, una **llave** `pk_…` o una **llave privada** para firmar.

El dueño del negocio te entrega esos datos desde el panel de su puente.

| Si quieres… | Lee |
|---|---|
| Imprimir tu primer ticket en 5 minutos | [inicio-rapido.md](inicio-rapido.md) |
| Elegir entre origen, llave y firma | [niveles-de-seguridad.md](niveles-de-seguridad.md) |
| Conocer todos los métodos del SDK | [sdk.md](sdk.md) |
| Llamar a la API HTTP sin el SDK | [referencia/api-v1.md](../referencia/api-v1.md) · [openapi.yaml](../referencia/openapi.yaml) |
| Firmar tickets desde tu backend (PHP, Node.js, Python) | [firmar-documentos.md](firmar-documentos.md) |
| Resolver un error de conexión o de certificado | [problemas-comunes.md](problemas-comunes.md) |
| Saber qué operaciones lleva un ticket | [contrato-documento-v1.md](../referencia/contrato-documento-v1.md) |

## Piezas

| Pieza | Dónde |
|---|---|
| SDK JavaScript (navegador y Node.js 18+, sin dependencias, con tipos TypeScript) | [`sdk/js`](../../sdk/js/) · CDN: `https://cdn.jsdelivr.net/gh/bayalad7/prisma@v0.1.0/sdk/js/dist/prisma.umd.js` (npm: ⏳ próximamente) |
| Firmadores de referencia para el nivel 3 | [`sdk/firmadores`](../../sdk/firmadores/): PHP 8.2, Node.js y Python |
| Vector de prueba de firma | [`testdata/vector-firma`](../../testdata/vector-firma/LEEME.md) |
| Ejemplos | [`ejemplos/pagina-simple.html`](ejemplos/pagina-simple.html) (nivel 2) · [`ejemplos/backend-php-firmado/`](ejemplos/backend-php-firmado/) (nivel 3) |
| Esquema JSON del documento | [`schema/documento-v1.schema.json`](../../schema/documento-v1.schema.json) |
