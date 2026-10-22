package admin

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/cola"
	"github.com/bayalad7/prisma/internal/conector"
	"github.com/bayalad7/prisma/internal/contrato"
	"github.com/bayalad7/prisma/internal/dev"
	"github.com/bayalad7/prisma/internal/escpos"
	"github.com/bayalad7/prisma/internal/servidor"
)

var patronAlias = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

// ClientePanel identifica en el historial los trabajos que imprime el propio panel.
const ClientePanel = "cli_panel"

// destinoVirtual es el destino de las impresoras virtuales: los tickets quedan en la memoria del puente.
const destinoVirtual = "memoria"

type impresoraJSON struct {
	Alias                     string      `json:"alias"`
	Nombre                    string      `json:"nombre"`
	Tipo                      string      `json:"tipo"`
	Destino                   string      `json:"destino"`
	AnchoMM                   int         `json:"ancho_mm"`
	Columnas                  int         `json:"columnas"`
	ColumnasFuenteB           int         `json:"columnas_fuente_b"`
	AnchoPx                   int         `json:"ancho_px"`
	PaginaCodigos             string      `json:"pagina_codigos"`
	ComandoPaginaCodigos      int         `json:"comando_pagina_codigos"`
	SoportaQRNativo           bool        `json:"soporta_qr_nativo"`
	SoportaCorte              bool        `json:"soporta_corte"`
	SoportaCajon              bool        `json:"soporta_cajon"`
	SoportaImagen             bool        `json:"soporta_imagen"`
	SoportaCodigoBarrasNativo bool        `json:"soporta_codigo_barras_nativo"`
	ConfirmarImpresion        bool        `json:"confirmar_impresion"`
	Baudios                   int         `json:"baudios"`
	Activa                    bool        `json:"activa"`
	Virtual                   bool        `json:"virtual"`
	Simulacion                string      `json:"simulacion"`
	SimulacionRetrasoMS       int         `json:"simulacion_retraso_ms"`
	SimulacionFallaPct        int         `json:"simulacion_falla_pct"`
	TicketsEnMemoria          int         `json:"tickets_en_memoria"`
	TicketsRecibidos          int         `json:"tickets_recibidos"` // los que hay ahora en su rollo
	Estado                    cola.Estado `json:"estado,omitempty"`
	Pendientes                int         `json:"pendientes"`
	UltimoError               string      `json:"ultimo_error,omitempty"`
}

func (a *Admin) aJSON(imp almacen.Impresora) impresoraJSON {
	j := impresoraJSON{
		Alias: imp.Alias, Nombre: imp.Nombre, Tipo: imp.Tipo, Destino: imp.Destino, AnchoMM: imp.AnchoMM,
		Columnas: imp.Columnas, ColumnasFuenteB: imp.ColumnasFuenteB, AnchoPx: imp.AnchoPx,
		PaginaCodigos: imp.PaginaCodigos, ComandoPaginaCodigos: imp.ComandoPaginaCodigos,
		SoportaQRNativo: imp.SoportaQRNativo, SoportaCorte: imp.SoportaCorte, SoportaCajon: imp.SoportaCajon,
		SoportaImagen: imp.SoportaImagen, SoportaCodigoBarrasNativo: imp.SoportaCodigoBarrasNativo,
		ConfirmarImpresion: imp.ConfirmarImpresion, Baudios: imp.Baudios, Activa: imp.Activa, Estado: cola.Desconocido,
		Virtual: imp.Tipo == almacen.ConexionVirtual, Simulacion: imp.Simulacion, SimulacionRetrasoMS: imp.SimulacionRetrasoMS,
		SimulacionFallaPct: imp.SimulacionFallaPct, TicketsEnMemoria: imp.TicketsEnMemoria,
	}
	if j.Virtual {
		j.TicketsRecibidos = len(conector.RollosVirtuales.Listar(imp.Alias))
	}
	if !imp.Activa {
		j.Estado = "inactiva"
	} else if info, ok := a.op.Cola.EstadoDe(imp.Alias); ok {
		j.Estado, j.Pendientes, j.UltimoError = info.Estado, info.Pendientes, info.UltimoError
	}
	return j
}

// aImpresora valida los datos del formulario con mensajes para el dueño (la base solo devolvería
// «datos inválidos»).
func (j impresoraJSON) aImpresora() (almacen.Impresora, error) {
	imp := almacen.Impresora{
		Alias: strings.TrimSpace(j.Alias), Nombre: strings.TrimSpace(j.Nombre), Tipo: j.Tipo, Destino: strings.TrimSpace(j.Destino),
		AnchoMM: j.AnchoMM, Columnas: j.Columnas, ColumnasFuenteB: j.ColumnasFuenteB, AnchoPx: j.AnchoPx,
		PaginaCodigos: strings.ToUpper(j.PaginaCodigos), ComandoPaginaCodigos: j.ComandoPaginaCodigos,
		SoportaQRNativo: j.SoportaQRNativo, SoportaCorte: j.SoportaCorte, SoportaCajon: j.SoportaCajon,
		SoportaImagen: j.SoportaImagen, SoportaCodigoBarrasNativo: j.SoportaCodigoBarrasNativo,
		ConfirmarImpresion: j.ConfirmarImpresion, Baudios: j.Baudios, Activa: j.Activa,
		Simulacion: j.Simulacion, SimulacionRetrasoMS: j.SimulacionRetrasoMS, SimulacionFallaPct: j.SimulacionFallaPct,
		TicketsEnMemoria: j.TicketsEnMemoria,
	}
	if imp.Tipo == almacen.ConexionVirtual {
		imp.Destino = destinoVirtual // vive dentro del puente: no tiene dirección
	}
	if imp.Nombre == "" {
		imp.Nombre = imp.Alias
	}
	switch {
	case !patronAlias.MatchString(imp.Alias):
		return imp, invalido("El alias solo admite minúsculas, dígitos, - y _ (hasta 32 caracteres), por ejemplo «caja».")
	case !slices.Contains([]string{almacen.ConexionRed, almacen.ConexionUSB, almacen.ConexionBluetooth, almacen.ConexionArchivo, almacen.ConexionVirtual}, imp.Tipo):
		return imp, invalido("Elige el tipo de conexión: red, USB, Bluetooth, archivo o virtual.")
	case imp.Destino == "":
		return imp, invalido("Falta el destino: la IP, el nombre de la impresora, el puerto serie o la carpeta.")
	case imp.AnchoMM != 58 && imp.AnchoMM != 80:
		return imp, invalido("El ancho del papel debe ser 58 u 80 mm.")
	case imp.Columnas < 16 || imp.Columnas > 96 || imp.ColumnasFuenteB < imp.Columnas || imp.ColumnasFuenteB > 128:
		return imp, invalido("Revisa las columnas: entre 16 y 96 con la fuente A, y al menos las mismas con la fuente B.")
	case imp.AnchoPx < 128 || imp.AnchoPx > 1024 || imp.AnchoPx%8 != 0:
		return imp, invalido("Los puntos por línea deben ser múltiplo de 8 entre 128 y 1024 (384 en 58 mm, 576 en 80 mm).")
	case !slices.Contains(escpos.PaginasSoportadas(), imp.PaginaCodigos):
		return imp, invalido("Página de códigos no soportada; usa " + strings.Join(escpos.PaginasSoportadas(), ", ") + ".")
	case imp.ComandoPaginaCodigos < 0 || imp.ComandoPaginaCodigos > 255:
		return imp, invalido("El comando de la página de códigos (ESC t n) va de 0 a 255.")
	}
	if imp.Tipo == almacen.ConexionBluetooth {
		if imp.Baudios == 0 {
			imp.Baudios = 9600
		}
		if imp.Baudios < 1200 || imp.Baudios > 921600 {
			return imp, invalido("La velocidad del puerto serie va de 1200 a 921600 baudios.")
		}
	} else {
		imp.Baudios = 0
	}
	// Solo los conectores de red y Bluetooth pueden leer la respuesta de la impresora.
	if imp.Tipo != almacen.ConexionRed && imp.Tipo != almacen.ConexionBluetooth {
		imp.ConfirmarImpresion = false
	}
	if imp.Tipo != almacen.ConexionVirtual {
		imp.Simulacion, imp.SimulacionRetrasoMS, imp.SimulacionFallaPct, imp.TicketsEnMemoria = almacen.SimulacionNormal, 3000, 30, 20
		return imp, nil
	}
	if imp.Simulacion == "" {
		imp.Simulacion = almacen.SimulacionNormal
	}
	if imp.TicketsEnMemoria == 0 {
		imp.TicketsEnMemoria = 20
	}
	switch {
	case !slices.Contains([]string{almacen.SimulacionNormal, almacen.SimulacionFueraDeLinea, almacen.SimulacionLenta, almacen.SimulacionFallaAleatoria}, imp.Simulacion):
		return imp, invalido("Elige qué simula la impresora virtual: normal, fuera de línea, lenta o falla aleatoria.")
	case imp.SimulacionRetrasoMS < 0 || imp.SimulacionRetrasoMS > 30000:
		return imp, invalido("El retraso de la impresora lenta va de 0 a 30 000 ms (30 segundos).")
	case imp.SimulacionFallaPct < 0 || imp.SimulacionFallaPct > 100:
		return imp, invalido("El porcentaje de fallas va de 0 a 100.")
	case imp.TicketsEnMemoria < 1 || imp.TicketsEnMemoria > 200:
		return imp, invalido("La impresora virtual recuerda de 1 a 200 tickets.")
	}
	return imp, nil
}

func (a *Admin) listarImpresoras(w http.ResponseWriter, r *http.Request) {
	lista, err := a.op.Almacen.Impresoras.Listar(r.Context())
	if err != nil {
		a.error(w, r, err)
		return
	}
	resp := []impresoraJSON{}
	for _, imp := range lista {
		resp = append(resp, a.aJSON(imp))
	}
	responder(w, resp)
}

func (a *Admin) crearImpresora(w http.ResponseWriter, r *http.Request) {
	var j impresoraJSON
	if err := leerJSON(w, r, &j); err != nil {
		a.error(w, r, err)
		return
	}
	imp, err := j.aImpresora()
	if err != nil {
		a.error(w, r, err)
		return
	}
	if err := a.op.Almacen.Impresoras.Crear(r.Context(), &imp); err != nil {
		if errors.Is(err, almacen.ErrDuplicado) {
			err = rechazo(http.StatusConflict, "ALIAS_DUPLICADO", fmt.Sprintf("Ya existe una impresora con el alias «%s».", imp.Alias), "Elige otro alias.")
		}
		a.error(w, r, err)
		return
	}
	a.registrar(imp)
	a.op.Log.Info("impresora agregada desde el panel", "alias", imp.Alias, "tipo", imp.Tipo)
	responder(w, a.aJSON(imp))
}

// registrar deja la cola igual que la base para esa impresora.
func (a *Admin) registrar(imp almacen.Impresora) {
	if !imp.Activa {
		a.op.Servidor.QuitarImpresora(imp.Alias)
		return
	}
	if err := a.op.Servidor.AsegurarRegistrada(imp); err != nil {
		a.op.Log.Warn("no se pudo preparar la impresora", "alias", imp.Alias, "error", err.Error())
	}
}

func (a *Admin) impresora(w http.ResponseWriter, r *http.Request) (*almacen.Impresora, bool) {
	return a.impresoraPorAlias(w, r, r.PathValue("alias"))
}

func (a *Admin) impresoraPorAlias(w http.ResponseWriter, r *http.Request, alias string) (*almacen.Impresora, bool) {
	imp, err := a.op.Almacen.Impresoras.Obtener(r.Context(), alias)
	if errors.Is(err, almacen.ErrNoEncontrado) {
		err = &conector.Error{Codigo: conector.ErrImpresoraNoEncontrada, Mensaje: fmt.Sprintf("No existe la impresora «%s».", alias),
			Sugerencia: "Recarga la lista de impresoras."}
	}
	if err != nil {
		a.error(w, r, err)
		return nil, false
	}
	return imp, true
}

func (a *Admin) actualizarImpresora(w http.ResponseWriter, r *http.Request) {
	previa, ok2 := a.impresora(w, r)
	if !ok2 {
		return
	}
	var j impresoraJSON
	if err := leerJSON(w, r, &j); err != nil {
		a.error(w, r, err)
		return
	}
	imp, err := j.aImpresora()
	if err != nil {
		a.error(w, r, err)
		return
	}
	if previa.Tipo == almacen.ConexionVirtual && imp.Tipo != almacen.ConexionVirtual {
		defer conector.RollosVirtuales.Vaciar(imp.Alias)
	}
	if imp.Alias != previa.Alias {
		a.error(w, r, invalido("El alias no se puede cambiar: las aplicaciones ya lo usan. Crea otra impresora si necesitas otro alias."))
		return
	}
	imp.ID = previa.ID
	if err := a.op.Almacen.Impresoras.Actualizar(r.Context(), &imp); err != nil {
		a.error(w, r, err)
		return
	}
	actual, err := a.op.Almacen.Impresoras.Obtener(r.Context(), imp.Alias)
	if err != nil {
		a.error(w, r, err)
		return
	}
	a.registrar(*actual)
	responder(w, a.aJSON(*actual))
}

func (a *Admin) eliminarImpresora(w http.ResponseWriter, r *http.Request) {
	imp, ok2 := a.impresora(w, r)
	if !ok2 {
		return
	}
	if err := a.op.Almacen.Impresoras.Eliminar(r.Context(), imp.Alias); err != nil {
		a.error(w, r, err)
		return
	}
	a.op.Servidor.QuitarImpresora(imp.Alias)
	conector.RollosVirtuales.Vaciar(imp.Alias)
	a.op.Log.Info("impresora eliminada desde el panel", "alias", imp.Alias)
	responder(w, ok)
}

// probarConector abre una conexión de prueba sin imprimir.
func probarConector(ctx context.Context, imp almacen.Impresora) error {
	con, err := conector.Nuevo(imp)
	if err != nil {
		return err
	}
	return con.Probar(ctx)
}

func (a *Admin) probarImpresora(w http.ResponseWriter, r *http.Request) {
	imp, ok2 := a.impresora(w, r)
	if !ok2 {
		return
	}
	ctx, fin := ctxCorto(r, 10*time.Second)
	defer fin()
	if err := probarConector(ctx, *imp); err != nil {
		a.error(w, r, err)
		return
	}
	responder(w, map[string]any{"ok": true, "mensaje": "La impresora respondió: está en línea."})
}

// probarDestino prueba una conexión antes de guardar la impresora.
func (a *Admin) probarDestino(w http.ResponseWriter, r *http.Request) {
	var d struct {
		Tipo, Destino string
		Baudios       int
	}
	if err := leerJSON(w, r, &d); err != nil {
		a.error(w, r, err)
		return
	}
	j := impresoraJSON{Alias: "prueba", Tipo: d.Tipo, Destino: d.Destino, Baudios: d.Baudios, AnchoMM: 80, Columnas: 48,
		ColumnasFuenteB: 64, AnchoPx: 576, PaginaCodigos: "CP858", ComandoPaginaCodigos: 19}
	imp, err := j.aImpresora()
	if err != nil {
		a.error(w, r, err)
		return
	}
	ctx, fin := ctxCorto(r, 10*time.Second)
	defer fin()
	if err := probarConector(ctx, imp); err != nil {
		a.error(w, r, err)
		return
	}
	responder(w, map[string]any{"ok": true, "mensaje": "La impresora respondió: está lista para imprimir."})
}

func (a *Admin) detectar(w http.ResponseWriter, r *http.Request) {
	lista := conector.Detectar(r.Context(), conector.OpcionesDeteccion{Red: r.URL.Query().Get("red") != ""})
	if lista == nil {
		lista = []conector.Detectada{}
	}
	responder(w, lista)
}

// impresoraParaImprimir obtiene la impresora del camino y exige que esté activa.
func (a *Admin) impresoraParaImprimir(w http.ResponseWriter, r *http.Request) (*almacen.Impresora, bool) {
	return a.impresoraActiva(w, r, r.PathValue("alias"))
}

func (a *Admin) impresoraActiva(w http.ResponseWriter, r *http.Request, alias string) (*almacen.Impresora, bool) {
	imp, ok2 := a.impresoraPorAlias(w, r, alias)
	if ok2 && !imp.Activa {
		a.error(w, r, rechazo(http.StatusConflict, "IMPRESORA_INACTIVA", "La impresora está desactivada.", "Actívala para imprimir en ella."))
		return nil, false
	}
	return imp, ok2
}

// imprimirPrueba encola bytes generados por el panel, guarda el trabajo, espera el resultado y
// responde como /v1/imprimir (más la imagen del ticket); devuelve si se imprimió. advs son las
// advertencias del render.
func (a *Admin) imprimirPrueba(w http.ResponseWriter, r *http.Request, imp *almacen.Impresora, referencia string, datos []byte, advs []escpos.Advertencia) bool {
	if err := a.op.Servidor.AsegurarRegistrada(*imp); err != nil {
		a.error(w, r, err)
		return false
	}
	ctx := r.Context()
	id := servidor.NuevoTrabajoID()
	t := &almacen.Trabajo{ID: id, DocumentoID: id, ClienteID: ClientePanel, ImpresoraAlias: imp.Alias,
		Virtual: imp.Tipo == almacen.ConexionVirtual, Referencia: referencia}
	for _, adv := range advs {
		t.Advertencias = append(t.Advertencias, adv.Codigo)
	}
	if err := a.op.Almacen.Trabajos.Crear(ctx, t); err != nil {
		a.error(w, r, err)
		return false
	}
	canal, err := a.op.Cola.Encolar(imp.Alias, id, datos)
	var res cola.Resultado
	if err != nil {
		res = cola.Resultado{TrabajoID: id, Err: err}
	} else {
		res = <-canal // la cola limita cada trabajo a 60 s
	}
	estado, codigo := almacen.TrabajoCompletado, ""
	if !res.OK {
		estado, codigo = almacen.TrabajoFallido, conector.Codigo(res.Err)
		if codigo == "" {
			codigo = "ERROR_INTERNO"
		}
	}
	advertencias := t.Advertencias
	if res.Aviso != nil {
		advertencias = append(advertencias, res.Aviso.Codigo)
		advs = append(advs, escpos.Advertencia{Codigo: res.Aviso.Codigo, Mensaje: res.Aviso.Mensaje})
	}
	if advs == nil {
		advs = []escpos.Advertencia{}
	}
	if err := a.op.Almacen.Trabajos.Finalizar(context.WithoutCancel(ctx), id, estado, codigo, advertencias, res.Duracion); err != nil {
		a.op.Log.Error("no se pudo guardar el resultado del trabajo", "trabajo", id, "error", err.Error())
	}
	a.op.Log.Info("trabajo", "trabajo", id, "cliente", ClientePanel, "impresora", imp.Alias, "virtual", t.Virtual, "estado", estado,
		"codigo_error", codigo, "duracion_ms", res.Duracion.Milliseconds())
	if !res.OK {
		servidor.ResponderError(w, r, res.Err, id, a.op.Log)
		return false
	}
	pagina := dev.Simular(datos, *imp)
	resp := map[string]any{"ok": true, "trabajo_id": id, "impresora": imp.Alias, "duracion_ms": res.Duracion.Milliseconds(),
		"advertencias": advs, "vista": dev.VistaTexto(pagina, false),
		"imagen": "data:image/png;base64," + base64.StdEncoding.EncodeToString(dev.RasterPNG(pagina, imp.AnchoMM))}
	if res.Aviso != nil {
		resp["aviso"] = res.Aviso.Mensaje
	}
	responder(w, resp)
	return true
}

func (a *Admin) diagnostico(w http.ResponseWriter, r *http.Request) {
	imp, ok2 := a.impresoraParaImprimir(w, r)
	if !ok2 {
		return
	}
	doc := contrato.Documento{Version: "1", Impresora: imp.Alias, Copias: 1, Operaciones: ticketDiagnostico(*imp, a.op.Ahora())}
	datos, _, err := escpos.Renderizar(doc, *imp, escpos.RecursosMapa{})
	if err != nil {
		a.error(w, r, err)
		return
	}
	if a.imprimirPrueba(w, r, imp, "DIAGNOSTICO", datos, nil) {
		a.marcarPaso(r, almacen.AjustePrimeraPrueba)
	}
}

// marcarPaso guarda que se completó un paso de «Primeros pasos» (un error no afecta la respuesta).
func (a *Admin) marcarPaso(r *http.Request, clave string) {
	if err := a.op.Almacen.Ajustes.Establecer(context.WithoutCancel(r.Context()), clave, "1"); err != nil {
		a.op.Log.Warn("no se pudo guardar el paso de primeros pasos", "paso", clave, "error", err.Error())
	}
}

// ticketDiagnostico describe la impresora y prueba el ancho, los acentos, la alineación y el QR.
func ticketDiagnostico(imp almacen.Impresora, ahora time.Time) []contrato.Operacion {
	texto := func(v string, al contrato.Alineacion, negrita bool) *contrato.Texto {
		return &contrato.Texto{Valor: v, Alineacion: al, Negrita: negrita, Tamano: [2]int{1, 1}, Fuente: "A"}
	}
	par := func(i, d string) *contrato.Par { return &contrato.Par{Izquierda: i, Derecha: d} }
	var regla strings.Builder
	for k := 1; k <= imp.Columnas; k++ {
		if k%10 == 0 {
			regla.WriteByte(byte('0' + (k/10)%10))
		} else {
			regla.WriteByte('.')
		}
	}
	ops := []contrato.Operacion{
		&contrato.Texto{Valor: "PRUEBA DE IMPRESORA", Alineacion: contrato.Centro, Negrita: true, Tamano: [2]int{1, 2}, Fuente: "A"},
		&contrato.Separador{Caracter: "="},
		par("Alias", imp.Alias),
		par("Nombre", imp.Nombre),
		par("Conexión", conexionDiagnostico(imp)),
		par("Papel", fmt.Sprintf("%d mm, %d columnas", imp.AnchoMM, imp.Columnas)),
		par("Página", fmt.Sprintf("%s (ESC t %d)", imp.PaginaCodigos, imp.ComandoPaginaCodigos)),
		par("Fecha", ahora.Format("02/01/2006 15:04")),
		&contrato.Separador{Caracter: "-"},
		texto("Regla de columnas (una sola línea):", contrato.Izquierda, false),
		texto(regla.String(), contrato.Izquierda, false),
		&contrato.Salto{Lineas: 1},
		texto("Acentos: "+escpos.LineaPruebaPaginas, contrato.Izquierda, false),
		&contrato.Salto{Lineas: 1},
		texto("Izquierda", contrato.Izquierda, false),
		texto("Centro", contrato.Centro, false),
		texto("Derecha", contrato.Derecha, false),
		&contrato.Separador{Caracter: "-"},
	}
	if imp.SoportaQRNativo || imp.SoportaImagen {
		ops = append(ops, &contrato.QR{Valor: "PRISMA " + imp.Alias, Tamano: 5, Correccion: "M", Alineacion: contrato.Centro})
	}
	return append(ops, texto("PRISMA · puente de impresión", contrato.Centro, false), &contrato.Cortar{})
}

// conexionDiagnostico describe la conexión en el ticket de diagnóstico.
func conexionDiagnostico(imp almacen.Impresora) string {
	if imp.Tipo == almacen.ConexionVirtual {
		return "virtual (sin papel)"
	}
	return imp.Tipo + " " + imp.Destino
}

func (a *Admin) paginasCodigos(w http.ResponseWriter, r *http.Request) {
	imp, ok2 := a.impresoraParaImprimir(w, r)
	if !ok2 {
		return
	}
	datos, err := escpos.PruebaPaginas(*imp)
	if err != nil {
		a.error(w, r, err)
		return
	}
	if a.imprimirPrueba(w, r, imp, "PAGINAS", datos, nil) && imp.Tipo != almacen.ConexionVirtual {
		a.marcarPaso(r, almacen.AjusteAcentosProbados)
	}
}
