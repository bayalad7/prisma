package servicio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLosInstaladoresConfianEnElCertificado: cada instalador ejecuta «prisma tls confiar» después de
// iniciar el servicio, sin detenerse si falla (un certificado de 0.2.0), y cada desinstalador lo
// quita (ADR 0020).
func TestLosInstaladoresConfianEnElCertificado(t *testing.T) {
	leer := func(ruta string) string {
		t.Helper()
		datos, err := os.ReadFile(filepath.Join("..", "..", "empaquetado", ruta))
		if err != nil {
			t.Fatal(err)
		}
		return string(datos)
	}
	iss := leer("windows/prisma.iss")
	run, _, _ := strings.Cut(strings.SplitN(iss, "[Run]", 2)[1], "[UninstallRun]")
	if i, j := strings.Index(run, "servicio iniciar"), strings.Index(run, `"tls confiar --esperar 60s"`); i < 0 || j < i {
		t.Error("prisma.iss: [Run] debe ejecutar «tls confiar --esperar 60s» después de iniciar el servicio")
	}
	desinstalar := strings.SplitN(iss, "[UninstallRun]", 2)[1]
	if i, j := strings.Index(desinstalar, `"tls confiar --quitar"`), strings.Index(desinstalar, "servicio desinstalar"); i < 0 || i > j {
		t.Error("prisma.iss: [UninstallRun] debe quitar el certificado antes de desinstalar el servicio")
	}
	for ruta, linea := range map[string]string{
		"macos/scripts/postinstall":            "prisma tls confiar --esperar 60s ||",
		"linux/scripts/despues-de-instalar.sh": "prisma tls confiar --esperar 60s >/dev/null 2>&1 ||",
		"linux/scripts/antes-de-quitar.sh":     "prisma tls confiar --quitar >/dev/null 2>&1 || true",
	} {
		if !strings.Contains(leer(ruta), linea) {
			t.Errorf("%s debe tener «%s» (sin detener la instalación si falla)", ruta, linea)
		}
	}
}

// simularHerramientas reemplaza las herramientas del sistema (las pruebas no tocan el almacén real)
// y devuelve los comandos que se habrían corrido y un certificado PEM de prueba.
func simularHerramientas(t *testing.T, respuestas map[string]string) (corridos *[]string, pem string) {
	t.Helper()
	var lista []string
	original := ejecutar
	t.Cleanup(func() { ejecutar = original })
	ejecutar = func(nombre string, args ...string) ([]byte, error) {
		linea := nombre + " " + strings.Join(args, " ")
		lista = append(lista, strings.TrimSpace(linea))
		for prefijo, salida := range respuestas {
			if strings.HasPrefix(linea, prefijo) {
				return []byte(salida), nil
			}
		}
		return nil, nil
	}
	pem = filepath.Join(t.TempDir(), "certificado.pem")
	os.WriteFile(pem, []byte("-----BEGIN CERTIFICATE-----\nx\n-----END CERTIFICATE-----\n"), 0o644)
	return &lista, pem
}
