package admin

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bayalad7/prisma/internal/conector"
	"github.com/bayalad7/prisma/internal/contrato"
)

func TestProbarImpresion(t *testing.T) {
	b := nuevoBanco(t)
	s := b.configurado()
	conector.RollosVirtuales.Vaciar("prueba")
	t.Cleanup(func() { conector.RollosVirtuales.Vaciar("prueba") })
	s.exigir("POST", "/admin/api/impresoras", perfilVirtual(map[string]any{"soporta_cajon": false}), 200)

	// Plantillas: los ejemplos del repositorio y, con una impresora, su diagnóstico; todas son
	// documentos válidos una vez que el panel les pone un id.
	r := s.exigir("GET", "/admin/api/pruebas/plantillas?impresora=prueba", nil, 200)
	var plantillas []plantillaPrueba
	if err := json.Unmarshal(r.Body.Bytes(), &plantillas); err != nil {
		t.Fatal(err)
	}
	ids := map[string]plantillaPrueba{}
	for _, p := range plantillas {
		ids[p.ID] = p
		var doc map[string]any
		json.Unmarshal(p.Documento, &doc)
		doc["id"] = "0b6a3d4e-5f60-4a1b-8c2d-3e4f5a6b7c8d"
		datos, _ := json.Marshal(doc)
		if _, err := contrato.Decodificar(datos); err != nil {
			t.Errorf("la plantilla %s no es un documento válido: %v", p.ID, err)
		}
	}
	if len(plantillas) != len(plantillasPrueba)+1 || plantillas[0].ID != "diagnostico" {
		t.Fatalf("plantillas: %v", ids)
	}
	if f := ids["todas-las-operaciones"].Faltan; len(f) != 1 || f[0] != "logo-tienda" {
		t.Fatalf("debe avisar de la imagen guardada que falta: %v", f)
	}
	if strings.Contains(r.Body.String(), "con-raw") || strings.Contains(r.Body.String(), `"firma"`) {
		t.Fatal("no se ofrecen las plantillas con raw ni firmadas")
	}

	doc := func(ops ...map[string]any) map[string]any {
		return map[string]any{"version": "1", "id": "0b6a3d4e-5f60-4a1b-8c2d-3e4f5a6b7c8d", "impresora": "prueba",
			"referencia": "PRUEBA-PANEL", "operaciones": ops}
	}
	texto := map[string]any{"tipo": "texto", "valor": "Hola"}

	// Vista previa: la imagen y las advertencias del render, sin imprimir nada.
	r = s.exigir("POST", "/admin/api/pruebas/vista-previa", doc(texto, map[string]any{"tipo": "abrir_cajon"}), 200)
	if img, _ := r.datos["imagen"].(string); !strings.HasPrefix(img, "data:image/png;base64,") {
		t.Fatalf("vista previa sin imagen: %.80v", r.datos)
	}
	if !strings.Contains(r.Body.String(), `"codigo":"SIN_CAJON"`) {
		t.Fatalf("la vista previa debe traer la advertencia del cajón: %s", r.Body.String())
	}
	if n := len(conector.RollosVirtuales.Listar("prueba")); n != 0 {
		t.Fatalf("la vista previa no debe imprimir: hay %d tickets", n)
	}

	// Los errores del contrato llegan con la ruta del campo, como en /v1.
	r = s.exigirError("POST", "/admin/api/pruebas/vista-previa", doc(map[string]any{"tipo": "texto"}), 400, "DOCUMENTO_INVALIDO")
	if !strings.Contains(r.Body.String(), `"ruta":"/operaciones/0`) {
		t.Fatalf("el error debe señalar la operación: %s", r.Body.String())
	}
	sinImpresora := doc(texto)
	sinImpresora["impresora"] = "no-existe"
	s.exigirError("POST", "/admin/api/pruebas/vista-previa", sinImpresora, 404, "IMPRESORA_NO_ENCONTRADA")

	// Imprimir: responde como /v1/imprimir, el ticket llega al rollo y el historial lo guarda con
	// sus advertencias a nombre del panel.
	r = s.exigir("POST", "/admin/api/pruebas/imprimir", doc(texto, map[string]any{"tipo": "abrir_cajon"}), 200)
	trabajo, _ := r.datos["trabajo_id"].(string)
	if r.datos["ok"] != true || r.datos["impresora"] != "prueba" || r.datos["duracion_ms"] == nil || !strings.Contains(r.Body.String(), "SIN_CAJON") {
		t.Fatalf("respuesta de imprimir: %s", r.Body.String())
	}
	if tk, ok := conector.RollosVirtuales.Ticket("prueba", trabajo); !ok || len(tk.Datos) == 0 {
		t.Fatalf("el ticket debe llegar al rollo: %+v", tk)
	}
	h := s.exigir("GET", "/admin/api/trabajos", nil, 200).Body.String()
	if !strings.Contains(h, trabajo) || !strings.Contains(h, `"cliente_id":"cli_panel"`) || !strings.Contains(h, "SIN_CAJON") {
		t.Fatalf("historial: %s", h)
	}

	// Una impresora desactivada no imprime.
	s.exigir("PUT", "/admin/api/impresoras/prueba", perfilVirtual(map[string]any{"activa": false}), 200)
	s.exigirError("POST", "/admin/api/pruebas/imprimir", doc(texto), 409, "IMPRESORA_INACTIVA")
}
