package almacen

import (
	"errors"
	"fmt"
	"testing"
	"time"
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
	exigirError(t, err, ErrOrigenInvalido)
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

	// Deshacer el rechazo la devuelve a pendiente; una pendiente no se «restaura».
	exigirSinError(t, a.Solicitudes.Restaurar(ctx, "https://mala.com"))
	if s, _ := a.Solicitudes.Obtener(ctx, "https://mala.com"); s.Estado != SolicitudPendiente || s.Intentos != 2 {
		t.Fatalf("restaurada: %+v", s)
	}
	exigirError(t, a.Solicitudes.Restaurar(ctx, "https://mala.com"), ErrNoEncontrado)

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
	a, r := nuevoAlmacen(t)
	for i := 0; i < MaxSolicitudes; i++ {
		_, err := a.Solicitudes.Registrar(ctx, fmt.Sprintf("https://sitio%d.com", i))
		exigirSinError(t, err)
		r.avanzar(time.Second)
	}
	// Un origen ya registrado sigue contando sus intentos (y deja de ser el más antiguo).
	s, err := a.Solicitudes.Registrar(ctx, "https://sitio0.com")
	exigirSinError(t, err)
	if s.Intentos != 2 {
		t.Fatalf("intentos = %d", s.Intentos)
	}
	// Lleno: el origen nuevo reemplaza a la pendiente con la actividad más antigua (sitio1).
	_, err = a.Solicitudes.Registrar(ctx, "https://una-mas.com")
	exigirSinError(t, err)
	if _, err := a.Solicitudes.Obtener(ctx, "https://sitio1.com"); !errors.Is(err, ErrNoEncontrado) {
		t.Fatalf("sitio1 debía salir: %v", err)
	}
	if n, _ := a.Solicitudes.Contar(ctx); n != MaxSolicitudes {
		t.Fatalf("total = %d", n)
	}
	// Si todas son rechazadas, no se reemplaza ninguna.
	todas, _ := a.Solicitudes.Listar(ctx, "")
	for _, s := range todas {
		exigirSinError(t, a.Solicitudes.Rechazar(ctx, s.Origen))
	}
	_, err = a.Solicitudes.Registrar(ctx, "https://otra.com")
	exigirError(t, err, ErrLimiteSolicitudes)
}

func TestSolicitudesPurgar(t *testing.T) {
	a, r := nuevoAlmacen(t)
	for _, o := range []string{"https://vieja.com", "https://rechazada-vieja.com", "https://rechazada-reciente.com"} {
		_, err := a.Solicitudes.Registrar(ctx, o)
		exigirSinError(t, err)
	}
	exigirSinError(t, a.Solicitudes.Rechazar(ctx, "https://rechazada-vieja.com"))
	exigirSinError(t, a.Solicitudes.Rechazar(ctx, "https://rechazada-reciente.com"))
	r.avanzar(60 * 24 * time.Hour)
	_, err := a.Solicitudes.Registrar(ctx, "https://rechazada-reciente.com") // sigue intentando
	exigirSinError(t, err)
	_, err = a.Solicitudes.Registrar(ctx, "https://nueva.com")
	exigirSinError(t, err)
	r.avanzar(40 * 24 * time.Hour) // vieja: 100 días; rechazada-vieja: 100; rechazada-reciente: 40; nueva: 40
	n, err := a.Solicitudes.Purgar(ctx)
	exigirSinError(t, err)
	quedan, _ := a.Solicitudes.Listar(ctx, "")
	if n != 3 || len(quedan) != 1 || quedan[0].Origen != "https://rechazada-reciente.com" {
		t.Fatalf("purgadas %d, quedan %+v", n, quedan)
	}
}
