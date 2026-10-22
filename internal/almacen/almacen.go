// Package almacen guarda el estado del puente en SQLite (modernc.org/sqlite, Go puro):
// impresoras, clientes, orígenes, solicitudes, trabajos, idempotencia, recursos y ajustes.
// Nunca guarda el contenido de los tickets ni llaves en claro.
package almacen

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

//go:embed migraciones/*.sql
var migracionesFS embed.FS

var (
	// ErrNoEncontrado: el registro pedido no existe.
	ErrNoEncontrado = errors.New("no encontrado")
	// ErrDuplicado: ya existe un registro con la misma clave única.
	ErrDuplicado = errors.New("ya existe")
	// ErrInvalido: los datos violan una restricción del esquema.
	ErrInvalido = errors.New("datos inválidos")
)

// Almacen es el punto de acceso a la base de datos.
type Almacen struct {
	mu    sync.RWMutex // protege db: Restaurar la reemplaza con el puente en marcha
	db    *sql.DB
	ruta  string // vacía en las bases en memoria
	ahora func() time.Time

	Impresoras   *RepoImpresoras
	Clientes     *RepoClientes
	Solicitudes  *RepoSolicitudes
	Trabajos     *RepoTrabajos
	IDsRecientes *RepoIDsRecientes
	Recursos     *RepoRecursos
	Ajustes      *RepoAjustes
}

const pragmas = "_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_txlock=immediate"

// Abrir abre (o crea) la base de datos en la ruta indicada y aplica las migraciones pendientes.
// Crea el directorio si no existe, con permisos restringidos al usuario del servicio.
func Abrir(ctx context.Context, ruta string) (*Almacen, error) {
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return nil, fmt.Errorf("no se pudo crear el directorio de datos: %w", err)
	}
	db, err := abrirArchivo(ctx, ruta)
	if err != nil {
		return nil, err
	}
	a := nuevo(db)
	a.ruta = ruta
	if runtime.GOOS != "windows" {
		// Contiene hashes de llaves y de la contraseña del panel.
		if err := os.Chmod(ruta, 0o600); err != nil {
			a.Cerrar()
			return nil, fmt.Errorf("no se pudieron restringir los permisos de %q: %w", ruta, err)
		}
	}
	return a, nil
}

// AbrirMemoria abre una base de datos en memoria con todas las migraciones aplicadas (para pruebas).
func AbrirMemoria(ctx context.Context) (*Almacen, error) {
	db, err := sql.Open("sqlite", ":memory:?"+pragmas)
	if err != nil {
		return nil, err
	}
	if err := preparar(ctx, db); err != nil {
		return nil, err
	}
	return nuevo(db), nil
}

// abrirArchivo abre la base de un archivo y le aplica las migraciones.
func abrirArchivo(ctx context.Context, ruta string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", ruta+"?"+pragmas+"&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)")
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir la base de datos %q: %w", ruta, err)
	}
	if err := preparar(ctx, db); err != nil {
		return nil, fmt.Errorf("base de datos %q: %w", ruta, err)
	}
	return db, nil
}

// preparar configura el pool y aplica las migraciones pendientes.
func preparar(ctx context.Context, db *sql.DB) error {
	// Una sola conexión: serializa las escrituras (sin SQLITE_BUSY) y mantiene viva la base en memoria.
	db.SetMaxOpenConns(1)
	db.SetConnMaxIdleTime(0)
	db.SetConnMaxLifetime(0)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return err
	}
	if err := migrar(ctx, db, migracionesFS); err != nil {
		db.Close()
		return err
	}
	return nil
}

func nuevo(db *sql.DB) *Almacen {
	a := &Almacen{db: db, ahora: time.Now}
	a.Impresoras = &RepoImpresoras{a}
	a.Clientes = &RepoClientes{a}
	a.Solicitudes = &RepoSolicitudes{a}
	a.Trabajos = &RepoTrabajos{a}
	a.IDsRecientes = &RepoIDsRecientes{a}
	a.Recursos = &RepoRecursos{a}
	a.Ajustes = &RepoAjustes{a}
	return a
}

// bd devuelve la conexión vigente.
func (a *Almacen) bd() *sql.DB {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.db
}

// Cerrar libera la base de datos.
func (a *Almacen) Cerrar() error {
	return a.bd().Close()
}

// VersionEsquema devuelve la versión de migración aplicada.
func (a *Almacen) VersionEsquema(ctx context.Context) (int, error) {
	return versionEsquema(ctx, a.bd())
}

// enTransaccion ejecuta fn dentro de una transacción; la confirma si fn no devuelve error.
func (a *Almacen) enTransaccion(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := a.bd().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// traducirError convierte las violaciones de restricciones de SQLite en errores del paquete.
func traducirError(err error, que string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%s: %w", que, ErrNoEncontrado)
	}
	var se *sqlite.Error
	if errors.As(err, &se) {
		switch se.Code() {
		case sqlite3.SQLITE_CONSTRAINT_UNIQUE, sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY:
			return fmt.Errorf("%s: %w", que, ErrDuplicado)
		case sqlite3.SQLITE_CONSTRAINT_CHECK, sqlite3.SQLITE_CONSTRAINT_NOTNULL:
			return fmt.Errorf("%s: %w (%s)", que, ErrInvalido, se.Error())
		case sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY:
			return fmt.Errorf("%s: %w (referencia inexistente)", que, ErrNoEncontrado)
		}
	}
	return fmt.Errorf("%s: %w", que, err)
}

// EsOcupada indica que la base estaba bloqueada por otra conexión o por otro proceso más allá
// del busy_timeout (SQLITE_BUSY o SQLITE_LOCKED): un estado transitorio, no un error del puente.
func EsOcupada(err error) bool {
	var se *sqlite.Error
	if !errors.As(err, &se) {
		return false
	}
	codigo := se.Code() & 0xff // código primario, sin la parte extendida
	return codigo == sqlite3.SQLITE_BUSY || codigo == sqlite3.SQLITE_LOCKED
}

// Conversión de tipos para columnas INTEGER.

func ms(t time.Time) int64 { return t.UnixMilli() }

func desdeMs(v int64) time.Time { return time.UnixMilli(v).UTC() }

func desdeMsNulo(v sql.NullInt64) time.Time {
	if !v.Valid {
		return time.Time{}
	}
	return desdeMs(v.Int64)
}

func bit(b bool) int {
	if b {
		return 1
	}
	return 0
}

func textoNulo(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// filaEscaneable es común a *sql.Row y *sql.Rows.
type filaEscaneable interface {
	Scan(dest ...any) error
}
