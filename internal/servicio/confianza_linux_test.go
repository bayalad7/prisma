package servicio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestConfiarCertificado (Linux): copia el certificado a ca-certificates y actualiza el almacén;
// quitar borra el archivo, y sin archivo no hace nada.
func TestConfiarCertificado(t *testing.T) {
	corridos, pem := simularHerramientas(t, nil)
	original := rutaConfianzaLinux
	rutaConfianzaLinux = filepath.Join(t.TempDir(), "prisma-puente.crt")
	t.Cleanup(func() { rutaConfianzaLinux = original })

	if err := ConfiarCertificado(pem); err != nil {
		t.Fatal(err)
	}
	if datos, _ := os.ReadFile(rutaConfianzaLinux); !strings.Contains(string(datos), "BEGIN CERTIFICATE") ||
		strings.Join(*corridos, "\n") != "update-ca-certificates" {
		t.Fatalf("comandos: %v", *corridos)
	}
	*corridos = nil
	if err := QuitarConfianza(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(rutaConfianzaLinux); err == nil || len(*corridos) != 1 {
		t.Fatalf("debía quitar el archivo y actualizar: %v", *corridos)
	}
	*corridos = nil
	if err := QuitarConfianza(); err != nil || len(*corridos) != 0 {
		t.Fatalf("sin certificado no debía hacer nada: %v %v", err, *corridos)
	}
}
