package cola

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/bayalad7/prisma/internal/conector"
)

// falso es un conector controlable que registra el orden y la concurrencia de los envíos.
type falso struct {
	mu         sync.Mutex
	enviados   []string
	activos    int
	maxActivos int
	demora     time.Duration
	falla      error // devuelto por Enviar y Probar
	pruebas    int
	liberar    chan struct{} // si no es nil, Enviar espera a que se cierre
}

func (f *falso) Enviar(ctx context.Context, datos []byte) error {
	f.mu.Lock()
	f.activos++
	f.maxActivos = max(f.maxActivos, f.activos)
	liberar, demora, falla := f.liberar, f.demora, f.falla
	f.mu.Unlock()
	if liberar != nil {
		<-liberar
	}
	time.Sleep(demora)
	f.mu.Lock()
	f.activos--
	f.enviados = append(f.enviados, string(datos))
	f.mu.Unlock()
	return falla
}

func (f *falso) Probar(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pruebas++
	return f.falla
}

func esperar(t *testing.T, ch <-chan Resultado) Resultado {
	t.Helper()
	select {
	case r := <-ch:
		return r
	case <-time.After(5 * time.Second):
		t.Fatal("el trabajo no terminó")
	}
	return Resultado{}
}

func TestFIFOUnoALaVez(t *testing.T) {
	c := Nueva(Opciones{})
	defer c.Cerrar()
	f := &falso{demora: 5 * time.Millisecond}
	c.Registrar("caja", f)
	var canales []<-chan Resultado
	for i := range 10 {
		ch, err := c.Encolar("caja", fmt.Sprintf("t%d", i), []byte(fmt.Sprint(i)))
		if err != nil {
			t.Fatal(err)
		}
		canales = append(canales, ch)
	}
	for i, ch := range canales {
		r := esperar(t, ch)
		if !r.OK || r.TrabajoID != fmt.Sprintf("t%d", i) || r.Duracion <= 0 {
			t.Fatalf("resultado %d: %+v", i, r)
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.maxActivos != 1 {
		t.Errorf("una impresora imprimió %d trabajos a la vez", f.maxActivos)
	}
	for i, d := range f.enviados {
		if d != fmt.Sprint(i) {
			t.Fatalf("orden %v", f.enviados)
		}
	}
}

func TestImpresorasDistintasEnParalelo(t *testing.T) {
	c := Nueva(Opciones{})
	defer c.Cerrar()
	a, b := &falso{liberar: make(chan struct{})}, &falso{}
	c.Registrar("caja", a)
	c.Registrar("cocina", b)
	bloqueado, _ := c.Encolar("caja", "a1", []byte("a"))
	// cocina imprime aunque caja esté atascada.
	if r := esperar(t, mustEncolar(t, c, "cocina", "b1")); !r.OK {
		t.Fatalf("cocina: %+v", r)
	}
	close(a.liberar)
	esperar(t, bloqueado)
}

func mustEncolar(t *testing.T, c *Cola, alias, id string) <-chan Resultado {
	t.Helper()
	ch, err := c.Encolar(alias, id, []byte(id))
	if err != nil {
		t.Fatal(err)
	}
	return ch
}

func TestColaLlena(t *testing.T) {
	c := Nueva(Opciones{MaxPorImpresora: 20})
	defer c.Cerrar()
	f := &falso{liberar: make(chan struct{})}
	c.Registrar("caja", f)
	var canales []<-chan Resultado
	for i := range 20 {
		canales = append(canales, mustEncolar(t, c, "caja", fmt.Sprint(i)))
	}
	_, err := c.Encolar("caja", "21", nil)
	if conector.Codigo(err) != conector.ErrColaLlena {
		t.Fatalf("esperaba COLA_LLENA, obtuve %v", err)
	}
	if info, _ := c.EstadoDe("caja"); info.Pendientes != 20 {
		t.Errorf("pendientes = %d", info.Pendientes)
	}
	close(f.liberar)
	for _, ch := range canales {
		esperar(t, ch)
	}
	// Con la fila vacía vuelve a aceptar.
	esperar(t, mustEncolar(t, c, "caja", "nuevo"))
}

func TestImpresoraDesconocida(t *testing.T) {
	c := Nueva(Opciones{})
	defer c.Cerrar()
	if _, err := c.Encolar("nadie", "t", nil); conector.Codigo(err) != conector.ErrImpresoraNoEncontrada {
		t.Fatalf("obtuve %v", err)
	}
}

func TestEstadosYPruebaPeriodica(t *testing.T) {
	var mu sync.Mutex
	var cambios []Estado
	c := Nueva(Opciones{IntervaloPrueba: 20 * time.Millisecond, AlCambiarEstado: func(alias string, e Estado, err error) {
		mu.Lock()
		cambios = append(cambios, e)
		mu.Unlock()
	}})
	defer c.Cerrar()
	f := &falso{}
	c.Registrar("caja", f)

	esperarEstado := func(want Estado) {
		t.Helper()
		limite := time.Now().Add(3 * time.Second)
		for {
			if info, _ := c.EstadoDe("caja"); info.Estado == want {
				return
			}
			if time.Now().After(limite) {
				info, _ := c.EstadoDe("caja")
				t.Fatalf("estado %s, quiero %s", info.Estado, want)
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	esperarEstado(EnLinea)

	falla := &conector.Error{Codigo: conector.ErrImpresoraInalcanzable, Mensaje: "x", Sugerencia: "y"}
	f.mu.Lock()
	f.falla = falla
	f.mu.Unlock()
	esperarEstado(FueraDeLinea)
	if info, _ := c.EstadoDe("caja"); info.UltimoError != conector.ErrImpresoraInalcanzable {
		t.Errorf("último error: %+v", info)
	}

	f.mu.Lock()
	f.falla = nil
	f.liberar = make(chan struct{})
	f.mu.Unlock()
	esperarEstado(EnLinea)
	ch := mustEncolar(t, c, "caja", "t1")
	esperarEstado(Ocupada)
	close(f.liberar)
	esperar(t, ch)
	esperarEstado(EnLinea)

	mu.Lock()
	defer mu.Unlock()
	if len(cambios) < 4 || cambios[0] != EnLinea {
		t.Errorf("cambios de estado: %v", cambios)
	}
}

func TestErrorDelConectorLlegaAlResultado(t *testing.T) {
	c := Nueva(Opciones{})
	defer c.Cerrar()
	falla := &conector.Error{Codigo: conector.ErrTiempoAgotado, Mensaje: "x", Sugerencia: "y"}
	c.Registrar("caja", &falso{falla: falla})
	r := esperar(t, mustEncolar(t, c, "caja", "t1"))
	if r.OK || !errors.Is(r.Err, falla) {
		t.Fatalf("resultado: %+v", r)
	}
}

func TestQuitarYCerrarRespondenALosPendientes(t *testing.T) {
	c := Nueva(Opciones{})
	f := &falso{liberar: make(chan struct{})}
	c.Registrar("caja", f)
	enCurso := mustEncolar(t, c, "caja", "t1")
	enEspera := mustEncolar(t, c, "caja", "t2")
	time.Sleep(20 * time.Millisecond) // t1 ya está en Enviar
	listo := make(chan struct{})
	go func() { c.Quitar("caja"); close(listo) }()
	time.Sleep(20 * time.Millisecond) // Quitar ya canceló y espera a que termine t1
	close(f.liberar)
	<-listo
	if r := esperar(t, enCurso); !r.OK {
		t.Errorf("el trabajo en curso debe terminar: %+v", r)
	}
	if r := esperar(t, enEspera); !errors.Is(r.Err, ErrCerrada) {
		t.Errorf("el trabajo en espera debe recibir ErrCerrada: %+v", r)
	}
	if _, err := c.Encolar("caja", "t3", nil); conector.Codigo(err) != conector.ErrImpresoraNoEncontrada {
		t.Errorf("tras quitarla: %v", err)
	}
	c.Cerrar()
}

func TestRegistrarDeNuevoCambiaElConector(t *testing.T) {
	c := Nueva(Opciones{})
	defer c.Cerrar()
	viejo, nuevo := &falso{}, &falso{}
	c.Registrar("caja", viejo)
	c.Registrar("caja", nuevo)
	esperar(t, mustEncolar(t, c, "caja", "t1"))
	if len(nuevo.enviados) != 1 || len(viejo.enviados) != 0 {
		t.Fatal("debe usarse el conector nuevo")
	}
	if len(c.Estados()) != 1 {
		t.Fatal("no debe duplicarse la impresora")
	}
}
