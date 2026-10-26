package servicio

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

// En macOS, el llavero del sistema: lo usan Safari, Chrome y Firefox (que importa las raíces del
// sistema).
const llaveroSistema = "/Library/Keychains/System.keychain"

func confiarEnSistema(rutaPEM string) error {
	return correr("security", "add-trusted-cert", "-d", "-r", "trustRoot", "-k", llaveroSistema, rutaPEM)
}

// quitarDeSistema quita cada certificado cuyo nombre empiece por «PRISMA » (así se llaman los
// autofirmados del puente): primero su configuración de confianza de administrador, que macOS
// guarda aparte y seguiría vigente, y después el certificado del llavero.
func quitarDeSistema() error {
	salida, err := ejecutar("security", "find-certificate", "-a", "-c", "PRISMA ", "-Z", "-p", llaveroSistema)
	if err != nil {
		return nil // ninguno
	}
	dir, err := os.MkdirTemp("", "prisma-confianza-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	var huella string
	var pem []string
	lineas := bufio.NewScanner(bytes.NewReader(salida))
	for lineas.Scan() {
		linea := strings.TrimSpace(lineas.Text())
		if h, ok := strings.CutPrefix(linea, "SHA-1 hash: "); ok {
			huella = h
			continue
		}
		if linea == "-----BEGIN CERTIFICATE-----" || len(pem) > 0 {
			pem = append(pem, linea)
		}
		if linea != "-----END CERTIFICATE-----" || huella == "" {
			continue
		}
		ruta := filepath.Join(dir, huella+".pem")
		if err := os.WriteFile(ruta, []byte(strings.Join(pem, "\n")+"\n"), 0o600); err != nil {
			return err
		}
		pem = nil
		// Puede no tener configuración de confianza (otro certificado con ese nombre): se ignora.
		ejecutar("security", "remove-trusted-cert", "-d", ruta)
		if err := correr("security", "delete-certificate", "-Z", huella, llaveroSistema); err != nil {
			return err
		}
	}
	return nil
}
