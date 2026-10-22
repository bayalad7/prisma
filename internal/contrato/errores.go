package contrato

import (
	"fmt"
	"strings"
)

// Códigos de error del contrato (docs/contrato-documento-v1.md, §9).
const (
	ErrDocumentoDemasiadoGrande = "DOCUMENTO_DEMASIADO_GRANDE"
	ErrJSONInvalido             = "JSON_INVALIDO"
	ErrVersionNoSoportada       = "VERSION_NO_SOPORTADA"
	ErrOperacionDesconocida     = "OPERACION_DESCONOCIDA"
	ErrDemasiadasOperaciones    = "DEMASIADAS_OPERACIONES"
	ErrDocumentoInvalido        = "DOCUMENTO_INVALIDO"
	ErrCodigoBarrasInvalido     = "CODIGO_BARRAS_INVALIDO"
	ErrImagenInvalida           = "IMAGEN_INVALIDA"
	ErrFirmaRequerida           = "FIRMA_REQUERIDA"
	ErrFirmaInvalida            = "FIRMA_INVALIDA"
	ErrFirmaExpirada            = "FIRMA_EXPIRADA"
	ErrRecursoNoEncontrado      = "RECURSO_NO_ENCONTRADO"
	ErrIDReutilizado            = "ID_REUTILIZADO"
	ErrFilaNoCabe               = "FILA_NO_CABE"
	ErrOperacionNoSoportada     = "OPERACION_NO_SOPORTADA"
)

// Error es un rechazo del documento con código estable, mensaje y sugerencia en español.
type Error struct {
	Codigo              string    `json:"codigo"`
	Mensaje             string    `json:"mensaje"`
	Sugerencia          string    `json:"sugerencia"`
	Detalles            []Detalle `json:"detalles,omitempty"`
	VersionesSoportadas []string  `json:"versiones_soportadas,omitempty"`
}

// Detalle señala el campo (JSON Pointer) y la regla que falló.
type Detalle struct {
	Ruta    string `json:"ruta"`
	Regla   string `json:"regla"`
	Mensaje string `json:"mensaje"`
}

func (e *Error) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %s", e.Codigo, e.Mensaje)
	for _, d := range e.Detalles {
		ruta := d.Ruta
		if ruta == "" {
			ruta = "/"
		}
		fmt.Fprintf(&b, "\n  %s (%s): %s", ruta, d.Regla, d.Mensaje)
	}
	return b.String()
}

// NuevoError crea un error con un único detalle (ruta y regla pueden ir vacías).
func NuevoError(codigo, mensaje, sugerencia, ruta, regla string) *Error {
	e := &Error{Codigo: codigo, Mensaje: mensaje, Sugerencia: sugerencia}
	if ruta != "" || regla != "" {
		e.Detalles = []Detalle{{Ruta: ruta, Regla: regla, Mensaje: mensaje}}
	}
	return e
}

// RutaOperacion devuelve el JSON Pointer de la operación i (y de un campo, si se indica).
func RutaOperacion(i int, campos ...string) string {
	r := fmt.Sprintf("/operaciones/%d", i)
	for _, c := range campos {
		r += "/" + c
	}
	return r
}
