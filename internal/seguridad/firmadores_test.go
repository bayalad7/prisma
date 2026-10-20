package seguridad

import (
	"encoding/json"
	"os"
	"testing"
)

// TestCasosJCSDeLosFirmadores comprueba que sdk/firmadores/casos-jcs.json describe la forma
// canónica que calcula el puente. Los firmadores de referencia (PHP, Node.js y Python) se prueban
// contra ese mismo archivo, así que sus firmas son las que el puente verifica.
func TestCasosJCSDeLosFirmadores(t *testing.T) {
	datos, err := os.ReadFile("../../sdk/firmadores/casos-jcs.json")
	if err != nil {
		t.Fatal(err)
	}
	var casos []struct {
		Nombre   string          `json:"nombre"`
		Entrada  json.RawMessage `json:"entrada"`
		Canonico string          `json:"canonico"`
	}
	if err := json.Unmarshal(datos, &casos); err != nil {
		t.Fatal(err)
	}
	if len(casos) < 5 {
		t.Fatalf("solo %d casos", len(casos))
	}
	for _, c := range casos {
		obtenido, err := CanonicoSinFirma(c.Entrada)
		if err != nil {
			t.Fatalf("%s: %v", c.Nombre, err)
		}
		if string(obtenido) != c.Canonico {
			t.Errorf("%s:\n puente:  %s\n archivo: %s", c.Nombre, obtenido, c.Canonico)
		}
	}
}
