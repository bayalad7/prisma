package seguridad

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

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

// OpcionesFirma completa los campos del nivel 3 antes de firmar.
type OpcionesFirma struct {
	ClienteID string
	Emitido   time.Time     // cero = ahora
	Vigencia  time.Duration // expira = emitido + vigencia; 90 s por defecto (máximo 120 s)
	NuevoID   string        // si no está vacío, reemplaza el id del documento
}

// FirmarDocumento agrega cliente_id, emitido, expira y firma a un documento JSON, firmando la
// forma canónica JCS sin el campo firma. Es lo que hace el backend de un cliente de nivel 3.
func FirmarDocumento(crudo []byte, privada ed25519.PrivateKey, op OpcionesFirma) ([]byte, error) {
	if len(privada) != ed25519.PrivateKeySize {
		return nil, errors.New("la llave privada Ed25519 no es válida")
	}
	if op.Emitido.IsZero() {
		op.Emitido = time.Now()
	}
	if op.Vigencia == 0 {
		op.Vigencia = 90 * time.Second
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(crudo, &obj); err != nil {
		return nil, fmt.Errorf("el documento no es un objeto JSON: %w", err)
	}
	poner := func(clave, valor string) {
		b, _ := json.Marshal(valor)
		obj[clave] = b
	}
	poner("cliente_id", op.ClienteID)
	poner("emitido", op.Emitido.UTC().Format(time.RFC3339))
	poner("expira", op.Emitido.Add(op.Vigencia).UTC().Format(time.RFC3339))
	if op.NuevoID != "" {
		poner("id", op.NuevoID)
	}
	delete(obj, "firma")
	sinFirma, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	canonico, err := jcs.Transform(sinFirma)
	if err != nil {
		return nil, err
	}
	poner("firma", base64.StdEncoding.EncodeToString(ed25519.Sign(privada, canonico)))
	firmado, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(firmado, '\n'), nil
}

// LlavePrivadaPEM y LlavePublicaPEM codifican llaves Ed25519 en PEM (PKCS #8 y PKIX).
func LlavePrivadaPEM(k ed25519.PrivateKey) ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(k)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil
}

func LlavePublicaPEM(k ed25519.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(k)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), nil
}

// LeerLlavePrivada lee una llave privada Ed25519 en PEM (PKCS #8).
func LeerLlavePrivada(datos []byte) (ed25519.PrivateKey, error) {
	b, _ := pem.Decode(datos)
	if b == nil {
		return nil, errors.New("no es un archivo PEM")
	}
	k, err := x509.ParsePKCS8PrivateKey(b.Bytes)
	if err != nil {
		return nil, fmt.Errorf("no es una llave privada PKCS #8: %w", err)
	}
	privada, ok := k.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("la llave privada no es Ed25519")
	}
	return privada, nil
}

// LeerLlavePublica acepta una llave pública Ed25519 en PEM (PKIX) o sus 32 bytes en base64.
func LeerLlavePublica(datos []byte) (ed25519.PublicKey, error) {
	if b, _ := pem.Decode(datos); b != nil {
		k, err := x509.ParsePKIXPublicKey(b.Bytes)
		if err != nil {
			return nil, fmt.Errorf("no es una llave pública PKIX: %w", err)
		}
		publica, ok := k.(ed25519.PublicKey)
		if !ok {
			return nil, errors.New("la llave pública no es Ed25519")
		}
		return publica, nil
	}
	crudo, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(datos)))
	if err != nil || len(crudo) != ed25519.PublicKeySize {
		return nil, errors.New("la llave pública debe ser PEM o 32 bytes en base64")
	}
	return ed25519.PublicKey(crudo), nil
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

// HuellaLlavePublica identifica una llave pública para mostrarla: SHA-256 de sus 32 bytes,
// los primeros 8 en hexadecimal.
func HuellaLlavePublica(k ed25519.PublicKey) string {
	h := sha256.Sum256(k)
	partes := make([]string, 8)
	for i := range partes {
		partes[i] = fmt.Sprintf("%02X", h[i])
	}
	return "SHA-256 " + strings.Join(partes, ":") + ":…"
}
