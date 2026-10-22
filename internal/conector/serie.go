package conector

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"go.bug.st/serial"
)

// Serie envía por un puerto serie: impresoras Bluetooth (perfil SPP) emparejadas con el sistema
// (COMx en Windows, /dev/rfcommN en Linux, /dev/cu.* en macOS) o impresoras seriales clásicas.
type Serie struct {
	Puerto  string
	Baudios int
	// Confirmar consulta el estado después del ticket (DLE EOT 1) y espera la respuesta: en
	// Bluetooth, Drain termina cuando los datos salieron del equipo, no cuando llegaron.
	Confirmar bool
}

func NuevoSerie(puerto string, baudios int) *Serie {
	return &Serie{Puerto: puerto, Baudios: baudios}
}

func (s *Serie) abrir() (serial.Port, error) {
	p, err := serial.Open(s.Puerto, &serial.Mode{BaudRate: s.Baudios})
	if err != nil {
		return nil, s.traducir(err)
	}
	return p, nil
}

func (s *Serie) Enviar(ctx context.Context, datos []byte) error {
	p, err := s.abrir()
	if err != nil {
		return err
	}
	defer p.Close()
	if s.Confirmar {
		// Descarta lo que la impresora haya enviado antes, para no tomarlo como respuesta.
		p.ResetInputBuffer()
	}
	err = conPlazo(ctx, TiempoEscritura, func() error {
		for len(datos) > 0 {
			n, err := p.Write(datos)
			if err != nil {
				return err
			}
			datos = datos[n:]
		}
		return p.Drain()
	}, func() { p.Close() })
	switch {
	case err == nil:
		if s.Confirmar {
			return s.confirmar(ctx, p)
		}
		return nil
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled):
		return errorTiempo("recibir los datos", err)
	}
	return s.traducir(err)
}

// confirmar envía la consulta de estado y espera un byte de respuesta.
func (s *Serie) confirmar(ctx context.Context, p serial.Port) error {
	var b [1]byte
	err := conPlazo(ctx, TiempoConfirmacion+time.Second, func() error {
		if _, err := p.Write(consultaEstado); err != nil {
			return err
		}
		if err := p.SetReadTimeout(TiempoConfirmacion); err != nil {
			return err
		}
		n, err := p.Read(b[:])
		if err == nil && n == 0 {
			err = os.ErrDeadlineExceeded
		}
		return err
	}, func() { p.Close() })
	if err != nil {
		return errorSinConfirmacion(s.Puerto, err)
	}
	return interpretarEstado(b[0])
}

// Probar abre y cierra el puerto. En Bluetooth esto establece la conexión con la impresora,
// así que también detecta si está apagada o fuera de alcance.
func (s *Serie) Probar(context.Context) error {
	p, err := s.abrir()
	if err != nil {
		return err
	}
	return p.Close()
}

func (s *Serie) traducir(err error) error {
	var pe *serial.PortError
	if errors.As(err, &pe) {
		switch pe.Code() {
		case serial.PortBusy:
			return nuevoError(ErrPuertoOcupado, fmt.Sprintf("El puerto %s está en uso por otro programa.", s.Puerto),
				"Cierra el programa que lo usa (otra aplicación de impresión o un monitor serie).", err)
		case serial.PortNotFound:
			return nuevoError(ErrImpresoraNoEncontrada, fmt.Sprintf("No existe el puerto %s.", s.Puerto),
				"Empareja la impresora y usa el puerto que asignó el sistema (ver docs/conectores.md).", err)
		case serial.PermissionDenied:
			return nuevoError(ErrPermisoDenegado, fmt.Sprintf("No hay permiso para abrir %s.", s.Puerto),
				"En Linux agrega el usuario del servicio al grupo dialout.", err)
		}
	}
	// En Linux y macOS la biblioteca devuelve el error del sistema tal cual al abrir el dispositivo.
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nuevoError(ErrImpresoraNoEncontrada, fmt.Sprintf("No existe el puerto %s.", s.Puerto),
			"Empareja la impresora y usa el puerto que asignó el sistema (ver docs/conectores.md).", err)
	case errors.Is(err, fs.ErrPermission):
		return nuevoError(ErrPermisoDenegado, fmt.Sprintf("No hay permiso para abrir %s.", s.Puerto),
			"En Linux agrega el usuario del servicio al grupo dialout.", err)
	}
	return nuevoError(ErrImpresoraInalcanzable, fmt.Sprintf("No se pudo comunicar con la impresora en %s.", s.Puerto),
		"Revisa que esté encendida, emparejada y dentro del alcance del Bluetooth.", err)
}
