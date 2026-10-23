package admin

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode"

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

// Búsqueda del manual: un índice por sección (cada título del documento) que se arma una vez desde
// lo embebido, sin servicios externos. No distingue mayúsculas, acentos ni la ñ, y busca por
// prefijo: «contrasena olvid» encuentra «Olvidé la contraseña del panel».

type seccionManual struct {
	ruta, titulo, seccion, ancla string
	texto                        []rune // en texto plano, para el extracto
	normal, normalTitulo         string // plegados para comparar
	normalSeccion                string
}

type resultadoManual struct {
	Ruta     string `json:"ruta"`
	Titulo   string `json:"titulo"`
	Seccion  string `json:"seccion,omitempty"`
	Ancla    string `json:"ancla,omitempty"`
	Extracto string `json:"extracto"`
}

var (
	reEnlaceMD   = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`)
	reMarcasMD   = regexp.MustCompile("[*`]+|^\\s*(#{1,6}|>|[-*]|\\d+\\.)\\s+")
	reTituloMD   = regexp.MustCompile(`^(#{1,6})\s+(.*?)\s*#*\s*$`)
	reSeparadorM = regexp.MustCompile(`^\s*\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)*\|?\s*$`)
	reCercaMD    = regexp.MustCompile("^\\s*```")
)

// plano quita la sintaxis de Markdown de una línea, como la dibuja manual.js.
func plano(linea string) string {
	linea = reEnlaceMD.ReplaceAllString(linea, "$1")
	linea = strings.ReplaceAll(linea, `\|`, "\x00")
	linea = reMarcasMD.ReplaceAllString(linea, "")
	linea = strings.ReplaceAll(linea, "|", " ")
	return strings.TrimSpace(strings.ReplaceAll(linea, "\x00", "|"))
}

// plegar pasa a minúsculas sin acentos, letra por letra (mismo largo en runas, para los extractos).
func plegar(r []rune) string {
	b := make([]rune, len(r))
	for i, c := range r {
		c = unicode.ToLower(c)
		switch c {
		case 'á', 'à', 'ä', 'â':
			c = 'a'
		case 'é', 'è', 'ë', 'ê':
			c = 'e'
		case 'í', 'ì', 'ï', 'î':
			c = 'i'
		case 'ó', 'ò', 'ö', 'ô':
			c = 'o'
		case 'ú', 'ù', 'ü', 'û':
			c = 'u'
		case 'ñ':
			c = 'n'
		}
		b[i] = c
	}
	return string(b)
}

// anclaDe es el ancla de un título, como la calcula manual.js (y GitHub).
func anclaDe(t string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(strings.TrimSpace(t)) {
		switch {
		case unicode.IsSpace(c):
			b.WriteRune('-')
		case unicode.IsLetter(c), unicode.IsNumber(c), c == '_', c == '-':
			b.WriteRune(c)
		}
	}
	return b.String()
}

// seccionesDe parte un documento por sus títulos (## y más), con la descripción del front-matter
// (la de los diagramas) en la primera.
func seccionesDe(ruta, titulo, texto string) []seccionManual {
	descripcion := ""
	if strings.HasPrefix(texto, "---\n") {
		if meta, resto, ok := strings.Cut(texto[4:], "\n---\n"); ok {
			texto = resto
			for _, l := range strings.Split(meta, "\n") {
				if v, ok := strings.CutPrefix(l, "descripcion:"); ok {
					descripcion = strings.Trim(strings.TrimSpace(v), `"`)
				}
			}
		}
	}
	var secciones []seccionManual
	usadas := map[string]int{}
	actual := seccionManual{ruta: ruta, titulo: titulo}
	var cuerpo []string
	if descripcion != "" {
		cuerpo = append(cuerpo, descripcion)
	}
	cerrar := func() {
		actual.texto = []rune(strings.Join(cuerpo, " "))
		actual.normal = plegar(actual.texto)
		actual.normalTitulo = plegar([]rune(actual.titulo))
		actual.normalSeccion = plegar([]rune(actual.seccion))
		secciones = append(secciones, actual)
	}
	enCodigo, mermaid := false, false
	for _, linea := range strings.Split(texto, "\n") {
		if reCercaMD.MatchString(linea) {
			enCodigo = !enCodigo
			mermaid = enCodigo && strings.Contains(linea, "mermaid")
			continue
		}
		if enCodigo {
			if !mermaid {
				cuerpo = append(cuerpo, strings.TrimSpace(linea))
			}
			continue
		}
		if m := reTituloMD.FindStringSubmatch(linea); m != nil {
			t := plano(m[2])
			ancla := anclaDe(t)
			if n := usadas[ancla]; n > 0 {
				usadas[ancla] = n + 1
				ancla = fmt.Sprintf("%s-%d", ancla, n)
			} else {
				usadas[ancla] = 1
			}
			if len(m[1]) == 1 {
				continue // el título del documento
			}
			cerrar()
			actual = seccionManual{ruta: ruta, titulo: titulo, seccion: t, ancla: ancla}
			cuerpo = nil
			continue
		}
		if reSeparadorM.MatchString(linea) || strings.HasPrefix(strings.TrimSpace(linea), "{{diagrama:") {
			continue
		}
		if p := plano(linea); p != "" {
			cuerpo = append(cuerpo, p)
		}
	}
	cerrar()
	return secciones
}

var indiceBusqueda = sync.OnceValues(func() ([]seccionManual, error) {
	grupos, err := indiceManual()
	if err != nil {
		return nil, err
	}
	var secciones []seccionManual
	for _, g := range grupos {
		for _, d := range g.Documentos {
			datos, err := fs.ReadFile(docs.Manual, d.Ruta)
			if err != nil {
				return nil, err
			}
			secciones = append(secciones, seccionesDe(d.Ruta, d.Titulo, string(datos))...)
		}
	}
	return secciones, nil
})

// extracto: unos 160 caracteres alrededor de la primera coincidencia.
func extracto(s seccionManual, palabra string) string {
	inicio := 0
	if i := strings.Index(s.normal, palabra); i >= 0 {
		inicio = len([]rune(s.normal[:i]))
	}
	desde := max(0, inicio-60)
	// Empieza en una palabra completa.
	for desde > 0 && desde < inicio && !unicode.IsSpace(s.texto[desde-1]) {
		desde++
	}
	hasta := min(len(s.texto), desde+160)
	e := strings.TrimSpace(string(s.texto[desde:hasta]))
	if desde > 0 {
		e = "…" + e
	}
	if hasta < len(s.texto) {
		e += "…"
	}
	return e
}

// GET /admin/api/manual/buscar?q=…: las secciones que contienen todas las palabras, las mejores
// primero (en el título del documento pesan más, luego en el de la sección), hasta 20 y 3 por documento.
func (a *Admin) buscarManual(w http.ResponseWriter, r *http.Request) {
	secciones, err := indiceBusqueda()
	if err != nil {
		a.error(w, r, err)
		return
	}
	var palabras []string
	for _, p := range strings.FieldsFunc(plegar([]rune(r.URL.Query().Get("q"))), func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c) && c != '_'
	}) {
		if len([]rune(p)) >= 2 && len(palabras) < 8 {
			palabras = append(palabras, p)
		}
	}
	type candidato struct {
		s      seccionManual
		puntos int
	}
	var encontrados []candidato
	for _, s := range secciones {
		puntos := 0
		for _, p := range palabras {
			enTitulo, enSeccion, veces := strings.Contains(s.normalTitulo, p), strings.Contains(s.normalSeccion, p), strings.Count(s.normal, p)
			if !enTitulo && !enSeccion && veces == 0 {
				puntos = -1
				break
			}
			if enTitulo {
				puntos += 10
			}
			if enSeccion {
				puntos += 6
			}
			puntos += min(veces, 5)
		}
		if len(palabras) > 0 && puntos > 0 {
			// Ante un empate, primero las guías del dueño (administracion/) que las técnicas.
			if strings.HasPrefix(s.ruta, "administracion/") {
				puntos += 3
			}
			encontrados = append(encontrados, candidato{s, puntos})
		}
	}
	slices.SortStableFunc(encontrados, func(x, y candidato) int { return y.puntos - x.puntos })
	resultados := []resultadoManual{}
	porDocumento := map[string]int{}
	for _, c := range encontrados {
		if len(resultados) == 20 || porDocumento[c.s.ruta] == 3 {
			continue
		}
		porDocumento[c.s.ruta]++
		resultados = append(resultados, resultadoManual{Ruta: c.s.ruta, Titulo: c.s.titulo, Seccion: c.s.seccion, Ancla: c.s.ancla,
			Extracto: extracto(c.s, palabras[0])})
	}
	responder(w, resultados)
}
