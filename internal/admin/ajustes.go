package admin

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/config"
	"github.com/bayalad7/prisma/internal/escpos"
	"github.com/bayalad7/prisma/internal/servidor"
)

func fecha(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// --- Inicio -------------------------------------------------------------------------------------

func (a *Admin) nombrePuente(r *http.Request) string {
	v, ok2, _ := a.op.Almacen.Ajustes.Obtener(r.Context(), servidor.AjusteNombrePuente)
	if !ok2 || v == "" {
		return "PRISMA"
	}
	return v
}

func (a *Admin) resumen(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	lista, err := a.op.Almacen.Impresoras.Listar(ctx)
	if err != nil {
		a.error(w, r, err)
		return
	}
	type impresoraResumen struct {
		Alias      string `json:"alias"`
		Nombre     string `json:"nombre"`
		Estado     any    `json:"estado"`
		Pendientes int    `json:"pendientes"`
		Virtual    bool   `json:"virtual"`
		AnchoMM    int    `json:"ancho_mm"`
	}
	imps := []impresoraResumen{}
	for _, imp := range lista {
		j := a.aJSON(imp)
		imps = append(imps, impresoraResumen{j.Alias, j.Nombre, j.Estado, j.Pendientes, j.Virtual, j.AnchoMM})
	}
	pendientes, err := a.op.Almacen.Solicitudes.Listar(ctx, "pendiente")
	if err != nil {
		a.error(w, r, err)
		return
	}
	ahora := a.op.Ahora()
	hoy := time.Date(ahora.Year(), ahora.Month(), ahora.Day(), 0, 0, 0, 0, ahora.Location())
	conteo, err := a.op.Almacen.Trabajos.Contar(ctx, hoy)
	if err != nil {
		a.error(w, r, err)
		return
	}
	// Las pruebas del panel (diagnóstico, páginas de códigos) se cuentan aparte: no son tickets del negocio.
	pruebas, err := a.op.Almacen.Trabajos.ContarDe(ctx, hoy, ClientePanel)
	if err != nil {
		a.error(w, r, err)
		return
	}
	clientes, err := a.op.Almacen.Clientes.Listar(ctx)
	if err != nil {
		a.error(w, r, err)
		return
	}
	fisicas, virtuales := 0, 0
	for _, imp := range lista {
		if imp.Tipo == almacen.ConexionVirtual {
			virtuales++
		} else {
			fisicas++
		}
	}
	pruebaImpresa, acentosProbados := a.pasosDePrueba(r)
	ultimos, err := a.trabajosJSON(r, almacen.FiltroTrabajos{Limite: 8})
	if err != nil {
		a.error(w, r, err)
		return
	}
	var nueva any // null si no hay versión nueva o el aviso está desactivado
	if v := a.op.Aviso.Disponible(); v != nil && a.buscarActualizaciones(r) {
		nueva = map[string]string{"version": v.Numero, "url": v.URL, "fecha": fecha(v.Fecha)}
	}
	responder(w, map[string]any{
		"puente": map[string]any{"nombre": a.nombrePuente(r), "version": a.op.Version, "direccion": a.direccion(),
			"modo_tls": a.op.Config.Servidor.ModoTLS, "iniciado": fecha(a.op.Servidor.Iniciado())},
		"actualizacion":          nueva,
		"certificado":            a.avisoCertificado(),
		"impresoras":             imps,
		"solicitudes_pendientes": len(pendientes),
		"trabajos_hoy": map[string]int{
			"completados": conteo[almacen.TrabajoCompletado] - pruebas[almacen.TrabajoCompletado],
			"fallidos":    conteo[almacen.TrabajoFallido] - pruebas[almacen.TrabajoFallido],
			"pruebas":     pruebas[almacen.TrabajoCompletado] + pruebas[almacen.TrabajoFallido],
		},
		"certificado_actual": a.estadoTLS()["certificado"], // nil en el modo solo_localhost_http
		"primeros_pasos": map[string]any{
			"impresoras_fisicas": fisicas, "impresoras_virtuales": virtuales, "aplicaciones": len(clientes),
			"prueba_impresa": pruebaImpresa, "acentos_probados": acentosProbados,
		},
		"ultimos_trabajos": ultimos,
	})
}

// --- Trabajos (sin contenido: el puente nunca guarda los tickets) ---------------------------------

type trabajoJSON struct {
	ID             string   `json:"id"`
	ClienteID      string   `json:"cliente_id"`
	ClienteNombre  string   `json:"cliente_nombre"`
	ImpresoraAlias string   `json:"impresora_alias"`
	Virtual        bool     `json:"virtual"`
	Referencia     string   `json:"referencia"`
	Estado         string   `json:"estado"`
	CodigoError    string   `json:"codigo_error"`
	Advertencias   []string `json:"advertencias"`
	DuracionMS     *int64   `json:"duracion_ms"`
	Creado         string   `json:"creado"`
}

var nombresEspeciales = map[string]string{ClientePanel: "Panel", "cli_consola": "Consola (CLI)"}

func (a *Admin) trabajosJSON(r *http.Request, f almacen.FiltroTrabajos) ([]trabajoJSON, error) {
	lista, err := a.op.Almacen.Trabajos.Listar(r.Context(), f)
	if err != nil {
		return nil, err
	}
	nombres := map[string]string{}
	clientes, err := a.op.Almacen.Clientes.Listar(r.Context())
	if err != nil {
		return nil, err
	}
	for _, c := range clientes {
		nombres[c.ID] = c.Nombre
	}
	resp := []trabajoJSON{}
	for _, t := range lista {
		j := trabajoJSON{ID: t.ID, ClienteID: t.ClienteID, ImpresoraAlias: t.ImpresoraAlias, Virtual: t.Virtual, Referencia: t.Referencia,
			Estado: t.Estado, CodigoError: t.CodigoError, Advertencias: t.Advertencias, Creado: fecha(t.Creado)}
		if j.ClienteNombre = nombres[t.ClienteID]; j.ClienteNombre == "" {
			if j.ClienteNombre = nombresEspeciales[t.ClienteID]; j.ClienteNombre == "" {
				j.ClienteNombre = t.ClienteID
			}
		}
		if j.Advertencias == nil {
			j.Advertencias = []string{}
		}
		if !t.Finalizado.IsZero() {
			ms := t.Duracion.Milliseconds()
			j.DuracionMS = &ms
		}
		resp = append(resp, j)
	}
	return resp, nil
}

func (a *Admin) listarTrabajos(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := almacen.FiltroTrabajos{ClienteID: q.Get("cliente"), ImpresoraAlias: q.Get("impresora"), Estado: q.Get("estado"), Limite: 500}
	// Las fechas llegan como días (AAAA-MM-DD) en la hora local del puente; «hasta» incluye el día.
	if d := q.Get("desde"); d != "" {
		t, err := time.ParseInLocation(time.DateOnly, d, time.Local)
		if err != nil {
			a.error(w, r, invalido("La fecha «desde» debe tener el formato AAAA-MM-DD."))
			return
		}
		f.Desde = t
	}
	if d := q.Get("hasta"); d != "" {
		t, err := time.ParseInLocation(time.DateOnly, d, time.Local)
		if err != nil {
			a.error(w, r, invalido("La fecha «hasta» debe tener el formato AAAA-MM-DD."))
			return
		}
		f.Hasta = t.AddDate(0, 0, 1)
	}
	lista, err := a.trabajosJSON(r, f)
	if err != nil {
		a.error(w, r, err)
		return
	}
	responder(w, lista)
}

// --- Imágenes -----------------------------------------------------------------------------------

var patronRecurso = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

func (a *Admin) listarRecursos(w http.ResponseWriter, r *http.Request) {
	lista, err := a.op.Almacen.Recursos.Listar(r.Context())
	if err != nil {
		a.error(w, r, err)
		return
	}
	type recursoJSON struct {
		Nombre  string `json:"nombre"`
		AnchoPx int    `json:"ancho_px"`
		AltoPx  int    `json:"alto_px"`
		Bytes   int    `json:"bytes"`
	}
	resp := []recursoJSON{}
	for _, rec := range lista {
		// El listado no trae el PNG; el tamaño se consulta aparte (son pocas imágenes).
		completo, err := a.op.Almacen.Recursos.Obtener(r.Context(), rec.Nombre)
		if err != nil {
			a.error(w, r, err)
			return
		}
		resp = append(resp, recursoJSON{rec.Nombre, rec.AnchoPx, rec.AltoPx, len(completo.PNG)})
	}
	responder(w, resp)
}

// leerPNG obtiene el archivo del formulario y comprueba que sea un PNG dentro del límite.
func leerPNG(w http.ResponseWriter, r *http.Request) ([]byte, int, int, error) {
	r.Body = http.MaxBytesReader(w, r.Body, almacen.MaxBytesRecurso+64<<10)
	archivo, _, err := r.FormFile("archivo")
	if err != nil {
		return nil, 0, 0, invalido(fmt.Sprintf("Elige un archivo PNG de hasta %d KB.", almacen.MaxBytesRecurso/1024))
	}
	defer archivo.Close()
	datos, err := io.ReadAll(io.LimitReader(archivo, almacen.MaxBytesRecurso+1))
	if err != nil || len(datos) > almacen.MaxBytesRecurso {
		return nil, 0, 0, invalido(fmt.Sprintf("La imagen supera los %d KB.", almacen.MaxBytesRecurso/1024))
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(datos))
	if err != nil {
		return nil, 0, 0, rechazo(http.StatusBadRequest, "IMAGEN_INVALIDA", "El archivo no es un PNG válido.", "Expórtalo como PNG desde tu editor de imágenes.")
	}
	if cfg.Width > 4096 || cfg.Height > 4096 {
		return nil, 0, 0, rechazo(http.StatusBadRequest, "IMAGEN_INVALIDA", "La imagen mide más de 4096 píxeles por lado.", "Redúcela antes de subirla.")
	}
	return datos, cfg.Width, cfg.Height, nil
}

func (a *Admin) guardarRecurso(w http.ResponseWriter, r *http.Request) {
	datos, ancho, alto, err := leerPNG(w, r)
	if err != nil {
		a.error(w, r, err)
		return
	}
	nombre := strings.TrimSpace(r.FormValue("nombre"))
	if !patronRecurso.MatchString(nombre) {
		a.error(w, r, invalido("El nombre solo admite minúsculas, dígitos, - y _ (hasta 64 caracteres), por ejemplo «logo-tienda»."))
		return
	}
	if _, err := escpos.VistaPrevia1Bit(datos, 8); err != nil {
		a.error(w, r, rechazo(http.StatusBadRequest, "IMAGEN_INVALIDA", "No se pudo leer la imagen.", "Expórtala de nuevo como PNG."))
		return
	}
	if err := a.op.Almacen.Recursos.Guardar(r.Context(), &almacen.Recurso{Nombre: nombre, PNG: datos, AnchoPx: ancho, AltoPx: alto}); err != nil {
		a.error(w, r, err)
		return
	}
	a.op.Log.Info("imagen guardada desde el panel", "recurso", nombre, "bytes", len(datos))
	responder(w, ok)
}

func anchoVista(v string) int {
	n, err := strconv.Atoi(v)
	if err != nil || n < 8 || n > 1024 {
		return 576
	}
	return n
}

func responderPNG(w http.ResponseWriter, datos []byte) {
	w.Header().Set("Content-Type", "image/png")
	w.Write(datos)
}

func (a *Admin) vistaPreviaRecurso(w http.ResponseWriter, r *http.Request) {
	datos, _, _, err := leerPNG(w, r)
	if err != nil {
		a.error(w, r, err)
		return
	}
	vista, err := escpos.VistaPrevia1Bit(datos, anchoVista(r.FormValue("ancho")))
	if err != nil {
		a.error(w, r, rechazo(http.StatusBadRequest, "IMAGEN_INVALIDA", "No se pudo leer la imagen.", "Expórtala de nuevo como PNG."))
		return
	}
	responderPNG(w, vista)
}

func (a *Admin) imagenRecurso(w http.ResponseWriter, r *http.Request) {
	rec, err := a.op.Almacen.Recursos.Obtener(r.Context(), r.PathValue("nombre"))
	if err != nil {
		a.error(w, r, err)
		return
	}
	vista, err := escpos.VistaPrevia1Bit(rec.PNG, anchoVista(r.URL.Query().Get("ancho")))
	if err != nil {
		a.error(w, r, err)
		return
	}
	responderPNG(w, vista)
}

func (a *Admin) eliminarRecurso(w http.ResponseWriter, r *http.Request) {
	if err := a.op.Almacen.Recursos.Eliminar(r.Context(), r.PathValue("nombre")); err != nil {
		a.error(w, r, err)
		return
	}
	responder(w, ok)
}

// --- Ajustes ------------------------------------------------------------------------------------

func (a *Admin) entero(r *http.Request, clave string, porDefecto int) int {
	v, ok2, _ := a.op.Almacen.Ajustes.Obtener(r.Context(), clave)
	if n, err := strconv.Atoi(v); ok2 && err == nil {
		return n
	}
	return porDefecto
}

func (a *Admin) soloRedPrivada(r *http.Request) bool {
	v, _, _ := a.op.Almacen.Ajustes.Obtener(r.Context(), almacen.AjusteSoloRedPrivada)
	return v != "0"
}

// buscarActualizaciones: el puente consulta GitHub una vez al día salvo que el dueño lo desactive.
func (a *Admin) buscarActualizaciones(r *http.Request) bool {
	v, _, _ := a.op.Almacen.Ajustes.Obtener(r.Context(), almacen.AjusteBuscarActualizaciones)
	return v != "0"
}

func (a *Admin) leerAjustes(w http.ResponseWriter, r *http.Request) {
	retencion, err := a.op.Almacen.Ajustes.RetencionDias(r.Context())
	if err != nil {
		retencion = almacen.RetencionDiasPorDefecto
	}
	lim := a.op.Config.Limites
	resp := map[string]any{
		"nombre_puente":          a.nombrePuente(r),
		"limite_ip":              a.entero(r, servidor.AjusteLimiteIP, lim.PorIP),
		"limite_cliente":         a.entero(r, servidor.AjusteLimiteCliente, lim.PorCliente),
		"retencion_dias":         retencion,
		"solo_red_privada":       a.soloRedPrivada(r),
		"buscar_actualizaciones": a.buscarActualizaciones(r),
		"direccion":              a.direccion(),
		"tls":                    a.estadoTLS(),
	}
	responder(w, resp)
}

func (a *Admin) rutaCertificado() string {
	cfg := a.op.Config
	switch cfg.Servidor.ModoTLS {
	case config.TLSAutofirmado:
		cert, _ := servidor.RutasAutofirmado(cfg.DirDatos())
		return cert
	case config.TLSArchivos:
		return cfg.Servidor.Certificado
	}
	return ""
}

func (a *Admin) estadoTLS() map[string]any {
	resp := map[string]any{"modo": a.op.Config.Servidor.ModoTLS}
	ruta := a.rutaCertificado()
	if ruta == "" {
		return resp
	}
	c, err := servidor.LeerCertificado(ruta)
	if err != nil {
		return resp
	}
	ips, sinCubrir := []string{}, []string{}
	for _, ip := range c.IPAddresses {
		ips = append(ips, ip.String())
	}
	for _, ip := range servidor.IPsSinCubrir(c) {
		sinCubrir = append(sinCubrir, ip.String())
	}
	estado, dias := servidor.Vencimiento(c, a.op.Ahora())
	resp["certificado"] = map[string]any{"vence": c.NotAfter.Format("02/01/2006"), "nombres": c.DNSNames, "ips": ips,
		"ips_sin_cubrir": sinCubrir, "huella": servidor.Huella(c), "estado": estado, "dias_restantes": dias}
	if a.op.Config.Servidor.ModoTLS == config.TLSAutofirmado {
		// Los de 0.2.0 no están limitados a la red local: prisma tls confiar los rechaza (ADR 0020).
		resp["certificado"].(map[string]any)["limitado"] = servidor.LimitadoALaRedLocal(c)
	}
	return resp
}

// avisoCertificado resume el vencimiento para Inicio: nil si el certificado está vigente.
func (a *Admin) avisoCertificado() any {
	cert, ok := a.estadoTLS()["certificado"].(map[string]any)
	if !ok || cert["estado"] == servidor.CertificadoVigente {
		return nil
	}
	return map[string]any{"estado": cert["estado"], "vence": cert["vence"], "dias_restantes": cert["dias_restantes"],
		"autofirmado": a.op.Config.Servidor.ModoTLS == config.TLSAutofirmado}
}

func (a *Admin) guardarAjustes(w http.ResponseWriter, r *http.Request) {
	var d struct {
		NombrePuente          *string `json:"nombre_puente"`
		LimiteIP              *int    `json:"limite_ip"`
		LimiteCliente         *int    `json:"limite_cliente"`
		RetencionDias         *int    `json:"retencion_dias"`
		SoloRedPrivada        *bool   `json:"solo_red_privada"`
		BuscarActualizaciones *bool   `json:"buscar_actualizaciones"`
	}
	if err := leerJSON(w, r, &d); err != nil {
		a.error(w, r, err)
		return
	}
	cambios := map[string]string{}
	if d.NombrePuente != nil {
		n := strings.TrimSpace(*d.NombrePuente)
		if n == "" || len([]rune(n)) > 60 {
			a.error(w, r, invalido("El nombre del puente debe tener entre 1 y 60 caracteres."))
			return
		}
		cambios[servidor.AjusteNombrePuente] = n
	}
	if (d.LimiteIP == nil) != (d.LimiteCliente == nil) {
		a.error(w, r, invalido("Envía los dos límites juntos."))
		return
	}
	if d.LimiteIP != nil {
		if *d.LimiteIP < 0 || *d.LimiteIP > 100000 || *d.LimiteCliente < 0 || *d.LimiteCliente > 100000 {
			a.error(w, r, invalido("Los límites van de 0 (sin límite) a 100000 peticiones por minuto."))
			return
		}
		cambios[servidor.AjusteLimiteIP], cambios[servidor.AjusteLimiteCliente] = strconv.Itoa(*d.LimiteIP), strconv.Itoa(*d.LimiteCliente)
	}
	if d.RetencionDias != nil {
		if *d.RetencionDias < 1 || *d.RetencionDias > 3650 {
			a.error(w, r, invalido("El historial se conserva de 1 a 3650 días."))
			return
		}
		cambios[almacen.AjusteRetencionDias] = strconv.Itoa(*d.RetencionDias)
	}
	if d.SoloRedPrivada != nil {
		cambios[almacen.AjusteSoloRedPrivada] = map[bool]string{true: "1", false: "0"}[*d.SoloRedPrivada]
	}
	if d.BuscarActualizaciones != nil {
		cambios[almacen.AjusteBuscarActualizaciones] = map[bool]string{true: "1", false: "0"}[*d.BuscarActualizaciones]
	}
	if err := a.op.Almacen.Ajustes.EstablecerVarios(r.Context(), cambios); err != nil {
		a.error(w, r, err)
		return
	}
	if d.LimiteIP != nil {
		a.op.Servidor.ActualizarLimites(*d.LimiteIP, *d.LimiteCliente)
	}
	a.op.Log.Info("ajustes cambiados desde el panel", "claves", len(cambios))
	responder(w, ok)
}

// regenerarTLS crea otro certificado autofirmado con los nombres e IPs actuales; el servidor lo
// toma al detectar el cambio de archivo, sin reiniciar.
func (a *Admin) regenerarTLS(w http.ResponseWriter, r *http.Request) {
	if a.op.Config.Servidor.ModoTLS != config.TLSAutofirmado {
		a.error(w, r, rechazo(http.StatusConflict, "TLS_NO_AUTOFIRMADO", "Solo se puede regenerar el certificado autofirmado.",
			"Con certificado propio, reemplaza los archivos configurados en config.toml."))
		return
	}
	if _, err := servidor.GenerarAutofirmado(a.op.Config.DirDatos()); err != nil {
		a.error(w, r, err)
		return
	}
	a.op.Log.Info("certificado autofirmado regenerado desde el panel")
	responder(w, ok)
}

func (a *Admin) descargarCertificado(w http.ResponseWriter, r *http.Request) {
	ruta := a.rutaCertificado()
	datos, err := os.ReadFile(ruta)
	if ruta == "" || err != nil {
		a.error(w, r, rechazo(http.StatusNotFound, "SIN_CERTIFICADO", "Este puente no usa un certificado.", ""))
		return
	}
	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Header().Set("Content-Disposition", `attachment; filename="prisma.crt"`)
	w.Write(datos)
}

// --- Respaldo -----------------------------------------------------------------------------------

func (a *Admin) respaldo(w http.ResponseWriter, r *http.Request) {
	dir, err := os.MkdirTemp("", "prisma-respaldo-")
	if err != nil {
		a.error(w, r, err)
		return
	}
	defer os.RemoveAll(dir)
	ruta := filepath.Join(dir, "respaldo.db")
	if err := a.op.Almacen.Respaldar(r.Context(), ruta); err != nil {
		a.error(w, r, err)
		return
	}
	f, err := os.Open(ruta)
	if err != nil {
		a.error(w, r, err)
		return
	}
	defer f.Close()
	nombre := "prisma-respaldo-" + a.op.Ahora().Format("2006-01-02-1504") + ".db"
	w.Header().Set("Content-Type", "application/vnd.sqlite3")
	w.Header().Set("Content-Disposition", `attachment; filename="`+nombre+`"`)
	io.Copy(w, f)
	a.op.Log.Info("respaldo descargado desde el panel")
}

// restaurar reemplaza la base por el respaldo. Todas las sesiones terminan: la contraseña pudo
// cambiar con el respaldo.
func (a *Admin) restaurar(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 512<<20)
	archivo, _, err := r.FormFile("archivo")
	if err != nil {
		a.error(w, r, invalido("Elige el archivo de respaldo (.db) que descargaste del panel."))
		return
	}
	defer archivo.Close()
	dir, err := os.MkdirTemp("", "prisma-restaurar-")
	if err != nil {
		a.error(w, r, err)
		return
	}
	defer os.RemoveAll(dir)
	ruta := filepath.Join(dir, "respaldo.db")
	f, err := os.Create(ruta)
	if err != nil {
		a.error(w, r, err)
		return
	}
	_, err = io.Copy(f, archivo)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		a.error(w, r, err)
		return
	}
	ctx := r.Context()
	if err := a.op.Almacen.Restaurar(ctx, ruta); err != nil {
		if errors.Is(err, almacen.ErrRespaldoInvalido) {
			err = rechazo(http.StatusBadRequest, "RESPALDO_INVALIDO", "El archivo no es un respaldo válido de PRISMA.",
				"Usa un archivo descargado desde Ajustes → Respaldo de este puente o de una versión igual o anterior.")
		}
		a.error(w, r, err)
		return
	}
	if err := a.op.Servidor.SincronizarImpresoras(ctx); err != nil {
		a.op.Log.Error("no se pudieron registrar las impresoras restauradas", "error", err.Error())
	}
	if err := a.op.Servidor.CargarAjustes(ctx); err != nil {
		a.op.Log.Error("no se pudieron cargar los ajustes restaurados", "error", err.Error())
	}
	a.sesiones.cerrarTodas()
	borrarCookie(w, r)
	a.op.Log.Warn("base de datos restaurada desde un respaldo")
	responder(w, ok)
}

// pasosDePrueba dice si ya se imprimió un ticket de prueba y si ya se probaron los acentos en una
// impresora física. Si las marcas aún no existen (un puente que viene de 0.1.0), los deduce de las
// pruebas del panel que quedan en el historial y las guarda.
func (a *Admin) pasosDePrueba(r *http.Request) (prueba, acentos bool) {
	ctx := r.Context()
	marcado := func(clave string) bool { v, _, _ := a.op.Almacen.Ajustes.Obtener(ctx, clave); return v == "1" }
	prueba, acentos = marcado(almacen.AjustePrimeraPrueba), marcado(almacen.AjusteAcentosProbados)
	if prueba && acentos {
		return
	}
	lista, err := a.op.Almacen.Trabajos.Listar(ctx, almacen.FiltroTrabajos{ClienteID: ClientePanel, Estado: almacen.TrabajoCompletado, Limite: 500})
	if err != nil {
		return
	}
	for _, t := range lista {
		if !prueba && t.Referencia == "DIAGNOSTICO" {
			prueba = true
			a.marcarPaso(r, almacen.AjustePrimeraPrueba)
		}
		if !acentos && t.Referencia == "PAGINAS" && !t.Virtual {
			acentos = true
			a.marcarPaso(r, almacen.AjusteAcentosProbados)
		}
	}
	return
}
