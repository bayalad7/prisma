package escpos

import (
	"fmt"
	"strings"

	"github.com/bayalad7/prisma/internal/contrato"
)

// Módulos codifica un código de barras como secuencia de módulos (true = barra), sin zonas
// de silencio. Se usa cuando la impresora no tiene códigos de barras nativos, para calcular
// el ancho de los nativos y en la previsualización.
func Modulos(simbologia, valor string) ([]bool, error) {
	switch simbologia {
	case contrato.EAN13:
		return ean(completarGS1(valor, 13))
	case contrato.UPCA:
		return ean("0" + completarGS1(valor, 12))
	case contrato.EAN8:
		return ean(completarGS1(valor, 8))
	case contrato.ITF:
		return itf(valor)
	case contrato.CODE39:
		return code39(valor)
	case contrato.CODE128:
		return code128(valor)
	}
	return nil, fmt.Errorf("simbología %q no soportada", simbologia)
}

// completarGS1 agrega el dígito verificador si falta.
func completarGS1(valor string, largo int) string {
	if len(valor) == largo-1 {
		return valor + string(rune('0'+contrato.DigitoControlGS1(valor)))
	}
	return valor
}

// anchos agrega módulos alternando barra/espacio según una lista de anchos, empezando por barra.
func anchos(m []bool, ws string) []bool {
	barra := true
	for _, w := range ws {
		for range int(w - '0') {
			m = append(m, barra)
		}
		barra = !barra
	}
	return m
}

func bits(m []bool, patron string) []bool {
	for _, b := range patron {
		m = append(m, b == '1')
	}
	return m
}

var (
	eanL = [10]string{"0001101", "0011001", "0010011", "0111101", "0100011", "0110001", "0101111", "0111011", "0110111", "0001011"}
	// Paridad de los 6 dígitos izquierdos de EAN-13 según el primer dígito (L = impar, G = par).
	eanParidad = [10]string{"LLLLLL", "LLGLGG", "LLGGLG", "LLGGGL", "LGLLGG", "LGGLLG", "LGGGLL", "LGLGLG", "LGLGGL", "LGGLGL"}
)

func eanR(d int) string {
	var b strings.Builder
	for _, c := range eanL[d] {
		if c == '0' {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
	}
	return b.String()
}

func eanG(d int) string {
	r := []rune(eanR(d))
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

func ean(v string) ([]bool, error) {
	for _, c := range v {
		if c < '0' || c > '9' {
			return nil, fmt.Errorf("%q no es numérico", v)
		}
	}
	d := func(i int) int { return int(v[i] - '0') }
	var m []bool
	switch len(v) {
	case 13:
		m = bits(m, "101")
		for i := 1; i <= 6; i++ {
			if eanParidad[d(0)][i-1] == 'L' {
				m = bits(m, eanL[d(i)])
			} else {
				m = bits(m, eanG(d(i)))
			}
		}
		m = bits(m, "01010")
		for i := 7; i <= 12; i++ {
			m = bits(m, eanR(d(i)))
		}
		return bits(m, "101"), nil
	case 8:
		m = bits(m, "101")
		for i := 0; i < 4; i++ {
			m = bits(m, eanL[d(i)])
		}
		m = bits(m, "01010")
		for i := 4; i < 8; i++ {
			m = bits(m, eanR(d(i)))
		}
		return bits(m, "101"), nil
	}
	return nil, fmt.Errorf("longitud %d no válida", len(v))
}

// ITF: cada dígito son 5 elementos (n = estrecho, w = ancho); los pares se intercalan barra/espacio.
var itfPatrones = [10]string{"nnwwn", "wnnnw", "nwnnw", "wwnnn", "nnwnw", "wnwnn", "nwwnn", "nnnww", "wnnwn", "nwnwn"}

const anchoW = 3 // relación ancho/estrecho para ITF y CODE39

func elemento(c byte) string {
	if c == 'w' {
		return string(rune('0' + anchoW))
	}
	return "1"
}

func itf(v string) ([]bool, error) {
	if len(v) == 0 || len(v)%2 != 0 {
		return nil, fmt.Errorf("ITF requiere un número par de dígitos")
	}
	m := anchos(nil, "1111")
	for i := 0; i < len(v); i += 2 {
		a, b := v[i]-'0', v[i+1]-'0'
		if a > 9 || b > 9 {
			return nil, fmt.Errorf("%q no es numérico", v)
		}
		var ws strings.Builder
		for k := 0; k < 5; k++ {
			ws.WriteString(elemento(itfPatrones[a][k]))
			ws.WriteString(elemento(itfPatrones[b][k]))
		}
		m = anchos(m, ws.String())
	}
	return anchos(m, elemento('w')+"11"), nil
}

// CODE39: 9 elementos por carácter (barra primero), 3 anchos, separados por un espacio estrecho.
var code39Patrones = map[rune]string{
	'0': "nnnwwnwnn", '1': "wnnwnnnnw", '2': "nnwwnnnnw", '3': "wnwwnnnnn", '4': "nnnwwnnnw",
	'5': "wnnwwnnnn", '6': "nnwwwnnnn", '7': "nnnwnnwnw", '8': "wnnwnnwnn", '9': "nnwwnnwnn",
	'A': "wnnnnwnnw", 'B': "nnwnnwnnw", 'C': "wnwnnwnnn", 'D': "nnnnwwnnw", 'E': "wnnnwwnnn",
	'F': "nnwnwwnnn", 'G': "nnnnnwwnw", 'H': "wnnnnwwnn", 'I': "nnwnnwwnn", 'J': "nnnnwwwnn",
	'K': "wnnnnnnww", 'L': "nnwnnnnww", 'M': "wnwnnnnwn", 'N': "nnnnwnnww", 'O': "wnnnwnnwn",
	'P': "nnwnwnnwn", 'Q': "nnnnnnwww", 'R': "wnnnnnwwn", 'S': "nnwnnnwwn", 'T': "nnnnwnwwn",
	'U': "wwnnnnnnw", 'V': "nwwnnnnnw", 'W': "wwwnnnnnn", 'X': "nwnnwnnnw", 'Y': "wwnnwnnnn",
	'Z': "nwwnwnnnn", '-': "nwnnnnwnw", '.': "wwnnnnwnn", ' ': "nwwnnnwnn", '*': "nwnnwnwnn",
	'$': "nwnwnwnnn", '/': "nwnwnnnwn", '+': "nwnnnwnwn", '%': "nnnwnwnwn",
}

func code39(v string) ([]bool, error) {
	var m []bool
	for i, c := range "*" + v + "*" {
		p, ok := code39Patrones[c]
		if !ok || (c == '*' && i != 0 && i != len(v)+1) {
			return nil, fmt.Errorf("CODE39 no admite el carácter %q", c)
		}
		if i > 0 {
			m = append(m, false) // espacio entre caracteres
		}
		var ws strings.Builder
		for k := 0; k < 9; k++ {
			ws.WriteString(elemento(p[k]))
		}
		m = anchos(m, ws.String())
	}
	return m, nil
}

// CODE128: anchos de los 107 símbolos (barra, espacio, …). 103–105 = START A/B/C, 106 = STOP.
var code128Patrones = [107]string{
	"212222", "222122", "222221", "121223", "121322", "131222", "122213", "122312", "132212", "221213",
	"221312", "231212", "112232", "122132", "122231", "113222", "123122", "123221", "223211", "221132",
	"221231", "213212", "223112", "312131", "311222", "321122", "321221", "312212", "322112", "322211",
	"212123", "212321", "232121", "111323", "131123", "131321", "112313", "132113", "132311", "211313",
	"231113", "231311", "112133", "112331", "132131", "113123", "113321", "133121", "313121", "211331",
	"231131", "213113", "213311", "213131", "311123", "311321", "331121", "312113", "312311", "332111",
	"314111", "221411", "431111", "111224", "111422", "121124", "121421", "141122", "141221", "112214",
	"112412", "122114", "122411", "142112", "142211", "241211", "221114", "413111", "241112", "134111",
	"111242", "121142", "121241", "114212", "124112", "124211", "411212", "421112", "421211", "212141",
	"214121", "412121", "111143", "111341", "131141", "114113", "114311", "411113", "411311", "113141",
	"114131", "311141", "411131", "211412", "211214", "211232", "2331112",
}

const code128InicioB = 104

// code128 usa siempre el juego B (ASCII 32–126), igual que el comando nativo que envía el puente.
func code128(v string) ([]bool, error) {
	m := anchos(nil, code128Patrones[code128InicioB])
	suma := code128InicioB
	for i, c := range []byte(v) {
		if c < 32 || c > 126 {
			return nil, fmt.Errorf("CODE128 no admite el byte 0x%02X", c)
		}
		val := int(c) - 32
		suma += (i + 1) * val
		m = anchos(m, code128Patrones[val])
	}
	m = anchos(m, code128Patrones[suma%103])
	return anchos(m, code128Patrones[106]), nil
}
