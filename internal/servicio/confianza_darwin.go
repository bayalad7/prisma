package servicio

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// En macOS, el llavero del sistema: lo usan Safari, Chrome y Firefox (que importa las raíces del
// sistema).
const llaveroSistema = "/Library/Keychains/System.keychain"

// confiarEnSistema agrega el certificado como raíz de confianza de administrador, si no está ya.
// No quita los anteriores: en macOS, quitar una confianza de administrador espera una autorización
// con ventana, y desde un instalador (o sin pantalla) se quedaría esperando (ADR 0020).
func confiarEnSistema(rutaPEM string) error {
	datos, err := os.ReadFile(rutaPEM)
	if err != nil {
		return err
	}
	bloque, _ := pem.Decode(datos)
	if bloque == nil || bloque.Type != "CERTIFICATE" {
		return fmt.Errorf("%s no es un certificado PEM", rutaPEM)
	}
	huella := fmt.Sprintf("%X", sha1.Sum(bloque.Bytes))
	for _, c := range certificadosDePrisma() {
		if c.huella == huella {
			return nil // ya está: agregarlo otra vez no cambia nada
		}
	}
	return correr("security", "add-trusted-cert", "-d", "-r", "trustRoot", "-k", llaveroSistema, rutaPEM)
}

type certificadoLlavero struct{ huella, pem string }

// certificadosDePrisma: los del llavero del sistema cuyo nombre empieza por «PRISMA » (así se llaman
// los autofirmados del puente), con su huella SHA-1 y su PEM.
func certificadosDePrisma() []certificadoLlavero {
	salida, err := ejecutar("security", "find-certificate", "-a", "-c", "PRISMA ", "-Z", "-p", llaveroSistema)
	if err != nil {
		return nil // ninguno
	}
	var encontrados []certificadoLlavero
	var actual certificadoLlavero
	var pem []string
	lineas := bufio.NewScanner(bytes.NewReader(salida))
	for lineas.Scan() {
		linea := strings.TrimSpace(lineas.Text())
		if h, ok := strings.CutPrefix(linea, "SHA-1 hash: "); ok {
			actual = certificadoLlavero{huella: strings.ToUpper(h)}
			continue
		}
		if linea == "-----BEGIN CERTIFICATE-----" || len(pem) > 0 {
			pem = append(pem, linea)
		}
		if linea == "-----END CERTIFICATE-----" && actual.huella != "" {
			actual.pem = strings.Join(pem, "\n") + "\n"
			encontrados = append(encontrados, actual)
			actual, pem = certificadoLlavero{}, nil
		}
	}
	return encontrados
}

// quitarDeSistema quita cada certificado del puente: primero su configuración de confianza de
// administrador, que macOS guarda aparte y seguiría vigente (pide autorización con una ventana), y
// después el certificado del llavero.
func quitarDeSistema() error {
	encontrados := certificadosDePrisma()
	if len(encontrados) == 0 {
		return nil
	}
	dir, err := os.MkdirTemp("", "prisma-confianza-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	for _, c := range encontrados {
		ruta := filepath.Join(dir, c.huella+".pem")
		if err := os.WriteFile(ruta, []byte(c.pem), 0o600); err != nil {
			return err
		}
		if err := correr("security", "remove-trusted-cert", "-d", ruta); err != nil {
			return fmt.Errorf("macOS pide autorizar el cambio en una ventana; ejecútalo desde la Terminal de este Mac: %w", err)
		}
		if err := correr("security", "delete-certificate", "-Z", c.huella, llaveroSistema); err != nil {
			return err
		}
	}
	return nil
}
