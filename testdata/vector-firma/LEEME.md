# Vector de prueba de firma (nivel 3)

Archivos para comprobar que tu firmador produce exactamente lo que PRISMA espera. La llave es el
vector de prueba 1 de RFC 8032 §7.1: es **pública**, no la uses fuera de pruebas.

| Archivo | Contenido |
|---|---|
| `llave-privada.pem` | Llave privada Ed25519 (PKCS #8). |
| `llave-publica.pem` | Llave pública Ed25519 (PKIX), la que se registra en el puente. |
| `documento.json` | Documento antes de firmar. |
| `canonico.txt` | Forma canónica JCS (RFC 8785) del documento con `cliente_id = "cli_7Hq2MZp4XwR9"`, `emitido = "2026-10-14T18:30:00Z"` y `expira = "2026-10-14T18:31:30Z"`, sin `firma`. Son los bytes UTF-8 que se firman; el archivo no termina en salto de línea. |
| `firma.txt` | Firma Ed25519 de `canonico.txt` en base64 estándar. |
| `firmado.json` | El documento listo para enviar. |

Ed25519 es determinista: con esta llave y esa forma canónica, tu firmador debe producir
exactamente `firma.txt`. Si no coincide, compara primero tu forma canónica con `canonico.txt`.

Para firmar tus propios documentos de prueba: `prisma dev firmar documento.json --llave llave-privada.pem --cliente cli_… --nuevo-id`.
