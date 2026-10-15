package seguridad

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gowebpki/jcs"
)

// ErrFirmaInvalida: la firma no corresponde al documento con esa llave pública.
var ErrFirmaInvalida = errors.New("la firma no es válida para este documento")

// CanonicoSinFirma devuelve la forma canónica JCS (RFC 8785) del documento sin el campo
// "firma": exactamente los bytes que el cliente firma (contrato v1, §7).
func CanonicoSinFirma(crudo []byte) ([]byte, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(crudo, &obj); err != nil {
		return nil, fmt.Errorf("el documento no es un objeto JSON: %w", err)
	}
	delete(obj, "firma")
	sinFirma, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return jcs.Transform(sinFirma)
}

// HashContenido es el SHA-256 de la forma canónica sin firma; identifica el contenido de un
// documento para la idempotencia (§8).
func HashContenido(crudo []byte) ([]byte, error) {
	c, err := CanonicoSinFirma(crudo)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(c)
	return h[:], nil
}

// VerificarFirma comprueba la firma Ed25519 (base64 estándar) del documento recibido.
func VerificarFirma(crudo []byte, firma string, publica ed25519.PublicKey) error {
	if len(publica) != ed25519.PublicKeySize {
		return fmt.Errorf("la llave pública debe medir %d bytes", ed25519.PublicKeySize)
	}
	sello, err := base64.StdEncoding.Strict().DecodeString(firma)
	if err != nil || len(sello) != ed25519.SignatureSize {
		return ErrFirmaInvalida
	}
	c, err := CanonicoSinFirma(crudo)
	if err != nil {
		return err
	}
	if !ed25519.Verify(publica, c, sello) {
		return ErrFirmaInvalida
	}
	return nil
}
