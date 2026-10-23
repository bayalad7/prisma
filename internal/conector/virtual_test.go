package conector

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/bayalad7/prisma/internal/almacen"
)

func impresoraVirtual(sim string) almacen.Impresora {
	return almacen.Impresora{Alias: "prueba", Tipo: almacen.ConexionVirtual, Destino: "memoria", AnchoMM: 80, Columnas: 48,
		ColumnasFuenteB: 64, AnchoPx: 576, Simulacion: sim, SimulacionRetrasoMS: 3000, SimulacionFallaPct: 30, TicketsEnMemoria: 3}
}

func TestNuevoCreaLaImpresoraVirtual(t *testing.T) {
	con, err := Nuevo(impresoraVirtual(almacen.SimulacionNormal))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := con.(*Virtual); !ok {
		t.Fatalf("Nuevo devolvió %T", con)
	}
}

func TestVirtualGuardaLosBytesExactosEnElRollo(t *testing.T) {
	r := NuevosRollos()
	v := NuevoVirtual(impresoraVirtual(almacen.SimulacionNormal), r)
	if err := v.Probar(context.Background()); err != nil {
		t.Fatal(err)
	}
	for k := 1; k <= 5; k++ {
		datos := []byte(fmt.Sprintf("\x1b@ticket %d\n", k))
		if err := v.Enviar(ConTrabajo(context.Background(), fmt.Sprintf("trb_%d", k)), datos); err != nil {
			t.Fatal(err)
		}
		datos[0] = 'X' // el rollo guarda una copia
	}
	lista := r.Listar("prueba")
	if len(lista) != 3 || lista[0].TrabajoID != "trb_5" || lista[2].TrabajoID != "trb_3" {
		t.Fatalf("el rollo debe tener los 3 más recientes, del más nuevo al más viejo: %+v", lista)
	}
	if !bytes.Equal(lista[0].Datos, []byte("\x1b@ticket 5\n")) {
		t.Fatalf("bytes guardados: %q", lista[0].Datos)
	}
	if _, ok := r.Ticket("prueba", "trb_1"); ok {
		t.Fatal("el ticket más viejo debió salir del rollo")
	}
	r.Vaciar("prueba")
	if len(r.Listar("prueba")) != 0 {
		t.Fatal("Vaciar debe borrar el rollo")
	}
}

func TestRolloLimitaLaMemoria(t *testing.T) {
	r := NuevosRollos()
	grande := make([]byte, maxBytesRollo/2+1)
	for k := range 3 {
		r.agregar("prueba", 20, TicketVirtual{TrabajoID: fmt.Sprint(k), Datos: grande})
	}
	if n := len(r.Listar("prueba")); n != 1 {
		t.Fatalf("con tickets de más de la mitad del límite debe quedar 1, quedan %d", n)
	}
}

func TestVirtualSimulaFueraDeLinea(t *testing.T) {
	r := NuevosRollos()
	v := NuevoVirtual(impresoraVirtual(almacen.SimulacionFueraDeLinea), r)
	if Codigo(v.Probar(context.Background())) != ErrImpresoraInalcanzable {
		t.Fatal("Probar debe responder IMPRESORA_INALCANZABLE")
	}
	if err := v.Enviar(context.Background(), []byte("x")); Codigo(err) != ErrImpresoraInalcanzable {
		t.Fatalf("Enviar: %v", err)
	}
	if len(r.Listar("prueba")) != 0 {
		t.Fatal("un ticket que no llegó no se guarda")
	}
}

func TestVirtualSimulaLentitud(t *testing.T) {
	r := NuevosRollos()
	v := NuevoVirtual(impresoraVirtual(almacen.SimulacionLenta), r)
	var esperado time.Duration
	v.esperar = func(_ context.Context, d time.Duration) error { esperado = d; return nil }
	if err := v.Enviar(context.Background(), []byte("x")); err != nil || esperado != 3*time.Second {
		t.Fatalf("Enviar: %v, esperó %v", err, esperado)
	}

	// Si se cancela mientras espera (p. ej. al cerrar la cola), el ticket no llega.
	v = NuevoVirtual(impresoraVirtual(almacen.SimulacionLenta), r)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	err := v.Enviar(ctx, []byte("y"))
	if Codigo(err) != ErrTiempoAgotado || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelado: %v", err)
	}
	if len(r.Listar("prueba")) != 1 {
		t.Fatal("solo debe estar el ticket que terminó de llegar")
	}
}

func TestVirtualSimulaFallasAleatorias(t *testing.T) {
	r := NuevosRollos()
	v := NuevoVirtual(impresoraVirtual(almacen.SimulacionFallaAleatoria), r)
	v.azar = func() float64 { return 0.29 } // 29 < 30 %: falla
	if err := v.Enviar(context.Background(), []byte("x")); Codigo(err) != ErrImpresoraInalcanzable {
		t.Fatalf("debía fallar: %v", err)
	}
	v.azar = func() float64 { return 0.30 }
	if err := v.Enviar(context.Background(), []byte("x")); err != nil {
		t.Fatalf("no debía fallar: %v", err)
	}
	if len(r.Listar("prueba")) != 1 {
		t.Fatal("solo se guarda el ticket que llegó")
	}
}
