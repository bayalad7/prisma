package admin

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestGaleriaDeEstilosSoloEnModoDesarrollo(t *testing.T) {
	b := nuevoBanco(t)
	n := b.nuevoNavegador("127.0.0.1")
	n.exigir("GET", "/admin/estilos", nil, 404)
	n.exigir("GET", "/admin/estilos.js", nil, 404)
	n.exigir("GET", "/admin/estilos.html", nil, 404)

	op := b.adm.op
	op.Config.Dev.ModoDesarrollo = true
	b.adm = Nuevo(op)
	b.srv.MontarPanel(b.adm.Handler())
	b.h = b.srv.Handler()
	for ruta, tipo := range map[string]string{"/admin/estilos": "text/html", "/admin/estilos.js": "text/javascript"} {
		r := n.exigir("GET", ruta, nil, 200)
		if !strings.HasPrefix(r.Header().Get("Content-Type"), tipo) || r.Header().Get("Content-Security-Policy") != cspPanel {
			t.Errorf("%s: cabeceras %v", ruta, r.Header())
		}
	}
	n.exigir("GET", "/admin/estilos.html", nil, 404)
}

func TestArchivosDelSistemaDeDiseno(t *testing.T) {
	b := nuevoBanco(t)
	n := b.nuevoNavegador("127.0.0.1")
	for ruta, tipo := range map[string]string{"/admin/textos.js": "text/javascript", "/admin/sistema.js": "text/javascript", "/admin/iconos.svg": "image/svg+xml"} {
		r := n.exigir("GET", ruta, nil, 200)
		if !strings.HasPrefix(r.Header().Get("Content-Type"), tipo) {
			t.Errorf("%s: Content-Type %q", ruta, r.Header().Get("Content-Type"))
		}
	}
}

// tokensCSS devuelve las variables de color de cada bloque de tokens de panel.css.
func tokensCSS(t *testing.T) (claro, oscuroAuto, oscuroManual map[string]string) {
	t.Helper()
	datos, err := fs.ReadFile(archivosWeb, "web/panel.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(datos)
	bloque := func(inicio string) map[string]string {
		_, resto, ok := strings.Cut(css, inicio)
		if !ok {
			t.Fatalf("panel.css no tiene el bloque %q", inicio)
		}
		cuerpo, _, _ := strings.Cut(resto, "}")
		vars := map[string]string{}
		for _, decl := range strings.Split(cuerpo, ";") {
			nombre, valor, ok := strings.Cut(strings.TrimSpace(decl), ":")
			if ok && strings.HasPrefix(nombre, "--") {
				vars[nombre] = strings.TrimSpace(valor)
			}
		}
		return vars
	}
	return bloque(":root {"), bloque(`:root:not([data-theme="light"]) {`), bloque(`:root[data-theme="dark"] {`)
}

func TestTemaOscuroIgual(t *testing.T) {
	_, auto, manual := tokensCSS(t)
	if len(auto) == 0 || fmt.Sprint(auto) != fmt.Sprint(manual) {
		t.Errorf("los bloques del tema oscuro difieren:\nautomático: %v\nmanual:     %v", auto, manual)
	}
}

var reHex = regexp.MustCompile(`^#([0-9a-fA-F]{6})$`)

func luminancia(t *testing.T, hex string) float64 {
	t.Helper()
	m := reHex.FindStringSubmatch(hex)
	if m == nil {
		t.Fatalf("color %q: se esperaba #rrggbb", hex)
	}
	v, _ := strconv.ParseUint(m[1], 16, 32)
	canal := func(c uint64) float64 {
		s := float64(c) / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*canal(v>>16&0xff) + 0.7152*canal(v>>8&0xff) + 0.0722*canal(v&0xff)
}

func contraste(t *testing.T, a, b string) float64 {
	la, lb := luminancia(t, a), luminancia(t, b)
	return (math.Max(la, lb) + 0.05) / (math.Min(la, lb) + 0.05)
}

// TestContrasteDeLosTokens: WCAG 2.1 AA. Texto 4.5:1 (1.4.3) y bordes de controles e iconos 3:1 (1.4.11).
func TestContrasteDeLosTokens(t *testing.T) {
	texto := [][2]string{
		{"tinta", "fondo"}, {"tinta", "superficie"}, {"tinta", "superficie-2"}, {"tinta", "superficie-3"},
		{"tenue", "fondo"}, {"tenue", "superficie"}, {"tenue", "superficie-2"},
		{"acento", "fondo"}, {"acento", "superficie"}, {"acento-fuerte", "acento-suave"}, {"sobre-acento", "acento"},
		{"sobre-error", "error"}, {"superficie", "aviso"}, {"papel-tinta", "papel"},
	}
	for _, e := range []string{"ok", "aviso", "error", "info", "desconocido", "virtual"} {
		texto = append(texto, [2]string{e, e + "-suave"}, [2]string{e, "superficie"}, [2]string{"tinta", e + "-suave"})
	}
	controles := [][2]string{{"borde-control", "superficie"}, {"borde-control", "fondo"}, {"borde-control", "superficie-2"}, {"superficie", "ok"}}

	claro, oscuro, _ := tokensCSS(t)
	for nombre, tema := range map[string]map[string]string{"claro": claro, "oscuro": oscuro} {
		valor := func(token string) string {
			if v, ok := tema["--"+token]; ok {
				return v
			}
			return claro["--"+token] // el tema oscuro hereda lo que no redefine
		}
		revisar := func(pares [][2]string, minimo float64) {
			for _, p := range pares {
				if c := contraste(t, valor(p[0]), valor(p[1])); c < minimo {
					t.Errorf("tema %s: --%s sobre --%s tiene contraste %.2f:1 (mínimo %.1f:1)", nombre, p[0], p[1], c, minimo)
				}
			}
		}
		revisar(texto, 4.5)
		revisar(controles, 3)
	}
}

// TestSpriteDeIconos: el sprite solo tiene formas básicas (nada que ejecute ni cargue recursos) y
// contiene todos los iconos que usa el panel.
func TestSpriteDeIconos(t *testing.T) {
	datos, err := fs.ReadFile(archivosWeb, "web/iconos.svg")
	if err != nil {
		t.Fatal(err)
	}
	elementos := map[string]bool{"svg": true, "symbol": true, "path": true, "circle": true, "rect": true}
	atributos := map[string]bool{"xmlns": true, "id": true, "viewBox": true, "d": true, "cx": true, "cy": true, "r": true,
		"x": true, "y": true, "width": true, "height": true, "rx": true, "fill": true, "stroke-dasharray": true}
	ids := map[string]bool{}
	dec := xml.NewDecoder(strings.NewReader(string(datos)))
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		inicio, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if !elementos[inicio.Name.Local] {
			t.Errorf("elemento no permitido en el sprite: <%s>", inicio.Name.Local)
		}
		for _, a := range inicio.Attr {
			if !atributos[a.Name.Local] || a.Name.Space != "" && a.Name.Local != "xmlns" {
				t.Errorf("atributo no permitido en <%s>: %s", inicio.Name.Local, a.Name.Local)
			}
			if a.Name.Local == "fill" && a.Value != "currentColor" {
				t.Errorf("<%s>: fill=%q; los iconos usan currentColor", inicio.Name.Local, a.Value)
			}
			if a.Name.Local == "id" {
				ids[strings.TrimPrefix(a.Value, "i-")] = true
			}
		}
	}

	usados := map[string]bool{"auto": true, "sol": true, "luna": true} // los del selector de tema
	reUso := regexp.MustCompile(`icono(?:\(\s*|:\s*)'([a-z-]+)'`)
	reEstados := regexp.MustCompile(`ICONO_ESTADO = \{([^}]*)\}`)
	reValor := regexp.MustCompile(`'([a-z-]+)'`)
	for _, js := range []string{"web/sistema.js", "web/panel.js", "web/estilos.js"} {
		fuente, err := fs.ReadFile(archivosWeb, js)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range reUso.FindAllStringSubmatch(string(fuente), -1) {
			usados[m[1]] = true
		}
		if m := reEstados.FindStringSubmatch(string(fuente)); m != nil {
			for _, v := range reValor.FindAllStringSubmatch(m[1], -1) {
				usados[v[1]] = true
			}
		}
	}
	for nombre := range usados {
		if !ids[nombre] {
			t.Errorf("el panel usa el icono %q, que no está en iconos.svg", nombre)
		}
	}
}

// TestRutasDeLasPantallas: cada pantalla del panel tiene su ruta (sirve index.html), las demás
// responden 404 y la lista del servidor coincide con las secciones de panel.js.
func TestRutasDeLasPantallas(t *testing.T) {
	b := nuevoBanco(t)
	n := b.nuevoNavegador("127.0.0.1")
	indice := n.exigir("GET", "/admin/", nil, 200).Body.String()
	for _, v := range vistasPanel {
		r := n.exigir("GET", "/admin/"+v, nil, 200)
		if r.Body.String() != indice || r.Header().Get("Content-Security-Policy") != cspPanel {
			t.Errorf("/admin/%s debe servir el panel con su CSP", v)
		}
	}
	for _, ruta := range []string{"/admin/no-existe", "/admin/impresoras/caja", "/admin/clientes", "/admin/index.htm"} {
		n.exigir("GET", ruta, nil, 404)
	}

	fuente, err := fs.ReadFile(archivosWeb, "web/panel.js")
	if err != nil {
		t.Fatal(err)
	}
	var rutas []string
	for _, m := range regexp.MustCompile(`ruta: '([a-z]+)'`).FindAllStringSubmatch(string(fuente), -1) {
		rutas = append(rutas, m[1])
	}
	if fmt.Sprint(rutas) != fmt.Sprint(vistasPanel) {
		t.Errorf("las rutas de panel.js %v no coinciden con vistasPanel %v", rutas, vistasPanel)
	}
}
