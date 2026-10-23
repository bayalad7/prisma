package admin

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/bayalad7/prisma/docs"
)

type diagramaManifiesto struct {
	ID              string   `json:"id"`
	Fuente          string   `json:"fuente"`
	Titulo          string   `json:"titulo"`
	Descripcion     string   `json:"descripcion"`
	Audiencia       []string `json:"audiencia"`
	Pantallas       []string `json:"pantallas"`
	SeccionesManual []string `json:"secciones_manual"`
	Enlaces         []struct {
		Nodo    string `json:"nodo"`
		Destino string `json:"destino"`
		Texto   string `json:"texto"`
		Caja    struct {
			X     float64 `json:"x"`
			Y     float64 `json:"y"`
			Ancho float64 `json:"ancho"`
			Alto  float64 `json:"alto"`
		} `json:"caja"`
	} `json:"enlaces"`
	Alt    string `json:"alt"`
	Hash   string `json:"hash"`
	Claro  string `json:"claro"`
	Oscuro string `json:"oscuro"`
}

func leerManifiesto(t *testing.T) []diagramaManifiesto {
	t.Helper()
	datos, err := fs.ReadFile(archivosDiagramas, "ayuda/diagramas/manifiesto.json")
	if err != nil {
		t.Fatal(err)
	}
	var m struct{ Diagramas []diagramaManifiesto }
	if err := json.Unmarshal(datos, &m); err != nil {
		t.Fatal(err)
	}
	return m.Diagramas
}

// frontMatterDiagrama lee los campos del bloque YAML del principio (listas en una línea: [a, b]).
func frontMatterDiagrama(texto string) map[string][]string {
	campos := map[string][]string{}
	meta, _, ok := strings.Cut(strings.TrimPrefix(texto, "---\n"), "\n---\n")
	if !ok || !strings.HasPrefix(texto, "---\n") {
		return campos
	}
	for _, linea := range strings.Split(meta, "\n") {
		clave, valor, ok := strings.Cut(linea, ":")
		if !ok {
			continue
		}
		valor = strings.TrimSpace(valor)
		if strings.HasPrefix(valor, "[") {
			var l []string
			for _, x := range strings.Split(strings.Trim(valor, "[]"), ",") {
				if x = strings.TrimSpace(x); x != "" {
					l = append(l, x)
				}
			}
			campos[clave] = l
		} else {
			campos[clave] = []string{strings.Trim(valor, `"`)}
		}
	}
	return campos
}

// TestDiagramasVigentes: cada diagrama de docs/diagramas/ está dibujado y su manifiesto corresponde
// al bloque mermaid (por su SHA-256) y al front-matter actuales.
func TestDiagramasVigentes(t *testing.T) {
	const generar = "Diagramas desactualizados: ejecuta scripts/docs/generar-diagramas"
	porID := map[string]diagramaManifiesto{}
	for _, d := range leerManifiesto(t) {
		porID[d.ID] = d
	}
	archivos, err := fs.Glob(docs.Manual, "diagramas/[0-9][0-9]-*.md")
	if err != nil || len(archivos) < 20 {
		t.Fatalf("diagramas: %d, %v", len(archivos), err)
	}
	for _, a := range archivos {
		id := strings.TrimSuffix(path.Base(a), ".md")
		datos, _ := fs.ReadFile(docs.Manual, a)
		texto := strings.ReplaceAll(string(datos), "\r\n", "\n")
		_, bloque, ok := strings.Cut(texto, "```mermaid\n")
		if !ok {
			continue
		}
		bloque, _, _ = strings.Cut(bloque, "\n```")
		d, ok := porID[id]
		if !ok {
			t.Errorf("%s: no está en el manifiesto. %s", id, generar)
			continue
		}
		delete(porID, id)
		suma := sha256.Sum256([]byte(bloque))
		if d.Hash != hex.EncodeToString(suma[:]) {
			t.Errorf("%s: el bloque mermaid cambió. %s", id, generar)
		}
		fm := frontMatterDiagrama(texto)
		if strings.Join(fm["titulo"], "") != d.Titulo || strings.Join(fm["descripcion"], "") != d.Descripcion ||
			!slices.Equal(fm["audiencia"], d.Audiencia) || !slices.Equal(fm["pantallas"], d.Pantallas) ||
			!slices.Equal(fm["secciones_manual"], d.SeccionesManual) {
			t.Errorf("%s: el front-matter cambió. %s", id, generar)
		}
		// Los nodos con enlace: los mismos del front-matter, con su caja dentro de la imagen.
		var enlaces []string
		for _, e := range d.Enlaces {
			enlaces = append(enlaces, e.Nodo+"="+e.Destino)
			c := e.Caja
			if e.Texto == "" || c.Ancho <= 0 || c.Alto <= 0 || c.X < 0 || c.Y < 0 || c.X+c.Ancho > 1.001 || c.Y+c.Alto > 1.001 {
				t.Errorf("%s: el enlace del nodo %s no tiene texto o su caja está fuera de la imagen: %+v", id, e.Nodo, e)
			}
		}
		if !slices.Equal(fm["enlaces"], enlaces) {
			t.Errorf("%s: los enlaces cambiaron (%v en el front-matter, %v en el manifiesto). %s", id, fm["enlaces"], enlaces, generar)
		}
		for _, f := range []string{d.Claro, d.Oscuro} {
			if _, err := fs.Stat(archivosDiagramas, "ayuda/diagramas/"+f); err != nil {
				t.Errorf("%s: falta %s. %s", id, f, generar)
			}
		}
	}
	for id := range porID {
		t.Errorf("%s está en el manifiesto, pero ya no en docs/diagramas. %s", id, generar)
	}
}

var (
	reSVGScript   = regexp.MustCompile(`(?i)<script|<foreignObject|<iframe|<object|<embed|javascript:`)
	reSVGEventos  = regexp.MustCompile(`(?i)\son[a-z]+\s*=`)
	reSVGEnlace   = regexp.MustCompile(`(?i)\s(xlink:)?href\s*=\s*["']([^"']*)["']`)
	reSVGExternos = regexp.MustCompile(`(?i)url\(\s*['"]?(https?:|//)|@import`)
)

// TestDiagramasSaneados: los SVG no traen código, manejadores de eventos, foreignObject ni recursos de
// afuera (el panel los muestra como imágenes, y además se sanean al generarlos).
func TestDiagramasSaneados(t *testing.T) {
	for _, d := range leerManifiesto(t) {
		for _, f := range []string{d.Claro, d.Oscuro} {
			gz, err := fs.ReadFile(archivosDiagramas, "ayuda/diagramas/"+f)
			if err != nil {
				t.Fatal(err)
			}
			r, err := gzip.NewReader(bytes.NewReader(gz))
			if err != nil {
				t.Fatalf("%s: %v", f, err)
			}
			svg, _ := io.ReadAll(r)
			if !bytes.HasPrefix(bytes.TrimSpace(svg), []byte("<svg")) {
				t.Errorf("%s no es un SVG", f)
			}
			if m := reSVGScript.Find(svg); m != nil {
				t.Errorf("%s contiene %q", f, m)
			}
			if m := reSVGEventos.Find(svg); m != nil {
				t.Errorf("%s contiene un manejador de eventos %q", f, m)
			}
			for _, m := range reSVGEnlace.FindAllSubmatch(svg, -1) {
				if !bytes.HasPrefix(m[2], []byte("#")) {
					t.Errorf("%s enlaza afuera: %q", f, m[2])
				}
			}
			if m := reSVGExternos.Find(svg); m != nil {
				t.Errorf("%s carga algo de afuera: %q", f, m)
			}
		}
	}
}

func TestServirDiagramas(t *testing.T) {
	b := nuevoBanco(t)
	sin := b.nuevoNavegador("127.0.0.1")
	sin.exigirError("GET", "/admin/diagramas/07-orden-autorizacion.claro.svg", nil, 401, "SESION_REQUERIDA")
	s := b.configurado()

	r := s.exigir("GET", "/admin/api/diagramas", nil, 200)
	if !strings.Contains(r.Body.String(), `"id": "07-orden-autorizacion"`) {
		t.Fatalf("manifiesto: %.200s", r.Body.String())
	}
	// Con gzip, tal cual; sin gzip, descomprimido.
	conGzip := s.pedir("GET", "/admin/diagramas/07-orden-autorizacion.oscuro.svg", nil, func(r *http.Request) { r.Header.Set("Accept-Encoding", "gzip, br") })
	if conGzip.Code != 200 || conGzip.Header().Get("Content-Encoding") != "gzip" || conGzip.Header().Get("Content-Type") != "image/svg+xml" ||
		conGzip.Header().Get("Content-Security-Policy") == "" {
		t.Fatalf("con gzip: %d %v", conGzip.Code, conGzip.Header())
	}
	plano := s.exigir("GET", "/admin/diagramas/07-orden-autorizacion.oscuro.svg?descargar=1", nil, 200)
	if plano.Header().Get("Content-Encoding") != "" || !bytes.HasPrefix(plano.Body.Bytes(), []byte("<svg")) ||
		!strings.Contains(plano.Header().Get("Content-Disposition"), "07-orden-autorizacion.oscuro.svg") {
		t.Fatalf("sin gzip: %v %.40q", plano.Header(), plano.Body.String())
	}
	for _, ruta := range []string{"07-orden-autorizacion.svg", "manifiesto.json", "../web/panel.js", "07-orden-autorizacion.claro.svg.gz", "99-no-existe.claro.svg"} {
		s.exigirError("GET", "/admin/diagramas/"+ruta, nil, 404, "NO_ENCONTRADO")
	}
}

// TestDiagramasDeLosErrores: cada diagrama que el panel ofrece para un código de error existe.
func TestDiagramasDeLosErrores(t *testing.T) {
	ids := map[string]bool{}
	for _, d := range leerManifiesto(t) {
		ids[d.ID] = true
	}
	errores := destinosDeTextos(t, "diagramasDeError")
	if len(errores) < 5 {
		t.Fatalf("diagramasDeError: %v", errores)
	}
	for codigo, id := range errores {
		if !ids[id] {
			t.Errorf("ayuda.diagramasDeError.%s lleva al diagrama %q, que no existe", codigo, id)
		}
	}
}
