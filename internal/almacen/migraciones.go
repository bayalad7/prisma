package almacen

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
)

type migracion struct {
	version int
	nombre  string
	sql     string
}

// cargarMigraciones lee migraciones/NNNN_nombre.sql y exige que estén numeradas 1, 2, 3… sin huecos.
func cargarMigraciones(fsys fs.FS) ([]migracion, error) {
	archivos, err := fs.Glob(fsys, "migraciones/*.sql")
	if err != nil {
		return nil, err
	}
	var ms []migracion
	for _, archivo := range archivos {
		base := path.Base(archivo)
		numero, nombre, ok := strings.Cut(strings.TrimSuffix(base, ".sql"), "_")
		v, err := strconv.Atoi(numero)
		if !ok || err != nil || v < 1 {
			return nil, fmt.Errorf("migración con nombre inválido %q (se espera NNNN_nombre.sql)", base)
		}
		contenido, err := fs.ReadFile(fsys, archivo)
		if err != nil {
			return nil, err
		}
		ms = append(ms, migracion{version: v, nombre: nombre, sql: string(contenido)})
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].version < ms[j].version })
	for i, m := range ms {
		if m.version != i+1 {
			return nil, fmt.Errorf("las migraciones deben numerarse sin huecos: se esperaba %04d y se encontró %04d", i+1, m.version)
		}
	}
	return ms, nil
}

func versionEsquema(ctx context.Context, db *sql.DB) (int, error) {
	var v int
	err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&v)
	return v, err
}

// migrar aplica, cada una en su propia transacción, las migraciones posteriores a PRAGMA user_version.
func migrar(ctx context.Context, db *sql.DB, fsys fs.FS) error {
	ms, err := cargarMigraciones(fsys)
	if err != nil {
		return err
	}
	actual, err := versionEsquema(ctx, db)
	if err != nil {
		return fmt.Errorf("no se pudo leer la versión del esquema: %w", err)
	}
	if actual > len(ms) {
		return fmt.Errorf("la base de datos usa el esquema %d, pero esta versión de PRISMA solo conoce hasta el %d; "+
			"actualiza PRISMA o restaura un respaldo compatible", actual, len(ms))
	}
	for _, m := range ms[actual:] {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, m.sql); err != nil {
			tx.Rollback()
			return fmt.Errorf("falló la migración %04d_%s: %w", m.version, m.nombre, err)
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", m.version)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("falló la migración %04d_%s: %w", m.version, m.nombre, err)
		}
	}
	return nil
}
