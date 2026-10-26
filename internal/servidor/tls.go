package servidor

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

// ProveedorCertificado entrega el certificado para cada conexión TLS. Permite agregar en el
// futuro un proveedor automático (ACME) sin cambiar el servidor.
type ProveedorCertificado interface {
	ObtenerCertificado(*tls.ClientHelloInfo) (*tls.Certificate, error)
}

// ConfigTLS crea la configuración TLS del servidor (TLS 1.2 como mínimo).
func ConfigTLS(p ProveedorCertificado) *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS12, GetCertificate: p.ObtenerCertificado}
}

// Archivos usa un certificado y una llave PEM del disco y los recarga si cambian, sin reiniciar.
type Archivos struct {
	Certificado, Llave string

	mu       sync.Mutex
	actual   *tls.Certificate
	modCert  time.Time
	modLlave time.Time
	revisado time.Time
}

// NuevoArchivos carga el par; falla si no forma un certificado válido.
func NuevoArchivos(certificado, llave string) (*Archivos, error) {
	a := &Archivos{Certificado: certificado, Llave: llave}
	if err := a.recargar(); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *Archivos) recargar() error {
	ic, err := os.Stat(a.Certificado)
	if err != nil {
		return err
	}
	il, err := os.Stat(a.Llave)
	if err != nil {
		return err
	}
	par, err := tls.LoadX509KeyPair(a.Certificado, a.Llave)
	if err != nil {
		return fmt.Errorf("el certificado y la llave no forman un par válido: %w", err)
	}
	a.actual, a.modCert, a.modLlave = &par, ic.ModTime(), il.ModTime()
	return nil
}

// ObtenerCertificado revisa como máximo cada 5 s si los archivos cambiaron. Si la recarga falla
// (por ejemplo, a mitad de una copia), sigue usando el certificado anterior.
func (a *Archivos) ObtenerCertificado(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if time.Since(a.revisado) > 5*time.Second {
		a.revisado = time.Now()
		ic, err1 := os.Stat(a.Certificado)
		il, err2 := os.Stat(a.Llave)
		if err1 == nil && err2 == nil && (!ic.ModTime().Equal(a.modCert) || !il.ModTime().Equal(a.modLlave)) {
			a.recargar()
		}
	}
	return a.actual, nil
}

// Archivos del certificado autofirmado dentro de la carpeta de datos.
const (
	carpetaTLS          = "tls"
	archivoCertificado  = "certificado.pem"
	archivoLlave        = "llave.pem"
	vigenciaAutofirmado = 825 * 24 * time.Hour // máximo que aceptan macOS e iOS
)

// RutasAutofirmado devuelve dónde se guarda el certificado autofirmado.
func RutasAutofirmado(dirDatos string) (certificado, llave string) {
	dir := filepath.Join(dirDatos, carpetaTLS)
	return filepath.Join(dir, archivoCertificado), filepath.Join(dir, archivoLlave)
}

// CargarAutofirmado usa el certificado autofirmado de la carpeta de datos; lo genera la primera vez.
// Si el que hay ya venció, genera otro: uno vencido no le sirve a ningún cliente (habrá que volver
// a instalarlo en las cajas, y el panel lo avisa).
func CargarAutofirmado(dirDatos string) (*Archivos, error) {
	cert, llave := RutasAutofirmado(dirDatos)
	_, err := os.Stat(cert)
	generar := errors.Is(err, fs.ErrNotExist)
	if c, err := LeerCertificado(cert); err == nil {
		if estado, _ := Vencimiento(c, time.Now()); estado == CertificadoVencido {
			generar = true
		}
	}
	if generar {
		if _, err := GenerarAutofirmado(dirDatos); err != nil {
			return nil, err
		}
	}
	return NuevoArchivos(cert, llave)
}

// NombresLocales devuelve el hostname y las IPs de este equipo, que van en el certificado.
func NombresLocales() (dns []string, ips []net.IP) {
	dns = []string{"localhost"}
	if h, err := os.Hostname(); err == nil && h != "" {
		h = strings.ToLower(h)
		dns = append(dns, h)
		if !strings.Contains(h, ".") {
			dns = append(dns, h+".local")
		}
	}
	ips = []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		dirs, _ := iface.Addrs()
		for _, d := range dirs {
			if ipn, ok := d.(*net.IPNet); ok && !ipn.IP.IsLinkLocalUnicast() {
				ips = append(ips, ipn.IP)
			}
		}
	}
	return dns, ips
}

// redesPrivadas son los rangos en los que puede estar el puente: loopback, redes privadas
// (RFC 1918 y fc00::/7) y enlace local.
var redesPrivadas = []string{"127.0.0.0/8", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "169.254.0.0/16",
	"::1/128", "fc00::/7", "fe80::/10"}

// dominiosPermitidos: localhost, los nombres .local (mDNS) y los nombres de este equipo.
func dominiosPermitidos(dns []string) []string {
	permitidos := []string{"localhost", "local"}
	for _, d := range dns {
		if d != "localhost" && !strings.HasSuffix(d, ".local") && !slices.Contains(permitidos, d) {
			permitidos = append(permitidos, d)
		}
	}
	return permitidos
}

// redesPermitidas: las redes privadas y, por si el equipo tiene una IP pública, sus propias IPs.
func redesPermitidas(ips []net.IP) []*net.IPNet {
	var redes []*net.IPNet
	for _, r := range redesPrivadas {
		_, red, _ := net.ParseCIDR(r)
		redes = append(redes, red)
	}
	for _, ip := range ips {
		dentro := false
		for _, red := range redes {
			dentro = dentro || red.Contains(ip)
		}
		if !dentro {
			bits := 128
			if ip.To4() != nil {
				ip, bits = ip.To4(), 32
			}
			redes = append(redes, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
		}
	}
	return redes
}

// LimitadoALaRedLocal dice si el certificado lleva las restricciones de nombre de PRISMA 0.3.0 o
// posterior. Solo en uno así se puede confiar como raíz en un equipo sin riesgo (prisma tls confiar).
func LimitadoALaRedLocal(c *x509.Certificate) bool {
	return c.PermittedDNSDomainsCritical && len(c.PermittedDNSDomains) > 0 && len(c.PermittedIPRanges) > 0
}

// GenerarAutofirmado crea (o reemplaza) el certificado autofirmado con el hostname y las IPs
// actuales de este equipo. Usa ECDSA P-256, dura 825 días y está limitado a la red local.
func GenerarAutofirmado(dirDatos string) (*x509.Certificate, error) {
	llave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	serie, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}
	dns, ips := NombresLocales()
	ahora := time.Now()
	plantilla := &x509.Certificate{
		SerialNumber: serie,
		Subject:      pkix.Name{CommonName: "PRISMA " + dns[len(dns)-1], Organization: []string{"PRISMA (autofirmado)"}},
		NotBefore:    ahora.Add(-time.Hour),
		NotAfter:     ahora.Add(vigenciaAutofirmado),
		// Autofirmado como CA: es la forma que aceptan todos los almacenes de confianza al importarlo.
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:              dns,
		IPAddresses:           ips,
		// Limitado a la red local (ADR 0020): aunque un equipo confíe en él como raíz, su llave no
		// sirve para hacerse pasar por ningún sitio de Internet.
		PermittedDNSDomainsCritical: true,
		PermittedDNSDomains:         dominiosPermitidos(dns),
		PermittedIPRanges:           redesPermitidas(ips),
	}
	der, err := x509.CreateCertificate(rand.Reader, plantilla, plantilla, &llave.PublicKey, llave)
	if err != nil {
		return nil, err
	}
	llaveDER, err := x509.MarshalPKCS8PrivateKey(llave)
	if err != nil {
		return nil, err
	}
	rutaCert, rutaLlave := RutasAutofirmado(dirDatos)
	if err := os.MkdirAll(filepath.Dir(rutaCert), 0o700); err != nil {
		return nil, err
	}
	// La llave primero y con permisos restringidos; el certificado es público.
	if err := os.WriteFile(rutaLlave, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: llaveDER}), 0o600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(rutaCert, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		return nil, err
	}
	return x509.ParseCertificate(der)
}

// LeerCertificado lee el primer certificado de un archivo PEM.
func LeerCertificado(ruta string) (*x509.Certificate, error) {
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return nil, err
	}
	b, _ := pem.Decode(datos)
	if b == nil || b.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("%s no contiene un certificado PEM", ruta)
	}
	return x509.ParseCertificate(b.Bytes)
}

// Huella devuelve la huella SHA-256 del certificado, en el formato que muestran los navegadores.
func Huella(c *x509.Certificate) string {
	h := sha256.Sum256(c.Raw)
	partes := make([]string, len(h))
	for i, b := range h {
		partes[i] = fmt.Sprintf("%02X", b)
	}
	return strings.Join(partes, ":")
}

// IPsSinCubrir devuelve las IPs actuales de este equipo que el certificado no incluye (por
// ejemplo, si el router le asignó otra). Los clientes que usen esas IPs verán un error de TLS.
func IPsSinCubrir(c *x509.Certificate) []net.IP {
	_, actuales := NombresLocales()
	var faltan []net.IP
	for _, ip := range actuales {
		cubierta := false
		for _, cip := range c.IPAddresses {
			if cip.Equal(ip) {
				cubierta = true
				break
			}
		}
		if !cubierta {
			faltan = append(faltan, ip)
		}
	}
	return faltan
}

// Estados de vencimiento de un certificado.
const (
	CertificadoVigente   = "vigente"
	CertificadoPorVencer = "por_vencer" // vence en menos de AvisoVencimiento
	CertificadoVencido   = "vencido"
)

// AvisoVencimiento es cuánto antes del vencimiento se avisa en la bitácora y en el panel.
const AvisoVencimiento = 60 * 24 * time.Hour

// Vencimiento devuelve el estado del certificado y los días que le quedan (negativos si venció).
func Vencimiento(c *x509.Certificate, ahora time.Time) (estado string, dias int) {
	resta := c.NotAfter.Sub(ahora)
	dias = int(resta.Hours() / 24)
	switch {
	case resta <= 0:
		return CertificadoVencido, dias
	case resta < AvisoVencimiento:
		return CertificadoPorVencer, dias
	}
	return CertificadoVigente, dias
}
