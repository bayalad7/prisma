package almacen

import (
	"reflect"
	"testing"
	"time"
)

func trabajoDePrueba(id, cliente, alias string) *Trabajo {
	return &Trabajo{ID: id, DocumentoID: "doc-" + id, ClienteID: cliente, ImpresoraAlias: alias, Referencia: "PED-" + id}
}

func TestTrabajosCicloDeVida(t *testing.T) {
	a, r := nuevoAlmacen(t)
	tr := trabajoDePrueba("trb_1", "cli_a", "caja")
	exigirSinError(t, a.Trabajos.Crear(ctx, tr))
	exigirError(t, a.Trabajos.Crear(ctx, tr), ErrDuplicado)
	if tr.Estado != TrabajoEncolado {
		t.Fatalf("estado inicial %q", tr.Estado)
	}

	exigirSinError(t, a.Trabajos.MarcarImprimiendo(ctx, "trb_1"))
	exigirError(t, a.Trabajos.MarcarImprimiendo(ctx, "trb_1"), ErrNoEncontrado)

	r.avanzar(2 * time.Second)
	exigirError(t, a.Trabajos.Finalizar(ctx, "trb_1", TrabajoImprimiendo, "", nil, 0), ErrInvalido)
	exigirSinError(t, a.Trabajos.Finalizar(ctx, "trb_1", TrabajoCompletado, "", []string{"SIN_CUCHILLA"}, 1500*time.Millisecond))
	exigirError(t, a.Trabajos.Finalizar(ctx, "trb_1", TrabajoFallido, "X", nil, 0), ErrNoEncontrado)

	leido, err := a.Trabajos.Obtener(ctx, "trb_1")
	exigirSinError(t, err)
	if leido.Estado != TrabajoCompletado || leido.Duracion != 1500*time.Millisecond ||
		!reflect.DeepEqual(leido.Advertencias, []string{"SIN_CUCHILLA"}) || leido.Referencia != "PED-trb_1" ||
		leido.Finalizado.Sub(leido.Creado) != 2*time.Second {
		t.Fatalf("trabajo finalizado: %+v", leido)
	}

	exigirSinError(t, a.Trabajos.Crear(ctx, trabajoDePrueba("trb_2", "cli_a", "caja")))
	exigirSinError(t, a.Trabajos.Finalizar(ctx, "trb_2", TrabajoFallido, "IMPRESORA_SIN_PAPEL", nil, 0))
	fallido, _ := a.Trabajos.Obtener(ctx, "trb_2")
	if fallido.CodigoError != "IMPRESORA_SIN_PAPEL" || fallido.Advertencias == nil {
		t.Fatalf("trabajo fallido: %+v", fallido)
	}
	_, err = a.Trabajos.Obtener(ctx, "trb_x")
	exigirError(t, err, ErrNoEncontrado)
}

func TestTrabajosMarcarInterrumpidos(t *testing.T) {
	a, _ := nuevoAlmacen(t)
	exigirSinError(t, a.Trabajos.Crear(ctx, trabajoDePrueba("trb_1", "cli_a", "caja")))
	exigirSinError(t, a.Trabajos.Crear(ctx, trabajoDePrueba("trb_2", "cli_a", "caja")))
	exigirSinError(t, a.Trabajos.Crear(ctx, trabajoDePrueba("trb_3", "cli_a", "caja")))
	exigirSinError(t, a.Trabajos.MarcarImprimiendo(ctx, "trb_2"))
	exigirSinError(t, a.Trabajos.Finalizar(ctx, "trb_3", TrabajoCompletado, "", nil, 0))

	n, err := a.Trabajos.MarcarInterrumpidos(ctx)
	exigirSinError(t, err)
	if n != 2 {
		t.Fatalf("interrumpidos = %d", n)
	}
	tr, _ := a.Trabajos.Obtener(ctx, "trb_2")
	if tr.Estado != TrabajoFallido || tr.CodigoError != CodigoPuenteReiniciado || tr.Finalizado.IsZero() {
		t.Fatalf("trabajo interrumpido: %+v", tr)
	}
}

func TestTrabajosListarYPurgar(t *testing.T) {
	a, r := nuevoAlmacen(t)
	datos := []struct{ id, cliente, alias string }{
		{"trb_1", "cli_a", "caja"}, {"trb_2", "cli_b", "caja"}, {"trb_3", "cli_a", "cocina"}, {"trb_4", "cli_a", "caja"},
	}
	for _, d := range datos {
		exigirSinError(t, a.Trabajos.Crear(ctx, trabajoDePrueba(d.id, d.cliente, d.alias)))
		r.avanzar(24 * time.Hour)
	}
	exigirSinError(t, a.Trabajos.Finalizar(ctx, "trb_1", TrabajoCompletado, "", nil, 0))
	exigirSinError(t, a.Trabajos.Finalizar(ctx, "trb_2", TrabajoFallido, "X", nil, 0))

	ids := func(f FiltroTrabajos) []string {
		t.Helper()
		lista, err := a.Trabajos.Listar(ctx, f)
		exigirSinError(t, err)
		var r []string
		for _, tr := range lista {
			r = append(r, tr.ID)
		}
		return r
	}
	if got := ids(FiltroTrabajos{}); !reflect.DeepEqual(got, []string{"trb_4", "trb_3", "trb_2", "trb_1"}) {
		t.Errorf("todos = %v", got)
	}
	if got := ids(FiltroTrabajos{ClienteID: "cli_a", ImpresoraAlias: "caja"}); !reflect.DeepEqual(got, []string{"trb_4", "trb_1"}) {
		t.Errorf("filtrados = %v", got)
	}
	if got := ids(FiltroTrabajos{Estado: TrabajoEncolado, Limite: 1}); !reflect.DeepEqual(got, []string{"trb_4"}) {
		t.Errorf("encolados con límite = %v", got)
	}

	// Purga los terminados anteriores al tercer día; los pendientes se conservan aunque sean viejos.
	limite := time.Date(2026, 10, 16, 18, 0, 0, 0, time.UTC)
	exigirSinError(t, a.Trabajos.MarcarImprimiendo(ctx, "trb_3"))
	n, err := a.Trabajos.Purgar(ctx, limite)
	exigirSinError(t, err)
	if n != 2 {
		t.Fatalf("purgados = %d", n)
	}
	if got := ids(FiltroTrabajos{}); !reflect.DeepEqual(got, []string{"trb_4", "trb_3"}) {
		t.Errorf("tras purgar = %v", got)
	}
}
