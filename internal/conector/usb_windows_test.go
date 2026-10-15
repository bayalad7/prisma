//go:build windows

package conector

import (
	"context"
	"testing"
)

func TestWinspoolImpresoraInexistente(t *testing.T) {
	w := &Winspool{Nombre: "PRISMA impresora que no existe"}
	exigirCodigo(t, w.Probar(context.Background()), ErrImpresoraNoEncontrada)
	exigirCodigo(t, w.Enviar(context.Background(), []byte("x")), ErrImpresoraNoEncontrada)
}

func TestEnumerarImpresorasWindows(t *testing.T) {
	ds, err := impresorasWindows()
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range ds {
		if d.Tipo != "usb" || d.Destino == "" {
			t.Errorf("impresora mal leída: %+v", d)
		}
	}
	t.Logf("%d impresoras instaladas", len(ds))
}
