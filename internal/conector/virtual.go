package conector

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/bayalad7/prisma/internal/almacen"
)

// Virtual es una impresora que vive dentro del puente: recibe exactamente los bytes ESC/POS que
// recibiría una impresora física y los guarda en un rollo en memoria para mostrarlos en el panel.
// No usa red, puertos ni archivos. Puede simular fallas que la API v1 ya sabe informar, para que
// una aplicación pruebe cómo las maneja.
type Virtual struct {
	Perfil  almacen.Impresora
	rollos  *Rollos
	azar    func() float64 // [0, 1)
	esperar func(context.Context, time.Duration) error
	ahora   func() time.Time
}

// NuevoVirtual crea el conector de una impresora virtual que guarda sus tickets en rollos.
func NuevoVirtual(imp almacen.Impresora, rollos *Rollos) *Virtual {
	return &Virtual{Perfil: imp, rollos: rollos, azar: rand.Float64, esperar: esperarCtx, ahora: time.Now}
}

func esperarCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Probar responde como una impresora en línea, salvo que simule estar fuera de línea.
func (v *Virtual) Probar(context.Context) error {
	if v.Perfil.Simulacion == almacen.SimulacionFueraDeLinea {
		return v.fueraDeLinea()
	}
	return nil
}

// Enviar aplica la simulación y, si el ticket «llega», lo agrega al rollo. Un ticket que falla no se
// guarda: no se imprimió, y la aplicación puede reintentarlo sin duplicarlo.
func (v *Virtual) Enviar(ctx context.Context, datos []byte) error {
	imp := v.Perfil
	switch imp.Simulacion {
	case almacen.SimulacionFueraDeLinea:
		return v.fueraDeLinea()
	case almacen.SimulacionLenta:
		if err := v.esperar(ctx, time.Duration(imp.SimulacionRetrasoMS)*time.Millisecond); err != nil {
			return nuevoError(ErrTiempoAgotado, fmt.Sprintf("La impresora virtual «%s» no terminó de recibir el ticket.", imp.Alias),
				"Se canceló mientras simulaba ser lenta. Reintenta, o baja el retraso de la simulación en el panel.", err)
		}
	case almacen.SimulacionFallaAleatoria:
		if v.azar()*100 < float64(imp.SimulacionFallaPct) {
			return nuevoError(ErrImpresoraInalcanzable,
				fmt.Sprintf("La impresora virtual «%s» simuló una falla (falla aleatoria en %d de cada 100 tickets).", imp.Alias, imp.SimulacionFallaPct),
				"El ticket no se imprimió: tu aplicación puede reintentarlo. Para quitar las fallas, cambia la simulación a «Normal» en el panel.", nil)
		}
	}
	v.rollos.agregar(imp.Alias, imp.TicketsEnMemoria, TicketVirtual{
		TrabajoID: trabajoDe(ctx), Recibido: v.ahora(), Datos: append([]byte(nil), datos...),
	})
	return nil
}

func (v *Virtual) fueraDeLinea() error {
	return nuevoError(ErrImpresoraInalcanzable, fmt.Sprintf("La impresora virtual «%s» está simulando que está fuera de línea.", v.Perfil.Alias),
		"Es una prueba: en el panel, cambia su simulación a «Normal» para que vuelva a recibir tickets.", nil)
}

// TicketVirtual es un ticket que recibió una impresora virtual: los bytes ESC/POS tal como llegaron.
type TicketVirtual struct {
	TrabajoID string
	Recibido  time.Time
	Datos     []byte
}

// maxBytesRollo limita la memoria de cada impresora virtual, además del número de tickets.
const maxBytesRollo = 16 << 20

// Rollos guarda en memoria los últimos tickets de cada impresora virtual. Nunca se escriben en la
// base, la bitácora ni el disco: se pierden al reiniciar el puente.
type Rollos struct {
	mu       sync.Mutex
	porAlias map[string][]TicketVirtual // del más antiguo al más reciente
}

// NuevosRollos crea un almacén vacío.
func NuevosRollos() *Rollos { return &Rollos{porAlias: map[string][]TicketVirtual{}} }

// RollosVirtuales son los rollos del puente: los usan el conector virtual y el panel.
var RollosVirtuales = NuevosRollos()

func (r *Rollos) agregar(alias string, limite int, t TicketVirtual) {
	if limite <= 0 {
		limite = 20
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	lista := append(r.porAlias[alias], t)
	total := 0
	for _, x := range lista {
		total += len(x.Datos)
	}
	for len(lista) > 1 && (len(lista) > limite || total > maxBytesRollo) {
		total -= len(lista[0].Datos)
		lista = lista[1:]
	}
	r.porAlias[alias] = append([]TicketVirtual(nil), lista...)
}

// Listar devuelve los tickets de una impresora, del más reciente al más antiguo.
func (r *Rollos) Listar(alias string) []TicketVirtual {
	r.mu.Lock()
	defer r.mu.Unlock()
	lista := r.porAlias[alias]
	res := make([]TicketVirtual, len(lista))
	for i, t := range lista {
		res[len(lista)-1-i] = t
	}
	return res
}

// Ticket busca el ticket de un trabajo.
func (r *Rollos) Ticket(alias, trabajoID string) (TicketVirtual, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.porAlias[alias] {
		if t.TrabajoID == trabajoID {
			return t, true
		}
	}
	return TicketVirtual{}, false
}

// Vaciar borra los tickets de una impresora (también al eliminarla).
func (r *Rollos) Vaciar(alias string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.porAlias, alias)
}
