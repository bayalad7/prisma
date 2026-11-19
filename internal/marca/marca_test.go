package marca

import (
	"bytes"
	"encoding/xml"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var actualizar = flag.Bool("actualizar", false, "reescribe los archivos de la marca")

const raiz = "../.."

// TestArchivosDeLaMarca: los SVG, PNG, .ico y .icns del repositorio salen de la geometría de este
// paquete. Si cambias la marca: go test ./internal/marca -actualizar
func TestArchivosDeLaMarca(t *testing.T) {
	archivos, err := Archivos()
	if err != nil {
		t.Fatal(err)
	}
	for ruta, datos := range archivos {
		destino := filepath.Join(raiz, filepath.FromSlash(ruta))
		if *actualizar {
			if err := os.MkdirAll(filepath.Dir(destino), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(destino, datos, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		enDisco, err := os.ReadFile(destino)
		if err != nil {
			t.Errorf("%s: %v (genera los archivos con go test ./internal/marca -actualizar)", ruta, err)
			continue
		}
		if strings.HasSuffix(ruta, ".svg") {
			// Git en Windows puede convertir los saltos de línea de los archivos de texto.
			enDisco = bytes.ReplaceAll(enDisco, []byte("\r\n"), []byte("\n"))
		}
		if !bytes.Equal(enDisco, datos) {
			t.Errorf("%s no coincide con la geometría de internal/marca: go test ./internal/marca -actualizar", ruta)
		}
	}
}

// TestSVGLimpios: los SVG de la marca solo tienen formas básicas, sin fuentes, estilos, scripts,
// imágenes ni referencias externas.
func TestSVGLimpios(t *testing.T) {
	permitidos := map[string]bool{"svg": true, "g": true, "path": true, "circle": true}
	for _, svg := range []string{SVGIsotipo(Isotipo, Claro), SVGIsotipo(Bandeja, Oscuro), SVGHorizontal(Claro), SVGIcono()} {
		dec := xml.NewDecoder(strings.NewReader(svg))
		for {
			tok, err := dec.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("SVG inválido: %v\n%s", err, svg)
			}
			if e, ok := tok.(xml.StartElement); ok && !permitidos[e.Name.Local] {
				t.Errorf("elemento <%s> no permitido en %s", e.Name.Local, svg)
			}
		}
	}
}

// TestIconosDibujanElLogo: el rasterizador pinta la placa, la tinta y el acento donde deben ir.
func TestIconosDibujanElLogo(t *testing.T) {
	img, err := Icono(64, false)
	if err != nil {
		t.Fatal(err)
	}
	en := func(x, y float64) [4]uint8 { // unidades de 32
		i := img.PixOffset(int(x*2), int(y*2))
		return [4]uint8(img.Pix[i : i+4])
	}
	placa, acento, tinta := hexA(Placa), hexA(AcentoPlaca), hexA(TintaPlaca)
	casos := []struct {
		nombre string
		x, y   float64
		quiere [4]uint8
	}{
		{"esquina fuera de la placa", 0.2, 0.2, [4]uint8{0, 0, 0, 0}},
		{"placa", 16, 1.5, [4]uint8{placa.R, placa.G, placa.B, 255}},
		{"puente", 16, 16, [4]uint8{acento.R, acento.G, acento.B, 255}},
		{"documento", 10, 8, [4]uint8{tinta.R, tinta.G, tinta.B, 255}},
		{"ticket", 16, 21, [4]uint8{tinta.R, tinta.G, tinta.B, 255}},
	}
	for _, c := range casos {
		if got := en(c.x, c.y); got != c.quiere {
			t.Errorf("%s (%.1f, %.1f): %v, se esperaba %v", c.nombre, c.x, c.y, got, c.quiere)
		}
	}
}

