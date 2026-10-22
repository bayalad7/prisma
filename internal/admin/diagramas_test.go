package admin

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
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
	Alt             string   `json:"alt"`
	Hash            string   `json:"hash"`
	Claro           string   `json:"claro"`
	Oscuro          string   `json:"oscuro"`
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
