package servicio

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"
)

// TestReconoceLosCertificadosDePrisma (Windows): al confiar y al desinstalar se quitan del almacén
// raíz solo los certificados del puente (por su organización), nunca los de otros. El almacén real
// se prueba en el CI, con el servicio instalado (trabajo «Servicio»).
func TestReconoceLosCertificadosDePrisma(t *testing.T) {
	crear := func(organizacion string) []byte {
		llave, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		p := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "PRISMA equipo", Organization: []string{organizacion}},
			NotBefore: time.Now(), NotAfter: time.Now().Add(time.Hour)}
		der, _ := x509.CreateCertificate(rand.Reader, p, p, &llave.PublicKey, llave)
		return der
	}
	if !esDePrisma(crear(OrganizacionCertificado)) {
		t.Error("debe reconocer el certificado del puente")
	}
	if esDePrisma(crear("Otra empresa")) || esDePrisma([]byte("no es un certificado")) {
		t.Error("no debe tocar certificados de otros")
	}
}
