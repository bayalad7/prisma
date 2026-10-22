// Package contrato define los tipos del documento v1 (docs/contrato-documento-v1.md) y su validación:
// JSON Schema embebido más las reglas de negocio que no dependen de la impresora.
package contrato

import (
	"encoding/json"
	"fmt"
	"time"
)

// Tipos de operación.
const (
	TipoTexto        = "texto"
	TipoSeparador    = "separador"
	TipoFila         = "fila"
	TipoPar          = "par"
	TipoSalto        = "salto"
	TipoQR           = "qr"
	TipoCodigoBarras = "codigo_barras"
	TipoImagen       = "imagen"
	TipoCortar       = "cortar"
	TipoAbrirCajon   = "abrir_cajon"
	TipoRaw          = "raw"
)

// Alineacion horizontal.
type Alineacion string

const (
	Izquierda Alineacion = "izquierda"
	Centro    Alineacion = "centro"
	Derecha   Alineacion = "derecha"
)

// Documento es un documento v1 ya validado, con los valores por defecto aplicados.
type Documento struct {
	Version     string
	ID          string
	Impresora   string
	Copias      int
	Referencia  string
	Operaciones []Operacion

	// Nivel 3 (vacíos si el documento no viene firmado).
	ClienteID string
	Emitido   time.Time
	Expira    time.Time
	Firma     string

	// Crudo es el JSON tal como se recibió: la firma y la idempotencia se calculan sobre él.
	Crudo []byte
}

// Firmado indica si el documento trae los campos del nivel 3.
func (d *Documento) Firmado() bool { return d.Firma != "" }

// Operacion es cualquiera de los once tipos de operación.
type Operacion interface {
	Tipo() string
}

type Texto struct {
	Valor      string     `json:"valor"`
	Alineacion Alineacion `json:"alineacion"`
	Negrita    bool       `json:"negrita"`
	Subrayado  bool       `json:"subrayado"`
	Invertido  bool       `json:"invertido"`
	Tamano     [2]int     `json:"tamano"` // [ancho, alto]
	Fuente     string     `json:"fuente"` // "A" | "B"
}

type Separador struct {
	Caracter string `json:"caracter"`
}

type Fila struct {
	Columnas []Columna `json:"columnas"`
	Negrita  bool      `json:"negrita"`
}

type Columna struct {
	Texto      string     `json:"texto"`
	Ancho      Ancho      `json:"ancho"`
	Alineacion Alineacion `json:"alineacion"`
}

// Ancho de una columna: un porcentaje de 1 a 100, o 0 para "auto".
type Ancho int

// AnchoAuto reparte el ancho sobrante.
const AnchoAuto Ancho = 0

func (a *Ancho) UnmarshalJSON(b []byte) error {
	var s string
	if json.Unmarshal(b, &s) == nil {
		if s != "auto" {
			return fmt.Errorf("ancho %q no es válido", s)
		}
		*a = AnchoAuto
		return nil
	}
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*a = Ancho(n)
	return nil
}

type Par struct {
	Izquierda string `json:"izquierda"`
	Derecha   string `json:"derecha"`
	Negrita   bool   `json:"negrita"`
}

type Salto struct {
	Lineas int `json:"lineas"`
}

type QR struct {
	Valor      string     `json:"valor"`
	Tamano     int        `json:"tamano"`
	Correccion string     `json:"correccion"` // L | M | Q | H
	Alineacion Alineacion `json:"alineacion"`
}

// Simbologías de código de barras.
const (
	CODE128 = "CODE128"
	CODE39  = "CODE39"
	EAN13   = "EAN13"
	EAN8    = "EAN8"
	UPCA    = "UPCA"
	ITF     = "ITF"
)

type CodigoBarras struct {
	Valor      string     `json:"valor"`
	Simbologia string     `json:"simbologia"`
	Texto      string     `json:"texto"` // arriba | abajo | ninguno
	Alto       int        `json:"alto"`
	Alineacion Alineacion `json:"alineacion"`
}

type Imagen struct {
	Base64     string     `json:"base64"`
	Recurso    string     `json:"recurso"`
	Alineacion Alineacion `json:"alineacion"`
	AnchoMax   int        `json:"ancho_max"`

	// PNG decodificado de Base64 durante la validación (vacío si usa Recurso).
	PNG []byte `json:"-"`
}

type Cortar struct {
	Parcial bool `json:"parcial"`
}

type AbrirCajon struct {
	Pin int `json:"pin"`
}

type Raw struct {
	Base64 string `json:"base64"`

	// Datos decodificados de Base64 durante la validación.
	Datos []byte `json:"-"`
}

func (*Texto) Tipo() string        { return TipoTexto }
func (*Separador) Tipo() string    { return TipoSeparador }
func (*Fila) Tipo() string         { return TipoFila }
func (*Par) Tipo() string          { return TipoPar }
func (*Salto) Tipo() string        { return TipoSalto }
func (*QR) Tipo() string           { return TipoQR }
func (*CodigoBarras) Tipo() string { return TipoCodigoBarras }
func (*Imagen) Tipo() string       { return TipoImagen }
func (*Cortar) Tipo() string       { return TipoCortar }
func (*AbrirCajon) Tipo() string   { return TipoAbrirCajon }
func (*Raw) Tipo() string          { return TipoRaw }

// nuevaOperacion devuelve la operación del tipo indicado con sus valores por defecto,
// de modo que los campos omitidos en el JSON los conserven.
func nuevaOperacion(tipo string) Operacion {
	switch tipo {
	case TipoTexto:
		return &Texto{Alineacion: Izquierda, Tamano: [2]int{1, 1}, Fuente: "A"}
	case TipoSeparador:
		return &Separador{Caracter: "-"}
	case TipoFila:
		return &Fila{}
	case TipoPar:
		return &Par{}
	case TipoSalto:
		return &Salto{Lineas: 1}
	case TipoQR:
		return &QR{Tamano: 6, Correccion: "M", Alineacion: Centro}
	case TipoCodigoBarras:
		return &CodigoBarras{Texto: "abajo", Alto: 80, Alineacion: Centro}
	case TipoImagen:
		return &Imagen{Alineacion: Centro, AnchoMax: 100}
	case TipoCortar:
		return &Cortar{}
	case TipoAbrirCajon:
		return &AbrirCajon{Pin: 2}
	case TipoRaw:
		return &Raw{}
	}
	return nil
}

// UnmarshalJSON aplica los valores por defecto de cada columna.
func (c *Columna) UnmarshalJSON(b []byte) error {
	type sinMetodos Columna
	aux := sinMetodos{Ancho: AnchoAuto, Alineacion: Izquierda}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	*c = Columna(aux)
	return nil
}

// sobre es la forma JSON del documento, antes de convertir las operaciones a sus tipos.
type sobre struct {
	Version     string            `json:"version"`
	ID          string            `json:"id"`
	Impresora   string            `json:"impresora"`
	Copias      int               `json:"copias"`
	Referencia  string            `json:"referencia"`
	Operaciones []json.RawMessage `json:"operaciones"`
	ClienteID   string            `json:"cliente_id"`
	Emitido     string            `json:"emitido"`
	Expira      string            `json:"expira"`
	Firma       string            `json:"firma"`
}

// convertir pasa el JSON (ya validado contra el esquema) a los tipos Go.
func convertir(datos []byte) (*Documento, error) {
	s := sobre{Copias: 1}
	if err := json.Unmarshal(datos, &s); err != nil {
		return nil, err
	}
	doc := &Documento{
		Version: s.Version, ID: s.ID, Impresora: s.Impresora, Copias: s.Copias, Referencia: s.Referencia,
		ClienteID: s.ClienteID, Firma: s.Firma, Crudo: datos,
	}
	var err error
	if s.Emitido != "" {
		if doc.Emitido, err = time.Parse(time.RFC3339Nano, s.Emitido); err != nil {
			return nil, fmt.Errorf("emitido: %w", err)
		}
	}
	if s.Expira != "" {
		if doc.Expira, err = time.Parse(time.RFC3339Nano, s.Expira); err != nil {
			return nil, fmt.Errorf("expira: %w", err)
		}
	}
	for i, crudo := range s.Operaciones {
		var t struct {
			Tipo string `json:"tipo"`
		}
		if err := json.Unmarshal(crudo, &t); err != nil {
			return nil, fmt.Errorf("operaciones/%d: %w", i, err)
		}
		op := nuevaOperacion(t.Tipo)
		if op == nil {
			return nil, fmt.Errorf("operaciones/%d: tipo %q desconocido", i, t.Tipo)
		}
		if err := json.Unmarshal(crudo, op); err != nil {
			return nil, fmt.Errorf("operaciones/%d: %w", i, err)
		}
		doc.Operaciones = append(doc.Operaciones, op)
	}
	return doc, nil
}
