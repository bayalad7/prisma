package servidor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/conector"
	"github.com/bayalad7/prisma/internal/config"
	"github.com/bayalad7/prisma/internal/dev"
)

// TestImpresoraVirtualCoincideConLosGolden: cada ejemplo válido del repositorio, impreso por
// /v1/imprimir en una impresora virtual de 58 y de 80 mm, deja en el rollo la misma vista y la misma
// traza ESC/POS que los golden de internal/dev (go test ./internal/dev). Lo que se ve en el panel es
// lo que saldría en el papel.
func TestImpresoraVirtualCoincideConLosGolden(t *testing.T) {
	e := nuevoEntorno(t, config.Limites{})
	ctx := context.Background()
	c := e.niveles[2]
	cli, err := e.alm.Clientes.Obtener(ctx, c.id)
	if err != nil {
		t.Fatal(err)
	}
	cli.OperacionesSensibles = []string{almacen.OperacionAbrirCajon, almacen.OperacionRaw}
	if err := e.alm.Clientes.Actualizar(ctx, cli); err != nil {
		t.Fatal(err)
	}
	// El mismo logo que usan los golden (recursosPrueba de internal/dev).
	logo := image.NewGray(image.Rect(0, 0, 240, 60))
	for y := range 60 {
		for x := range 240 {
			v := uint8(255)
			if (x/20+y/20)%2 == 0 {
				v = 0
			}
			logo.SetGray(x, y, color.Gray{Y: v})
		}
	}
	var b bytes.Buffer
	png.Encode(&b, logo)
	if err := e.alm.Recursos.Guardar(ctx, &almacen.Recurso{Nombre: "logo-tienda", PNG: b.Bytes(), AnchoPx: 240, AltoPx: 60}); err != nil {
		t.Fatal(err)
	}

	perfiles := map[int]almacen.Impresora{}
	for _, mm := range []int{58, 80} {
		p, _ := dev.PerfilPrueba(mm)
		p.Alias, p.Tipo, p.Destino = fmt.Sprintf("virtual-%d", mm), almacen.ConexionVirtual, "memoria"
		p.Simulacion = almacen.SimulacionNormal
		if err := e.alm.Impresoras.Crear(ctx, &p); err != nil {
			t.Fatal(err)
		}
		perfiles[mm] = p
		t.Cleanup(func() { conector.RollosVirtuales.Vaciar(p.Alias) })
	}

	archivos, _ := filepath.Glob(filepath.Join("..", "..", "ejemplos", "validos", "*.json"))
	if len(archivos) == 0 {
		t.Fatal("no se encontraron los ejemplos válidos")
	}
	for _, a := range archivos {
		for _, mm := range []int{58, 80} {
			nombre := fmt.Sprintf("%s-%d", strings.TrimSuffix(filepath.Base(a), ".json"), mm)
			t.Run(nombre, func(t *testing.T) {
				crudo, err := os.ReadFile(a)
				if err != nil {
					t.Fatal(err)
				}
				var doc map[string]any
				if err := json.Unmarshal(crudo, &doc); err != nil {
					t.Fatal(err)
				}
				doc["impresora"], doc["id"] = perfiles[mm].Alias, uuid4()
				datos, origen, llave := cuerpo(t, doc), c.origen, c.llave
				if _, firmado := doc["firma"]; firmado {
					// La firma del ejemplo ya expiró: lo firma de nuevo el cliente de nivel 3.
					for _, campo := range []string{"firma", "cliente_id", "emitido", "expira"} {
						delete(doc, campo)
					}
					c3 := e.niveles[3]
					datos, origen, llave = e.firmar(doc, c3.id, time.Now()), c3.origen, c3.llave
				}
				r := e.imprimir(datos, origen, llave)
				if r.estado != http.StatusOK {
					t.Fatalf("imprimir: %d %s", r.estado, r.crudo)
				}
				ticket, ok := conector.RollosVirtuales.Ticket(perfiles[mm].Alias, r.cuerpo["trabajo_id"].(string))
				if !ok {
					t.Fatal("el ticket no llegó al rollo")
				}

				golden, err := os.ReadFile(filepath.Join("..", "dev", "testdata", "golden", nombre+".txt"))
				if err != nil {
					t.Fatal(err)
				}
				vista, traza := seccionesGolden(t, strings.ReplaceAll(string(golden), "\r\n", "\n"))
				if got := dev.VistaTexto(dev.Simular(ticket.Datos, perfiles[mm]), false); got != vista {
					t.Errorf("la vista del rollo no coincide con el golden %s", nombre)
				}
				if got := dev.Traza(ticket.Datos); got != traza {
					t.Errorf("la traza del rollo no coincide con el golden %s", nombre)
				}
			})
		}
	}
}

// seccionesGolden separa la vista y la traza de un golden de internal/dev.
func seccionesGolden(t *testing.T, golden string) (vista, traza string) {
	t.Helper()
	const cabVista, cabAdv, cabTraza = "== Vista ==\n", "\n== Advertencias ==\n", "\n== Traza ==\n"
	i, j, k := strings.Index(golden, cabVista), strings.Index(golden, cabAdv), strings.Index(golden, cabTraza)
	if i != 0 || j < 0 || k < j {
		t.Fatal("golden con un formato inesperado")
	}
	return golden[len(cabVista):j], golden[k+len(cabTraza):]
}
