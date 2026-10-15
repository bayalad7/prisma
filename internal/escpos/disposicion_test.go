package escpos

import (
	"reflect"
	"strings"
	"testing"

	"github.com/bayalad7/prisma/internal/contrato"
)

func textos(ls [][]rune) []string {
	out := make([]string, len(ls))
	for i, l := range ls {
		out[i] = string(l)
	}
	return out
}

func TestPartir(t *testing.T) {
	casos := []struct {
		texto string
		ancho int
		want  []string
	}{
		{"", 10, []string{""}},
		{"Hola", 10, []string{"Hola"}},
		{"Pan de caja integral grande con ajonjolí", 30, []string{"Pan de caja integral grande", "con ajonjolí"}},
		{"  espacios   internos  ", 30, []string{"espacios   internos"}},
		{"uno\ndos\n\ntres", 10, []string{"uno", "dos", "", "tres"}},
		{"supercalifragilístico", 8, []string{"supercal", "ifragilí", "stico"}},
		{"ab supercalifragilístico", 8, []string{"ab super", "califrag", "ilístico"}},
		{"exacto10ch", 10, []string{"exacto10ch"}},
		{"ñandú ñandú", 5, []string{"ñandú", "ñandú"}},
	}
	for _, c := range casos {
		got := textos(partir([]rune(c.texto), c.ancho))
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("partir(%q, %d) = %q, quiero %q", c.texto, c.ancho, got, c.want)
		}
		for _, l := range got {
			if len([]rune(l)) > c.ancho {
				t.Errorf("línea %q excede %d columnas", l, c.ancho)
			}
		}
	}
}

func cols(anchos ...contrato.Ancho) []contrato.Columna {
	c := make([]contrato.Columna, len(anchos))
	for i, a := range anchos {
		c[i] = contrato.Columna{Ancho: a}
	}
	return c
}

func TestRepartirColumnas(t *testing.T) {
	auto := contrato.AnchoAuto
	casos := []struct {
		c     int
		cols  []contrato.Columna
		want  []int
		error bool
	}{
		// Ejemplo resuelto de la especificación (§4.3).
		{48, cols(12, auto, 25), []int{5, 30, 11}, false},
		{32, cols(12, auto, 25), []int{3, 20, 7}, false},
		// Sin columnas auto, el sobrante va a la última.
		{48, cols(50, 25), []int{23, 24}, false},
		// Varias auto: partes iguales y el residuo de izquierda a derecha.
		{48, cols(auto, auto, auto), []int{16, 15, 15}, false},
		{48, cols(auto), []int{48}, false},
		{48, cols(60, 60), nil, true},
		{48, cols(50, 50, auto), nil, true},
	}
	for _, c := range casos {
		got, err := repartirColumnas(c.c, c.cols)
		if (err != nil) != c.error || !reflect.DeepEqual(got, c.want) {
			t.Errorf("repartirColumnas(%d, %v) = %v, %v; quiero %v (error %v)", c.c, c.cols, got, err, c.want, c.error)
		}
		if err == nil {
			total := len(got) - 1
			for _, w := range got {
				total += w
			}
			if total != c.c {
				t.Errorf("la fila debe ocupar %d columnas, ocupa %d", c.c, total)
			}
		}
	}
}

func TestComponerFilaEjemploDeLaEspecificacion(t *testing.T) {
	celdas := [][]rune{[]rune("1"), []rune("Pan de caja integral grande con ajonjolí"), []rune("$62.50")}
	alins := []contrato.Alineacion{contrato.Izquierda, contrato.Izquierda, contrato.Derecha}
	got := textos(componerFila(celdas, []int{5, 30, 11}, alins))
	want := []string{
		"1     Pan de caja integral grande         $62.50",
		"      con ajonjolí                              ",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("\n got %q\nwant %q", got, want)
	}
}

func TestComponerFilaAlineaCadaLinea(t *testing.T) {
	celdas := [][]rune{[]rune("centro largo de verdad"), []rune("der")}
	alins := []contrato.Alineacion{contrato.Centro, contrato.Derecha}
	got := textos(componerFila(celdas, []int{10, 5}, alins))
	want := []string{"  centro     der", " largo de       ", "  verdad        "}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("\n got %q\nwant %q", got, want)
	}
	for _, l := range got {
		if len([]rune(l)) != 16 {
			t.Errorf("renglón %q debe medir 16", l)
		}
	}
}

func TestAlinear(t *testing.T) {
	if got := string(alinear([]rune("ab"), 6, contrato.Centro)); got != "  ab  " {
		t.Errorf("centro: %q", got)
	}
	if got := string(alinear([]rune("abc"), 6, contrato.Centro)); got != " abc  " {
		t.Errorf("centro impar: %q", got)
	}
	if got := string(alinear([]rune("ab"), 6, contrato.Derecha)); got != strings.Repeat(" ", 4)+"ab" {
		t.Errorf("derecha: %q", got)
	}
}
