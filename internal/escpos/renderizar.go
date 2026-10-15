// Package escpos convierte un documento v1 en bytes ESC/POS según el perfil de la impresora.
package escpos

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/contrato"
)

// Códigos de advertencia: el trabajo se imprime, pero no exactamente como se pidió.
const (
	AdvCaracterSustituido = "CARACTER_SUSTITUIDO"
	AdvSinCuchilla        = "SIN_CUCHILLA"
	AdvSinCajon           = "SIN_CAJON"
	AdvQRReducido         = "QR_REDUCIDO"
)

// Advertencia se guarda con el trabajo (trabajos.advertencias).
type Advertencia struct {
	Codigo  string `json:"codigo"`
	Mensaje string `json:"mensaje"`
	Ruta    string `json:"ruta,omitempty"`
}

// ErrRecursoNoEncontrado lo devuelve Recursos cuando la imagen no existe.
var ErrRecursoNoEncontrado = errors.New("recurso no encontrado")

// Recursos entrega las imágenes guardadas en el puente.
type Recursos interface {
	ImagenRecurso(nombre string) ([]byte, error)
}

// RecursosMapa es una implementación en memoria (pruebas y herramientas).
type RecursosMapa map[string][]byte

func (r RecursosMapa) ImagenRecurso(nombre string) ([]byte, error) {
	if png, ok := r[nombre]; ok {
		return png, nil
	}
	return nil, ErrRecursoNoEncontrado
}

// RecursosDirectorio lee <dir>/<nombre>.png (herramientas de desarrollo).
type RecursosDirectorio string

func (d RecursosDirectorio) ImagenRecurso(nombre string) ([]byte, error) {
	datos, err := os.ReadFile(filepath.Join(string(d), nombre+".png"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrRecursoNoEncontrado
	}
	return datos, err
}

// RecursosAlmacen entrega las imágenes guardadas en la base de datos del puente.
type RecursosAlmacen struct {
	Ctx  context.Context
	Repo *almacen.RepoRecursos
}

func (r RecursosAlmacen) ImagenRecurso(nombre string) ([]byte, error) {
	rec, err := r.Repo.Obtener(r.Ctx, nombre)
	if errors.Is(err, almacen.ErrNoEncontrado) {
		return nil, ErrRecursoNoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return rec.PNG, nil
}

// Comandos ESC/POS.
const (
	esc = 0x1B
	gs  = 0x1D
	lf  = 0x0A
)

type renderizador struct {
	perfil   almacen.Impresora
	recursos Recursos
	cod      *codificador
	b        bytes.Buffer
	advs     []Advertencia
}

// Renderizar genera los bytes ESC/POS del documento completo (con sus copias) y las advertencias.
// Los errores son *contrato.Error con los códigos que dependen de la impresora
// (FILA_NO_CABE, OPERACION_NO_SOPORTADA, RECURSO_NO_ENCONTRADO, IMAGEN_INVALIDA).
func Renderizar(doc contrato.Documento, perfil almacen.Impresora, recursos Recursos) ([]byte, []Advertencia, error) {
	if perfil.Columnas < 1 || perfil.ColumnasFuenteB < 1 || perfil.AnchoPx < 8 {
		return nil, nil, fmt.Errorf("perfil de la impresora %q incompleto: columnas, columnas_fuente_b y ancho_px son obligatorios", perfil.Alias)
	}
	cod, err := nuevoCodificador(perfil.PaginaCodigos)
	if err != nil {
		return nil, nil, fmt.Errorf("impresora %q: %w", perfil.Alias, err)
	}
	if recursos == nil {
		recursos = RecursosMapa{}
	}
	r := &renderizador{perfil: perfil, recursos: recursos, cod: cod}

	r.inicializar()
	for i, op := range doc.Operaciones {
		if err := r.operacion(i, op); err != nil {
			return nil, nil, err
		}
	}
	if a := cod.advertencia(); a != nil {
		r.advs = append(r.advs, *a)
	}

	copias := max(doc.Copias, 1)
	return bytes.Repeat(r.b.Bytes(), copias), r.advs, nil
}

func (r *renderizador) escribir(b ...byte) { r.b.Write(b) }

func (r *renderizador) inicializar() {
	r.escribir(esc, '@')
	r.paginaCodigos()
}

func (r *renderizador) paginaCodigos() {
	r.escribir(esc, 't', byte(r.perfil.ComandoPaginaCodigos))
}

// estilo es el conjunto completo de atributos de texto; cada operación lo establece entero.
type estilo struct {
	alineacion         contrato.Alineacion
	negrita, subrayado bool
	invertido, fuenteB bool
	ancho, alto        int
}

var estiloNormal = estilo{alineacion: contrato.Izquierda, ancho: 1, alto: 1}

func bit(b bool) byte {
	if b {
		return 1
	}
	return 0
}

func codigoAlineacion(a contrato.Alineacion) byte {
	switch a {
	case contrato.Centro:
		return 1
	case contrato.Derecha:
		return 2
	}
	return 0
}

func (r *renderizador) aplicar(e estilo) {
	r.escribir(esc, 'a', codigoAlineacion(e.alineacion))
	r.escribir(esc, 'E', bit(e.negrita))
	r.escribir(esc, '-', bit(e.subrayado))
	r.escribir(gs, 'B', bit(e.invertido))
	r.escribir(gs, '!', byte((e.ancho-1)<<4|(e.alto-1)))
	r.escribir(esc, 'M', bit(e.fuenteB))
}

func (r *renderizador) lineas(ls [][]rune) {
	for _, l := range ls {
		r.b.Write(r.cod.codificar(l))
		r.escribir(lf)
	}
}

func (r *renderizador) advertir(codigo, mensaje, ruta string) {
	r.advs = append(r.advs, Advertencia{Codigo: codigo, Mensaje: mensaje, Ruta: ruta})
}

func noSoportada(i int, mensaje, sugerencia string) error {
	return contrato.NuevoError(contrato.ErrOperacionNoSoportada, mensaje, sugerencia, contrato.RutaOperacion(i), "capacidad")
}

func (r *renderizador) operacion(i int, op contrato.Operacion) error {
	switch o := op.(type) {
	case *contrato.Texto:
		e := estilo{alineacion: o.Alineacion, negrita: o.Negrita, subrayado: o.Subrayado, invertido: o.Invertido,
			fuenteB: o.Fuente == "B", ancho: o.Tamano[0], alto: o.Tamano[1]}
		cols := r.perfil.Columnas
		if e.fuenteB {
			cols = r.perfil.ColumnasFuenteB
		}
		r.aplicar(e)
		r.lineas(partir(r.cod.preparar(o.Valor), cols/e.ancho))
		r.aplicar(estiloNormal)

	case *contrato.Separador:
		c := r.cod.preparar(o.Caracter)
		linea := make([]rune, r.perfil.Columnas)
		for k := range linea {
			linea[k] = c[0]
		}
		r.aplicar(estiloNormal)
		r.lineas([][]rune{linea})

	case *contrato.Fila:
		anchos, err := repartirColumnas(r.perfil.Columnas, o.Columnas)
		if err != nil {
			return contrato.NuevoError(contrato.ErrFilaNoCabe,
				fmt.Sprintf("Las columnas no caben en la impresora %q (%d columnas): %v.", r.perfil.Alias, r.perfil.Columnas, err),
				"Reduce los porcentajes fijos o la cantidad de columnas.", contrato.RutaOperacion(i, "columnas"), "ancho_minimo")
		}
		celdas := make([][]rune, len(o.Columnas))
		alins := make([]contrato.Alineacion, len(o.Columnas))
		for k, c := range o.Columnas {
			celdas[k], alins[k] = r.cod.preparar(c.Texto), c.Alineacion
		}
		r.fila(celdas, anchos, alins, o.Negrita)

	case *contrato.Par:
		izq, der := r.cod.preparar(o.Izquierda), r.cod.preparar(o.Derecha)
		u := r.perfil.Columnas - 1
		d := min(len(der), u/2)
		r.fila([][]rune{izq, der}, []int{u - d, d}, []contrato.Alineacion{contrato.Izquierda, contrato.Derecha}, o.Negrita)

	case *contrato.Salto:
		r.aplicar(estiloNormal)
		r.escribir(esc, 'd', byte(o.Lineas))

	case *contrato.QR:
		return r.qr(i, o)

	case *contrato.CodigoBarras:
		return r.codigoBarras(i, o)

	case *contrato.Imagen:
		return r.imagen(i, o)

	case *contrato.Cortar:
		r.aplicar(estiloNormal)
		if !r.perfil.SoportaCorte {
			r.escribir(esc, 'd', 4)
			r.advertir(AdvSinCuchilla, "La impresora no tiene cuchilla: se avanzó el papel en lugar de cortar.", contrato.RutaOperacion(i))
			return nil
		}
		// GS V 65/66 n: avanza hasta la posición de corte (+n) y corta (65 total, 66 parcial).
		modo := byte(65)
		if o.Parcial {
			modo = 66
		}
		r.escribir(gs, 'V', modo, 0)

	case *contrato.AbrirCajon:
		if !r.perfil.SoportaCajon {
			r.advertir(AdvSinCajon, "La impresora no tiene cajón configurado: no se envió el pulso.", contrato.RutaOperacion(i))
			return nil
		}
		// ESC p m t1 t2: pulso de 50 ms encendido y 500 ms apagado.
		r.escribir(esc, 'p', bit(o.Pin == 5), 25, 250)

	case *contrato.Raw:
		r.b.Write(o.Datos)
		// Los bytes crudos pueden cambiar la página de códigos; las siguientes operaciones la necesitan.
		r.paginaCodigos()

	default:
		return fmt.Errorf("operación %T sin renderizador", op)
	}
	return nil
}

func (r *renderizador) fila(celdas [][]rune, anchos []int, alins []contrato.Alineacion, negrita bool) {
	e := estiloNormal
	e.negrita = negrita
	r.aplicar(e)
	r.lineas(componerFila(celdas, anchos, alins))
	r.aplicar(estiloNormal)
}

// imprimirMapa envía un mapa de bits con GS v 0 en bandas de hasta 255 líneas. La alineación se
// resuelve con un margen de puntos en blanco, que funciona en cualquier impresora.
func (r *renderizador) imprimirMapa(m *Mapa, a contrato.Alineacion) {
	r.aplicar(estiloNormal)
	desde := desplazamiento(m.Ancho, r.perfil.AnchoPx, a)
	bytesFila := (desde + m.Ancho + 7) / 8
	for y0 := 0; y0 < m.Alto; y0 += 255 {
		alto := min(255, m.Alto-y0)
		r.escribir(gs, 'v', '0', 0, byte(bytesFila), byte(bytesFila>>8), byte(alto), byte(alto>>8))
		fila := make([]byte, bytesFila)
		for y := y0; y < y0+alto; y++ {
			clear(fila)
			for x := range m.Ancho {
				if m.En(x, y) {
					px := desde + x
					fila[px/8] |= 0x80 >> (px % 8)
				}
			}
			r.b.Write(fila)
		}
	}
}

func (r *renderizador) imagen(i int, o *contrato.Imagen) error {
	if !r.perfil.SoportaImagen {
		return noSoportada(i, fmt.Sprintf("La impresora %q no imprime imágenes.", r.perfil.Alias),
			"Quita la operación imagen o usa otra impresora.")
	}
	datos := o.PNG
	if o.Recurso != "" {
		var err error
		datos, err = r.recursos.ImagenRecurso(o.Recurso)
		if errors.Is(err, ErrRecursoNoEncontrado) {
			return contrato.NuevoError(contrato.ErrRecursoNoEncontrado,
				fmt.Sprintf("No existe la imagen %q en el puente.", o.Recurso),
				"Súbela en el panel del puente o envíala en base64.", contrato.RutaOperacion(i, "recurso"), "recurso")
		}
		if err != nil {
			return err
		}
	}
	m, err := convertirPNG(datos, r.perfil.AnchoPx*o.AnchoMax/100)
	if err != nil {
		return contrato.NuevoError(contrato.ErrImagenInvalida, "La imagen no es un PNG válido.",
			"Revisa la imagen guardada o vuelve a subirla.", contrato.RutaOperacion(i), "png_valido")
	}
	r.imprimirMapa(m, o.Alineacion)
	return nil
}

var codigoCorreccion = map[string]byte{"L": 48, "M": 49, "Q": 50, "H": 51}

func (r *renderizador) qr(i int, o *contrato.QR) error {
	mods, err := modulosQR(o.Valor, o.Correccion)
	if err != nil {
		return noSoportada(i, "El contenido no cabe en un código QR con ese nivel de corrección.", "Acorta el valor o usa una corrección menor.")
	}
	n := len(mods)
	tamano := o.Tamano
	for tamano > 1 && n*tamano > r.perfil.AnchoPx {
		tamano--
	}
	if n*tamano > r.perfil.AnchoPx {
		return noSoportada(i, fmt.Sprintf("El código QR (%d módulos) no cabe en el papel.", n), "Acorta el valor o usa una corrección menor.")
	}
	if tamano != o.Tamano {
		r.advertir(AdvQRReducido, fmt.Sprintf("El QR se imprimió con tamaño %d en lugar de %d para que quepa en el papel.", tamano, o.Tamano),
			contrato.RutaOperacion(i, "tamano"))
	}

	if !r.perfil.SoportaQRNativo {
		if !r.perfil.SoportaImagen {
			return noSoportada(i, fmt.Sprintf("La impresora %q no tiene QR nativo ni imprime imágenes.", r.perfil.Alias), "Usa otra impresora.")
		}
		m, err := MapaQR(o.Valor, o.Correccion, tamano)
		if err != nil {
			return err
		}
		r.imprimirMapa(m, o.Alineacion)
		return nil
	}

	datos := []byte(o.Valor)
	e := estiloNormal
	e.alineacion = o.Alineacion
	r.aplicar(e)
	r.escribir(gs, '(', 'k', 4, 0, '1', 'A', '2', 0)                         // modelo 2
	r.escribir(gs, '(', 'k', 3, 0, '1', 'C', byte(tamano))                   // tamaño del módulo
	r.escribir(gs, '(', 'k', 3, 0, '1', 'E', codigoCorreccion[o.Correccion]) // corrección
	largo := len(datos) + 3
	r.escribir(gs, '(', 'k', byte(largo), byte(largo>>8), '1', 'P', '0') // almacenar
	r.b.Write(datos)
	// Imprimir. Deja la posición al inicio de la línea: un LF aquí agregaría una línea en blanco.
	r.escribir(gs, '(', 'k', 3, 0, '1', 'Q', '0')
	r.aplicar(estiloNormal)
	return nil
}

var codigoSimbologia = map[string]byte{
	contrato.UPCA: 65, contrato.EAN13: 67, contrato.EAN8: 68, contrato.CODE39: 69, contrato.ITF: 70, contrato.CODE128: 73,
}

var codigoHRI = map[string]byte{"ninguno": 0, "arriba": 1, "abajo": 2}

func (r *renderizador) codigoBarras(i int, o *contrato.CodigoBarras) error {
	mods, err := Modulos(o.Simbologia, o.Valor)
	if err != nil {
		return contrato.NuevoError(contrato.ErrCodigoBarrasInvalido, "No se pudo codificar el código de barras: "+err.Error()+".",
			"Revisa el valor para la simbología indicada.", contrato.RutaOperacion(i, "valor"), "simbologia")
	}
	// Ancho de módulo: 3 puntos si cabe, si no 2 (el mínimo que leen los lectores con fiabilidad).
	modulo := 3
	if len(mods)*modulo > r.perfil.AnchoPx {
		modulo = 2
	}
	if len(mods)*modulo > r.perfil.AnchoPx {
		return noSoportada(i, fmt.Sprintf("El código de barras (%d módulos) no cabe en el papel.", len(mods)),
			"Acorta el valor o usa una simbología más compacta.")
	}

	if !r.perfil.SoportaCodigoBarrasNativo {
		if !r.perfil.SoportaImagen {
			return noSoportada(i, fmt.Sprintf("La impresora %q no tiene códigos de barras nativos ni imprime imágenes.", r.perfil.Alias), "Usa otra impresora.")
		}
		texto := [][]rune{r.cod.preparar(textoLegible(o))}
		hri := estiloNormal
		hri.alineacion = o.Alineacion
		if o.Texto == "arriba" {
			r.aplicar(hri)
			r.lineas(texto)
		}
		r.imprimirMapa(mapaBarras(mods, modulo, o.Alto), o.Alineacion)
		if o.Texto == "abajo" {
			r.aplicar(hri)
			r.lineas(texto)
		}
		r.aplicar(estiloNormal)
		return nil
	}

	datos := []byte(o.Valor)
	if o.Simbologia == contrato.CODE128 {
		// Juego B explícito; la llave '{' se envía duplicada.
		datos = append([]byte("{B"), bytes.ReplaceAll(datos, []byte("{"), []byte("{{"))...)
	}
	e := estiloNormal
	e.alineacion = o.Alineacion
	r.aplicar(e)
	r.escribir(gs, 'H', codigoHRI[o.Texto]) // posición del texto legible
	r.escribir(gs, 'f', 0)                  // fuente A para el texto legible
	r.escribir(gs, 'h', byte(o.Alto))
	r.escribir(gs, 'w', byte(modulo))
	r.escribir(gs, 'k', codigoSimbologia[o.Simbologia], byte(len(datos)))
	r.b.Write(datos) // imprime y deja la posición al inicio de la línea, como el QR
	r.aplicar(estiloNormal)
	return nil
}

// textoLegible es el texto bajo/sobre las barras cuando se dibujan como imagen.
func textoLegible(o *contrato.CodigoBarras) string {
	switch o.Simbologia {
	case contrato.EAN13:
		return completarGS1(o.Valor, 13)
	case contrato.EAN8:
		return completarGS1(o.Valor, 8)
	case contrato.UPCA:
		return completarGS1(o.Valor, 12)
	case contrato.CODE39:
		return "*" + o.Valor + "*"
	}
	return o.Valor
}
