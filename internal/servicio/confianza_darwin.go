package servicio

import (
	"bufio"
	"bytes"
	"strings"
)

// En macOS, el llavero del sistema: lo usan Safari, Chrome y Firefox (que importa las raíces del
// sistema).
const llaveroSistema = "/Library/Keychains/System.keychain"

func confiarEnSistema(rutaPEM string) error {
	return correr("security", "add-trusted-cert", "-d", "-r", "trustRoot", "-k", llaveroSistema, rutaPEM)
}

// quitarDeSistema borra por su huella SHA-1 cada certificado cuyo nombre empiece por «PRISMA »:
// así se llaman los autofirmados del puente.
func quitarDeSistema() error {
	salida, err := ejecutar("security", "find-certificate", "-a", "-c", "PRISMA ", "-Z", llaveroSistema)
	if err != nil {
		return nil // ninguno
	}
	lineas := bufio.NewScanner(bytes.NewReader(salida))
	for lineas.Scan() {
		if huella, ok := strings.CutPrefix(strings.TrimSpace(lineas.Text()), "SHA-1 hash: "); ok {
			if err := correr("security", "delete-certificate", "-Z", huella, llaveroSistema); err != nil {
				return err
			}
		}
	}
	return nil
}
