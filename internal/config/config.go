// Package config carga y valida config.toml, que solo contiene lo necesario para arrancar
// (red, TLS, rutas). Todo lo demás se administra desde el panel y vive en la base de datos.
package config

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/bayalad7/prisma/internal/rutas"
)

// Modos de TLS admitidos en servidor.modo_tls.
const (
	TLSAutofirmado       = "autofirmado"
	TLSArchivos          = "archivos"
	TLSSoloLocalhostHTTP = "solo_localhost_http"
	descripcionModosTLS  = `"autofirmado", "archivos" o "solo_localhost_http"`
)

type Config struct {
	Servidor Servidor `toml:"servidor"`
	Datos    Datos    `toml:"datos"`
	Dev      Dev      `toml:"dev"`
	Limites  Limites  `toml:"limites"`
}

// Limites de peticiones a la API pública. 0 desactiva el límite.
type Limites struct {
	PorIP      int `toml:"peticiones_ip_por_minuto"`
	PorCliente int `toml:"peticiones_cliente_por_minuto"`
}

type Servidor struct {
	Escuchar    string `toml:"escuchar"`
	Puerto      int    `toml:"puerto"`
	ModoTLS     string `toml:"modo_tls"`
	Certificado string `toml:"certificado"`
	Llave       string `toml:"llave"`
}

type Datos struct {
	// Ruta del directorio de datos; vacío = ruta por defecto del sistema.
	Ruta string `toml:"ruta"`
}

type Dev struct {
	ModoDesarrollo bool `toml:"modo_desarrollo"`
}

// PorDefecto devuelve la configuración que se usa cuando no existe config.toml.
func PorDefecto() Config {
	return Config{
		Servidor: Servidor{
			Escuchar: "0.0.0.0",
			Puerto:   8443,
			ModoTLS:  TLSAutofirmado,
		},
		Limites: Limites{PorIP: 120, PorCliente: 60},
	}
}

// DirDatos devuelve el directorio de datos efectivo.
func (c Config) DirDatos() string {
	if c.Datos.Ruta != "" {
		return c.Datos.Ruta
	}
	return rutas.PorDefecto().Datos
}

// RutaBaseDatos devuelve la ruta efectiva del archivo SQLite.
func (c Config) RutaBaseDatos() string {
	return rutas.BaseDatos(c.DirDatos())
}

// Origen describe de dónde salió la configuración cargada.
type Origen struct {
	Ruta   string // archivo consultado
	Existe bool   // false = no existe y se usan los valores por defecto
}

// Cargar lee la configuración. Si rutaExplicita está vacía se usa la ruta por defecto del
// sistema y, si ese archivo no existe, los valores por defecto. Si rutaExplicita no está
// vacía, el archivo debe existir.
func Cargar(rutaExplicita string) (Config, Origen, error) {
	ruta := rutaExplicita
	if ruta == "" {
		ruta = rutas.PorDefecto().Config
	}
	origen := Origen{Ruta: ruta}

	if _, err := os.Stat(ruta); err != nil {
		if errors.Is(err, fs.ErrNotExist) && rutaExplicita == "" {
			c := PorDefecto()
			return c, origen, c.Validar()
		}
		if errors.Is(err, fs.ErrNotExist) {
			return Config{}, origen, fmt.Errorf("no existe el archivo de configuración %q", ruta)
		}
		return Config{}, origen, fmt.Errorf("no se puede leer el archivo de configuración %q: %w", ruta, err)
	}
	origen.Existe = true

	c, desconocidas, err := leerArchivo(ruta)
	if err != nil {
		return Config{}, origen, err
	}
	// Se reportan juntas las claves desconocidas y los valores inválidos.
	problemas := desconocidas
	var ev *ErrorValidacion
	if err := c.Validar(); errors.As(err, &ev) {
		problemas = append(problemas, ev.Problemas...)
	}
	if len(problemas) > 0 {
		return Config{}, origen, &ErrorValidacion{Problemas: problemas}
	}
	return c, origen, nil
}

func leerArchivo(ruta string) (Config, []Problema, error) {
	c := PorDefecto()
	md, err := toml.DecodeFile(ruta, &c)
	if err != nil {
		var perr toml.ParseError
		if errors.As(err, &perr) {
			detalle := perr.Message
			if perr.LastKey != "" {
				detalle = fmt.Sprintf("clave %q: %s", perr.LastKey, perr.Message)
			}
			return Config{}, nil, fmt.Errorf("línea %d: %s", perr.Position.Line, detalle)
		}
		return Config{}, nil, err
	}

	var problemas []Problema
	for _, clave := range md.Undecoded() {
		problemas = append(problemas, Problema{
			Clave:   clave.String(),
			Mensaje: "clave desconocida; revisa que esté bien escrita y en la sección correcta",
		})
	}

	// Las rutas relativas de certificado y llave se resuelven respecto al archivo de configuración.
	dir := filepath.Dir(ruta)
	if c.Servidor.Certificado != "" && !filepath.IsAbs(c.Servidor.Certificado) {
		c.Servidor.Certificado = filepath.Join(dir, c.Servidor.Certificado)
	}
	if c.Servidor.Llave != "" && !filepath.IsAbs(c.Servidor.Llave) {
		c.Servidor.Llave = filepath.Join(dir, c.Servidor.Llave)
	}
	return c, problemas, nil
}

// Problema es un error de validación de una clave concreta.
type Problema struct {
	Clave   string
	Mensaje string
}

// ErrorValidacion reúne todos los problemas encontrados, para corregirlos de una sola vez.
type ErrorValidacion struct {
	Problemas []Problema
}

func (e *ErrorValidacion) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "la configuración tiene %d problema(s):", len(e.Problemas))
	for _, p := range e.Problemas {
		fmt.Fprintf(&b, "\n  - %s: %s", p.Clave, p.Mensaje)
	}
	return b.String()
}

// Validar comprueba la configuración y devuelve *ErrorValidacion si algo está mal.
func (c Config) Validar() error {
	var ps []Problema
	agregar := func(clave, formato string, args ...any) {
		ps = append(ps, Problema{Clave: clave, Mensaje: fmt.Sprintf(formato, args...)})
	}

	s := c.Servidor
	if s.Escuchar == "" {
		agregar("servidor.escuchar", `no puede estar vacío; usa "0.0.0.0" para toda la red o "127.0.0.1" solo para este equipo`)
	} else if s.Escuchar != "localhost" && net.ParseIP(s.Escuchar) == nil {
		agregar("servidor.escuchar", "%q no es una dirección IP válida (ej. \"0.0.0.0\" o \"127.0.0.1\")", s.Escuchar)
	}
	if s.Puerto < 1 || s.Puerto > 65535 {
		agregar("servidor.puerto", "debe estar entre 1 y 65535 (valor: %d)", s.Puerto)
	}

	switch s.ModoTLS {
	case TLSAutofirmado, TLSSoloLocalhostHTTP:
		if s.Certificado != "" || s.Llave != "" {
			agregar("servidor.certificado", "certificado y llave solo se usan con modo_tls = %q; déjalos vacíos o cambia el modo", TLSArchivos)
		}
		if s.ModoTLS == TLSSoloLocalhostHTTP && s.Escuchar != "" && !esLoopback(s.Escuchar) {
			agregar("servidor.escuchar", "con modo_tls = %q el puente sirve HTTP sin cifrar y solo puede escuchar en este equipo (\"127.0.0.1\", \"::1\" o \"localhost\"), no en %q",
				TLSSoloLocalhostHTTP, s.Escuchar)
		}
	case TLSArchivos:
		validarArchivoTLS(s.Certificado, "servidor.certificado", "certificado", agregar)
		validarArchivoTLS(s.Llave, "servidor.llave", "llave privada", agregar)
		if s.Certificado != "" && s.Llave != "" && len(ps) == 0 {
			if _, err := tls.LoadX509KeyPair(s.Certificado, s.Llave); err != nil {
				agregar("servidor.certificado", "el certificado y la llave no forman un par válido en formato PEM: %v", err)
			}
		}
	default:
		agregar("servidor.modo_tls", "%q no es válido; usa %s", s.ModoTLS, descripcionModosTLS)
	}

	if c.Limites.PorIP < 0 {
		agregar("limites.peticiones_ip_por_minuto", "no puede ser negativo (0 desactiva el límite)")
	}
	if c.Limites.PorCliente < 0 {
		agregar("limites.peticiones_cliente_por_minuto", "no puede ser negativo (0 desactiva el límite)")
	}

	if c.Datos.Ruta != "" && !filepath.IsAbs(c.Datos.Ruta) {
		agregar("datos.ruta", "debe ser una ruta absoluta (valor: %q); déjala vacía para usar la ruta por defecto del sistema", c.Datos.Ruta)
	}

	if len(ps) > 0 {
		return &ErrorValidacion{Problemas: ps}
	}
	return nil
}

func validarArchivoTLS(ruta, clave, nombre string, agregar func(string, string, ...any)) {
	if ruta == "" {
		agregar(clave, "es obligatorio con modo_tls = %q: indica la ruta del archivo de %s en formato PEM", TLSArchivos, nombre)
		return
	}
	info, err := os.Stat(ruta)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		agregar(clave, "no existe el archivo %q", ruta)
	case err != nil:
		agregar(clave, "no se puede leer %q: %v", ruta, err)
	case info.IsDir():
		agregar(clave, "%q es un directorio, no un archivo", ruta)
	}
}

func esLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
