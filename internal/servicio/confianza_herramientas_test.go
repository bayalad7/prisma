//go:build !windows

package servicio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	os.WriteFile(pem, []byte("-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----\n"), 0o644)
	return &lista, pem
}
