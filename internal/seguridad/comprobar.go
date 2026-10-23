package seguridad

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/contrato"
)

// Comprobador de permisos del panel («¿Por qué mi app no puede imprimir?»): recorre los mismos
// pasos que Identificar, VerificarDocumento y VerificarPermisos para una situación descrita, sin
// petición real. No registra solicitudes, no calcula argon2id ni necesita la llave ni la firma:
// supone que la llave enviada es la de la aplicación y que la firma, si la hay, es correcta y
// vigente. TestComprobadorCoincideConLaAPI (internal/servidor) lo compara con /v1/imprimir.

// Situacion describe cómo intenta imprimir una aplicación.
type Situacion struct {
	ClienteID   string   // la aplicación (su llave y su firma son las que se suponen)
	Origen      string   // encabezado Origin; vacío si imprime desde un servidor o la terminal
	ConLlave    bool     // envía su llave de cliente
	Firmado     bool     // su backend firma el documento (nivel 3)
	Impresora   string   // alias
	Operaciones []string // tipos de operación del documento
}

// Pasos de la autorización, en orden.
const (
	PasoOrigen      = "origen"
	PasoLlave       = "llave"
	PasoFirma       = "firma"
	PasoImpresora   = "impresora"
	PasoOperaciones = "operaciones"
)

// Estados de un paso.
const (
	PasoOK       = "ok"
	PasoFalla    = "falla"
	PasoNoAplica = "no_aplica"
)

// Paso es el resultado de un paso; con Estado = falla, trae el rechazo que daría la API.
type Paso struct {
	ID         string `json:"id"`
	Estado     string `json:"estado"`
	Codigo     string `json:"codigo,omitempty"`
	Mensaje    string `json:"mensaje,omitempty"`
	Sugerencia string `json:"sugerencia,omitempty"`
}

// Comprobacion es el recorrido completo. Los pasos se detienen en el primero que falla, como en la
// API. Identificado es la aplicación a nombre de la cual se imprimiría (puede no ser la elegida:
// un origen aprobado identifica a su dueño).
type Comprobacion struct {
	Pasos        []Paso           `json:"pasos"`
	Codigo       string           `json:"codigo,omitempty"`
	Estado       int              `json:"estado_http"`
	Identificado *almacen.Cliente `json:"-"`
	// DuenioOrigen es la aplicación a la que pertenece el origen, si está aprobado.
	DuenioOrigen *almacen.Cliente `json:"-"`
}

func (c *Comprobacion) paso(id, estado string) {
	c.Pasos = append(c.Pasos, Paso{ID: id, Estado: estado})
}

func (c *Comprobacion) falla(id string, r *Rechazo) *Comprobacion {
	c.Pasos = append(c.Pasos, Paso{ID: id, Estado: PasoFalla, Codigo: r.Codigo, Mensaje: r.Mensaje, Sugerencia: r.Sugerencia})
	c.Codigo, c.Estado = r.Codigo, r.Estado
	return c
}

// Comprobar simula la autorización de la situación. Solo devuelve error si falla la base de datos o
// la aplicación no existe (almacen.ErrNoEncontrado).
func (a *Autorizador) Comprobar(ctx context.Context, s Situacion) (*Comprobacion, error) {
	app, err := a.alm.Clientes.Obtener(ctx, s.ClienteID)
	if err != nil {
		return nil, err
	}
	c := &Comprobacion{Estado: http.StatusOK}
	var cli *almacen.Cliente

	// Paso 1: el origen identifica a su dueño (Identificar → ClienteDeOrigen, sin registrar la solicitud).
	if s.Origen == "" {
		c.paso(PasoOrigen, PasoNoAplica)
	} else {
		noAutorizado := rechazo(http.StatusForbidden, ErrOrigenNoAutorizado,
			fmt.Sprintf("El origen %q no está autorizado para imprimir en este puente.", s.Origen),
			"Pide al dueño del puente que apruebe este origen en el panel de administración (quedó como solicitud pendiente).")
		norm, err := almacen.NormalizarOrigen(s.Origen)
		if err != nil {
			noAutorizado.Sugerencia = "Imprime desde una página servida por https:// o http:// con un origen aprobado."
			return c.falla(PasoOrigen, noAutorizado), nil
		}
		duenio, err := a.alm.Clientes.ObtenerPorOrigen(ctx, norm)
		if err != nil {
			if errors.Is(err, almacen.ErrNoEncontrado) {
				return c.falla(PasoOrigen, noAutorizado), nil
			}
			return nil, err
		}
		if !duenio.Activo {
			return c.falla(PasoOrigen, desactivado(duenio)), nil
		}
		cli, c.DuenioOrigen = duenio, duenio
		c.paso(PasoOrigen, PasoOK)
	}

	// Paso 2: la llave (Identificar → verificarLlave); se supone que es la de la aplicación.
	if !s.ConLlave {
		if cli != nil && (cli.Nivel == almacen.NivelLlave || (cli.Nivel == almacen.NivelFirma && cli.ExigirLlave)) {
			return c.falla(PasoLlave, rechazo(http.StatusUnauthorized, ErrLlaveRequerida, "Esta aplicación debe enviar su llave de cliente.",
				"Agrega el encabezado Authorization: Bearer pk_… con la llave que entregó el panel.")), nil
		}
		c.paso(PasoLlave, PasoNoAplica)
	} else {
		switch {
		case app.LlaveHash == "":
			return c.falla(PasoLlave, llaveInvalida("La llave de cliente no es válida.")), nil
		case cli != nil && app.ID != cli.ID:
			return c.falla(PasoLlave, llaveInvalida("La llave no corresponde a la aplicación de este origen.")), nil
		case cli != nil:
		case !app.Activo:
			return c.falla(PasoLlave, desactivado(app)), nil
		case app.Nivel == almacen.NivelOrigen:
			return c.falla(PasoLlave, rechazo(http.StatusForbidden, ErrOrigenRequerido,
				"Esta aplicación es de nivel 1: solo puede imprimir desde un navegador con un origen aprobado.",
				"Imprime desde la página web autorizada, o pide al dueño del puente que suba la aplicación a nivel 2.")), nil
		default:
			cli = app
		}
		c.paso(PasoLlave, PasoOK)
	}

	// Paso 3: el documento (VerificarDocumento). Sin origen ni llave, solo identifica una firma.
	if cli == nil {
		credenciales := rechazo(http.StatusUnauthorized, ErrCredencialesRequeridas,
			"La petición no identifica a ninguna aplicación autorizada.",
			"Desde un navegador, imprime desde un origen aprobado; desde un servidor, envía la llave (Bearer pk_…) o un documento firmado.")
		if !s.Firmado {
			return c.falla(PasoFirma, credenciales), nil
		}
		switch {
		case !app.Activo:
			return c.falla(PasoFirma, desactivado(app)), nil
		case app.Nivel != almacen.NivelFirma:
			return c.falla(PasoFirma, credenciales), nil
		case app.ExigirLlave:
			return c.falla(PasoFirma, rechazo(http.StatusUnauthorized, ErrLlaveRequerida, "Esta aplicación debe enviar también su llave de cliente.",
				"Agrega el encabezado Authorization: Bearer pk_… además de firmar el documento.")), nil
		}
		cli = app
	}
	if cli.Nivel == almacen.NivelFirma {
		switch {
		case !s.Firmado:
			return c.falla(PasoFirma, rechazo(http.StatusUnauthorized, contrato.ErrFirmaRequerida, "Esta aplicación debe enviar los documentos firmados.",
				"Agrega cliente_id, emitido, expira y firma (ver la especificación, §7).")), nil
		case cli.ID != app.ID:
			// El backend de la aplicación firma con su propio cliente_id.
			return c.falla(PasoFirma, firmaInvalida("El cliente_id del documento no corresponde a la aplicación que lo envía.")), nil
		}
		c.paso(PasoFirma, PasoOK)
	} else {
		c.paso(PasoFirma, PasoNoAplica)
	}
	c.Identificado = cli

	// Paso 5: permisos (VerificarPermisos).
	sobre := &contrato.Sobre{Impresora: s.Impresora, Tipos: s.Operaciones}
	if err := a.VerificarPermisos(ctx, cli, sobre); err != nil {
		var r *Rechazo
		if !errors.As(err, &r) {
			return nil, err
		}
		if r.Codigo == ErrImpresoraNoPermitida {
			return c.falla(PasoImpresora, r), nil
		}
		c.paso(PasoImpresora, PasoOK)
		return c.falla(PasoOperaciones, r), nil
	}
	c.paso(PasoImpresora, PasoOK)
	c.paso(PasoOperaciones, PasoOK)
	return c, nil
}
