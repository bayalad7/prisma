package almacen

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"time"
)

// VentanaIdempotencia es cuánto tiempo se recuerda un id de documento (contrato v1, §8).
const VentanaIdempotencia = 10 * time.Minute

// IDReciente registra un documento aceptado para detectar reintentos y repeticiones.
type IDReciente struct {
	ClienteID     string
	DocumentoID   string
	HashContenido []byte // SHA-256 de la forma canónica JCS del documento sin "firma"
	TrabajoID     string
	Expira        time.Time
}

// Resultado de Reservar.
const (
	ReservaNueva         = iota // el id no se había visto: se registró
	ReservaDuplicada            // mismo id y mismo contenido: devolver el trabajo original sin imprimir
	ReservaIDReutilizado        // mismo id con otro contenido: ID_REUTILIZADO
)

type RepoIDsRecientes struct{ a *Almacen }

// Reservar registra el id del documento de forma atómica. Si el id ya está vigente para ese cliente,
// no lo modifica y devuelve el registro existente junto con ReservaDuplicada o ReservaIDReutilizado.
// Expira se calcula como ahora + VentanaIdempotencia si viene en cero.
func (r *RepoIDsRecientes) Reservar(ctx context.Context, nuevo IDReciente) (int, *IDReciente, error) {
	ahora := r.a.ahora()
	if nuevo.Expira.IsZero() {
		nuevo.Expira = ahora.Add(VentanaIdempotencia)
	}
	resultado := ReservaNueva
	var existente *IDReciente
	err := r.a.enTransaccion(ctx, func(tx *sql.Tx) error {
		// Un registro vencido no cuenta: se reemplaza.
		if _, err := tx.ExecContext(ctx, `DELETE FROM ids_recientes WHERE cliente_id = ? AND documento_id = ? AND expira <= ?`,
			nuevo.ClienteID, nuevo.DocumentoID, ms(ahora)); err != nil {
			return err
		}
		var e IDReciente
		var expira int64
		err := tx.QueryRowContext(ctx, `SELECT cliente_id, documento_id, hash_contenido, trabajo_id, expira
			FROM ids_recientes WHERE cliente_id = ? AND documento_id = ?`, nuevo.ClienteID, nuevo.DocumentoID).
			Scan(&e.ClienteID, &e.DocumentoID, &e.HashContenido, &e.TrabajoID, &expira)
		switch {
		case err == nil:
			e.Expira = desdeMs(expira)
			existente = &e
			resultado = ReservaIDReutilizado
			if bytes.Equal(e.HashContenido, nuevo.HashContenido) {
				resultado = ReservaDuplicada
			}
			return nil
		case !errors.Is(err, sql.ErrNoRows):
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO ids_recientes (cliente_id, documento_id, hash_contenido, trabajo_id, expira)
			VALUES (?, ?, ?, ?, ?)`, nuevo.ClienteID, nuevo.DocumentoID, nuevo.HashContenido, nuevo.TrabajoID, ms(nuevo.Expira))
		return err
	})
	if err != nil {
		return 0, nil, traducirError(err, "id de documento "+nuevo.DocumentoID)
	}
	return resultado, existente, nil
}

// Purgar borra los registros vencidos.
func (r *RepoIDsRecientes) Purgar(ctx context.Context) (int64, error) {
	res, err := r.a.db.ExecContext(ctx, `DELETE FROM ids_recientes WHERE expira <= ?`, ms(r.a.ahora()))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
