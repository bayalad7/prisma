// Package conector envía bytes ESC/POS a una impresora por red, USB, Bluetooth (puerto serie)
// o a un archivo. Cada tipo de conexión implementa Conector; las variantes por sistema operativo
// están en archivos con build tags.
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

// Codigo devuelve el código de un error de conector, o "" si no lo es.
func Codigo(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Codigo
	}
	return ""
}

// Tiempos por defecto.
const (
	TiempoConexion    = 3 * time.Second
	TiempoEscritura   = 10 * time.Second
	BaudiosPorDefecto = 9600
)

// Nuevo crea el conector que corresponde al tipo de conexión de la impresora.
func Nuevo(imp almacen.Impresora) (Conector, error) {
	if imp.Destino == "" {
		return nil, nuevoError(ErrImpresoraNoEncontrada, fmt.Sprintf("La impresora %q no tiene destino configurado.", imp.Alias),
			"Indica la dirección, el nombre o el puerto de la impresora en el panel.", nil)
	}
	switch imp.Tipo {
	case almacen.ConexionRed:
		return NuevoRed(imp.Destino), nil
	case almacen.ConexionUSB:
		return nuevoUSB(imp.Destino), nil
	case almacen.ConexionBluetooth:
		baudios := imp.Baudios
		if baudios == 0 {
			baudios = BaudiosPorDefecto
		}
		return NuevoSerie(imp.Destino, baudios), nil
	case almacen.ConexionArchivo:
		return NuevoArchivo(imp.Destino, imp), nil
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
		<-listo
		return ctx.Err()
	}
}

func errorTiempo(que string, causa error) *Error {
	return nuevoError(ErrTiempoAgotado, fmt.Sprintf("La impresora no respondió a tiempo al %s.", que),
		"Revisa que esté encendida, con papel y sin la tapa abierta.", causa)
}
