package dev

import (
	"encoding/base64"
	"fmt"
	"html"
	"strings"
	"unicode/utf8"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/contrato"
	"github.com/bayalad7/prisma/internal/escpos"
)

// PerfilPrueba devuelve un perfil típico de 58 u 80 mm con todas las capacidades y CP858.
func PerfilPrueba(anchoMM int) (almacen.Impresora, error) {
	p := almacen.Impresora{
		Alias: fmt.Sprintf("prueba-%d", anchoMM), Nombre: "Impresora virtual", Tipo: almacen.ConexionArchivo, Destino: "-",
		AnchoMM: anchoMM, PaginaCodigos: "CP858", ComandoPaginaCodigos: 19,
		SoportaQRNativo: true, SoportaCorte: true, SoportaCajon: true, SoportaImagen: true, SoportaCodigoBarrasNativo: true, Activa: true,
	}
	switch anchoMM {
	case 80:
		p.Columnas, p.ColumnasFuenteB, p.AnchoPx = 48, 64, 576
	case 58:
		p.Columnas, p.ColumnasFuenteB, p.AnchoPx = 32, 42, 384
	default:
		return p, fmt.Errorf("ancho %d mm no soportado; usa 58 u 80", anchoMM)
	}
	return p, nil
}

// Previsualizar renderiza el documento y devuelve un fragmento HTML autocontenido que simula
// el ticket impreso (lo usan el panel y el SDK), junto con las advertencias del renderizado.
func Previsualizar(doc contrato.Documento, perfil almacen.Impresora, recursos escpos.Recursos) (string, []escpos.Advertencia, error) {
	datos, advs, err := escpos.Renderizar(doc, perfil, recursos)
	if err != nil {
		return "", nil, err
	}
	return HTML(Simular(datos, perfil)), advs, nil
}

func anchoRun(r Run) int { return utf8.RuneCountInString(r.Texto) * r.Ancho }

func altoLinea(el Elemento) int {
	alto := 1
	for _, r := range el.Runs {
		alto = max(alto, r.Alto)
	}
	return alto
}

func desplazar(ocupa, total int, a contrato.Alineacion) int {
	falta := max(total-ocupa, 0)
	switch a {
	case contrato.Centro:
		return falta / 2
	case contrato.Derecha:
		return falta
	}
	return 0
}

// VistaTexto dibuja la página en texto para la terminal. Los caracteres de tamaño ancho n ocupan
// n columnas, la fuente B se muestra con el mismo ancho que la A y las imágenes se aproximan con
// bloques. Con color, la negrita, el subrayado y el invertido usan secuencias ANSI.
func VistaTexto(p *Pagina, color bool) string {
	w := p.Columnas
	var b strings.Builder
	borde := strings.Repeat("─", w)
	b.WriteString("┌" + borde + "┐\n")
	fila := func(contenido string, ocupa int) {
		b.WriteString("│" + contenido + strings.Repeat(" ", max(w-ocupa, 0)) + "│\n")
	}
	continuada := false
	cerrar := func() string {
		if continuada {
			b.WriteString("↳ La fuente B es más angosta que la A: en el papel esa línea es una sola.\n")
		}
		return b.String()
	}
	for k, el := range p.Elementos {
		switch el.Tipo {
		case ElemLinea:
			var celdas []celda
			for i := range el.Runs {
				celdas = append(celdas, celdasRun(&el.Runs[i])...)
			}
			if len(celdas) <= w {
				izq := desplazar(len(celdas), w, el.Alineacion)
				fila(strings.Repeat(" ", izq)+pintar(celdas, color), izq+len(celdas))
			} else {
				// Solo ocurre con fuente B, que cabe en el papel pero no en la cuadrícula de la A.
				continuada = true
				fila(pintar(celdas[:w], color), w)
				for resto := celdas[w:]; len(resto) > 0; {
					n := min(len(resto), w-1)
					fila("↳"+pintar(resto[:n], color), n+1)
					resto = resto[n:]
				}
			}
			for range altoLinea(el) - 1 {
				fila("", 0)
			}
		case ElemImagen:
			for _, l := range bloques(el.Mapa, p.AnchoPx, w, el.Alineacion) {
				fila(l, w)
			}
		case ElemCorte:
			marca := "✂ - - - corte total "
			if el.Parcial {
				marca = "✂ · · · corte parcial "
			}
			b.WriteString("└" + padDerecha(marca, w, "─") + "┘\n")
			if k == len(p.Elementos)-1 {
				return cerrar()
			}
			b.WriteString("┌" + borde + "┐\n")
		case ElemCajon:
			nota := fmt.Sprintf("[ abre cajón · pin %d ]", el.Pin)
			n := utf8.RuneCountInString(nota)
			izq := desplazar(n, w, contrato.Centro)
			fila(strings.Repeat(" ", izq)+nota, izq+n)
		}
	}
	b.WriteString("└" + borde + "┘\n")
	return cerrar()
}

func padDerecha(s string, w int, relleno string) string {
	n := utf8.RuneCountInString(s)
	if n >= w {
		return s
	}
	return s + strings.Repeat(relleno, w-n)
}

// celda es una columna de la terminal con el estilo del tramo al que pertenece.
type celda struct {
	r   rune
	run *Run
}

// celdasRun expande un tramo: un carácter de ancho n ocupa n columnas.
func celdasRun(r *Run) []celda {
	var cs []celda
	for _, c := range r.Texto {
		cs = append(cs, celda{c, r})
		for range r.Ancho - 1 {
			cs = append(cs, celda{' ', r})
		}
	}
	return cs
}

// pintar convierte celdas a texto, con secuencias ANSI para negrita, subrayado e invertido.
func pintar(cs []celda, color bool) string {
	var b strings.Builder
	var actual string
	for _, c := range cs {
		if color {
			if sgr := sgrRun(c.run); sgr != actual {
				if actual != "" {
					b.WriteString("\x1b[0m")
				}
				b.WriteString(sgr)
				actual = sgr
			}
		}
		b.WriteRune(c.r)
	}
	if actual != "" {
		b.WriteString("\x1b[0m")
	}
	return b.String()
}

func sgrRun(r *Run) string {
	var sgr []string
	if r.Negrita {
		sgr = append(sgr, "1")
	}
	if r.Subrayado {
		sgr = append(sgr, "4")
	}
	if r.Invertido {
		sgr = append(sgr, "7")
	}
	if len(sgr) == 0 {
		return ""
	}
	return "\x1b[" + strings.Join(sgr, ";") + "m"
}

// bloques aproxima un mapa de bits con medios bloques: cada carácter cubre (anchoPx/cols)
// puntos de ancho y el doble de alto, partido en dos mitades.
func bloques(m *escpos.Mapa, anchoPx, cols int, a contrato.Alineacion) []string {
	celda := max(anchoPx/cols, 1)
	desde := desplazar(m.Ancho, anchoPx, a)
	negro := func(cx, y0 int) bool {
		total, n := 0, 0
		for y := y0; y < min(y0+celda, m.Alto); y++ {
			for x := cx*celda - desde; x < (cx+1)*celda-desde; x++ {
				if x < 0 || x >= m.Ancho {
					continue
				}
				total++
				if m.En(x, y) {
					n++
				}
			}
		}
		return total > 0 && n*2 >= total
	}
	var lineas []string
	for y := 0; y < m.Alto; y += 2 * celda {
		var l strings.Builder
		for cx := range cols {
			arriba, abajo := negro(cx, y), negro(cx, y+celda)
			switch {
			case arriba && abajo:
				l.WriteRune('█')
			case arriba:
				l.WriteRune('▀')
			case abajo:
				l.WriteRune('▄')
			default:
				l.WriteRune(' ')
			}
		}
		lineas = append(lineas, l.String())
	}
	return lineas
}

// HTML dibuja la página como un fragmento HTML autocontenido. Todo el texto del documento se
// escapa: el panel lo muestra y el documento viene de terceros.
func HTML(p *Pagina) string {
	celdaA := float64(p.AnchoPx) / float64(p.Columnas)
	celdaB := float64(p.AnchoPx) / float64(p.ColumnasB)
	const altoA, altoB = 24.0, 17.0 // alto de línea en puntos para la fuente A y B

	var b strings.Builder
	fmt.Fprintf(&b, `<div class="prisma-ticket" style="width:%dpx">`, p.AnchoPx)
	b.WriteString(`<style>.prisma-ticket{box-sizing:content-box;background:#fff;color:#000;padding:16px;` +
		`font-family:ui-monospace,Consolas,"Courier New",monospace;box-shadow:0 1px 4px rgba(0,0,0,.25)}` +
		`.prisma-ticket .l{display:flex;white-space:pre;overflow:visible}` +
		`.prisma-ticket .c{display:inline-block;text-align:center;overflow:visible;transform-origin:left center}` +
		`.prisma-ticket .n{font-weight:700}.prisma-ticket .s{text-decoration:underline}` +
		`.prisma-ticket .i{background:#000;color:#fff}` +
		`.prisma-ticket img{display:block;image-rendering:pixelated}` +
		`.prisma-ticket .corte{border-top:2px dashed #888;margin:12px -16px;font:12px sans-serif;color:#888;padding:2px 16px}` +
		`.prisma-ticket .nota{font:12px sans-serif;color:#888;text-align:center;margin:4px 0}</style>`)

	for _, el := range p.Elementos {
		switch el.Tipo {
		case ElemLinea:
			ancho, alto := 0.0, 0.0
			for _, r := range el.Runs {
				celda, a := celdaA, altoA
				if r.FuenteB {
					celda, a = celdaB, altoB
				}
				ancho += celda * float64(anchoRun(r))
				alto = max(alto, a*float64(r.Alto))
			}
			if alto == 0 {
				alto = altoA
			}
			izq := float64(desplazar(int(ancho), p.AnchoPx, el.Alineacion))
			fmt.Fprintf(&b, `<div class="l" style="height:%.0fpx;padding-left:%.1fpx">`, alto, izq)
			for _, r := range el.Runs {
				celda, a := celdaA, altoA
				if r.FuenteB {
					celda, a = celdaB, altoB
				}
				clases := "c"
				if r.Negrita {
					clases += " n"
				}
				if r.Subrayado {
					clases += " s"
				}
				if r.Invertido {
					clases += " i"
				}
				// El glifo ocupa ~0,6 em de ancho: el tamaño de letra sale del alto y scaleX ajusta el ancho.
				fs := celda / 0.6 * float64(r.Alto)
				escala := float64(r.Ancho) / float64(r.Alto)
				for _, c := range r.Texto {
					fmt.Fprintf(&b, `<span class="%s" style="width:%.2fpx;height:%.0fpx;line-height:%.0fpx;font-size:%.1fpx"><span style="display:inline-block;transform:scaleX(%.3f)">%s</span></span>`,
						clases, celda*float64(r.Ancho), a*float64(r.Alto), a*float64(r.Alto), fs, escala, html.EscapeString(string(c)))
				}
			}
			b.WriteString(`</div>`)
		case ElemImagen:
			izq := desplazar(el.Mapa.Ancho, p.AnchoPx, el.Alineacion)
			fmt.Fprintf(&b, `<img alt="" width="%d" height="%d" style="margin-left:%dpx" src="data:image/png;base64,%s">`,
				el.Mapa.Ancho, el.Mapa.Alto, izq, base64.StdEncoding.EncodeToString(el.Mapa.PNG()))
		case ElemCorte:
			tipo := "corte total"
			if el.Parcial {
				tipo = "corte parcial"
			}
			fmt.Fprintf(&b, `<div class="corte">✂ %s</div>`, tipo)
		case ElemCajon:
			fmt.Fprintf(&b, `<div class="nota">abre cajón · pin %d</div>`, el.Pin)
		}
	}
	b.WriteString(`</div>`)
	return b.String()
}

// DocumentoHTML envuelve el fragmento en una página completa (para abrirla en el navegador).
func DocumentoHTML(fragmento, titulo string) string {
	return `<!doctype html><html lang="es"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">` +
		`<title>` + html.EscapeString(titulo) + `</title><style>body{background:#eee;margin:0;padding:24px;display:flex;justify-content:center}</style>` +
		`</head><body>` + fragmento + `</body></html>`
}
