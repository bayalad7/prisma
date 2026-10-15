package escpos

import (
	"strings"
	"testing"
)

const acentos = "á é í ó ú ü Á É Í Ó Ú Ü ñ Ñ ¡ ¿ € $ ° º ª"

func TestCodificacionPorPagina(t *testing.T) {
	casos := []struct {
		pagina      string
		esperado    string // texto tras preparar (lo que representa cada byte)
		sustituidos string // caracteres que deben aparecer en la advertencia
	}{
		// CP437 no tiene Á Í Ó Ú ni €.
		{"CP437", "á é í ó ú ü A É I O U Ü ñ Ñ ¡ ¿ ? $ ° º ª", "Á→A"},
		// CP850 lo tiene todo salvo €.
		{"CP850", "á é í ó ú ü Á É Í Ó Ú Ü ñ Ñ ¡ ¿ ? $ ° º ª", "€→?"},
		{"CP858", acentos, ""},
		{"WPC1252", acentos, ""},
	}
	for _, c := range casos {
		t.Run(c.pagina, func(t *testing.T) {
			cod, err := nuevoCodificador(c.pagina)
			if err != nil {
				t.Fatal(err)
			}
			prep := cod.preparar(acentos)
			if string(prep) != c.esperado {
				t.Fatalf("preparar:\n got %q\nwant %q", string(prep), c.esperado)
			}
			bs := cod.codificar(prep)
			if len(bs) != len([]rune(c.esperado)) {
				t.Fatalf("debe haber un byte por carácter: %d bytes para %d runas", len(bs), len(prep))
			}
			// Ida y vuelta: cada byte decodifica al carácter preparado.
			for k, b := range bs {
				var got rune
				if b < 0x80 {
					got = rune(b)
				} else {
					got = cod.tabla.DecodeByte(b)
				}
				if got != prep[k] {
					t.Errorf("byte 0x%02X decodifica a %q, quiero %q", b, got, prep[k])
				}
			}
			adv := cod.advertencia()
			switch {
			case c.sustituidos == "" && adv != nil:
				t.Errorf("no esperaba advertencia: %s", adv.Mensaje)
			case c.sustituidos != "" && (adv == nil || !strings.Contains(adv.Mensaje, c.sustituidos)):
				t.Errorf("la advertencia debe mencionar %q: %+v", c.sustituidos, adv)
			}
		})
	}
}

func TestCodificacionNFCYEquivalentes(t *testing.T) {
	cod, _ := nuevoCodificador("CP437")
	// "e" + acento combinante se compone a "é"; comillas tipográficas y raya pasan a ASCII.
	// El espacio duro existe en CP437 (0xFF) y se conserva.
	got := string(cod.preparar("Cafe\u0301 \u201crico\u201d \u2014 100\u00A0g"))
	if got != "Caf\u00e9 \"rico\" - 100\u00A0g" {
		t.Fatalf("got %q", got)
	}
	// Una marca combinante sin base se descarta y se advierte.
	if got := string(cod.preparar("\u0301x")); got != "x" {
		t.Fatalf("got %q", got)
	}
	if adv := cod.advertencia(); adv == nil || !strings.Contains(adv.Mensaje, "U+0301") {
		t.Fatalf("advertencia: %+v", adv)
	}
	// Sin equivalente posible: '?'.
	if got := string(cod.preparar("日本")); got != "??" {
		t.Fatalf("got %q", got)
	}
}

func TestPaginaDesconocida(t *testing.T) {
	if _, err := nuevoCodificador("UTF8"); err == nil {
		t.Fatal("esperaba error")
	}
	if c, err := nuevoCodificador("cp858"); err != nil || c.pagina != "CP858" {
		t.Fatal("el nombre no debe distinguir mayúsculas")
	}
}
