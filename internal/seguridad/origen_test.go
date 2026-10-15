package seguridad

import (
	"errors"
	"testing"
)

func TestNormalizarOrigen(t *testing.T) {
	validos := map[string]string{
		"https://miapp.com":           "https://miapp.com",
		"HTTPS://MiApp.COM/":          "https://miapp.com",
		"https://miapp.com:443":       "https://miapp.com",
		"http://localhost:80":         "http://localhost",
		"http://localhost:5173":       "http://localhost:5173",
		"https://tienda.example:8443": "https://tienda.example:8443",
		"http://[::1]:3000":           "http://[::1]:3000",
		"http://[::1]":                "http://[::1]",
		"  https://miapp.com  ":       "https://miapp.com",
	}
	for entrada, esperado := range validos {
		obtenido, err := NormalizarOrigen(entrada)
		if err != nil || obtenido != esperado {
			t.Errorf("NormalizarOrigen(%q) = %q, %v; quiero %q", entrada, obtenido, err, esperado)
		}
	}

	invalidos := []string{
		"", "miapp.com", "ftp://miapp.com", "https://*.miapp.com", "https://miapp.com/ruta",
		"https://usuario@miapp.com", "https://miapp.com?x=1", "https://miapp.com#a", "https://", "null",
	}
	for _, entrada := range invalidos {
		if _, err := NormalizarOrigen(entrada); !errors.Is(err, ErrOrigenInvalido) {
			t.Errorf("NormalizarOrigen(%q) debería fallar, obtuve %v", entrada, err)
		}
	}
}
