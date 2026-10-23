package servidor

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/cola"
	"github.com/bayalad7/prisma/internal/conector"
	"github.com/bayalad7/prisma/internal/contrato"
	"github.com/bayalad7/prisma/internal/dev"
	"github.com/bayalad7/prisma/internal/escpos"
	"github.com/bayalad7/prisma/internal/seguridad"
)

// GET /v1/salud: sin autenticación ni datos sensibles. Un origen desconocido queda registrado
// como solicitud pendiente, pero no recibe cabeceras CORS.
func (s *Servidor) salud(w http.ResponseWriter, r *http.Request) {
	if origen := r.Header.Get("Origin"); origen != "" {
		if _, err := s.aut.ClienteDeOrigen(r.Context(), origen); err == nil {
			permitirCORS(w, origen)
		}
	}
	nombre, ok, _ := s.op.Almacen.Ajustes.Obtener(r.Context(), AjusteNombrePuente)
	if !ok || nombre == "" {
		nombre = "PRISMA"
	}
	responderJSON(w, http.StatusOK, map[string]any{"ok": true, "version": s.op.Version, "nombre_puente": nombre})
}

// identificar aplica los pasos 1 y 2, el CORS y el límite por cliente. Devuelve nil si la
// petición no trae origen ni llave (solo válido para documentos firmados).
func (s *Servidor) identificar(w http.ResponseWriter, r *http.Request) (*almacen.Cliente, bool) {
	origen := r.Header.Get("Origin")
	cli, err := s.aut.Identificar(r.Context(), origen, r.Header.Get("Authorization"))
	if err != nil {
		s.responderError(w, r, err)
		return nil, false
	}
	if cli != nil {
		permitirCORS(w, origen)
		if !s.limitarCliente(w, r, cli) {
			return nil, false
		}
	}
	return cli, true
}

type impresoraVisible struct {
	Alias   string      `json:"alias"`
	Nombre  string      `json:"nombre"`
	AnchoMM int         `json:"ancho_mm"`
	Estado  cola.Estado `json:"estado"`
	Virtual bool        `json:"virtual"` // no imprime en papel: el ticket se ve en el panel del puente
}

// GET /v1/impresoras: solo las impresoras activas que el cliente puede usar.
func (s *Servidor) impresoras(w http.ResponseWriter, r *http.Request) {
	cli, ok := s.identificar(w, r)
	if !ok {
		return
	}
	if cli == nil {
		s.responderError(w, r, rechazo(http.StatusUnauthorized, seguridad.ErrCredencialesRequeridas,
			"La petición no identifica a ninguna aplicación autorizada.",
			"Desde un navegador, usa un origen aprobado; desde un servidor, envía la llave (Bearer pk_…)."))
		return
	}
	lista, err := s.op.Almacen.Impresoras.Listar(r.Context())
	if err != nil {
		s.responderError(w, r, err)
		return
	}
	visibles := []impresoraVisible{}
	for _, imp := range lista {
		if !imp.Activa {
			continue
		}
		permitida, err := s.op.Almacen.Clientes.PuedeImprimirEn(r.Context(), cli.ID, imp.Alias)
		if err != nil {
			s.responderError(w, r, err)
			return
		}
		if !permitida {
			continue
		}
		estado := cola.Desconocido
		if info, ok := s.op.Cola.EstadoDe(imp.Alias); ok {
			estado = info.Estado
		}
		visibles = append(visibles, impresoraVisible{Alias: imp.Alias, Nombre: imp.Nombre, AnchoMM: imp.AnchoMM, Estado: estado,
			Virtual: imp.Tipo == almacen.ConexionVirtual})
	}
	responderJSON(w, http.StatusOK, visibles)
}

// autorizarDocumento aplica los pasos comunes a imprimir y previsualizar: identificación, lectura
// del cuerpo, firma y permisos. Nada costoso (esquema completo, imágenes, render) ocurre antes.
// Con idempotencia = true también aplica el paso 4, que puede responder directamente.
func (s *Servidor) autorizarDocumento(w http.ResponseWriter, r *http.Request, idempotencia bool) (*almacen.Cliente, *contrato.Sobre, []byte, bool) {
	cli, ok := s.identificar(w, r)
	if !ok {
		return nil, nil, nil, false
	}
	cuerpo, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxBytesCuerpo))
	if err != nil {
		s.responderError(w, r, err)
		return nil, nil, nil, false
	}
	sobre, err := contrato.LeerSobre(cuerpo)
	if err != nil {
		s.responderError(w, r, err)
		return nil, nil, nil, false
	}
	identificado := cli != nil
	if cli, err = s.aut.VerificarDocumento(r.Context(), cli, sobre); err != nil {
		s.responderError(w, r, err)
		return nil, nil, nil, false
	}
	if !identificado && !s.limitarCliente(w, r, cli) {
		return nil, nil, nil, false
	}
	if idempotencia && s.responderSiRepetido(w, r, cli, sobre) {
		return nil, nil, nil, false
	}
	if err := s.aut.VerificarPermisos(r.Context(), cli, sobre); err != nil {
		s.responderError(w, r, err)
		return nil, nil, nil, false
	}
	return cli, sobre, cuerpo, true
}

// responderSiRepetido aplica la idempotencia (contrato §8): el mismo id con el mismo contenido
// devuelve el trabajo original sin imprimir; con otro contenido, ID_REUTILIZADO.
func (s *Servidor) responderSiRepetido(w http.ResponseWriter, r *http.Request, cli *almacen.Cliente, sobre *contrato.Sobre) bool {
	previo, err := s.op.Almacen.IDsRecientes.Buscar(r.Context(), cli.ID, sobre.ID)
	if errors.Is(err, almacen.ErrNoEncontrado) || sobre.ID == "" {
		return false
	}
	if err != nil {
		s.responderError(w, r, err)
		return true
	}
	if s.liberarSiFallo(r.Context(), previo) {
		return false // el original no se imprimió: se procesa como nuevo
	}
	s.responderDuplicado(w, r, previo, sobre.Crudo)
	return true
}

// liberarSiFallo libera el id si su trabajo falló. Normalmente ya lo liberó finalizar; esto cubre
// el caso de que esa liberación haya fallado: un reintento nunca recibe «ok» por algo que no salió.
func (s *Servidor) liberarSiFallo(ctx context.Context, previo *almacen.IDReciente) bool {
	t, err := s.op.Almacen.Trabajos.Obtener(ctx, previo.TrabajoID)
	if err != nil || t.Estado != almacen.TrabajoFallido {
		return false
	}
	return s.op.Almacen.IDsRecientes.Liberar(ctx, previo.ClienteID, previo.DocumentoID, previo.TrabajoID) == nil
}

func (s *Servidor) responderDuplicado(w http.ResponseWriter, r *http.Request, previo *almacen.IDReciente, crudo []byte) {
	hash, err := seguridad.HashContenido(crudo)
	if err != nil {
		s.responderError(w, r, err)
		return
	}
	if !bytes.Equal(hash, previo.HashContenido) {
		s.responderError(w, r, contrato.NuevoError(contrato.ErrIDReutilizado,
			fmt.Sprintf("El id %q ya se usó en los últimos 10 minutos con otro contenido.", previo.DocumentoID),
			"Genera un id (UUID v4) nuevo para cada ticket y reutilízalo solo al reintentar ese mismo ticket.", "/id", "idempotencia"))
		return
	}
	t, err := s.op.Almacen.Trabajos.Obtener(r.Context(), previo.TrabajoID)
	if err != nil {
		s.responderError(w, r, err)
		return
	}
	resp := map[string]any{"ok": true, "duplicado": true, "trabajo_id": t.ID, "impresora": t.ImpresoraAlias,
		"estado": t.Estado, "advertencias": advertenciasDeCodigos(t.Advertencias)}
	if !t.Finalizado.IsZero() {
		resp["duracion_ms"] = t.Duracion.Milliseconds()
	}
	responderJSON(w, http.StatusOK, resp)
}

func advertenciasDeCodigos(codigos []string) []escpos.Advertencia {
	advs := []escpos.Advertencia{}
	for _, c := range codigos {
		advs = append(advs, escpos.Advertencia{Codigo: c})
	}
	return advs
}

// impresoraActiva obtiene la impresora del documento, o IMPRESORA_NO_ENCONTRADA.
func (s *Servidor) impresoraActiva(ctx context.Context, alias string) (*almacen.Impresora, error) {
	imp, err := s.op.Almacen.Impresoras.Obtener(ctx, alias)
	if errors.Is(err, almacen.ErrNoEncontrado) || (err == nil && !imp.Activa) {
		return nil, &conector.Error{Codigo: conector.ErrImpresoraNoEncontrada,
			Mensaje:    fmt.Sprintf("No hay ninguna impresora activa con el alias %q.", alias),
			Sugerencia: "Consulta GET /v1/impresoras para ver las impresoras disponibles."}
	}
	return imp, err
}

// NuevoTrabajoID genera un identificador de trabajo.
func NuevoTrabajoID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "trb_" + hex.EncodeToString(b)
}

// POST /v1/imprimir.
func (s *Servidor) imprimir(w http.ResponseWriter, r *http.Request) {
	cli, _, cuerpo, ok := s.autorizarDocumento(w, r, true)
	if !ok {
		return
	}
	ctx := r.Context()

	// Paso 6: validación completa, perfil, render y cola.
	doc, err := contrato.Decodificar(cuerpo)
	if err != nil {
		s.responderError(w, r, err)
		return
	}
	imp, err := s.impresoraActiva(ctx, doc.Impresora)
	if err != nil {
		s.responderError(w, r, err)
		return
	}
	datos, advs, err := escpos.Renderizar(*doc, *imp, escpos.RecursosAlmacen{Ctx: ctx, Repo: s.op.Almacen.Recursos})
	if err != nil {
		s.responderError(w, r, err)
		return
	}
	if err := s.AsegurarRegistrada(*imp); err != nil {
		s.responderError(w, r, err)
		return
	}

	hash, err := seguridad.HashContenido(cuerpo)
	if err != nil {
		s.responderError(w, r, err)
		return
	}
	trabajoID := NuevoTrabajoID()
	// La reserva es atómica: si otra petición con el mismo id ganó la carrera, se responde como repetido.
	reserva := almacen.IDReciente{ClienteID: cli.ID, DocumentoID: doc.ID, HashContenido: hash, TrabajoID: trabajoID}
	res, previo, err := s.op.Almacen.IDsRecientes.Reservar(ctx, reserva)
	if err == nil && res != almacen.ReservaNueva && s.liberarSiFallo(ctx, previo) {
		res, previo, err = s.op.Almacen.IDsRecientes.Reservar(ctx, reserva)
	}
	if err != nil {
		s.responderError(w, r, err)
		return
	}
	if res != almacen.ReservaNueva {
		s.responderDuplicado(w, r, previo, cuerpo)
		return
	}

	var codigos []string
	for _, a := range advs {
		codigos = append(codigos, a.Codigo)
	}
	t := &almacen.Trabajo{ID: trabajoID, DocumentoID: doc.ID, ClienteID: cli.ID, ImpresoraAlias: imp.Alias,
		Virtual: imp.Tipo == almacen.ConexionVirtual, Referencia: doc.Referencia, Advertencias: codigos}
	if err := s.op.Almacen.Trabajos.Crear(ctx, t); err != nil {
		s.op.Almacen.IDsRecientes.Liberar(context.WithoutCancel(ctx), cli.ID, doc.ID, trabajoID)
		s.responderError(w, r, err)
		return
	}
	canal, err := s.op.Cola.Encolar(imp.Alias, trabajoID, datos)
	if err != nil {
		s.finalizar(t, cola.Resultado{TrabajoID: trabajoID, Err: err}, codigos)
		s.responderErrorTrabajo(w, r, err, trabajoID)
		return
	}

	// El resultado se registra aunque el cliente se desconecte o se agote la espera.
	listo := make(chan cola.Resultado, 1)
	go func() {
		res := <-canal
		s.finalizar(t, res, codigos)
		listo <- res
	}()
	select {
	case res := <-listo:
		if !res.OK {
			s.responderErrorTrabajo(w, r, res.Err, trabajoID)
			return
		}
		if advs == nil {
			advs = []escpos.Advertencia{}
		}
		if res.Aviso != nil {
			advs = append(advs, escpos.Advertencia{Codigo: res.Aviso.Codigo, Mensaje: res.Aviso.Mensaje})
		}
		responderJSON(w, http.StatusOK, map[string]any{"ok": true, "trabajo_id": trabajoID, "impresora": imp.Alias,
			"duracion_ms": res.Duracion.Milliseconds(), "advertencias": advs})
	case <-time.After(s.op.TiempoRespuesta):
		responderJSON(w, http.StatusAccepted, map[string]any{"ok": true, "trabajo_id": trabajoID, "impresora": imp.Alias,
			"estado": almacen.TrabajoEncolado, "advertencias": advs})
	case <-ctx.Done():
	}
}

// finalizar guarda el resultado del trabajo. Si falló, libera el id del documento para que el
// cliente pueda reintentarlo sin que se tome como repetido.
func (s *Servidor) finalizar(t *almacen.Trabajo, res cola.Resultado, codigos []string) {
	ctx := context.Background()
	estado, codigo := almacen.TrabajoCompletado, ""
	if !res.OK {
		estado, codigo = almacen.TrabajoFallido, conector.Codigo(res.Err)
		if codigo == "" {
			codigo = "ERROR_INTERNO"
		}
		if err := s.op.Almacen.IDsRecientes.Liberar(ctx, t.ClienteID, t.DocumentoID, t.ID); err != nil {
			// El reintento igual se imprime: un duplicado de un trabajo fallido se procesa como nuevo.
			s.op.Log.Error("no se pudo liberar el id del trabajo fallido", "trabajo", t.ID, "error", err.Error())
		}
	}
	if res.Aviso != nil {
		codigos = append(slices.Clip(codigos), res.Aviso.Codigo)
	}
	if err := s.op.Almacen.Trabajos.Finalizar(ctx, t.ID, estado, codigo, codigos, res.Duracion); err != nil {
		s.op.Log.Error("no se pudo guardar el resultado del trabajo", "trabajo", t.ID, "error", err.Error())
	}
	s.op.Log.Info("trabajo", "trabajo", t.ID, "cliente", t.ClienteID, "impresora", t.ImpresoraAlias, "virtual", t.Virtual,
		"estado", estado, "codigo_error", codigo, "duracion_ms", res.Duracion.Milliseconds())
}

// POST /v1/previsualizar: misma autorización que imprimir (sin idempotencia); devuelve el HTML de
// la simulación sin imprimir nada.
func (s *Servidor) previsualizar(w http.ResponseWriter, r *http.Request) {
	_, _, cuerpo, ok := s.autorizarDocumento(w, r, false)
	if !ok {
		return
	}
	doc, err := contrato.Decodificar(cuerpo)
	if err != nil {
		s.responderError(w, r, err)
		return
	}
	imp, err := s.impresoraActiva(r.Context(), doc.Impresora)
	if err != nil {
		s.responderError(w, r, err)
		return
	}
	html, _, err := dev.Previsualizar(*doc, *imp, escpos.RecursosAlmacen{Ctx: r.Context(), Repo: s.op.Almacen.Recursos})
	if err != nil {
		s.responderError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// El fragmento solo usa estilos en línea e imágenes data:.
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src data:; style-src 'unsafe-inline'; frame-ancestors 'none'")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, html)
}
