package escpos

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/unicode/norm"
)

// Páginas de códigos soportadas (impresoras.pagina_codigos).
var paginas = map[string]*charmap.Charmap{
	"CP437":   charmap.CodePage437,
	"CP850":   charmap.CodePage850,
	"CP858":   charmap.CodePage858,
	"WPC1252": charmap.Windows1252,
}

// PaginaPorComando traduce el n de ESC t n a la página de códigos en impresoras compatibles con Epson.
// La usan las herramientas de desarrollo para decodificar texto.
func PaginaPorComando(n int) string {
	switch n {
	case 2:
		return "CP850"
	case 16:
		return "WPC1252"
	case 19:
		return "CP858"
	}
	return "CP437"
}

// DecodificarTexto convierte bytes de la página indicada a texto (herramientas de desarrollo).
func DecodificarTexto(pagina string, b []byte) string {
	tabla, ok := paginas[strings.ToUpper(pagina)]
	if !ok {
		tabla = charmap.CodePage437
	}
	rs := make([]rune, len(b))
	for i, c := range b {
		if c < 0x80 {
			rs[i] = rune(c)
		} else {
			rs[i] = tabla.DecodeByte(c)
		}
	}
	return string(rs)
}

// PaginasSoportadas devuelve los nombres válidos para impresoras.pagina_codigos.
func PaginasSoportadas() []string {
	var n []string
	for k := range paginas {
		n = append(n, k)
	}
	sort.Strings(n)
	return n
}

// Equivalentes de un solo carácter para signos tipográficos frecuentes sin acento que quitar.
var equivalentes = map[rune]rune{
	'“': '"', '”': '"', '„': '"', '«': '"', '»': '"',
	'‘': '\'', '’': '\'', '‚': '\'', '´': '\'',
	'–': '-', '—': '-', '−': '-', '‐': '-',
	'…': '.', '•': '*', '·': '.',
	' ': ' ', ' ': ' ', ' ': ' ',
}

// codificador convierte texto UTF-8 a la página de códigos de la impresora, un byte por carácter.
type codificador struct {
	pagina      string
	tabla       *charmap.Charmap
	sustituidos map[rune]rune // carácter original → sustituto ('?' si no hubo equivalente)
}

func nuevoCodificador(pagina string) (*codificador, error) {
	tabla, ok := paginas[strings.ToUpper(pagina)]
	if !ok {
		return nil, fmt.Errorf("página de códigos %q no soportada; usa una de %v", pagina, PaginasSoportadas())
	}
	return &codificador{pagina: strings.ToUpper(pagina), tabla: tabla, sustituidos: map[rune]rune{}}, nil
}

func (c *codificador) codificable(r rune) (byte, bool) {
	if r >= 0x20 && r < 0x7F {
		return byte(r), true
	}
	b, ok := c.tabla.EncodeRune(r)
	if !ok || b < 0x80 {
		return 0, false
	}
	return b, true
}

// preparar normaliza el texto (NFC), descarta marcas combinantes sueltas y sustituye lo que la
// página no puede representar: primero por la versión sin acento, si no por '?'. El resultado
// tiene exactamente un rune por columna impresa, así que la disposición mide en runas.
func (c *codificador) preparar(s string) []rune {
	var res []rune
	for _, r := range norm.NFC.String(s) {
		if unicode.Is(unicode.Mn, r) {
			c.sustituidos[r] = 0 // marca combinante sin carácter base: se descarta
			continue
		}
		if _, ok := c.codificable(r); ok || r == '\n' {
			res = append(res, r)
			continue
		}
		sustituto := c.sustituto(r)
		c.sustituidos[r] = sustituto
		res = append(res, sustituto)
	}
	return res
}

func (c *codificador) sustituto(r rune) rune {
	if e, ok := equivalentes[r]; ok {
		if _, ok := c.codificable(e); ok {
			return e
		}
	}
	// Versión sin acento: la primera runa de la descomposición canónica (Á → A, ñ → n).
	if base, _ := utf8.DecodeRuneInString(norm.NFD.String(string(r))); base != r {
		if _, ok := c.codificable(base); ok {
			return base
		}
	}
	return '?'
}

// codificar convierte runas ya preparadas a bytes.
func (c *codificador) codificar(rs []rune) []byte {
	out := make([]byte, 0, len(rs))
	for _, r := range rs {
		b, ok := c.codificable(r)
		if !ok {
			b = '?'
		}
		out = append(out, b)
	}
	return out
}

// advertencia resume los caracteres sustituidos (nil si no hubo).
func (c *codificador) advertencia() *Advertencia {
	if len(c.sustituidos) == 0 {
		return nil
	}
	var partes []string
	for r, s := range c.sustituidos {
		switch s {
		case 0:
			partes = append(partes, fmt.Sprintf("U+%04X (marca suelta) descartada", r))
		default:
			partes = append(partes, fmt.Sprintf("%c→%c", r, s))
		}
	}
	sort.Strings(partes)
	return &Advertencia{
		Codigo:  AdvCaracterSustituido,
		Mensaje: fmt.Sprintf("La página de códigos %s no tiene algunos caracteres; se imprimieron así: %s.", c.pagina, strings.Join(partes, ", ")),
	}
}
