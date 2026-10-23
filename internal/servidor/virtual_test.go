package servidor

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/conector"
	"github.com/bayalad7/prisma/internal/config"
	"github.com/bayalad7/prisma/internal/contrato"
	"github.com/bayalad7/prisma/internal/escpos"
)

// agregarVirtual registra la impresora virtual «prueba» y vacía su rollo al terminar.
func (e *entorno) agregarVirtual(sim string) *almacen.Impresora {
	e.t.Helper()
	imp := &almacen.Impresora{Alias: "prueba", Nombre: "Para probar", Tipo: almacen.ConexionVirtual, Destino: "memoria", AnchoMM: 80,
		Columnas: 48, ColumnasFuenteB: 64, AnchoPx: 576, PaginaCodigos: "CP858", ComandoPaginaCodigos: 19, SoportaQRNativo: true,
		SoportaCorte: true, SoportaImagen: true, SoportaCodigoBarrasNativo: true, Activa: true, Simulacion: sim}
	if err := e.alm.Impresoras.Crear(context.Background(), imp); err != nil {
		e.t.Fatal(err)
	}
	conector.RollosVirtuales.Vaciar("prueba")
	e.t.Cleanup(func() { conector.RollosVirtuales.Vaciar("prueba") })
	return imp
}

// TestImpresoraVirtualDePrincipioAFin: una aplicación imprime por la API v1 en la impresora
// virtual sin cambiar nada; el rollo recibe exactamente los bytes del renderizador y el contenido
// del ticket no llega a la bitácora ni a la base.
func TestImpresoraVirtualDePrincipioAFin(t *testing.T) {
	e := nuevoEntorno(t, config.Limites{})
	var bitacora bytes.Buffer
	e.srv.op.Log = slog.New(slog.NewJSONHandler(&bitacora, nil))
	imp := e.agregarVirtual(almacen.SimulacionNormal)
	c := e.niveles[2]
	cab := map[string]string{"Origin": c.origen, "Authorization": "Bearer " + c.llave}

	r := e.pedir(http.MethodGet, "/v1/impresoras", nil, cab)
	virtuales := map[string]bool{}
	var lista []map[string]any
	if err := json.Unmarshal([]byte(r.crudo), &lista); err != nil {
		t.Fatal(err)
	}
	for _, i := range lista {
		virtuales[i["alias"].(string)], _ = i["virtual"].(bool)
	}
	if !virtuales["prueba"] || virtuales["caja"] {
		t.Fatalf("/v1/impresoras debe marcar solo «prueba» como virtual: %s", r.crudo)
	}

	const secreto = "Cliente Ramírez · tarjeta 4111"
	doc := documento("prueba", map[string]any{"tipo": "texto", "valor": secreto}, map[string]any{"tipo": "qr", "valor": "https://ejemplo.mx"},
		map[string]any{"tipo": "cortar"})
	datosDoc := cuerpo(t, doc)
	r = e.imprimir(datosDoc, c.origen, c.llave)
	if r.estado != http.StatusOK {
		t.Fatalf("imprimir: %d %s", r.estado, r.crudo)
	}
	trabajoID := r.cuerpo["trabajo_id"].(string)

	ticket, ok := conector.RollosVirtuales.Ticket("prueba", trabajoID)
	if !ok {
		t.Fatal("el ticket no llegó al rollo")
	}
	decodificado, err := contrato.Decodificar(datosDoc)
	if err != nil {
		t.Fatal(err)
	}
	esperados, _, err := escpos.Renderizar(*decodificado, *imp, escpos.RecursosMapa{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ticket.Datos, esperados) {
		t.Fatal("la impresora virtual debe recibir exactamente los bytes que recibiría una física")
	}
	trabajo, err := e.alm.Trabajos.Obtener(context.Background(), trabajoID)
	if err != nil || !trabajo.Virtual || trabajo.Estado != almacen.TrabajoCompletado {
		t.Fatalf("trabajo: %+v, %v", trabajo, err)
	}

	if strings.Contains(bitacora.String(), "Ramírez") || !strings.Contains(bitacora.String(), `"virtual":true`) {
		t.Fatalf("la bitácora no debe tener el contenido del ticket y sí la marca virtual:\n%s", bitacora.String())
	}
	respaldo := filepath.Join(t.TempDir(), "respaldo.db")
	if err := e.alm.Respaldar(context.Background(), respaldo); err != nil {
		t.Fatal(err)
	}
	base, _ := os.ReadFile(respaldo)
	if bytes.Contains(base, []byte("Ramírez")) || bytes.Contains(base, esperados) {
		t.Fatal("el contenido del ticket virtual no debe guardarse en la base")
	}
}

// TestImpresoraVirtualQueFallaLiberaElID: con la simulación «fuera de línea» el trabajo falla con un
// código que la v1 ya conoce, no queda en el rollo y su id se libera: el reintento se imprime.
func TestImpresoraVirtualQueFallaLiberaElID(t *testing.T) {
	e := nuevoEntorno(t, config.Limites{})
	imp := e.agregarVirtual(almacen.SimulacionFueraDeLinea)
	c := e.niveles[2]
	datos := cuerpo(t, documento("prueba"))

	r := e.imprimir(datos, c.origen, c.llave)
	if r.codigo() != conector.ErrImpresoraInalcanzable {
		t.Fatalf("debía fallar con IMPRESORA_INALCANZABLE: %d %s", r.estado, r.crudo)
	}
	if len(conector.RollosVirtuales.Listar("prueba")) != 0 {
		t.Fatal("un ticket que falló no debe quedar en el rollo")
	}

	time.Sleep(5 * time.Millisecond) // otra marca de actualización: el puente cambia el conector
	imp.Simulacion = almacen.SimulacionNormal
	if err := e.alm.Impresoras.Actualizar(context.Background(), imp); err != nil {
		t.Fatal(err)
	}
	r = e.imprimir(datos, c.origen, c.llave)
	if r.estado != http.StatusOK || r.cuerpo["duplicado"] == true {
		t.Fatalf("el reintento debe imprimirse como nuevo: %d %s", r.estado, r.crudo)
	}
	if len(conector.RollosVirtuales.Listar("prueba")) != 1 {
		t.Fatal("el reintento debe llegar al rollo")
	}
}
