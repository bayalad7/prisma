package config

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func escribir(t *testing.T, contenido string) string {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(ruta, []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func TestPorDefectoEsValida(t *testing.T) {
	if err := PorDefecto().Validar(); err != nil {
		t.Fatalf("la configuración por defecto debe ser válida: %v", err)
	}
}

func TestCargarArchivoCompleto(t *testing.T) {
	dirDatos := t.TempDir()
	ruta := escribir(t, `
[servidor]
escuchar = "127.0.0.1"
puerto = 9443
modo_tls = "autofirmado"

[datos]
ruta = '`+dirDatos+`'

[dev]
modo_desarrollo = true
`)
	c, origen, err := Cargar(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if !origen.Existe || origen.Ruta != ruta {
		t.Errorf("origen = %+v", origen)
	}
	if c.Servidor.Escuchar != "127.0.0.1" || c.Servidor.Puerto != 9443 || !c.Dev.ModoDesarrollo {
		t.Errorf("valores leídos incorrectos: %+v", c)
	}
	if c.DirDatos() != dirDatos {
		t.Errorf("DirDatos = %q, quiero %q", c.DirDatos(), dirDatos)
	}
}

func TestCargarAplicaPorDefectoAClavesOmitidas(t *testing.T) {
	c, _, err := Cargar(escribir(t, "[servidor]\npuerto = 9000\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Servidor.Escuchar != "0.0.0.0" || c.Servidor.ModoTLS != TLSAutofirmado || c.Servidor.Puerto != 9000 {
		t.Errorf("no se aplicaron los valores por defecto: %+v", c.Servidor)
	}
}

func TestCargarRutaExplicitaInexistente(t *testing.T) {
	_, _, err := Cargar(filepath.Join(t.TempDir(), "no-existe.toml"))
	if err == nil || !strings.Contains(err.Error(), "no existe") {
		t.Fatalf("esperaba error de archivo inexistente, obtuve %v", err)
	}
}

func TestErroresDeLectura(t *testing.T) {
	casos := []struct {
		nombre, contenido, contiene string
	}{
		{"sintaxis", "[servidor]\npuerto = \n", "línea 2"},
		{"tipo incorrecto", "[servidor]\npuerto = \"ocho\"\n", "puerto"},
		{"clave desconocida", "[servidor]\npurto = 8443\n", "servidor.purto: clave desconocida"},
		{"sección desconocida", "[red]\npuerto = 8443\n", "red"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			_, _, err := Cargar(escribir(t, c.contenido))
			if err == nil || !strings.Contains(err.Error(), c.contiene) {
				t.Fatalf("esperaba un error que contenga %q, obtuve %v", c.contiene, err)
			}
		})
	}
}

func TestValidar(t *testing.T) {
	casos := []struct {
		nombre  string
		cambiar func(*Config)
		clave   string
	}{
		{"escuchar vacío", func(c *Config) { c.Servidor.Escuchar = "" }, "servidor.escuchar"},
		{"escuchar inválido", func(c *Config) { c.Servidor.Escuchar = "mi-pc" }, "servidor.escuchar"},
		{"puerto cero", func(c *Config) { c.Servidor.Puerto = 0 }, "servidor.puerto"},
		{"puerto alto", func(c *Config) { c.Servidor.Puerto = 70000 }, "servidor.puerto"},
		{"modo desconocido", func(c *Config) { c.Servidor.ModoTLS = "letsencrypt" }, "servidor.modo_tls"},
		{"archivos sin certificado", func(c *Config) { c.Servidor.ModoTLS = TLSArchivos }, "servidor.certificado"},
		{"archivos inexistentes", func(c *Config) {
			c.Servidor.ModoTLS = TLSArchivos
			c.Servidor.Certificado = filepath.Join(os.TempDir(), "prisma-no-existe.pem")
			c.Servidor.Llave = filepath.Join(os.TempDir(), "prisma-no-existe.key")
		}, "servidor.llave"},
		{"certificado con autofirmado", func(c *Config) { c.Servidor.Certificado = "cert.pem" }, "servidor.certificado"},
		{"http fuera de localhost", func(c *Config) { c.Servidor.ModoTLS = TLSSoloLocalhostHTTP }, "servidor.escuchar"},
		{"ruta de datos relativa", func(c *Config) { c.Datos.Ruta = "datos" }, "datos.ruta"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			c := PorDefecto()
			caso.cambiar(&c)
			err := c.Validar()
			var ev *ErrorValidacion
			if !errors.As(err, &ev) {
				t.Fatalf("esperaba *ErrorValidacion, obtuve %v", err)
			}
			for _, p := range ev.Problemas {
				if p.Clave == caso.clave {
					return
				}
			}
			t.Fatalf("ningún problema para %q en: %v", caso.clave, err)
		})
	}
}

func TestValidarReportaTodosLosProblemas(t *testing.T) {
	c := PorDefecto()
	c.Servidor.Puerto = -1
	c.Servidor.ModoTLS = "x"
	var ev *ErrorValidacion
	if !errors.As(c.Validar(), &ev) || len(ev.Problemas) != 2 {
		t.Fatalf("esperaba 2 problemas, obtuve %v", ev)
	}
}

func TestSoloLocalhostHTTPEnLoopback(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "::1", "localhost"} {
		c := PorDefecto()
		c.Servidor.ModoTLS = TLSSoloLocalhostHTTP
		c.Servidor.Escuchar = host
		if err := c.Validar(); err != nil {
			t.Errorf("%s: %v", host, err)
		}
	}
}

func TestModoArchivosConParValido(t *testing.T) {
	dir := t.TempDir()
	llave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	plantilla := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "prisma-prueba"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, plantilla, plantilla, &llave.PublicKey, llave)
	if err != nil {
		t.Fatal(err)
	}
	llaveDER, err := x509.MarshalECPrivateKey(llave)
	if err != nil {
		t.Fatal(err)
	}
	escribirPEM := func(nombre, tipo string, datos []byte) {
		if err := os.WriteFile(filepath.Join(dir, nombre), pem.EncodeToMemory(&pem.Block{Type: tipo, Bytes: datos}), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	escribirPEM("cert.pem", "CERTIFICATE", der)
	escribirPEM("llave.pem", "EC PRIVATE KEY", llaveDER)
	escribirPEM("otra.pem", "CERTIFICATE", der)

	// Rutas relativas: se resuelven respecto al directorio del config.toml.
	ruta := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(ruta, []byte("[servidor]\nmodo_tls = \"archivos\"\ncertificado = \"cert.pem\"\nllave = \"llave.pem\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Cargar(ruta); err != nil {
		t.Fatalf("par válido rechazado: %v", err)
	}

	c := PorDefecto()
	c.Servidor.ModoTLS = TLSArchivos
	c.Servidor.Certificado = filepath.Join(dir, "cert.pem")
	c.Servidor.Llave = filepath.Join(dir, "otra.pem")
	if err := c.Validar(); err == nil || !strings.Contains(err.Error(), "par válido") {
		t.Fatalf("esperaba error de par inválido, obtuve %v", err)
	}
}

func TestLimites(t *testing.T) {
	c, _, err := Cargar(escribir(t, "[limites]\npeticiones_ip_por_minuto = 0\npeticiones_cliente_por_minuto = 30\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Limites.PorIP != 0 || c.Limites.PorCliente != 30 {
		t.Fatalf("límites: %+v", c.Limites)
	}
	if d := PorDefecto().Limites; d.PorIP != 120 || d.PorCliente != 60 {
		t.Fatalf("por defecto: %+v", d)
	}
	_, _, err = Cargar(escribir(t, "[limites]\npeticiones_ip_por_minuto = -1\n"))
	if err == nil || !strings.Contains(err.Error(), "limites.peticiones_ip_por_minuto") {
		t.Fatalf("esperaba error de límite negativo: %v", err)
	}
}

// La configuración que copian los instaladores (Inno Setup y el .deb) es válida y nunca activa
// el modo de desarrollo, que habilita «prisma imprimir» sin autorización.
func TestConfiguracionDeLosInstaladores(t *testing.T) {
	cfg, origen, err := Cargar("../../empaquetado/config.toml")
	if err != nil {
		t.Fatalf("%s: %v", origen.Ruta, err)
	}
	if cfg.Dev.ModoDesarrollo {
		t.Fatal("empaquetado/config.toml activa modo_desarrollo")
	}
	if cfg.Servidor.ModoTLS != TLSAutofirmado || cfg.Servidor.Puerto != 8443 {
		t.Fatalf("los instaladores asumen HTTPS autofirmado en el puerto 8443: %+v", cfg.Servidor)
	}
}
