package servidor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bayalad7/prisma/internal/config"
	"github.com/bayalad7/prisma/internal/seguridad"
)

// TestComprobadorCoincideConLaAPI: el comprobador de permisos del panel debe dar, para cada
// situación, el mismo resultado que /v1/imprimir. Recorre {nivel 1, 2, 3} × {normal, exige llave
// (nivel 3), desactivada} × {origen propio, desconocido, de otra aplicación, sin origen} ×
// {con llave, sin llave} × {firmado, sin firmar} × {impresora permitida, no permitida, abrir_cajon}.
func TestComprobadorCoincideConLaAPI(t *testing.T) {
	e := nuevoEntorno(t, config.Limites{})
	ctx := context.Background()
	for nivel := 1; nivel <= 3; nivel++ {
		if err := e.alm.Clientes.EstablecerImpresoras(ctx, e.niveles[nivel].id, false, []string{"caja"}); err != nil {
			t.Fatal(err)
		}
	}
	casos := map[string]struct {
		impresora string
		ops       []map[string]any
	}{
		"permitida":    {"caja", nil},
		"no_permitida": {"cocina", nil},
		"cajon":        {"caja", []map[string]any{{"tipo": "texto", "valor": "Venta"}, {"tipo": "abrir_cajon"}}},
	}
	for nivel := 1; nivel <= 3; nivel++ {
		c := e.niveles[nivel]
		otra := e.niveles[nivel%3+1]
		for _, variante := range []string{"normal", "exige_llave", "desactivada"} {
			if variante == "exige_llave" && nivel != 3 {
				continue
			}
			cli, _ := e.alm.Clientes.Obtener(ctx, c.id)
			cli.Activo, cli.ExigirLlave = variante != "desactivada", variante == "exige_llave"
			if err := e.alm.Clientes.Actualizar(ctx, cli); err != nil {
				t.Fatal(err)
			}
			for _, origen := range []string{"propio", "desconocido", "otra", "ausente"} {
				for _, llave := range []bool{true, false} {
					for _, firmado := range []bool{true, false} {
						for nombreCaso, caso := range casos {
							nombre := fmt.Sprintf("nivel%d/%s/origen_%s/llave_%v/firmado_%v/%s", nivel, variante, origen, llave, firmado, nombreCaso)
							t.Run(nombre, func(t *testing.T) {
								cabOrigen := map[string]string{"propio": c.origen, "desconocido": "https://desconocido.example",
									"otra": otra.origen, "ausente": ""}[origen]
								doc := documento(caso.impresora, caso.ops...)
								datos := cuerpo(t, doc)
								if firmado {
									datos = e.firmar(doc, c.id, time.Now())
								}
								cabLlave := ""
								if llave {
									cabLlave = c.llave
								}
								real := e.imprimir(datos, cabOrigen, cabLlave)

								tipos := []string{}
								for _, op := range doc["operaciones"].([]map[string]any) {
									tipos = append(tipos, op["tipo"].(string))
								}
								comp, err := e.srv.aut.Comprobar(ctx, seguridad.Situacion{ClienteID: c.id, Origen: cabOrigen, ConLlave: llave,
									Firmado: firmado, Impresora: caso.impresora, Operaciones: tipos})
								if err != nil {
									t.Fatal(err)
								}
								if comp.Codigo != real.codigo() || (comp.Codigo != "" && comp.Estado != real.estado) {
									t.Fatalf("comprobador %d %q, API %d %q\n%s", comp.Estado, comp.Codigo, real.estado, real.codigo(), real.crudo)
								}
							})
						}
					}
				}
			}
		}
	}
}
