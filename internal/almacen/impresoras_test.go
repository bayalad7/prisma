package almacen

import (
	"testing"
	"time"
)

func TestImpresorasCrearYObtener(t *testing.T) {
	a, _ := nuevoAlmacen(t)
	imp := impresoraDePrueba("caja")
	imp.Tipo, imp.Destino, imp.Baudios = ConexionBluetooth, "COM5", 9600
	exigirSinError(t, a.Impresoras.Crear(ctx, imp))
	if imp.ID == 0 || imp.Creada.IsZero() {
		t.Fatal("Crear debe completar ID y Creada")
	}
	leida, err := a.Impresoras.Obtener(ctx, "caja")
	exigirSinError(t, err)
	if *leida != *imp {
		t.Fatalf("leída distinta de la creada:\n%+v\n%+v", leida, imp)
	}
	_, err = a.Impresoras.Obtener(ctx, "no-existe")
	exigirError(t, err, ErrNoEncontrado)
}

func TestImpresorasRestricciones(t *testing.T) {
	a, _ := nuevoAlmacen(t)
	exigirSinError(t, a.Impresoras.Crear(ctx, impresoraDePrueba("caja")))
	exigirError(t, a.Impresoras.Crear(ctx, impresoraDePrueba("caja")), ErrDuplicado)

	casos := map[string]func(*Impresora){
		"tipo":     func(i *Impresora) { i.Tipo = "wifi" },
		"columnas": func(i *Impresora) { i.Columnas = 0 },
		"comando":  func(i *Impresora) { i.ComandoPaginaCodigos = 256 },
		"destino":  func(i *Impresora) { i.Destino = "" },
		"alias":    func(i *Impresora) { i.Alias = "" },
	}
	for nombre, cambiar := range casos {
		imp := impresoraDePrueba("otra-" + nombre)
		cambiar(imp)
		if err := a.Impresoras.Crear(ctx, imp); err == nil {
			t.Errorf("%s: se esperaba error", nombre)
		}
	}
}

func TestImpresorasListarActualizarEliminar(t *testing.T) {
	a, r := nuevoAlmacen(t)
	for _, alias := range []string{"cocina", "barra", "caja"} {
		exigirSinError(t, a.Impresoras.Crear(ctx, impresoraDePrueba(alias)))
	}
	lista, err := a.Impresoras.Listar(ctx)
	exigirSinError(t, err)
	if len(lista) != 3 || lista[0].Alias != "barra" || lista[2].Alias != "cocina" {
		t.Fatalf("orden incorrecto: %v", lista)
	}

	r.avanzar(time.Minute)
	imp, _ := a.Impresoras.Obtener(ctx, "caja")
	imp.Alias, imp.SoportaCajon, imp.Columnas = "caja-1", true, 32
	exigirSinError(t, a.Impresoras.Actualizar(ctx, imp))
	leida, err := a.Impresoras.Obtener(ctx, "caja-1")
	exigirSinError(t, err)
	if !leida.SoportaCajon || leida.Columnas != 32 || !leida.Actualizada.After(leida.Creada) {
		t.Fatalf("actualización no aplicada: %+v", leida)
	}
	imp.Alias = "barra"
	exigirError(t, a.Impresoras.Actualizar(ctx, imp), ErrDuplicado)
	exigirError(t, a.Impresoras.Actualizar(ctx, &Impresora{ID: 999, Alias: "x"}), ErrNoEncontrado)

	exigirSinError(t, a.Impresoras.Eliminar(ctx, "caja-1"))
	exigirError(t, a.Impresoras.Eliminar(ctx, "caja-1"), ErrNoEncontrado)
}
