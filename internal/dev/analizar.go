// Package dev reúne herramientas de desarrollo: decodificador de ESC/POS, impresora virtual
// y previsualización en texto y HTML.
package dev

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/bayalad7/prisma/internal/escpos"
)

const (
	esc = 0x1B
	gs  = 0x1D
	lf  = 0x0A
)

// Comando es un comando ESC/POS (o un tramo de texto) reconocido en un flujo de bytes.
type Comando struct {
	Pos    int
	Bytes  []byte // bytes completos del comando
	Nombre string // "ESC a", "GS v 0", "TEXTO", "LF", …
	Args   []int  // parámetros numéricos
	Datos  []byte // carga útil: texto, raster, datos de QR o de código de barras
}

// Analizar divide los bytes en comandos. La concatenación de Bytes reproduce la entrada.
func Analizar(datos []byte) []Comando {
	var cs []Comando
	for i := 0; i < len(datos); {
		c := analizarUno(datos, i)
		cs = append(cs, c)
		i += len(c.Bytes)
	}
	return cs
}

func analizarUno(d []byte, i int) Comando {
	resto := d[i:]
	tomar := func(n int, nombre string, args ...int) Comando {
		if n > len(resto) {
			return Comando{Pos: i, Bytes: resto, Nombre: "INCOMPLETO"}
		}
		return Comando{Pos: i, Bytes: resto[:n], Nombre: nombre, Args: args}
	}
	arg := func(k int) int {
		if k < len(resto) {
			return int(resto[k])
		}
		return 0
	}

	switch b := resto[0]; {
	case b == esc && len(resto) > 1:
		switch c := resto[1]; c {
		case '@':
			return tomar(2, "ESC @")
		case '2':
			return tomar(2, "ESC 2")
		case 't', 'a', 'E', '-', 'M', 'd', '!', 'J', '3', 'G', 'r':
			return tomar(3, "ESC "+string(c), arg(2))
		case 'p':
			return tomar(5, "ESC p", arg(2), arg(3), arg(4))
		default:
			return tomar(2, fmt.Sprintf("ESC 0x%02X", c))
		}
	case b == gs && len(resto) > 1:
		switch c := resto[1]; c {
		case '!', 'B', 'h', 'H', 'w', 'f':
			return tomar(3, "GS "+string(c), arg(2))
		case 'V':
			if m := arg(2); m == 65 || m == 66 {
				return tomar(4, "GS V", m, arg(3))
			}
			return tomar(3, "GS V", arg(2))
		case 'v':
			if len(resto) < 8 || resto[2] != '0' {
				return tomar(len(resto), "INCOMPLETO")
			}
			x, y := arg(4)|arg(5)<<8, arg(6)|arg(7)<<8
			cmd := tomar(8+x*y, "GS v 0", arg(3), x, y)
			if cmd.Nombre != "INCOMPLETO" {
				cmd.Datos = cmd.Bytes[8:]
			}
			return cmd
		case '(':
			if arg(2) != 'k' || len(resto) < 7 {
				return tomar(2, "GS (")
			}
			p := arg(3) | arg(4)<<8
			cmd := tomar(5+p, "GS ( k", arg(5), arg(6))
			if cmd.Nombre != "INCOMPLETO" && p > 2 {
				cmd.Datos = cmd.Bytes[7:]
			}
			return cmd
		case 'k':
			m := arg(2)
			if m <= 6 { // formato terminado en NUL
				fin := strings.IndexByte(string(resto[3:]), 0)
				if fin < 0 {
					return tomar(len(resto)+1, "INCOMPLETO")
				}
				cmd := tomar(3+fin+1, "GS k", m)
				cmd.Datos = cmd.Bytes[3 : 3+fin]
				return cmd
			}
			n := arg(3)
			cmd := tomar(4+n, "GS k", m)
			if cmd.Nombre != "INCOMPLETO" {
				cmd.Datos = cmd.Bytes[4:]
			}
			return cmd
		default:
			return tomar(2, fmt.Sprintf("GS 0x%02X", c))
		}
	case b == 0x10 && len(resto) > 1 && resto[1] == 0x04:
		return tomar(3, "DLE EOT", arg(2))
	case b == lf:
		return tomar(1, "LF")
	case b == 0x0D:
		return tomar(1, "CR")
	case b < 0x20:
		return tomar(1, fmt.Sprintf("CTRL 0x%02X", b))
	}
	// Texto: bytes imprimibles consecutivos.
	n := 0
	for n < len(resto) && resto[n] >= 0x20 {
		n++
	}
	c := tomar(n, "TEXTO")
	c.Datos = c.Bytes
	return c
}

var (
	nombresAlineacion = []string{"izquierda", "centro", "derecha"}
	nombresHRI        = []string{"ninguno", "arriba", "abajo", "arriba y abajo"}
	nombresCorreccion = map[int]string{48: "L", 49: "M", 50: "Q", 51: "H"}
	simbologias       = map[int]string{65: "UPCA", 66: "UPCE", 67: "EAN13", 68: "EAN8", 69: "CODE39", 70: "ITF", 72: "CODE93", 73: "CODE128"}
)

func siNo(n int) string {
	if n&1 == 1 {
		return "sí"
	}
	return "no"
}

func nombre(lista []string, n int) string {
	if n >= 0 && n < len(lista) {
		return lista[n]
	}
	return fmt.Sprintf("¿%d?", n)
}

// Traza devuelve una línea legible por comando: posición, bytes y significado.
func Traza(datos []byte) string {
	var b strings.Builder
	pagina := "CP437"
	for _, c := range Analizar(datos) {
		desc := describir(c, &pagina)
		hex := fmt.Sprintf("% X", c.Bytes)
		if len(c.Bytes) > 8 {
			hex = fmt.Sprintf("% X …", c.Bytes[:8])
		}
		fmt.Fprintf(&b, "%06X  %-26s %s\n", c.Pos, hex, desc)
	}
	return b.String()
}

func resumen(d []byte) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256(d))[:15]
}

func describir(c Comando, pagina *string) string {
	a := func(k int) int {
		if k < len(c.Args) {
			return c.Args[k]
		}
		return 0
	}
	switch c.Nombre {
	case "TEXTO":
		return fmt.Sprintf("texto %q", escpos.DecodificarTexto(*pagina, c.Datos))
	case "LF":
		return "salto de línea"
	case "ESC @":
		return "inicializar impresora"
	case "DLE EOT":
		return fmt.Sprintf("consultar estado %d (no imprime)", a(0))
	case "ESC t":
		*pagina = escpos.PaginaPorComando(a(0))
		return fmt.Sprintf("página de códigos %d (%s)", a(0), *pagina)
	case "ESC a":
		return "alineación " + nombre(nombresAlineacion, a(0)%48)
	case "ESC E":
		return "negrita " + siNo(a(0))
	case "ESC -":
		return fmt.Sprintf("subrayado %d", a(0)%48)
	case "ESC M":
		return "fuente " + map[bool]string{false: "A", true: "B"}[a(0)&1 == 1]
	case "ESC d":
		return fmt.Sprintf("avanzar %d línea(s)", a(0))
	case "ESC p":
		pin := 2
		if a(0)&1 == 1 {
			pin = 5
		}
		return fmt.Sprintf("abrir cajón pin %d (%d ms / %d ms)", pin, a(1)*2, a(2)*2)
	case "GS B":
		return "invertido " + siNo(a(0))
	case "GS !":
		return fmt.Sprintf("tamaño %d×%d", a(0)>>4+1, a(0)&0x0F+1)
	case "GS V":
		tipo := "total"
		if m := a(0); m == 1 || m == 49 || m == 66 {
			tipo = "parcial"
		}
		if len(c.Args) > 1 {
			return fmt.Sprintf("avanzar hasta la cuchilla (+%d) y corte %s", a(1), tipo)
		}
		return "corte " + tipo
	case "GS v 0":
		return fmt.Sprintf("imagen %d×%d puntos (%d bytes, %s)", a(1)*8, a(2), len(c.Datos), resumen(c.Datos))
	case "GS ( k":
		switch a(1) {
		case 65:
			return "QR: modelo 2"
		case 67:
			return fmt.Sprintf("QR: módulo de %d puntos", int(c.Bytes[7]))
		case 69:
			return "QR: corrección " + nombresCorreccion[int(c.Bytes[7])]
		case 80:
			return fmt.Sprintf("QR: almacenar %d bytes %q", len(c.Datos)-1, string(c.Datos[1:]))
		case 81:
			return "QR: imprimir"
		}
		return fmt.Sprintf("GS ( k función %d", a(1))
	case "GS h":
		return fmt.Sprintf("alto del código de barras %d puntos", a(0))
	case "GS H":
		return "texto legible " + nombre(nombresHRI, a(0)%48)
	case "GS w":
		return fmt.Sprintf("ancho de módulo %d puntos", a(0))
	case "GS f":
		return "fuente del texto legible " + map[bool]string{false: "A", true: "B"}[a(0)&1 == 1]
	case "GS k":
		return fmt.Sprintf("código de barras %s %q", simbologias[a(0)], string(c.Datos))
	case "INCOMPLETO":
		return "comando incompleto al final de los datos"
	}
	return "comando no reconocido"
}
