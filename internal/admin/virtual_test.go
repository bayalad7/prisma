package admin

import (
	"bytes"
	"image/png"
	"strings"
	"testing"

	"github.com/bayalad7/prisma/internal/conector"
)

func perfilVirtual(cambios map[string]any) map[string]any {
	p := map[string]any{"alias": "prueba", "nombre": "Para probar", "tipo": "virtual", "destino": "", "ancho_mm": 80, "columnas": 48,
		"columnas_fuente_b": 64, "ancho_px": 576, "pagina_codigos": "CP858", "comando_pagina_codigos": 19, "soporta_corte": true,
		"soporta_qr_nativo": true, "soporta_imagen": true, "soporta_codigo_barras_nativo": true, "activa": true}
	for k, v := range cambios {
		p[k] = v
	}
	return p
}

func TestImpresoraVirtualDesdeElPanel(t *testing.T) {
	b := nuevoBanco(t)
	s := b.configurado()
	conector.RollosVirtuales.Vaciar("prueba")
	t.Cleanup(func() { conector.RollosVirtuales.Vaciar("prueba") })

	s.exigirError("POST", "/admin/api/impresoras", perfilVirtual(map[string]any{"simulacion": "incendio"}), 400, "DATOS_INVALIDOS")
	s.exigirError("POST", "/admin/api/impresoras", perfilVirtual(map[string]any{"simulacion": "lenta", "simulacion_retraso_ms": 60000}), 400, "DATOS_INVALIDOS")
	r := s.exigir("POST", "/admin/api/impresoras", perfilVirtual(nil), 200)
	if r.datos["virtual"] != true || r.datos["destino"] != "memoria" || r.datos["simulacion"] != "normal" || r.datos["tickets_en_memoria"] != float64(20) {
		t.Fatalf("alta de la impresora virtual: %v", r.datos)
	}

	// El ticket de diagnóstico llega al rollo y trae su imagen.
	r = s.exigir("POST", "/admin/api/impresoras/prueba/diagnostico", nil, 200)
	trabajo, _ := r.datos["trabajo_id"].(string)
	if img, _ := r.datos["imagen"].(string); !strings.HasPrefix(img, "data:image/png;base64,") {
		t.Fatalf("el diagnóstico debe traer la imagen del ticket: %.60q", img)
	}
	rollo := s.exigir("GET", "/admin/api/impresoras/prueba/rollo", nil, 200)
	if !strings.Contains(rollo.Body.String(), trabajo) || !strings.Contains(rollo.Body.String(), `"referencia":"DIAGNOSTICO"`) ||
		!strings.Contains(rollo.Body.String(), `"cliente_nombre":"Panel"`) {
		t.Fatalf("rollo: %s", rollo.Body.String())
	}
	if tr := s.exigir("GET", "/admin/api/trabajos", nil, 200); !strings.Contains(tr.Body.String(), `"virtual":true`) {
		t.Fatalf("el historial debe marcar el trabajo como virtual: %s", tr.Body.String())
	}

	base := "/admin/api/impresoras/prueba/rollo/" + trabajo
	img := s.exigir("GET", base+"/imagen", nil, 200)
	if img.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("imagen: %v", img.Header())
	}
	if _, err := png.Decode(bytes.NewReader(img.Body.Bytes())); err != nil {
		t.Fatal(err)
	}
	ticket, _ := conector.RollosVirtuales.Ticket("prueba", trabajo)
	if bin := s.exigir("GET", base+"/bin", nil, 200); !bytes.Equal(bin.Body.Bytes(), ticket.Datos) {
		t.Fatal("el .bin debe ser exactamente lo que recibió la impresora")
	}
	if tz := s.exigir("GET", base+"/traza", nil, 200); !strings.Contains(tz.datos["traza"].(string), "inicializar impresora") {
		t.Fatalf("traza: %v", tz.datos)
	}
	s.exigirError("GET", "/admin/api/impresoras/prueba/rollo/trb_no_existe/imagen", nil, 404, "TICKET_NO_ENCONTRADO")

	// Simular «fuera de línea»: el diagnóstico falla y no se guarda nada nuevo.
	s.exigir("PUT", "/admin/api/impresoras/prueba", perfilVirtual(map[string]any{"simulacion": "fuera_de_linea"}), 200)
	s.exigirError("POST", "/admin/api/impresoras/prueba/probar", nil, 503, "IMPRESORA_INALCANZABLE")
	if n := len(conector.RollosVirtuales.Listar("prueba")); n != 1 {
		t.Fatalf("el rollo debe seguir con 1 ticket, tiene %d", n)
	}

	s.exigir("DELETE", "/admin/api/impresoras/prueba/rollo", nil, 200)
	if n := len(conector.RollosVirtuales.Listar("prueba")); n != 0 {
		t.Fatalf("el rollo debe quedar vacío, tiene %d", n)
	}
	s.exigir("POST", "/admin/api/impresoras", map[string]any{"alias": "caja", "tipo": "archivo", "destino": t.TempDir(), "ancho_mm": 80,
		"columnas": 48, "columnas_fuente_b": 64, "ancho_px": 576, "pagina_codigos": "CP858", "comando_pagina_codigos": 19, "activa": true}, 200)
	s.exigirError("GET", "/admin/api/impresoras/caja/rollo", nil, 409, "IMPRESORA_NO_VIRTUAL")
}
