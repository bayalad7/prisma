package marca

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"flag"
	"fmt"
	"image/color"
	"image/png"
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
			if !bytes.Equal(enDisco, datos) {
				t.Errorf("%s no coincide con la geometría de internal/marca: go test ./internal/marca -actualizar", ruta)
			}
			continue
		}
		// Las imágenes se comparan por sus píxeles, no por sus bytes: el compresor de PNG cambia
		// entre versiones de Go, y el redondeo de punto flotante, entre arquitecturas.
		if err := mismasImagenes(enDisco, datos); err != nil {
			t.Errorf("%s no coincide con la geometría de internal/marca (%v): go test ./internal/marca -actualizar", ruta, err)
		}
	}
}

// pngsDe devuelve los PNG de un archivo: él mismo, o los que lleva dentro un .ico o un .icns.
func pngsDe(datos []byte) ([][]byte, error) {
	switch {
	case bytes.HasPrefix(datos, []byte("\x89PNG")):
		return [][]byte{datos}, nil
	case bytes.HasPrefix(datos, []byte("icns")):
		var res [][]byte
		for i := 8; i+8 <= len(datos); {
			n := int(binary.BigEndian.Uint32(datos[i+4:]))
			if n < 8 || i+n > len(datos) {
				return nil, fmt.Errorf("entrada .icns inválida en %d", i)
			}
			res = append(res, datos[i+8:i+n])
			i += n
		}
		return res, nil
	case len(datos) > 6 && binary.LittleEndian.Uint16(datos[2:]) == 1:
		var res [][]byte
		for k := 0; k < int(binary.LittleEndian.Uint16(datos[4:])); k++ {
			e := 6 + 16*k
			n := int(binary.LittleEndian.Uint32(datos[e+8:]))
			o := int(binary.LittleEndian.Uint32(datos[e+12:]))
			if o+n > len(datos) {
				return nil, fmt.Errorf("entrada .ico inválida %d", k)
			}
			res = append(res, datos[o:o+n])
		}
		return res, nil
	}
	return nil, fmt.Errorf("formato desconocido")
}

// mismasImagenes compara dos PNG, .ico o .icns imagen por imagen, en color premultiplicado y con una
// tolerancia de 2/255 por canal.
func mismasImagenes(a, b []byte) error {
	pa, err := pngsDe(a)
	if err != nil {
		return err
	}
	pb, err := pngsDe(b)
	if err != nil {
		return err
	}
	if len(pa) != len(pb) {
		return fmt.Errorf("%d imágenes, se esperaban %d", len(pa), len(pb))
	}
	for k := range pa {
		ia, err := png.Decode(bytes.NewReader(pa[k]))
		if err != nil {
			return err
		}
		ib, err := png.Decode(bytes.NewReader(pb[k]))
		if err != nil {
			return err
		}
		if ia.Bounds() != ib.Bounds() {
			return fmt.Errorf("imagen %d mide %v, se esperaba %v", k, ia.Bounds(), ib.Bounds())
		}
		r := ia.Bounds()
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				ca := color.RGBAModel.Convert(ia.At(x, y)).(color.RGBA)
				cb := color.RGBAModel.Convert(ib.At(x, y)).(color.RGBA)
				for _, d := range []int{int(ca.R) - int(cb.R), int(ca.G) - int(cb.G), int(ca.B) - int(cb.B), int(ca.A) - int(cb.A)} {
					if d > 2 || d < -2 {
						return fmt.Errorf("imagen %d, píxel (%d, %d): %v, se esperaba %v", k, x, y, ca, cb)
					}
				}
			}
		}
	}
	return nil
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

// TestLosInstaladoresUsanLaMarca: los tres instaladores ponen el logo de la marca donde se busca
// el programa (menú Inicio, menú de aplicaciones de Linux, Spotlight y Launchpad).
func TestLosInstaladoresUsanLaMarca(t *testing.T) {
	leer := func(ruta string) string {
		datos, err := os.ReadFile(filepath.Join(raiz, filepath.FromSlash(ruta)))
		if err != nil {
			t.Fatal(err)
		}
		return string(datos)
	}
	exigir := func(ruta string, partes ...string) {
		texto := leer(ruta)
		for _, p := range partes {
			if !strings.Contains(texto, p) {
				t.Errorf("%s no contiene %q", ruta, p)
			}
		}
	}
	exigir("empaquetado/windows/prisma.iss", `SetupIconFile=..\marca\prisma.ico`, `Source: "..\marca\prisma.ico"`,
		`Name: "{autoprograms}\PRISMA"`, `IconFilename: "{app}\prisma.ico"`)
	deb := []string{"src: empaquetado/linux/prisma.desktop", "src: empaquetado/marca/prisma-icono.svg"}
	for _, n := range TamanosLinux {
		deb = append(deb, fmt.Sprintf("src: empaquetado/marca/png/prisma-%d.png\n    dst: /usr/share/icons/hicolor/%dx%d/apps/prisma.png", n, n, n))
	}
	exigir("empaquetado/linux/nfpm.yaml", deb...)
	exigir("empaquetado/linux/prisma.desktop", "Icon=prisma", "Name=PRISMA")
	exigir("empaquetado/macos/construir-pkg.sh", "empaquetado/marca/prisma.icns", "Applications/PRISMA.app")
	exigir("empaquetado/macos/PRISMA.app/Contents/Info.plist", "<string>prisma</string>", "<string>abrir-panel</string>")
}
