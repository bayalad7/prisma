package servicio

import (
	"strings"
	"testing"
)

// TestConfiarCertificado (macOS): por cada certificado «PRISMA …» del llavero del sistema quita su
// configuración de confianza y lo borra por su huella; después agrega el nuevo como raíz.
func TestConfiarCertificado(t *testing.T) {
	encontrados := "SHA-1 hash: AB12\n-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----\n" +
		"SHA-1 hash: CD34\n-----BEGIN CERTIFICATE-----\nBBBB\n-----END CERTIFICATE-----\n"
	corridos, pem := simularHerramientas(t, map[string]string{"security find-certificate": encontrados})
	if err := ConfiarCertificado(pem); err != nil {
		t.Fatal(err)
	}
	todo := strings.Join(*corridos, "\n")
	for _, h := range []string{"AB12", "CD34"} {
		quitar, borrar := strings.Index(todo, "remove-trusted-cert -d "), strings.Index(todo, "delete-certificate -Z "+h+" "+llaveroSistema)
		if quitar < 0 || borrar < 0 || !strings.Contains(todo, h+".pem") {
			t.Fatalf("debía quitar la confianza y borrar %s:\n%s", h, todo)
		}
	}
	if !strings.HasSuffix(todo, "add-trusted-cert -d -r trustRoot -k "+llaveroSistema+" "+pem) {
		t.Fatalf("comandos:\n%s", todo)
	}
}
