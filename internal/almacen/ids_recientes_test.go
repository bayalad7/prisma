package almacen

import (
	"bytes"
	"testing"
	"time"
)

func TestIDsRecientesReservar(t *testing.T) {
	a, r := nuevoAlmacen(t)
	hashA, hashB := bytes.Repeat([]byte{1}, 32), bytes.Repeat([]byte{2}, 32)
	nuevo := IDReciente{ClienteID: "cli_a", DocumentoID: "doc-1", HashContenido: hashA, TrabajoID: "trb_1"}

	res, existente, err := a.IDsRecientes.Reservar(ctx, nuevo)
	exigirSinError(t, err)
	if res != ReservaNueva || existente != nil {
		t.Fatalf("primera reserva: %d %+v", res, existente)
	}

	// Mismo id y mismo contenido: duplicado, devuelve el trabajo original.
	r.avanzar(time.Minute)
	repetido := nuevo
	repetido.TrabajoID = "trb_2"
	res, existente, err = a.IDsRecientes.Reservar(ctx, repetido)
	exigirSinError(t, err)
	if res != ReservaDuplicada || existente.TrabajoID != "trb_1" {
		t.Fatalf("reintento: %d %+v", res, existente)
	}

	// Mismo id con otro contenido: ID_REUTILIZADO.
	otro := nuevo
	otro.HashContenido = hashB
	res, _, err = a.IDsRecientes.Reservar(ctx, otro)
	exigirSinError(t, err)
	if res != ReservaIDReutilizado {
		t.Fatalf("contenido distinto: %d", res)
	}

	// El mismo id de otro cliente es independiente.
	deOtroCliente := otro
	deOtroCliente.ClienteID = "cli_b"
	res, _, err = a.IDsRecientes.Reservar(ctx, deOtroCliente)
	exigirSinError(t, err)
	if res != ReservaNueva {
		t.Fatalf("otro cliente: %d", res)
	}

	// Pasada la ventana, el id vuelve a estar libre.
	r.avanzar(VentanaIdempotencia)
	res, _, err = a.IDsRecientes.Reservar(ctx, otro)
	exigirSinError(t, err)
	if res != ReservaNueva {
		t.Fatalf("tras la ventana: %d", res)
	}

	_, _, err = a.IDsRecientes.Reservar(ctx, IDReciente{ClienteID: "cli_a", DocumentoID: "doc-2", HashContenido: []byte{1}, TrabajoID: "t"})
	exigirError(t, err, ErrInvalido)
}

func TestIDsRecientesPurgar(t *testing.T) {
	a, r := nuevoAlmacen(t)
	hash := bytes.Repeat([]byte{1}, 32)
	for i, doc := range []string{"doc-1", "doc-2", "doc-3"} {
		_, _, err := a.IDsRecientes.Reservar(ctx, IDReciente{ClienteID: "cli_a", DocumentoID: doc, HashContenido: hash,
			TrabajoID: "trb", Expira: r.t.Add(time.Duration(i+1) * time.Minute)})
		exigirSinError(t, err)
	}
	r.avanzar(2 * time.Minute)
	n, err := a.IDsRecientes.Purgar(ctx)
	exigirSinError(t, err)
	if n != 2 {
		t.Fatalf("purgados = %d", n)
	}
}

func TestIDsRecientesBuscarYLiberar(t *testing.T) {
	a, r := nuevoAlmacen(t)
	hash := bytes.Repeat([]byte{1}, 32)
	_, err := a.IDsRecientes.Buscar(ctx, "cli_a", "doc-1")
	exigirError(t, err, ErrNoEncontrado)
	_, _, err = a.IDsRecientes.Reservar(ctx, IDReciente{ClienteID: "cli_a", DocumentoID: "doc-1", HashContenido: hash, TrabajoID: "trb_1"})
	exigirSinError(t, err)
	e, err := a.IDsRecientes.Buscar(ctx, "cli_a", "doc-1")
	exigirSinError(t, err)
	if e.TrabajoID != "trb_1" || !bytes.Equal(e.HashContenido, hash) {
		t.Fatalf("encontrado: %+v", e)
	}

	// Liberar con otro trabajo no hace nada; con el suyo, deja el id libre.
	exigirSinError(t, a.IDsRecientes.Liberar(ctx, "cli_a", "doc-1", "trb_otro"))
	_, err = a.IDsRecientes.Buscar(ctx, "cli_a", "doc-1")
	exigirSinError(t, err)
	exigirSinError(t, a.IDsRecientes.Liberar(ctx, "cli_a", "doc-1", "trb_1"))
	_, err = a.IDsRecientes.Buscar(ctx, "cli_a", "doc-1")
	exigirError(t, err, ErrNoEncontrado)

	// Un registro vencido no se encuentra.
	_, _, err = a.IDsRecientes.Reservar(ctx, IDReciente{ClienteID: "cli_a", DocumentoID: "doc-2", HashContenido: hash, TrabajoID: "trb_2"})
	exigirSinError(t, err)
	r.avanzar(VentanaIdempotencia)
	_, err = a.IDsRecientes.Buscar(ctx, "cli_a", "doc-2")
	exigirError(t, err, ErrNoEncontrado)
}
