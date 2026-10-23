package docs

import (
	"encoding/json"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// documentosDelManual lee manual.json desde lo embebido (lo mismo que sirve el panel).
func documentosDelManual(t *testing.T) []string {
	t.Helper()
	datos, err := fs.ReadFile(Manual, "manual.json")
	if err != nil {
		t.Fatal(err)
	}
	var indice struct {
		Grupos []struct {
			ID, Titulo string
			Documentos []string
		}
	}
	if err := json.Unmarshal(datos, &indice); err != nil {
		t.Fatal(err)
	}
	var docs []string
	for _, g := range indice.Grupos {
		if g.ID == "" || g.Titulo == "" || len(g.Documentos) == 0 {
			t.Errorf("grupo incompleto: %+v", g)
		}
		docs = append(docs, g.Documentos...)
	}
	return docs
}

// TestIndiceDelManual: cada documento del índice está embebido una sola vez, y las guías del
// panel y todos los diagramas están en el índice (un archivo nuevo no queda fuera de la Ayuda).
func TestIndiceDelManual(t *testing.T) {
	docs := documentosDelManual(t)
	vistos := map[string]bool{}
	for _, d := range docs {
		if vistos[d] {
			t.Errorf("%s está dos veces en manual.json", d)
		}
		vistos[d] = true
		if _, err := fs.Stat(Manual, d); err != nil {
			t.Errorf("%s está en manual.json pero no en el binario: %v", d, err)
		}
	}
	for _, patron := range []string{"administracion/*.md", "diagramas/[0-9][0-9]-*.md"} {
		archivos, _ := filepath.Glob(filepath.Join(raiz, "docs", filepath.FromSlash(patron)))
		for _, a := range archivos {
			ruta := strings.TrimPrefix(filepath.ToSlash(a), raiz+"/docs/")
			if !vistos[ruta] {
				t.Errorf("%s no está en docs/manual.json: no aparece en la Ayuda del panel", ruta)
			}
		}
	}
}

var (
	reHTML           = regexp.MustCompile(`<[a-zA-Z/!][^>]*>`)
	reNotaAlPie      = regexp.MustCompile(`\[\^`)
	reDefinicion     = regexp.MustCompile(`^\s*\[[^\]]+\]:\s`)
	reReferencia     = regexp.MustCompile(`\]\[[^\]]*\]`)
	reEntidad        = regexp.MustCompile(`&[a-zA-Z]+;|&#\d+;`)
	reGuionBajo      = regexp.MustCompile(`(^|[\s(])_[^_\s][^_]*_($|[\s.,;:)])`)
	reTituloProfundo = regexp.MustCompile(`^#{5,}\s`)
	reCerca          = regexp.MustCompile("^\\s*```")
	reSeparadorTab   = regexp.MustCompile(`^\s*\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)+\|?\s*$`)
)

// TestSintaxisDelManual: el panel dibuja el manual con su propio renderizador (manual.js, ADR
// 0017), que admite el subconjunto de Markdown que usa docs/. Este prueba falla si un documento
// del índice usa algo que no dibujaría bien.
func TestSintaxisDelManual(t *testing.T) {
	for _, ruta := range documentosDelManual(t) {
		datos, _ := fs.ReadFile(Manual, ruta)
		texto := string(datos)
		if _, resto, ok := frontMatter(texto); ok {
			texto = resto
		}
		enCodigo := false
		anterior := ""
		for n, linea := range strings.Split(texto, "\n") {
			if reCerca.MatchString(linea) {
				enCodigo = !enCodigo
				anterior = linea
				continue
			}
			if enCodigo {
				continue
			}
			sinCodigo := reCodigoLinea.ReplaceAllString(linea, "")
			mal := func(que string) { t.Errorf("%s:%d: %s: %q", ruta, n+1, que, strings.TrimSpace(linea)) }
			switch {
			case reHTML.MatchString(sinCodigo):
				mal("HTML crudo (el panel lo mostraría como texto)")
			case reNotaAlPie.MatchString(sinCodigo), reDefinicion.MatchString(sinCodigo), reReferencia.MatchString(sinCodigo):
				mal("notas al pie o enlaces por referencia")
			case reEntidad.MatchString(sinCodigo):
				mal("entidad HTML")
			case reGuionBajo.MatchString(sinCodigo):
				mal("cursiva con _ (usa *)")
			case reTituloProfundo.MatchString(linea):
				mal("título de nivel 5 o 6")
			case strings.TrimSpace(linea) != "" && strings.Trim(strings.TrimSpace(linea), "=") == "":
				mal("título subrayado con ===")
			case strings.TrimSpace(linea) == "---" && strings.TrimSpace(anterior) != "":
				mal("--- justo después de un párrafo (GitHub lo toma como título; deja una línea vacía)")
			case reSeparadorTab.MatchString(linea) && !strings.HasPrefix(strings.TrimSpace(anterior), "|"):
				mal("tabla sin | al principio de la fila de títulos")
			}
			anterior = linea
		}
		if enCodigo {
			t.Errorf("%s: un bloque de código no se cierra", ruta)
		}
	}
}

var reInclusion = regexp.MustCompile(`\{\{diagrama:([^}]*)\}\}`)

// TestInclusionDeDiagramas: {{diagrama:id}} pone el visor de un diagrama en una página del manual.
// Debe ir solo en su renglón (como un párrafo) y nombrar un diagrama de docs/diagramas/.
func TestInclusionDeDiagramas(t *testing.T) {
	usos := 0
	for _, ruta := range documentosDelManual(t) {
		datos, _ := fs.ReadFile(Manual, ruta)
		for n, linea := range strings.Split(string(datos), "\n") {
			// Un ejemplo entre comillas invertidas (`{{diagrama:…}}`) es texto, no una inclusión.
			for _, m := range reInclusion.FindAllStringSubmatch(reCodigoLinea.ReplaceAllString(linea, ""), -1) {
				usos++
				if strings.TrimSpace(linea) != m[0] {
					t.Errorf("%s:%d: %s debe ir solo en su renglón", ruta, n+1, m[0])
				}
				if _, err := fs.Stat(Manual, "diagramas/"+m[1]+".md"); err != nil {
					t.Errorf("%s:%d: el diagrama %q no existe en docs/diagramas/", ruta, n+1, m[1])
				}
			}
		}
	}
	if usos == 0 {
		t.Error("ningún documento del manual incluye un diagrama: ¿cambió la sintaxis?")
	}
}
