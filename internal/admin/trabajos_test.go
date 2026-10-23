package admin

import (
	"io/fs"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/bayalad7/prisma/internal/almacen"
)

// TestCodigosDeTrabajosExplicados: cada código que puede quedar guardado en un trabajo (errores de
// los conectores y de la cola, y advertencias del renderizador) tiene en textos.js su explicación
// para el detalle del trabajo. Los códigos se leen de las constantes, así que uno nuevo sin
// explicar hace fallar la prueba.
func TestCodigosDeTrabajosExplicados(t *testing.T) {
	textos, err := fs.ReadFile(archivosWeb, "web/textos.js")
	if err != nil {
		t.Fatal(err)
	}
	seccion := func(nombre string) string {
		_, resto, ok := strings.Cut(string(textos), "\n    "+nombre+": {\n")
		if !ok {
			t.Fatalf("textos.js no tiene trabajos.%s", nombre)
		}
		bloque, _, _ := strings.Cut(resto, "\n    },\n")
		return bloque
	}
	constantes := func(archivo, patron string) []string {
		fuente, err := os.ReadFile(archivo)
		if err != nil {
			t.Fatal(err)
		}
		var codigos []string
		for _, m := range regexp.MustCompile(patron).FindAllStringSubmatch(string(fuente), -1) {
			codigos = append(codigos, m[1])
		}
		if len(codigos) == 0 {
			t.Fatalf("no se encontraron códigos en %s", archivo)
		}
		return codigos
	}

	errores := append(constantes("../conector/conector.go", `\tErr[A-Za-z]+ += "([A-Z_]+)"`), almacen.CodigoPuenteReiniciado, "ERROR_INTERNO")
	avisos := append(constantes("../escpos/renderizar.go", `\tAdv[A-Za-z]+ += "([A-Z_]+)"`), constantes("../conector/conector.go", `const AvisoFueraDeLinea = "([A-Z_]+)"`)...)
	for nombre, lista := range map[string][]string{"codigos": errores, "avisos": avisos} {
		bloque := "\n" + seccion(nombre)
		for _, codigo := range lista {
			if !strings.Contains(bloque, "\n      "+codigo+": {") {
				t.Errorf("el código %s no tiene explicación en textos.js (trabajos.%s)", codigo, nombre)
			}
		}
	}
}
