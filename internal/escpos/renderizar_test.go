package escpos

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/contrato"
)

func perfil80() almacen.Impresora {
	return almacen.Impresora{
		Alias: "caja", AnchoMM: 80, Columnas: 48, ColumnasFuenteB: 64, AnchoPx: 576,
		PaginaCodigos: "CP858", ComandoPaginaCodigos: 19,
		SoportaQRNativo: true, SoportaCorte: true, SoportaCajon: true, SoportaImagen: true, SoportaCodigoBarrasNativo: true,
	}
}

func doc(ops ...contrato.Operacion) contrato.Documento {
	return contrato.Documento{Version: "1", Copias: 1, Operaciones: ops}
}

func renderizar(t *testing.T, d contrato.Documento, p almacen.Impresora) ([]byte, []Advertencia) {
	t.Helper()
	b, advs, err := Renderizar(d, p, RecursosMapa{"logo": pngPrueba(40, 20)})
	if err != nil {
		t.Fatal(err)
	}
	return b, advs
}

func pngPrueba(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			if x < w/2 {
				img.Set(x, y, color.Black)
			} else {
				img.Set(x, y, color.Transparent)
			}
		}
	}
	var b bytes.Buffer
	png.Encode(&b, img)
	return b.Bytes()
}

var inicio = []byte{esc, '@', esc, 't', 19}

func TestInicioYEstilosCompletos(t *testing.T) {
	b, _ := renderizar(t, doc(&contrato.Texto{Valor: "Hola", Alineacion: contrato.Centro, Negrita: true, Subrayado: true,
		Invertido: true, Tamano: [2]int{2, 3}, Fuente: "B"}), perfil80())
	want := append(append([]byte{}, inicio...),
		esc, 'a', 1, esc, 'E', 1, esc, '-', 1, gs, 'B', 1, gs, '!', 0x12, esc, 'M', 1,
		'H', 'o', 'l', 'a', lf,
		esc, 'a', 0, esc, 'E', 0, esc, '-', 0, gs, 'B', 0, gs, '!', 0, esc, 'M', 0)
	if !bytes.Equal(b, want) {
		t.Fatalf("\n got % X\nwant % X", b, want)
	}
}

func TestTextoParteSegunFuenteYTamano(t *testing.T) {
	largo := strings.Repeat("palabra ", 10) // 79 caracteres
	contar := func(tx *contrato.Texto) int {
		b, _ := renderizar(t, doc(tx), perfil80())
		return bytes.Count(b, []byte{lf})
	}
	// Fuente A 1x: 48 columnas → 2 líneas; tamaño 2x: 24 → 4 líneas; fuente B 1x: 64 → 2 líneas.
	if n := contar(&contrato.Texto{Valor: largo, Tamano: [2]int{1, 1}, Fuente: "A"}); n != 2 {
		t.Errorf("1x A: %d líneas", n)
	}
	if n := contar(&contrato.Texto{Valor: largo, Tamano: [2]int{2, 2}, Fuente: "A"}); n != 4 {
		t.Errorf("2x A: %d líneas", n)
	}
	if n := contar(&contrato.Texto{Valor: largo, Tamano: [2]int{1, 1}, Fuente: "B"}); n != 2 {
		t.Errorf("1x B: %d líneas", n)
	}
}

func lineasTexto(b []byte) []string {
	var out []string
	for _, l := range bytes.Split(b, []byte{lf}) {
		// Quita los comandos de estilo que preceden al texto (6 comandos de 3 bytes).
		if i := bytes.LastIndex(l, []byte{esc, 'M'}); i >= 0 {
			l = l[i+3:]
		}
		out = append(out, string(l))
	}
	return out
}

func TestFilaYParConAcentosYAuto(t *testing.T) {
	d := doc(
		&contrato.Fila{Columnas: []contrato.Columna{
			{Texto: "2", Ancho: 10, Alineacion: contrato.Derecha},
			{Texto: "Jabón de tocador con aroma a lavanda y manzanilla", Ancho: contrato.AnchoAuto, Alineacion: contrato.Izquierda},
			{Texto: "$45.00", Ancho: 20, Alineacion: contrato.Derecha},
		}},
		&contrato.Par{Izquierda: "Cambio en efectivo para el cliente número uno", Derecha: "$36.50"},
	)
	p := perfil80()
	p.PaginaCodigos, p.ComandoPaginaCodigos = "WPC1252", 16 // 1 byte por carácter y legible como Latin-1
	b, advs := renderizar(t, d, p)
	if len(advs) != 0 {
		t.Fatalf("no esperaba advertencias: %+v", advs)
	}
	got := lineasTexto(b)
	// U = 46: 10 % → 4, 20 % → 9, auto → 33 («…aroma a lavanda» mide 36 y no cabe).
	want := []string{
		"   2 Jabón de tocador con aroma a         $45.00",
		"     lavanda y manzanilla                       ",
		// par: U = 47, derecha = 6, izquierda = 41.
		"Cambio en efectivo para el cliente número $36.50",
		"uno                                             ",
	}
	for i, w := range want {
		if g := decodificar1252(got[i]); g != w {
			t.Errorf("renglón %d:\n got %q\nwant %q", i, g, w)
		}
		if n := len([]rune(w)); n != 48 {
			t.Fatalf("la expectativa %d mide %d", i, n)
		}
	}
}

func decodificar1252(s string) string {
	cod, _ := nuevoCodificador("WPC1252")
	var out []rune
	for _, b := range []byte(s) {
		if b < 0x80 {
			out = append(out, rune(b))
		} else {
			out = append(out, cod.tabla.DecodeByte(b))
		}
	}
	return string(out)
}

func TestFilaNoCabe(t *testing.T) {
	_, _, err := Renderizar(doc(&contrato.Fila{Columnas: []contrato.Columna{
		{Texto: "a", Ancho: 50}, {Texto: "b", Ancho: 50}, {Texto: "c", Ancho: contrato.AnchoAuto},
	}}), perfil80(), nil)
	exigirCodigo(t, err, contrato.ErrFilaNoCabe)
}

func exigirCodigo(t *testing.T, err error, codigo string) {
	t.Helper()
	var ce *contrato.Error
	if !errors.As(err, &ce) || ce.Codigo != codigo {
		t.Fatalf("esperaba %s, obtuve %v", codigo, err)
	}
}

func TestCopiasRepitenElTicket(t *testing.T) {
	d := doc(&contrato.Texto{Valor: "x", Tamano: [2]int{1, 1}, Fuente: "A"}, &contrato.Cortar{})
	una, _ := renderizar(t, d, perfil80())
	d.Copias = 3
	tres, _ := renderizar(t, d, perfil80())
	if !bytes.Equal(tres, bytes.Repeat(una, 3)) {
		t.Fatal("cada copia debe repetir el ticket completo")
	}
}

func TestCorteYCajonSegunPerfil(t *testing.T) {
	d := doc(&contrato.AbrirCajon{Pin: 5}, &contrato.Cortar{Parcial: true})
	b, advs := renderizar(t, d, perfil80())
	if !bytes.Contains(b, []byte{esc, 'p', 1, 25, 250}) || !bytes.HasSuffix(b, []byte{gs, 'V', 66, 0}) || len(advs) != 0 {
		t.Fatalf("con capacidades: % X %+v", b, advs)
	}

	p := perfil80()
	p.SoportaCorte, p.SoportaCajon = false, false
	b, advs = renderizar(t, d, p)
	if bytes.Contains(b, []byte{esc, 'p'}) || bytes.Contains(b, []byte{gs, 'V'}) || !bytes.HasSuffix(b, []byte{esc, 'd', 4}) {
		t.Fatalf("sin capacidades: % X", b)
	}
	if len(advs) != 2 || advs[0].Codigo != AdvSinCajon || advs[1].Codigo != AdvSinCuchilla {
		t.Fatalf("advertencias: %+v", advs)
	}
}

func TestRawRestableceLaPagina(t *testing.T) {
	b, _ := renderizar(t, doc(&contrato.Raw{Datos: []byte{esc, 't', 0}}), perfil80())
	want := append(append([]byte{}, inicio...), esc, 't', 0, esc, 't', 19)
	if !bytes.Equal(b, want) {
		t.Fatalf("\n got % X\nwant % X", b, want)
	}
}

func TestQRNativoYComoImagen(t *testing.T) {
	q := &contrato.QR{Valor: "PRISMA", Tamano: 6, Correccion: "H", Alineacion: contrato.Centro}
	b, _ := renderizar(t, doc(q), perfil80())
	for _, parte := range [][]byte{
		{gs, '(', 'k', 3, 0, '1', 'C', 6},
		{gs, '(', 'k', 3, 0, '1', 'E', 51},
		append([]byte{gs, '(', 'k', 9, 0, '1', 'P', '0'}, "PRISMA"...),
		{gs, '(', 'k', 3, 0, '1', 'Q', '0'},
	} {
		if !bytes.Contains(b, parte) {
			t.Errorf("falta % X", parte)
		}
	}

	p := perfil80()
	p.SoportaQRNativo = false
	b, _ = renderizar(t, doc(q), p)
	if bytes.Contains(b, []byte{gs, '(', 'k'}) || !bytes.Contains(b, []byte{gs, 'v', '0'}) {
		t.Fatal("sin QR nativo debe imprimirse como imagen")
	}

	p.SoportaImagen = false
	_, _, err := Renderizar(doc(q), p, nil)
	exigirCodigo(t, err, contrato.ErrOperacionNoSoportada)
}

func TestQRSeReduceParaCaber(t *testing.T) {
	p := perfil80()
	p.AnchoPx = 120
	_, advs := renderizar(t, doc(&contrato.QR{Valor: "https://facturas.example.com/t/FAC-A-000981", Tamano: 16, Correccion: "M"}), p)
	if len(advs) != 1 || advs[0].Codigo != AdvQRReducido {
		t.Fatalf("advertencias: %+v", advs)
	}
}

func TestCodigoBarrasNativoYComoImagen(t *testing.T) {
	cb := &contrato.CodigoBarras{Valor: "Pedido{1}", Simbologia: contrato.CODE128, Texto: "abajo", Alto: 60, Alineacion: contrato.Centro}
	b, _ := renderizar(t, doc(cb), perfil80())
	datos := []byte("{BPedido{{1}")
	want := append([]byte{gs, 'H', 2, gs, 'f', 0, gs, 'h', 60, gs, 'w', 3, gs, 'k', 73, byte(len(datos))}, datos...)
	if !bytes.Contains(b, want) {
		t.Fatalf("comando nativo:\n got % X\nwant % X", b, want)
	}

	p := perfil80()
	p.SoportaCodigoBarrasNativo = false
	b, _ = renderizar(t, doc(cb), p)
	if bytes.Contains(b, []byte{gs, 'k'}) || !bytes.Contains(b, []byte{gs, 'v', '0'}) || !bytes.Contains(b, []byte("Pedido{1}\n")) {
		t.Fatal("sin códigos nativos debe imprimir imagen y el texto legible")
	}
}

func TestCodigoBarrasDemasiadoAncho(t *testing.T) {
	p := perfil80()
	p.AnchoPx = 384
	_, _, err := Renderizar(doc(&contrato.CodigoBarras{Valor: strings.Repeat("A", 40), Simbologia: contrato.CODE128, Alto: 50}), p, nil)
	exigirCodigo(t, err, contrato.ErrOperacionNoSoportada)
}

func TestImagenEnBandasYAlineada(t *testing.T) {
	alto := 600
	b, _ := renderizar(t, doc(&contrato.Imagen{PNG: pngPrueba(100, alto), Alineacion: contrato.Derecha, AnchoMax: 100}), perfil80())
	// 600 líneas → bandas de 255, 255 y 90; alineada a la derecha ocupa los 576 puntos (72 bytes por fila).
	bandas := bytes.Count(b, []byte{gs, 'v', '0', 0, 72, 0})
	if bandas != 3 || !bytes.Contains(b, []byte{gs, 'v', '0', 0, 72, 0, 90, 0}) {
		t.Fatalf("bandas: %d", bandas)
	}
}

func TestImagenNuncaSeAgrandaYRespetaAnchoMax(t *testing.T) {
	m, err := convertirPNG(pngPrueba(100, 50), 576)
	if err != nil || m.Ancho != 100 || m.Alto != 50 {
		t.Fatalf("no debe agrandarse: %+v %v", m, err)
	}
	m, _ = convertirPNG(pngPrueba(1000, 500), 288)
	if m.Ancho != 288 || m.Alto != 144 {
		t.Fatalf("reducción proporcional: %dx%d", m.Ancho, m.Alto)
	}
	// La mitad izquierda es negra y la derecha transparente (blanca).
	if !m.En(10, 10) || m.En(280, 10) {
		t.Fatal("colores mal convertidos")
	}
}

func TestRecursoNoEncontradoYSinImagenes(t *testing.T) {
	_, _, err := Renderizar(doc(&contrato.Imagen{Recurso: "no-existe", AnchoMax: 100}), perfil80(), nil)
	exigirCodigo(t, err, contrato.ErrRecursoNoEncontrado)

	p := perfil80()
	p.SoportaImagen = false
	_, _, err = Renderizar(doc(&contrato.Imagen{Recurso: "logo", AnchoMax: 100}), p, nil)
	exigirCodigo(t, err, contrato.ErrOperacionNoSoportada)
}

func TestPerfilIncompleto(t *testing.T) {
	p := perfil80()
	p.PaginaCodigos = "UTF8"
	if _, _, err := Renderizar(doc(), p, nil); err == nil {
		t.Fatal("esperaba error de página de códigos")
	}
	p = perfil80()
	p.Columnas = 0
	if _, _, err := Renderizar(doc(), p, nil); err == nil {
		t.Fatal("esperaba error de perfil incompleto")
	}
}
