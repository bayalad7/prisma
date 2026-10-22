package escpos

import (
	"fmt"

	"github.com/bayalad7/prisma/internal/contrato"
)

// partir reparte el texto en líneas de como máximo `ancho` columnas (contrato v1, §3):
// párrafos por '\n', reparto voraz por palabras, palabras más largas que la línea cortadas
// en seco y espacios de los extremos de cada línea eliminados.
func partir(texto []rune, ancho int) [][]rune {
	if ancho < 1 {
		ancho = 1
	}
	var lineas [][]rune
	inicio := 0
	for i := 0; i <= len(texto); i++ {
		if i == len(texto) || texto[i] == '\n' {
			lineas = append(lineas, partirParrafo(texto[inicio:i], ancho)...)
			inicio = i + 1
		}
	}
	return lineas
}

func partirParrafo(p []rune, ancho int) [][]rune {
	// Cada palabra conserva los espacios que la preceden: dentro de una línea se respetan,
	// en un corte de línea se descartan.
	type pieza struct{ sep, palabra []rune }
	var piezas []pieza
	for i := 0; i < len(p); {
		j := i
		for j < len(p) && p[j] == ' ' {
			j++
		}
		k := j
		for k < len(p) && p[k] != ' ' {
			k++
		}
		if k > j {
			piezas = append(piezas, pieza{p[i:j], p[j:k]})
		}
		i = k
	}

	var lineas [][]rune
	var actual []rune
	cerrar := func() {
		lineas = append(lineas, actual)
		actual = nil
	}
	for _, pz := range piezas {
		w := pz.palabra
		switch {
		case len(actual) == 0:
		case len(actual)+len(pz.sep)+len(w) <= ancho:
			actual = append(append(actual, pz.sep...), w...)
			continue
		case len(w) > ancho && ancho-len(actual)-len(pz.sep) >= 1:
			// Palabra más larga que una línea: se corta en seco en el último carácter que cabe.
			cabe := ancho - len(actual) - len(pz.sep)
			actual = append(append(actual, pz.sep...), w[:cabe]...)
			w = w[cabe:]
			cerrar()
		default:
			cerrar()
		}
		for len(w) > ancho {
			actual = append([]rune(nil), w[:ancho]...)
			w = w[ancho:]
			cerrar()
		}
		actual = append([]rune(nil), w...)
	}
	if len(actual) > 0 || len(lineas) == 0 {
		cerrar()
	}
	return lineas
}

// alinear rellena la línea con espacios hasta `ancho` columnas.
func alinear(linea []rune, ancho int, a contrato.Alineacion) []rune {
	falta := ancho - len(linea)
	if falta <= 0 {
		return linea[:ancho]
	}
	izq := 0
	switch a {
	case contrato.Centro:
		izq = falta / 2
	case contrato.Derecha:
		izq = falta
	}
	out := make([]rune, 0, ancho)
	out = append(out, espacios(izq)...)
	out = append(out, linea...)
	return append(out, espacios(falta-izq)...)
}

func espacios(n int) []rune {
	r := make([]rune, n)
	for i := range r {
		r[i] = ' '
	}
	return r
}

// repartirColumnas aplica el algoritmo de reparto de §4.3 sobre C columnas de la línea.
func repartirColumnas(c int, cols []contrato.Columna) ([]int, error) {
	n := len(cols)
	u := c - (n - 1)
	anchos := make([]int, n)
	suma, autos := 0, 0
	for i, col := range cols {
		if col.Ancho == contrato.AnchoAuto {
			autos++
			continue
		}
		suma += int(col.Ancho)
		anchos[i] = u * int(col.Ancho) / 100
	}
	if suma > 100 {
		return nil, fmt.Errorf("los porcentajes suman %d; el máximo es 100", suma)
	}
	r := u
	for _, w := range anchos {
		r -= w
	}
	if autos == 0 {
		anchos[n-1] += r
	} else {
		base, resto := r/autos, r%autos
		for i, col := range cols {
			if col.Ancho != contrato.AnchoAuto {
				continue
			}
			anchos[i] = base
			if resto > 0 {
				anchos[i]++
				resto--
			}
		}
	}
	for i, w := range anchos {
		if w < 1 {
			return nil, fmt.Errorf("la columna %d se queda sin espacio en una línea de %d columnas", i+1, c)
		}
	}
	return anchos, nil
}

// componerFila parte cada celda en su ancho y devuelve los renglones físicos completos.
func componerFila(celdas [][]rune, anchos []int, alins []contrato.Alineacion) [][]rune {
	partidas := make([][][]rune, len(celdas))
	alto := 0
	for i, celda := range celdas {
		if anchos[i] == 0 {
			partidas[i] = [][]rune{{}}
		} else {
			partidas[i] = partir(celda, anchos[i])
		}
		alto = max(alto, len(partidas[i]))
	}
	renglones := make([][]rune, alto)
	for f := range alto {
		var r []rune
		for i := range celdas {
			if i > 0 {
				r = append(r, ' ')
			}
			var linea []rune
			if f < len(partidas[i]) {
				linea = partidas[i][f]
			}
			r = append(r, alinear(linea, anchos[i], alins[i])...)
		}
		renglones[f] = r
	}
	return renglones
}
