package dev

import (
	"bytes"
	"fmt"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/contrato"
	"github.com/bayalad7/prisma/internal/escpos"
)

// Tipos de elemento de una página simulada.
const (
	ElemLinea = iota
	ElemImagen
	ElemCorte
	ElemCajon
)

// Run es un tramo de texto con un mismo estilo.
type Run struct {
	Texto                                  string
	Negrita, Subrayado, Invertido, FuenteB bool
	Ancho, Alto                            int
}

// Elemento es lo que la impresora virtual dejó en el papel, en orden.
type Elemento struct {
	Tipo       int
	Alineacion contrato.Alineacion
	Runs       []Run        // ElemLinea (vacío = línea en blanco)
	Mapa       *escpos.Mapa // ElemImagen
	Parcial    bool         // ElemCorte
	Pin        int          // ElemCajon
}

// Pagina es el resultado de simular un flujo ESC/POS en una impresora con el perfil dado.
type Pagina struct {
	AnchoPx, Columnas, ColumnasB int
	Elementos                    []Elemento
}

type estadoImpresora struct {
	alineacion                             contrato.Alineacion
	negrita, subrayado, invertido, fuenteB bool
	ancho, alto                            int
	pagina                                 string
	altoBarras, moduloBarras, hri          int
	qrModulo, qrCorreccion                 int
	qrDatos                                []byte
}

func estadoInicial() estadoImpresora {
	return estadoImpresora{alineacion: contrato.Izquierda, ancho: 1, alto: 1, pagina: "CP437",
		altoBarras: 162, moduloBarras: 3, qrModulo: 3, qrCorreccion: 48}
}

// Simular ejecuta los bytes en una impresora virtual con el perfil indicado.
func Simular(datos []byte, perfil almacen.Impresora) *Pagina {
	p := &Pagina{AnchoPx: perfil.AnchoPx, Columnas: perfil.Columnas, ColumnasB: perfil.ColumnasFuenteB}
	e := estadoInicial()
	var linea []Run
	alineacionLinea := e.alineacion
	pendiente := false

	cerrarLinea := func() {
		p.Elementos = append(p.Elementos, Elemento{Tipo: ElemLinea, Alineacion: alineacionLinea, Runs: linea})
		linea, pendiente = nil, false
	}
	vaciar := func() { // termina una línea a medias antes de una imagen, un corte, etc.
		if pendiente {
			cerrarLinea()
		}
	}
	imagen := func(m *escpos.Mapa, a contrato.Alineacion) {
		vaciar()
		p.Elementos = append(p.Elementos, Elemento{Tipo: ElemImagen, Mapa: m, Alineacion: a})
	}
	textoLegible := func(s string) {
		p.Elementos = append(p.Elementos, Elemento{Tipo: ElemLinea, Alineacion: e.alineacion, Runs: []Run{{Texto: s, Ancho: 1, Alto: 1}}})
	}

	for _, c := range Analizar(datos) {
		a := func(k int) int {
			if k < len(c.Args) {
				return c.Args[k]
			}
			return 0
		}
		switch c.Nombre {
		case "TEXTO":
			if !pendiente {
				alineacionLinea, pendiente = e.alineacion, true
			}
			linea = append(linea, Run{Texto: escpos.DecodificarTexto(e.pagina, c.Datos), Negrita: e.negrita, Subrayado: e.subrayado,
				Invertido: e.invertido, FuenteB: e.fuenteB, Ancho: e.ancho, Alto: e.alto})
		case "LF":
			if !pendiente {
				alineacionLinea = e.alineacion
			}
			cerrarLinea()
		case "ESC @":
			vaciar()
			e = estadoInicial()
		case "ESC t":
			e.pagina = escpos.PaginaPorComando(a(0))
		case "ESC a":
			e.alineacion = []contrato.Alineacion{contrato.Izquierda, contrato.Centro, contrato.Derecha}[min(a(0)%48, 2)]
		case "ESC E":
			e.negrita = a(0)&1 == 1
		case "ESC -":
			e.subrayado = a(0)%48 > 0
		case "ESC M":
			e.fuenteB = a(0)&1 == 1
		case "GS B":
			e.invertido = a(0)&1 == 1
		case "GS !":
			e.ancho, e.alto = a(0)>>4+1, a(0)&0x0F+1
		case "ESC d":
			vaciar()
			for range a(0) {
				p.Elementos = append(p.Elementos, Elemento{Tipo: ElemLinea, Alineacion: e.alineacion})
			}
		case "ESC p":
			vaciar()
			pin := 2
			if a(0)&1 == 1 {
				pin = 5
			}
			p.Elementos = append(p.Elementos, Elemento{Tipo: ElemCajon, Pin: pin})
		case "GS V":
			vaciar()
			m := a(0)
			p.Elementos = append(p.Elementos, Elemento{Tipo: ElemCorte, Parcial: m == 1 || m == 49 || m == 66})
		case "GS v 0":
			imagen(mapaRaster(c.Datos, a(1), a(2)), contrato.Izquierda)
		case "GS h":
			e.altoBarras = a(0)
		case "GS w":
			e.moduloBarras = a(0)
		case "GS H":
			e.hri = a(0) % 48
		case "GS ( k":
			switch a(1) {
			case 67:
				e.qrModulo = int(c.Bytes[7])
			case 69:
				e.qrCorreccion = int(c.Bytes[7])
			case 80:
				e.qrDatos = append([]byte(nil), c.Datos[1:]...)
			case 81:
				correccion := nombresCorreccion[e.qrCorreccion]
				if m, err := escpos.MapaQR(string(e.qrDatos), correccion, e.qrModulo); err == nil {
					imagen(m, e.alineacion)
				}
			}
		case "GS k":
			simb := simbologias[a(0)]
			valor := string(c.Datos)
			if simb == contrato.CODE128 {
				valor = string(bytes.ReplaceAll(bytes.TrimPrefix(c.Datos, []byte("{B")), []byte("{{"), []byte("{")))
			}
			m, err := escpos.MapaBarras(simb, valor, e.moduloBarras, e.altoBarras)
			if err != nil {
				vaciar()
				textoLegible(fmt.Sprintf("[código %s no válido: %v]", simb, err))
				continue
			}
			if e.hri == 1 || e.hri == 3 {
				vaciar()
				textoLegible(valor)
			}
			imagen(m, e.alineacion)
			if e.hri == 2 || e.hri == 3 {
				textoLegible(valor)
			}
		}
	}
	vaciar()
	return p
}

func mapaRaster(datos []byte, anchoBytes, alto int) *escpos.Mapa {
	m := escpos.NuevoMapa(anchoBytes*8, alto)
	for y := range alto {
		for x := range anchoBytes * 8 {
			if datos[y*anchoBytes+x/8]&(0x80>>(x%8)) != 0 {
				m.Poner(x, y, true)
			}
		}
	}
	return m
}
