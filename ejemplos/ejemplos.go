// Package ejemplos embebe los documentos válidos de ejemplo, que el panel ofrece como plantillas en
// «Probar impresión» (los mismos archivos que usan las pruebas del puente y del SDK).
package ejemplos

import "embed"

// Validos contiene validos/*.json.
//
//go:embed validos/*.json
var Validos embed.FS
