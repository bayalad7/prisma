package almacen

import (
	"reflect"
	"testing"
)

func TestClientesCrearYObtener(t *testing.T) {
	a, _ := nuevoAlmacen(t)
	c := clienteDePrueba("cli_tienda01")
	c.LlaveHash = "$argon2id$v=19$m=8,t=1,p=1$b3RyYQ$b3Rybw"
	c.OperacionesSensibles = []string{OperacionAbrirCajon}
	exigirSinError(t, a.Clientes.Crear(ctx, c))

	leido, err := a.Clientes.Obtener(ctx, "cli_tienda01")
	exigirSinError(t, err)
	if !reflect.DeepEqual(leido, c) {
		t.Fatalf("leído distinto del creado:\n%+v\n%+v", leido, c)
	}
	if !leido.PermiteOperacion(OperacionAbrirCajon) || leido.PermiteOperacion(OperacionRaw) {
		t.Error("operaciones sensibles mal leídas")
	}
	porPrefijo, err := a.Clientes.ObtenerPorPrefijo(ctx, "pk_tienda01")
	exigirSinError(t, err)
	if porPrefijo.ID != c.ID {
		t.Error("ObtenerPorPrefijo devolvió otro cliente")
	}
	_, err = a.Clientes.Obtener(ctx, "cli_noexiste")
	exigirError(t, err, ErrNoEncontrado)
}

func TestClientesRestriccionesPorNivel(t *testing.T) {
	a, _ := nuevoAlmacen(t)

	sinLlave := clienteDePrueba("cli_sinllave")
	sinLlave.LlaveHash, sinLlave.LlavePrefijo = "", ""
	exigirError(t, a.Clientes.Crear(ctx, sinLlave), ErrInvalido)

	firma := &Cliente{ID: "cli_firma001", Nombre: "Backend", Nivel: NivelFirma, Activo: true}
	exigirError(t, a.Clientes.Crear(ctx, firma), ErrInvalido)
	firma.LlavePublica = make([]byte, 32)
	exigirSinError(t, a.Clientes.Crear(ctx, firma))

	origen := &Cliente{ID: "cli_origen01", Nombre: "Web", Nivel: NivelOrigen, Activo: true}
	exigirSinError(t, a.Clientes.Crear(ctx, origen))

	exigirError(t, a.Clientes.Crear(ctx, &Cliente{ID: "sinprefijo", Nombre: "x", Nivel: 1}), ErrInvalido)
	exigirError(t, a.Clientes.Crear(ctx, &Cliente{ID: "cli_nivel4", Nombre: "x", Nivel: 4}), ErrInvalido)

	malHash := clienteDePrueba("cli_malhash")
	malHash.LlaveHash = "sha256:abc"
	exigirError(t, a.Clientes.Crear(ctx, malHash), ErrInvalido)

	repetido := clienteDePrueba("cli_otro0001")
	repetido.LlavePrefijo = "pk_firma001"
	exigirSinError(t, a.Clientes.Crear(ctx, repetido))
	otro := clienteDePrueba("cli_otro0002")
	otro.LlavePrefijo = "pk_firma001"
	exigirError(t, a.Clientes.Crear(ctx, otro), ErrDuplicado)
}

func TestClientesActualizarYEliminar(t *testing.T) {
	a, _ := nuevoAlmacen(t)
	c := clienteDePrueba("cli_tienda01")
	exigirSinError(t, a.Clientes.Crear(ctx, c))
	exigirSinError(t, a.Clientes.AgregarOrigen(ctx, c.ID, "https://tienda.example"))

	c.Nivel, c.LlavePublica = NivelFirma, make([]byte, 32)
	c.LlaveHash, c.LlavePrefijo, c.Activo, c.ExigirLlave = "", "", false, false
	c.OperacionesSensibles = []string{OperacionRaw, OperacionAbrirCajon}
	exigirSinError(t, a.Clientes.Actualizar(ctx, c))
	leido, _ := a.Clientes.Obtener(ctx, c.ID)
	if leido.Nivel != NivelFirma || leido.Activo || leido.LlavePrefijo != "" || len(leido.OperacionesSensibles) != 2 {
		t.Fatalf("actualización no aplicada: %+v", leido)
	}

	exigirSinError(t, a.Clientes.Eliminar(ctx, c.ID))
	exigirError(t, a.Clientes.Eliminar(ctx, c.ID), ErrNoEncontrado)
	_, err := a.Clientes.ObtenerPorOrigen(ctx, "https://tienda.example")
	exigirError(t, err, ErrNoEncontrado)
	var n int
	exigirSinError(t, a.db.QueryRowContext(ctx, `SELECT count(*) FROM clientes_origenes`).Scan(&n))
	if n != 0 {
		t.Error("los orígenes del cliente debían borrarse en cascada")
	}
}

func TestClientesOrigenes(t *testing.T) {
	a, _ := nuevoAlmacen(t)
	uno, dos := clienteDePrueba("cli_uno00001"), clienteDePrueba("cli_dos00001")
	exigirSinError(t, a.Clientes.Crear(ctx, uno))
	exigirSinError(t, a.Clientes.Crear(ctx, dos))

	exigirSinError(t, a.Clientes.AgregarOrigen(ctx, uno.ID, "HTTPS://MiApp.com:443/"))
	exigirSinError(t, a.Clientes.AgregarOrigen(ctx, uno.ID, "http://localhost:5173"))
	exigirError(t, a.Clientes.AgregarOrigen(ctx, dos.ID, "https://miapp.com"), ErrDuplicado)
	exigirError(t, a.Clientes.AgregarOrigen(ctx, dos.ID, "https://*.miapp.com"), ErrOrigenInvalido)
	exigirError(t, a.Clientes.AgregarOrigen(ctx, "cli_noexiste", "https://otra.com"), ErrNoEncontrado)

	origenes, err := a.Clientes.Origenes(ctx, uno.ID)
	exigirSinError(t, err)
	if !reflect.DeepEqual(origenes, []string{"http://localhost:5173", "https://miapp.com"}) {
		t.Fatalf("orígenes = %v", origenes)
	}
	c, err := a.Clientes.ObtenerPorOrigen(ctx, "https://miapp.com/")
	exigirSinError(t, err)
	if c.ID != uno.ID {
		t.Error("ObtenerPorOrigen devolvió otro cliente")
	}

	exigirSinError(t, a.Clientes.QuitarOrigen(ctx, uno.ID, "https://miapp.com"))
	exigirError(t, a.Clientes.QuitarOrigen(ctx, uno.ID, "https://miapp.com"), ErrNoEncontrado)
	exigirSinError(t, a.Clientes.AgregarOrigen(ctx, dos.ID, "https://miapp.com"))
}

func TestClientesImpresorasPermitidas(t *testing.T) {
	a, _ := nuevoAlmacen(t)
	for _, alias := range []string{"caja", "cocina", "barra"} {
		exigirSinError(t, a.Impresoras.Crear(ctx, impresoraDePrueba(alias)))
	}
	c := clienteDePrueba("cli_tienda01")
	exigirSinError(t, a.Clientes.Crear(ctx, c))

	puede := func(alias string) bool {
		t.Helper()
		ok, err := a.Clientes.PuedeImprimirEn(ctx, c.ID, alias)
		exigirSinError(t, err)
		return ok
	}
	if !puede("cocina") || !puede("cualquiera") {
		t.Fatal("con todas_las_impresoras debe poder imprimir en cualquiera")
	}

	exigirSinError(t, a.Clientes.EstablecerImpresoras(ctx, c.ID, false, []string{"caja", "cocina", "caja"}))
	permitidas, err := a.Clientes.ImpresorasPermitidas(ctx, c.ID)
	exigirSinError(t, err)
	if !reflect.DeepEqual(permitidas, []string{"caja", "cocina"}) {
		t.Fatalf("permitidas = %v", permitidas)
	}
	if !puede("caja") || puede("barra") {
		t.Fatal("permisos explícitos mal aplicados")
	}

	// Un alias inexistente revierte todo el cambio.
	exigirError(t, a.Clientes.EstablecerImpresoras(ctx, c.ID, false, []string{"barra", "no-existe"}), ErrNoEncontrado)
	if permitidas, _ := a.Clientes.ImpresorasPermitidas(ctx, c.ID); len(permitidas) != 2 || puede("barra") {
		t.Fatalf("el cambio fallido debía revertirse: %v", permitidas)
	}

	// Borrar las impresoras permitidas NO convierte al cliente en "todas".
	exigirSinError(t, a.Impresoras.Eliminar(ctx, "caja"))
	exigirSinError(t, a.Impresoras.Eliminar(ctx, "cocina"))
	if puede("barra") || puede("cocina") {
		t.Fatal("tras borrar sus impresoras el cliente no debe poder imprimir en ninguna")
	}
	leido, _ := a.Clientes.Obtener(ctx, c.ID)
	if leido.TodasLasImpresoras {
		t.Fatal("todas_las_impresoras no debe activarse solo")
	}

	exigirSinError(t, a.Clientes.EstablecerImpresoras(ctx, c.ID, true, nil))
	if !puede("barra") {
		t.Fatal("con todas = true debe poder imprimir en cualquiera")
	}
	exigirError(t, a.Clientes.EstablecerImpresoras(ctx, "cli_noexiste", true, nil), ErrNoEncontrado)
	_, err = a.Clientes.PuedeImprimirEn(ctx, "cli_noexiste", "barra")
	exigirError(t, err, ErrNoEncontrado)
}

func TestClientesExigirLlave(t *testing.T) {
	a, _ := nuevoAlmacen(t)
	c := &Cliente{ID: "cli_firma002", Nombre: "Backend", Nivel: NivelFirma, LlavePublica: make([]byte, 32), ExigirLlave: true, Activo: true}
	exigirError(t, a.Clientes.Crear(ctx, c), ErrInvalido)
	c.LlaveHash, c.LlavePrefijo = hashPrueba, "pk_firma002"
	exigirSinError(t, a.Clientes.Crear(ctx, c))
	leido, _ := a.Clientes.Obtener(ctx, c.ID)
	if !leido.ExigirLlave || leido.LlaveHash != hashPrueba {
		t.Fatalf("leído: %+v", leido)
	}
	// Prefijo sin hash (o al revés) no es válido.
	solo := clienteDePrueba("cli_solopref")
	solo.LlaveHash = ""
	exigirError(t, a.Clientes.Crear(ctx, solo), ErrInvalido)
}
