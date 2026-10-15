package contrato

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const dirEjemplos = "../../ejemplos"

func leer(t *testing.T, ruta string) []byte {
	t.Helper()
	datos, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	return datos
}

func TestEjemplosValidos(t *testing.T) {
	archivos, _ := filepath.Glob(filepath.Join(dirEjemplos, "validos", "*.json"))
	if len(archivos) < 6 {
		t.Fatalf("se esperaban al menos 6 ejemplos válidos, hay %d", len(archivos))
	}
	for _, a := range archivos {
		t.Run(filepath.Base(a), func(t *testing.T) {
			doc, err := Decodificar(leer(t, a))
			if err != nil {
				t.Fatal(err)
			}
			if len(doc.Operaciones) == 0 || doc.Version != "1" || doc.Copias < 1 {
				t.Fatalf("documento incompleto: %+v", doc)
			}
		})
	}
}

// entradaIndice es una fila de ejemplos/invalidos/indice.json.
type entradaIndice struct {
	Archivo string `json:"archivo"`
	Codigo  string `json:"codigo"`
	Detecta string `json:"detecta"`
	Ruta    string `json:"ruta"`
	Regla   string `json:"regla"`
}

// dependeDeOtroPaso: casos que Decodificar no puede detectar porque necesitan la impresora
// (escpos) o la llave pública del cliente (seguridad). Sus pruebas están en esos paquetes.
func dependeDeOtroPaso(e entradaIndice) bool {
	return e.Regla == "ancho_minimo" || e.Regla == "firma_ed25519"
}

func TestEjemplosInvalidos(t *testing.T) {
	var indice []entradaIndice
	if err := json.Unmarshal(leer(t, filepath.Join(dirEjemplos, "invalidos", "indice.json")), &indice); err != nil {
		t.Fatal(err)
	}
	for _, e := range indice {
		t.Run(e.Archivo, func(t *testing.T) {
			_, err := Decodificar(leer(t, filepath.Join(dirEjemplos, "invalidos", e.Archivo)))
			if dependeDeOtroPaso(e) {
				if err != nil {
					t.Fatalf("debía pasar Decodificar (lo detecta otro paso), obtuve %v", err)
				}
				return
			}
			var ce *Error
			if !errors.As(err, &ce) {
				t.Fatalf("esperaba *Error %s, obtuve %v", e.Codigo, err)
			}
			if ce.Codigo != e.Codigo {
				t.Fatalf("código %s, quiero %s\n%v", ce.Codigo, e.Codigo, ce)
			}
			if ce.Mensaje == "" || ce.Sugerencia == "" {
				t.Errorf("falta mensaje o sugerencia: %+v", ce)
			}
			if e.Detecta == "esquema" || e.Ruta != "" {
				for _, d := range ce.Detalles {
					if d.Ruta == e.Ruta && d.Regla == e.Regla {
						return
					}
				}
				t.Fatalf("ningún detalle con ruta %q y regla %q:\n%v", e.Ruta, e.Regla, ce)
			}
		})
	}
}

func TestValoresPorDefectoYTipos(t *testing.T) {
	doc, err := Decodificar([]byte(`{"version":"1","id":"3f2b8c1e-5a4d-4e7b-9c2a-1d0e6f8a7b90","impresora":"caja",
		"operaciones":[{"tipo":"texto","valor":"Hola"},{"tipo":"fila","columnas":[{"texto":"a"},{"texto":"b","ancho":30}]},
		{"tipo":"separador"},{"tipo":"salto"},{"tipo":"qr","valor":"x"},{"tipo":"codigo_barras","valor":"400638133393","simbologia":"EAN13"},
		{"tipo":"imagen","recurso":"logo"},{"tipo":"cortar"},{"tipo":"abrir_cajon"},{"tipo":"raw","base64":"G0A="}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Copias != 1 || doc.Firmado() {
		t.Errorf("sobre: %+v", doc)
	}
	tx := doc.Operaciones[0].(*Texto)
	if tx.Alineacion != Izquierda || tx.Tamano != [2]int{1, 1} || tx.Fuente != "A" || tx.Negrita {
		t.Errorf("texto: %+v", tx)
	}
	f := doc.Operaciones[1].(*Fila)
	if f.Columnas[0].Ancho != AnchoAuto || f.Columnas[0].Alineacion != Izquierda || f.Columnas[1].Ancho != 30 {
		t.Errorf("fila: %+v", f)
	}
	if s := doc.Operaciones[2].(*Separador); s.Caracter != "-" {
		t.Errorf("separador: %+v", s)
	}
	if s := doc.Operaciones[3].(*Salto); s.Lineas != 1 {
		t.Errorf("salto: %+v", s)
	}
	if q := doc.Operaciones[4].(*QR); q.Tamano != 6 || q.Correccion != "M" || q.Alineacion != Centro {
		t.Errorf("qr: %+v", q)
	}
	if c := doc.Operaciones[5].(*CodigoBarras); c.Texto != "abajo" || c.Alto != 80 || c.Alineacion != Centro {
		t.Errorf("codigo_barras: %+v", c)
	}
	if im := doc.Operaciones[6].(*Imagen); im.AnchoMax != 100 || im.Alineacion != Centro || im.Recurso != "logo" {
		t.Errorf("imagen: %+v", im)
	}
	if c := doc.Operaciones[7].(*Cortar); c.Parcial {
		t.Errorf("cortar: %+v", c)
	}
	if c := doc.Operaciones[8].(*AbrirCajon); c.Pin != 2 {
		t.Errorf("abrir_cajon: %+v", c)
	}
	if r := doc.Operaciones[9].(*Raw); string(r.Datos) != "\x1b@" {
		t.Errorf("raw: %+v", r)
	}
}

func TestDocumentoFirmado(t *testing.T) {
	doc, err := Decodificar(leer(t, filepath.Join(dirEjemplos, "validos", "firmado.json")))
	if err != nil {
		t.Fatal(err)
	}
	if !doc.Firmado() || doc.ClienteID != "cli_7Hq2MZp4XwR9" || doc.Expira.Sub(doc.Emitido) != 90*time.Second {
		t.Fatalf("campos de firma: %+v", doc)
	}
	if len(doc.Crudo) == 0 {
		t.Error("Crudo debe conservar el JSON recibido")
	}
}

func TestImagenEnLineaSeDecodifica(t *testing.T) {
	doc, err := Decodificar(leer(t, filepath.Join(dirEjemplos, "validos", "venta-logo-qr.json")))
	if err != nil {
		t.Fatal(err)
	}
	im := doc.Operaciones[0].(*Imagen)
	if !strings.HasPrefix(string(im.PNG), firmaPNG) {
		t.Fatal("la imagen debe quedar decodificada en PNG")
	}
}

func TestJSONInvalido(t *testing.T) {
	casos := map[string]string{
		"bom":           "\xEF\xBB\xBF{}",
		"utf8":          `{"version":"1","x":"` + "\xff" + `"}`,
		"dos objetos":   `{"version":"1"} {}`,
		"anidamiento":   strings.Repeat("[", 40) + strings.Repeat("]", 40),
		"clave anidada": `{"version":"1","operaciones":[{"tipo":"texto","tipo":"raw"}]}`,
		"vacío":         ``,
		"incompleto":    `{"version":"1",`,
	}
	for nombre, cuerpo := range casos {
		_, err := Decodificar([]byte(cuerpo))
		var ce *Error
		if !errors.As(err, &ce) || ce.Codigo != ErrJSONInvalido {
			t.Errorf("%s: esperaba JSON_INVALIDO, obtuve %v", nombre, err)
		}
	}
}

func TestDigitoControlGS1(t *testing.T) {
	casos := map[string]int{"400638133393": 1, "9638507": 4, "03600029145": 2, "750103131130": 9}
	for digitos, esperado := range casos {
		if d := DigitoControlGS1(digitos); d != esperado {
			t.Errorf("DigitoControlGS1(%s) = %d, quiero %d", digitos, d, esperado)
		}
	}
}

func TestErrorEsJSONEstable(t *testing.T) {
	_, err := Decodificar([]byte(`{"version":"2"}`))
	b, _ := json.Marshal(err)
	if !strings.Contains(string(b), `"codigo":"VERSION_NO_SOPORTADA"`) || !strings.Contains(string(b), `"versiones_soportadas":["1"]`) {
		t.Fatalf("JSON del error: %s", b)
	}
}
