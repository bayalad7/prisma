package admin

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/bayalad7/prisma/ejemplos"
	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/conector"
	"github.com/bayalad7/prisma/internal/contrato"
	"github.com/bayalad7/prisma/internal/dev"
	"github.com/bayalad7/prisma/internal/escpos"
	"github.com/bayalad7/prisma/internal/seguridad"
	"github.com/bayalad7/prisma/internal/servidor"
)

// Área «Probar impresión»: plantillas, vista previa y envío de documentos desde el panel. El
// documento sigue el mismo camino que en /v1/imprimir (contrato → render → cola → conector); la
// autorización es la sesión del panel y el trabajo queda a nombre de la aplicación «Panel».

// plantillasPrueba son los ejemplos que se ofrecen, en orden. con-raw y firmado se omiten: el
// primero repite operaciones de «todas» y el segundo solo sirve firmado por un backend.
var plantillasPrueba = []string{"venta-simple", "venta-logo-qr", "comanda-cocina", "todas-las-operaciones"}

type plantillaPrueba struct {
	ID        string          `json:"id"`
	Documento json.RawMessage `json:"documento"`
	// Faltan son las imágenes guardadas que usa la plantilla y no existen en este puente.
	Faltan []string `json:"faltan,omitempty"`
}

// GET /admin/api/pruebas/plantillas[?impresora=alias]: con una impresora, agrega su ticket de diagnóstico.
func (a *Admin) plantillas(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var lista []plantillaPrueba
	if alias := r.URL.Query().Get("impresora"); alias != "" {
		if imp, err := a.op.Almacen.Impresoras.Obtener(ctx, alias); err == nil {
			lista = append(lista, plantillaPrueba{ID: "diagnostico", Documento: documentoJSON(imp.Alias, "DIAGNOSTICO", ticketDiagnostico(*imp, a.op.Ahora()))})
		}
	}
	for _, id := range plantillasPrueba {
		datos, err := ejemplos.Validos.ReadFile("validos/" + id + ".json")
		if err != nil {
			a.error(w, r, err)
			return
		}
		p := plantillaPrueba{ID: id, Documento: datos}
		var doc struct {
			Operaciones []struct{ Tipo, Recurso string } `json:"operaciones"`
		}
		json.Unmarshal(datos, &doc)
		for _, op := range doc.Operaciones {
			if op.Tipo == "imagen" && op.Recurso != "" {
				if _, err := a.op.Almacen.Recursos.Obtener(ctx, op.Recurso); errors.Is(err, almacen.ErrNoEncontrado) {
					p.Faltan = append(p.Faltan, op.Recurso)
				}
			}
		}
		lista = append(lista, p)
	}
	responder(w, lista)
}

// documentoJSON escribe operaciones del puente como un documento v1 (sin id: lo pone el panel al
// enviarlo). Los tipos del contrato no llevan "tipo" ni omiten los textos vacíos, que el esquema
// rechazaría en campos con valores fijos.
func documentoJSON(impresora, referencia string, ops []contrato.Operacion) json.RawMessage {
	lista := make([]map[string]any, 0, len(ops))
	for _, op := range ops {
		var m map[string]any
		datos, _ := json.Marshal(op)
		json.Unmarshal(datos, &m)
		for k, v := range m {
			if v == "" {
				delete(m, k)
			}
		}
		m["tipo"] = op.Tipo()
		lista = append(lista, m)
	}
	datos, _ := json.Marshal(map[string]any{"version": "1", "impresora": impresora, "referencia": referencia, "operaciones": lista})
	return datos
}

// documentoDePrueba valida el documento del cuerpo, obtiene su impresora y lo convierte a bytes.
func (a *Admin) documentoDePrueba(w http.ResponseWriter, r *http.Request) (*contrato.Documento, *almacen.Impresora, []byte, []escpos.Advertencia, bool) {
	cuerpo, err := io.ReadAll(http.MaxBytesReader(w, r.Body, servidor.MaxBytesCuerpo))
	if err != nil {
		a.error(w, r, err)
		return nil, nil, nil, nil, false
	}
	doc, err := contrato.Decodificar(cuerpo)
	if err != nil {
		a.error(w, r, err)
		return nil, nil, nil, nil, false
	}
	imp, ok2 := a.impresoraActiva(w, r, doc.Impresora)
	if !ok2 {
		return nil, nil, nil, nil, false
	}
	datos, advs, err := escpos.Renderizar(*doc, *imp, escpos.RecursosAlmacen{Ctx: r.Context(), Repo: a.op.Almacen.Recursos})
	if err != nil {
		a.error(w, r, err)
		return nil, nil, nil, nil, false
	}
	return doc, imp, datos, advs, true
}

// POST /admin/api/pruebas/vista-previa: valida y dibuja el documento al ancho de su impresora, sin imprimir.
func (a *Admin) vistaPreviaPrueba(w http.ResponseWriter, r *http.Request) {
	_, imp, datos, advs, ok2 := a.documentoDePrueba(w, r)
	if !ok2 {
		return
	}
	if advs == nil {
		advs = []escpos.Advertencia{}
	}
	pagina := dev.Simular(datos, *imp)
	responder(w, map[string]any{"impresora": imp.Alias, "advertencias": advs, "bytes": len(datos),
		"imagen": "data:image/png;base64," + base64.StdEncoding.EncodeToString(dev.RasterPNG(pagina, imp.AnchoMM))})
}

// POST /admin/api/pruebas/imprimir: imprime el documento en su impresora (física o virtual).
func (a *Admin) imprimirDocumentoPrueba(w http.ResponseWriter, r *http.Request) {
	doc, imp, datos, advs, ok2 := a.documentoDePrueba(w, r)
	if !ok2 {
		return
	}
	if a.imprimirPrueba(w, r, imp, doc.Referencia, datos, advs) {
		a.marcarPaso(r, almacen.AjustePrimeraPrueba)
	}
}

// datosPermisos es la situación que describe el comprobador de permisos del panel.
type datosPermisos struct {
	ClienteID   string   `json:"cliente_id"`
	Origen      string   `json:"origen"`
	ConLlave    bool     `json:"con_llave"`
	Firmado     bool     `json:"firmado"`
	Impresora   string   `json:"impresora"`
	Operaciones []string `json:"operaciones"`
}

// PasoDisponible es el último paso del comprobador: después de autorizar, /v1/imprimir exige que
// la impresora exista y esté activa.
const PasoDisponible = "disponible"

// POST /admin/api/pruebas/permisos: «¿Por qué mi app no puede imprimir?». Simula, sin imprimir ni
// registrar nada, los pasos de autorización de /v1/imprimir para la situación descrita. No pide la
// llave ni la firma de la aplicación: supone que son las correctas.
func (a *Admin) comprobarPermisos(w http.ResponseWriter, r *http.Request) {
	var d datosPermisos
	if err := leerJSON(w, r, &d); err != nil {
		a.error(w, r, err)
		return
	}
	ctx := r.Context()
	c, err := a.op.Servidor.Autorizador().Comprobar(ctx, seguridad.Situacion{ClienteID: d.ClienteID, Origen: d.Origen,
		ConLlave: d.ConLlave, Firmado: d.Firmado, Impresora: d.Impresora, Operaciones: d.Operaciones})
	if err != nil {
		a.error(w, r, err)
		return
	}
	if c.Codigo == "" {
		imp, err := a.op.Almacen.Impresoras.Obtener(ctx, d.Impresora)
		switch {
		case errors.Is(err, almacen.ErrNoEncontrado) || (err == nil && !imp.Activa):
			c.Pasos = append(c.Pasos, seguridad.Paso{ID: PasoDisponible, Estado: seguridad.PasoFalla, Codigo: conector.ErrImpresoraNoEncontrada,
				Mensaje: fmt.Sprintf("No hay ninguna impresora activa con el alias %q.", d.Impresora), Sugerencia: "Consulta GET /v1/impresoras para ver las impresoras disponibles."})
			c.Codigo, c.Estado = conector.ErrImpresoraNoEncontrada, http.StatusNotFound
		case err != nil:
			a.error(w, r, err)
			return
		default:
			c.Pasos = append(c.Pasos, seguridad.Paso{ID: PasoDisponible, Estado: seguridad.PasoOK})
		}
	}
	resp := map[string]any{"pasos": c.Pasos, "codigo": c.Codigo, "estado_http": c.Estado}
	if c.Identificado != nil {
		resp["identificado"] = map[string]string{"id": c.Identificado.ID, "nombre": c.Identificado.Nombre}
	}
	if c.DuenioOrigen != nil {
		resp["origen_de"] = map[string]string{"id": c.DuenioOrigen.ID, "nombre": c.DuenioOrigen.Nombre}
	}
	responder(w, resp)
}
