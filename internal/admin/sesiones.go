package admin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/seguridad"
)

const (
	cookieSesion      = "prisma_sesion"
	inactividadMaxima = 30 * time.Minute
	duracionMaxima    = 12 * time.Hour
	fallosPermitidos  = 5
	bloqueo           = 5 * time.Minute
	longitudMinima    = 10
	vigenciaCodigo    = 30 * time.Minute
	fallosCodigo      = 10 // intentos fallidos del código inicial antes de anularlo
)

// sesion vive solo en memoria: al reiniciar el puente hay que volver a entrar.
type sesion struct {
	csrf   string
	hash   string // hash de la contraseña con la que se entró: si cambia, la sesión termina
	creada time.Time
	ultima time.Time
}

type sesiones struct {
	mu    sync.Mutex
	m     map[[32]byte]*sesion
	ahora func() time.Time
}

func nuevasSesiones(ahora func() time.Time) *sesiones {
	return &sesiones{m: map[[32]byte]*sesion{}, ahora: ahora}
}

func aleatorio(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *sesiones) crear(hash string) (token string, ses *sesion) {
	token = aleatorio(32)
	ahora := s.ahora()
	ses = &sesion{csrf: aleatorio(24), hash: hash, creada: ahora, ultima: ahora}
	s.mu.Lock()
	s.m[sha256.Sum256([]byte(token))] = ses
	s.mu.Unlock()
	return token, ses
}

// obtener devuelve la sesión vigente y renueva su última actividad.
func (s *sesiones) obtener(token string) *sesion {
	if token == "" {
		return nil
	}
	clave := sha256.Sum256([]byte(token))
	s.mu.Lock()
	defer s.mu.Unlock()
	ses := s.m[clave]
	if ses == nil {
		return nil
	}
	ahora := s.ahora()
	if ahora.Sub(ses.ultima) > inactividadMaxima || ahora.Sub(ses.creada) > duracionMaxima {
		delete(s.m, clave)
		return nil
	}
	ses.ultima = ahora
	return ses
}

func (s *sesiones) cerrar(token string) {
	s.mu.Lock()
	delete(s.m, sha256.Sum256([]byte(token)))
	s.mu.Unlock()
}

func (s *sesiones) cerrarTodas() {
	s.mu.Lock()
	clear(s.m)
	s.mu.Unlock()
}

// intentos lleva los inicios de sesión fallidos por IP para bloquear temporalmente.
type intentos struct {
	mu           sync.Mutex
	porIP        map[string]*registroIntentos
	fallosCodigo int
	ahora        func() time.Time
	turno        chan struct{} // argon2id usa 64 MiB: una verificación a la vez
}

type registroIntentos struct {
	fallos int
	hasta  time.Time
}

func nuevosIntentos(ahora func() time.Time) *intentos {
	return &intentos{porIP: map[string]*registroIntentos{}, ahora: ahora, turno: make(chan struct{}, 1)}
}

// bloqueado devuelve cuánto falta si la IP está bloqueada.
func (in *intentos) bloqueado(ip string) time.Duration {
	in.mu.Lock()
	defer in.mu.Unlock()
	if reg := in.porIP[ip]; reg != nil && in.ahora().Before(reg.hasta) {
		return reg.hasta.Sub(in.ahora())
	}
	return 0
}

// fallo registra un intento fallido y devuelve cuántos quedan (0 = recién bloqueada).
func (in *intentos) fallo(ip string) int {
	in.mu.Lock()
	defer in.mu.Unlock()
	reg := in.porIP[ip]
	if reg == nil || (!reg.hasta.IsZero() && in.ahora().After(reg.hasta)) {
		reg = &registroIntentos{}
		in.porIP[ip] = reg
	}
	reg.fallos++
	if reg.fallos >= fallosPermitidos {
		reg.fallos, reg.hasta = 0, in.ahora().Add(bloqueo)
		return 0
	}
	return fallosPermitidos - reg.fallos
}

func (in *intentos) exito(ip string) {
	in.mu.Lock()
	delete(in.porIP, ip)
	in.mu.Unlock()
}

// --- Middleware ---------------------------------------------------------------------------------

type claveSesion struct{}

func sesionDe(r *http.Request) *sesion {
	s, _ := r.Context().Value(claveSesion{}).(*sesion)
	return s
}

func tokenDe(r *http.Request) string {
	c, err := r.Cookie(cookieSesion)
	if err != nil {
		return ""
	}
	return c.Value
}

// sesionValida comprueba la cookie y que la contraseña no haya cambiado desde que se entró
// (por ejemplo, con «prisma admin restablecer-contrasena»).
func (a *Admin) sesionValida(r *http.Request) (*sesion, string) {
	token := tokenDe(r)
	ses := a.sesiones.obtener(token)
	if ses == nil {
		return nil, ""
	}
	hash, _, err := a.op.Almacen.Ajustes.Obtener(r.Context(), almacen.AjusteContrasenaAdmin)
	if err != nil || hash != ses.hash {
		a.sesiones.cerrar(token)
		return nil, ""
	}
	return ses, token
}

func (a *Admin) debeCambiar(ctx context.Context) bool {
	v, _, _ := a.op.Almacen.Ajustes.Obtener(ctx, almacen.AjusteDebeCambiar)
	return v == "1"
}

// conSesion exige una sesión válida y, en las peticiones que modifican, el token CSRF y el mismo
// origen. Con permitirSiDebeCambiar = false, una sesión que debe cambiar la contraseña no avanza.
func (a *Admin) conSesion(h http.HandlerFunc, permitirSiDebeCambiar bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ses, _ := a.sesionValida(r)
		if ses == nil {
			a.error(w, r, rechazo(http.StatusUnauthorized, "SESION_REQUERIDA", "La sesión terminó.", "Vuelve a entrar al panel."))
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(ses.csrf)) != 1 || !mismoOrigen(r) {
				a.error(w, r, rechazo(http.StatusForbidden, "CSRF_INVALIDO", "La petición no viene del panel.", "Recarga la página e inténtalo de nuevo."))
				return
			}
		}
		if !permitirSiDebeCambiar && a.debeCambiar(r.Context()) {
			a.error(w, r, rechazo(http.StatusForbidden, "CAMBIO_DE_CONTRASENA_REQUERIDO",
				"Antes de seguir hay que cambiar la contraseña del panel.", "Recarga la página."))
			return
		}
		h(w, r.WithContext(context.WithValue(r.Context(), claveSesion{}, ses)))
	}
}

// mismoOrigen: si el navegador envía Origin, debe ser el del propio puente.
func mismoOrigen(r *http.Request) bool {
	o := r.Header.Get("Origin")
	if o == "" {
		return true
	}
	return o == "https://"+r.Host || o == "http://"+r.Host
}

func (a *Admin) ponerCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{Name: cookieSesion, Value: token, Path: "/admin", HttpOnly: true,
		Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode, MaxAge: int(duracionMaxima.Seconds())})
}

func borrarCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: cookieSesion, Value: "", Path: "/admin", HttpOnly: true,
		Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode, MaxAge: -1})
}

// --- Acceso -------------------------------------------------------------------------------------

func (a *Admin) sesion(w http.ResponseWriter, r *http.Request) {
	hash, configurado, err := a.op.Almacen.Ajustes.Obtener(r.Context(), almacen.AjusteContrasenaAdmin)
	if err != nil {
		a.error(w, r, err)
		return
	}
	resp := map[string]any{"configurado": configurado && hash != "", "local": esLocal(r), "autenticado": false, "debe_cambiar": false}
	if ses, _ := a.sesionValida(r); ses != nil {
		resp["autenticado"], resp["csrf"], resp["debe_cambiar"] = true, ses.csrf, a.debeCambiar(r.Context())
	}
	responder(w, resp)
}

func (a *Admin) entrar(w http.ResponseWriter, r *http.Request) {
	ip := ipDe(r).String()
	if falta := a.accesos.bloqueado(ip); falta > 0 {
		a.error(w, r, rechazo(http.StatusTooManyRequests, "ACCESO_BLOQUEADO", "Hubo demasiados intentos fallidos desde este equipo.",
			fmt.Sprintf("Espera %d minuto(s) y vuelve a intentarlo.", int(falta.Minutes())+1)))
		return
	}
	var d struct{ Contrasena string }
	if err := leerJSON(w, r, &d); err != nil {
		a.error(w, r, err)
		return
	}
	hash, ok2, err := a.op.Almacen.Ajustes.Obtener(r.Context(), almacen.AjusteContrasenaAdmin)
	if err != nil {
		a.error(w, r, err)
		return
	}
	if !ok2 || hash == "" {
		a.error(w, r, rechazo(http.StatusConflict, "SIN_CONFIGURAR", "El panel aún no tiene contraseña.", "Recarga la página para hacer la configuración inicial."))
		return
	}
	a.accesos.turno <- struct{}{}
	valida, _ := seguridad.VerificarContrasena(d.Contrasena, hash)
	<-a.accesos.turno
	if !valida {
		quedan := a.accesos.fallo(ip)
		sugerencia := fmt.Sprintf("Te quedan %d intento(s) antes de un bloqueo de 5 minutos.", quedan)
		if quedan == 0 {
			sugerencia = "Por seguridad, este equipo queda bloqueado 5 minutos."
		}
		a.op.Log.Warn("intento fallido de entrar al panel", "ip", ip)
		a.error(w, r, rechazo(http.StatusUnauthorized, "CONTRASENA_INCORRECTA", "La contraseña no es correcta.", sugerencia))
		return
	}
	a.accesos.exito(ip)
	token, ses := a.sesiones.crear(hash)
	a.ponerCookie(w, r, token)
	a.op.Log.Info("sesión iniciada en el panel", "ip", ip)
	responder(w, map[string]any{"ok": true, "csrf": ses.csrf, "debe_cambiar": a.debeCambiar(r.Context())})
}

func (a *Admin) salir(w http.ResponseWriter, r *http.Request) {
	a.sesiones.cerrar(tokenDe(r))
	borrarCookie(w, r)
	responder(w, ok)
}

// configuracionInicial crea la primera contraseña. Desde el propio equipo basta con pedirlo;
// desde otro equipo hace falta el código de un solo uso de «prisma admin codigo-inicial».
func (a *Admin) configuracionInicial(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if hash, ok2, _ := a.op.Almacen.Ajustes.Obtener(ctx, almacen.AjusteContrasenaAdmin); ok2 && hash != "" {
		a.error(w, r, rechazo(http.StatusConflict, "YA_CONFIGURADO", "El panel ya tiene contraseña.", "Entra con ella, o restablécela con «prisma admin restablecer-contrasena»."))
		return
	}
	var d struct{ Contrasena, Codigo string }
	if err := leerJSON(w, r, &d); err != nil {
		a.error(w, r, err)
		return
	}
	if !esLocal(r) {
		if err := a.verificarCodigoInicial(ctx, d.Codigo); err != nil {
			a.error(w, r, err)
			return
		}
	}
	if len([]rune(d.Contrasena)) < longitudMinima {
		a.error(w, r, invalido(fmt.Sprintf("La contraseña debe tener al menos %d caracteres.", longitudMinima)))
		return
	}
	hash, err := seguridad.HashContrasena(d.Contrasena)
	if err != nil {
		a.error(w, r, err)
		return
	}
	if err := a.op.Almacen.Ajustes.EstablecerVarios(ctx, map[string]string{almacen.AjusteContrasenaAdmin: hash, almacen.AjusteDebeCambiar: "0"}); err != nil {
		a.error(w, r, err)
		return
	}
	a.op.Almacen.Ajustes.Eliminar(ctx, almacen.AjusteCodigoInicial)
	token, ses := a.sesiones.crear(hash)
	a.ponerCookie(w, r, token)
	a.op.Log.Info("contraseña del panel creada", "ip", ipDe(r).String())
	responder(w, map[string]any{"ok": true, "csrf": ses.csrf})
}

func (a *Admin) verificarCodigoInicial(ctx context.Context, codigo string) error {
	sinCodigo := rechazo(http.StatusForbidden, "CODIGO_REQUERIDO", "Desde otro equipo hace falta el código de configuración.",
		"En el equipo del puente ejecuta «prisma admin codigo-inicial» y escribe aquí el código que muestra.")
	guardado, ok2, err := a.op.Almacen.Ajustes.Obtener(ctx, almacen.AjusteCodigoInicial)
	if err != nil {
		return err
	}
	hash, venceTxt, _ := strings.Cut(guardado, "|")
	vence, _ := strconv.ParseInt(venceTxt, 10, 64)
	if !ok2 || hash == "" || a.op.Ahora().UnixMilli() > vence {
		return sinCodigo
	}
	if codigo == "" || !seguridad.VerificarCodigo(codigo, hash) {
		a.accesos.mu.Lock()
		a.accesos.fallosCodigo++
		anular := a.accesos.fallosCodigo >= fallosCodigo
		if anular {
			a.accesos.fallosCodigo = 0
		}
		a.accesos.mu.Unlock()
		if anular {
			a.op.Almacen.Ajustes.Eliminar(ctx, almacen.AjusteCodigoInicial)
			return rechazo(http.StatusForbidden, "CODIGO_ANULADO", "Hubo demasiados intentos con códigos incorrectos y el código se anuló.",
				"Genera otro con «prisma admin codigo-inicial».")
		}
		return rechazo(http.StatusForbidden, "CODIGO_INVALIDO", "El código de configuración no es correcto.",
			"Revísalo en el equipo del puente; vence a los 30 minutos.")
	}
	return nil
}

// GuardarCodigoInicial genera el código de un solo uso y lo guarda (lo usa la CLI).
func GuardarCodigoInicial(ctx context.Context, alm *almacen.Almacen, ahora time.Time) (codigo string, vence time.Time, err error) {
	codigo, hash, err := seguridad.GenerarCodigoInicial()
	if err != nil {
		return "", time.Time{}, err
	}
	vence = ahora.Add(vigenciaCodigo)
	err = alm.Ajustes.Establecer(ctx, almacen.AjusteCodigoInicial, hash+"|"+strconv.FormatInt(vence.UnixMilli(), 10))
	return codigo, vence, err
}

func (a *Admin) cambiarContrasena(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var d struct{ Actual, Contrasena string }
	if err := leerJSON(w, r, &d); err != nil {
		a.error(w, r, err)
		return
	}
	hashActual, _, err := a.op.Almacen.Ajustes.Obtener(ctx, almacen.AjusteContrasenaAdmin)
	if err != nil {
		a.error(w, r, err)
		return
	}
	// Con el cambio obligatorio (después de restablecerla) ya se demostró la contraseña al entrar.
	if !a.debeCambiar(ctx) {
		a.accesos.turno <- struct{}{}
		valida, _ := seguridad.VerificarContrasena(d.Actual, hashActual)
		<-a.accesos.turno
		if !valida {
			a.error(w, r, rechazo(http.StatusUnauthorized, "CONTRASENA_INCORRECTA", "La contraseña actual no es correcta.", ""))
			return
		}
	}
	if len([]rune(d.Contrasena)) < longitudMinima {
		a.error(w, r, invalido(fmt.Sprintf("La contraseña debe tener al menos %d caracteres.", longitudMinima)))
		return
	}
	hash, err := seguridad.HashContrasena(d.Contrasena)
	if err != nil {
		a.error(w, r, err)
		return
	}
	if err := a.op.Almacen.Ajustes.EstablecerVarios(ctx, map[string]string{almacen.AjusteContrasenaAdmin: hash, almacen.AjusteDebeCambiar: "0"}); err != nil {
		a.error(w, r, err)
		return
	}
	// Las demás sesiones terminan solas (su hash ya no coincide); esta sigue con el nuevo.
	sesionDe(r).hash = hash
	a.op.Log.Info("contraseña del panel cambiada", "ip", ipDe(r).String())
	responder(w, ok)
}
