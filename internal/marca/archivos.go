package marca

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
)

// Icono rasteriza el icono de app en un cuadrado de lado n. De 24 px para abajo usa la variante de
// bandeja con menos margen, para que el puente y el ticket no se empasten. Con macos, la placa
// ocupa el 80 % del lienzo, como piden las guías de iconos de macOS.
func Icono(n int, macos bool) (*image.NRGBA, error) {
	u := float64(n) / 32 // píxeles por unidad
	lienzo, margenPlaca := 32.0, 0.0
	if macos {
		margenPlaca = 3.2
		lienzo = 32 - 2*margenPlaca
	}
	figs, margen := Isotipo, 3.0
	if n <= 24 {
		figs, margen = Bandeja, 2.0
	}
	escalaPlaca := lienzo / 32
	capas := []capa{{d: placa, color: hexA(Placa), escala: u * escalaPlaca, dx: margenPlaca * u, dy: margenPlaca * u}}
	escala := (32 - 2*margen) / 32 * escalaPlaca
	desp := (margenPlaca + margen*escalaPlaca) * u
	for _, f := range figs {
		col := TintaPlaca
		if f.Acento {
			col = AcentoPlaca
		}
		capas = append(capas, capa{d: f.D, color: hexA(col), escala: u * escala, dx: desp, dy: desp})
	}
	return pintar(n, capas)
}

func pngDe(img image.Image) ([]byte, error) {
	var b bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	err := enc.Encode(&b, img)
	return b.Bytes(), err
}

func iconoPNG(n int, macos bool) ([]byte, error) {
	img, err := Icono(n, macos)
	if err != nil {
		return nil, err
	}
	return pngDe(img)
}

// ICO arma un .ico con una imagen PNG por tamaño (Windows Vista en adelante).
func ICO(tamanos ...int) ([]byte, error) {
	var cab, datos bytes.Buffer
	binary.Write(&cab, binary.LittleEndian, [3]uint16{0, 1, uint16(len(tamanos))})
	desp := 6 + 16*len(tamanos)
	for _, n := range tamanos {
		p, err := iconoPNG(n, false)
		if err != nil {
			return nil, err
		}
		lado := uint8(n)
		if n >= 256 {
			lado = 0
		}
		cab.Write([]byte{lado, lado, 0, 0})
		binary.Write(&cab, binary.LittleEndian, [2]uint16{1, 32})
		binary.Write(&cab, binary.LittleEndian, [2]uint32{uint32(len(p)), uint32(desp)})
		desp += len(p)
		datos.Write(p)
	}
	return append(cab.Bytes(), datos.Bytes()...), nil
}

// ICNS arma un .icns de macOS con imágenes PNG.
func ICNS() ([]byte, error) {
	tipos := []struct {
		tipo string
		n    int
	}{{"icp4", 16}, {"icp5", 32}, {"ic11", 32}, {"icp6", 64}, {"ic12", 64}, {"ic07", 128},
		{"ic08", 256}, {"ic13", 256}, {"ic09", 512}, {"ic14", 512}, {"ic10", 1024}}
	var cuerpo bytes.Buffer
	for _, t := range tipos {
		p, err := iconoPNG(t.n, true)
		if err != nil {
			return nil, err
		}
		cuerpo.WriteString(t.tipo)
		binary.Write(&cuerpo, binary.BigEndian, uint32(8+len(p)))
		cuerpo.Write(p)
	}
	var b bytes.Buffer
	b.WriteString("icns")
	binary.Write(&b, binary.BigEndian, uint32(8+cuerpo.Len()))
	b.Write(cuerpo.Bytes())
	return b.Bytes(), nil
}

// TamanosLinux son los iconos PNG del tema hicolor que instala el .deb.
var TamanosLinux = []int{16, 24, 32, 48, 64, 128, 256, 512}

// Archivos devuelve todos los archivos de la marca con su ruta desde la raíz del repositorio.
func Archivos() (map[string][]byte, error) {
	a := map[string][]byte{
		"empaquetado/marca/prisma.svg":                   []byte(SVGIsotipo(Isotipo, Claro)),
		"empaquetado/marca/prisma-oscuro.svg":            []byte(SVGIsotipo(Isotipo, Oscuro)),
		"empaquetado/marca/prisma-una-tinta.svg":         []byte(SVGIsotipo(Isotipo, UnaTinta)),
		"empaquetado/marca/prisma-bandeja.svg":           []byte(SVGIsotipo(Bandeja, Claro)),
		"empaquetado/marca/prisma-bandeja-oscuro.svg":    []byte(SVGIsotipo(Bandeja, Oscuro)),
		"empaquetado/marca/prisma-horizontal.svg":        []byte(SVGHorizontal(Claro)),
		"empaquetado/marca/prisma-horizontal-oscuro.svg": []byte(SVGHorizontal(Oscuro)),
		"empaquetado/marca/prisma-icono.svg":             []byte(SVGIcono()),
		"internal/admin/web/favicon.svg":                 []byte(SVGIcono()),
	}
	ico, err := ICO(16, 24, 32, 48, 64, 128, 256)
	if err != nil {
		return nil, err
	}
	a["empaquetado/marca/prisma.ico"] = ico
	if a["internal/admin/web/favicon.ico"], err = ICO(16, 32, 48); err != nil {
		return nil, err
	}
	if a["empaquetado/marca/prisma.icns"], err = ICNS(); err != nil {
		return nil, err
	}
	for _, n := range TamanosLinux {
		p, err := iconoPNG(n, false)
		if err != nil {
			return nil, err
		}
		a[fmt.Sprintf("empaquetado/marca/png/prisma-%d.png", n)] = p
	}
	return a, nil
}
