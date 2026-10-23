package docs

import (
	"regexp"
	"strings"
	"testing"
)

var (
	reAristaFlujo = regexp.MustCompile(`^\s*(.+?)\s*(?:--\s*"?([^">]*?)"?\s*)?-->\s*(.+?)\s*$`)
	reNodoFlujo   = regexp.MustCompile(`^(\w+)(?:([\[{])"([^"]*)"[\]}])?(?::::\w+)?$`)
	reEnlaceFlujo = regexp.MustCompile("(?m)^- `(\\w+)`: \\[([^\\]]+)\\]\\(([^)]+)\\)")
)

// TestArbolDeDiagnostico: el panel arma «Mi impresora no imprime» desde el diagrama 23 (manual.js,
// flujo). El árbol debe estar completo: un inicio, cada pregunta con al menos dos respuestas
// nombradas, cada final con su solución en la lista «Soluciones en el manual», y una rama que
// sugiera probar con la impresora virtual.
func TestArbolDeDiagnostico(t *testing.T) {
	texto := leer(t, "docs/diagramas/23-diagnostico-problemas.md")
	_, bloque, _ := strings.Cut(texto, "```mermaid\n")
	bloque, _, _ = strings.Cut(bloque, "\n```")

	pregunta := map[string]bool{}
	etiqueta := map[string]string{}
	respuestas := map[string][]string{}
	llegan := map[string]bool{}
	inicio := ""
	for _, linea := range strings.Split(bloque, "\n") {
		m := reAristaFlujo.FindStringSubmatch(linea)
		if m == nil {
			continue
		}
		var ids [2]string
		for k, lado := range []string{m[1], m[3]} {
			n := reNodoFlujo.FindStringSubmatch(lado)
			if n == nil {
				t.Fatalf("no se entiende el nodo %q en %q", lado, linea)
			}
			ids[k] = n[1]
			if n[3] != "" {
				etiqueta[n[1]], pregunta[n[1]] = n[3], n[2] == "{"
			}
		}
		if inicio == "" {
			inicio = ids[0]
		}
		respuestas[ids[0]] = append(respuestas[ids[0]], m[2])
		llegan[ids[1]] = true
	}
	if inicio == "" || llegan[inicio] || len(respuestas[inicio]) != 1 {
		t.Fatalf("el árbol debe empezar en un nodo con una sola salida: %q", inicio)
	}
	enlaces := map[string]string{}
	for _, m := range reEnlaceFlujo.FindAllStringSubmatch(texto, -1) {
		enlaces[m[1]] = m[3]
		if etiqueta[m[1]] == "" {
			t.Errorf("la solución %q no es un nodo del diagrama", m[1])
		}
	}
	for id := range etiqueta {
		switch {
		case id == inicio:
		case pregunta[id]:
			if len(respuestas[id]) < 2 {
				t.Errorf("la pregunta %q necesita al menos dos respuestas", id)
			}
			for _, r := range respuestas[id] {
				if strings.TrimSpace(r) == "" {
					t.Errorf("la pregunta %q tiene una respuesta sin texto", id)
				}
			}
		case len(respuestas[id]) > 0:
			t.Errorf("%q tiene respuestas, pero no es pregunta: escríbelo con {…}", id)
		case enlaces[id] == "":
			t.Errorf("el final %q («%s») no tiene su solución en «Soluciones en el manual»", id, etiqueta[id])
		}
	}
	virtual := false
	for id, e := range etiqueta {
		if pregunta[id] && strings.Contains(e, "virtual") && strings.Contains(enlaces[id], "probar-sin-impresora") {
			virtual = true
		}
	}
	if !virtual {
		t.Error("una pregunta debe sugerir probar con la impresora virtual, con su enlace a probar-sin-impresora.md")
	}
}
