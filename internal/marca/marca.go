// Package marca guarda la geometría oficial del logo de PRISMA (Conversión, ADR 0022) y genera
// desde ella los archivos de la marca: SVG, PNG, .ico y .icns. Es la única fuente: el logo del
// panel (internal/admin/web/sistema.js), el favicon y los iconos de los instaladores salen de aquí
// con «go test ./internal/marca -actualizar», y la misma prueba falla si alguno quedó viejo.
//
// El isotipo está en una rejilla de 32 × 32: arriba el documento que manda la app (con la esquina
// doblada), en medio el puente en acento que lo atraviesa (PRISMA) y abajo el ticket dentado.
package marca

import (
	"fmt"
	"strings"
)

// Colores de la marca (paleta «Magenta de proceso»). Deben coincidir con los tokens --marca-* de
// internal/admin/web/panel.css; lo revisa TestColoresDelPanelCoincidenConLaMarca.
const (
	TintaClaro   = "#171717"
	AcentoClaro  = "#be185d"
	TintaOscuro  = "#ededed"
	AcentoOscuro = "#f472b6"
	// Placa del icono de app: el color secundario (el 30 %) con la tinta clara encima.
	Placa       = "#1a1a1a"
	TintaPlaca  = "#f2f2f2"
	AcentoPlaca = "#f472b6"
)

// Figura es una forma rellena del logo: su trazado (solo M, L, H, V, A y Z absolutos) y si va en
// acento o en tinta.
type Figura struct {
	D      string
	Acento bool
}

// Isotipo completo: para 24 px o más.
var Isotipo = []Figura{
	{D: "M8.5 2H19L26 9V12H6V4.5A2.5 2.5 0 0 1 8.5 2Z"},
	{D: "M4.5 14H27.5A2 2 0 0 1 27.5 18H4.5A2 2 0 0 1 4.5 14Z", Acento: true},
	{D: "M6 20H26V29L22.67 26.5L19.33 29L16 26.5L12.67 29L9.33 26.5L6 29Z"},
}

// Bandeja: la variante simplificada para 16 px (bandeja del sistema, favicon chico): el puente
// más grueso, la esquina más marcada y dos muescas en lugar de tres.
var Bandeja = []Figura{
	{D: "M8.5 1.5H18L26 9.5V12H6V4A2.5 2.5 0 0 1 8.5 1.5Z"},
	{D: "M4 13.5H28A2.5 2.5 0 0 1 28 18.5H4A2.5 2.5 0 0 1 4 13.5Z", Acento: true},
	{D: "M6 20H26V30.5L21 27.5L16 30.5L11 27.5L6 30.5Z"},
}

// placa: cuadrado de 32 con radio 7 (7/32 del lado, como el icono de Aula).
const placa = "M7 0H25A7 7 0 0 1 32 7V25A7 7 0 0 1 25 32H7A7 7 0 0 1 0 25V7A7 7 0 0 1 7 0Z"

// Letras de «prisma», dibujadas (no escritas) para que se vean igual sin fuentes: trazo 4.4, curvas
// de radio 6, altura x de 12.8 a 29.2 y 3.6 de separación entre letras. Ocupan 101 × 40.
const (
	letrasTrazos = "M2.2 12.8V37.8M22.2 12.8V29.2M22.2 21A6 6 0 0 1 28.2 15H29M34.8 12.8V29.2" +
		"M50.8 15H45.8A3 3 0 0 0 45.8 21H47.8A3 3 0 0 1 47.8 27H42.8" +
		"M58.8 12.8V29.2M58.8 20A5 5 0 0 1 68.8 20V29.2M68.8 20A5 5 0 0 1 78.8 20V29.2M98.8 12.8V29.2"
	letrasCirculos = `<circle cx="8.2" cy="21" r="6"/><circle cx="92.8" cy="21" r="6"/>`
)

// Tema de color de una versión del logo.
type Tema struct{ Tinta, Acento string }

var (
	Claro      = Tema{TintaClaro, AcentoClaro}
	Oscuro     = Tema{TintaOscuro, AcentoOscuro}
	UnaTinta   = Tema{"currentColor", "currentColor"}
	SobrePlaca = Tema{TintaPlaca, AcentoPlaca}
)

func figurasSVG(figs []Figura, t Tema) string {
	var b strings.Builder
	for _, f := range figs {
		color := t.Tinta
		if f.Acento {
			color = t.Acento
		}
		fmt.Fprintf(&b, `<path fill="%s" d="%s"/>`, color, f.D)
	}
	return b.String()
}

const cabecera = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="%s" role="img" aria-label="PRISMA">`

// SVGIsotipo devuelve el isotipo (o su variante de bandeja) como SVG limpio de 32 × 32.
func SVGIsotipo(figs []Figura, t Tema) string {
	return fmt.Sprintf(cabecera, "0 0 32 32") + figurasSVG(figs, t) + "</svg>\n"
}

// SVGHorizontal devuelve el isotipo con «prisma» a la derecha (152 × 40).
func SVGHorizontal(t Tema) string {
	return fmt.Sprintf(cabecera, "0 0 152 40") +
		`<g transform="scale(1.25)">` + figurasSVG(Isotipo, t) + `</g>` +
		fmt.Sprintf(`<g transform="translate(50 0)" fill="none" stroke="%s" stroke-width="4.4"><path d="%s"/>%s</g>`,
			t.Tinta, letrasTrazos, letrasCirculos) +
		fmt.Sprintf(`<circle cx="84.8" cy="6.4" r="2.4" fill="%s"/>`, t.Tinta) +
		"</svg>\n"
}

// SVGIcono devuelve el icono de app: el isotipo claro sobre la placa secundaria, con 3 unidades de
// margen. Es el mismo en fondos claros y oscuros (favicon, menú Inicio, lanzadores).
func SVGIcono() string {
	return fmt.Sprintf(cabecera, "0 0 32 32") +
		fmt.Sprintf(`<path fill="%s" d="%s"/>`, Placa, placa) +
		`<g transform="translate(3 3) scale(0.8125)">` + figurasSVG(Isotipo, SobrePlaca) + `</g></svg>` + "\n"
}
