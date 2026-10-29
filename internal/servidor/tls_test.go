package servidor

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"os"
	"testing"
	"time"
)

// TestAutoridadLocalYCertificadoDelServidor (ADR 0021): el puente presenta un certificado del
// servidor que no es autoridad (Firefox no acepta una autoridad como certificado del sitio), firmado
// por una autoridad local limitada a la red local: vale para los nombres e IPs del equipo, pero la
// llave de la autoridad no sirve para firmar un sitio de Internet ni una IP pública (ADR 0020).
func TestAutoridadLocalYCertificadoDelServidor(t *testing.T) {
	dir := t.TempDir()
	gen, err := GenerarAutofirmado(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	ca, hoja := gen.Autoridad, gen.Servidor
	if !gen.AutoridadNueva || !ca.IsCA || !LimitadoALaRedLocal(ca) || !ca.MaxPathLenZero {
		t.Fatalf("autoridad: nueva %v, CA %v, límites %v %v", gen.AutoridadNueva, ca.IsCA, ca.PermittedDNSDomains, ca.PermittedIPRanges)
	}
	if hoja.IsCA || hoja.CheckSignatureFrom(ca) != nil {
		t.Fatal("el certificado del servidor no debe ser autoridad y debe firmarlo la autoridad local")
	}
	if c, err := AutoridadLocal(dir); err != nil || !c.Equal(ca) {
		t.Fatalf("AutoridadLocal: %v", err)
	}

	raices := x509.NewCertPool()
	raices.AddCert(ca)
	verificar := func(c *x509.Certificate, nombre string) error {
		_, err := c.Verify(x509.VerifyOptions{Roots: raices, DNSName: nombre, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}})
		return err
	}
	for _, nombre := range append([]string{"localhost", "127.0.0.1"}, hoja.DNSNames...) {
		if err := verificar(hoja, nombre); err != nil {
			t.Errorf("el certificado del servidor debe valer para %s: %v", nombre, err)
		}
	}
	// El puente presenta la cadena: el certificado del servidor y la autoridad.
	par, err := tls.LoadX509KeyPair(RutasAutofirmado(dir))
	if err != nil || len(par.Certificate) != 2 {
		t.Fatalf("cadena: %d certificados, %v", len(par.Certificate), err)
	}

	llaveCA, err := tls.LoadX509KeyPair(RutasAutoridad(dir))
	if err != nil {
		t.Fatal(err)
	}
	firmar := func(dns []string, ips []net.IP) *x509.Certificate {
		llave, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		plantilla := &x509.Certificate{SerialNumber: big.NewInt(time.Now().UnixNano()), Subject: pkix.Name{CommonName: "hoja"},
			NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), DNSNames: dns, IPAddresses: ips,
			KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
		der, err := x509.CreateCertificate(rand.Reader, plantilla, ca, &llave.PublicKey, llaveCA.PrivateKey)
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
			t.Errorf("la llave de la autoridad no debe poder firmar %s", c.nombre)
		}
	}
}

// TestCargarRenuevaConLaMismaAutoridad: al arrancar, un certificado del servidor por vencer se renueva
// solo con la misma autoridad, porque ningún equipo tiene que volver a confiar en nada.
func TestCargarRenuevaConLaMismaAutoridad(t *testing.T) {
	dir := t.TempDir()
	gen, err := GenerarAutofirmado(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	llaveCA, err := tls.LoadX509KeyPair(RutasAutoridad(dir))
	if err != nil {
		t.Fatal(err)
	}
	// Un certificado del servidor que vence en 10 días, firmado por la misma autoridad.
	llave, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	plantilla := &x509.Certificate{SerialNumber: big.NewInt(7), Subject: pkix.Name{CommonName: "por vencer"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(10 * 24 * time.Hour), DNSNames: []string{"localhost"},
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, plantilla, gen.Autoridad, &llave.PublicKey, llaveCA.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	rutaCert, rutaLlave := RutasAutofirmado(dir)
	if err := escribirPar(rutaCert, rutaLlave, [][]byte{der, gen.Autoridad.Raw}, llave); err != nil {
		t.Fatal(err)
	}

	if _, err := CargarAutofirmado(dir); err != nil {
		t.Fatal(err)
	}
	hoja, err := LeerCertificado(rutaCert)
	if err != nil {
		t.Fatal(err)
	}
	if estado, _ := Vencimiento(hoja, time.Now()); estado != CertificadoVigente || hoja.CheckSignatureFrom(gen.Autoridad) != nil {
		t.Fatalf("debía renovarse con la misma autoridad: %s, vence %s", estado, hoja.NotAfter)
	}
	if ca, err := AutoridadLocal(dir); err != nil || !ca.Equal(gen.Autoridad) {
		t.Fatal("la autoridad no debía cambiar")
	}
}

// TestRegenerarConservaLaAutoridad (ADR 0021): regenerar renueva solo el certificado del servidor con
// la misma autoridad, así que los equipos que confían en ella no tienen que hacer nada; con
// nuevaAutoridad, o si no había autoridad (puentes de antes de 0.3.1), se crea otra.
func TestRegenerarConservaLaAutoridad(t *testing.T) {
	dir := t.TempDir()
	primero, err := GenerarAutofirmado(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := GenerarAutofirmado(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if segundo.AutoridadNueva || !segundo.Autoridad.Equal(primero.Autoridad) || segundo.Servidor.Equal(primero.Servidor) {
		t.Fatal("regenerar debe emitir otro certificado del servidor con la misma autoridad")
	}
	if err := segundo.Servidor.CheckSignatureFrom(primero.Autoridad); err != nil {
		t.Fatal(err)
	}
	tercero, err := GenerarAutofirmado(dir, true)
	if err != nil || !tercero.AutoridadNueva || tercero.Autoridad.Equal(primero.Autoridad) {
		t.Fatalf("con nuevaAutoridad debe crear otra: %v", err)
	}
	// En el mismo equipo, con otro sujeto: si no, Firefox confunde la vieja con la nueva
	// (SEC_ERROR_UNKNOWN) cuando las dos están en sus raíces.
	if bytes.Equal(tercero.Autoridad.RawSubject, primero.Autoridad.RawSubject) || tercero.Servidor.Issuer.String() != tercero.Autoridad.Subject.String() {
		t.Fatalf("dos autoridades del mismo equipo con el mismo sujeto: %s", primero.Autoridad.Subject)
	}

	// Sin autoridad (como un puente de 0.3.0): se crea al regenerar.
	ruta, llave := RutasAutoridad(dir)
	os.Remove(ruta)
	os.Remove(llave)
	if _, err := AutoridadLocal(dir); err == nil {
		t.Fatal("sin archivos no hay autoridad")
	}
	cuarto, err := GenerarAutofirmado(dir, false)
	if err != nil || !cuarto.AutoridadNueva {
		t.Fatalf("sin autoridad debe crear una: %v", err)
	}
}
