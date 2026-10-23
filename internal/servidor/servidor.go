// Package servidor expone la API pública /v1 por HTTPS: autoriza cada petición (origen, llave,
// firma, idempotencia y permisos, en ese orden), valida el documento, lo renderiza y lo encola.
package servidor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/cola"
	"github.com/bayalad7/prisma/internal/conector"
	"github.com/bayalad7/prisma/internal/config"
	"github.com/bayalad7/prisma/internal/contrato"
	"github.com/bayalad7/prisma/internal/seguridad"
)

// MaxBytesCuerpo es el máximo que se lee de una petición. El contrato limita el documento a 256 KB;
// el margen permite responder DOCUMENTO_DEMASIADO_GRANDE con el tamaño exacto.
const MaxBytesCuerpo = 300 * 1024

// AjusteNombrePuente es el nombre que muestra GET /v1/salud.
const AjusteNombrePuente = "puente.nombre"

// Opciones de construcción del servidor.
type Opciones struct {
	Almacen *almacen.Almacen
	Cola    *cola.Cola
	Limites config.Limites
	Version string
	Log     *slog.Logger

	// TiempoRespuesta es cuánto espera POST /v1/imprimir el resultado de la impresora antes de
	// responder 202 con el trabajo en cola. 60 s por defecto.
	TiempoRespuesta time.Duration
	// NuevoConector crea el conector de una impresora (conector.Nuevo por defecto).
	NuevoConector func(almacen.Impresora) (conector.Conector, error)
}

// Servidor atiende la API pública.
type Servidor struct {
	op         Opciones
	aut        *seguridad.Autorizador
	porIP      *seguridad.Limitador
	porCliente *seguridad.Limitador
	porPanel   *seguridad.Limitador

	mu          sync.Mutex
	registradas map[string]time.Time // alias → versión del perfil registrada en la cola
	panel       http.Handler
	iniciado    time.Time
}

// LimitePanelPorMinuto es el límite por IP del panel (/admin), aparte del de la API: abrir el panel
// cuesta una decena de peticiones y no debe restarle a una aplicación que imprime desde el mismo
// equipo, ni al revés. Es fijo; el inicio de sesión tiene además su propio bloqueo por intentos.
const LimitePanelPorMinuto = 600

// Ajustes de la base de datos que usa el servidor. Mandan sobre los valores de config.toml.
const (
	AjusteLimiteIP      = "limites.ip_por_minuto"
	AjusteLimiteCliente = "limites.cliente_por_minuto"
)

// Nuevo crea el servidor.
func Nuevo(op Opciones) *Servidor {
	if op.Log == nil {
		op.Log = slog.New(slog.DiscardHandler)
	}
	if op.TiempoRespuesta <= 0 {
		op.TiempoRespuesta = 60 * time.Second
	}
	if op.NuevoConector == nil {
		op.NuevoConector = conector.Nuevo
	}
	return &Servidor{
		op: op, aut: seguridad.NuevoAutorizador(op.Almacen),
		porIP: seguridad.NuevoLimitador(op.Limites.PorIP), porCliente: seguridad.NuevoLimitador(op.Limites.PorCliente),
		porPanel:    seguridad.NuevoLimitador(LimitePanelPorMinuto),
		registradas: map[string]time.Time{}, iniciado: time.Now(),
	}
}

// MontarPanel sirve el panel de administración en /admin con las mismas protecciones (cabeceras y
// bitácora) que la API y su propio límite por IP (LimitePanelPorMinuto).
func (s *Servidor) MontarPanel(h http.Handler) { s.panel = h }

// Iniciado es el momento en que arrancó el servidor.
func (s *Servidor) Iniciado() time.Time { return s.iniciado }

// CargarAjustes aplica los límites guardados en la base (el panel los edita); si no hay, quedan
// los de config.toml.
func (s *Servidor) CargarAjustes(ctx context.Context) error {
	for clave, lim := range map[string]*seguridad.Limitador{AjusteLimiteIP: s.porIP, AjusteLimiteCliente: s.porCliente} {
		v, ok, err := s.op.Almacen.Ajustes.Obtener(ctx, clave)
		if err != nil {
			return err
		}
		if n, err := strconv.Atoi(v); ok && err == nil && n >= 0 {
			lim.Configurar(n)
		}
	}
	return nil
}

// ActualizarLimites cambia los límites en caliente (no los guarda).
func (s *Servidor) ActualizarLimites(porIP, porCliente int) {
	s.porIP.Configurar(porIP)
	s.porCliente.Configurar(porCliente)
}

// Autorizador permite ajustar el reloj en las pruebas.
func (s *Servidor) Autorizador() *seguridad.Autorizador { return s.aut }

// Limitadores permite ajustar el reloj de los límites en las pruebas.
func (s *Servidor) Limitadores() (porIP, porCliente *seguridad.Limitador) {
	return s.porIP, s.porCliente
}

// LimitadorPanel permite ajustar el límite del panel en las pruebas.
func (s *Servidor) LimitadorPanel() *seguridad.Limitador { return s.porPanel }

// esPanel dice si la petición es del panel de administración montado en /admin.
func (s *Servidor) esPanel(r *http.Request) bool {
	return s.panel != nil && (r.URL.Path == "/admin" || strings.HasPrefix(r.URL.Path, "/admin/"))
}

// RegistrarImpresoras agrega a la cola todas las impresoras activas (al arrancar).
func (s *Servidor) RegistrarImpresoras(ctx context.Context) error {
	lista, err := s.op.Almacen.Impresoras.Listar(ctx)
	if err != nil {
		return err
	}
	for _, imp := range lista {
		if imp.Activa {
			if err := s.AsegurarRegistrada(imp); err != nil {
				s.op.Log.Warn("no se pudo preparar la impresora", "alias", imp.Alias, "error", err.Error())
			}
		}
	}
	return nil
}

// AsegurarRegistrada registra la impresora en la cola, o actualiza su conector si el perfil
// cambió desde que se registró (desde el panel o la CLI con el puente en marcha).
func (s *Servidor) AsegurarRegistrada(imp almacen.Impresora) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.registradas[imp.Alias]; ok && v.Equal(imp.Actualizada) {
		return nil
	}
	con, err := s.op.NuevoConector(imp)
	if err != nil {
		return err
	}
	s.op.Cola.Registrar(imp.Alias, con)
	s.registradas[imp.Alias] = imp.Actualizada
	return nil
}

// QuitarImpresora la saca de la cola (al eliminarla o cambiarle el alias).
func (s *Servidor) QuitarImpresora(alias string) {
	s.mu.Lock()
	delete(s.registradas, alias)
	s.mu.Unlock()
	s.op.Cola.Quitar(alias)
}

// SincronizarImpresoras deja la cola igual que la base: quita las que ya no existen y vuelve a
// registrar las activas (por ejemplo, después de restaurar un respaldo).
func (s *Servidor) SincronizarImpresoras(ctx context.Context) error {
	lista, err := s.op.Almacen.Impresoras.Listar(ctx)
	if err != nil {
		return err
	}
	existen := map[string]bool{}
	for _, imp := range lista {
		existen[imp.Alias] = true
	}
	s.mu.Lock()
	var sobran []string
	for alias := range s.registradas {
		if !existen[alias] {
			sobran = append(sobran, alias)
		}
	}
	clear(s.registradas)
	s.mu.Unlock()
	for _, alias := range sobran {
		s.op.Cola.Quitar(alias)
	}
	return s.RegistrarImpresoras(ctx)
}

// ruta asocia métodos con manejadores para una misma ruta.
type ruta map[string]http.HandlerFunc

// rutas son las de la API pública v1. docs/referencia/openapi.yaml debe listar las mismas (lo
// comprueba la prueba de docs/).
func (s *Servidor) rutas() map[string]ruta {
	return map[string]ruta{
		"/v1/salud":         {http.MethodGet: s.salud},
		"/v1/impresoras":    {http.MethodGet: s.impresoras},
		"/v1/imprimir":      {http.MethodPost: s.imprimir},
		"/v1/previsualizar": {http.MethodPost: s.previsualizar},
	}
}

// RutasPublicas devuelve cada ruta de la API v1 con sus métodos, sin OPTIONS (el preflight de CORS
// lo atienden todas).
func RutasPublicas() map[string][]string {
	var s *Servidor // los métodos solo se nombran, no se llaman
	salida := map[string][]string{}
	for ruta, metodos := range s.rutas() {
		for m := range metodos {
			salida[ruta] = append(salida[ruta], m)
		}
		slices.Sort(salida[ruta])
	}
	return salida
}

// Handler devuelve el manejador HTTP completo: cabeceras de seguridad, límite por IP, bitácora
// y las rutas de la API.
func (s *Servidor) Handler() http.Handler {
	rutas := s.rutas()
	enrutar := func(w http.ResponseWriter, r *http.Request) {
		if s.esPanel(r) {
			s.panel.ServeHTTP(w, r)
			return
		}
		metodos, ok := rutas[r.URL.Path]
		if !ok {
			s.responderError(w, r, rechazo(http.StatusNotFound, "RUTA_NO_ENCONTRADA",
				fmt.Sprintf("No existe la ruta %s.", r.URL.Path), "Las rutas de la API v1 son /v1/salud, /v1/impresoras, /v1/imprimir y /v1/previsualizar."))
			return
		}
		if r.Method == http.MethodOptions {
			s.preflight(w, r)
			return
		}
		h, ok := metodos[r.Method]
		if !ok {
			var permitidos []string
			for m := range metodos {
				permitidos = append(permitidos, m)
			}
			w.Header().Set("Allow", strings.Join(append(permitidos, http.MethodOptions), ", "))
			s.responderError(w, r, rechazo(http.StatusMethodNotAllowed, "METODO_NO_PERMITIDO",
				fmt.Sprintf("%s no acepta el método %s.", r.URL.Path, r.Method), "Usa "+strings.Join(permitidos, " o ")+"."))
			return
		}
		h(w, r)
	}
	return s.cabeceras(s.limitarIP(s.bitacora(http.HandlerFunc(enrutar))))
}

func rechazo(estado int, codigo, mensaje, sugerencia string) *seguridad.Rechazo {
	return &seguridad.Rechazo{Estado: estado, Codigo: codigo, Mensaje: mensaje, Sugerencia: sugerencia}
}

// cabeceras agrega las cabeceras de seguridad a todas las respuestas.
func (s *Servidor) cabeceras(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := w.Header()
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "no-referrer")
		c.Set("Cache-Control", "no-store")
		c.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		c.Add("Vary", "Origin")
		h.ServeHTTP(w, r)
	})
}

func ipDe(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// limitarIP aplica el límite de peticiones por IP antes que cualquier otra cosa: el de la API o,
// para /admin, el del panel. Cada uno lleva su cuenta.
func (s *Servidor) limitarIP(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panel := s.esPanel(r)
		limitador := s.porIP
		if panel {
			limitador = s.porPanel
		}
		if ok, espera := limitador.Permitir(ipDe(r)); !ok {
			// Sin cabeceras CORS, el navegador lo vería como un error de red y el SDK diría «origen
			// no autorizado». La respuesta solo lleva el error, así que puede leerla cualquier origen.
			// El panel es del mismo origen: no las necesita.
			if !panel {
				permitirCORS(w, r.Header.Get("Origin"))
			}
			s.demasiadas(w, r, espera)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (s *Servidor) demasiadas(w http.ResponseWriter, r *http.Request, espera time.Duration) {
	w.Header().Set("Retry-After", strconv.Itoa(int(espera.Seconds())+1))
	s.responderError(w, r, rechazo(http.StatusTooManyRequests, seguridad.ErrDemasiadasPeticiones,
		"Se superó el límite de peticiones por minuto.", "Espera unos segundos y vuelve a intentarlo."))
}

// limitarCliente aplica el límite por cliente una vez identificado.
func (s *Servidor) limitarCliente(w http.ResponseWriter, r *http.Request, cli *almacen.Cliente) bool {
	anotarCliente(r, cli)
	if ok, espera := s.porCliente.Permitir(cli.ID); !ok {
		s.demasiadas(w, r, espera)
		return false
	}
	return true
}

// registro guarda datos de la petición para la bitácora (nunca el contenido ni las llaves).
type registro struct {
	estado  int
	cliente string
}

type claveRegistro struct{}

func anotarCliente(r *http.Request, cli *almacen.Cliente) {
	if reg, ok := r.Context().Value(claveRegistro{}).(*registro); ok && cli != nil {
		reg.cliente = cli.ID
	}
}

type escritorConEstado struct {
	http.ResponseWriter
	reg *registro
}

func (e *escritorConEstado) WriteHeader(estado int) {
	e.reg.estado = estado
	e.ResponseWriter.WriteHeader(estado)
}

func (s *Servidor) bitacora(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inicio := time.Now()
		reg := &registro{estado: http.StatusOK}
		r = r.WithContext(context.WithValue(r.Context(), claveRegistro{}, reg))
		h.ServeHTTP(&escritorConEstado{ResponseWriter: w, reg: reg}, r)
		s.op.Log.Info("petición", "metodo", r.Method, "ruta", r.URL.Path, "estado", reg.estado,
			"duracion_ms", time.Since(inicio).Milliseconds(), "ip", ipDe(r), "cliente", reg.cliente)
	})
}

// permitirCORS devuelve el origen exacto (nunca "*"), solo para orígenes autorizados.
func permitirCORS(w http.ResponseWriter, origen string) {
	if origen != "" {
		w.Header().Set("Access-Control-Allow-Origin", origen)
		w.Header().Set("Access-Control-Expose-Headers", "Retry-After")
	}
}

// preflight responde las peticiones OPTIONS de CORS. Un origen desconocido queda registrado como
// solicitud pendiente y no recibe cabeceras CORS.
func (s *Servidor) preflight(w http.ResponseWriter, r *http.Request) {
	origen := r.Header.Get("Origin")
	if origen == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if _, err := s.aut.ClienteDeOrigen(r.Context(), origen); err != nil {
		s.responderError(w, r, err)
		return
	}
	c := w.Header()
	permitirCORS(w, origen)
	c.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	c.Set("Access-Control-Max-Age", "600")
	if strings.EqualFold(r.Header.Get("Access-Control-Request-Private-Network"), "true") {
		c.Set("Access-Control-Allow-Private-Network", "true")
	}
	w.WriteHeader(http.StatusNoContent)
}

func responderJSON(w http.ResponseWriter, estado int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(estado)
	json.NewEncoder(w).Encode(v)
}

// respuestaError es el formato de todos los errores: {ok:false, codigo, mensaje, sugerencia, …}.
type respuestaError struct {
	OK                  bool               `json:"ok"`
	Codigo              string             `json:"codigo"`
	Mensaje             string             `json:"mensaje"`
	Sugerencia          string             `json:"sugerencia"`
	Detalles            []contrato.Detalle `json:"detalles,omitempty"`
	VersionesSoportadas []string           `json:"versiones_soportadas,omitempty"`
	TrabajoID           string             `json:"trabajo_id,omitempty"`
}

// estadoContrato es el estado HTTP de cada código del contrato (§9).
var estadoContrato = map[string]int{
	contrato.ErrDocumentoDemasiadoGrande: http.StatusRequestEntityTooLarge,
	contrato.ErrFirmaRequerida:           http.StatusUnauthorized,
	contrato.ErrFirmaInvalida:            http.StatusUnauthorized,
	contrato.ErrFirmaExpirada:            http.StatusUnauthorized,
	contrato.ErrRecursoNoEncontrado:      http.StatusNotFound,
	contrato.ErrIDReutilizado:            http.StatusConflict,
	contrato.ErrFilaNoCabe:               http.StatusUnprocessableEntity,
	contrato.ErrOperacionNoSoportada:     http.StatusUnprocessableEntity,
}

// estadoConector es el estado HTTP de los fallos de impresión.
var estadoConector = map[string]int{
	conector.ErrImpresoraNoEncontrada: http.StatusNotFound,
	conector.ErrImpresoraInalcanzable: http.StatusServiceUnavailable,
	conector.ErrPuertoOcupado:         http.StatusServiceUnavailable,
	conector.ErrTiempoAgotado:         http.StatusGatewayTimeout,
	conector.ErrPermisoDenegado:       http.StatusServiceUnavailable,
	conector.ErrColaLlena:             http.StatusServiceUnavailable,
}

// responderError traduce cualquier error a la respuesta JSON. Los errores no previstos se
// registran completos en la bitácora y al cliente solo le llega un mensaje genérico.
func (s *Servidor) responderError(w http.ResponseWriter, r *http.Request, err error) {
	ResponderError(w, r, err, "", s.op.Log)
}

func (s *Servidor) responderErrorTrabajo(w http.ResponseWriter, r *http.Request, err error, trabajoID string) {
	ResponderError(w, r, err, trabajoID, s.op.Log)
}

// ResponderError escribe cualquier error con el formato de la API ({ok:false, codigo, mensaje,
// sugerencia, …}) y el estado HTTP que le corresponde. Lo usa también el panel.
func ResponderError(w http.ResponseWriter, r *http.Request, err error, trabajoID string, log *slog.Logger) {
	resp := respuestaError{TrabajoID: trabajoID}
	estado := http.StatusInternalServerError
	var (
		rz *seguridad.Rechazo
		ce *contrato.Error
		ke *conector.Error
		mb *http.MaxBytesError
	)
	switch {
	case errors.As(err, &rz):
		estado, resp.Codigo, resp.Mensaje, resp.Sugerencia = rz.Estado, rz.Codigo, rz.Mensaje, rz.Sugerencia
	case errors.As(err, &ce):
		estado = http.StatusBadRequest
		if e, ok := estadoContrato[ce.Codigo]; ok {
			estado = e
		}
		resp.Codigo, resp.Mensaje, resp.Sugerencia = ce.Codigo, ce.Mensaje, ce.Sugerencia
		resp.Detalles, resp.VersionesSoportadas = ce.Detalles, ce.VersionesSoportadas
	case errors.As(err, &ke):
		estado = estadoConector[ke.Codigo]
		if estado == 0 {
			estado = http.StatusServiceUnavailable
		}
		if ke.Codigo == conector.ErrColaLlena {
			w.Header().Set("Retry-After", "5")
		}
		resp.Codigo, resp.Mensaje, resp.Sugerencia = ke.Codigo, ke.Mensaje, ke.Sugerencia
	case errors.As(err, &mb):
		estado, resp.Codigo = http.StatusRequestEntityTooLarge, contrato.ErrDocumentoDemasiadoGrande
		resp.Mensaje = fmt.Sprintf("El cuerpo supera los %d bytes.", MaxBytesCuerpo)
		resp.Sugerencia = "Reduce las imágenes en línea o guárdalas en el puente y usa \"recurso\"."
	case errors.Is(err, cola.ErrCerrada):
		estado, resp.Codigo = http.StatusServiceUnavailable, "PUENTE_NO_DISPONIBLE"
		resp.Mensaje, resp.Sugerencia = "El puente se está deteniendo.", "Vuelve a intentarlo en unos segundos."
	case almacen.EsOcupada(err):
		// Otro proceso (un respaldo manual, un visor de SQLite) tiene la base bloqueada.
		estado, resp.Codigo = http.StatusServiceUnavailable, "PUENTE_OCUPADO"
		w.Header().Set("Retry-After", "2")
		resp.Mensaje = "La base de datos del puente está ocupada por otro programa."
		resp.Sugerencia = "Vuelve a intentarlo en unos segundos; si persiste, cierra el programa que tiene abierto prisma.db."
		log.Warn("base de datos ocupada", "metodo", r.Method, "ruta", r.URL.Path)
	default:
		log.Error("error interno", "metodo", r.Method, "ruta", r.URL.Path, "error", err.Error())
		resp.Codigo, resp.Mensaje = "ERROR_INTERNO", "Ocurrió un error interno en el puente."
		resp.Sugerencia = "Vuelve a intentarlo; si persiste, revisa la bitácora del puente."
	}
	responderJSON(w, estado, resp)
}
