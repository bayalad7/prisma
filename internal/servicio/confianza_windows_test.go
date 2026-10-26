package servicio

import (
	"strings"
	"testing"
)

// TestConfiarCertificado (Windows): quita del almacén raíz del equipo los certificados de PRISMA y
// después agrega el nuevo con certutil.
func TestConfiarCertificado(t *testing.T) {
	corridos, pem := simularHerramientas(t, nil)
	if err := ConfiarCertificado(pem); err != nil {
		t.Fatal(err)
	}
	c := *corridos
	if len(c) != 2 || !strings.HasPrefix(c[0], "powershell") || !strings.Contains(c[0], `Cert:\LocalMachine\Root`) ||
		!strings.Contains(c[0], OrganizacionCertificado) || c[1] != "certutil -addstore -f Root "+pem {
		t.Fatalf("comandos:\n%s", strings.Join(c, "\n"))
	}
}
