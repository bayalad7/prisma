//go:build !windows

package conector

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDispositivoInexistente(t *testing.T) {
	d := nuevoUSB("/dev/usb/lp-prisma-no-existe")
	exigirCodigo(t, d.Probar(context.Background()), ErrImpresoraNoEncontrada)
}

func TestDispositivoEscribe(t *testing.T) {
	// Un archivo común en /dev no existe en las pruebas; se usa la implementación con otra ruta.
	ruta := filepath.Join(t.TempDir(), "lp0")
	os.WriteFile(ruta, nil, 0o600)
	d := &Dispositivo{Ruta: ruta}
	if err := d.Enviar(context.Background(), []byte("hola")); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(ruta); string(got) != "hola" {
		t.Fatalf("contenido %q", got)
	}
}

func TestCUPSColaInexistente(t *testing.T) {
	c := nuevoUSB("prisma-cola-que-no-existe")
	if _, ok := c.(*CUPS); !ok {
		t.Fatalf("un nombre sin /dev/ debe usar CUPS: %T", c)
	}
	// Con o sin CUPS instalado, una cola inexistente no se encuentra.
	exigirCodigo(t, c.Probar(context.Background()), ErrImpresoraNoEncontrada)
}
