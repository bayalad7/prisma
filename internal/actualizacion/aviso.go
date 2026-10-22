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

// Comprobador consulta GitHub periódicamente mientras Activo devuelva true.
type Comprobador struct {
	Cliente *Cliente
	Actual  string
	Aviso   *Aviso
	Activo  func(context.Context) bool // el dueño puede desactivarlo en el panel
	Inicial time.Duration              // espera antes de la primera consulta
	Cada    time.Duration
	Log     *slog.Logger
}

// Correr bloquea hasta que ctx termina. Las compilaciones de desarrollo no consultan.
func (c *Comprobador) Correr(ctx context.Context) {
	if Desarrollo(c.Actual) {
		return
	}
	espera := c.Inicial
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(espera):
		}
		espera = c.Cada
		if !c.Activo(ctx) {
			c.Aviso.Poner(nil)
			continue
		}
		v, err := c.Cliente.Ultima(ctx)
		if errors.Is(err, ErrSinVersiones) {
			continue
		}
		if err != nil {
			c.Log.Warn("no se pudo comprobar si hay una versión nueva", "error", err.Error())
			continue
		}
		if EsNueva(c.Actual, v.Numero) {
			if c.Aviso.Disponible() == nil {
				c.Log.Info("hay una versión nueva de PRISMA", "actual", c.Actual, "disponible", v.Numero)
			}
			c.Aviso.Poner(v)
		} else {
			c.Aviso.Poner(nil)
		}
	}
}
