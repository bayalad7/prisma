package escpos

import (
	"strings"
	"testing"

	"github.com/bayalad7/prisma/internal/contrato"
)

func TestTablaCode128(t *testing.T) {
	vistos := map[string]bool{}
	for i, p := range code128Patrones {
		suma := 0
		for _, c := range p {
			suma += int(c - '0')
		}
		want := 11
		if i == 106 {
			want = 13
		}
		if suma != want {
			t.Errorf("símbolo %d (%s) suma %d módulos, quiero %d", i, p, suma, want)
		}
		if vistos[p] {
			t.Errorf("símbolo %d (%s) repetido", i, p)
		}
		vistos[p] = true
	}
}

func TestTablaCode39(t *testing.T) {
	vistos := map[string]bool{}
	for c, p := range code39Patrones {
		if len(p) != 9 || strings.Count(p, "w") != 3 {
			t.Errorf("%q: %s debe tener 9 elementos y 3 anchos", c, p)
		}
		if vistos[p] {
			t.Errorf("%q: patrón repetido", c)
		}
		vistos[p] = true
	}
	if len(code39Patrones) != 44 {
		t.Errorf("CODE39 tiene 43 caracteres más el delimitador, hay %d", len(code39Patrones))
	}
}

func TestModulos(t *testing.T) {
	casos := []struct {
		simb, valor string
		modulos     int
	}{
		{contrato.EAN13, "4006381333931", 95},
		{contrato.EAN13, "400638133393", 95}, // el verificador se calcula
		{contrato.UPCA, "036000291452", 95},
		{contrato.EAN8, "96385074", 67},
		{contrato.ITF, "1234", 4 + 2*2*(2*anchoW+3) + anchoW + 2},
		{contrato.CODE39, "AB", 4*(6+3*anchoW) + 3},
		{contrato.CODE128, "Pedido#812/a", 11*(1+12+1) + 13},
	}
	for _, c := range casos {
		m, err := Modulos(c.simb, c.valor)
		if err != nil {
			t.Fatalf("%s %s: %v", c.simb, c.valor, err)
		}
		if len(m) != c.modulos {
			t.Errorf("%s %s: %d módulos, quiero %d", c.simb, c.valor, len(m), c.modulos)
		}
		if !m[0] || !m[len(m)-1] {
			t.Errorf("%s: debe empezar y terminar con barra", c.simb)
		}
	}
	// EAN-13 con verificador calculado = con verificador explícito; UPC-A = EAN-13 con 0 inicial.
	a, _ := Modulos(contrato.EAN13, "400638133393")
	b, _ := Modulos(contrato.EAN13, "4006381333931")
	u, _ := Modulos(contrato.UPCA, "036000291452")
	e, _ := Modulos(contrato.EAN13, "0036000291452")
	if string(boolsATexto(a)) != string(boolsATexto(b)) || string(boolsATexto(u)) != string(boolsATexto(e)) {
		t.Error("equivalencias GS1 incorrectas")
	}
	// Guardas de EAN-13: 101 al inicio y al final, 01010 al centro (módulos 45–49).
	if s := boolsATexto(b); s[:3] != "101" || s[92:] != "101" || s[45:50] != "01010" {
		t.Errorf("guardas EAN-13: %s", s)
	}

	for _, inv := range [][2]string{{contrato.ITF, "123"}, {contrato.CODE39, "abc"}, {contrato.CODE128, "ñ"}, {contrato.EAN8, "12"}, {"PDF417", "x"}} {
		if _, err := Modulos(inv[0], inv[1]); err == nil {
			t.Errorf("%s %q debía fallar", inv[0], inv[1])
		}
	}
}

func boolsATexto(m []bool) string {
	var b strings.Builder
	for _, v := range m {
		if v {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
	}
	return b.String()
}
