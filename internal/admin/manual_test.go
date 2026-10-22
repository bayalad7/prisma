package admin

import (
	"encoding/json"
	"io/fs"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/bayalad7/prisma/docs"
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
	for _, ruta := range []string{"administracion/impresoras.md", "administracion/README.md", "no-imprime"} {
		if r := s.exigir("GET", "/admin/ayuda/"+ruta, nil, 200); r.Body.String() != indice {
			t.Fatalf("/admin/ayuda/%s debe servir el panel", ruta)
		}
	}
	s.exigir("GET", "/admin/ayuda/administracion/../go.mod", nil, 404)
}

func TestBuscarEnElManual(t *testing.T) {
	b := nuevoBanco(t)
	s := b.configurado()
	buscar := func(q string) []resultadoManual {
		t.Helper()
		var r []resultadoManual
		if err := json.Unmarshal(s.exigir("GET", "/admin/api/manual/buscar?q="+url.QueryEscape(q), nil, 200).Body.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		return r
	}

	// Sin acentos ni la ñ, y por prefijo.
	r := buscar("olvide contrasena")
	if len(r) == 0 || r[0].Ruta != "administracion/problemas-comunes.md" || r[0].Ancla != "olvidé-la-contraseña-del-panel" {
		t.Fatalf("«olvide contrasena» debe encontrar primero la sección del panel (las guías del dueño ganan los empates): %+v", r)
	}
	// Los diagramas se encuentran por su descripción del front-matter.
	if r = buscar("mismos bytes papel"); !slices.ContainsFunc(r, func(x resultadoManual) bool { return x.Ruta == "diagramas/29-impresora-virtual.md" }) {
		t.Fatalf("la descripción del diagrama 29 debe encontrarse: %+v", r)
	}
	// El título del documento pesa más que el texto.
	if r = buscar("impresora virtual"); len(r) == 0 || !strings.Contains(r[0].Titulo+r[0].Seccion, "virtual") && !strings.Contains(r[0].Titulo+r[0].Seccion, "Virtual") {
		t.Fatalf("el primer resultado debe hablar de la impresora virtual: %+v", r)
	}
	for _, x := range r {
		if x.Extracto == "" {
			t.Fatalf("cada resultado lleva un extracto: %+v", x)
		}
	}
	if len(buscar("a")) != 0 || len(buscar("")) != 0 || len(buscar("palabraquenoexiste")) != 0 {
		t.Fatal("sin palabras útiles o sin coincidencias no hay resultados")
	}
	if len(buscar("impresora")) > 20 {
		t.Fatal("como mucho 20 resultados")
	}
}

// TestAnclasDeLaBusqueda: cada sección que encuentra la búsqueda apunta a un ancla que existe en su
// documento, calculada igual que en docs (y en manual.js, que lo comprueba el recorrido del panel).
func TestAnclasDeLaBusqueda(t *testing.T) {
	secciones, err := indiceBusqueda()
	if err != nil || len(secciones) < 200 {
		t.Fatalf("secciones: %d, %v", len(secciones), err)
	}
	vistas := map[string]bool{}
	for _, s := range secciones {
		if s.seccion == "" {
			continue
		}
		clave := s.ruta + "#" + s.ancla
		if vistas[clave] {
			t.Errorf("ancla repetida: %s", clave)
		}
		vistas[clave] = true
	}
}

// destinosDeTextos lee de textos.js un objeto { clave: 'ruta#ancla' } de la sección ayuda.
func destinosDeTextos(t *testing.T, objeto string) map[string]string {
	t.Helper()
	fuente, err := fs.ReadFile(archivosWeb, "web/textos.js")
	if err != nil {
		t.Fatal(err)
	}
	_, resto, ok := strings.Cut(string(fuente), "\n    "+objeto+": {\n")
	if !ok {
		t.Fatalf("textos.js no tiene ayuda.%s", objeto)
	}
	bloque, _, _ := strings.Cut(resto, "\n    },")
	destinos := map[string]string{}
	for _, m := range regexp.MustCompile(`(?m)^\s+(\w+): '([^']+)',`).FindAllStringSubmatch(bloque, -1) {
		destinos[m[1]] = m[2]
	}
	return destinos
}

// TestAyudaDeCadaPantalla: cada pantalla del panel tiene su sección del manual, cada código de
// error y advertencia documentado tiene su enlace, y cada destino (documento y ancla) existe.
func TestAyudaDeCadaPantalla(t *testing.T) {
	grupos, err := indiceManual()
	if err != nil {
		t.Fatal(err)
	}
	anclas := map[string]bool{}
	for _, g := range grupos {
		for _, d := range g.Documentos {
			anclas[d.Ruta] = true
		}
	}
	secciones, err := indiceBusqueda()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range secciones {
		if s.ancla != "" {
			anclas[s.ruta+"#"+s.ancla] = true
		}
	}
	revisar := func(que string, destinos map[string]string) {
		for clave, destino := range destinos {
			if !anclas[destino] {
				t.Errorf("ayuda.%s.%s lleva a %q, que no está en el manual", que, clave, destino)
			}
		}
	}

	pantallas := destinosDeTextos(t, "pantallas")
	revisar("pantallas", pantallas)
	fuente, _ := fs.ReadFile(archivosWeb, "web/panel.js")
	for _, m := range regexp.MustCompile(`\{ id: '([a-z]+)', ruta:`).FindAllStringSubmatch(string(fuente), -1) {
		if m[1] != "ayuda" && pantallas[m[1]] == "" {
			t.Errorf("la pantalla %q no tiene su sección en ayuda.pantallas", m[1])
		}
	}
	for _, sub := range []string{"rollo", "permisos"} {
		if pantallas[sub] == "" {
			t.Errorf("la subpantalla %q no tiene su sección en ayuda.pantallas", sub)
		}
	}

	errores := destinosDeTextos(t, "errores")
	revisar("errores", errores)
	codigos, _ := fs.ReadFile(docs.Manual, "referencia/codigos-de-error.md")
	n := 0
	for _, m := range regexp.MustCompile("(?m)^\\| `([A-Z_]+)`").FindAllStringSubmatch(string(codigos), -1) {
		n++
		if errores[m[1]] == "" {
			t.Errorf("el código %s no tiene enlace al manual en ayuda.errores", m[1])
		}
	}
	if n < 60 {
		t.Fatalf("códigos leídos de codigos-de-error.md: %d", n)
	}
	revisar("campos", destinosDeTextos(t, "campos"))
}
