package admin

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"math/big"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bayalad7/prisma/internal/actualizacion"
	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/cola"
	"github.com/bayalad7/prisma/internal/config"
	"github.com/bayalad7/prisma/internal/seguridad"
	"github.com/bayalad7/prisma/internal/servidor"
)

const contrasenaPrueba = "una contraseña larga"

type banco struct {
	t     *testing.T
	alm   *almacen.Almacen
	srv   *servidor.Servidor
	adm   *Admin
	h     http.Handler
	aviso *actualizacion.Aviso

	mu    sync.Mutex
	ahora time.Time
}

func (b *banco) reloj() time.Time {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.ahora
}

func (b *banco) avanzar(d time.Duration) {
	b.mu.Lock()
	b.ahora = b.ahora.Add(d)
	b.mu.Unlock()
}

func nuevoBanco(t *testing.T) *banco {
	t.Helper()
	ctx := context.Background()
	alm, err := almacen.Abrir(ctx, filepath.Join(t.TempDir(), "prisma.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { alm.Cerrar() })
	q := cola.Nueva(cola.Opciones{IntervaloPrueba: time.Hour})
	t.Cleanup(q.Cerrar)
	cfg := config.PorDefecto()
	cfg.Datos.Ruta = t.TempDir()
	srv := servidor.Nuevo(servidor.Opciones{Almacen: alm, Cola: q, Limites: config.Limites{}, Version: "prueba", TiempoRespuesta: 10 * time.Second})
	b := &banco{t: t, alm: alm, srv: srv, ahora: time.Date(2026, 10, 19, 12, 0, 0, 0, time.Local)}
	b.aviso = &actualizacion.Aviso{}
	b.adm = Nuevo(Opciones{Almacen: alm, Servidor: srv, Cola: q, Config: cfg, Version: "prueba", Ahora: b.reloj, Aviso: b.aviso})
	srv.MontarPanel(b.adm.Handler())
	b.h = srv.Handler()
	return b
}

// navegador guarda la cookie y el token CSRF.
type navegador struct {
	b      *banco
	ip     string
	cookie *http.Cookie
	csrf   string
}

func (b *banco) nuevoNavegador(ip string) *navegador { return &navegador{b: b, ip: ip} }

type respuesta struct {
	*httptest.ResponseRecorder
	datos map[string]any
}

func (r respuesta) codigo() string { s, _ := r.datos["codigo"].(string); return s }

// pedir envía la petición; cuerpo puede ser nil, un valor JSON o *multipart.
func (s *navegador) pedir(metodo, ruta string, cuerpo any, ajustes ...func(*http.Request)) respuesta {
	s.b.t.Helper()
	var lector io.Reader
	tipo := ""
	switch c := cuerpo.(type) {
	case nil:
	case formulario:
		lector, tipo = bytes.NewReader(c.datos), c.tipo
	default:
		j, _ := json.Marshal(c)
		lector, tipo = bytes.NewReader(j), "application/json"
	}
	r := httptest.NewRequest(metodo, ruta, lector)
	r.RemoteAddr = net.JoinHostPort(s.ip, "50000")
	if tipo != "" {
		r.Header.Set("Content-Type", tipo)
	}
	if s.cookie != nil {
		r.AddCookie(s.cookie)
	}
	if metodo != http.MethodGet && s.csrf != "" {
		r.Header.Set("X-CSRF-Token", s.csrf)
	}
	for _, f := range ajustes {
		f(r)
	}
	w := httptest.NewRecorder()
	s.b.h.ServeHTTP(w, r)
	resp := respuesta{ResponseRecorder: w}
	if strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		json.Unmarshal(w.Body.Bytes(), &resp.datos)
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == cookieSesion {
			if c.MaxAge < 0 {
				s.cookie = nil
			} else {
				s.cookie = c
			}
		}
	}
	if csrf, ok := resp.datos["csrf"].(string); ok {
		s.csrf = csrf
	}
	return resp
}

func (s *navegador) exigir(metodo, ruta string, cuerpo any, estado int) respuesta {
	s.b.t.Helper()
	r := s.pedir(metodo, ruta, cuerpo)
	if r.Code != estado {
		s.b.t.Fatalf("%s %s: estado %d, se esperaba %d: %s", metodo, ruta, r.Code, estado, r.Body.String())
	}
	return r
}

func (s *navegador) exigirError(metodo, ruta string, cuerpo any, estado int, codigo string) respuesta {
	s.b.t.Helper()
	r := s.exigir(metodo, ruta, cuerpo, estado)
	if r.codigo() != codigo {
		s.b.t.Fatalf("%s %s: código %q, se esperaba %q: %s", metodo, ruta, r.codigo(), codigo, r.Body.String())
	}
	return r
}

// configurado deja una contraseña y devuelve una sesión iniciada desde la red local.
func (b *banco) configurado() *navegador {
	b.t.Helper()
	hash, err := seguridad.HashContrasena(contrasenaPrueba)
	if err != nil {
		b.t.Fatal(err)
	}
	if err := b.alm.Ajustes.Establecer(context.Background(), almacen.AjusteContrasenaAdmin, hash); err != nil {
		b.t.Fatal(err)
	}
	s := b.nuevoNavegador("192.168.1.20")
	s.exigir("POST", "/admin/api/entrar", map[string]string{"contrasena": contrasenaPrueba}, 200)
	return s
}

type formulario struct {
	datos []byte
	tipo  string
}

func nuevoFormulario(t *testing.T, campos map[string]string, archivo []byte) formulario {
	var buf bytes.Buffer
	m := multipart.NewWriter(&buf)
	for k, v := range campos {
		m.WriteField(k, v)
	}
	if archivo != nil {
		w, _ := m.CreateFormFile("archivo", "archivo.bin")
		w.Write(archivo)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	return formulario{buf.Bytes(), m.FormDataContentType()}
}

func pngPrueba(t *testing.T, w, h int) []byte {
	img := image.NewGray(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.Gray{Y: uint8((x * 255) / w)})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// --- Acceso -------------------------------------------------------------------------------------

func TestConfiguracionInicial(t *testing.T) {
	b := nuevoBanco(t)
	remoto := b.nuevoNavegador("192.168.1.30")

	r := remoto.exigir("GET", "/admin/api/sesion", nil, 200)
	if r.datos["configurado"] != false || r.datos["local"] != false || r.datos["autenticado"] != false {
		t.Fatalf("sesión inicial: %v", r.datos)
	}
	// Desde otro equipo hace falta el código.
	remoto.exigirError("POST", "/admin/api/configuracion-inicial", map[string]string{"contrasena": contrasenaPrueba}, 403, "CODIGO_REQUERIDO")
	codigo, _, err := GuardarCodigoInicial(context.Background(), b.alm, b.reloj())
	if err != nil {
		t.Fatal(err)
	}
	remoto.exigirError("POST", "/admin/api/configuracion-inicial", map[string]string{"contrasena": contrasenaPrueba, "codigo": "AAAA-AAAA"}, 403, "CODIGO_INVALIDO")
	remoto.exigirError("POST", "/admin/api/configuracion-inicial", map[string]string{"contrasena": "corta", "codigo": codigo}, 400, "DATOS_INVALIDOS")

	// El código vence a los 30 minutos.
	b.avanzar(31 * time.Minute)
	remoto.exigirError("POST", "/admin/api/configuracion-inicial", map[string]string{"contrasena": contrasenaPrueba, "codigo": codigo}, 403, "CODIGO_REQUERIDO")
	codigo, _, _ = GuardarCodigoInicial(context.Background(), b.alm, b.reloj())
	// Se acepta en minúsculas y sin guion.
	remoto.exigir("POST", "/admin/api/configuracion-inicial",
		map[string]string{"contrasena": contrasenaPrueba, "codigo": strings.ToLower(strings.ReplaceAll(codigo, "-", ""))}, 200)
	if remoto.cookie == nil || remoto.csrf == "" {
		t.Fatal("la configuración inicial debe iniciar la sesión")
	}
	remoto.exigir("GET", "/admin/api/resumen", nil, 200)
	if _, ok, _ := b.alm.Ajustes.Obtener(context.Background(), almacen.AjusteCodigoInicial); ok {
		t.Fatal("el código debe ser de un solo uso")
	}

	// Ya configurado: ni siquiera desde el propio equipo se puede repetir.
	b.nuevoNavegador("127.0.0.1").exigirError("POST", "/admin/api/configuracion-inicial", map[string]string{"contrasena": contrasenaPrueba}, 409, "YA_CONFIGURADO")
}

func TestConfiguracionInicialLocalSinCodigo(t *testing.T) {
	b := nuevoBanco(t)
	local := b.nuevoNavegador("127.0.0.1")
	if r := local.exigir("GET", "/admin/api/sesion", nil, 200); r.datos["local"] != true {
		t.Fatalf("127.0.0.1 debe ser local: %v", r.datos)
	}
	local.exigir("POST", "/admin/api/configuracion-inicial", map[string]string{"contrasena": contrasenaPrueba}, 200)
	if r := local.exigir("GET", "/admin/api/sesion", nil, 200); r.datos["autenticado"] != true || r.datos["configurado"] != true {
		t.Fatalf("sesión después de configurar: %v", r.datos)
	}
}

func TestCodigoInicialSeAnulaTrasMuchosFallos(t *testing.T) {
	b := nuevoBanco(t)
	GuardarCodigoInicial(context.Background(), b.alm, b.reloj())
	remoto := b.nuevoNavegador("10.0.0.8")
	for range fallosCodigo - 1 {
		remoto.exigirError("POST", "/admin/api/configuracion-inicial", map[string]string{"contrasena": contrasenaPrueba, "codigo": "ZZZZ-ZZZZ"}, 403, "CODIGO_INVALIDO")
	}
	remoto.exigirError("POST", "/admin/api/configuracion-inicial", map[string]string{"contrasena": contrasenaPrueba, "codigo": "ZZZZ-ZZZZ"}, 403, "CODIGO_ANULADO")
	if _, ok, _ := b.alm.Ajustes.Obtener(context.Background(), almacen.AjusteCodigoInicial); ok {
		t.Fatal("el código debe anularse")
	}
}

func TestEntrarBloqueaTrasFallosYCookieSegura(t *testing.T) {
	b := nuevoBanco(t)
	b.configurado()
	s := b.nuevoNavegador("192.168.1.40")
	for i := 1; i < fallosPermitidos; i++ {
		r := s.exigirError("POST", "/admin/api/entrar", map[string]string{"contrasena": "otra"}, 401, "CONTRASENA_INCORRECTA")
		if want := fmt.Sprintf("Te quedan %d", fallosPermitidos-i); !strings.Contains(r.datos["sugerencia"].(string), want) {
			t.Fatalf("sugerencia %q, se esperaba %q", r.datos["sugerencia"], want)
		}
	}
	s.exigirError("POST", "/admin/api/entrar", map[string]string{"contrasena": "otra"}, 401, "CONTRASENA_INCORRECTA")
	// Bloqueada: ni la contraseña correcta entra.
	s.exigirError("POST", "/admin/api/entrar", map[string]string{"contrasena": contrasenaPrueba}, 429, "ACCESO_BLOQUEADO")
	// Otro equipo no está bloqueado.
	b.nuevoNavegador("192.168.1.41").exigir("POST", "/admin/api/entrar", map[string]string{"contrasena": contrasenaPrueba}, 200)

	b.avanzar(bloqueo + time.Second)
	r := s.pedir("POST", "/admin/api/entrar", map[string]string{"contrasena": contrasenaPrueba}, func(r *http.Request) { r.TLS = &tls.ConnectionState{} })
	if r.Code != 200 {
		t.Fatalf("tras el bloqueo: %d %s", r.Code, r.Body.String())
	}
	c := s.cookie
	if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode || c.Path != "/admin" {
		t.Fatalf("cookie sin las protecciones: %+v", c)
	}
}

func TestSesionCSRFYVencimiento(t *testing.T) {
	b := nuevoBanco(t)
	s := b.configurado()

	b.nuevoNavegador("192.168.1.20").exigirError("GET", "/admin/api/resumen", nil, 401, "SESION_REQUERIDA")
	s.exigir("GET", "/admin/api/resumen", nil, 200)

	// Sin token CSRF, con otro token o desde otro origen.
	sinToken := *s
	sinToken.csrf = ""
	sinToken.exigirError("PUT", "/admin/api/ajustes", map[string]any{"retencion_dias": 10}, 403, "CSRF_INVALIDO")
	otro := *s
	otro.csrf = "falso"
	otro.exigirError("PUT", "/admin/api/ajustes", map[string]any{"retencion_dias": 10}, 403, "CSRF_INVALIDO")
	if r := s.pedir("PUT", "/admin/api/ajustes", map[string]any{"retencion_dias": 10}, func(r *http.Request) {
		r.Header.Set("Origin", "https://malicioso.example")
	}); r.codigo() != "CSRF_INVALIDO" {
		t.Fatalf("origen ajeno: %d %s", r.Code, r.Body.String())
	}
	s.exigir("PUT", "/admin/api/ajustes", map[string]any{"retencion_dias": 10}, 200)

	// La actividad renueva la sesión; 30 minutos sin actividad la terminan.
	b.avanzar(20 * time.Minute)
	s.exigir("GET", "/admin/api/resumen", nil, 200)
	b.avanzar(20 * time.Minute)
	s.exigir("GET", "/admin/api/resumen", nil, 200)
	b.avanzar(inactividadMaxima + time.Second)
	s.exigirError("GET", "/admin/api/resumen", nil, 401, "SESION_REQUERIDA")

	// Las peticiones automáticas del panel (el rollo en vivo) comprueban la sesión sin renovarla:
	// una pantalla abierta no impide el cierre por inactividad.
	s = b.configurado()
	sondeo := func(r *http.Request) { r.Header.Set(cabeceraSondeo, "1") }
	b.avanzar(20 * time.Minute)
	if r := s.pedir("GET", "/admin/api/resumen", nil, sondeo); r.Code != 200 {
		t.Fatalf("el sondeo con la sesión vigente debe responder: %d %s", r.Code, r.Body.String())
	}
	b.avanzar(20 * time.Minute)
	if r := s.pedir("GET", "/admin/api/resumen", nil, sondeo); r.codigo() != "SESION_REQUERIDA" {
		t.Fatalf("el sondeo no debe renovar la sesión: %d %s", r.Code, r.Body.String())
	}

	// Restablecer la contraseña desde la CLI cierra las sesiones abiertas.
	s = b.configurado()
	hash, _ := seguridad.HashContrasena("otra contraseña larga")
	b.alm.Ajustes.Establecer(context.Background(), almacen.AjusteContrasenaAdmin, hash)
	s.exigirError("GET", "/admin/api/resumen", nil, 401, "SESION_REQUERIDA")
}

func TestSalir(t *testing.T) {
	b := nuevoBanco(t)
	s := b.configurado()
	galleta := s.cookie
	s.exigir("POST", "/admin/api/salir", nil, 200)
	if s.cookie != nil {
		t.Fatal("salir debe borrar la cookie")
	}
	s.cookie = galleta
	s.exigirError("GET", "/admin/api/resumen", nil, 401, "SESION_REQUERIDA")
}

func TestCambioObligatorio(t *testing.T) {
	b := nuevoBanco(t)
	s := b.configurado()
	b.alm.Ajustes.Establecer(context.Background(), almacen.AjusteDebeCambiar, "1")
	if r := s.exigir("GET", "/admin/api/sesion", nil, 200); r.datos["debe_cambiar"] != true {
		t.Fatalf("debe_cambiar: %v", r.datos)
	}
	s.exigirError("GET", "/admin/api/impresoras", nil, 403, "CAMBIO_DE_CONTRASENA_REQUERIDO")
	s.exigir("POST", "/admin/api/contrasena", map[string]string{"contrasena": "nueva contraseña segura"}, 200)
	s.exigir("GET", "/admin/api/impresoras", nil, 200)

	// Fuera del cambio obligatorio hay que dar la actual.
	s.exigirError("POST", "/admin/api/contrasena", map[string]string{"actual": "mal", "contrasena": "otra más larga"}, 401, "CONTRASENA_INCORRECTA")
	s.exigir("POST", "/admin/api/contrasena", map[string]string{"actual": "nueva contraseña segura", "contrasena": "otra más larga"}, 200)
	s.exigir("GET", "/admin/api/impresoras", nil, 200)
}

func TestSoloRedPrivada(t *testing.T) {
	b := nuevoBanco(t)
	s := b.configurado()
	publica := b.nuevoNavegador("8.8.8.8")
	publica.exigirError("GET", "/admin/", nil, 403, "ACCESO_SOLO_RED_LOCAL")
	publica.exigirError("GET", "/admin/api/sesion", nil, 403, "ACCESO_SOLO_RED_LOCAL")
	b.nuevoNavegador("::1").exigir("GET", "/admin/", nil, 200)
	b.nuevoNavegador("fd00::5").exigir("GET", "/admin/", nil, 200)

	s.exigir("PUT", "/admin/api/ajustes", map[string]any{"solo_red_privada": false}, 200)
	publica.exigir("GET", "/admin/api/sesion", nil, 200)
	if r := s.exigir("GET", "/admin/api/ajustes", nil, 200); r.datos["solo_red_privada"] != false {
		t.Fatalf("ajustes: %v", r.datos)
	}
}

func TestArchivosEstaticosYCSP(t *testing.T) {
	b := nuevoBanco(t)
	n := b.nuevoNavegador("127.0.0.1")
	if r := n.pedir("GET", "/admin", nil); r.Code != http.StatusMovedPermanently || r.Header().Get("Location") != "/admin/" {
		t.Fatalf("/admin: %d %v", r.Code, r.Header())
	}
	for ruta, tipo := range map[string]string{"/admin/": "text/html", "/admin/panel.js": "text/javascript", "/admin/panel.css": "text/css"} {
		r := n.exigir("GET", ruta, nil, 200)
		if !strings.HasPrefix(r.Header().Get("Content-Type"), tipo) {
			t.Errorf("%s: Content-Type %q", ruta, r.Header().Get("Content-Type"))
		}
		if r.Header().Get("Content-Security-Policy") != cspPanel || r.Header().Get("X-Frame-Options") != "DENY" {
			t.Errorf("%s: cabeceras %v", ruta, r.Header())
		}
	}
	n.exigir("GET", "/admin/demo.js", nil, 404)
	n.exigir("GET", "/admin/../go.mod", nil, 404)
	n.exigirError("GET", "/admin/api/no-existe", nil, 404, "RUTA_NO_ENCONTRADA")
}

// --- Terminado cuando: impresora, cliente nivel 2, aprobar origen y ticket de diagnóstico ---------

func TestFlujoCompletoDesdeElPanel(t *testing.T) {
	b := nuevoBanco(t)
	s := b.configurado()
	dir := t.TempDir()

	imp := map[string]any{"alias": "caja", "nombre": "Caja", "tipo": "archivo", "destino": dir, "ancho_mm": 80, "columnas": 48,
		"columnas_fuente_b": 64, "ancho_px": 576, "pagina_codigos": "CP858", "comando_pagina_codigos": 19, "soporta_corte": true,
		"soporta_qr_nativo": true, "soporta_codigo_barras_nativo": true, "soporta_imagen": true, "activa": true}
	s.exigir("POST", "/admin/api/impresoras/probar-destino", map[string]any{"tipo": "archivo", "destino": dir}, 200)
	s.exigir("POST", "/admin/api/impresoras", imp, 200)
	s.exigirError("POST", "/admin/api/impresoras", imp, 409, "ALIAS_DUPLICADO")
	malo := map[string]any{"alias": "Caja Principal", "tipo": "archivo", "destino": dir, "ancho_mm": 80}
	s.exigirError("POST", "/admin/api/impresoras", malo, 400, "DATOS_INVALIDOS")
	s.exigir("POST", "/admin/api/impresoras/caja/probar", nil, 200)

	// Aplicación de nivel 2: la llave se muestra una sola vez.
	r := s.exigir("POST", "/admin/api/clientes", map[string]any{"nombre": "Punto de venta", "nivel": 2, "todas_las_impresoras": false,
		"impresoras": []string{"caja"}, "origenes": []string{}}, 200)
	llave, _ := r.datos["llave"].(string)
	cli := r.datos["cliente"].(map[string]any)
	if !strings.HasPrefix(llave, "pk_") || !strings.HasPrefix(llave, cli["llave_prefijo"].(string)) {
		t.Fatalf("llave %q, cliente %v", llave, cli)
	}
	if lista := s.exigir("GET", "/admin/api/clientes", nil, 200); strings.Contains(lista.Body.String(), llave) {
		t.Fatal("la llave completa no debe volver a mostrarse")
	}

	// Un origen desconocido intenta imprimir y queda pendiente.
	origen := "http://192.168.1.77:5173"
	imprimir := func() *httptest.ResponseRecorder {
		doc := fmt.Sprintf(`{"version":"1","id":"%s","impresora":"caja","operaciones":[{"tipo":"texto","valor":"Hola"},{"tipo":"cortar"}]}`, uuid4())
		req := httptest.NewRequest("POST", "/v1/imprimir", strings.NewReader(doc))
		req.RemoteAddr = "192.168.1.77:40000"
		req.Header.Set("Origin", origen)
		req.Header.Set("Authorization", "Bearer "+llave)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		b.h.ServeHTTP(w, req)
		return w
	}
	if w := imprimir(); w.Code != http.StatusForbidden {
		t.Fatalf("origen sin aprobar: %d %s", w.Code, w.Body.String())
	}
	sol := s.exigir("GET", "/admin/api/solicitudes", nil, 200)
	if !strings.Contains(sol.Body.String(), origen) {
		t.Fatalf("la solicitud no aparece: %s", sol.Body.String())
	}
	if r := s.exigir("GET", "/admin/api/resumen", nil, 200); r.datos["solicitudes_pendientes"] != float64(1) {
		t.Fatalf("resumen: %v", r.datos)
	}
	r = s.exigir("POST", "/admin/api/solicitudes/aprobar", map[string]any{"origen": origen, "cliente_id": cli["id"]}, 200)
	if fmt.Sprint(r.datos["cliente"].(map[string]any)["origenes"]) != "["+origen+"]" {
		t.Fatalf("aprobar: %v", r.datos)
	}
	if w := imprimir(); w.Code != http.StatusOK {
		t.Fatalf("imprimir tras aprobar: %d %s", w.Code, w.Body.String())
	}

	// Ticket de diagnóstico.
	r = s.exigir("POST", "/admin/api/impresoras/caja/diagnostico", nil, 200)
	vista, _ := r.datos["vista"].(string)
	for _, want := range []string{"PRUEBA DE IMPRESORA", "Alias", "caja", "CP858", "áéíóú", "Derecha"} {
		if !strings.Contains(vista, want) {
			t.Errorf("la vista del diagnóstico no contiene %q:\n%s", want, vista)
		}
	}
	regla := ".........1.........2.........3.........4........"
	if !strings.Contains(vista, regla) {
		t.Errorf("falta la regla de columnas:\n%s", vista)
	}
	archivos, _ := filepath.Glob(filepath.Join(dir, "*.bin"))
	if len(archivos) != 2 {
		t.Fatalf("se esperaban 2 trabajos impresos en %s, hay %d", dir, len(archivos))
	}
	s.exigir("POST", "/admin/api/impresoras/caja/paginas-codigos", nil, 200)

	trabajos := s.exigir("GET", "/admin/api/trabajos", nil, 200)
	var lista []trabajoJSON
	json.Unmarshal(trabajos.Body.Bytes(), &lista)
	porReferencia := map[string]string{}
	for _, t := range lista {
		porReferencia[t.Referencia+"/"+t.Estado] = t.ClienteNombre
	}
	if len(lista) != 3 || porReferencia["PAGINAS/completado"] != "Panel" || porReferencia["DIAGNOSTICO/completado"] != "Panel" ||
		porReferencia["/completado"] != "Punto de venta" {
		t.Fatalf("trabajos: %+v", lista)
	}
	hoy := time.Now().Format(time.DateOnly)
	filtrados := s.exigir("GET", "/admin/api/trabajos?cliente="+cli["id"].(string)+"&desde="+hoy+"&hasta="+hoy, nil, 200)
	json.Unmarshal(filtrados.Body.Bytes(), &lista)
	if len(lista) != 1 {
		t.Fatalf("filtro por cliente: %+v", lista)
	}
	s.exigirError("GET", "/admin/api/trabajos?desde=ayer", nil, 400, "DATOS_INVALIDOS")
}

func uuid4() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func TestImpresoraEditarYEliminar(t *testing.T) {
	b := nuevoBanco(t)
	s := b.configurado()
	imp := map[string]any{"alias": "barra", "tipo": "bluetooth", "destino": "COM9", "ancho_mm": 58, "columnas": 32,
		"columnas_fuente_b": 42, "ancho_px": 384, "pagina_codigos": "cp850", "comando_pagina_codigos": 2, "activa": true}
	r := s.exigir("POST", "/admin/api/impresoras", imp, 200)
	if r.datos["baudios"] != float64(9600) || r.datos["pagina_codigos"] != "CP850" || r.datos["nombre"] != "barra" {
		t.Fatalf("valores por defecto: %v", r.datos)
	}
	imp["activa"], imp["nombre"] = false, "Barra"
	r = s.exigir("PUT", "/admin/api/impresoras/barra", imp, 200)
	if r.datos["estado"] != "inactiva" || r.datos["nombre"] != "Barra" {
		t.Fatalf("editar: %v", r.datos)
	}
	s.exigirError("POST", "/admin/api/impresoras/barra/diagnostico", nil, 409, "IMPRESORA_INACTIVA")
	imp["alias"] = "otra"
	s.exigirError("PUT", "/admin/api/impresoras/barra", imp, 400, "DATOS_INVALIDOS")
	s.exigir("DELETE", "/admin/api/impresoras/barra", nil, 200)
	s.exigirError("DELETE", "/admin/api/impresoras/barra", nil, 404, "IMPRESORA_NO_ENCONTRADA")
}

func TestClientesNivelesYLlaves(t *testing.T) {
	b := nuevoBanco(t)
	s := b.configurado()

	s.exigirError("POST", "/admin/api/clientes", map[string]any{"nombre": "Menú", "nivel": 1, "todas_las_impresoras": true}, 400, "DATOS_INVALIDOS")
	r := s.exigir("POST", "/admin/api/clientes", map[string]any{"nombre": "Menú", "nivel": 1, "todas_las_impresoras": true,
		"origenes": []string{"HTTPS://Menu.Example:443"}}, 200)
	if _, hay := r.datos["llave"]; hay {
		t.Fatal("el nivel 1 no lleva llave")
	}
	menu := r.datos["cliente"].(map[string]any)
	if fmt.Sprint(menu["origenes"]) != "[https://menu.example]" || menu["llave_prefijo"] != nil {
		t.Fatalf("nivel 1: %v", menu)
	}
	// Un origen solo pertenece a una aplicación.
	s.exigirError("POST", "/admin/api/clientes", map[string]any{"nombre": "Otra", "nivel": 2, "origenes": []string{"https://menu.example"}}, 409, "ORIGEN_DUPLICADO")

	// Nivel 3: la llave pública se valida.
	publica, _, _ := ed25519.GenerateKey(rand.Reader)
	pem, _ := seguridad.LlavePublicaPEM(publica)
	s.exigirError("POST", "/admin/api/validar-llave-publica", map[string]any{"llave": "no es una llave"}, 400, "LLAVE_PUBLICA_INVALIDA")
	r = s.exigir("POST", "/admin/api/validar-llave-publica", map[string]any{"llave": string(pem)}, 200)
	if r.datos["formato"] != "PEM" || r.datos["huella"] != seguridad.HuellaLlavePublica(publica) {
		t.Fatalf("validar PEM: %v", r.datos)
	}
	r = s.exigir("POST", "/admin/api/validar-llave-publica", map[string]any{"llave": base64.StdEncoding.EncodeToString(publica)}, 200)
	if r.datos["formato"] != "base64" {
		t.Fatalf("validar base64: %v", r.datos)
	}
	s.exigirError("POST", "/admin/api/clientes", map[string]any{"nombre": "Pedidos", "nivel": 3, "todas_las_impresoras": true}, 400, "LLAVE_PUBLICA_INVALIDA")
	r = s.exigir("POST", "/admin/api/clientes", map[string]any{"nombre": "Pedidos", "nivel": 3, "todas_las_impresoras": true,
		"llave_publica": string(pem), "operaciones_sensibles": []string{"abrir_cajon"}}, 200)
	pedidos := r.datos["cliente"].(map[string]any)
	if pedidos["huella_publica"] != seguridad.HuellaLlavePublica(publica) || r.datos["llave"] != nil {
		t.Fatalf("nivel 3: %v", r.datos)
	}
	s.exigirError("POST", "/admin/api/clientes", map[string]any{"nombre": "X", "nivel": 2, "operaciones_sensibles": []string{"formatear"}}, 400, "DATOS_INVALIDOS")

	// Pasar a nivel 2 genera la llave; regenerarla invalida la anterior; revocarla la quita.
	id := menu["id"].(string)
	menu["nivel"] = 2
	r = s.exigir("PUT", "/admin/api/clientes/"+id, menu, 200)
	llave1, _ := r.datos["llave"].(string)
	r = s.exigir("POST", "/admin/api/clientes/"+id+"/llave", nil, 200)
	llave2, _ := r.datos["llave"].(string)
	if llave1 == "" || llave2 == "" || llave1 == llave2 {
		t.Fatalf("llaves %q %q", llave1, llave2)
	}
	c, _ := b.alm.Clientes.Obtener(context.Background(), id)
	if !strings.HasPrefix(llave2, c.LlavePrefijo) {
		t.Fatal("la llave guardada debe ser la última")
	}
	s.exigir("DELETE", "/admin/api/clientes/"+id+"/llave", nil, 200)
	c, _ = b.alm.Clientes.Obtener(context.Background(), id)
	if c.LlaveHash != "" || c.Nivel != 2 {
		t.Fatalf("revocar: %+v", c)
	}
	// Desactivar y eliminar.
	menu["activo"] = false
	if r := s.exigir("PUT", "/admin/api/clientes/"+id, menu, 200); r.datos["cliente"].(map[string]any)["activo"] != false {
		t.Fatalf("desactivar: %v", r.datos)
	}
	s.exigir("DELETE", "/admin/api/clientes/"+id, nil, 200)
	s.exigirError("DELETE", "/admin/api/clientes/"+id, nil, 404, "CLIENTE_NO_ENCONTRADO")
}

func TestSolicitudesAprobarConAplicacionNuevaRechazarYOlvidar(t *testing.T) {
	b := nuevoBanco(t)
	s := b.configurado()
	ctx := context.Background()
	for _, o := range []string{"https://a.example", "https://b.example", "https://c.example"} {
		if _, err := b.alm.Solicitudes.Registrar(ctx, o); err != nil {
			t.Fatal(err)
		}
	}
	r := s.exigir("POST", "/admin/api/solicitudes/aprobar", map[string]any{"origen": "https://a.example", "nuevo": map[string]any{"nombre": "App A", "nivel": 2}}, 200)
	if !strings.HasPrefix(r.datos["llave"].(string), "pk_") || r.datos["cliente"].(map[string]any)["todas_las_impresoras"] != true {
		t.Fatalf("aprobar nueva: %v", r.datos)
	}
	s.exigirError("POST", "/admin/api/solicitudes/aprobar", map[string]any{"origen": "https://a.example", "nuevo": map[string]any{"nombre": "Otra", "nivel": 1}}, 404, "SOLICITUD_NO_ENCONTRADA")
	s.exigirError("POST", "/admin/api/solicitudes/aprobar", map[string]any{"origen": "https://b.example", "nuevo": map[string]any{"nivel": 3}}, 400, "DATOS_INVALIDOS")
	s.exigir("POST", "/admin/api/solicitudes/rechazar", map[string]any{"origen": "https://b.example"}, 200)
	s.exigir("POST", "/admin/api/solicitudes/olvidar", map[string]any{"origen": "https://c.example"}, 200)
	var lista []solicitudJSON
	json.Unmarshal(s.exigir("GET", "/admin/api/solicitudes", nil, 200).Body.Bytes(), &lista)
	if len(lista) != 1 || lista[0].Origen != "https://b.example" || lista[0].Estado != "rechazada" {
		t.Fatalf("solicitudes: %+v", lista)
	}
}

func TestRecursos(t *testing.T) {
	b := nuevoBanco(t)
	s := b.configurado()
	imagen := pngPrueba(t, 800, 200)

	r := s.exigir("POST", "/admin/api/recursos/vista-previa", nuevoFormulario(t, map[string]string{"ancho": "384"}, imagen), 200)
	if cfg, err := png.DecodeConfig(bytes.NewReader(r.Body.Bytes())); err != nil || cfg.Width != 384 {
		t.Fatalf("vista previa: %v %+v", err, cfg)
	}
	s.exigirError("POST", "/admin/api/recursos", nuevoFormulario(t, map[string]string{"nombre": "logo"}, []byte("no es png")), 400, "IMAGEN_INVALIDA")
	s.exigirError("POST", "/admin/api/recursos", nuevoFormulario(t, map[string]string{"nombre": "Logo Tienda"}, imagen), 400, "DATOS_INVALIDOS")
	s.exigir("POST", "/admin/api/recursos", nuevoFormulario(t, map[string]string{"nombre": "logo"}, imagen), 200)

	var lista []map[string]any
	json.Unmarshal(s.exigir("GET", "/admin/api/recursos", nil, 200).Body.Bytes(), &lista)
	if len(lista) != 1 || lista[0]["ancho_px"] != float64(800) || lista[0]["bytes"] != float64(len(imagen)) {
		t.Fatalf("recursos: %v", lista)
	}
	r = s.exigir("GET", "/admin/api/recursos/logo/imagen?ancho=576", nil, 200)
	if cfg, err := png.DecodeConfig(bytes.NewReader(r.Body.Bytes())); err != nil || cfg.Width != 576 {
		t.Fatalf("imagen: %v %+v", err, cfg)
	}
	s.exigir("DELETE", "/admin/api/recursos/logo", nil, 200)
	s.exigirError("GET", "/admin/api/recursos/logo/imagen", nil, 404, "NO_ENCONTRADO")
}

func TestAjustesYLimites(t *testing.T) {
	b := nuevoBanco(t)
	s := b.configurado()
	s.exigir("PUT", "/admin/api/ajustes", map[string]any{"nombre_puente": "  Abarrotes  ", "limite_ip": 300, "limite_cliente": 90, "retencion_dias": 7}, 200)
	r := s.exigir("GET", "/admin/api/ajustes", nil, 200)
	if r.datos["nombre_puente"] != "Abarrotes" || r.datos["limite_ip"] != float64(300) || r.datos["limite_cliente"] != float64(90) ||
		r.datos["retencion_dias"] != float64(7) || r.datos["solo_red_privada"] != true {
		t.Fatalf("ajustes: %v", r.datos)
	}
	porIP, porCliente := b.srv.Limitadores()
	if porIP.PorMinuto() != 300 || porCliente.PorMinuto() != 90 {
		t.Fatalf("límites en caliente: %d %d", porIP.PorMinuto(), porCliente.PorMinuto())
	}
	s.exigirError("PUT", "/admin/api/ajustes", map[string]any{"limite_ip": 10}, 400, "DATOS_INVALIDOS")
	s.exigirError("PUT", "/admin/api/ajustes", map[string]any{"retencion_dias": 0}, 400, "DATOS_INVALIDOS")
	s.exigirError("PUT", "/admin/api/ajustes", map[string]any{"nombre_puente": " "}, 400, "DATOS_INVALIDOS")

	// Certificado autofirmado: se regenera y se descarga.
	s.exigir("POST", "/admin/api/tls/regenerar", nil, 200)
	r = s.exigir("GET", "/admin/api/ajustes", nil, 200)
	cert, _ := r.datos["tls"].(map[string]any)["certificado"].(map[string]any)
	if cert == nil || cert["huella"] == "" {
		t.Fatalf("tls: %v", r.datos["tls"])
	}
	r = s.exigir("GET", "/admin/api/tls/certificado", nil, 200)
	if !strings.Contains(r.Body.String(), "BEGIN CERTIFICATE") || !strings.Contains(r.Header().Get("Content-Disposition"), "prisma.crt") {
		t.Fatalf("descarga: %v", r.Header())
	}
}

func TestRespaldoYRestaurar(t *testing.T) {
	b := nuevoBanco(t)
	s := b.configurado()
	imp := map[string]any{"alias": "caja", "tipo": "archivo", "destino": t.TempDir(), "ancho_mm": 80, "columnas": 48,
		"columnas_fuente_b": 64, "ancho_px": 576, "pagina_codigos": "CP858", "comando_pagina_codigos": 19, "activa": true}
	s.exigir("POST", "/admin/api/impresoras", imp, 200)
	r := s.exigir("GET", "/admin/api/respaldo", nil, 200)
	respaldo := r.Body.Bytes()
	if !bytes.HasPrefix(respaldo, []byte("SQLite format 3\x00")) || !strings.Contains(r.Header().Get("Content-Disposition"), ".db") {
		t.Fatalf("respaldo: %d bytes, %v", len(respaldo), r.Header())
	}

	imp["alias"] = "cocina"
	s.exigir("POST", "/admin/api/impresoras", imp, 200)
	s.exigirError("POST", "/admin/api/restaurar", nuevoFormulario(t, nil, []byte("basura")), 400, "RESPALDO_INVALIDO")
	s.exigir("POST", "/admin/api/restaurar", nuevoFormulario(t, nil, respaldo), 200)

	// Las sesiones terminan; al volver a entrar está la base del respaldo.
	if s.cookie != nil {
		t.Fatal("restaurar debe cerrar la sesión")
	}
	s.exigir("POST", "/admin/api/entrar", map[string]string{"contrasena": contrasenaPrueba}, 200)
	var lista []impresoraJSON
	json.Unmarshal(s.exigir("GET", "/admin/api/impresoras", nil, 200).Body.Bytes(), &lista)
	if len(lista) != 1 || lista[0].Alias != "caja" {
		t.Fatalf("impresoras restauradas: %+v", lista)
	}
	if _, ok := b.srvCola().EstadoDe("cocina"); ok {
		t.Fatal("la impresora que no está en el respaldo debe salir de la cola")
	}
}

func (b *banco) srvCola() *cola.Cola { return b.adm.op.Cola }

func TestNoRegistraContrasenasNiLlaves(t *testing.T) {
	var bitacora bytes.Buffer
	b := nuevoBanco(t)
	b.adm.op.Log = slogJSON(&bitacora)
	s := b.configurado()
	r := s.exigir("POST", "/admin/api/clientes", map[string]any{"nombre": "App", "nivel": 2, "todas_las_impresoras": true}, 200)
	llave := r.datos["llave"].(string)
	s.pedir("POST", "/admin/api/entrar", map[string]string{"contrasena": "secreto-incorrecto"})
	for _, secreto := range []string{llave, contrasenaPrueba, "secreto-incorrecto"} {
		if strings.Contains(bitacora.String(), secreto) {
			t.Fatalf("la bitácora contiene un secreto: %s", bitacora.String())
		}
	}
	if !strings.Contains(bitacora.String(), "aplicación creada") {
		t.Fatalf("la bitácora debería registrar la acción: %s", bitacora.String())
	}
}

func slogJSON(w io.Writer) *slog.Logger { return slog.New(slog.NewJSONHandler(w, nil)) }

func TestDiagnosticoSinQRNiImagenes(t *testing.T) {
	b := nuevoBanco(t)
	s := b.configurado()
	imp := map[string]any{"alias": "basica", "tipo": "archivo", "destino": t.TempDir(), "ancho_mm": 58, "columnas": 32,
		"columnas_fuente_b": 42, "ancho_px": 384, "pagina_codigos": "CP437", "comando_pagina_codigos": 0, "activa": true}
	s.exigir("POST", "/admin/api/impresoras", imp, 200)
	r := s.exigir("POST", "/admin/api/impresoras/basica/diagnostico", nil, 200)
	if vista := r.datos["vista"].(string); !strings.Contains(vista, ".........1.........2.........3..") {
		t.Fatalf("regla de 32 columnas:\n%s", vista)
	}
}

func TestAvisoDeVersionNueva(t *testing.T) {
	b := nuevoBanco(t)
	s := b.configurado()
	if r := s.exigir("GET", "/admin/api/resumen", nil, 200); r.datos["actualizacion"] != nil {
		t.Fatalf("sin versión nueva: %v", r.datos["actualizacion"])
	}
	b.aviso.Poner(&actualizacion.Version{Numero: "9.9.9", URL: "https://github.com/bayalad7/prisma/releases/tag/v9.9.9"})
	r := s.exigir("GET", "/admin/api/resumen", nil, 200)
	if v, _ := r.datos["actualizacion"].(map[string]any); v["version"] != "9.9.9" || v["url"] == "" {
		t.Fatalf("aviso: %v", r.datos["actualizacion"])
	}
	if r := s.exigir("GET", "/admin/api/ajustes", nil, 200); r.datos["buscar_actualizaciones"] != true {
		t.Fatalf("por defecto se buscan actualizaciones: %v", r.datos)
	}
	// Desactivado en Ajustes: el aviso desaparece.
	s.exigir("PUT", "/admin/api/ajustes", map[string]any{"buscar_actualizaciones": false}, 200)
	if r := s.exigir("GET", "/admin/api/resumen", nil, 200); r.datos["actualizacion"] != nil {
		t.Fatalf("aviso desactivado: %v", r.datos["actualizacion"])
	}
	if v, _, _ := b.alm.Ajustes.Obtener(context.Background(), almacen.AjusteBuscarActualizaciones); v != "0" {
		t.Fatalf("ajuste guardado: %q", v)
	}
}

func TestAvisoDeCertificadoVencido(t *testing.T) {
	b := nuevoBanco(t)
	s := b.configurado()
	if r := s.exigir("GET", "/admin/api/resumen", nil, 200); r.datos["certificado"] != nil {
		t.Fatalf("sin certificado no hay aviso: %v", r.datos["certificado"])
	}
	// Certificado autofirmado vencido en la carpeta de datos.
	llave, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	plantilla := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: b.reloj().Add(-48 * time.Hour), NotAfter: b.reloj().Add(-24 * time.Hour)}
	der, _ := x509.CreateCertificate(rand.Reader, plantilla, plantilla, &llave.PublicKey, llave)
	ruta, _ := servidor.RutasAutofirmado(b.adm.op.Config.DirDatos())
	os.MkdirAll(filepath.Dir(ruta), 0o700)
	os.WriteFile(ruta, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644)

	r := s.exigir("GET", "/admin/api/resumen", nil, 200)
	aviso, _ := r.datos["certificado"].(map[string]any)
	if aviso["estado"] != servidor.CertificadoVencido || aviso["autofirmado"] != true {
		t.Fatalf("aviso: %v", r.datos["certificado"])
	}
	tls := s.exigir("GET", "/admin/api/ajustes", nil, 200).datos["tls"].(map[string]any)
	if tls["certificado"].(map[string]any)["estado"] != servidor.CertificadoVencido {
		t.Fatalf("ajustes: %v", tls)
	}
}
