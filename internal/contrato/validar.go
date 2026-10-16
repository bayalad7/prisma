package contrato

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"

	"github.com/bayalad7/prisma/schema"
)

// Límites del contrato v1 (§5).
const (
	MaxBytesDocumento = 256 * 1024
	MaxOperaciones    = 500
	maxProfundidad    = 32
)

// VersionesSoportadas son las versiones del documento que acepta este puente.
var VersionesSoportadas = []string{"1"}

var esquema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schema.DocumentoV1))
	if err != nil {
		return nil, err
	}
	const url = "https://prisma.local/schema/documento-v1.schema.json"
	c := jsonschema.NewCompiler()
	if err := c.AddResource(url, doc); err != nil {
		return nil, err
	}
	return c.Compile(url)
})

// Decodificar valida un documento recibido y lo convierte a tipos Go. Aplica, en orden:
// tamaño, JSON (I-JSON), versión, JSON Schema y reglas del documento que no dependen de la
// impresora. Los errores son siempre *Error.
func Decodificar(datos []byte) (*Documento, error) {
	if err := revisarInicio(datos); err != nil {
		return nil, err
	}
	if err := validarEsquema(datos); err != nil {
		return nil, err
	}
	doc, err := convertir(datos)
	if err != nil {
		return nil, NuevoError(ErrDocumentoInvalido, "El documento no cumple el contrato v1: "+err.Error()+".",
			"Revisa los tipos de los campos.", "", "tipo")
	}
	if err := validarReglas(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// revisarInicio aplica los tres primeros pasos: tamaño, JSON estricto y versión.
func revisarInicio(datos []byte) error {
	if len(datos) > MaxBytesDocumento {
		return NuevoError(ErrDocumentoDemasiadoGrande,
			fmt.Sprintf("El documento mide %d bytes; el máximo es %d (256 KB).", len(datos), MaxBytesDocumento),
			"Reduce las imágenes en línea o guárdalas en el puente y usa \"recurso\".", "", "tamano_maximo")
	}
	if err := revisarJSON(datos); err != nil {
		return NuevoError(ErrJSONInvalido, "El cuerpo no es JSON válido: "+err.Error()+".",
			"Envía un único objeto JSON en UTF-8, sin BOM y sin claves repetidas.", "", "json")
	}
	return revisarVersion(datos)
}

// Sobre son los datos del documento que necesita la autorización (firma, idempotencia y
// permisos) antes de la validación completa, que es más costosa.
type Sobre struct {
	ID, Impresora, ClienteID, Firma string
	Emitido, Expira                 string // tal como vienen (RFC 3339); vacías si faltan
	Tipos                           []string
	Crudo                           []byte
}

// LeerSobre aplica los primeros pasos de la validación (tamaño, JSON estricto y versión) y extrae
// el sobre sin validar el esquema. Los campos con un tipo inesperado quedan vacíos: Decodificar
// los rechazará después con un error preciso.
func LeerSobre(datos []byte) (*Sobre, error) {
	if err := revisarInicio(datos); err != nil {
		return nil, err
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(datos, &obj); err != nil {
		return nil, NuevoError(ErrDocumentoInvalido, "El documento debe ser un objeto JSON.",
			"Envía un objeto con version, id, impresora y operaciones.", "", "type")
	}
	cadena := func(clave string) string {
		var s string
		json.Unmarshal(obj[clave], &s)
		return s
	}
	s := &Sobre{ID: cadena("id"), Impresora: cadena("impresora"), ClienteID: cadena("cliente_id"), Firma: cadena("firma"),
		Emitido: cadena("emitido"), Expira: cadena("expira"), Crudo: datos}
	var ops []json.RawMessage
	json.Unmarshal(obj["operaciones"], &ops)
	for _, op := range ops {
		var t struct {
			Tipo string `json:"tipo"`
		}
		json.Unmarshal(op, &t)
		s.Tipos = append(s.Tipos, t.Tipo)
	}
	return s, nil
}

// revisarJSON exige un único valor JSON en UTF-8 válido, sin BOM, sin claves duplicadas
// y con anidamiento razonable.
func revisarJSON(datos []byte) error {
	if bytes.HasPrefix(datos, []byte("\xEF\xBB\xBF")) {
		return errors.New("empieza con una marca BOM")
	}
	if !utf8.Valid(datos) {
		return errors.New("contiene bytes que no son UTF-8 válido")
	}
	dec := json.NewDecoder(bytes.NewReader(datos))
	dec.UseNumber()
	if err := recorrerJSON(dec, 0); err != nil {
		return traducirErrorJSON(err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return errors.New("hay contenido después del objeto JSON")
	}
	return nil
}

func recorrerJSON(dec *json.Decoder, profundidad int) error {
	if profundidad > maxProfundidad {
		return errors.New("el anidamiento es demasiado profundo")
	}
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		claves := map[string]bool{}
		for dec.More() {
			tok, err := dec.Token()
			if err != nil {
				return err
			}
			clave := tok.(string)
			if claves[clave] {
				return fmt.Errorf("la clave %q está repetida", clave)
			}
			claves[clave] = true
			if err := recorrerJSON(dec, profundidad+1); err != nil {
				return err
			}
		}
	case '[':
		for dec.More() {
			if err := recorrerJSON(dec, profundidad+1); err != nil {
				return err
			}
		}
	}
	_, err = dec.Token() // cierre } o ]
	return err
}

func traducirErrorJSON(err error) error {
	var se *json.SyntaxError
	if errors.As(err, &se) {
		return fmt.Errorf("error de sintaxis cerca del byte %d", se.Offset)
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return errors.New("el JSON está incompleto")
	}
	return err
}

// revisarVersion se comprueba antes que el esquema, para que un documento de otra versión
// reciba VERSION_NO_SOPORTADA y no un error genérico.
func revisarVersion(datos []byte) error {
	var obj map[string]json.RawMessage
	if json.Unmarshal(datos, &obj) != nil {
		return nil // no es un objeto: lo reporta el esquema
	}
	sugerencia := "Envía \"version\": \"1\" (como cadena) o actualiza PRISMA en el equipo del negocio."
	crudo, ok := obj["version"]
	if !ok {
		return &Error{Codigo: ErrVersionNoSoportada, Mensaje: "Falta el campo version.", Sugerencia: sugerencia,
			Detalles:            []Detalle{{Ruta: "", Regla: "required", Mensaje: "falta el campo \"version\""}},
			VersionesSoportadas: VersionesSoportadas}
	}
	var v string
	if json.Unmarshal(crudo, &v) != nil {
		return &Error{Codigo: ErrVersionNoSoportada, Mensaje: "El campo version debe ser una cadena, por ejemplo \"1\".",
			Sugerencia: sugerencia, Detalles: []Detalle{{Ruta: "/version", Regla: "const", Mensaje: "debe ser una cadena"}},
			VersionesSoportadas: VersionesSoportadas}
	}
	for _, s := range VersionesSoportadas {
		if v == s {
			return nil
		}
	}
	return &Error{Codigo: ErrVersionNoSoportada, Mensaje: fmt.Sprintf("Este puente no soporta documentos versión %s.", v),
		Sugerencia:          "Actualiza PRISMA en el equipo del negocio o envía el documento en una versión soportada.",
		Detalles:            []Detalle{{Ruta: "/version", Regla: "const", Mensaje: fmt.Sprintf("versión %q no soportada", v)}},
		VersionesSoportadas: VersionesSoportadas}
}

var rutaTipo = regexp.MustCompile(`^/operaciones/\d+/tipo$`)

func validarEsquema(datos []byte) error {
	sch, err := esquema()
	if err != nil {
		return fmt.Errorf("no se pudo compilar el esquema embebido: %w", err)
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(datos))
	if err != nil {
		return NuevoError(ErrJSONInvalido, "El cuerpo no es JSON válido.", "Envía un único objeto JSON.", "", "json")
	}
	err = sch.Validate(inst)
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		return err
	}
	var hs []*jsonschema.ValidationError
	hojas(ve, &hs)
	var detalles []Detalle
	vistos := map[Detalle]bool{}
	codigo := ErrDocumentoInvalido
	for _, h := range hs {
		d := Detalle{Ruta: puntero(h.InstanceLocation), Regla: h.ErrorKind.KeywordPath()[0]}
		d.Mensaje = mensajeEsquema(h, d.Ruta)
		if vistos[d] {
			continue
		}
		vistos[d] = true
		detalles = append(detalles, d)
		switch {
		case d.Regla == "enum" && rutaTipo.MatchString(d.Ruta):
			codigo = ErrOperacionDesconocida
		case d.Regla == "maxItems" && d.Ruta == "/operaciones" && codigo == ErrDocumentoInvalido:
			codigo = ErrDemasiadasOperaciones
		}
	}
	sort.SliceStable(detalles, func(i, j int) bool { return detalles[i].Ruta < detalles[j].Ruta })

	e := &Error{Codigo: codigo, Detalles: detalles}
	switch codigo {
	case ErrOperacionDesconocida:
		e.Mensaje = "El documento contiene un tipo de operación que no existe en la versión 1."
		e.Sugerencia = "Usa texto, separador, fila, par, salto, qr, codigo_barras, imagen, cortar, abrir_cajon o raw."
	case ErrDemasiadasOperaciones:
		e.Mensaje = fmt.Sprintf("El documento tiene más de %d operaciones.", MaxOperaciones)
		e.Sugerencia = "Divide el ticket en varios documentos."
	default:
		e.Mensaje = "El documento no cumple el contrato v1."
		e.Sugerencia = "Revisa los detalles; cada uno indica el campo y la regla que falló."
	}
	return e
}

// hojas recoge los errores concretos. oneOf/anyOf se reportan como un solo error,
// porque sus causas (cada alternativa fallida) confunden más de lo que ayudan.
func hojas(e *jsonschema.ValidationError, out *[]*jsonschema.ValidationError) {
	switch e.ErrorKind.(type) {
	case *kind.OneOf, *kind.AnyOf:
		*out = append(*out, e)
		return
	}
	if len(e.Causes) == 0 {
		*out = append(*out, e)
		return
	}
	for _, c := range e.Causes {
		hojas(c, out)
	}
}

func puntero(ubicacion []string) string {
	var b strings.Builder
	for _, p := range ubicacion {
		b.WriteByte('/')
		b.WriteString(strings.NewReplacer("~", "~0", "/", "~1").Replace(p))
	}
	return b.String()
}

var nombresTipo = map[string]string{
	"string": "una cadena", "integer": "un entero", "number": "un número", "boolean": "un booleano (true o false)",
	"object": "un objeto", "array": "un arreglo", "null": "null",
}

// formatos explica el patrón esperado según el nombre del campo.
var formatos = map[string]string{
	"id":         "debe ser un UUID v4 en minúsculas",
	"impresora":  "debe ser un alias: minúsculas, dígitos, - o _, hasta 32 caracteres (nunca una IP)",
	"cliente_id": "debe empezar con cli_ seguido de 8 a 40 letras o dígitos",
	"emitido":    "debe ser una fecha RFC 3339 con zona horaria, ej. 2026-10-14T18:30:00Z",
	"expira":     "debe ser una fecha RFC 3339 con zona horaria, ej. 2026-10-14T18:31:30Z",
	"firma":      "debe ser base64 estándar de 64 bytes (88 caracteres)",
	"base64":     "debe ser base64 estándar sin prefijo data: (en imagen, un PNG)",
	"recurso":    "debe ser un nombre de recurso: minúsculas, dígitos, - o _",
}

func mensajeEsquema(e *jsonschema.ValidationError, ruta string) string {
	campo := ruta[strings.LastIndex(ruta, "/")+1:]
	switch k := e.ErrorKind.(type) {
	case *kind.Type:
		var quiero []string
		for _, w := range k.Want {
			quiero = append(quiero, nombresTipo[w])
		}
		return fmt.Sprintf("debe ser %s, no %s", strings.Join(quiero, " o "), nombresTipo[k.Got])
	case *kind.Enum:
		var opciones []string
		for _, w := range k.Want {
			opciones = append(opciones, fmt.Sprint(w))
		}
		return "debe ser uno de: " + strings.Join(opciones, ", ")
	case *kind.Const:
		return fmt.Sprintf("debe ser %v", k.Want)
	case *kind.Required:
		return "falta el campo " + listaCampos(k.Missing)
	case *kind.DependentRequired:
		return fmt.Sprintf("con %q también se requiere %s", k.Prop, listaCampos(k.Missing))
	case *kind.AdditionalProperties:
		return "campo no permitido: " + listaCampos(k.Properties)
	case *kind.MinItems:
		return fmt.Sprintf("debe tener al menos %d elemento(s)", k.Want)
	case *kind.MaxItems:
		return fmt.Sprintf("debe tener como máximo %d elemento(s)", k.Want)
	case *kind.MinLength:
		return fmt.Sprintf("debe tener al menos %d carácter(es)", k.Want)
	case *kind.MaxLength:
		return fmt.Sprintf("debe tener como máximo %d caracteres (tiene %d)", k.Want, k.Got)
	case *kind.Minimum:
		return "debe ser mayor o igual que " + k.Want.RatString()
	case *kind.Maximum:
		return "debe ser menor o igual que " + k.Want.RatString()
	case *kind.Pattern:
		if strings.Contains(e.SchemaURL, "/codigo_barras/") {
			return "no es válido para la simbología indicada (revisa los caracteres y la longitud)"
		}
		if f, ok := formatos[campo]; ok {
			return f
		}
		return "contiene caracteres no permitidos (los caracteres de control solo pueden ir en raw)"
	case *kind.OneOf:
		if strings.HasPrefix(ruta, "/operaciones/") {
			return "la imagen debe traer exactamente uno de base64 o recurso"
		}
		return "debe cumplir exactamente una de las alternativas"
	case *kind.AnyOf:
		if campo == "ancho" {
			return "debe ser un entero de 1 a 100 o \"auto\""
		}
		return "no cumple ninguna de las alternativas"
	case *kind.FalseSchema:
		return "no se admiten más elementos"
	}
	return "no cumple la regla " + strings.Join(e.ErrorKind.KeywordPath(), "/")
}

func listaCampos(campos []string) string {
	q := make([]string, len(campos))
	for i, c := range campos {
		q[i] = fmt.Sprintf("%q", c)
	}
	return strings.Join(q, ", ")
}
