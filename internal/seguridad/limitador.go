package seguridad

import (
	"math"
	"sync"
	"time"
)

// Limitador aplica un límite de peticiones por minuto a cada clave (una IP o un cliente) con una
// cubeta de fichas: admite ráfagas de hasta porMinuto peticiones y se recarga de forma continua.
type Limitador struct {
	mu        sync.Mutex
	porMinuto int
	cubetas   map[string]*cubeta
	limpieza  time.Time
	Ahora     func() time.Time
}

type cubeta struct {
	fichas float64
	ultima time.Time
}

// NuevoLimitador crea un limitador; porMinuto = 0 lo desactiva.
func NuevoLimitador(porMinuto int) *Limitador {
	return &Limitador{porMinuto: porMinuto, cubetas: map[string]*cubeta{}, Ahora: time.Now}
}

// Permitir consume una ficha de la clave. Si no hay, devuelve cuánto esperar para reintentar.
func (l *Limitador) Permitir(clave string) (bool, time.Duration) {
	if l == nil || l.porMinuto <= 0 {
		return true, 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	ahora := l.Ahora()
	capacidad := float64(l.porMinuto)
	porSegundo := capacidad / 60

	if ahora.Sub(l.limpieza) > time.Minute {
		// Una cubeta que lleva un minuto sin uso ya está llena: equivale a no tenerla.
		for k, c := range l.cubetas {
			if ahora.Sub(c.ultima) > time.Minute {
				delete(l.cubetas, k)
			}
		}
		l.limpieza = ahora
	}

	c, ok := l.cubetas[clave]
	if !ok {
		c = &cubeta{fichas: capacidad, ultima: ahora}
		l.cubetas[clave] = c
	}
	c.fichas = math.Min(capacidad, c.fichas+ahora.Sub(c.ultima).Seconds()*porSegundo)
	c.ultima = ahora
	if c.fichas >= 1 {
		c.fichas--
		return true, 0
	}
	espera := time.Duration((1 - c.fichas) / porSegundo * float64(time.Second))
	return false, espera
}
