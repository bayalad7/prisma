package almacen

import (
	"context"
	"errors"
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
		LlaveHash: make([]byte, 32), LlavePrefijo: "pk_" + id[4:], TodasLasImpresoras: true, Activo: true,
	}
}

var ctx = context.Background()
