# Herramientas para integradores

| Carpeta | Qué contiene |
|---|---|
| [`js/`](js/) | SDK JavaScript (`@bayalad7/prisma`) para el navegador y Node.js 18+: arma documentos, los valida como el puente, reintenta los errores de red sin imprimir dos veces y explica los errores de conexión. |
| [`firmadores/`](firmadores/) | Firmadores de referencia del nivel 3 en PHP, Node.js y Python, probados contra la canonicalización del puente. |

Guías: [docs/integradores/](../docs/integradores/README.md). Pruebas: `make test-sdk` o `.\build.ps1 test-sdk`.
