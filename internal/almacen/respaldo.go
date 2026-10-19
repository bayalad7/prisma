package almacen

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
)

// cabeceraSQLite son los primeros bytes de todo archivo de base de datos SQLite.
const cabeceraSQLite = "SQLite format 3\x00"

// Respaldar escribe una copia completa y coherente de la base en destino, con el puente en
// marcha (VACUUM INTO). El destino no debe existir.
func (a *Almacen) Respaldar(ctx context.Context, destino string) error {
	_, err := a.bd().ExecContext(ctx, "VACUUM INTO ?", destino)
	return err
}

// ErrRespaldoInvalido: el archivo no es un respaldo que este puente pueda usar.
var ErrRespaldoInvalido = errors.New("el archivo no es un respaldo válido de PRISMA")

// ValidarRespaldo comprueba que el archivo sea una base SQLite íntegra de PRISMA con un esquema
// que esta versión conoce.
func ValidarRespaldo(ctx context.Context, ruta string) error {
	f, err := os.Open(ruta)
	if err != nil {
		return err
	}
	cab := make([]byte, len(cabeceraSQLite))
	_, err = io.ReadFull(f, cab)
	f.Close()
	if err != nil || !bytes.Equal(cab, []byte(cabeceraSQLite)) {
		return fmt.Errorf("%w: no es una base de datos SQLite", ErrRespaldoInvalido)
	}
	db, err := sql.Open("sqlite", "file:"+ruta+"?mode=ro")
	if err != nil {
		return err
	}
	defer db.Close()
	var integridad string
	if err := db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&integridad); err != nil || integridad != "ok" {
		return fmt.Errorf("%w: la base está dañada", ErrRespaldoInvalido)
	}
	version, err := versionEsquema(ctx, db)
	if err != nil {
		return err
	}
	conocidas, err := cargarMigraciones(migracionesFS)
	if err != nil {
		return err
	}
	if version < 1 || version > len(conocidas) {
		return fmt.Errorf("%w: usa el esquema %d y este puente conoce del 1 al %d", ErrRespaldoInvalido, version, len(conocidas))
	}
	var tablas int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_schema WHERE type = 'table'
		AND name IN ('impresoras', 'clientes', 'trabajos', 'ajustes')`).Scan(&tablas); err != nil || tablas != 4 {
		return fmt.Errorf("%w: no tiene las tablas de PRISMA", ErrRespaldoInvalido)
	}
	return nil
}

// Restaurar reemplaza la base en uso por el respaldo indicado y la vuelve a abrir (aplicando las
// migraciones si el respaldo es de una versión anterior). Mientras dura, las demás consultas
// esperan. Si algo falla, la base anterior queda intacta.
func (a *Almacen) Restaurar(ctx context.Context, origen string) error {
	if a.ruta == "" {
		return errors.New("una base en memoria no se puede restaurar")
	}
	if err := ValidarRespaldo(ctx, origen); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	anterior := a.ruta + ".antes-de-restaurar"
	if err := a.db.Close(); err != nil {
		return err
	}
	reabrirAnterior := func(causa error) error {
		os.Remove(a.ruta)
		os.Rename(anterior, a.ruta)
		if db, err := abrirArchivo(ctx, a.ruta); err == nil {
			a.db = db
		}
		return causa
	}
	os.Remove(anterior)
	if err := os.Rename(a.ruta, anterior); err != nil {
		db, _ := abrirArchivo(ctx, a.ruta)
		a.db = db
		return err
	}
	// Los archivos -wal y -shm pertenecen a la base anterior; la copia de VACUUM INTO no los usa.
	os.Remove(a.ruta + "-wal")
	os.Remove(a.ruta + "-shm")
	if err := copiarArchivo(origen, a.ruta); err != nil {
		return reabrirAnterior(err)
	}
	db, err := abrirArchivo(ctx, a.ruta)
	if err != nil {
		return reabrirAnterior(err)
	}
	a.db = db
	return nil
}

func copiarArchivo(origen, destino string) error {
	o, err := os.Open(origen)
	if err != nil {
		return err
	}
	defer o.Close()
	d, err := os.OpenFile(destino, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(d, o); err != nil {
		d.Close()
		return err
	}
	return d.Close()
}
