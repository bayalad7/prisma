package almacen

import (
	"bytes"
	"testing"
	"time"
)

func TestRecursos(t *testing.T) {
	a, r := nuevoAlmacen(t)
	logo := &Recurso{Nombre: "logo-tienda", PNG: []byte("\x89PNG..."), AnchoPx: 200, AltoPx: 80}
	exigirSinError(t, a.Recursos.Guardar(ctx, logo))
	exigirSinError(t, a.Recursos.Guardar(ctx, &Recurso{Nombre: "firma", PNG: []byte{1}, AnchoPx: 1, AltoPx: 1}))

	leido, err := a.Recursos.Obtener(ctx, "logo-tienda")
	exigirSinError(t, err)
	if !bytes.Equal(leido.PNG, logo.PNG) || leido.AnchoPx != 200 || !leido.Creado.Equal(logo.Creado) {
		t.Fatalf("recurso leído: %+v", leido)
	}

	// Guardar con el mismo nombre reemplaza.
	r.avanzar(time.Hour)
	exigirSinError(t, a.Recursos.Guardar(ctx, &Recurso{Nombre: "logo-tienda", PNG: []byte("nuevo"), AnchoPx: 100, AltoPx: 40}))
	leido, _ = a.Recursos.Obtener(ctx, "logo-tienda")
	if string(leido.PNG) != "nuevo" || leido.AnchoPx != 100 {
		t.Fatalf("no se reemplazó: %+v", leido)
	}

	lista, err := a.Recursos.Listar(ctx)
	exigirSinError(t, err)
	if len(lista) != 2 || lista[0].Nombre != "firma" || lista[1].PNG != nil {
		t.Fatalf("listado: %+v", lista)
	}

	exigirError(t, a.Recursos.Guardar(ctx, &Recurso{Nombre: "grande", PNG: make([]byte, MaxBytesRecurso+1), AnchoPx: 1, AltoPx: 1}), ErrInvalido)
	exigirError(t, a.Recursos.Guardar(ctx, &Recurso{Nombre: "vacio", AnchoPx: 1, AltoPx: 1}), ErrInvalido)

	exigirSinError(t, a.Recursos.Eliminar(ctx, "firma"))
	exigirError(t, a.Recursos.Eliminar(ctx, "firma"), ErrNoEncontrado)
	_, err = a.Recursos.Obtener(ctx, "firma")
	exigirError(t, err, ErrNoEncontrado)
}
