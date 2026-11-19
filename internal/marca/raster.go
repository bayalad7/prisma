package marca

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"strconv"
)

// punto en unidades de la rejilla de 32.
type punto struct{ x, y float64 }

// poligonos aplana un trazado SVG (M, L, H, V, A y Z absolutos; arcos circulares sin rotación) en
// contornos cerrados.
func poligonos(d string) ([][]punto, error) {
	var (
		res    [][]punto
		actual []punto
		pos    punto
		i      int
	)
	numero := func() (float64, error) {
		for i < len(d) && (d[i] == ' ' || d[i] == ',') {
			i++
		}
		j := i
		for j < len(d) && (d[j] == '-' && j == i || d[j] == '.' || d[j] >= '0' && d[j] <= '9') {
			j++
		}
		if j == i {
			return 0, fmt.Errorf("se esperaba un número en %q, posición %d", d, i)
		}
		v, err := strconv.ParseFloat(d[i:j], 64)
		i = j
		return v, err
	}
	cerrar := func() {
		if len(actual) > 2 {
			res = append(res, actual)
		}
		actual = nil
	}
	for i < len(d) {
		c := d[i]
		i++
		switch c {
		case ' ', ',':
			continue
		case 'M':
			cerrar()
			x, err := numero()
			if err != nil {
				return nil, err
			}
			y, err := numero()
			if err != nil {
				return nil, err
			}
			pos = punto{x, y}
			actual = []punto{pos}
		case 'L':
			x, err := numero()
			if err != nil {
				return nil, err
			}
			y, err := numero()
			if err != nil {
				return nil, err
			}
			pos = punto{x, y}
			actual = append(actual, pos)
		case 'H':
			x, err := numero()
			if err != nil {
				return nil, err
			}
			pos.x = x
			actual = append(actual, pos)
		case 'V':
			y, err := numero()
			if err != nil {
				return nil, err
			}
			pos.y = y
			actual = append(actual, pos)
		case 'A':
			var v [7]float64
			for k := range v {
				n, err := numero()
				if err != nil {
					return nil, err
				}
				v[k] = n
			}
			fin := punto{v[5], v[6]}
			actual = append(actual, arco(pos, fin, v[0], v[3] != 0, v[4] != 0)...)
			pos = fin
		case 'Z':
			cerrar()
		default:
			return nil, fmt.Errorf("comando %q no admitido en %q", c, d)
		}
	}
	cerrar()
	return res, nil
}

// arco devuelve los puntos de un arco circular de radio r de p1 a p2 (SVG 1.1, F.6.5), sin p1.
func arco(p1, p2 punto, r float64, grande, horario bool) []punto {
	x1 := (p1.x - p2.x) / 2
	y1 := (p1.y - p2.y) / 2
	if l := (x1*x1 + y1*y1) / (r * r); l > 1 {
		r *= math.Sqrt(l)
	}
	signo := 1.0
	if grande == horario {
		signo = -1
	}
	num := r*r*r*r - r*r*y1*y1 - r*r*x1*x1
	coef := signo * math.Sqrt(math.Max(0, num/(r*r*y1*y1+r*r*x1*x1)))
	cx := coef*y1 + (p1.x+p2.x)/2
	cy := coef*-x1 + (p1.y+p2.y)/2
	t1 := math.Atan2(p1.y-cy, p1.x-cx)
	dt := math.Atan2(p2.y-cy, p2.x-cx) - t1
	if horario && dt < 0 {
		dt += 2 * math.Pi
	}
	if !horario && dt > 0 {
		dt -= 2 * math.Pi
	}
	n := int(math.Ceil(math.Abs(dt) / (math.Pi / 64)))
	if n < 2 {
		n = 2
	}
	pts := make([]punto, 0, n)
	for k := 1; k <= n; k++ {
		t := t1 + dt*float64(k)/float64(n)
		pts = append(pts, punto{cx + r*math.Cos(t), cy + r*math.Sin(t)})
	}
	pts[len(pts)-1] = p2
	return pts
}

// capa es una forma lista para pintar: su trazado y su color, con el desplazamiento y la escala
// que la llevan de la rejilla de 32 a píxeles.
type capa struct {
	d      string
	color  color.NRGBA
	escala float64
	dx, dy float64
}

const submuestras = 8 // filas de muestreo por píxel; en horizontal la cobertura es exacta

// pintar rasteriza las capas en orden (cada una encima de la anterior) en una imagen de lado n.
func pintar(n int, capas []capa) (*image.NRGBA, error) {
	// Acumulado premultiplicado por píxel.
	r := make([]float64, n*n)
	g := make([]float64, n*n)
	b := make([]float64, n*n)
	a := make([]float64, n*n)
	cob := make([]float64, n*n)
	for _, c := range capas {
		polys, err := poligonos(c.d)
		if err != nil {
			return nil, err
		}
		clear(cob)
		cubrir(cob, n, polys, c)
		ca := float64(c.color.A) / 255
		for i, v := range cob {
			if v <= 0 {
				continue
			}
			v = math.Min(v, 1) * ca
			r[i] = float64(c.color.R)/255*v + r[i]*(1-v)
			g[i] = float64(c.color.G)/255*v + g[i]*(1-v)
			b[i] = float64(c.color.B)/255*v + b[i]*(1-v)
			a[i] = v + a[i]*(1-v)
		}
	}
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for i := range a {
		if a[i] <= 0 {
			continue
		}
		a8 := math.Round(a[i] * 255)
		img.Pix[i*4+0] = uint8(math.Round(math.Min(1, r[i]/a[i]) * 255))
		img.Pix[i*4+1] = uint8(math.Round(math.Min(1, g[i]/a[i]) * 255))
		img.Pix[i*4+2] = uint8(math.Round(math.Min(1, b[i]/a[i]) * 255))
		img.Pix[i*4+3] = uint8(a8)
	}
	return img, nil
}

// cubrir suma en cob la cobertura (0–1) de los polígonos, con la regla de relleno distinta de cero.
func cubrir(cob []float64, n int, polys [][]punto, c capa) {
	type cruce struct {
		x   float64
		dir int
	}
	peso := 1.0 / submuestras
	var cruces []cruce
	for fila := 0; fila < n*submuestras; fila++ {
		y := (float64(fila) + 0.5) / submuestras
		cruces = cruces[:0]
		for _, p := range polys {
			for k := range p {
				a := p[k]
				b := p[(k+1)%len(p)]
				ay, by := a.y*c.escala+c.dy, b.y*c.escala+c.dy
				if ay == by || y < math.Min(ay, by) || y >= math.Max(ay, by) {
					continue
				}
				ax, bx := a.x*c.escala+c.dx, b.x*c.escala+c.dx
				x := ax + (y-ay)*(bx-ax)/(by-ay)
				dir := 1
				if by < ay {
					dir = -1
				}
				cruces = append(cruces, cruce{x, dir})
			}
		}
		sort.Slice(cruces, func(i, j int) bool { return cruces[i].x < cruces[j].x })
		fila0 := (fila / submuestras) * n
		vuelta := 0
		for k := 0; k+1 < len(cruces); k++ {
			vuelta += cruces[k].dir
			if vuelta != 0 {
				tramo(cob[fila0:fila0+n], cruces[k].x, cruces[k+1].x, peso)
			}
		}
	}
}

// tramo suma peso a la fila entre x0 y x1, con la fracción exacta en los píxeles de los extremos.
func tramo(fila []float64, x0, x1, peso float64) {
	n := float64(len(fila))
	x0, x1 = math.Max(0, x0), math.Min(n, x1)
	if x1 <= x0 {
		return
	}
	i0, i1 := int(x0), int(x1)
	if i0 == i1 {
		fila[i0] += (x1 - x0) * peso
		return
	}
	fila[i0] += (float64(i0+1) - x0) * peso
	for i := i0 + 1; i < i1 && i < len(fila); i++ {
		fila[i] += peso
	}
	if i1 < len(fila) {
		fila[i1] += (x1 - float64(i1)) * peso
	}
}

func hexA(h string) color.NRGBA {
	v, _ := strconv.ParseUint(h[1:], 16, 32)
	return color.NRGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}
}
