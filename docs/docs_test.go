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
