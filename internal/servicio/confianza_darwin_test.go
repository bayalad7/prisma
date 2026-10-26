package servicio

import (
	"strings"
	"testing"
)

const llaveroDePrueba = "SHA-1 hash: AB12\n-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----\n" +
	"SHA-1 hash: CD34\n-----BEGIN CERTIFICATE-----\nBBBB\n-----END CERTIFICATE-----\n"

// TestConfiarCertificado (macOS): confiar solo agrega (nunca quita, porque quitar pide una
// autorización que un instalador no puede dar), y no hace nada si el certificado ya está.
func TestConfiarCertificado(t *testing.T) {
	corridos, pem := simularHerramientas(t, map[string]string{"security find-certificate": llaveroDePrueba})
	if err := ConfiarCertificado(pem); err != nil {
		t.Fatal(err)
	}
	todo := strings.Join(*corridos, "\n")
	if strings.Contains(todo, "remove-trusted-cert") || strings.Contains(todo, "delete-certificate") ||
		!strings.HasSuffix(todo, "add-trusted-cert -d -r trustRoot -k "+llaveroSistema+" "+pem) {
		t.Fatalf("comandos:\n%s", todo)
	}
}

// TestConfiarCertificadoQueYaEsta (macOS): si el llavero ya tiene ese certificado (misma huella
// SHA-1 de sus bytes, aquí tres ceros), no vuelve a agregarlo.
func TestConfiarCertificadoQueYaEsta(t *testing.T) {
	corridos, pem := simularHerramientas(t, map[string]string{
		"security find-certificate": "SHA-1 hash: 29E2DCFBB16F63BB0254DF7585A15BB6FB5E927D\n-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----\n"})
	if err := ConfiarCertificado(pem); err != nil {
		t.Fatal(err)
	}
	if todo := strings.Join(*corridos, "\n"); strings.Contains(todo, "add-trusted-cert") {
		t.Fatalf("no debía volver a agregarlo:\n%s", todo)
	}
}

// TestQuitarConfianza (macOS): por cada certificado «PRISMA …» quita su configuración de confianza
// y lo borra del llavero por su huella.
func TestQuitarConfianza(t *testing.T) {
	corridos, _ := simularHerramientas(t, map[string]string{"security find-certificate": llaveroDePrueba})
	if err := QuitarConfianza(); err != nil {
		t.Fatal(err)
	}
	todo := strings.Join(*corridos, "\n")
	for _, h := range []string{"AB12", "CD34"} {
		if !strings.Contains(todo, h+".pem") || !strings.Contains(todo, "delete-certificate -Z "+h+" "+llaveroSistema) {
			t.Fatalf("debía quitar la confianza y borrar %s:\n%s", h, todo)
		}
	}
}
