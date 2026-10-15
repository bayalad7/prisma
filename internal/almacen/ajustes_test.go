package almacen

import "testing"

func TestAjustes(t *testing.T) {
	a, _ := nuevoAlmacen(t)
	if _, ok, err := a.Ajustes.Obtener(ctx, AjusteContrasenaAdmin); err != nil || ok {
		t.Fatalf("no debería existir: ok=%v err=%v", ok, err)
	}
	exigirSinError(t, a.Ajustes.Establecer(ctx, AjusteContrasenaAdmin, "hash-1"))
	exigirSinError(t, a.Ajustes.Establecer(ctx, AjusteContrasenaAdmin, "hash-2"))
	if v, ok, _ := a.Ajustes.Obtener(ctx, AjusteContrasenaAdmin); !ok || v != "hash-2" {
		t.Fatalf("valor = %q", v)
	}

	exigirSinError(t, a.Ajustes.EstablecerVarios(ctx, map[string]string{AjusteDebeCambiar: "1", "otro": "x"}))
	if v, _, _ := a.Ajustes.Obtener(ctx, AjusteDebeCambiar); v != "1" {
		t.Fatalf("debe_cambiar = %q", v)
	}

	exigirSinError(t, a.Ajustes.Eliminar(ctx, "otro"))
	if _, ok, _ := a.Ajustes.Obtener(ctx, "otro"); ok {
		t.Fatal("no se eliminó")
	}
}

func TestAjustesRetencionDias(t *testing.T) {
	a, _ := nuevoAlmacen(t)
	if d, err := a.Ajustes.RetencionDias(ctx); err != nil || d != RetencionDiasPorDefecto {
		t.Fatalf("por defecto: %d %v", d, err)
	}
	exigirSinError(t, a.Ajustes.Establecer(ctx, AjusteRetencionDias, "7"))
	if d, _ := a.Ajustes.RetencionDias(ctx); d != 7 {
		t.Fatalf("retención = %d", d)
	}
	exigirSinError(t, a.Ajustes.Establecer(ctx, AjusteRetencionDias, "0"))
	_, err := a.Ajustes.RetencionDias(ctx)
	exigirError(t, err, ErrInvalido)
}
