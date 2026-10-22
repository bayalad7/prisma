package admin

import (
	"strings"
	"testing"
)

func TestManualDelPanel(t *testing.T) {
	b := nuevoBanco(t)
	sin := b.nuevoNavegador("127.0.0.1")
	sin.exigirError("GET", "/admin/manual/administracion/README.md", nil, 401, "SESION_REQUERIDA")
	s := b.configurado()

	r := s.exigir("GET", "/admin/api/manual", nil, 200)
	if !strings.Contains(r.Body.String(), `{"ruta":"administracion/probar-sin-impresora.md","titulo":"Probar sin impresora"}`) {
		t.Fatalf("el índice debe traer cada documento con su título: %.300s", r.Body.String())
	}
	if !strings.Contains(r.Body.String(), `"titulo":"Impresora virtual"`) {
		t.Fatal("el título de un diagrama con front-matter es su encabezado")
	}

	md := s.exigir("GET", "/admin/manual/administracion/impresoras.md", nil, 200)
	if md.Header().Get("Content-Type") != "text/markdown; charset=utf-8" || !strings.HasPrefix(md.Body.String(), "# Impresoras") {
		t.Fatalf("documento: %q %.40q", md.Header().Get("Content-Type"), md.Body.String())
	}
	if img := s.exigir("GET", "/admin/manual/administracion/img/probar-sin-impresora/tarjeta-virtual.webp", nil, 200); img.Header().Get("Content-Type") != "image/webp" {
		t.Fatalf("imagen: %v", img.Header())
	}
	// Solo lo del índice: ni otros Markdown de docs/, ni archivos fuera, ni rutas con «..».
	for _, ruta := range []string{"ux/ANALISIS-PANEL-V1.md", "manual.json", "doc.go", "../go.mod", "administracion/../../go.mod",
		"adr/README.md", "administracion/img/probar-sin-impresora/capturar.mjs"} {
		s.exigirError("GET", "/admin/manual/"+ruta, nil, 404, "NO_ENCONTRADO")
	}
	// Cada página del manual tiene su dirección en el panel.
	indice := s.exigir("GET", "/admin/", nil, 200).Body.String()
	for _, ruta := range []string{"administracion/impresoras.md", "administracion/README.md"} {
		if r := s.exigir("GET", "/admin/ayuda/"+ruta, nil, 200); r.Body.String() != indice {
			t.Fatalf("/admin/ayuda/%s debe servir el panel", ruta)
		}
	}
	s.exigir("GET", "/admin/ayuda/administracion/../go.mod", nil, 404)
}
