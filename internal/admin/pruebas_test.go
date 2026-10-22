package admin

import (
	"bytes"
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

func TestComprobarPermisos(t *testing.T) {
	b := nuevoBanco(t)
	s := b.configurado()
	s.exigir("POST", "/admin/api/impresoras", perfilVirtual(nil), 200)
	cli := s.exigir("POST", "/admin/api/clientes", map[string]any{"nombre": "Tienda", "nivel": 2, "origenes": []string{"https://tienda.example"},
		"todas_las_impresoras": true}, 200)
	id, _ := cli.datos["id"].(string)
	if id == "" {
		id, _ = cli.datos["cliente"].(map[string]any)["id"].(string)
	}
	comprobar := func(cambios map[string]any) respuesta {
		d := map[string]any{"cliente_id": id, "origen": "https://tienda.example", "con_llave": true, "impresora": "prueba", "operaciones": []string{"texto"}}
		for k, v := range cambios {
			d[k] = v
		}
		return s.exigir("POST", "/admin/api/pruebas/permisos", d, 200)
	}

	r := comprobar(nil)
	if r.datos["codigo"] != "" || r.datos["identificado"].(map[string]any)["nombre"] != "Tienda" || !strings.Contains(r.Body.String(), `"id":"disponible","estado":"ok"`) {
		t.Fatalf("todo en orden: %s", r.Body.String())
	}
	if r = comprobar(map[string]any{"con_llave": false}); r.datos["codigo"] != "LLAVE_REQUERIDA" || r.datos["estado_http"] != float64(401) {
		t.Fatalf("sin llave: %s", r.Body.String())
	}
	if r = comprobar(map[string]any{"operaciones": []string{"texto", "abrir_cajon"}}); r.datos["codigo"] != "OPERACION_NO_PERMITIDA" {
		t.Fatalf("cajón: %s", r.Body.String())
	}
	if r = comprobar(map[string]any{"impresora": "no-existe"}); r.datos["codigo"] != "IMPRESORA_NO_ENCONTRADA" {
		t.Fatalf("impresora inexistente: %s", r.Body.String())
	}

	// Un origen desconocido falla igual que en la API, pero no queda como solicitud pendiente.
	if r = comprobar(map[string]any{"origen": "https://otra.example"}); r.datos["codigo"] != "ORIGEN_NO_AUTORIZADO" {
		t.Fatalf("origen desconocido: %s", r.Body.String())
	}
	if sol := s.exigir("GET", "/admin/api/solicitudes", nil, 200); strings.Contains(sol.Body.String(), "otra.example") {
		t.Fatalf("el comprobador no debe registrar solicitudes: %s", sol.Body.String())
	}
	// El origen de otra aplicación la identifica a ella: la llave de esta ya no corresponde.
	otra := s.exigir("POST", "/admin/api/clientes", map[string]any{"nombre": "Otra", "nivel": 2, "origenes": []string{"https://otra-app.example"},
		"todas_las_impresoras": true}, 200)
	if r = comprobar(map[string]any{"origen": "https://otra-app.example"}); r.datos["codigo"] != "LLAVE_INVALIDA" ||
		r.datos["origen_de"].(map[string]any)["nombre"] != "Otra" {
		t.Fatalf("origen de otra aplicación: %s %v", r.Body.String(), otra.datos)
	}
	s.exigirError("POST", "/admin/api/pruebas/permisos", map[string]any{"cliente_id": "cli_no_existe"}, 404, "NO_ENCONTRADO")
}

// TestProbarImpresionNoGuardaElContenido: lo que se prueba desde el panel (vista previa, envío a la
// virtual, documentos con errores, el rollo y su traza) nunca deja el contenido del ticket en la
// bitácora ni en la base; el respaldo que descarga el panel tampoco lo tiene.
func TestProbarImpresionNoGuardaElContenido(t *testing.T) {
	b := nuevoBanco(t)
	s := b.configurado()
	conector.RollosVirtuales.Vaciar("prueba")
	t.Cleanup(func() { conector.RollosVirtuales.Vaciar("prueba") })
	s.exigir("POST", "/admin/api/impresoras", perfilVirtual(nil), 200)

	const secreto = "Cliente Ramírez · tarjeta 4111"
	doc := func(ops ...map[string]any) map[string]any {
		return map[string]any{"version": "1", "id": "0b6a3d4e-5f60-4a1b-8c2d-3e4f5a6b7c8d", "impresora": "prueba", "operaciones": ops}
	}
	texto := map[string]any{"tipo": "texto", "valor": secreto}
	s.exigir("POST", "/admin/api/pruebas/vista-previa", doc(texto, map[string]any{"tipo": "cortar"}), 200)
	r := s.exigir("POST", "/admin/api/pruebas/imprimir", doc(texto, map[string]any{"tipo": "qr", "valor": secreto}), 200)
	trabajo, _ := r.datos["trabajo_id"].(string)
	s.exigir("POST", "/admin/api/pruebas/vista-previa", doc(texto, map[string]any{"tipo": "codigo_barras", "simbologia": "EAN13", "valor": secreto}), 400)
	s.exigir("POST", "/admin/api/pruebas/imprimir", doc(map[string]any{"tipo": "inventado", "valor": secreto}), 400)
	s.exigir("GET", "/admin/api/impresoras/prueba/rollo", nil, 200)
	s.exigir("GET", "/admin/api/impresoras/prueba/rollo/"+trabajo+"/traza", nil, 200)
	s.exigir("GET", "/admin/api/impresoras/prueba/rollo/"+trabajo+"/imagen", nil, 200)

	if ticket, ok := conector.RollosVirtuales.Ticket("prueba", trabajo); !ok || !bytes.Contains(ticket.Datos, []byte("tarjeta 4111")) {
		t.Fatal("el ticket debe estar en el rollo, en memoria")
	}
	if bit := b.bitacora.String(); strings.Contains(bit, "Ramírez") || strings.Contains(bit, "4111") || !strings.Contains(bit, `"virtual":true`) {
		t.Fatalf("la bitácora no debe tener el contenido del ticket y sí la marca virtual:\n%s", bit)
	}
	respaldo := s.exigir("GET", "/admin/api/respaldo", nil, 200).Body.Bytes()
	if len(respaldo) == 0 || bytes.Contains(respaldo, []byte("Ramírez")) || bytes.Contains(respaldo, []byte("tarjeta 4111")) {
		t.Fatalf("el respaldo no debe tener el contenido del ticket (%d bytes)", len(respaldo))
	}
}
