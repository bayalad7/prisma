package servicio

import (
	"strings"
	"testing"
)

// TestConfiarCertificado (macOS): borra por su huella los certificados «PRISMA …» del llavero del
// sistema y después agrega el nuevo como raíz de confianza.
func TestConfiarCertificado(t *testing.T) {
	corridos, pem := simularHerramientas(t, map[string]string{"security find-certificate": "SHA-1 hash: AB12\nkeychain: x\nSHA-1 hash: CD34\n"})
	if err := ConfiarCertificado(pem); err != nil {
		t.Fatal(err)
	}
	todo := strings.Join(*corridos, "\n")
	if !strings.Contains(todo, "delete-certificate -Z AB12 "+llaveroSistema) || !strings.Contains(todo, "delete-certificate -Z CD34 "+llaveroSistema) ||
		!strings.HasSuffix(todo, "add-trusted-cert -d -r trustRoot -k "+llaveroSistema+" "+pem) {
		t.Fatalf("comandos:\n%s", todo)
	}
}
