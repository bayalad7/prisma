package escpos

import (
	"bytes"
	"image"
	"image/color"
	"image/png"

	"github.com/skip2/go-qrcode"

	"github.com/bayalad7/prisma/internal/contrato"
)

// Mapa es una imagen de 1 bit: true = punto impreso (negro).
type Mapa struct {
	Ancho, Alto int
	Pix         []bool
}

func NuevoMapa(ancho, alto int) *Mapa {
	return &Mapa{Ancho: ancho, Alto: alto, Pix: make([]bool, ancho*alto)}
}

func (m *Mapa) En(x, y int) bool { return m.Pix[y*m.Ancho+x] }

func (m *Mapa) Poner(x, y int, v bool) { m.Pix[y*m.Ancho+x] = v }

// PNG codifica el mapa como PNG en escala de grises (para la previsualización).
func (m *Mapa) PNG() []byte {
	img := image.NewGray(image.Rect(0, 0, m.Ancho, m.Alto))
	for i, v := range m.Pix {
		if !v {
			img.Pix[i] = 0xFF
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

// convertirPNG decodifica un PNG, lo compone sobre blanco, lo reduce (nunca lo agranda) hasta
// anchoMax puntos y lo pasa a 1 bit con difusión de error Floyd–Steinberg.
func convertirPNG(datos []byte, anchoMax int) (*Mapa, error) {
	img, err := png.Decode(bytes.NewReader(datos))
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	tw := min(w, max(anchoMax, 1))
	th := max(1, (h*tw+w/2)/w)

	// Luminancia 0 (negro) a 1 (blanco) promediando el área de origen de cada punto.
	gris := make([]float64, tw*th)
	for y := range th {
		y0, y1 := y*h/th, max((y+1)*h/th, y*h/th+1)
		for x := range tw {
			x0, x1 := x*w/tw, max((x+1)*w/tw, x*w/tw+1)
			var suma float64
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					suma += luminancia(img.At(b.Min.X+sx, b.Min.Y+sy))
				}
			}
			gris[y*tw+x] = suma / float64((y1-y0)*(x1-x0))
		}
	}

	m := NuevoMapa(tw, th)
	for y := range th {
		for x := range tw {
			viejo := gris[y*tw+x]
			nuevo := 1.0
			if viejo < 0.5 {
				nuevo = 0
				m.Poner(x, y, true)
			}
			e := viejo - nuevo
			difundir := func(dx, dy int, f float64) {
				if nx, ny := x+dx, y+dy; nx >= 0 && nx < tw && ny < th {
					gris[ny*tw+nx] += e * f
				}
			}
			difundir(1, 0, 7.0/16)
			difundir(-1, 1, 3.0/16)
			difundir(0, 1, 5.0/16)
			difundir(1, 1, 1.0/16)
		}
	}
	return m, nil
}

// luminancia compone el color sobre blanco (lo transparente es blanco).
func luminancia(c color.Color) float64 {
	r, g, b, a := c.RGBA() // premultiplicado, 0..0xFFFF
	blanco := float64(0xFFFF - a)
	return (0.299*(float64(r)+blanco) + 0.587*(float64(g)+blanco) + 0.114*(float64(b)+blanco)) / 0xFFFF
}

// mapaBarras dibuja los módulos con el ancho de módulo y el alto indicados.
func mapaBarras(mods []bool, modulo, alto int) *Mapa {
	m := NuevoMapa(len(mods)*modulo, alto)
	for i, v := range mods {
		if !v {
			continue
		}
		for x := i * modulo; x < (i+1)*modulo; x++ {
			for y := range alto {
				m.Poner(x, y, true)
			}
		}
	}
	return m
}

var nivelesQR = map[string]qrcode.RecoveryLevel{"L": qrcode.Low, "M": qrcode.Medium, "Q": qrcode.High, "H": qrcode.Highest}

// modulosQR genera la matriz del QR (sin zona de silencio).
func modulosQR(valor, correccion string) ([][]bool, error) {
	q, err := qrcode.New(valor, nivelesQR[correccion])
	if err != nil {
		return nil, err
	}
	q.DisableBorder = true
	return q.Bitmap(), nil
}

// MapaQR dibuja el QR con `tamano` puntos por módulo.
func MapaQR(valor, correccion string, tamano int) (*Mapa, error) {
	mods, err := modulosQR(valor, correccion)
	if err != nil {
		return nil, err
	}
	n := len(mods)
	m := NuevoMapa(n*tamano, n*tamano)
	for y, fila := range mods {
		for x, v := range fila {
			if !v {
				continue
			}
			for dy := range tamano {
				for dx := range tamano {
					m.Poner(x*tamano+dx, y*tamano+dy, true)
				}
			}
		}
	}
	return m, nil
}

// MapaBarras dibuja un código de barras con el ancho de módulo y alto indicados.
func MapaBarras(simbologia, valor string, modulo, alto int) (*Mapa, error) {
	mods, err := Modulos(simbologia, valor)
	if err != nil {
		return nil, err
	}
	return mapaBarras(mods, modulo, alto), nil
}

// desplazamiento devuelve el margen izquierdo para alinear algo de `ancho` puntos en `total`.
func desplazamiento(ancho, total int, a contrato.Alineacion) int {
	falta := max(total-ancho, 0)
	switch a {
	case contrato.Centro:
		return falta / 2
	case contrato.Derecha:
		return falta
	}
	return 0
}
