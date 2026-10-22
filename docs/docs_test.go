package docs

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/bayalad7/prisma/internal/servidor"
)

const raiz = ".."

func leer(t *testing.T, ruta string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(raiz, ruta))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestOpenAPICoincideConLasRutas compara las rutas y métodos de openapi.yaml con los del puente.
func TestOpenAPICoincideConLasRutas(t *testing.T) {
	documentadas := map[string][]string{}
	ruta := ""
	reRuta := regexp.MustCompile(`^  (/\S+):\s*$`)
	reMetodo := regexp.MustCompile(`^    (get|post|put|patch|delete):\s*$`)
	enPaths := false
	sc := bufio.NewScanner(strings.NewReader(leer(t, "docs/referencia/openapi.yaml")))
	for sc.Scan() {
		linea := sc.Text()
		switch {
		case linea == "paths:":
			enPaths = true
		case enPaths && linea != "" && !strings.HasPrefix(linea, " "):
			enPaths = false
		case enPaths && reRuta.MatchString(linea):
			ruta = reRuta.FindStringSubmatch(linea)[1]
			documentadas[ruta] = nil
		case enPaths && ruta != "" && reMetodo.MatchString(linea):
			documentadas[ruta] = append(documentadas[ruta], strings.ToUpper(reMetodo.FindStringSubmatch(linea)[1]))
		}
	}
	for r := range documentadas {
		slices.Sort(documentadas[r])
	}
	reales := servidor.RutasPublicas()
	for r, metodos := range reales {
		if !slices.Equal(documentadas[r], metodos) {
			t.Errorf("%s: el puente acepta %v y openapi.yaml documenta %v", r, metodos, documentadas[r])
		}
	}
	for r := range documentadas {
		if _, ok := reales[r]; !ok {
			t.Errorf("openapi.yaml documenta %s, que el puente no tiene", r)
		}
	}
}

// noSonCodigos son literales en MAYÚSCULAS del código que no son códigos de error.
var noSonCodigos = map[string]string{
	"NO_COLOR": "variable de entorno estándar para desactivar el color de la terminal",
}

// TestCodigosDeErrorDocumentados exige que cada código del puente y del SDK aparezca en
// docs/referencia/codigos-de-error.md.
func TestCodigosDeErrorDocumentados(t *testing.T) {
	doc := leer(t, "docs/referencia/codigos-de-error.md")
	codigos := map[string]string{} // código → archivo donde aparece
	reGo := regexp.MustCompile(`"([A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+)"`)
	for _, dir := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(raiz, dir), func(ruta string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(ruta, ".go") || strings.HasSuffix(ruta, "_test.go") {
				return err
			}
			b, err := os.ReadFile(ruta)
			if err != nil {
				return err
			}
			for _, m := range reGo.FindAllStringSubmatch(string(b), -1) {
				codigos[m[1]] = ruta
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	sdk := leer(t, "sdk/js/src/prisma.mjs")
	for _, m := range regexp.MustCompile(`codigo: '([A-Z][A-Z0-9_]+)'`).FindAllStringSubmatch(sdk, -1) {
		codigos[m[1]] = "sdk/js/src/prisma.mjs"
	}
	for _, lista := range regexp.MustCompile(`\[('[A-Z]+(?:_[A-Z0-9]+)+'(?:, *'[A-Z]+(?:_[A-Z0-9]+)+')*)\]\)`).FindAllStringSubmatch(sdk, -1) {
		for _, c := range regexp.MustCompile(`'([A-Z0-9_]+)'`).FindAllStringSubmatch(lista[1], -1) {
			codigos[c[1]] = "sdk/js/src/prisma.mjs (causas_posibles)"
		}
	}
	if len(codigos) < 40 {
		t.Fatalf("solo se encontraron %d códigos: la búsqueda dejó de funcionar", len(codigos))
	}
	for c, archivo := range codigos {
		if _, ok := noSonCodigos[c]; ok {
			continue
		}
		if !strings.Contains(doc, "`"+c+"`") {
			t.Errorf("el código %s (%s) no está en docs/referencia/codigos-de-error.md", c, archivo)
		}
	}
}

// clasesDiagramas son los colores comunes de docs/diagramas: externo, prisma, datos y seguridad.
var clasesDiagramas = []string{
	"classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43",
	"classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13",
	"classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400",
	"classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d",
}

// TestFormatoDeLosDiagramas exige que cada docs/diagramas/NN-nombre.md tenga su diagrama, sus
// fuentes en el código (que deben existir), su fecha de verificación, los colores comunes y una
// fila en el índice.
func TestFormatoDeLosDiagramas(t *testing.T) {
	archivos, err := filepath.Glob(filepath.Join(raiz, "docs", "diagramas", "[0-9][0-9]-*.md"))
	if err != nil || len(archivos) < 20 {
		t.Fatalf("diagramas encontrados: %d (%v)", len(archivos), err)
	}
	indice := leer(t, "docs/diagramas/README.md")
	pantallas, manual := pantallasDelPanel(t), map[string]bool{}
	for _, d := range documentosDelManual(t) {
		manual[strings.TrimSuffix(d, ".md")] = true
	}
	reFuente := regexp.MustCompile("(?m)^- `?([A-Za-z0-9_./-]+)")
	for _, ruta := range archivos {
		nombre := filepath.Base(ruta)
		texto := leer(t, filepath.ToSlash(filepath.Join("docs", "diagramas", nombre)))
		meta, resto, ok := frontMatter(texto)
		if !ok {
			t.Errorf("%s: falta el front-matter (titulo, descripcion, audiencia, pantallas, secciones_manual)", nombre)
		} else {
			for _, e := range revisarFrontMatter(meta, pantallas, manual) {
				t.Errorf("%s: front-matter: %s", nombre, e)
			}
			texto = resto
		}
		for _, seccion := range []string{"\n```mermaid\n", "**Fuentes en el código:**", "**Última verificación:**"} {
			if !strings.Contains(texto, seccion) {
				t.Errorf("%s: falta %q", nombre, strings.TrimSpace(seccion))
			}
		}
		if !strings.HasPrefix(texto, "# ") {
			t.Errorf("%s: debe empezar con el título", nombre)
		}
		if !strings.Contains(indice, "]("+nombre+")") {
			t.Errorf("%s no está en el índice docs/diagramas/README.md", nombre)
		}
		_, bloque, _ := strings.Cut(texto, "```mermaid\n")
		bloque, _, _ = strings.Cut(bloque, "\n```")
		if strings.Contains(bloque, "classDef") {
			for _, c := range clasesDiagramas {
				if !strings.Contains(bloque, c) {
					t.Errorf("%s: los colores deben ser los comunes; falta %q", nombre, c)
				}
			}
		}
		_, fuentes, _ := strings.Cut(texto, "**Fuentes en el código:**")
		fuentes, _, _ = strings.Cut(fuentes, "**Última verificación:**")
		citadas := 0
		for _, m := range reFuente.FindAllStringSubmatch(fuentes, -1) {
			if m[1] == "go" { // `go list …`
				continue
			}
			citadas++
			if _, err := os.Stat(filepath.Join(raiz, filepath.FromSlash(m[1]))); err != nil {
				t.Errorf("%s cita %s, que no existe", nombre, m[1])
			}
		}
		if citadas == 0 {
			t.Errorf("%s no cita ningún archivo del código", nombre)
		}
	}
}

// frontMatter separa el bloque YAML opcional del principio (entre líneas «---»). GitHub lo muestra
// como tabla; el panel lo leerá para la galería de diagramas (Fase 5 de 0.2.0).
func frontMatter(texto string) (meta, resto string, ok bool) {
	if !strings.HasPrefix(texto, "---\n") {
		return "", texto, false
	}
	meta, resto, ok = strings.Cut(texto[4:], "\n---\n")
	if !ok {
		return "", texto, false
	}
	return meta, strings.TrimLeft(resto, "\n"), true
}

// audienciasDiagramas son los públicos que admite el campo audiencia.
var audienciasDiagramas = []string{"usuario", "instalador", "integrador", "desarrollador"}

// pantallasDelPanel son las claves de ayuda.pantallas en textos.js: las pantallas (y subpantallas)
// donde un diagrama puede aparecer como ayuda.
func pantallasDelPanel(t *testing.T) map[string]bool {
	t.Helper()
	fuente := leer(t, "internal/admin/web/textos.js")
	_, resto, ok := strings.Cut(fuente, "\n    pantallas: {\n")
	if !ok {
		t.Fatal("textos.js no tiene ayuda.pantallas")
	}
	bloque, _, _ := strings.Cut(resto, "\n    },")
	claves := map[string]bool{}
	for _, m := range regexp.MustCompile(`(?m)^\s+(\w+):`).FindAllStringSubmatch(bloque, -1) {
		claves[m[1]] = true
	}
	return claves
}

// listaFrontMatter lee una lista «[a, b]» del front-matter.
func listaFrontMatter(v string) []string {
	var l []string
	for _, x := range strings.Split(strings.Trim(v, "[]"), ",") {
		if x = strings.TrimSpace(x); x != "" {
			l = append(l, x)
		}
	}
	return l
}

// revisarFrontMatter exige los campos de los diagramas: titulo, descripcion, audiencia (con valores
// conocidos), pantallas (claves de ayuda.pantallas) y secciones_manual (documentos del manual, sin
// .md). Las listas se escriben en una línea: [a, b].
func revisarFrontMatter(meta string, pantallas, manual map[string]bool) []string {
	campos := map[string]string{}
	for _, linea := range strings.Split(meta, "\n") {
		if clave, valor, ok := strings.Cut(linea, ":"); ok && !strings.HasPrefix(linea, " ") && !strings.HasPrefix(linea, "#") {
			campos[strings.TrimSpace(clave)] = strings.TrimSpace(valor)
		}
	}
	var errores []string
	for _, c := range []string{"titulo", "descripcion", "audiencia", "pantallas", "secciones_manual"} {
		if campos[c] == "" {
			errores = append(errores, "falta "+c)
		}
	}
	for _, c := range []string{"audiencia", "pantallas", "secciones_manual"} {
		if v := campos[c]; v != "" && !(strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]")) {
			errores = append(errores, c+" debe ser una lista [a, b]")
		}
	}
	for _, a := range listaFrontMatter(campos["audiencia"]) {
		if !slices.Contains(audienciasDiagramas, a) {
			errores = append(errores, "audiencia desconocida: "+a)
		}
	}
	if len(listaFrontMatter(campos["audiencia"])) == 0 {
		errores = append(errores, "audiencia vacía")
	}
	for _, p := range listaFrontMatter(campos["pantallas"]) {
		if !pantallas[p] {
			errores = append(errores, "pantalla desconocida (no está en ayuda.pantallas): "+p)
		}
	}
	for _, d := range listaFrontMatter(campos["secciones_manual"]) {
		if !manual[d] {
			errores = append(errores, "secciones_manual: "+d+".md no está en docs/manual.json")
		}
	}
	return errores
}

var (
	reCodigoBloque = regexp.MustCompile("(?s)```.*?```")
	reCodigoLinea  = regexp.MustCompile("`[^`\n]*`")
	reEnlace       = regexp.MustCompile(`\]\(([^)\s]+)\)`)
	reTitulo       = regexp.MustCompile(`^#{1,6}\s+(.*?)\s*#*\s*$`)
	reEnlaceTitulo = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
)

// anclaGitHub convierte un título en su ancla, como GitHub: minúsculas, sin puntuación salvo
// - y _, y los espacios como guiones.
func anclaGitHub(titulo string) string {
	titulo = reEnlaceTitulo.ReplaceAllString(titulo, "$1")
	var b strings.Builder
	for _, r := range strings.ToLower(titulo) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	return b.String()
}

func anclasDe(t *testing.T, ruta string, cache map[string]map[string]bool) map[string]bool {
	if a, ok := cache[ruta]; ok {
		return a
	}
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	anclas, vistas, enCodigo := map[string]bool{}, map[string]int{}, false
	for _, linea := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(linea, "```") {
			enCodigo = !enCodigo
			continue
		}
		if m := reTitulo.FindStringSubmatch(linea); m != nil && !enCodigo {
			a := anclaGitHub(m[1])
			if n := vistas[a]; n > 0 {
				anclas[a+"-"+strconv.Itoa(n)] = true // GitHub numera los títulos repetidos
			} else {
				anclas[a] = true
			}
			vistas[a]++
		}
	}
	cache[ruta] = anclas
	return anclas
}

// TestEnlacesDeLaDocumentacion revisa que cada enlace relativo de cada Markdown del repositorio
// apunte a un archivo que existe y, si lleva ancla, a un título que existe.
func TestEnlacesDeLaDocumentacion(t *testing.T) {
	cache := map[string]map[string]bool{}
	revisados := 0
	err := filepath.WalkDir(raiz, func(ruta string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "dist", "key", "__pycache__":
				if ruta != raiz {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(ruta, ".md") {
			return nil
		}
		b, err := os.ReadFile(ruta)
		if err != nil {
			return err
		}
		texto := reCodigoLinea.ReplaceAllString(reCodigoBloque.ReplaceAllString(string(b), ""), "x")
		for _, m := range reEnlace.FindAllStringSubmatch(texto, -1) {
			destino := m[1]
			if regexp.MustCompile(`^[a-z][a-z0-9+.-]*:`).MatchString(destino) {
				continue // http:, https:, mailto:
			}
			revisados++
			archivo, ancla, _ := strings.Cut(destino, "#")
			objetivo := ruta
			if archivo != "" {
				objetivo = filepath.Join(filepath.Dir(ruta), filepath.FromSlash(archivo))
			}
			info, err := os.Stat(objetivo)
			if err != nil {
				t.Errorf("%s: no existe %s", ruta, destino)
				continue
			}
			if ancla != "" && !info.IsDir() && strings.HasSuffix(objetivo, ".md") && !anclasDe(t, objetivo, cache)[ancla] {
				t.Errorf("%s: no existe el ancla de %s", ruta, destino)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if revisados < 300 {
		t.Fatalf("solo se revisaron %d enlaces: la búsqueda dejó de funcionar", revisados)
	}
}
