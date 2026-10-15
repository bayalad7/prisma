package almacen

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bayalad7/prisma/internal/seguridad"
)

// Estados de una solicitud de origen.
const (
	SolicitudPendiente = "pendiente"
	SolicitudRechazada = "rechazada"
)

// MaxSolicitudes limita cuántos orígenes desconocidos se recuerdan, para que nadie pueda llenar la base.
const MaxSolicitudes = 200

// ErrLimiteSolicitudes: ya hay MaxSolicitudes registradas y el origen nuevo no se guardó.
var ErrLimiteSolicitudes = errors.New("se alcanzó el límite de solicitudes de origen pendientes")

// Solicitud es un origen desconocido que intentó imprimir y espera la decisión del dueño.
type Solicitud struct {
	Origen     string
	PrimeraVez time.Time
	UltimaVez  time.Time
	Intentos   int
	Estado     string // pendiente | rechazada
}

type RepoSolicitudes struct{ a *Almacen }

// Registrar anota un intento de un origen desconocido: crea la solicitud o incrementa sus intentos.
// Una solicitud rechazada sigue rechazada.
func (r *RepoSolicitudes) Registrar(ctx context.Context, origen string) (*Solicitud, error) {
	norm, err := seguridad.NormalizarOrigen(origen)
	if err != nil {
		return nil, err
	}
	ahora := ms(r.a.ahora())
	err = r.a.enTransaccion(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE solicitudes_origen SET intentos = intentos + 1, ultima_vez = ?
			WHERE origen = ?`, ahora, norm)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			return nil
		}
		var total int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM solicitudes_origen`).Scan(&total); err != nil {
			return err
		}
		if total >= MaxSolicitudes {
			return ErrLimiteSolicitudes
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO solicitudes_origen (origen, primera_vez, ultima_vez) VALUES (?, ?, ?)`,
			norm, ahora, ahora)
		return err
	})
	if err != nil {
		return nil, traducirError(err, "solicitud "+norm)
	}
	return r.Obtener(ctx, norm)
}

func (r *RepoSolicitudes) Obtener(ctx context.Context, origen string) (*Solicitud, error) {
	norm, err := seguridad.NormalizarOrigen(origen)
	if err != nil {
		return nil, err
	}
	s, err := escanearSolicitud(r.a.db.QueryRowContext(ctx,
		`SELECT origen, primera_vez, ultima_vez, intentos, estado FROM solicitudes_origen WHERE origen = ?`, norm))
	return s, traducirError(err, "solicitud "+norm)
}

// Listar devuelve las solicitudes con el estado indicado ("" = todas), las más recientes primero.
func (r *RepoSolicitudes) Listar(ctx context.Context, estado string) ([]Solicitud, error) {
	filas, err := r.a.db.QueryContext(ctx, `SELECT origen, primera_vez, ultima_vez, intentos, estado
		FROM solicitudes_origen WHERE ? = '' OR estado = ? ORDER BY ultima_vez DESC`, estado, estado)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	var lista []Solicitud
	for filas.Next() {
		s, err := escanearSolicitud(filas)
		if err != nil {
			return nil, err
		}
		lista = append(lista, *s)
	}
	return lista, filas.Err()
}

// Aprobar asigna el origen al cliente y elimina la solicitud, en una sola transacción.
func (r *RepoSolicitudes) Aprobar(ctx context.Context, origen, clienteID string) error {
	norm, err := seguridad.NormalizarOrigen(origen)
	if err != nil {
		return err
	}
	return r.a.enTransaccion(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `DELETE FROM solicitudes_origen WHERE origen = ?`, norm)
		if err := exigirUnaFila(res, err, "solicitud "+norm); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO clientes_origenes (cliente_id, origen) VALUES (?, ?)`, clienteID, norm)
		return traducirError(err, "origen "+norm)
	})
}

// Rechazar marca la solicitud como rechazada; sus intentos se siguen contando sin molestar al dueño.
func (r *RepoSolicitudes) Rechazar(ctx context.Context, origen string) error {
	return r.cambiar(ctx, origen, `UPDATE solicitudes_origen SET estado = 'rechazada' WHERE origen = ?`)
}

// Eliminar olvida la solicitud; si el origen vuelve a intentar, aparecerá como pendiente.
func (r *RepoSolicitudes) Eliminar(ctx context.Context, origen string) error {
	return r.cambiar(ctx, origen, `DELETE FROM solicitudes_origen WHERE origen = ?`)
}

func (r *RepoSolicitudes) cambiar(ctx context.Context, origen, consulta string) error {
	norm, err := seguridad.NormalizarOrigen(origen)
	if err != nil {
		return err
	}
	res, err := r.a.db.ExecContext(ctx, consulta, norm)
	return exigirUnaFila(res, err, "solicitud "+norm)
}

func escanearSolicitud(f filaEscaneable) (*Solicitud, error) {
	var s Solicitud
	var primera, ultima int64
	if err := f.Scan(&s.Origen, &primera, &ultima, &s.Intentos, &s.Estado); err != nil {
		return nil, err
	}
	s.PrimeraVez, s.UltimaVez = desdeMs(primera), desdeMs(ultima)
	return &s, nil
}
