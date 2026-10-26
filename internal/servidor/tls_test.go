package servidor

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"
)

// TestCertificadoLimitadoALaRedLocal: el certificado autofirmado sirve para sus propios nombres e
// IPs, pero aunque un equipo confíe en él como raíz (prisma tls confiar), su llave no puede firmar
// un certificado válido para un sitio de Internet ni para una IP pública (ADR 0020).
func TestCertificadoLimitadoALaRedLocal(t *testing.T) {
	dir := t.TempDir()
	ca, err := GenerarAutofirmado(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !LimitadoALaRedLocal(ca) {
		t.Fatalf("faltan las restricciones de nombre: %v %v", ca.PermittedDNSDomains, ca.PermittedIPRanges)
	}
	raices := x509.NewCertPool()
	raices.AddCert(ca)
	verificar := func(c *x509.Certificate, nombre string) error {
		_, err := c.Verify(x509.VerifyOptions{Roots: raices, DNSName: nombre, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}})
		return err
	}
	for _, nombre := range append([]string{"localhost", "127.0.0.1"}, ca.DNSNames...) {
		if err := verificar(ca, nombre); err != nil {
			t.Errorf("el certificado debe servir para %s: %v", nombre, err)
		}
	}

	par, err := tls.LoadX509KeyPair(RutasAutofirmado(dir))
	if err != nil {
		t.Fatal(err)
	}
	firmar := func(dns []string, ips []net.IP) *x509.Certificate {
		llave, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		plantilla := &x509.Certificate{SerialNumber: big.NewInt(time.Now().UnixNano()), Subject: pkix.Name{CommonName: "hoja"},
			NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), DNSNames: dns, IPAddresses: ips,
			KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
		der, err := x509.CreateCertificate(rand.Reader, plantilla, ca, &llave.PublicKey, par.PrivateKey)
		if err != nil {
			t.Fatal(err)
		}
		c, _ := x509.ParseCertificate(der)
		return c
	}
	if err := verificar(firmar([]string{"caja.local"}, []net.IP{net.ParseIP("192.168.1.10")}), "caja.local"); err != nil {
		t.Errorf("dentro de la red local debe valer: %v", err)
	}
	for _, c := range []struct {
		nombre string
		hoja   *x509.Certificate
	}{
		{"banco.example.com", firmar([]string{"banco.example.com"}, nil)},
		{"8.8.8.8", firmar(nil, []net.IP{net.ParseIP("8.8.8.8")})},
	} {
		if err := verificar(c.hoja, c.nombre); err == nil {
			t.Errorf("la llave del puente no debe poder firmar %s", c.nombre)
		}
	}
}
