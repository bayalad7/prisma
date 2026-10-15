package almacen

import (
	"fmt"
	"testing"
	"time"

	"github.com/bayalad7/prisma/internal/seguridad"
)

func TestSolicitudesRegistrar(t *testing.T) {
	a, r := nuevoAlmacen(t)
	s, err := a.Solicitudes.Registrar(ctx, "https://Desconocida.com/")
	exigirSinError(t, err)
	if s.Origen != "https://desconocida.com" || s.Intentos != 1 || s.Estado != SolicitudPendiente {
		t.Fatalf("solicitud inicial: %+v", s)
	}
	r.avanzar(time.Hour)
	s, err = a.Solicitudes.Registrar(ctx, "https://desconocida.com")
	exigirSinError(t, err)
	if s.Intentos != 2 || !s.UltimaVez.After(s.PrimeraVez) {
		t.Fatalf("segundo intento: %+v", s)
	}
	_, err = a.Solicitudes.Registrar(ctx, "no es un origen")
	exigirError(t, err, seguridad.ErrOrigenInvalido)
}

func TestSolicitudesRechazarYEliminar(t *testing.T) {
	a, _ := nuevoAlmacen(t)
	_, err := a.Solicitudes.Registrar(ctx, "https://mala.com")
	exigirSinError(t, err)
	_, err = a.Solicitudes.Registrar(ctx, "https://buena.com")
	exigirSinError(t, err)
	exigirSinError(t, a.Solicitudes.Rechazar(ctx, "https://mala.com"))

	s, err := a.Solicitudes.Registrar(ctx, "https://mala.com")
	exigirSinError(t, err)
	if s.Estado != SolicitudRechazada || s.Intentos != 2 {
		t.Fatalf("una solicitud rechazada debe seguir rechazada: %+v", s)
	}
	pendientes, err := a.Solicitudes.Listar(ctx, SolicitudPendiente)
	exigirSinError(t, err)
	if len(pendientes) != 1 || pendientes[0].Origen != "https://buena.com" {
		t.Fatalf("pendientes = %+v", pendientes)
	}
	todas, _ := a.Solicitudes.Listar(ctx, "")
	if len(todas) != 2 {
		t.Fatalf("todas = %+v", todas)
	}

	exigirSinError(t, a.Solicitudes.Eliminar(ctx, "https://mala.com"))
	exigirError(t, a.Solicitudes.Eliminar(ctx, "https://mala.com"), ErrNoEncontrado)
	exigirError(t, a.Solicitudes.Rechazar(ctx, "https://nunca.com"), ErrNoEncontrado)
}

func TestSolicitudesAprobar(t *testing.T) {
	a, _ := nuevoAlmacen(t)
	c := clienteDePrueba("cli_tienda01")
	exigirSinError(t, a.Clientes.Crear(ctx, c))
	_, err := a.Solicitudes.Registrar(ctx, "https://nueva.com")
	exigirSinError(t, err)

	// Aprobar para un cliente inexistente no debe perder la solicitud.
	exigirError(t, a.Solicitudes.Aprobar(ctx, "https://nueva.com", "cli_noexiste"), ErrNoEncontrado)
	_, err = a.Solicitudes.Obtener(ctx, "https://nueva.com")
	exigirSinError(t, err)

	exigirSinError(t, a.Solicitudes.Aprobar(ctx, "https://nueva.com", c.ID))
	_, err = a.Solicitudes.Obtener(ctx, "https://nueva.com")
	exigirError(t, err, ErrNoEncontrado)
	duenio, err := a.Clientes.ObtenerPorOrigen(ctx, "https://nueva.com")
	exigirSinError(t, err)
	if duenio.ID != c.ID {
		t.Fatal("el origen aprobado debe pertenecer al cliente")
	}
	exigirError(t, a.Solicitudes.Aprobar(ctx, "https://nueva.com", c.ID), ErrNoEncontrado)
}

func TestSolicitudesLimite(t *testing.T) {
	a, _ := nuevoAlmacen(t)
	for i := 0; i < MaxSolicitudes; i++ {
		_, err := a.Solicitudes.Registrar(ctx, fmt.Sprintf("https://sitio%d.com", i))
		exigirSinError(t, err)
	}
	_, err := a.Solicitudes.Registrar(ctx, "https://una-mas.com")
	exigirError(t, err, ErrLimiteSolicitudes)
	// Un origen ya registrado sigue contando sus intentos.
	s, err := a.Solicitudes.Registrar(ctx, "https://sitio0.com")
	exigirSinError(t, err)
	if s.Intentos != 2 {
		t.Fatalf("intentos = %d", s.Intentos)
	}
}
