package actualizacion

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

// Aviso guarda la última versión disponible para mostrarla en el panel. Es seguro para uso
// concurrente.
type Aviso struct {
	mu      sync.Mutex
	version *Version
}

// Disponible devuelve la versión nueva, o nil si no hay ninguna (o no se ha comprobado).
func (a *Aviso) Disponible() *Version {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.version
}

// Poner fija la versión disponible (nil la quita).
func (a *Aviso) Poner(v *Version) {
	a.mu.Lock()
	a.version = v
	a.mu.Unlock()
}

// VersionVista es lo que se recuerda de una versión nueva para avisarla aunque se reinicie el puente.
type VersionVista struct {
	Numero string    `json:"numero"`
	URL    string    `json:"url"`
	Fecha  time.Time `json:"fecha"`
}

// Estado es el resultado de las comprobaciones: se guarda en la base para que la próxima se cuente
// con la fecha y hora reales (no con el reloj del proceso, que en Windows no avanza mientras el
// equipo está suspendido) y para que el aviso sobreviva a un reinicio.
type Estado struct {
	UltimoIntento  time.Time     `json:"ultimo_intento"`
	UltimoExito    time.Time     `json:"ultimo_exito"`
	Error          string        `json:"error,omitempty"`
	FallosSeguidos int           `json:"fallos_seguidos,omitempty"`
	Disponible     *VersionVista `json:"disponible,omitempty"`
}

// Comprobador consulta GitHub una vez al día mientras Activo devuelva true. Si la consulta falla
// (por ejemplo, al salir de una suspensión, antes de que vuelva la red), reintenta a los 15
// minutos, y duplica la espera en cada fallo hasta un día.
type Comprobador struct {
	Cliente   *Cliente
	Actual    string
	Aviso     *Aviso
	Activo    func(context.Context) bool // el dueño puede desactivarlo en el panel
	Inicial   time.Duration              // espera antes de la primera revisión
	Revisar   time.Duration              // cada cuánto se revisa si ya toca consultar
	Cada      time.Duration              // entre consultas que salieron bien
	Reintento time.Duration              // primera espera después de un fallo
	Log       *slog.Logger

	// Ahora, Cargar y Guardar: el reloj y dónde se guarda el Estado (las pruebas los reemplazan).
	Ahora   func() time.Time
	Cargar  func(context.Context) Estado
	Guardar func(context.Context, Estado)

	mu     sync.Mutex
	estado Estado
}

func (c *Comprobador) ahora() time.Time {
	if c.Ahora != nil {
		return c.Ahora()
	}
	return time.Now().Round(0) // sin el reloj monótono: la fecha y hora reales
}

// Correr bloquea hasta que ctx termina. Las compilaciones de desarrollo no consultan.
func (c *Comprobador) Correr(ctx context.Context) {
	if Desarrollo(c.Actual) {
		return
	}
	c.recuperar(ctx)
	if c.Revisar <= 0 {
		c.Revisar = 10 * time.Minute
	}
	if c.Cada <= 0 {
		c.Cada = 24 * time.Hour
	}
	if c.Reintento <= 0 {
		c.Reintento = 15 * time.Minute
	}
	espera := c.Inicial
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(espera):
		}
		espera = c.Revisar
		if !c.Activo(ctx) {
			c.Aviso.Poner(nil)
			continue
		}
		if c.toca() {
			c.Comprobar(ctx)
		}
	}
}

// recuperar carga el estado guardado y, si ya se había visto una versión nueva, la avisa de
// inmediato (después de actualizar deja de serlo y no se avisa).
func (c *Comprobador) recuperar(ctx context.Context) {
	if c.Cargar == nil {
		return
	}
	e := c.Cargar(ctx)
	c.mu.Lock()
	c.estado = e
	c.mu.Unlock()
	c.ponerAviso(e.Disponible)
}

func (c *Comprobador) ponerAviso(v *VersionVista) {
	if v == nil || !EsNueva(c.Actual, v.Numero) {
		c.Aviso.Poner(nil)
		return
	}
	c.Aviso.Poner(&Version{Etiqueta: "v" + v.Numero, Numero: v.Numero, URL: v.URL, Fecha: v.Fecha, Archivos: map[string]string{}})
}

// toca dice si ya corresponde consultar: nunca se consultó, pasó un día desde la última consulta
// buena o, después de un fallo, pasó la espera de reintento.
func (c *Comprobador) toca() bool {
	c.mu.Lock()
	e := c.estado
	c.mu.Unlock()
	ahora := c.ahora()
	if e.UltimoIntento.IsZero() {
		return true
	}
	if e.FallosSeguidos > 0 {
		espera := c.Reintento
		for i := 1; i < e.FallosSeguidos && espera < c.Cada; i++ {
			espera *= 2
		}
		return !ahora.Before(e.UltimoIntento.Add(min(espera, c.Cada)))
	}
	return !ahora.Before(e.UltimoExito.Add(c.Cada))
}

// Estado devuelve el resultado de la última comprobación (para el panel).
func (c *Comprobador) Estado() Estado {
	if c == nil {
		return Estado{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.estado
}

// Comprobar consulta GitHub ahora, actualiza el aviso y guarda el estado. La usan el ciclo diario y
// el botón «Comprobar ahora» del panel.
func (c *Comprobador) Comprobar(ctx context.Context) Estado {
	v, err := c.Cliente.Ultima(ctx)
	ahora := c.ahora()
	c.mu.Lock()
	e := c.estado
	e.UltimoIntento = ahora
	switch {
	case err != nil && !errors.Is(err, ErrSinVersiones):
		e.FallosSeguidos++
		e.Error = err.Error()
		if e.FallosSeguidos == 1 {
			// Solo el primero de una racha: un negocio sin Internet no llena la bitácora.
			c.Log.Warn("no se pudo comprobar si hay una versión nueva; se reintentará en unos minutos", "error", e.Error)
		}
	default:
		if e.FallosSeguidos > 0 {
			c.Log.Info("se volvió a comprobar si hay una versión nueva", "fallos_previos", e.FallosSeguidos)
		}
		e.FallosSeguidos, e.Error, e.UltimoExito, e.Disponible = 0, "", ahora, nil
		if v != nil && EsNueva(c.Actual, v.Numero) {
			if c.Aviso.Disponible() == nil {
				c.Log.Info("hay una versión nueva de PRISMA", "actual", c.Actual, "disponible", v.Numero)
			}
			e.Disponible = &VersionVista{Numero: v.Numero, URL: v.URL, Fecha: v.Fecha}
		}
	}
	c.estado = e
	c.mu.Unlock()
	if e.Error == "" {
		c.ponerAviso(e.Disponible)
	}
	if c.Guardar != nil {
		c.Guardar(ctx, e)
	}
	return e
}
