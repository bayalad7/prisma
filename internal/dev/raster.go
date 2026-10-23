package dev

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"sync"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/bayalad7/prisma/internal/escpos"
)

// PuntosPorMM es la resolución de las impresoras térmicas de tickets: 203 ppp (8 puntos por mm).
const PuntosPorMM = 203.0 / 25.4

// Colores del raster: papel térmico, tinta y las guías (corte, cajón), que no son parte del ticket.
var paletaRaster = color.Palette{
	color.RGBA{0xfb, 0xf9, 0xf4, 0xff},
	color.RGBA{0x1c, 0x1c, 0x22, 0xff},
	color.RGBA{0x96, 0x92, 0x88, 0xff},
}

const (
	pPapel = iota
	pTinta
	pGuia
)

const (
	margenVertical = 24 // puntos arriba y abajo del ticket
	interlineado   = 8  // puntos entre renglones: con la fuente A da 32 por renglón (≈ 1/6 de pulgada)
	altoCorte      = 36
)

// fuenteRaster es una fuente monoespaciada dibujada en celdas de ancho × alto puntos, como las
// fuentes internas de la impresora. Los glifos se guardan ya convertidos a 1 bit.
type fuenteRaster struct {
	ancho, alto int
	cara        font.Face
	base        int // renglón base dentro de la celda

	mu     sync.Mutex
	glifos map[rune][]bool
}

var (
	gomonoUna sync.Once
	gomonoTTF *opentype.Font
	fuentesMu sync.Mutex
	fuentes   = map[[2]int]*fuenteRaster{}
)

// fuente devuelve la fuente para celdas de ancho × alto puntos (se crea una vez por tamaño).
func fuente(ancho, alto int) *fuenteRaster {
	fuentesMu.Lock()
	defer fuentesMu.Unlock()
	if f, ok := fuentes[[2]int{ancho, alto}]; ok {
		return f
	}
	gomonoUna.Do(func() { gomonoTTF, _ = opentype.Parse(gomono.TTF) })
	// El avance de Go Mono es 0.6 em: el tamaño que llena la celda a lo ancho.
	cara, err := opentype.NewFace(gomonoTTF, &opentype.FaceOptions{Size: float64(ancho) / 0.6, DPI: 72, Hinting: font.HintingFull})
	f := &fuenteRaster{ancho: ancho, alto: alto, glifos: map[rune][]bool{}}
	if err == nil {
		f.cara = cara
		m := cara.Metrics()
		asc, desc := m.Ascent.Ceil(), m.Descent.Ceil()
		f.base = (alto-(asc+desc))/2 + asc
	}
	fuentes[[2]int{ancho, alto}] = f
	return f
}

// glifo devuelve los puntos del carácter en su celda (fila por fila).
func (f *fuenteRaster) glifo(r rune) []bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if g, ok := f.glifos[r]; ok {
		return g
	}
	g := make([]bool, f.ancho*f.alto)
	if f.cara != nil && r != ' ' {
		a := image.NewAlpha(image.Rect(0, 0, f.ancho, f.alto))
		avance, ok := f.cara.GlyphAdvance(r)
		if !ok {
			r, avance = '?', fixed.I(f.ancho)
		}
		x := (fixed.I(f.ancho) - avance) / 2
		d := font.Drawer{Dst: a, Src: image.Opaque, Face: f.cara, Dot: fixed.Point26_6{X: x, Y: fixed.I(f.base)}}
		d.DrawString(string(r))
		for i, v := range a.Pix {
			g[i] = v >= 0x80
		}
	}
	f.glifos[r] = g
	return g
}

// celdas devuelve la fuente de un tramo: A (ancho/columnas, el doble de alto) o B (17/9 del ancho).
func celdas(p *Pagina, fuenteB bool) *fuenteRaster {
	if fuenteB {
		a := max(p.AnchoPx/max(p.ColumnasB, 1), 4)
		return fuente(a, int(math.Round(float64(a)*17/9)))
	}
	a := max(p.AnchoPx/max(p.Columnas, 1), 4)
	return fuente(a, 2*a)
}

type lienzo struct {
	img      *image.Paletted
	x0       int // primer punto imprimible
	anchoImp int
}

func (l *lienzo) punto(x, y int, c uint8) {
	if x >= 0 && y >= 0 && x < l.img.Rect.Dx() && y < l.img.Rect.Dy() {
		l.img.Pix[y*l.img.Stride+x] = c
	}
}

// AnchoPapel devuelve el ancho del papel en puntos (80 mm → 639, 58 mm → 464).
func AnchoPapel(anchoMM int) int { return int(math.Round(float64(anchoMM) * PuntosPorMM)) }

// Rasterizar dibuja la página como saldría en el papel de una impresora térmica de 203 ppp:
// el texto con la fuente y el tamaño que recibió, las imágenes, QR y códigos de barras punto por
// punto, y guías en gris para los cortes y el cajón. anchoMM es el ancho del papel.
func Rasterizar(p *Pagina, anchoMM int) *image.Paletted {
	papel := max(AnchoPapel(anchoMM), p.AnchoPx)
	alto := margenVertical*2 + recorrer(p, nil)
	l := &lienzo{img: image.NewPaletted(image.Rect(0, 0, papel, alto), paletaRaster), x0: (papel - p.AnchoPx) / 2, anchoImp: p.AnchoPx}
	recorrer(p, l)
	return l.img
}

// RasterPNG es Rasterizar codificado como PNG.
func RasterPNG(p *Pagina, anchoMM int) []byte {
	var b bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	enc.Encode(&b, Rasterizar(p, anchoMM))
	return b.Bytes()
}

// recorrer calcula el alto de cada elemento y, si l no es nil, lo dibuja. Devuelve el alto total.
func recorrer(p *Pagina, l *lienzo) int {
	y := margenVertical
	for _, el := range p.Elementos {
		switch el.Tipo {
		case ElemLinea:
			alto := celdas(p, false).alto
			ancho := 0
			for _, r := range el.Runs {
				f := celdas(p, r.FuenteB)
				alto = max(alto, f.alto*r.Alto)
				ancho += utf8.RuneCountInString(r.Texto) * f.ancho * r.Ancho
			}
			if l != nil {
				x := l.x0 + desplazar(ancho, p.AnchoPx, el.Alineacion)
				for _, r := range el.Runs {
					x = l.tramo(p, r, x, y+alto-celdas(p, r.FuenteB).alto*r.Alto)
				}
			}
			y += alto + interlineado
		case ElemImagen:
			if l != nil {
				l.mapa(el.Mapa, l.x0+desplazar(el.Mapa.Ancho, p.AnchoPx, el.Alineacion), y)
			}
			y += el.Mapa.Alto + interlineado
		case ElemCorte:
			if l != nil {
				etiqueta := "corte"
				if el.Parcial {
					etiqueta = "corte parcial"
				}
				l.guia(y+altoCorte/2, el.Parcial, etiqueta)
			}
			y += altoCorte
		case ElemCajon:
			if l != nil {
				l.guia(y+altoCorte/2, true, "se abre el cajón")
			}
			y += altoCorte
		}
	}
	return y - margenVertical
}

// tramo dibuja un tramo de texto desde (x, y) y devuelve dónde termina.
func (l *lienzo) tramo(p *Pagina, r Run, x, y int) int {
	f := celdas(p, r.FuenteB)
	w, h := f.ancho*r.Ancho, f.alto*r.Alto
	fin := l.x0 + l.anchoImp
	for _, c := range r.Texto {
		if x+w > fin+1 {
			break // la impresora no imprime fuera del área
		}
		g := f.glifo(c)
		tinta := uint8(pTinta)
		if r.Invertido { // fondo negro y letras en el color del papel
			tinta = pPapel
			for yy := range h {
				for xx := range w {
					l.punto(x+xx, y+yy, pTinta)
				}
			}
		}
		for yy := range h {
			for xx := range w {
				gx, gy := xx/r.Ancho, yy/r.Alto
				on := g[gy*f.ancho+gx]
				// Negrita: la impresora repite cada punto uno a la derecha (doble golpe).
				if !on && r.Negrita && gx > 0 {
					on = g[gy*f.ancho+gx-1]
				}
				if on {
					l.punto(x+xx, y+yy, tinta)
				}
			}
		}
		if r.Subrayado {
			for xx := range w {
				l.punto(x+xx, y+h-2, tinta)
				l.punto(x+xx, y+h-1, tinta)
			}
		}
		x += w
	}
	return x
}

func (l *lienzo) mapa(m *escpos.Mapa, x, y int) {
	for yy := range m.Alto {
		for xx := range m.Ancho {
			if m.En(xx, yy) {
				l.punto(x+xx, y+yy, pTinta)
			}
		}
	}
}

// guia dibuja una línea de puntos en gris a lo ancho del papel con una etiqueta a la derecha.
// Marca dónde corta la cuchilla o cuándo se abre el cajón; no es parte de lo impreso.
func (l *lienzo) guia(y int, corta bool, etiqueta string) {
	f := fuente(9, 17)
	ancho := l.img.Rect.Dx()
	largo, hueco := 10, 6
	if corta {
		largo, hueco = 4, 6
	}
	texto := utf8.RuneCountInString(etiqueta) * f.ancho
	xTexto := ancho - texto - 8
	for x := 0; x < xTexto-8; x++ {
		if x%(largo+hueco) < largo {
			l.punto(x, y, pGuia)
			l.punto(x, y+1, pGuia)
		}
	}
	x := xTexto
	for _, c := range etiqueta {
		g := f.glifo(c)
		for yy := range f.alto {
			for xx := range f.ancho {
				if g[yy*f.ancho+xx] {
					l.punto(x+xx, y-f.alto/2+yy, pGuia)
				}
			}
		}
		x += f.ancho
	}
}
