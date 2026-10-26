package servicio

import (
	"errors"
	"testing"
	"time"

	"github.com/kardianos/service"
)

// servicioLento simula el gestor de servicios de Windows: después de desinstalar, el servicio sigue
// «marcado para eliminación» unas consultas más, y crearlo mientras tanto falla.
type servicioLento struct {
	service.Service
	instalado, corriendo bool
	borrandose           int // consultas que faltan para que desaparezca
	intentos             int
}

func (s *servicioLento) Status() (service.Status, error) {
	if s.borrandose > 0 {
		s.borrandose--
		return service.StatusStopped, nil
	}
	if !s.instalado {
		return service.StatusUnknown, service.ErrNotInstalled
	}
	if s.corriendo {
		return service.StatusRunning, nil
	}
	return service.StatusStopped, nil
}
func (s *servicioLento) Stop() error { s.corriendo = false; return nil }
func (s *servicioLento) Uninstall() error {
	s.instalado, s.borrandose = false, 4
	return nil
}
func (s *servicioLento) Install() error {
	s.intentos++
	if s.borrandose > 0 || s.intentos < 3 {
		return errors.New("service PRISMA already exists")
	}
	s.instalado = true
	return nil
}

// TestReinstalarEsperaAQueSeBorre: reinstalar un servicio en marcha lo detiene, lo quita, espera a
// que el sistema termine de borrarlo y reintenta crearlo, en lugar de dejar el equipo sin servicio
// (lo que pasó al reinstalar 0.3.0 en Windows). Si el borrado no termina a tiempo, informa el error.
func TestReinstalarEsperaAQueSeBorre(t *testing.T) {
	s := &servicioLento{instalado: true, corriendo: true}
	if err := reinstalar(s, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	if !s.instalado || s.corriendo || s.intentos < 3 {
		t.Fatalf("debía quedar instalado tras reintentar: %+v", s)
	}

	atascado := &servicioLento{instalado: true, intentos: -1000}
	if err := reinstalar(atascado, 600*time.Millisecond); err == nil || atascado.instalado {
		t.Fatal("si no se puede crear a tiempo, debe informar el error")
	}
}
