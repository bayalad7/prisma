// Package cola ordena los trabajos de impresión: una goroutine por impresora imprime en orden
// FIFO y de a un trabajo a la vez, y las impresoras distintas trabajan en paralelo. Esa misma
// goroutine prueba la impresora periódicamente, así nunca se prueba mientras imprime.
package cola

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/bayalad7/prisma/internal/conector"
)

// Estado de una impresora.
type Estado string

const (
	Desconocido  Estado = "desconocido"
	EnLinea      Estado = "en_linea"
	FueraDeLinea Estado = "fuera_de_linea"
	Ocupada      Estado = "ocupada"
)

// ErrCerrada: la cola se cerró (o se quitó la impresora) antes de imprimir el trabajo.
var ErrCerrada = errors.New("la cola de impresión se cerró antes de imprimir el trabajo")

// Resultado de un trabajo, entregado por el canal que devuelve Encolar.
type Resultado struct {
	TrabajoID string
	OK        bool
	Duracion  time.Duration
	Err       error // *conector.Error, ErrCerrada u otro
}

// Opciones de la cola. Los valores cero usan los valores por defecto.
type Opciones struct {
	MaxPorImpresora int           // trabajos en espera más el que se imprime; 20
	IntervaloPrueba time.Duration // 30 s
	TiempoTrabajo   time.Duration // máximo por trabajo; 60 s
	TiempoPrueba    time.Duration // máximo por prueba; 5 s

	// AlCambiarEstado se invoca (desde la goroutine de la impresora) cuando cambia su estado.
	AlCambiarEstado func(alias string, e Estado, err error)
	// AlIniciar se invoca justo antes de enviar un trabajo a la impresora.
	AlIniciar func(alias, trabajoID string)
}

// Cola administra una fila de trabajos por impresora.
type Cola struct {
	op         Opciones
	ctx        context.Context
	cancelar   context.CancelFunc
	mu         sync.Mutex
	impresoras map[string]*impresora
}

type trabajo struct {
	id        string
	datos     []byte
	resultado chan Resultado
}

type impresora struct {
	alias    string
	trabajos chan *trabajo
	fin      context.CancelFunc
	hecho    chan struct{}

	mu         sync.Mutex
	con        conector.Conector
	estado     Estado
	ultimo     error
	pendientes int  // en espera + en curso
	cerrada    bool // ya no acepta trabajos
}

// Nueva crea una cola vacía.
func Nueva(op Opciones) *Cola {
	if op.MaxPorImpresora <= 0 {
		op.MaxPorImpresora = 20
	}
	if op.IntervaloPrueba <= 0 {
		op.IntervaloPrueba = 30 * time.Second
	}
	if op.TiempoTrabajo <= 0 {
		op.TiempoTrabajo = 60 * time.Second
	}
	if op.TiempoPrueba <= 0 {
		op.TiempoPrueba = 5 * time.Second
	}
	ctx, cancelar := context.WithCancel(context.Background())
	return &Cola{op: op, ctx: ctx, cancelar: cancelar, impresoras: map[string]*impresora{}}
}

// Registrar agrega una impresora o, si ya existe, cambia su conector (los trabajos en espera
// se conservan y usarán el nuevo).
func (c *Cola) Registrar(alias string, con conector.Conector) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if imp, ok := c.impresoras[alias]; ok {
		imp.mu.Lock()
		imp.con = con
		imp.mu.Unlock()
		return
	}
	ctx, fin := context.WithCancel(c.ctx)
	imp := &impresora{alias: alias, con: con, estado: Desconocido, fin: fin,
		trabajos: make(chan *trabajo, c.op.MaxPorImpresora), hecho: make(chan struct{})}
	c.impresoras[alias] = imp
	go c.atender(ctx, imp)
}

// Quitar detiene la impresora. Los trabajos que esperaban terminan con ErrCerrada.
func (c *Cola) Quitar(alias string) {
	c.mu.Lock()
	imp, ok := c.impresoras[alias]
	delete(c.impresoras, alias)
	c.mu.Unlock()
	if ok {
		imp.fin()
		<-imp.hecho
	}
}

// Cerrar detiene todas las impresoras y espera a que terminen.
func (c *Cola) Cerrar() {
	c.mu.Lock()
	imps := make([]*impresora, 0, len(c.impresoras))
	for _, imp := range c.impresoras {
		imps = append(imps, imp)
	}
	c.impresoras = map[string]*impresora{}
	c.mu.Unlock()
	c.cancelar()
	for _, imp := range imps {
		<-imp.hecho
	}
}

// Encolar agrega un trabajo al final de la fila de la impresora. El resultado llega por el
// canal devuelto (con capacidad 1, así que nadie bloquea si no se lee).
func (c *Cola) Encolar(alias, trabajoID string, datos []byte) (<-chan Resultado, error) {
	c.mu.Lock()
	imp, ok := c.impresoras[alias]
	c.mu.Unlock()
	if !ok {
		return nil, &conector.Error{Codigo: conector.ErrImpresoraNoEncontrada,
			Mensaje:    fmt.Sprintf("No hay ninguna impresora activa con el alias %q.", alias),
			Sugerencia: "Revisa el alias o activa la impresora en el panel."}
	}
	imp.mu.Lock()
	defer imp.mu.Unlock()
	if imp.cerrada {
		return nil, ErrCerrada
	}
	if imp.pendientes >= c.op.MaxPorImpresora {
		return nil, &conector.Error{Codigo: conector.ErrColaLlena,
			Mensaje:    fmt.Sprintf("La impresora %q ya tiene %d trabajos pendientes.", alias, c.op.MaxPorImpresora),
			Sugerencia: "Espera a que termine de imprimir y vuelve a intentarlo; si no avanza, revisa la impresora."}
	}
	imp.pendientes++
	t := &trabajo{id: trabajoID, datos: datos, resultado: make(chan Resultado, 1)}
	// Bajo el candado y sin bloquear (pendientes ≤ capacidad del canal): así vaciar no puede
	// dejar un trabajo sin respuesta.
	imp.trabajos <- t
	return t.resultado, nil
}

// InfoImpresora resume el estado de una impresora para el panel y la API.
type InfoImpresora struct {
	Alias       string `json:"alias"`
	Estado      Estado `json:"estado"`
	Pendientes  int    `json:"pendientes"`
	UltimoError string `json:"ultimo_error,omitempty"` // código del último error
}

// Estados devuelve el estado de todas las impresoras, ordenadas por alias.
func (c *Cola) Estados() []InfoImpresora {
	c.mu.Lock()
	var lista []InfoImpresora
	for _, imp := range c.impresoras {
		lista = append(lista, imp.info())
	}
	c.mu.Unlock()
	sort.Slice(lista, func(i, j int) bool { return lista[i].Alias < lista[j].Alias })
	return lista
}

// EstadoDe devuelve el estado de una impresora.
func (c *Cola) EstadoDe(alias string) (InfoImpresora, bool) {
	c.mu.Lock()
	imp, ok := c.impresoras[alias]
	c.mu.Unlock()
	if !ok {
		return InfoImpresora{}, false
	}
	return imp.info(), true
}

func (imp *impresora) info() InfoImpresora {
	imp.mu.Lock()
	defer imp.mu.Unlock()
	return InfoImpresora{Alias: imp.alias, Estado: imp.estado, Pendientes: imp.pendientes, UltimoError: conector.Codigo(imp.ultimo)}
}

func (c *Cola) cambiar(imp *impresora, e Estado, err error) {
	imp.mu.Lock()
	anterior := imp.estado
	imp.estado, imp.ultimo = e, err
	imp.mu.Unlock()
	if anterior != e && c.op.AlCambiarEstado != nil {
		c.op.AlCambiarEstado(imp.alias, e, err)
	}
}

func (imp *impresora) conectorActual() conector.Conector {
	imp.mu.Lock()
	defer imp.mu.Unlock()
	return imp.con
}

// atender es la goroutine de una impresora.
func (c *Cola) atender(ctx context.Context, imp *impresora) {
	defer close(imp.hecho)
	cada := time.NewTicker(c.op.IntervaloPrueba)
	defer cada.Stop()
	c.probar(ctx, imp)
	for {
		select {
		case <-ctx.Done():
			c.vaciar(imp)
			return
		case t := <-imp.trabajos:
			c.imprimir(ctx, imp, t)
		case <-cada.C:
			c.probar(ctx, imp)
		}
	}
}

func (c *Cola) probar(ctx context.Context, imp *impresora) {
	ctx, fin := context.WithTimeout(ctx, c.op.TiempoPrueba)
	defer fin()
	if err := imp.conectorActual().Probar(ctx); err != nil {
		c.cambiar(imp, FueraDeLinea, err)
		return
	}
	c.cambiar(imp, EnLinea, nil)
}

func (c *Cola) imprimir(ctx context.Context, imp *impresora, t *trabajo) {
	if ctx.Err() != nil {
		c.terminar(imp, t, Resultado{TrabajoID: t.id, Err: ErrCerrada})
		return
	}
	if c.op.AlIniciar != nil {
		c.op.AlIniciar(imp.alias, t.id)
	}
	c.cambiar(imp, Ocupada, nil)
	ctx, fin := context.WithTimeout(conector.ConTrabajo(ctx, t.id), c.op.TiempoTrabajo)
	inicio := time.Now()
	err := imp.conectorActual().Enviar(ctx, t.datos)
	fin()
	r := Resultado{TrabajoID: t.id, OK: err == nil, Duracion: time.Since(inicio), Err: err}
	if err != nil {
		c.cambiar(imp, FueraDeLinea, err)
	} else {
		c.cambiar(imp, EnLinea, nil)
	}
	c.terminar(imp, t, r)
}

func (c *Cola) terminar(imp *impresora, t *trabajo, r Resultado) {
	imp.mu.Lock()
	imp.pendientes--
	imp.mu.Unlock()
	t.resultado <- r
}

// vaciar cierra la impresora a nuevos trabajos y responde ErrCerrada a los que esperaban.
func (c *Cola) vaciar(imp *impresora) {
	imp.mu.Lock()
	imp.cerrada = true
	var restantes []*trabajo
	for len(imp.trabajos) > 0 {
		restantes = append(restantes, <-imp.trabajos)
	}
	imp.mu.Unlock()
	for _, t := range restantes {
		c.terminar(imp, t, Resultado{TrabajoID: t.id, Err: ErrCerrada})
	}
}
