package admin

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/seguridad"
)

// Operaciones que el dueño habilita por aplicación.
var operacionesSensibles = []string{"abrir_cajon", "raw"}

type clienteJSON struct {
	ID                   string   `json:"id"`
	Nombre               string   `json:"nombre"`
	Nivel                int      `json:"nivel"`
	LlavePrefijo         *string  `json:"llave_prefijo"`
	ExigirLlave          bool     `json:"exigir_llave"`
	Origenes             []string `json:"origenes"`
	TodasLasImpresoras   bool     `json:"todas_las_impresoras"`
	Impresoras           []string `json:"impresoras"`
	OperacionesSensibles []string `json:"operaciones_sensibles"`
	Activo               bool     `json:"activo"`
	HuellaPublica        string   `json:"huella_publica,omitempty"`
}

// datosCliente es lo que envía el formulario de aplicaciones.
type datosCliente struct {
	Nombre               string   `json:"nombre"`
	Nivel                int      `json:"nivel"`
	Origenes             []string `json:"origenes"`
	TodasLasImpresoras   bool     `json:"todas_las_impresoras"`
	Impresoras           []string `json:"impresoras"`
	OperacionesSensibles []string `json:"operaciones_sensibles"`
	LlavePublica         string   `json:"llave_publica"`
	ExigirLlave          bool     `json:"exigir_llave"`
	Activo               *bool    `json:"activo"`
}

const alfabetoID = "abcdefghijklmnopqrstuvwxyz0123456789"

func nuevoIDCliente() string {
	b := make([]byte, 16)
	rand.Read(b)
	for i := range b {
		b[i] = alfabetoID[int(b[i])%len(alfabetoID)]
	}
	return "cli_" + string(b)
}

func (a *Admin) clienteAJSON(ctx context.Context, c almacen.Cliente) (clienteJSON, error) {
	j := clienteJSON{ID: c.ID, Nombre: c.Nombre, Nivel: c.Nivel, ExigirLlave: c.ExigirLlave, TodasLasImpresoras: c.TodasLasImpresoras,
		OperacionesSensibles: c.OperacionesSensibles, Activo: c.Activo, Impresoras: []string{}}
	if c.LlavePrefijo != "" {
		j.LlavePrefijo = &c.LlavePrefijo
	}
	if j.OperacionesSensibles == nil {
		j.OperacionesSensibles = []string{}
	}
	if len(c.LlavePublica) > 0 {
		j.HuellaPublica = seguridad.HuellaLlavePublica(c.LlavePublica)
	}
	var err error
	if j.Origenes, err = a.op.Almacen.Clientes.Origenes(ctx, c.ID); err != nil {
		return j, err
	}
	if j.Origenes == nil {
		j.Origenes = []string{}
	}
	if !c.TodasLasImpresoras {
		if j.Impresoras, err = a.op.Almacen.Clientes.ImpresorasPermitidas(ctx, c.ID); err != nil {
			return j, err
		}
		if j.Impresoras == nil {
			j.Impresoras = []string{}
		}
	}
	return j, nil
}

func (a *Admin) listarClientes(w http.ResponseWriter, r *http.Request) {
	lista, err := a.op.Almacen.Clientes.Listar(r.Context())
	if err != nil {
		a.error(w, r, err)
		return
	}
	resp := []clienteJSON{}
	for _, c := range lista {
		j, err := a.clienteAJSON(r.Context(), c)
		if err != nil {
			a.error(w, r, err)
			return
		}
		resp = append(resp, j)
	}
	responder(w, resp)
}

func errLlavePublica() error {
	return rechazo(http.StatusBadRequest, "LLAVE_PUBLICA_INVALIDA",
		"No es una llave pública Ed25519: pega el bloque PEM completo o los 32 bytes en base64.",
		"Genérala en el backend de la aplicación (para pruebas: «prisma dev llaves --salida carpeta»).")
}

func (a *Admin) validarLlavePublica(w http.ResponseWriter, r *http.Request) {
	var d struct{ Llave string }
	if err := leerJSON(w, r, &d); err != nil {
		a.error(w, r, err)
		return
	}
	v := strings.TrimSpace(d.Llave)
	k, err := seguridad.LeerLlavePublica([]byte(v))
	if err != nil {
		a.error(w, r, errLlavePublica())
		return
	}
	formato := "base64"
	if strings.HasPrefix(v, "-----BEGIN") {
		formato = "PEM"
	}
	responder(w, map[string]any{"ok": true, "formato": formato, "huella": seguridad.HuellaLlavePublica(k)})
}

// normalizar valida el formulario y deja los orígenes en su forma canónica.
func (a *Admin) normalizar(ctx context.Context, d *datosCliente, existente *almacen.Cliente) error {
	d.Nombre = strings.TrimSpace(d.Nombre)
	if d.Nombre == "" || len([]rune(d.Nombre)) > 80 {
		return invalido("Escribe el nombre de la aplicación (hasta 80 caracteres).")
	}
	if d.Nivel < 1 || d.Nivel > 3 {
		return invalido("El nivel de seguridad debe ser 1, 2 o 3.")
	}
	var origenes []string
	for _, o := range d.Origenes {
		n, err := almacen.NormalizarOrigen(o)
		if err != nil {
			return invalido(fmt.Sprintf("«%s» no es un origen válido: escribe esquema, host y puerto, por ejemplo https://pos.mitienda.mx.", o))
		}
		if !slices.Contains(origenes, n) {
			origenes = append(origenes, n)
		}
		if otro, err := a.op.Almacen.Clientes.ObtenerPorOrigen(ctx, n); err == nil && (existente == nil || otro.ID != existente.ID) {
			return rechazo(http.StatusConflict, "ORIGEN_DUPLICADO", fmt.Sprintf("El origen %s ya pertenece a «%s».", n, otro.Nombre),
				"Un origen solo puede pertenecer a una aplicación; quítalo de la otra primero.")
		}
	}
	d.Origenes = origenes
	if d.Nivel == 1 && len(d.Origenes) == 0 {
		return invalido("El nivel 1 se identifica solo por el origen: agrega al menos uno.")
	}
	for _, op := range d.OperacionesSensibles {
		if !slices.Contains(operacionesSensibles, op) {
			return invalido(fmt.Sprintf("Operación sensible desconocida: %s.", op))
		}
	}
	if !d.TodasLasImpresoras {
		for _, alias := range d.Impresoras {
			if _, err := a.op.Almacen.Impresoras.Obtener(ctx, alias); err != nil {
				return invalido(fmt.Sprintf("La impresora «%s» ya no existe.", alias))
			}
		}
	}
	if d.Nivel != 3 {
		d.ExigirLlave, d.LlavePublica = false, ""
	}
	return nil
}

// nuevaLlave genera la llave pk_ y guarda su hash en el cliente; la llave solo se devuelve una vez.
func nuevaLlave(c *almacen.Cliente) (string, error) {
	llave, prefijo, err := seguridad.GenerarLlave()
	if err != nil {
		return "", err
	}
	hash, err := seguridad.HashLlave(llave)
	if err != nil {
		return "", err
	}
	c.LlaveHash, c.LlavePrefijo = hash, prefijo
	return llave, nil
}

// guardarRelaciones deja los orígenes y las impresoras como en el formulario.
func (a *Admin) guardarRelaciones(ctx context.Context, id string, d datosCliente) error {
	actuales, err := a.op.Almacen.Clientes.Origenes(ctx, id)
	if err != nil {
		return err
	}
	for _, o := range actuales {
		if !slices.Contains(d.Origenes, o) {
			if err := a.op.Almacen.Clientes.QuitarOrigen(ctx, id, o); err != nil {
				return err
			}
		}
	}
	for _, o := range d.Origenes {
		if !slices.Contains(actuales, o) {
			if err := a.op.Almacen.Clientes.AgregarOrigen(ctx, id, o); err != nil {
				return err
			}
			// Si el origen esperaba aprobación, ya no hace falta.
			a.op.Almacen.Solicitudes.Eliminar(ctx, o)
		}
	}
	return a.op.Almacen.Clientes.EstablecerImpresoras(ctx, id, d.TodasLasImpresoras, d.Impresoras)
}

func (a *Admin) crearCliente(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var d datosCliente
	if err := leerJSON(w, r, &d); err != nil {
		a.error(w, r, err)
		return
	}
	if err := a.normalizar(ctx, &d, nil); err != nil {
		a.error(w, r, err)
		return
	}
	c := almacen.Cliente{ID: nuevoIDCliente(), Nombre: d.Nombre, Nivel: d.Nivel, ExigirLlave: d.ExigirLlave,
		OperacionesSensibles: d.OperacionesSensibles, TodasLasImpresoras: d.TodasLasImpresoras, Activo: d.Activo == nil || *d.Activo}
	if d.Nivel == 3 {
		k, err := seguridad.LeerLlavePublica([]byte(strings.TrimSpace(d.LlavePublica)))
		if err != nil {
			a.error(w, r, errLlavePublica())
			return
		}
		c.LlavePublica = k
	}
	var llave string
	if d.Nivel == 2 || c.ExigirLlave {
		var err error
		if llave, err = nuevaLlave(&c); err != nil {
			a.error(w, r, err)
			return
		}
	}
	a.crearYResponder(w, r, c, d, llave)
}

func (a *Admin) crearYResponder(w http.ResponseWriter, r *http.Request, c almacen.Cliente, d datosCliente, llave string) {
	ctx := r.Context()
	if err := a.op.Almacen.Clientes.Crear(ctx, &c); err != nil {
		a.error(w, r, err)
		return
	}
	if err := a.guardarRelaciones(ctx, c.ID, d); err != nil {
		a.op.Almacen.Clientes.Eliminar(context.WithoutCancel(ctx), c.ID)
		a.error(w, r, err)
		return
	}
	a.op.Log.Info("aplicación creada desde el panel", "cliente", c.ID, "nivel", c.Nivel)
	a.responderCliente(w, r, c.ID, llave)
}

func (a *Admin) responderCliente(w http.ResponseWriter, r *http.Request, id, llave string) {
	c, err := a.op.Almacen.Clientes.Obtener(r.Context(), id)
	if err != nil {
		a.error(w, r, err)
		return
	}
	j, err := a.clienteAJSON(r.Context(), *c)
	if err != nil {
		a.error(w, r, err)
		return
	}
	resp := map[string]any{"ok": true, "cliente": j}
	if llave != "" {
		resp["llave"] = llave
	}
	responder(w, resp)
}

func (a *Admin) cliente(w http.ResponseWriter, r *http.Request) (*almacen.Cliente, bool) {
	c, err := a.op.Almacen.Clientes.Obtener(r.Context(), r.PathValue("id"))
	if errors.Is(err, almacen.ErrNoEncontrado) {
		err = rechazo(http.StatusNotFound, "CLIENTE_NO_ENCONTRADO", "La aplicación ya no existe.", "Recarga la lista de aplicaciones.")
	}
	if err != nil {
		a.error(w, r, err)
		return nil, false
	}
	return c, true
}

func (a *Admin) actualizarCliente(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	c, ok2 := a.cliente(w, r)
	if !ok2 {
		return
	}
	var d datosCliente
	if err := leerJSON(w, r, &d); err != nil {
		a.error(w, r, err)
		return
	}
	if err := a.normalizar(ctx, &d, c); err != nil {
		a.error(w, r, err)
		return
	}
	c.Nombre, c.Nivel, c.ExigirLlave = d.Nombre, d.Nivel, d.ExigirLlave
	c.OperacionesSensibles, c.TodasLasImpresoras = d.OperacionesSensibles, d.TodasLasImpresoras
	if d.Activo != nil {
		c.Activo = *d.Activo
	}
	switch {
	case d.Nivel != 3:
		c.LlavePublica = nil
	case d.LlavePublica != "":
		k, err := seguridad.LeerLlavePublica([]byte(strings.TrimSpace(d.LlavePublica)))
		if err != nil {
			a.error(w, r, errLlavePublica())
			return
		}
		c.LlavePublica = k
	case len(c.LlavePublica) == 0:
		a.error(w, r, invalido("El nivel 3 necesita la llave pública."))
		return
	}
	// Al pasar a nivel 2 (o exigir llave en el 3) sin llave, se genera una y se muestra una vez.
	var llave string
	if (d.Nivel == 2 || c.ExigirLlave) && c.LlaveHash == "" {
		var err error
		if llave, err = nuevaLlave(c); err != nil {
			a.error(w, r, err)
			return
		}
	}
	if err := a.op.Almacen.Clientes.Actualizar(ctx, c); err != nil {
		a.error(w, r, err)
		return
	}
	if err := a.guardarRelaciones(ctx, c.ID, d); err != nil {
		a.error(w, r, err)
		return
	}
	a.op.Log.Info("aplicación actualizada desde el panel", "cliente", c.ID, "nivel", c.Nivel, "activo", c.Activo)
	a.responderCliente(w, r, c.ID, llave)
}

func (a *Admin) eliminarCliente(w http.ResponseWriter, r *http.Request) {
	c, ok2 := a.cliente(w, r)
	if !ok2 {
		return
	}
	if err := a.op.Almacen.Clientes.Eliminar(r.Context(), c.ID); err != nil {
		a.error(w, r, err)
		return
	}
	a.op.Log.Info("aplicación eliminada desde el panel", "cliente", c.ID)
	responder(w, ok)
}

// rotarLlave reemplaza la llave: la anterior deja de funcionar en ese momento.
func (a *Admin) rotarLlave(w http.ResponseWriter, r *http.Request) {
	c, ok2 := a.cliente(w, r)
	if !ok2 {
		return
	}
	if c.Nivel == 1 {
		a.error(w, r, invalido("El nivel 1 no usa llave."))
		return
	}
	llave, err := nuevaLlave(c)
	if err != nil {
		a.error(w, r, err)
		return
	}
	if c.Nivel == 3 {
		c.ExigirLlave = true
	}
	if err := a.op.Almacen.Clientes.Actualizar(r.Context(), c); err != nil {
		a.error(w, r, err)
		return
	}
	a.op.Log.Info("llave de aplicación regenerada desde el panel", "cliente", c.ID)
	responder(w, map[string]any{"ok": true, "llave": llave})
}

// revocarLlave quita la llave. En el nivel 2 la aplicación no imprime hasta tener otra; en el 3
// sigue imprimiendo con documentos firmados.
func (a *Admin) revocarLlave(w http.ResponseWriter, r *http.Request) {
	c, ok2 := a.cliente(w, r)
	if !ok2 {
		return
	}
	c.LlaveHash, c.LlavePrefijo, c.ExigirLlave = "", "", false
	if err := a.op.Almacen.Clientes.Actualizar(r.Context(), c); err != nil {
		a.error(w, r, err)
		return
	}
	a.op.Log.Info("llave de aplicación revocada desde el panel", "cliente", c.ID)
	responder(w, ok)
}

// --- Solicitudes de origen ----------------------------------------------------------------------

type solicitudJSON struct {
	Origen     string `json:"origen"`
	PrimeraVez string `json:"primera_vez"`
	UltimaVez  string `json:"ultima_vez"`
	Intentos   int    `json:"intentos"`
	Estado     string `json:"estado"`
}

func (a *Admin) listarSolicitudes(w http.ResponseWriter, r *http.Request) {
	lista, err := a.op.Almacen.Solicitudes.Listar(r.Context(), "")
	if err != nil {
		a.error(w, r, err)
		return
	}
	resp := []solicitudJSON{}
	for _, s := range lista {
		resp = append(resp, solicitudJSON{Origen: s.Origen, PrimeraVez: fecha(s.PrimeraVez), UltimaVez: fecha(s.UltimaVez),
			Intentos: s.Intentos, Estado: s.Estado})
	}
	responder(w, resp)
}

func (a *Admin) aprobarSolicitud(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var d struct {
		Origen    string `json:"origen"`
		ClienteID string `json:"cliente_id"`
		Nuevo     *struct {
			Nombre string `json:"nombre"`
			Nivel  int    `json:"nivel"`
		} `json:"nuevo"`
	}
	if err := leerJSON(w, r, &d); err != nil {
		a.error(w, r, err)
		return
	}
	if _, err := a.op.Almacen.Solicitudes.Obtener(ctx, d.Origen); err != nil {
		if errors.Is(err, almacen.ErrNoEncontrado) || errors.Is(err, almacen.ErrOrigenInvalido) {
			err = rechazo(http.StatusNotFound, "SOLICITUD_NO_ENCONTRADA", "Esa solicitud ya no existe.", "Recarga la lista.")
		}
		a.error(w, r, err)
		return
	}
	if d.Nuevo == nil {
		c, err := a.op.Almacen.Clientes.Obtener(ctx, d.ClienteID)
		if err != nil {
			a.error(w, r, rechazo(http.StatusNotFound, "CLIENTE_NO_ENCONTRADO", "La aplicación elegida ya no existe.", "Recarga la página."))
			return
		}
		if err := a.op.Almacen.Solicitudes.Aprobar(ctx, d.Origen, c.ID); err != nil {
			a.error(w, r, err)
			return
		}
		a.op.Log.Info("origen aprobado desde el panel", "cliente", c.ID)
		a.responderCliente(w, r, c.ID, "")
		return
	}
	// Aplicación nueva: todas las impresoras y sin operaciones sensibles (se ajusta después).
	if d.Nuevo.Nivel != 1 && d.Nuevo.Nivel != 2 {
		a.error(w, r, invalido("Al aprobar un origen, la aplicación nueva es de nivel 1 o 2; el nivel 3 se configura en Aplicaciones."))
		return
	}
	nombre := strings.TrimSpace(d.Nuevo.Nombre)
	if nombre == "" {
		nombre = d.Origen
	}
	c := almacen.Cliente{ID: nuevoIDCliente(), Nombre: nombre, Nivel: d.Nuevo.Nivel, TodasLasImpresoras: true, Activo: true}
	var llave string
	if c.Nivel == 2 {
		var err error
		if llave, err = nuevaLlave(&c); err != nil {
			a.error(w, r, err)
			return
		}
	}
	if err := a.op.Almacen.Clientes.Crear(ctx, &c); err != nil {
		a.error(w, r, err)
		return
	}
	if err := a.op.Almacen.Solicitudes.Aprobar(ctx, d.Origen, c.ID); err != nil {
		a.op.Almacen.Clientes.Eliminar(context.WithoutCancel(ctx), c.ID)
		a.error(w, r, err)
		return
	}
	a.op.Log.Info("origen aprobado con una aplicación nueva", "cliente", c.ID, "nivel", c.Nivel)
	a.responderCliente(w, r, c.ID, llave)
}

func (a *Admin) cambiarSolicitud(w http.ResponseWriter, r *http.Request, accion func(context.Context, string) error) {
	var d struct{ Origen string }
	if err := leerJSON(w, r, &d); err != nil {
		a.error(w, r, err)
		return
	}
	if err := accion(r.Context(), d.Origen); err != nil {
		if errors.Is(err, almacen.ErrNoEncontrado) || errors.Is(err, almacen.ErrOrigenInvalido) {
			err = rechazo(http.StatusNotFound, "SOLICITUD_NO_ENCONTRADA", "Esa solicitud ya no existe.", "Recarga la lista.")
		}
		a.error(w, r, err)
		return
	}
	responder(w, ok)
}

func (a *Admin) rechazarSolicitud(w http.ResponseWriter, r *http.Request) {
	a.cambiarSolicitud(w, r, a.op.Almacen.Solicitudes.Rechazar)
}

func (a *Admin) olvidarSolicitud(w http.ResponseWriter, r *http.Request) {
	a.cambiarSolicitud(w, r, a.op.Almacen.Solicitudes.Eliminar)
}
