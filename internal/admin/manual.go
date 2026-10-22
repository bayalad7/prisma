package admin

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/bayalad7/prisma/docs"
)

// Manual del panel: los Markdown de docs/ (docs.Manual), servidos solo con sesión y solo los que
// están en el índice, con sus imágenes. El panel los dibuja con su propio renderizador (manual.js),
// que crea nodos con textContent: el puente no genera HTML (ADR 0017).

type grupoManual struct {
	ID         string            `json:"id"`
	Titulo     string            `json:"titulo"`
	Documentos []documentoManual `json:"documentos"`
}

type documentoManual struct {
	Ruta   string `json:"ruta"`
	Titulo string `json:"titulo"`
}

// imagenManual: las capturas del manual, en docs/<carpeta>/img/<tema>/<nombre>.webp.
var imagenManual = regexp.MustCompile(`^[a-z]+/img/[a-z0-9-]+/[a-z0-9-]+\.webp$`)

var indiceManual = sync.OnceValues(func() ([]grupoManual, error) {
	datos, err := fs.ReadFile(docs.Manual, "manual.json")
	if err != nil {
		return nil, err
	}
	var indice struct {
		Grupos []struct {
			ID         string   `json:"id"`
			Titulo     string   `json:"titulo"`
			Documentos []string `json:"documentos"`
		} `json:"grupos"`
	}
	if err := json.Unmarshal(datos, &indice); err != nil {
		return nil, err
	}
	grupos := make([]grupoManual, 0, len(indice.Grupos))
	for _, g := range indice.Grupos {
		grupo := grupoManual{ID: g.ID, Titulo: g.Titulo}
		for _, ruta := range g.Documentos {
			texto, err := fs.ReadFile(docs.Manual, ruta)
			if err != nil {
				return nil, err
			}
			grupo.Documentos = append(grupo.Documentos, documentoManual{Ruta: ruta, Titulo: tituloMarkdown(string(texto), ruta)})
		}
		grupos = append(grupos, grupo)
	}
	return grupos, nil
})

// tituloMarkdown es el primer encabezado «# …», después del front-matter si lo hay.
func tituloMarkdown(texto, ruta string) string {
	if strings.HasPrefix(texto, "---\n") {
		if _, resto, ok := strings.Cut(texto[4:], "\n---\n"); ok {
			texto = resto
		}
	}
	for _, linea := range strings.Split(texto, "\n") {
		if t, ok := strings.CutPrefix(linea, "# "); ok {
			return strings.Trim(strings.TrimSpace(t), "`")
		}
	}
	return path.Base(ruta)
}

// GET /admin/api/manual: los grupos del manual con el título de cada documento.
func (a *Admin) listarManual(w http.ResponseWriter, r *http.Request) {
	grupos, err := indiceManual()
	if err != nil {
		a.error(w, r, err)
		return
	}
	responder(w, map[string]any{"grupos": grupos})
}

// GET /admin/manual/{ruta}: un Markdown del índice o una de sus imágenes.
func (a *Admin) servirManual(w http.ResponseWriter, r *http.Request) {
	ruta := strings.TrimPrefix(r.URL.Path, "/admin/manual/")
	grupos, err := indiceManual()
	if err != nil {
		a.error(w, r, err)
		return
	}
	tipo := ""
	if imagenManual.MatchString(ruta) {
		tipo = "image/webp"
	} else {
		for _, g := range grupos {
			for _, d := range g.Documentos {
				if d.Ruta == ruta {
					tipo = "text/markdown; charset=utf-8"
				}
			}
		}
	}
	datos, err := fs.ReadFile(docs.Manual, ruta)
	if tipo == "" || err != nil || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
		a.error(w, r, rechazo(http.StatusNotFound, "NO_ENCONTRADO", "Esa página del manual no existe.", "Vuelve al índice de la ayuda."))
		return
	}
	w.Header().Set("Content-Type", tipo)
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(datos)
}
