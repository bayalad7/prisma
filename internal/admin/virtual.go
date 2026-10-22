package admin

import (
	"net/http"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/conector"
	"github.com/bayalad7/prisma/internal/dev"
)

// Rollo de las impresoras virtuales: los tickets que recibieron, guardados solo en la memoria del
// puente (conector.RollosVirtuales). El panel es el único lugar donde se pueden ver; nunca pasan
// por la base ni por la bitácora.

type ticketJSON struct {
	TrabajoID     string   `json:"trabajo_id"`
	Recibido      string   `json:"recibido"`
	Bytes         int      `json:"bytes"`
	ClienteID     string   `json:"cliente_id"`
	ClienteNombre string   `json:"cliente_nombre"`
	Referencia    string   `json:"referencia"`
	Advertencias  []string `json:"advertencias"`
}

// impresoraVirtual exige que la impresora exista y sea virtual.
func (a *Admin) impresoraVirtual(w http.ResponseWriter, r *http.Request) (*almacen.Impresora, bool) {
	imp, ok2 := a.impresora(w, r)
	if ok2 && imp.Tipo != almacen.ConexionVirtual {
		a.error(w, r, rechazo(http.StatusConflict, "IMPRESORA_NO_VIRTUAL", "Esa impresora no es virtual: sus tickets salen en papel.",
			"Solo las impresoras virtuales guardan los tickets para verlos en el panel."))
		return nil, false
	}
	return imp, ok2
}

// GET /admin/api/impresoras/{alias}/rollo: los tickets del más reciente al más antiguo, con la
// aplicación y la referencia del trabajo (del historial).
func (a *Admin) listarRollo(w http.ResponseWriter, r *http.Request) {
	imp, ok2 := a.impresoraVirtual(w, r)
	if !ok2 {
		return
	}
	nombres := map[string]string{}
	if clientes, err := a.op.Almacen.Clientes.Listar(r.Context()); err == nil {
		for _, c := range clientes {
			nombres[c.ID] = c.Nombre
		}
	}
	resp := []ticketJSON{}
	for _, t := range conector.RollosVirtuales.Listar(imp.Alias) {
		j := ticketJSON{TrabajoID: t.TrabajoID, Recibido: fecha(t.Recibido), Bytes: len(t.Datos), Advertencias: []string{}}
		if trabajo, err := a.op.Almacen.Trabajos.Obtener(r.Context(), t.TrabajoID); err == nil {
			j.ClienteID, j.Referencia = trabajo.ClienteID, trabajo.Referencia
			if trabajo.Advertencias != nil {
				j.Advertencias = trabajo.Advertencias
			}
			if j.ClienteNombre = nombres[trabajo.ClienteID]; j.ClienteNombre == "" {
				j.ClienteNombre = nombresEspeciales[trabajo.ClienteID]
			}
		}
		resp = append(resp, j)
	}
	responder(w, resp)
}

// ticketVirtual busca el ticket {trabajo} en el rollo de la impresora {alias}.
func (a *Admin) ticketVirtual(w http.ResponseWriter, r *http.Request) (*almacen.Impresora, conector.TicketVirtual, bool) {
	imp, ok2 := a.impresoraVirtual(w, r)
	if !ok2 {
		return nil, conector.TicketVirtual{}, false
	}
	t, encontrado := conector.RollosVirtuales.Ticket(imp.Alias, r.PathValue("trabajo"))
	if !encontrado {
		a.error(w, r, rechazo(http.StatusNotFound, "TICKET_NO_ENCONTRADO", "Ese ticket ya no está en el rollo de la impresora virtual.",
			"La impresora recuerda solo los últimos tickets y los olvida al reiniciar el puente."))
		return nil, t, false
	}
	return imp, t, true
}

// GET …/rollo/{trabajo}/imagen: el ticket como saldría en el papel (PNG a 203 ppp).
func (a *Admin) imagenTicket(w http.ResponseWriter, r *http.Request) {
	imp, t, ok2 := a.ticketVirtual(w, r)
	if !ok2 {
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, no-store")
	if r.URL.Query().Get("descargar") != "" {
		w.Header().Set("Content-Disposition", `attachment; filename="`+imp.Alias+"-"+t.TrabajoID+`.png"`)
	}
	w.Write(dev.RasterPNG(dev.Simular(t.Datos, *imp), imp.AnchoMM))
}

// GET …/rollo/{trabajo}/bin: los bytes ESC/POS exactos (para «prisma dev decodificar»).
func (a *Admin) binTicket(w http.ResponseWriter, r *http.Request) {
	imp, t, ok2 := a.ticketVirtual(w, r)
	if !ok2 {
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Disposition", `attachment; filename="`+imp.Alias+"-"+t.TrabajoID+`.bin"`)
	w.Write(t.Datos)
}

// GET …/rollo/{trabajo}/traza: los comandos ESC/POS en texto legible.
func (a *Admin) trazaTicket(w http.ResponseWriter, r *http.Request) {
	_, t, ok2 := a.ticketVirtual(w, r)
	if !ok2 {
		return
	}
	responder(w, map[string]string{"traza": dev.Traza(t.Datos)})
}

// DELETE /admin/api/impresoras/{alias}/rollo: vacía el rollo.
func (a *Admin) vaciarRollo(w http.ResponseWriter, r *http.Request) {
	imp, ok2 := a.impresoraVirtual(w, r)
	if !ok2 {
		return
	}
	conector.RollosVirtuales.Vaciar(imp.Alias)
	a.op.Log.Info("rollo de impresora virtual vaciado desde el panel", "alias", imp.Alias)
	responder(w, ok)
}
