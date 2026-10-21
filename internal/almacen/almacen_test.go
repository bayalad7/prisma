package almacen

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// reloj controla el tiempo que ve el almacén en las pruebas.
type reloj struct{ t time.Time }

func (r *reloj) ahora() time.Time        { return r.t }
func (r *reloj) avanzar(d time.Duration) { r.t = r.t.Add(d) }

func nuevoAlmacen(t *testing.T) (*Almacen, *reloj) {
	t.Helper()
	a, err := AbrirMemoria(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Cerrar() })
	r := &reloj{t: time.Date(2026, 10, 14, 18, 0, 0, 0, time.UTC)}
	a.ahora = r.ahora
	return a, r
}

func exigirError(t *testing.T, err, esperado error) {
	t.Helper()
	if !errors.Is(err, esperado) {
		t.Fatalf("esperaba %v, obtuve %v", esperado, err)
	}
}

func exigirSinError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func impresoraDePrueba(alias string) *Impresora {
	return &Impresora{
		Alias: alias, Nombre: "Impresora " + alias, Tipo: ConexionRed, Destino: "192.168.1.50:9100",
		AnchoMM: 80, Columnas: 48, ColumnasFuenteB: 64, AnchoPx: 576,
		PaginaCodigos: "CP858", ComandoPaginaCodigos: 19,
		SoportaQRNativo: true, SoportaCorte: true, SoportaImagen: true, SoportaCodigoBarrasNativo: true, Activa: true,
	}
}

func clienteDePrueba(id string) *Cliente {
	return &Cliente{
		ID: id, Nombre: "App " + id, Nivel: NivelLlave,
		LlaveHash: hashPrueba, LlavePrefijo: "pk_" + id[4:], TodasLasImpresoras: true, Activo: true,
	}
}

var ctx = context.Background()

// hashPrueba tiene forma de PHC argon2id; las pruebas del almacén no lo verifican.
const hashPrueba = "$argon2id$v=19$m=8,t=1,p=1$c2Fs$aGFzaA"

// Una base bloqueada por otro proceso se reconoce como ocupada (estado transitorio).
func TestEsOcupada(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "prisma.db")
	a, err := Abrir(ctx, ruta)
	exigirSinError(t, err)
	defer a.Cerrar()
	otro, err := sql.Open("sqlite", ruta+"?_pragma=busy_timeout(0)")
	exigirSinError(t, err)
	defer otro.Close()
	tx, err := otro.BeginTx(ctx, nil)
	exigirSinError(t, err)
	_, err = tx.ExecContext(ctx, "INSERT INTO ajustes (clave, valor, actualizado) VALUES ('bloqueo', '1', 0)")
	exigirSinError(t, err)
	defer tx.Rollback()

	err = a.Ajustes.Establecer(ctx, "x", "y") // espera busy_timeout (5 s) y falla
	if !EsOcupada(err) {
		t.Fatalf("se esperaba base ocupada: %v", err)
	}
	if EsOcupada(ErrNoEncontrado) || EsOcupada(nil) {
		t.Fatal("solo los errores de bloqueo de SQLite son «ocupada»")
	}
}
