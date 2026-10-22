package almacen

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Estados de un trabajo.
const (
	TrabajoEncolado    = "encolado"
	TrabajoImprimiendo = "imprimiendo"
	TrabajoCompletado  = "completado"
	TrabajoFallido     = "fallido"
)

// CodigoPuenteReiniciado marca los trabajos que quedaron a medias cuando el puente se detuvo.
const CodigoPuenteReiniciado = "PUENTE_REINICIADO"

// Trabajo es un documento aceptado para una impresora. No guarda el contenido del ticket.
type Trabajo struct {
	ID             string
	DocumentoID    string
	ClienteID      string
	ImpresoraAlias string
	Virtual        bool // la impresora era virtual (el ticket no salió en papel)
	Referencia     string
	Estado         string
	CodigoError    string
	Advertencias   []string
	Duracion       time.Duration // 0 mientras no termina
	Creado         time.Time
	Finalizado     time.Time // cero mientras no termina
}

type RepoTrabajos struct{ a *Almacen }

const columnasTrabajo = `id, documento_id, cliente_id, impresora_alias, virtual, referencia, estado, codigo_error,
	advertencias, duracion_ms, creado, finalizado`

// Crear registra un trabajo nuevo en estado encolado.
func (r *RepoTrabajos) Crear(ctx context.Context, t *Trabajo) error {
	adv, err := codificarLista(t.Advertencias)
	if err != nil {
		return err
	}
	ahora := r.a.ahora()
	t.Estado = TrabajoEncolado
	_, err = r.a.bd().ExecContext(ctx, `INSERT INTO trabajos (id, documento_id, cliente_id, impresora_alias, virtual,
		referencia, estado, advertencias, creado) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.DocumentoID, t.ClienteID, t.ImpresoraAlias, bit(t.Virtual), textoNulo(t.Referencia), t.Estado, adv, ms(ahora))
	if err != nil {
		return traducirError(err, "trabajo "+t.ID)
	}
	t.Creado = desdeMs(ms(ahora))
	return nil
}

// MarcarImprimiendo pasa un trabajo encolado a imprimiendo.
func (r *RepoTrabajos) MarcarImprimiendo(ctx context.Context, id string) error {
	res, err := r.a.bd().ExecContext(ctx, `UPDATE trabajos SET estado = 'imprimiendo' WHERE id = ? AND estado = 'encolado'`, id)
	return exigirUnaFila(res, err, "trabajo encolado "+id)
}

// Finalizar cierra el trabajo como completado o fallido, con su código de error y advertencias.
func (r *RepoTrabajos) Finalizar(ctx context.Context, id, estado, codigoError string, advertencias []string, duracion time.Duration) error {
	if estado != TrabajoCompletado && estado != TrabajoFallido {
		return fmt.Errorf("trabajo %s: %w (estado final %q)", id, ErrInvalido, estado)
	}
	adv, err := codificarLista(advertencias)
	if err != nil {
		return err
	}
	res, err := r.a.bd().ExecContext(ctx, `UPDATE trabajos SET estado = ?, codigo_error = ?, advertencias = ?,
		duracion_ms = ?, finalizado = ? WHERE id = ? AND estado IN ('encolado', 'imprimiendo')`,
		estado, textoNulo(codigoError), adv, duracion.Milliseconds(), ms(r.a.ahora()), id)
	return exigirUnaFila(res, err, "trabajo pendiente "+id)
}

// MarcarInterrumpidos cierra como fallidos los trabajos que quedaron pendientes al detenerse el puente.
func (r *RepoTrabajos) MarcarInterrumpidos(ctx context.Context) (int64, error) {
	var n int64
	err := r.a.enTransaccion(ctx, func(tx *sql.Tx) error {
		// Como cualquier trabajo fallido, libera el id del documento: si no, el reintento del
		// cliente se respondería como repetido («ok», duplicado) y el ticket nunca se imprimiría.
		if _, err := tx.ExecContext(ctx, `DELETE FROM ids_recientes WHERE trabajo_id IN
			(SELECT id FROM trabajos WHERE estado IN ('encolado', 'imprimiendo'))`); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `UPDATE trabajos SET estado = 'fallido', codigo_error = ?, finalizado = ?
			WHERE estado IN ('encolado', 'imprimiendo')`, CodigoPuenteReiniciado, ms(r.a.ahora()))
		if err != nil {
			return err
		}
		n, err = res.RowsAffected()
		return err
	})
	return n, err
}

func (r *RepoTrabajos) Obtener(ctx context.Context, id string) (*Trabajo, error) {
	t, err := escanearTrabajo(r.a.bd().QueryRowContext(ctx, `SELECT `+columnasTrabajo+` FROM trabajos WHERE id = ?`, id))
	return t, traducirError(err, "trabajo "+id)
}

// FiltroTrabajos limita el listado. Los campos vacíos no filtran.
type FiltroTrabajos struct {
	ClienteID      string
	ImpresoraAlias string
	Estado         string
	Desde, Hasta   time.Time // creados en [Desde, Hasta); cero = sin límite
	Limite         int       // 0 = 100
}

// Listar devuelve los trabajos más recientes primero.
func (r *RepoTrabajos) Listar(ctx context.Context, f FiltroTrabajos) ([]Trabajo, error) {
	if f.Limite <= 0 {
		f.Limite = 100
	}
	hasta := int64(1 << 62)
	if !f.Hasta.IsZero() {
		hasta = ms(f.Hasta)
	}
	filas, err := r.a.bd().QueryContext(ctx, `SELECT `+columnasTrabajo+` FROM trabajos
		WHERE (?1 = '' OR cliente_id = ?1) AND (?2 = '' OR impresora_alias = ?2) AND (?3 = '' OR estado = ?3)
		AND creado >= ?5 AND creado < ?6
		ORDER BY creado DESC, id LIMIT ?4`, f.ClienteID, f.ImpresoraAlias, f.Estado, f.Limite, ms(f.Desde), hasta)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	var lista []Trabajo
	for filas.Next() {
		t, err := escanearTrabajo(filas)
		if err != nil {
			return nil, err
		}
		lista = append(lista, *t)
	}
	return lista, filas.Err()
}

// Contar devuelve cuántos trabajos hay en cada estado entre los creados desde la fecha indicada.
func (r *RepoTrabajos) Contar(ctx context.Context, desde time.Time) (map[string]int, error) {
	filas, err := r.a.bd().QueryContext(ctx, `SELECT estado, count(*) FROM trabajos WHERE creado >= ? GROUP BY estado`, ms(desde))
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	conteo := map[string]int{}
	for filas.Next() {
		var estado string
		var n int
		if err := filas.Scan(&estado, &n); err != nil {
			return nil, err
		}
		conteo[estado] = n
	}
	return conteo, filas.Err()
}

// Purgar borra los trabajos terminados creados antes de la fecha indicada.
func (r *RepoTrabajos) Purgar(ctx context.Context, antesDe time.Time) (int64, error) {
	res, err := r.a.bd().ExecContext(ctx, `DELETE FROM trabajos WHERE creado < ? AND estado IN ('completado', 'fallido')`, ms(antesDe))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func escanearTrabajo(f filaEscaneable) (*Trabajo, error) {
	var t Trabajo
	var referencia, codigo sql.NullString
	var adv string
	var duracion, finalizado sql.NullInt64
	var creado int64
	var virtual int
	err := f.Scan(&t.ID, &t.DocumentoID, &t.ClienteID, &t.ImpresoraAlias, &virtual, &referencia, &t.Estado, &codigo, &adv,
		&duracion, &creado, &finalizado)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(adv), &t.Advertencias); err != nil {
		return nil, fmt.Errorf("advertencias del trabajo %s: %w", t.ID, err)
	}
	t.Virtual, t.Referencia, t.CodigoError = virtual == 1, referencia.String, codigo.String
	t.Duracion = time.Duration(duracion.Int64) * time.Millisecond
	t.Creado, t.Finalizado = desdeMs(creado), desdeMsNulo(finalizado)
	return &t, nil
}
