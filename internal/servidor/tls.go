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
// Certificado «autofirmado» del puente (ADR 0021): una autoridad local, limitada a la red local, que
// es en lo único que confían los equipos, y un certificado del servidor firmado por ella, que el
// puente presenta junto con la autoridad. Firefox no acepta que la autoridad misma sea el
// certificado del sitio (MOZILLA_PKIX_ERROR_CA_CERT_USED_AS_END_ENTITY).
const (
	carpetaTLS            = "tls"
	archivoCertificado    = "certificado.pem" // el del servidor, seguido de la autoridad (la cadena)
	archivoLlave          = "llave.pem"
	archivoAutoridad      = "autoridad.pem"
	archivoLlaveAutoridad = "autoridad-llave.pem"
	vigenciaAutofirmado   = 825 * 24 * time.Hour      // máximo que aceptan macOS e iOS
	vigenciaAutoridad     = 10 * 365 * 24 * time.Hour // los equipos confían en ella una sola vez
	// OrganizacionAutoridad identifica la autoridad del puente en los almacenes de confianza (igual
	// que servicio.OrganizacionCertificado).
	OrganizacionAutoridad = "PRISMA (autofirmado)"
)

// RutasAutofirmado devuelve dónde se guarda el certificado del servidor (con la cadena) y su llave.
func RutasAutofirmado(dirDatos string) (certificado, llave string) {
	dir := filepath.Join(dirDatos, carpetaTLS)
	return filepath.Join(dir, archivoCertificado), filepath.Join(dir, archivoLlave)
}

// RutasAutoridad devuelve dónde se guarda la autoridad local y su llave.
func RutasAutoridad(dirDatos string) (certificado, llave string) {
	dir := filepath.Join(dirDatos, carpetaTLS)
	return filepath.Join(dir, archivoAutoridad), filepath.Join(dir, archivoLlaveAutoridad)
}

// AutoridadLocal devuelve la autoridad del puente si existe y está limitada a la red local. Los
// puentes de antes de 0.3.1 no tienen: su certificado es la autoridad misma.
func AutoridadLocal(dirDatos string) (*x509.Certificate, error) {
	ruta, _ := RutasAutoridad(dirDatos)
	ca, err := LeerCertificado(ruta)
	if err != nil {
		return nil, err
	}
	if !ca.IsCA || !LimitadoALaRedLocal(ca) {
		return nil, fmt.Errorf("%s no es una autoridad limitada a la red local", ruta)
	}
	return ca, nil
}

// CargarAutofirmado usa el certificado del servidor de la carpeta de datos; lo genera la primera vez.
// Si venció, está por vencer o no incluye alguna IP actual, y el puente tiene autoridad local, lo
// renueva con la misma autoridad: los equipos que confían en ella no tienen que hacer nada. Sin
// autoridad (antes de 0.3.1), solo lo genera si venció (habrá que volver a instalarlo, y el panel lo
// avisa).
func CargarAutofirmado(dirDatos string) (*Archivos, error) {
	cert, llave := RutasAutofirmado(dirDatos)
	_, err := os.Stat(cert)
	generar := errors.Is(err, fs.ErrNotExist)
	if c, err := LeerCertificado(cert); err == nil {
		estado, _ := Vencimiento(c, time.Now())
		_, conAutoridad := AutoridadLocal(dirDatos)
		renovable := conAutoridad == nil && (estado != CertificadoVigente || len(IPsSinCubrir(c)) > 0)
		if estado == CertificadoVencido {
			generar = true
		} else if renovable {
			// Solo con la misma autoridad; si cambió el nombre del equipo, lo decide el dueño.
			if _, err := RenovarConLaMismaAutoridad(dirDatos); err != nil && !errors.Is(err, ErrAutoridadNoCubre) {
				return nil, err
			}
		}
	}
	if generar {
		if _, err := GenerarAutofirmado(dirDatos, false); err != nil {
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

// LimitadoALaRedLocal dice si el certificado lleva las restricciones de nombre de la autoridad local
// (desde 0.3.0). Solo en uno así se puede confiar como raíz en un equipo sin riesgo.
func LimitadoALaRedLocal(c *x509.Certificate) bool {
	return c.PermittedDNSDomainsCritical && len(c.PermittedDNSDomains) > 0 && len(c.PermittedIPRanges) > 0
}

// Autofirmado es el resultado de generar: el certificado del servidor, la autoridad que lo firmó y si
// esa autoridad es nueva (entonces los equipos deben volver a confiar en ella).
type Autofirmado struct {
	Servidor, Autoridad *x509.Certificate
	AutoridadNueva      bool
}

// ErrAutoridadNoCubre: los nombres o IPs actuales del equipo quedan fuera de la autoridad local (por
// ejemplo, cambió el nombre del equipo); hace falta una autoridad nueva.
var ErrAutoridadNoCubre = errors.New("la autoridad local no cubre los nombres o IPs actuales del equipo")

// GenerarAutofirmado crea un certificado del servidor nuevo con el hostname y las IPs actuales,
// firmado por la autoridad local. Reutiliza la autoridad si existe, está vigente y cubre los nombres
// actuales; si no (o con nuevaAutoridad), crea otra.
func GenerarAutofirmado(dirDatos string, nuevaAutoridad bool) (*Autofirmado, error) {
	if !nuevaAutoridad {
		a, err := RenovarConLaMismaAutoridad(dirDatos)
		if err == nil {
			return a, nil
		}
		if !errors.Is(err, ErrAutoridadNoCubre) && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	ca, llaveCA, err := crearAutoridad(dirDatos)
	if err != nil {
		return nil, err
	}
	hoja, err := emitirServidor(dirDatos, ca, llaveCA)
	if err != nil {
		return nil, err
	}
	return &Autofirmado{Servidor: hoja, Autoridad: ca, AutoridadNueva: true}, nil
}

// RenovarConLaMismaAutoridad emite un certificado del servidor nuevo con la autoridad que ya hay.
// Devuelve fs.ErrNotExist si no hay autoridad y ErrAutoridadNoCubre si no sirve para los nombres
// actuales o vence antes de un día.
func RenovarConLaMismaAutoridad(dirDatos string) (*Autofirmado, error) {
	ca, err := AutoridadLocal(dirDatos)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", ErrAutoridadNoCubre, err)
	}
	if time.Until(ca.NotAfter) < 24*time.Hour {
		return nil, ErrAutoridadNoCubre
	}
	_, rutaLlave := RutasAutoridad(dirDatos)
	datos, err := os.ReadFile(rutaLlave)
	if err != nil {
		return nil, err
	}
	b, _ := pem.Decode(datos)
	if b == nil {
		return nil, fmt.Errorf("%s no contiene una llave PEM", rutaLlave)
	}
	llave, err := x509.ParsePKCS8PrivateKey(b.Bytes)
	if err != nil {
		return nil, err
	}
	firmante, ok := llave.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%s no es una llave ECDSA", rutaLlave)
	}
	// Antes de escribir nada: ¿la autoridad cubre los nombres y las IPs actuales?
	dns, ips := NombresLocales()
	if !cubre(ca, dns, ips) {
		return nil, ErrAutoridadNoCubre
	}
	hoja, err := emitirServidor(dirDatos, ca, firmante)
	if err != nil {
		return nil, err
	}
	return &Autofirmado{Servidor: hoja, Autoridad: ca}, nil
}

// cubre dice si un certificado del servidor con esos nombres e IPs sería válido bajo la autoridad.
func cubre(ca *x509.Certificate, dns []string, ips []net.IP) bool {
	for _, d := range dns {
		permitido := false
		for _, p := range ca.PermittedDNSDomains {
			permitido = permitido || d == p || strings.HasSuffix(d, "."+p)
		}
		if !permitido {
			return false
		}
	}
	for _, ip := range ips {
		permitida := false
		for _, red := range ca.PermittedIPRanges {
			permitida = permitida || red.Contains(ip)
		}
		if !permitida {
			return false
		}
	}
	return true
}

func nuevaSerie() (*big.Int, error) {
	return rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
}

// crearAutoridad crea la autoridad local, limitada a la red local (ADR 0020): aunque un equipo confíe
// en ella, su llave no sirve para hacerse pasar por ningún sitio de Internet.
func crearAutoridad(dirDatos string) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	llave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	serie, err := nuevaSerie()
	if err != nil {
		return nil, nil, err
	}
	dns, ips := NombresLocales()
	ahora := time.Now()
	plantilla := &x509.Certificate{
		SerialNumber:                serie,
		Subject:                     pkix.Name{CommonName: "PRISMA autoridad local " + dns[len(dns)-1], Organization: []string{OrganizacionAutoridad}},
		NotBefore:                   ahora.Add(-time.Hour),
		NotAfter:                    ahora.Add(vigenciaAutoridad),
		IsCA:                        true,
		BasicConstraintsValid:       true,
		MaxPathLenZero:              true, // solo firma certificados de servidor, nunca otras autoridades
		KeyUsage:                    x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		PermittedDNSDomainsCritical: true,
		PermittedDNSDomains:         dominiosPermitidos(dns),
		PermittedIPRanges:           redesPermitidas(ips),
	}
	der, err := x509.CreateCertificate(rand.Reader, plantilla, plantilla, &llave.PublicKey, llave)
	if err != nil {
		return nil, nil, err
	}
	rutaCert, rutaLlave := RutasAutoridad(dirDatos)
	if err := escribirPar(rutaCert, rutaLlave, [][]byte{der}, llave); err != nil {
		return nil, nil, err
	}
	ca, err := x509.ParseCertificate(der)
	return ca, llave, err
}

// emitirServidor crea el certificado del servidor (no es autoridad) con los nombres e IPs actuales,
// firmado por la autoridad, y lo guarda seguido de la autoridad para presentar la cadena completa.
func emitirServidor(dirDatos string, ca *x509.Certificate, llaveCA *ecdsa.PrivateKey) (*x509.Certificate, error) {
	llave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	serie, err := nuevaSerie()
	if err != nil {
		return nil, err
	}
	dns, ips := NombresLocales()
	ahora := time.Now()
	vence := ahora.Add(vigenciaAutofirmado)
	if ca.NotAfter.Before(vence) {
		vence = ca.NotAfter
	}
	plantilla := &x509.Certificate{
		SerialNumber:          serie,
		Subject:               pkix.Name{CommonName: "PRISMA " + dns[len(dns)-1], Organization: []string{"PRISMA"}},
		NotBefore:             ahora.Add(-time.Hour),
		NotAfter:              vence,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:              dns,
		IPAddresses:           ips,
	}
	der, err := x509.CreateCertificate(rand.Reader, plantilla, ca, &llave.PublicKey, llaveCA)
	if err != nil {
		return nil, err
	}
	rutaCert, rutaLlave := RutasAutofirmado(dirDatos)
	if err := escribirPar(rutaCert, rutaLlave, [][]byte{der, ca.Raw}, llave); err != nil {
		return nil, err
	}
	return x509.ParseCertificate(der)
}

// escribirPar guarda la llave (primero y solo para el usuario del servicio) y los certificados (PEM,
// públicos).
func escribirPar(rutaCert, rutaLlave string, certificados [][]byte, llave *ecdsa.PrivateKey) error {
	llaveDER, err := x509.MarshalPKCS8PrivateKey(llave)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(rutaCert), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(rutaLlave, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: llaveDER}), 0o600); err != nil {
		return err
	}
	var pemCerts []byte
	for _, der := range certificados {
		pemCerts = append(pemCerts, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})...)
	}
	return os.WriteFile(rutaCert, pemCerts, 0o644)
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
