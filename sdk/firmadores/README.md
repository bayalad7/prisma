# Firmadores de referencia (nivel 3)

Firman documentos v1 con Ed25519 sobre su forma canónica JCS, exactamente como los verifica el puente. Cada uno es un solo archivo, sin más dependencias que la criptografía estándar de su lenguaje: cópialo a tu backend.

| Lenguaje | Archivo | Requisitos | Prueba |
|---|---|---|---|
| PHP | [`php/PrismaFirmador.php`](php/PrismaFirmador.php) | PHP 8.2+ con `sodium` | `php sdk/firmadores/php/prueba.php` |
| Node.js | [`node/firmador.mjs`](node/firmador.mjs) | Node.js 18+ | `node --test "sdk/firmadores/node/*.test.mjs"` |
| Python | [`python/prisma_firmador.py`](python/prisma_firmador.py) | Python 3.9+ y `cryptography` | `python -m unittest discover -s sdk/firmadores/python` |

Las pruebas se ejecutan desde la raíz del repositorio y comprueban dos cosas:

1. **[`casos-jcs.json`](casos-jcs.json):** cada firmador produce la forma canónica exacta de cada caso (escapes, orden UTF-16 de las claves, Unicode). El puente comprueba **el mismo archivo** en `internal/seguridad`, así que las tres implementaciones y el puente coinciden byte a byte.
2. **[`testdata/vector-firma`](../../testdata/vector-firma/LEEME.md):** con la llave de prueba, el documento y la hora del vector, cada firmador obtiene exactamente `canonico.txt`, `firma.txt` y `firmado.json`.

Guía de uso: [docs/integradores/firmar-documentos.md](../../docs/integradores/firmar-documentos.md).
