package escpos

import (
	"bytes"
	"fmt"

	"github.com/bayalad7/prisma/internal/almacen"
)

// VistaPrevia1Bit convierte un PNG a 1 bit tal como se imprimiría con anchoMax puntos y lo
// devuelve como PNG (para mostrarlo en el panel).
func VistaPrevia1Bit(png []byte, anchoMax int) ([]byte, error) {
	m, err := convertirPNG(png, anchoMax)
	if err != nil {
		return nil, err
	}
	return m.PNG(), nil
}

// PaginaPrueba es una de las páginas de códigos que se imprimen para elegir la correcta.
type PaginaPrueba struct {
	Comando  int
	Pagina   string
	Etiqueta string
}

// PaginasPrueba son las páginas que prueba el panel (números de ESC t en impresoras Epson).
var PaginasPrueba = []PaginaPrueba{
	{0, "CP437", "PC437 (EE. UU.)"},
	{2, "CP850", "PC850 (Multilingüe)"},
	{16, "WPC1252", "WPC1252 (Windows)"},
	{19, "CP858", "PC858 (Multilingüe con €)"},
}

// LineaPruebaPaginas es el texto que debe leerse bien con la página correcta.
const LineaPruebaPaginas = "áéíóú ÁÉÍÓÚ ñÑ üÜ ¿¡ € °"

// PruebaPaginas genera un ticket que imprime la misma línea con cada página de códigos de
// PaginasPrueba, cada una precedida por su comando, para elegir en el papel la que se lee bien.
func PruebaPaginas(perfil almacen.Impresora) ([]byte, error) {
	var b bytes.Buffer
	escribir := func(bs ...byte) { b.Write(bs) }
	escribir(esc, '@')
	escribir(esc, 'a', 1, esc, 'E', 1)
	b.WriteString("PRUEBA DE PAGINAS DE CODIGOS\n")
	escribir(esc, 'E', 0)
	b.WriteString("Elige la linea que se lee bien:\n\n")
	escribir(esc, 'a', 0)
	for _, p := range PaginasPrueba {
		cod, err := nuevoCodificador(p.Pagina)
		if err != nil {
			return nil, err
		}
		escribir(esc, 't', byte(p.Comando))
		fmt.Fprintf(&b, "ESC t %-2d %s\n", p.Comando, p.Pagina)
		b.Write(cod.codificar(cod.preparar("  " + LineaPruebaPaginas)))
		escribir(lf, lf)
	}
	escribir(esc, 't', byte(perfil.ComandoPaginaCodigos))
	if perfil.SoportaCorte {
		escribir(gs, 'V', 65, 0)
	} else {
		escribir(esc, 'd', 4)
	}
	return b.Bytes(), nil
}
