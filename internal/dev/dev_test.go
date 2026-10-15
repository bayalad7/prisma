package dev

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bayalad7/prisma/internal/contrato"
	"github.com/bayalad7/prisma/internal/escpos"
)

var actualizar = flag.Bool("actualizar", false, "reescribe los archivos golden de testdata/golden")

const dirEjemplos = "../../ejemplos"

// recursosPrueba incluye "logo-tienda", que usa todas-las-operaciones.json.
func recursosPrueba() escpos.Recursos {
	img := image.NewGray(image.Rect(0, 0, 240, 60))
	for y := range 60 {
		for x := range 240 {
			v := uint8(255)
			if (x/20+y/20)%2 == 0 {
				v = 0
			}
			img.SetGray(x, y, color.Gray{Y: v})
		}
	}
	var b bytes.Buffer
	png.Encode(&b, img)
	return escpos.RecursosMapa{"logo-tienda": b.Bytes()}
}

func decodificarEjemplo(t *testing.T, ruta string) *contrato.Documento {
	t.Helper()
	datos, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := contrato.Decodificar(datos)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

// TestGolden compara la traza ESC/POS y la vista en texto de cada ejemplo válido a 58 y 80 mm.
// Para regenerar: go test ./internal/dev -actualizar
func TestGolden(t *testing.T) {
	archivos, _ := filepath.Glob(filepath.Join(dirEjemplos, "validos", "*.json"))
	for _, a := range archivos {
		for _, mm := range []int{58, 80} {
			nombre := strings.TrimSuffix(filepath.Base(a), ".json")
			t.Run(fmt.Sprintf("%s-%d", nombre, mm), func(t *testing.T) {
				perfil, _ := PerfilPrueba(mm)
				datos, advs, err := escpos.Renderizar(*decodificarEjemplo(t, a), perfil, recursosPrueba())
				if err != nil {
					t.Fatal(err)
				}
				var got strings.Builder
				got.WriteString("== Vista ==\n")
				got.WriteString(VistaTexto(Simular(datos, perfil), false))
				got.WriteString("\n== Advertencias ==\n")
				for _, adv := range advs {
					got.WriteString(adv.Codigo + ": " + adv.Mensaje + "\n")
				}
				got.WriteString("\n== Traza ==\n")
				got.WriteString(Traza(datos))

				ruta := filepath.Join("testdata", "golden", filepath.Base(t.Name())+".txt")
				if *actualizar {
					os.MkdirAll(filepath.Dir(ruta), 0o755)
					if err := os.WriteFile(ruta, []byte(got.String()), 0o644); err != nil {
						t.Fatal(err)
					}
					return
				}
				want, err := os.ReadFile(ruta)
				if err != nil {
					t.Fatalf("falta el golden %s (ejecuta con -actualizar): %v", ruta, err)
				}
				if got.String() != strings.ReplaceAll(string(want), "\r\n", "\n") {
					t.Errorf("la salida no coincide con %s; revisa el cambio y regenera con -actualizar.\n%s", ruta, primeraDiferencia(got.String(), string(want)))
				}
			})
		}
	}
}

func primeraDiferencia(a, b string) string {
	la, lb := strings.Split(a, "\n"), strings.Split(b, "\n")
	for i := range max(len(la), len(lb)) {
		var x, y string
		if i < len(la) {
			x = la[i]
		}
		if i < len(lb) {
			y = lb[i]
		}
		if x != y {
			return fmt.Sprintf("línea %d:\n got %s\nwant %s", i+1, x, y)
		}
	}
	return ""
}

// TestLineasNoExcedenElAncho comprueba en todos los ejemplos que ninguna línea simulada
// excede el ancho del papel, que es lo que rompería la alineación al imprimir.
func TestLineasNoExcedenElAncho(t *testing.T) {
	archivos, _ := filepath.Glob(filepath.Join(dirEjemplos, "validos", "*.json"))
	for _, a := range archivos {
		for _, mm := range []int{58, 80} {
			perfil, _ := PerfilPrueba(mm)
			datos, _, err := escpos.Renderizar(*decodificarEjemplo(t, a), perfil, recursosPrueba())
			if err != nil {
				t.Fatal(err)
			}
			for _, el := range Simular(datos, perfil).Elementos {
				if el.Tipo == ElemImagen && el.Mapa.Ancho > perfil.AnchoPx {
					t.Errorf("%s %d mm: imagen de %d puntos", filepath.Base(a), mm, el.Mapa.Ancho)
				}
				if el.Tipo != ElemLinea {
					continue
				}
				puntos := 0
				for _, r := range el.Runs {
					cols := perfil.Columnas
					if r.FuenteB {
						cols = perfil.ColumnasFuenteB
					}
					puntos += anchoRun(r) * perfil.AnchoPx / cols
				}
				if puntos > perfil.AnchoPx {
					t.Errorf("%s %d mm: línea de %d puntos: %+v", filepath.Base(a), mm, puntos, el.Runs)
				}
			}
		}
	}
}

func TestAnalizarEsSinPerdida(t *testing.T) {
	perfil, _ := PerfilPrueba(80)
	datos, _, err := escpos.Renderizar(*decodificarEjemplo(t, filepath.Join(dirEjemplos, "validos", "todas-las-operaciones.json")), perfil, recursosPrueba())
	if err != nil {
		t.Fatal(err)
	}
	var unido []byte
	for _, c := range Analizar(datos) {
		if c.Nombre == "INCOMPLETO" || strings.HasPrefix(c.Nombre, "ESC 0x") || strings.HasPrefix(c.Nombre, "GS 0x") {
			t.Errorf("comando no reconocido en %06X: % X", c.Pos, c.Bytes)
		}
		unido = append(unido, c.Bytes...)
	}
	if !bytes.Equal(unido, datos) {
		t.Fatal("la concatenación de comandos debe reproducir la entrada")
	}
	if !strings.Contains(Traza([]byte{0x1D, 'v', '0', 0, 10}), "comando incompleto") {
		t.Error("un comando truncado debe marcarse como incompleto")
	}
}

func TestSimuladorInterpretaEstilos(t *testing.T) {
	perfil, _ := PerfilPrueba(80)
	doc := contrato.Documento{Copias: 1, Operaciones: []contrato.Operacion{
		&contrato.Texto{Valor: "Hola", Alineacion: contrato.Derecha, Negrita: true, Invertido: true, Tamano: [2]int{2, 3}, Fuente: "B"},
		&contrato.CodigoBarras{Valor: "4006381333931", Simbologia: contrato.EAN13, Texto: "abajo", Alto: 50, Alineacion: contrato.Centro},
		&contrato.AbrirCajon{Pin: 5},
		&contrato.Cortar{Parcial: true},
	}}
	datos, _, err := escpos.Renderizar(doc, perfil, nil)
	if err != nil {
		t.Fatal(err)
	}
	els := Simular(datos, perfil).Elementos
	if len(els) != 5 {
		t.Fatalf("elementos: %+v", els)
	}
	r := els[0].Runs[0]
	if els[0].Alineacion != contrato.Derecha || r.Texto != "Hola" || !r.Negrita || !r.Invertido || !r.FuenteB || r.Ancho != 2 || r.Alto != 3 {
		t.Errorf("texto: %+v %+v", els[0], r)
	}
	if els[1].Tipo != ElemImagen || els[1].Mapa.Ancho != 95*3 || els[1].Mapa.Alto != 50 || els[1].Alineacion != contrato.Centro {
		t.Errorf("código de barras: %+v", els[1])
	}
	if els[2].Tipo != ElemLinea || els[2].Runs[0].Texto != "4006381333931" {
		t.Errorf("texto legible: %+v", els[2])
	}
	if els[3].Tipo != ElemCajon || els[3].Pin != 5 || els[4].Tipo != ElemCorte || !els[4].Parcial {
		t.Errorf("cajón y corte: %+v %+v", els[3], els[4])
	}
}

func TestPrevisualizarHTMLEscapaElTexto(t *testing.T) {
	perfil, _ := PerfilPrueba(80)
	doc := contrato.Documento{Copias: 1, Operaciones: []contrato.Operacion{
		&contrato.Texto{Valor: `<script>alert("x")</script> & ñ`, Tamano: [2]int{1, 1}, Fuente: "A"},
		&contrato.QR{Valor: "PRISMA", Tamano: 4, Correccion: "M", Alineacion: contrato.Centro},
		&contrato.Cortar{},
	}}
	h, advs, err := Previsualizar(doc, perfil, nil)
	if err != nil || len(advs) != 0 {
		t.Fatal(err, advs)
	}
	for _, prohibido := range []string{"<script", `alert("x")`} {
		if strings.Contains(h, prohibido) {
			t.Errorf("el HTML contiene %q sin escapar", prohibido)
		}
	}
	for _, esperado := range []string{"&lt;", "&amp;", ">ñ<", "data:image/png;base64,", "corte total", `class="prisma-ticket" style="width:576px"`} {
		if !strings.Contains(h, esperado) {
			t.Errorf("falta %q en el HTML", esperado)
		}
	}
}

func TestPrevisualizarPropagaErrores(t *testing.T) {
	perfil, _ := PerfilPrueba(58)
	_, _, err := Previsualizar(contrato.Documento{Copias: 1, Operaciones: []contrato.Operacion{
		&contrato.Imagen{Recurso: "no-existe", AnchoMax: 100},
	}}, perfil, nil)
	var ce *contrato.Error
	if !errors.As(err, &ce) || ce.Codigo != contrato.ErrRecursoNoEncontrado {
		t.Fatalf("esperaba RECURSO_NO_ENCONTRADO, obtuve %v", err)
	}
	if _, err := PerfilPrueba(76); err == nil {
		t.Error("76 mm no es un ancho de prueba válido")
	}
}

// TestEjemplosInvalidosConPerfil cubre los casos del índice que solo detecta el renderizador.
func TestEjemplosInvalidosConPerfil(t *testing.T) {
	datos, _ := os.ReadFile(filepath.Join(dirEjemplos, "invalidos", "indice.json"))
	var indice []struct{ Archivo, Codigo, Regla string }
	if err := json.Unmarshal(datos, &indice); err != nil {
		t.Fatal(err)
	}
	probados := 0
	for _, e := range indice {
		if e.Regla != "ancho_minimo" {
			continue
		}
		probados++
		doc := decodificarEjemplo(t, filepath.Join(dirEjemplos, "invalidos", e.Archivo))
		for _, mm := range []int{58, 80} {
			perfil, _ := PerfilPrueba(mm)
			_, _, err := escpos.Renderizar(*doc, perfil, nil)
			var ce *contrato.Error
			if !errors.As(err, &ce) || ce.Codigo != e.Codigo {
				t.Errorf("%s a %d mm: esperaba %s, obtuve %v", e.Archivo, mm, e.Codigo, err)
			}
		}
	}
	if probados == 0 {
		t.Fatal("el índice no tiene casos que dependan de la impresora")
	}
}
