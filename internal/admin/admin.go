// Package admin sirve el panel de administración en /admin: los archivos estáticos (embebidos)
// y la API interna /admin/api/*, separada de la API pública /v1 y protegida por sesión y CSRF.
package admin

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bayalad7/prisma/internal/actualizacion"
	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/cola"
	"github.com/bayalad7/prisma/internal/config"
	"github.com/bayalad7/prisma/internal/seguridad"
	"github.com/bayalad7/prisma/internal/servidor"
)

//go:embed web/index.html web/panel.css web/panel.js web/textos.js web/sistema.js web/manual.js web/iconos.svg web/estilos.html web/estilos.js
var archivosWeb embed.FS

// Opciones del panel.
type Opciones struct {
	Almacen  *almacen.Almacen
	Servidor *servidor.Servidor
	Cola     *cola.Cola
	Config   config.Config
	Version  string
	Log      *slog.Logger
	Ahora    func() time.Time     // reloj (pruebas)
	Aviso    *actualizacion.Aviso // versión nueva disponible (nil = sin aviso)
}

// Admin es el panel de administración.
type Admin struct {
	op       Opciones
	sesiones *sesiones
	accesos  *intentos
}

// Nuevo crea el panel.
func Nuevo(op Opciones) *Admin {
	if op.Log == nil {
		op.Log = slog.New(slog.DiscardHandler)
	}
	if op.Ahora == nil {
		op.Ahora = time.Now
	}
	return &Admin{op: op, sesiones: nuevasSesiones(op.Ahora), accesos: nuevosIntentos(op.Ahora)}
}

// cspPanel: solo recursos del propio puente. Los estilos que el panel aplica desde JavaScript
// (CSSOM) no necesitan 'unsafe-inline'.
const cspPanel = "default-src 'self'; img-src 'self' data: blob:; style-src 'self'; script-src 'self'; " +
	"connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'"

// Handler devuelve el manejador de /admin y /admin/*.
func (a *Admin) Handler() http.Handler {
	api := http.NewServeMux()
	publica := func(patron string, h http.HandlerFunc) { api.HandleFunc(patron, h) }
	privada := func(patron string, h http.HandlerFunc) { api.HandleFunc(patron, a.conSesion(h, false)) }

	publica("GET /admin/api/sesion", a.sesion)
	publica("POST /admin/api/entrar", a.entrar)
	publica("POST /admin/api/configuracion-inicial", a.configuracionInicial)
	api.HandleFunc("POST /admin/api/salir", a.conSesion(a.salir, true))
	api.HandleFunc("POST /admin/api/contrasena", a.conSesion(a.cambiarContrasena, true))

	privada("GET /admin/api/resumen", a.resumen)
	privada("GET /admin/api/detectar", a.detectar)
	privada("GET /admin/api/impresoras", a.listarImpresoras)
	privada("POST /admin/api/impresoras", a.crearImpresora)
	privada("POST /admin/api/impresoras/probar-destino", a.probarDestino)
	privada("PUT /admin/api/impresoras/{alias}", a.actualizarImpresora)
	privada("DELETE /admin/api/impresoras/{alias}", a.eliminarImpresora)
	privada("POST /admin/api/impresoras/{alias}/probar", a.probarImpresora)
	privada("POST /admin/api/impresoras/{alias}/diagnostico", a.diagnostico)
	privada("POST /admin/api/impresoras/{alias}/paginas-codigos", a.paginasCodigos)
	privada("POST /admin/api/impresoras/{alias}/acentos-correctos", a.acentosCorrectos)
	privada("GET /admin/api/impresoras/{alias}/rollo", a.listarRollo)
	privada("DELETE /admin/api/impresoras/{alias}/rollo", a.vaciarRollo)
	privada("GET /admin/api/impresoras/{alias}/rollo/{trabajo}/imagen", a.imagenTicket)
	privada("GET /admin/api/impresoras/{alias}/rollo/{trabajo}/bin", a.binTicket)
	privada("GET /admin/api/impresoras/{alias}/rollo/{trabajo}/traza", a.trazaTicket)

	privada("GET /admin/api/clientes", a.listarClientes)
	privada("POST /admin/api/clientes", a.crearCliente)
	privada("PUT /admin/api/clientes/{id}", a.actualizarCliente)
	privada("DELETE /admin/api/clientes/{id}", a.eliminarCliente)
	privada("POST /admin/api/clientes/{id}/llave", a.rotarLlave)
	privada("DELETE /admin/api/clientes/{id}/llave", a.revocarLlave)
	privada("POST /admin/api/validar-llave-publica", a.validarLlavePublica)

	privada("GET /admin/api/solicitudes", a.listarSolicitudes)
	privada("POST /admin/api/solicitudes/aprobar", a.aprobarSolicitud)
	privada("POST /admin/api/solicitudes/rechazar", a.rechazarSolicitud)
	privada("POST /admin/api/solicitudes/olvidar", a.olvidarSolicitud)

	privada("GET /admin/api/trabajos", a.listarTrabajos)

	privada("GET /admin/api/recursos", a.listarRecursos)
	privada("POST /admin/api/recursos", a.guardarRecurso)
	privada("POST /admin/api/recursos/vista-previa", a.vistaPreviaRecurso)
	privada("GET /admin/api/recursos/{nombre}/imagen", a.imagenRecurso)
	privada("DELETE /admin/api/recursos/{nombre}", a.eliminarRecurso)

	privada("GET /admin/api/pruebas/plantillas", a.plantillas)
	privada("POST /admin/api/pruebas/vista-previa", a.vistaPreviaPrueba)
	privada("POST /admin/api/pruebas/imprimir", a.imprimirDocumentoPrueba)
	privada("POST /admin/api/pruebas/permisos", a.comprobarPermisos)

	privada("GET /admin/api/manual", a.listarManual)
	privada("GET /admin/api/manual/buscar", a.buscarManual)

	privada("GET /admin/api/ajustes", a.leerAjustes)
	privada("PUT /admin/api/ajustes", a.guardarAjustes)
	privada("POST /admin/api/tls/regenerar", a.regenerarTLS)
	privada("GET /admin/api/tls/certificado", a.descargarCertificado)
	privada("GET /admin/api/respaldo", a.respaldo)
	privada("POST /admin/api/restaurar", a.restaurar)

	api.HandleFunc("/admin/api/", func(w http.ResponseWriter, r *http.Request) {
		a.error(w, r, rechazo(http.StatusNotFound, "RUTA_NO_ENCONTRADA", "Esa ruta no existe en la API del panel.", ""))
	})

	estaticos, _ := fs.Sub(archivosWeb, "web")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", cspPanel)
		if !a.redPermitida(r) {
			a.error(w, r, rechazo(http.StatusForbidden, "ACCESO_SOLO_RED_LOCAL",
				"El panel solo se puede abrir desde la red local de este negocio.",
				"Ábrelo desde un equipo de la misma red, o desde el propio equipo del puente."))
			return
		}
		switch {
		case r.URL.Path == "/admin":
			http.Redirect(w, r, "/admin/", http.StatusMovedPermanently)
		case strings.HasPrefix(r.URL.Path, "/admin/api/"):
			api.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, "/admin/manual/"):
			a.conSesion(a.servirManual, false)(w, r)
		default:
			servirEstatico(w, r, estaticos, a.op.Config.Dev.ModoDesarrollo)
		}
	})
}

var tiposEstaticos = map[string]string{
	"index.html": "text/html; charset=utf-8",
	"panel.css":  "text/css; charset=utf-8",
	"panel.js":   "text/javascript; charset=utf-8",
	"textos.js":  "text/javascript; charset=utf-8",
	"sistema.js": "text/javascript; charset=utf-8",
	"manual.js":  "text/javascript; charset=utf-8",
	"iconos.svg": "image/svg+xml",
}

// paginasDesarrollo: la galería del sistema de diseño (/admin/estilos) solo existe con
// modo_desarrollo; en producción responde 404.
var paginasDesarrollo = map[string]struct{ archivo, tipo string }{
	"estilos":    {"estilos.html", "text/html; charset=utf-8"},
	"estilos.js": {"estilos.js", "text/javascript; charset=utf-8"},
}

// vistasPanel son las rutas de las pantallas del panel: todas sirven index.html y panel.js dibuja
// la pantalla según la ruta (deben coincidir con SECCIONES de panel.js).
var vistasPanel = []string{"impresoras", "solicitudes", "trabajos", "pruebas", "aplicaciones", "imagenes", "ajustes", "ayuda"}

// subvistas son pantallas dentro de una sección: el comprobador de permisos, el rollo en vivo de
// una impresora virtual (/admin/pruebas/rollo/{alias}) y cada página del manual (/admin/ayuda/{ruta}).
var subvistas = regexp.MustCompile(`^(pruebas/(permisos|rollo/[a-z0-9][a-z0-9_-]{0,31})|ayuda/([a-z]+/[A-Za-z0-9-]+\.md|no-imprime))$`)

func servirEstatico(w http.ResponseWriter, r *http.Request, estaticos fs.FS, desarrollo bool) {
	nombre := strings.TrimPrefix(r.URL.Path, "/admin/")
	if nombre == "" || slices.Contains(vistasPanel, nombre) || subvistas.MatchString(nombre) {
		nombre = "index.html"
	}
	tipo, ok := tiposEstaticos[nombre]
	if p, existe := paginasDesarrollo[nombre]; existe && desarrollo {
		nombre, tipo, ok = p.archivo, p.tipo, true
	}
	if !ok || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
		http.NotFound(w, r)
		return
	}
	datos, err := fs.ReadFile(estaticos, nombre)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", tipo)
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(datos)
}

// redPermitida: por defecto el panel solo acepta IPs privadas; desde el propio equipo siempre.
func (a *Admin) redPermitida(r *http.Request) bool {
	ip := ipDe(r)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	v, ok, _ := a.op.Almacen.Ajustes.Obtener(r.Context(), almacen.AjusteSoloRedPrivada)
	if ok && v == "0" {
		return true
	}
	return ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

func ipDe(r *http.Request) net.IP {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return net.ParseIP(host)
}

func esLocal(r *http.Request) bool {
	ip := ipDe(r)
	return ip != nil && ip.IsLoopback()
}

// --- Respuestas ---------------------------------------------------------------------------------

func rechazo(estado int, codigo, mensaje, sugerencia string) *seguridad.Rechazo {
	return &seguridad.Rechazo{Estado: estado, Codigo: codigo, Mensaje: mensaje, Sugerencia: sugerencia}
}

func invalido(mensaje string) *seguridad.Rechazo {
	return rechazo(http.StatusBadRequest, "DATOS_INVALIDOS", mensaje, "")
}

// error responde con el formato de la API pública; traduce antes los errores de la base, que
// ResponderError trataría como internos.
func (a *Admin) error(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, almacen.ErrNoEncontrado):
		err = rechazo(http.StatusNotFound, "NO_ENCONTRADO", "Lo que buscas ya no existe.", "Recarga la página.")
	case errors.Is(err, almacen.ErrDuplicado):
		err = rechazo(http.StatusConflict, "DUPLICADO", "Ya existe un registro con ese nombre.", "Elige otro.")
	case errors.Is(err, almacen.ErrInvalido):
		err = invalido("Los datos no son válidos.")
	}
	servidor.ResponderError(w, r, err, "", a.op.Log)
}

func responder(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

// leerJSON decodifica el cuerpo (como máximo 256 KB).
func leerJSON(w http.ResponseWriter, r *http.Request, v any) error {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(v); err != nil {
		return invalido("El cuerpo de la petición no es JSON válido.")
	}
	return nil
}

var ok = map[string]bool{"ok": true}

// direccion es la URL con la que los equipos de la red llegan al puente (para mostrarla y en el
// código de integración): la primera IP privada IPv4 de este equipo.
func (a *Admin) direccion() string {
	cfg := a.op.Config.Servidor
	esquema := "https"
	if cfg.ModoTLS == config.TLSSoloLocalhostHTTP {
		esquema = "http"
	}
	host := "localhost"
	if ip := net.ParseIP(cfg.Escuchar); ip != nil && ip.IsLoopback() {
		host = cfg.Escuchar
	} else if cfg.ModoTLS != config.TLSSoloLocalhostHTTP {
		_, ips := servidor.NombresLocales()
		for _, ip := range ips {
			if ip.To4() != nil && ip.IsPrivate() {
				host = ip.String()
				break
			}
		}
	}
	return esquema + "://" + net.JoinHostPort(host, strconv.Itoa(cfg.Puerto))
}

// ctxCorto limita las operaciones con impresoras desde el panel.
func ctxCorto(r *http.Request, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), d)
}
