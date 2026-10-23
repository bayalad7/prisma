package admin

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/fs"
	"net/http"
	"regexp"
	"strings"
)

// Diagramas del panel: los SVG de docs/diagramas/ que genera scripts/docs/generar-diagramas (ADR
// 0018). Se guardan con gzip y se sirven así a los navegadores que lo aceptan (todos); el panel los
// muestra con <img>, así que su <style> interno funciona con la CSP del panel.

// archivoDiagrama: NN-nombre.claro.svg o NN-nombre.oscuro.svg.
var archivoDiagrama = regexp.MustCompile(`^[0-9]{2}-[a-z0-9-]+\.(claro|oscuro)\.svg$`)

// GET /admin/api/diagramas: el manifiesto (metadatos, texto alternativo y archivos de cada diagrama).
func (a *Admin) listarDiagramas(w http.ResponseWriter, r *http.Request) {
	datos, err := fs.ReadFile(archivosDiagramas, "ayuda/diagramas/manifiesto.json")
	if err != nil {
		a.error(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(datos)
}

// GET /admin/diagramas/{archivo}: un SVG; con ?descargar=1, como archivo.
func (a *Admin) servirDiagrama(w http.ResponseWriter, r *http.Request) {
	nombre := strings.TrimPrefix(r.URL.Path, "/admin/diagramas/")
	gz, err := fs.ReadFile(archivosDiagramas, "ayuda/diagramas/"+nombre+".gz")
	if !archivoDiagrama.MatchString(nombre) || err != nil || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
		a.error(w, r, rechazo(http.StatusNotFound, "NO_ENCONTRADO", "Ese diagrama no existe.", "Vuelve a la Ayuda."))
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Vary", "Accept-Encoding")
	if r.URL.Query().Get("descargar") == "1" {
		w.Header().Set("Content-Disposition", `attachment; filename="`+nombre+`"`)
	}
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(gz)
		return
	}
	lector, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		a.error(w, r, err)
		return
	}
	io.Copy(w, lector)
}
