// Package schema embebe los JSON Schema publicados, para que el puente valide exactamente
// el mismo esquema que reciben los integradores.
package schema

import _ "embed"

// DocumentoV1 es schema/documento-v1.schema.json.
//
//go:embed documento-v1.schema.json
var DocumentoV1 []byte
