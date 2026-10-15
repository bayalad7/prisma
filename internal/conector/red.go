package conector

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"
)

// Red envía por TCP (puerto 9100, "raw" o JetDirect).
type Red struct {
	Direccion       string // host:puerto; sin puerto se usa 9100
	TiempoConexion  time.Duration
	TiempoEscritura time.Duration
}

// NuevoRed crea un conector TCP con los tiempos por defecto (3 s de conexión, 10 s de escritura).
func NuevoRed(direccion string) *Red {
	if _, _, err := net.SplitHostPort(direccion); err != nil {
		direccion = net.JoinHostPort(direccion, "9100")
	}
	return &Red{Direccion: direccion, TiempoConexion: TiempoConexion, TiempoEscritura: TiempoEscritura}
}

func (r *Red) conectar(ctx context.Context) (net.Conn, error) {
	d := net.Dialer{Timeout: r.TiempoConexion}
	c, err := d.DialContext(ctx, "tcp", r.Direccion)
	if err != nil {
		return nil, r.traducir("conectar", err)
	}
	return c, nil
}

// Enviar abre una conexión, escribe todo y la cierra. Reintenta una vez solo si la conexión
// falló o no se llegó a escribir ningún byte: repetir una escritura parcial imprimiría el
// ticket dos veces.
func (r *Red) Enviar(ctx context.Context, datos []byte) error {
	var ultimo error
	for intento := 0; intento < 2; intento++ {
		if intento > 0 {
			select {
			case <-ctx.Done():
				return r.traducir("conectar", ctx.Err())
			case <-time.After(500 * time.Millisecond):
			}
		}
		escritos, err := r.enviarUnaVez(ctx, datos)
		if err == nil {
			return nil
		}
		ultimo = err
		if escritos > 0 || ctx.Err() != nil {
			break
		}
	}
	return ultimo
}

func (r *Red) enviarUnaVez(ctx context.Context, datos []byte) (int, error) {
	c, err := r.conectar(ctx)
	if err != nil {
		return 0, err
	}
	defer c.Close()
	plazo := time.Now().Add(r.TiempoEscritura)
	if d, ok := ctx.Deadline(); ok && d.Before(plazo) {
		plazo = d
	}
	c.SetWriteDeadline(plazo)
	n, err := c.Write(datos)
	if err != nil {
		return n, r.traducir("enviar los datos", err)
	}
	// Cerrar la escritura de forma ordenada asegura que la impresora reciba todo antes del cierre.
	if tcp, ok := c.(*net.TCPConn); ok {
		tcp.CloseWrite()
	}
	return n, nil
}

// Probar abre y cierra una conexión.
func (r *Red) Probar(ctx context.Context) error {
	c, err := r.conectar(ctx)
	if err != nil {
		return err
	}
	return c.Close()
}

func (r *Red) traducir(que string, err error) error {
	var ne net.Error
	switch {
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()):
		return errorTiempo(que, err)
	case errors.Is(err, context.Canceled):
		return err
	case conexionRechazada(err):
		return nuevoError(ErrImpresoraInalcanzable, fmt.Sprintf("La impresora en %s rechazó la conexión.", r.Direccion),
			"Revisa que la dirección y el puerto (normalmente 9100) sean correctos, o espera a que termine otro trabajo.", err)
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return nuevoError(ErrImpresoraNoEncontrada, fmt.Sprintf("No se encontró el equipo %q.", r.Direccion),
			"Usa la dirección IP de la impresora y asígnale una IP fija en el router.", err)
	}
	return nuevoError(ErrImpresoraInalcanzable, fmt.Sprintf("Falló la comunicación con la impresora en %s al %s.", r.Direccion, que),
		"Revisa que esté encendida y conectada a la misma red que este equipo.", err)
}
