package almacen

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ErrOrigenInvalido indica que un texto no es un origen web válido.
var ErrOrigenInvalido = errors.New("origen inválido")

// NormalizarOrigen convierte un origen a su forma canónica esquema://host[:puerto]:
// minúsculas, sin ruta ni "/" final y sin el puerto por defecto del esquema.
// Solo se admiten http y https, sin comodines, usuario, ruta, consulta ni fragmento.
func NormalizarOrigen(origen string) (string, error) {
	invalido := func(motivo string) (string, error) {
		return "", fmt.Errorf("%w %q: %s", ErrOrigenInvalido, origen, motivo)
	}
	if strings.Contains(origen, "*") {
		return invalido("no se admiten comodines; registra cada origen exacto")
	}
	u, err := url.Parse(strings.TrimSpace(origen))
	if err != nil {
		return invalido("no es una URL válida")
	}
	esquema := strings.ToLower(u.Scheme)
	if esquema != "http" && esquema != "https" {
		return invalido("debe empezar con https:// o http://")
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || (u.Path != "" && u.Path != "/") {
		return invalido("debe tener solo esquema, host y puerto (ej. https://miapp.com)")
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return invalido("falta el host")
	}
	puerto := u.Port()
	if (esquema == "https" && puerto == "443") || (esquema == "http" && puerto == "80") {
		puerto = ""
	}
	if puerto != "" {
		return esquema + "://" + net.JoinHostPort(host, puerto), nil
	}
	if strings.Contains(host, ":") { // IPv6
		host = "[" + host + "]"
	}
	return esquema + "://" + host, nil
}
