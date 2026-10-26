package servicio

import (
	"errors"
	"io/fs"
	"os"
)

// En Linux (Debian, Ubuntu, Raspberry Pi OS), el almacén de ca-certificates. Chrome y Chromium en
// Linux usan además su propia base (NSS); la guía de HTTPS explica cómo agregarlo ahí.
var rutaConfianzaLinux = "/usr/local/share/ca-certificates/prisma-puente.crt"

func confiarEnSistema(rutaPEM string) error {
	datos, err := os.ReadFile(rutaPEM)
	if err != nil {
		return err
	}
	if err := os.WriteFile(rutaConfianzaLinux, datos, 0o644); err != nil {
		return err
	}
	return correr("update-ca-certificates")
}

func quitarDeSistema() error {
	if err := os.Remove(rutaConfianzaLinux); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	return correr("update-ca-certificates")
}
