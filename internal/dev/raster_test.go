package dev

import (
	"bytes"
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bayalad7/prisma/internal/contrato"
	"github.com/bayalad7/prisma/internal/escpos"
)

var dirRaster = flag.String("raster", "", "guarda aquí el PNG de cada ejemplo para revisarlo a ojo")

// TestRasterDeLosEjemplos: cada ejemplo válido se dibuja al ancho del papel, con tinta solo dentro
// del área imprimible, y el PNG se puede decodificar.
func TestRasterDeLosEjemplos(t *testing.T) {
	archivos, _ := filepath.Glob(filepath.Join(dirEjemplos, "validos", "*.json"))
	if len(archivos) == 0 {
		t.Fatal("no hay ejemplos")
	}
	for _, a := range archivos {
		for _, mm := range []int{58, 80} {
			nombre := fmt.Sprintf("%s-%d", strings.TrimSuffix(filepath.Base(a), ".json"), mm)
			t.Run(nombre, func(t *testing.T) {
				perfil, _ := PerfilPrueba(mm)
				datos, _, err := escpos.Renderizar(*decodificarEjemplo(t, a), perfil, recursosPrueba())
				if err != nil {
					t.Fatal(err)
				}
				pagina := Simular(datos, perfil)
				img := Rasterizar(pagina, mm)
				if img.Rect.Dx() != AnchoPapel(mm) || img.Rect.Dy() <= 2*margenVertical {
					t.Fatalf("tamaño %v; se esperaba %d puntos de ancho", img.Rect, AnchoPapel(mm))
				}
				x0 := (img.Rect.Dx() - perfil.AnchoPx) / 2
				tinta := 0
				for y := range img.Rect.Dy() {
					for x := range img.Rect.Dx() {
						if img.Pix[y*img.Stride+x] != pTinta {
							continue
						}
						tinta++
						if x < x0 || x >= x0+perfil.AnchoPx {
							t.Fatalf("tinta fuera del área imprimible en (%d, %d)", x, y)
						}
					}
				}
				if tinta == 0 {
					t.Fatal("el ticket no tiene tinta")
				}
				cod := RasterPNG(pagina, mm)
				if _, err := png.Decode(bytes.NewReader(cod)); err != nil {
					t.Fatal(err)
				}
				if *dirRaster != "" {
					os.MkdirAll(*dirRaster, 0o755)
					os.WriteFile(filepath.Join(*dirRaster, nombre+".png"), cod, 0o644)
				}
			})
		}
	}
}

// tintaDe cuenta los puntos negros de un documento de una sola operación de texto.
func tintaDe(t *testing.T, op *contrato.Texto) (tinta, ancho, alto int) {
	t.Helper()
	perfil, _ := PerfilPrueba(80)
	doc := contrato.Documento{Version: "1", Impresora: perfil.Alias, Copias: 1, Operaciones: []contrato.Operacion{op}}
	datos, _, err := escpos.Renderizar(doc, perfil, escpos.RecursosMapa{})
	if err != nil {
		t.Fatal(err)
	}
	img := Rasterizar(Simular(datos, perfil), 80)
	minX, maxX := img.Rect.Dx(), 0
	for y := range img.Rect.Dy() {
		for x := range img.Rect.Dx() {
			if img.Pix[y*img.Stride+x] == pTinta {
				tinta++
				minX, maxX = min(minX, x), max(maxX, x)
			}
		}
	}
	return tinta, maxX - minX + 1, img.Rect.Dy()
}

func TestRasterRespetaLosEstilos(t *testing.T) {
	texto := func(f func(*contrato.Texto)) *contrato.Texto {
		op := &contrato.Texto{Valor: "Acentos: áéíóú ñ €", Alineacion: contrato.Izquierda, Tamano: [2]int{1, 1}, Fuente: "A"}
		f(op)
		return op
	}
	normal, anchoNormal, altoNormal := tintaDe(t, texto(func(*contrato.Texto) {}))
	if normal == 0 {
		t.Fatal("el texto normal no tiene tinta")
	}
	if negrita, _, _ := tintaDe(t, texto(func(o *contrato.Texto) { o.Negrita = true })); negrita <= normal {
		t.Errorf("la negrita (%d puntos) debería tener más tinta que el texto normal (%d)", negrita, normal)
	}
	if subrayado, _, _ := tintaDe(t, texto(func(o *contrato.Texto) { o.Subrayado = true })); subrayado <= normal {
		t.Errorf("el subrayado (%d puntos) debería agregar tinta (%d)", subrayado, normal)
	}
	if invertido, _, _ := tintaDe(t, texto(func(o *contrato.Texto) { o.Invertido = true })); invertido <= 2*normal {
		t.Errorf("el texto invertido (%d puntos) debería ser casi todo tinta (normal: %d)", invertido, normal)
	}
	if _, ancho, alto := tintaDe(t, texto(func(o *contrato.Texto) { o.Tamano = [2]int{2, 2} })); ancho < 2*anchoNormal-4 || alto <= altoNormal {
		t.Errorf("el doble tamaño mide %d × %d; el normal, %d × %d", ancho, alto, anchoNormal, altoNormal)
	}
	if _, ancho, _ := tintaDe(t, texto(func(o *contrato.Texto) { o.Fuente = "B" })); ancho >= anchoNormal {
		t.Errorf("la fuente B (%d puntos de ancho) debería ser más angosta que la A (%d)", ancho, anchoNormal)
	}
}
