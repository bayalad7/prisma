package almacen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
)

// Claves de ajustes conocidas.
const (
	AjusteContrasenaAdmin = "admin.contrasena_hash" // PHC argon2id
	AjusteDebeCambiar     = "admin.debe_cambiar"    // "1" = el panel obliga a cambiar la contraseña
	AjusteRetencionDias   = "trabajos.retencion_dias"
	AjusteCodigoInicial   = "admin.codigo_inicial"   // "<sha256>|<vence ms>": configuración inicial desde otro equipo
	AjusteSoloRedPrivada  = "panel.solo_red_privada" // "0" permite abrir el panel desde cualquier IP
)

// RetencionDiasPorDefecto es cuántos días se conserva el historial de trabajos.
const RetencionDiasPorDefecto = 30

type RepoAjustes struct{ a *Almacen }

// Obtener devuelve el valor y si existe.
func (r *RepoAjustes) Obtener(ctx context.Context, clave string) (string, bool, error) {
	var v string
	err := r.a.bd().QueryRowContext(ctx, `SELECT valor FROM ajustes WHERE clave = ?`, clave).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

// Establecer crea o reemplaza un ajuste.
func (r *RepoAjustes) Establecer(ctx context.Context, clave, valor string) error {
	_, err := r.a.bd().ExecContext(ctx, `INSERT INTO ajustes (clave, valor, actualizado) VALUES (?, ?, ?)
		ON CONFLICT (clave) DO UPDATE SET valor = excluded.valor, actualizado = excluded.actualizado`,
		clave, valor, ms(r.a.ahora()))
	return traducirError(err, "ajuste "+clave)
}

// EstablecerVarios guarda varios ajustes en una sola transacción.
func (r *RepoAjustes) EstablecerVarios(ctx context.Context, valores map[string]string) error {
	return r.a.enTransaccion(ctx, func(tx *sql.Tx) error {
		for clave, valor := range valores {
			if _, err := tx.ExecContext(ctx, `INSERT INTO ajustes (clave, valor, actualizado) VALUES (?, ?, ?)
				ON CONFLICT (clave) DO UPDATE SET valor = excluded.valor, actualizado = excluded.actualizado`,
				clave, valor, ms(r.a.ahora())); err != nil {
				return traducirError(err, "ajuste "+clave)
			}
		}
		return nil
	})
}

func (r *RepoAjustes) Eliminar(ctx context.Context, clave string) error {
	_, err := r.a.bd().ExecContext(ctx, `DELETE FROM ajustes WHERE clave = ?`, clave)
	return err
}

// RetencionDias devuelve los días de historial de trabajos (por defecto 30).
func (r *RepoAjustes) RetencionDias(ctx context.Context) (int, error) {
	v, ok, err := r.Obtener(ctx, AjusteRetencionDias)
	if err != nil || !ok {
		return RetencionDiasPorDefecto, err
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("ajuste %s: %w (valor %q, se espera un entero ≥ 1)", AjusteRetencionDias, ErrInvalido, v)
	}
	return n, nil
}
