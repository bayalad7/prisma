// Package conector envía bytes ESC/POS a una impresora por red, USB, Bluetooth (puerto serie),
// a un archivo o a una impresora virtual dentro del puente. Cada tipo de conexión implementa
// Conector; las variantes por sistema operativo están en archivos con build tags.
package conector

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bayalad7/prisma/internal/almacen"
)

// Conector entrega datos a una impresora concreta.
type Conector interface {
	// Enviar entrega los bytes completos de un trabajo.
	Enviar(ctx context.Context, datos []byte) error
	// Probar comprueba que la impresora está alcanzable sin imprimir nada.
	Probar(ctx context.Context) error
}

// Códigos de error de los conectores y la cola.
const (
	ErrImpresoraNoEncontrada = "IMPRESORA_NO_ENCONTRADA"
	ErrImpresoraInalcanzable = "IMPRESORA_INALCANZABLE"
	ErrPuertoOcupado         = "PUERTO_OCUPADO"
	ErrTiempoAgotado         = "TIEMPO_AGOTADO"
	ErrPermisoDenegado       = "PERMISO_DENEGADO"
	ErrColaLlena             = "COLA_LLENA"
)

// AvisoFueraDeLinea: la impresora recibió el ticket, pero informó que está fuera de línea (tapa
// abierta, sin papel o en error). Lo imprimirá al resolverse, así que el trabajo no falla: si
// fallara, el reintento de la aplicación lo imprimiría dos veces.
const AvisoFueraDeLinea = "IMPRESORA_FUERA_DE_LINEA"

// Aviso lo devuelve Enviar cuando el trabajo llegó a la impresora, pero con un problema que la
// persona debe atender. La cola lo trata como éxito y lo guarda como advertencia del trabajo.
type Aviso struct {
	Codigo  string
	Mensaje string
}

func (a *Aviso) Error() string { return a.Codigo + ": " + a.Mensaje }

// Error es un fallo de impresión con código estable, mensaje y sugerencia en español.
type Error struct {
	Codigo     string `json:"codigo"`
	Mensaje    string `json:"mensaje"`
	Sugerencia string `json:"sugerencia"`
	Causa      error  `json:"-"`
}

func (e *Error) Error() string {
	if e.Causa != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Codigo, e.Mensaje, e.Causa)
	}
	return e.Codigo + ": " + e.Mensaje
}

func (e *Error) Unwrap() error { return e.Causa }

func nuevoError(codigo, mensaje, sugerencia string, causa error) *Error {
	return &Error{Codigo: codigo, Mensaje: mensaje, Sugerencia: sugerencia, Causa: causa}
}

// Codigo devuelve el código de un error o aviso de conector, o "" si no lo es.
func Codigo(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Codigo
	}
	var a *Aviso
	if errors.As(err, &a) {
		return a.Codigo
	}
	return ""
}

// Tiempos por defecto.
const (
	TiempoConexion    = 3 * time.Second
	TiempoEscritura   = 10 * time.Second
	BaudiosPorDefecto = 9600
	// TiempoConfirmacion es cuánto se espera la respuesta a la consulta de estado después de
	// enviar el ticket. La impresora responde cuando le llega la consulta, así que cubre lo que
	// todavía no había leído de su búfer.
	TiempoConfirmacion = 10 * time.Second
)

// consultaEstado es DLE EOT 1 (estado de la impresora). Es un comando de tiempo real: la impresora
// lo responde al recibirlo, sin imprimir nada. Como viaja detrás del ticket, su respuesta confirma
// que el ticket completo llegó a la impresora.
var consultaEstado = []byte{0x10, 0x04, 0x01}

// interpretarEstado traduce el byte de estado de DLE EOT 1: los bits 1 y 4 valen 1 y los bits 0 y
// 7 valen 0; el bit 3 indica fuera de línea. Un byte con otro formato también confirma la
// recepción, pero no se interpreta.
func interpretarEstado(b byte) error {
	if b&0x93 == 0x12 && b&0x08 != 0 {
		return &Aviso{Codigo: AvisoFueraDeLinea,
			Mensaje: "La impresora recibió el ticket, pero está fuera de línea (tapa abierta, sin papel o en error); lo imprimirá al resolverlo."}
	}
	return nil
}

func errorSinConfirmacion(donde string, causa error) *Error {
	return nuevoError(ErrTiempoAgotado, fmt.Sprintf("La impresora en %s no confirmó que recibió el ticket.", donde),
		"Revisa que siga encendida y conectada. Si nunca confirma, puede que no admita la consulta de estado (DLE EOT): desactiva «Confirmar impresión» en su perfil.", causa)
}

// Nuevo crea el conector que corresponde al tipo de conexión de la impresora.
func Nuevo(imp almacen.Impresora) (Conector, error) {
	if imp.Destino == "" {
		return nil, nuevoError(ErrImpresoraNoEncontrada, fmt.Sprintf("La impresora %q no tiene destino configurado.", imp.Alias),
			"Indica la dirección, el nombre o el puerto de la impresora en el panel.", nil)
	}
	switch imp.Tipo {
	case almacen.ConexionRed:
		r := NuevoRed(imp.Destino)
		r.Confirmar = imp.ConfirmarImpresion
		return r, nil
	case almacen.ConexionUSB:
		return nuevoUSB(imp.Destino), nil
	case almacen.ConexionBluetooth:
		baudios := imp.Baudios
		if baudios == 0 {
			baudios = BaudiosPorDefecto
		}
		s := NuevoSerie(imp.Destino, baudios)
		s.Confirmar = imp.ConfirmarImpresion
		return s, nil
	case almacen.ConexionArchivo:
		return NuevoArchivo(imp.Destino, imp), nil
	case almacen.ConexionVirtual:
		return NuevoVirtual(imp, RollosVirtuales), nil
	}
	return nil, fmt.Errorf("tipo de conexión %q desconocido", imp.Tipo)
}

// claveTrabajo guarda en el contexto el id del trabajo (lo usa el conector de archivo).
type claveTrabajo struct{}

// ConTrabajo agrega el id del trabajo al contexto.
func ConTrabajo(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, claveTrabajo{}, id)
}

func trabajoDe(ctx context.Context) string {
	id, _ := ctx.Value(claveTrabajo{}).(string)
	return id
}

// esperaTrasCancelar es cuánto se espera a que fn termine después de cancelarla. Si no termina
// (una llamada del sistema que no se puede interrumpir), se abandona para no bloquear la cola de
// la impresora ni la detención del servicio.
var esperaTrasCancelar = 5 * time.Second

// conPlazo ejecuta fn y la abandona si vence el plazo o se cancela el contexto. cancelar se
// invoca en ese caso para desbloquear fn (p. ej. cerrando el puerto).
func conPlazo(ctx context.Context, plazo time.Duration, fn func() error, cancelar func()) error {
	ctx, fin := context.WithTimeout(ctx, plazo)
	defer fin()
	listo := make(chan error, 1)
	go func() { listo <- fn() }()
	select {
	case err := <-listo:
		return err
	case <-ctx.Done():
		if cancelar != nil {
			cancelar()
		}
		select {
		case <-listo:
		case <-time.After(esperaTrasCancelar):
		}
		return ctx.Err()
	}
}

func errorTiempo(que string, causa error) *Error {
	return nuevoError(ErrTiempoAgotado, fmt.Sprintf("La impresora no respondió a tiempo al %s.", que),
		"Revisa que esté encendida, con papel y sin la tapa abierta.", causa)
}
