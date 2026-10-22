package contrato

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"time"
)

// Límites que el esquema no puede expresar.
const (
	MaxBytesImagen = 100 * 1024
	MaxLadoImagen  = 4096 // píxeles por lado: evita «bombas» PNG que se expanden en memoria
	MaxBytesRaw    = 64 * 1024
	VigenciaMaxima = 120 * time.Second
	firmaPNG       = "\x89PNG\r\n\x1a\n"
)

// validarReglas comprueba las reglas del documento que no dependen de la impresora.
// Si hay varias fallas, devuelve el código de la primera con todos los detalles de ese código.
func validarReglas(doc *Documento) error {
	var errs []*Error
	agregar := func(e *Error) { errs = append(errs, e) }

	for i, op := range doc.Operaciones {
		switch o := op.(type) {
		case *Fila:
			suma := 0
			for _, c := range o.Columnas {
				suma += int(c.Ancho)
			}
			if suma > 100 {
				agregar(NuevoError(ErrFilaNoCabe,
					fmt.Sprintf("Los porcentajes de las columnas suman %d; el máximo es 100.", suma),
					"Reduce los porcentajes o usa \"auto\" en alguna columna.", RutaOperacion(i, "columnas"), "suma_porcentajes"))
			}
		case *CodigoBarras:
			if msg := errorDigitoControl(o.Simbologia, o.Valor); msg != "" {
				agregar(NuevoError(ErrCodigoBarrasInvalido, msg,
					"Corrige el dígito verificador u omítelo para que lo calcule el puente.", RutaOperacion(i, "valor"), "digito_verificador"))
			}
		case *Imagen:
			if o.Base64 == "" {
				continue
			}
			datos, regla, err := decodificarPNG(o.Base64)
			if err != nil {
				agregar(NuevoError(ErrImagenInvalida, "La imagen no es válida: "+err.Error()+".",
					"Envía un PNG de hasta 100 KB y 4096 px por lado, o guárdalo en el puente y usa \"recurso\".",
					RutaOperacion(i, "base64"), regla))
				continue
			}
			o.PNG = datos
		case *Raw:
			datos, err := base64.StdEncoding.Strict().DecodeString(o.Base64)
			switch {
			case err != nil:
				agregar(NuevoError(ErrDocumentoInvalido, "raw no contiene base64 estándar válido.",
					"Codifica los bytes con base64 estándar, con relleno =.", RutaOperacion(i, "base64"), "base64"))
			case len(datos) > MaxBytesRaw:
				agregar(NuevoError(ErrDocumentoInvalido, fmt.Sprintf("raw decodificado mide %d bytes; el máximo es 64 KB.", len(datos)),
					"Divide los bytes en varias operaciones raw o en varios documentos.", RutaOperacion(i, "base64"), "tamano_maximo"))
			default:
				o.Datos = datos
			}
		}
	}

	if doc.Firmado() {
		vigencia := doc.Expira.Sub(doc.Emitido)
		if vigencia <= 0 || vigencia > VigenciaMaxima {
			agregar(NuevoError(ErrFirmaInvalida,
				fmt.Sprintf("La vigencia de la firma (expira − emitido) es %s; debe ser mayor que 0 y de 120 s como máximo.", vigencia),
				"Firma cada documento justo antes de enviarlo, con expira = emitido + 120 s o menos.", "/expira", "vigencia_maxima"))
		}
	}

	if len(errs) == 0 {
		return nil
	}
	primero := errs[0]
	for _, e := range errs[1:] {
		if e.Codigo == primero.Codigo {
			primero.Detalles = append(primero.Detalles, e.Detalles...)
		}
	}
	return primero
}

func decodificarPNG(b64 string) ([]byte, string, error) {
	datos, err := base64.StdEncoding.Strict().DecodeString(b64)
	if err != nil {
		return nil, "base64", fmt.Errorf("el base64 no es válido")
	}
	if len(datos) > MaxBytesImagen {
		return nil, "tamano_maximo", fmt.Errorf("mide %d bytes decodificada; el máximo es 100 KB", len(datos))
	}
	if !bytes.HasPrefix(datos, []byte(firmaPNG)) {
		return nil, "png_valido", fmt.Errorf("no es un PNG")
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(datos))
	if err != nil {
		return nil, "png_valido", fmt.Errorf("el PNG está dañado o incompleto")
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > MaxLadoImagen || cfg.Height > MaxLadoImagen {
		return nil, "dimensiones", fmt.Errorf("mide %d×%d px; el máximo es %d px por lado", cfg.Width, cfg.Height, MaxLadoImagen)
	}
	// Decodificar completo detecta datos truncados que DecodeConfig no ve.
	if _, _, err := image.Decode(bytes.NewReader(datos)); err != nil {
		return nil, "png_valido", fmt.Errorf("el PNG está dañado o incompleto")
	}
	return datos, "", nil
}

// DigitoControlGS1 calcula el dígito verificador módulo 10 de EAN-13, EAN-8 y UPC-A
// para los dígitos sin verificador.
func DigitoControlGS1(digitos string) int {
	suma := 0
	// Desde la derecha, las posiciones impares pesan 3.
	for i := len(digitos) - 1; i >= 0; i-- {
		d := int(digitos[i] - '0')
		if (len(digitos)-1-i)%2 == 0 {
			d *= 3
		}
		suma += d
	}
	return (10 - suma%10) % 10
}

// longitudes completas (con dígito verificador) de las simbologías GS1.
var largoGS1 = map[string]int{EAN13: 13, EAN8: 8, UPCA: 12}

// errorDigitoControl devuelve el mensaje para el usuario si el dígito verificador no coincide ("" si es correcto).
func errorDigitoControl(simbologia, valor string) string {
	largo, ok := largoGS1[simbologia]
	if !ok || len(valor) != largo {
		return "" // sin verificador (lo calcula el puente) o simbología sin verificador obligatorio
	}
	esperado := DigitoControlGS1(valor[:largo-1])
	if int(valor[largo-1]-'0') != esperado {
		return fmt.Sprintf("El dígito verificador de %s %s es incorrecto: debería ser %d.", simbologia, valor, esperado)
	}
	return ""
}
