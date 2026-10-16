package almacen

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func memoriaSinMigrar(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?"+pragmas)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMigracionesCreanTodasLasTablas(t *testing.T) {
	a, _ := nuevoAlmacen(t)
	v, err := a.VersionEsquema(ctx)
	exigirSinError(t, err)
	ms, _ := cargarMigraciones(migracionesFS)
	if v != len(ms) || v < 1 {
		t.Fatalf("versión de esquema %d, migraciones %d", v, len(ms))
	}
	for _, tabla := range []string{"impresoras", "clientes", "clientes_impresoras", "clientes_origenes",
		"solicitudes_origen", "trabajos", "ids_recientes", "recursos", "ajustes"} {
		var n int
		exigirSinError(t, a.db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_schema WHERE type = 'table' AND name = ?`, tabla).Scan(&n))
		if n != 1 {
			t.Errorf("falta la tabla %s", tabla)
		}
	}
	var fk int
	exigirSinError(t, a.db.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&fk))
	if fk != 1 {
		t.Error("foreign_keys debe estar activo")
	}
}

func TestMigrarEsIdempotente(t *testing.T) {
	a, _ := nuevoAlmacen(t)
	exigirSinError(t, a.Ajustes.Establecer(ctx, "x", "1"))
	exigirSinError(t, migrar(ctx, a.db, migracionesFS))
	if v, ok, _ := a.Ajustes.Obtener(ctx, "x"); !ok || v != "1" {
		t.Fatal("volver a migrar no debe tocar los datos")
	}
}

func TestMigrarRechazaEsquemaMasNuevo(t *testing.T) {
	db := memoriaSinMigrar(t)
	exigirSinError(t, migrar(ctx, db, migracionesFS))
	_, err := db.ExecContext(ctx, `PRAGMA user_version = 99`)
	exigirSinError(t, err)
	err = migrar(ctx, db, migracionesFS)
	if err == nil || !strings.Contains(err.Error(), "esquema 99") {
		t.Fatalf("esperaba error de esquema más nuevo, obtuve %v", err)
	}
}

func TestMigracionFallidaSeRevierte(t *testing.T) {
	db := memoriaSinMigrar(t)
	fsys := fstest.MapFS{
		"migraciones/0001_a.sql": {Data: []byte(`CREATE TABLE a (x INTEGER);`)},
		"migraciones/0002_b.sql": {Data: []byte(`CREATE TABLE b (x INTEGER); INSERT INTO no_existe VALUES (1);`)},
	}
	if err := migrar(ctx, db, fsys); err == nil || !strings.Contains(err.Error(), "0002_b") {
		t.Fatalf("esperaba fallo en 0002_b, obtuve %v", err)
	}
	v, _ := versionEsquema(ctx, db)
	if v != 1 {
		t.Errorf("la versión debe quedar en 1, quedó en %d", v)
	}
	var n int
	exigirSinError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_schema WHERE name = 'b'`).Scan(&n))
	if n != 0 {
		t.Error("la tabla b de la migración fallida no debe existir")
	}
}

func TestMigracionesSinHuecos(t *testing.T) {
	fsys := fstest.MapFS{
		"migraciones/0001_a.sql": {Data: []byte(`SELECT 1;`)},
		"migraciones/0003_c.sql": {Data: []byte(`SELECT 1;`)},
	}
	if _, err := cargarMigraciones(fsys); err == nil {
		t.Fatal("esperaba error por hueco en la numeración")
	}
	fsys = fstest.MapFS{"migraciones/inicial.sql": {Data: []byte(`SELECT 1;`)}}
	if _, err := cargarMigraciones(fsys); err == nil {
		t.Fatal("esperaba error por nombre inválido")
	}
}

func TestAbrirArchivoPersiste(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "sub", "prisma.db")
	a, err := Abrir(ctx, ruta)
	exigirSinError(t, err)
	exigirSinError(t, a.Ajustes.Establecer(ctx, "x", "persistente"))
	exigirSinError(t, a.Cerrar())

	a, err = Abrir(ctx, ruta)
	exigirSinError(t, err)
	defer a.Cerrar()
	if v, ok, _ := a.Ajustes.Obtener(ctx, "x"); !ok || v != "persistente" {
		t.Fatal("el dato no persistió al reabrir")
	}
	var modo string
	exigirSinError(t, a.db.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&modo))
	if modo != "wal" {
		t.Errorf("journal_mode = %q, quiero wal", modo)
	}
}

// TestMigracion0002ConservaReferencias reconstruye la tabla clientes con datos reales: los
// orígenes y permisos no deben borrarse en cascada y los hashes SHA-256 viejos se descartan.
func TestMigracion0002ConservaReferencias(t *testing.T) {
	db := memoriaSinMigrar(t)
	todas, _ := cargarMigraciones(migracionesFS)
	solo1 := fstest.MapFS{"migraciones/0001_inicial.sql": {Data: []byte(todas[0].sql)}}
	exigirSinError(t, migrar(ctx, db, solo1))
	for _, q := range []string{
		`INSERT INTO impresoras (alias, nombre, tipo, destino, ancho_mm, columnas, columnas_fuente_b, ancho_px,
			pagina_codigos, comando_pagina_codigos, creada, actualizada) VALUES ('caja','c','red','x',80,48,64,576,'CP858',19,0,0)`,
		`INSERT INTO clientes (id, nombre, nivel, llave_hash, llave_prefijo, creado, actualizado)
			VALUES ('cli_viejo0001', 'Viejo', 2, randomblob(32), 'pk_viejo', 0, 0)`,
		`INSERT INTO clientes (id, nombre, nivel, llave_publica, creado, actualizado)
			VALUES ('cli_firma0001', 'Firma', 3, randomblob(32), 0, 0)`,
		`INSERT INTO clientes_origenes VALUES ('cli_viejo0001', 'https://viejo.example')`,
		`INSERT INTO clientes_impresoras VALUES ('cli_firma0001', 1)`,
	} {
		_, err := db.ExecContext(ctx, q)
		exigirSinError(t, err)
	}

	exigirSinError(t, migrar(ctx, db, migracionesFS))
	var origenes, permisos, fk int
	db.QueryRowContext(ctx, `SELECT count(*) FROM clientes_origenes`).Scan(&origenes)
	db.QueryRowContext(ctx, `SELECT count(*) FROM clientes_impresoras`).Scan(&permisos)
	db.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&fk)
	if origenes != 1 || permisos != 1 {
		t.Fatalf("la reconstrucción borró referencias: orígenes %d, permisos %d", origenes, permisos)
	}
	if fk != 1 {
		t.Fatal("las claves foráneas deben quedar activas después de migrar")
	}
	var nivel, activo int
	var hash sql.NullString
	exigirSinError(t, db.QueryRowContext(ctx, `SELECT nivel, activo, llave_hash FROM clientes WHERE id = 'cli_viejo0001'`).Scan(&nivel, &activo, &hash))
	if nivel != 1 || activo != 0 || hash.Valid {
		t.Fatalf("cliente con hash SHA-256: nivel %d, activo %d, hash %v", nivel, activo, hash)
	}
	exigirSinError(t, db.QueryRowContext(ctx, `SELECT nivel, activo FROM clientes WHERE id = 'cli_firma0001'`).Scan(&nivel, &activo))
	if nivel != 3 || activo != 1 {
		t.Fatalf("cliente de nivel 3: nivel %d, activo %d", nivel, activo)
	}
}
