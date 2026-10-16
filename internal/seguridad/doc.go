// Package seguridad decide quién puede imprimir: identifica al cliente por su origen y su llave,
// verifica las firmas Ed25519 del nivel 3, aplica los permisos sobre impresoras y operaciones
// sensibles, y gestiona contraseñas y llaves (argon2id).
package seguridad
