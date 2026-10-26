package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bayalad7/prisma/internal/servicio"
	"github.com/bayalad7/prisma/internal/servidor"
)

// TestTLSConfiarSoloConCertificadoLimitado: «prisma tls confiar» no agrega al almacén del sistema
// un certificado que no existe ni uno anterior a 0.3.0, sin límites de red local (ADR 0020); «tls
// info» dice si lo está. (La prueba nunca llega a tocar el almacén real.)
func TestTLSConfiarSoloConCertificadoLimitado(t *testing.T) {
	ruta, dirDatos := configDePrueba(t)
	if _, err := correr(t, "tls", "confiar", "--config", ruta); err == nil || !strings.Contains(err.Error(), "se genera al primer arranque") {
		t.Fatalf("sin certificado: %v", err)
	}

	// Un certificado como los de 0.2.0: CA autofirmada sin restricciones de nombre.
	llave, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	plantilla := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "PRISMA viejo", Organization: []string{servicio.OrganizacionCertificado}},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour), IsCA: true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, DNSNames: []string{"localhost"}}
	der, _ := x509.CreateCertificate(rand.Reader, plantilla, plantilla, &llave.PublicKey, llave)
	rutaCert, _ := servidor.RutasAutofirmado(dirDatos)
	os.MkdirAll(filepath.Dir(rutaCert), 0o700)
	os.WriteFile(rutaCert, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644)
	if _, err := correr(t, "tls", "confiar", "--config", ruta); err == nil || !strings.Contains(err.Error(), "versión anterior") {
		t.Fatalf("con un certificado sin límites: %v", err)
	}
	if salida, _ := correr(t, "tls", "info", "--config", ruta); !strings.Contains(salida, "limitado a la red local: no") {
		t.Fatalf("info del viejo:\n%s", salida)
	}

	salida, err := correr(t, "tls", "regenerar", "--config", ruta)
	if err != nil || !strings.Contains(salida, "limitado a la red local: sí") || !strings.Contains(salida, "prisma tls confiar") {
		t.Fatalf("regenerar: %v\n%s", err, salida)
	}
}
