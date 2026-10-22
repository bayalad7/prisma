package seguridad

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/contrato"
)

// Códigos de rechazo de la autorización.
const (
	ErrOrigenNoAutorizado     = "ORIGEN_NO_AUTORIZADO"
	ErrOrigenRequerido        = "ORIGEN_REQUERIDO"
	ErrCredencialesRequeridas = "CREDENCIALES_REQUERIDAS"
	ErrLlaveRequerida         = "LLAVE_REQUERIDA"
	ErrLlaveInvalida          = "LLAVE_INVALIDA"
	ErrClienteDesactivado     = "CLIENTE_DESACTIVADO"
	ErrImpresoraNoPermitida   = "IMPRESORA_NO_PERMITIDA"
	ErrOperacionNoPermitida   = "OPERACION_NO_PERMITIDA"
	ErrDemasiadasPeticiones   = "DEMASIADAS_PETICIONES"
)

// Rechazo es una petición no autorizada, con el estado HTTP que le corresponde.
type Rechazo struct {
	Estado     int
	Codigo     string
	Mensaje    string
	Sugerencia string
}

func (r *Rechazo) Error() string { return r.Codigo + ": " + r.Mensaje }

func rechazo(estado int, codigo, mensaje, sugerencia string) *Rechazo {
	return &Rechazo{Estado: estado, Codigo: codigo, Mensaje: mensaje, Sugerencia: sugerencia}
}

// Autorizador aplica, en orden, los pasos de autorización de la API pública.
type Autorizador struct {
	alm    *almacen.Almacen
	cache  *cacheLlaves
	turnos chan struct{} // cálculos argon2id simultáneos

	// Ahora y Tolerancia (diferencia de reloj aceptada en emitido/expira) se pueden cambiar en pruebas.
	Ahora      func() time.Time
	Tolerancia time.Duration
}

// NuevoAutorizador crea un autorizador sobre la base de datos del puente.
func NuevoAutorizador(alm *almacen.Almacen) *Autorizador {
	a := &Autorizador{alm: alm, turnos: make(chan struct{}, 2), Ahora: time.Now, Tolerancia: 30 * time.Second}
	a.cache = nuevaCacheLlaves(10000, 10*time.Minute, func() time.Time { return a.Ahora() })
	return a
}

// ClienteDeOrigen devuelve el cliente activo dueño del origen. Un origen desconocido se registra
// como solicitud pendiente para que el dueño del puente lo apruebe o lo rechace.
func (a *Autorizador) ClienteDeOrigen(ctx context.Context, origen string) (*almacen.Cliente, error) {
	noAutorizado := rechazo(http.StatusForbidden, ErrOrigenNoAutorizado,
		fmt.Sprintf("El origen %q no está autorizado para imprimir en este puente.", origen),
		"Pide al dueño del puente que apruebe este origen en el panel de administración (quedó como solicitud pendiente).")
	norm, err := almacen.NormalizarOrigen(origen)
	if err != nil {
		noAutorizado.Sugerencia = "Imprime desde una página servida por https:// o http:// con un origen aprobado."
		return nil, noAutorizado
	}
	cli, err := a.alm.Clientes.ObtenerPorOrigen(ctx, norm)
	if errors.Is(err, almacen.ErrNoEncontrado) {
		if _, err := a.alm.Solicitudes.Registrar(ctx, norm); err != nil && !errors.Is(err, almacen.ErrLimiteSolicitudes) {
			return nil, err
		}
		return nil, noAutorizado
	}
	if err != nil {
		return nil, err
	}
	if !cli.Activo {
		return nil, desactivado(cli)
	}
	return cli, nil
}

func desactivado(cli *almacen.Cliente) *Rechazo {
	return rechazo(http.StatusForbidden, ErrClienteDesactivado, fmt.Sprintf("La aplicación %q está desactivada en este puente.", cli.Nombre),
		"Pide al dueño del puente que la active en el panel de administración.")
}

// Identificar aplica los pasos 1 (origen) y 2 (llave de nivel 2). Devuelve nil, sin error, si la
// petición no trae ni origen ni llave: el cliente solo podrá salir de un documento firmado.
func (a *Autorizador) Identificar(ctx context.Context, origen, autorizacion string) (*almacen.Cliente, error) {
	var cli *almacen.Cliente
	if origen != "" {
		var err error
		if cli, err = a.ClienteDeOrigen(ctx, origen); err != nil {
			return nil, err
		}
	}

	llave, hayLlave, err := llaveDeEncabezado(autorizacion)
	if err != nil {
		return nil, err
	}
	if !hayLlave {
		if cli != nil && (cli.Nivel == almacen.NivelLlave || (cli.Nivel == almacen.NivelFirma && cli.ExigirLlave)) {
			return nil, rechazo(http.StatusUnauthorized, ErrLlaveRequerida, "Esta aplicación debe enviar su llave de cliente.",
				"Agrega el encabezado Authorization: Bearer pk_… con la llave que entregó el panel.")
		}
		return cli, nil
	}

	duenio, err := a.verificarLlave(ctx, llave)
	if err != nil {
		return nil, err
	}
	switch {
	case cli != nil && duenio.ID != cli.ID:
		return nil, llaveInvalida("La llave no corresponde a la aplicación de este origen.")
	case cli != nil:
		return cli, nil
	case !duenio.Activo:
		return nil, desactivado(duenio)
	case duenio.Nivel == almacen.NivelOrigen:
		return nil, rechazo(http.StatusForbidden, ErrOrigenRequerido,
			"Esta aplicación es de nivel 1: solo puede imprimir desde un navegador con un origen aprobado.",
			"Imprime desde la página web autorizada, o pide al dueño del puente que suba la aplicación a nivel 2.")
	}
	return duenio, nil
}

func llaveInvalida(mensaje string) *Rechazo {
	return rechazo(http.StatusUnauthorized, ErrLlaveInvalida, mensaje,
		"Revisa la llave de cliente o pide una nueva al dueño del puente.")
}

func llaveDeEncabezado(h string) (llave string, hay bool, err error) {
	if h == "" {
		return "", false, nil
	}
	esquema, valor, ok := strings.Cut(h, " ")
	if !ok || !strings.EqualFold(esquema, "Bearer") || strings.TrimSpace(valor) == "" {
		return "", false, llaveInvalida("El encabezado Authorization debe tener la forma «Bearer pk_…».")
	}
	return strings.TrimSpace(valor), true, nil
}

// verificarLlave busca al cliente por el prefijo público y compara con su hash argon2id. Una llave
// con formato inválido o prefijo desconocido se rechaza sin calcular nada; el resultado se guarda
// en caché y los cálculos simultáneos están limitados.
func (a *Autorizador) verificarLlave(ctx context.Context, llave string) (*almacen.Cliente, error) {
	prefijo, ok := PrefijoDeLlave(llave)
	if !ok {
		return nil, llaveInvalida("La llave de cliente no tiene un formato válido.")
	}
	cli, err := a.alm.Clientes.ObtenerPorPrefijo(ctx, prefijo)
	if errors.Is(err, almacen.ErrNoEncontrado) {
		return nil, llaveInvalida("La llave de cliente no es válida.")
	}
	if err != nil {
		return nil, err
	}
	if cli.LlaveHash == "" {
		return nil, llaveInvalida("La llave de cliente no es válida.")
	}
	valida, enCache := a.cache.buscar(llave, cli.LlaveHash)
	if !enCache {
		select {
		case a.turnos <- struct{}{}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		valida, err = VerificarContrasena(llave, cli.LlaveHash) // comparación en tiempo constante
		<-a.turnos
		if err != nil {
			return nil, err
		}
		a.cache.guardar(llave, cli.LlaveHash, valida)
	}
	if !valida {
		return nil, llaveInvalida("La llave de cliente no es válida.")
	}
	return cli, nil
}

func firmaInvalida(mensaje string) *Rechazo {
	return rechazo(http.StatusUnauthorized, contrato.ErrFirmaInvalida, mensaje,
		"Firma el documento sin el campo firma, en forma canónica JCS, con la llave privada registrada (ver la especificación, §7).")
}

// VerificarDocumento aplica el paso 3. Si la petición no identificó al cliente, lo identifica por
// el cliente_id del documento, que entonces debe venir firmado. Para el nivel 3 exige y verifica la
// firma y su vigencia; en todos los niveles respeta expira si viene.
func (a *Autorizador) VerificarDocumento(ctx context.Context, cli *almacen.Cliente, s *contrato.Sobre) (*almacen.Cliente, error) {
	if cli == nil {
		credenciales := rechazo(http.StatusUnauthorized, ErrCredencialesRequeridas,
			"La petición no identifica a ninguna aplicación autorizada.",
			"Desde un navegador, imprime desde un origen aprobado; desde un servidor, envía la llave (Bearer pk_…) o un documento firmado.")
		if s.ClienteID == "" || s.Firma == "" {
			return nil, credenciales
		}
		var err error
		cli, err = a.alm.Clientes.Obtener(ctx, s.ClienteID)
		if errors.Is(err, almacen.ErrNoEncontrado) {
			return nil, firmaInvalida("El cliente_id del documento no corresponde a ninguna aplicación registrada.")
		}
		if err != nil {
			return nil, err
		}
		if !cli.Activo {
			return nil, desactivado(cli)
		}
		if cli.Nivel != almacen.NivelFirma {
			return nil, credenciales
		}
		if cli.ExigirLlave {
			return nil, rechazo(http.StatusUnauthorized, ErrLlaveRequerida, "Esta aplicación debe enviar también su llave de cliente.",
				"Agrega el encabezado Authorization: Bearer pk_… además de firmar el documento.")
		}
	}

	ahora := a.Ahora()
	if cli.Nivel == almacen.NivelFirma {
		if s.Firma == "" {
			return nil, rechazo(http.StatusUnauthorized, contrato.ErrFirmaRequerida, "Esta aplicación debe enviar los documentos firmados.",
				"Agrega cliente_id, emitido, expira y firma (ver la especificación, §7).")
		}
		if s.ClienteID != cli.ID {
			return nil, firmaInvalida("El cliente_id del documento no corresponde a la aplicación que lo envía.")
		}
		if err := VerificarFirma(s.Crudo, s.Firma, cli.LlavePublica); err != nil {
			return nil, firmaInvalida("La firma no corresponde al documento: fue alterado o se firmó con otra llave.")
		}
		emitido, err1 := time.Parse(time.RFC3339Nano, s.Emitido)
		expira, err2 := time.Parse(time.RFC3339Nano, s.Expira)
		if err1 != nil || err2 != nil {
			return nil, firmaInvalida("emitido y expira deben ser fechas RFC 3339 con zona horaria.")
		}
		if vigencia := expira.Sub(emitido); vigencia <= 0 || vigencia > contrato.VigenciaMaxima {
			return nil, firmaInvalida(fmt.Sprintf("La vigencia de la firma es %s; debe ser mayor que 0 y de 120 s como máximo.", vigencia))
		}
		if emitido.After(ahora.Add(a.Tolerancia)) {
			return nil, firmaInvalida("El documento dice haber sido emitido en el futuro: revisa el reloj del servidor que firma.")
		}
	}
	if s.Expira != "" {
		if expira, err := time.Parse(time.RFC3339Nano, s.Expira); err == nil && ahora.After(expira.Add(a.Tolerancia)) {
			return nil, rechazo(http.StatusUnauthorized, contrato.ErrFirmaExpirada, "El documento ya expiró.",
				"Genera y firma el documento justo antes de enviarlo; revisa que el reloj del servidor esté en hora.")
		}
	}
	return cli, nil
}

// VerificarPermisos aplica el paso 5: la impresora y las operaciones sensibles del documento.
func (a *Autorizador) VerificarPermisos(ctx context.Context, cli *almacen.Cliente, s *contrato.Sobre) error {
	if s.Impresora != "" {
		ok, err := a.alm.Clientes.PuedeImprimirEn(ctx, cli.ID, s.Impresora)
		if err != nil {
			return err
		}
		if !ok {
			return rechazo(http.StatusForbidden, ErrImpresoraNoPermitida,
				fmt.Sprintf("La aplicación no tiene permiso para imprimir en %q.", s.Impresora),
				"Pide al dueño del puente que habilite esa impresora para la aplicación.")
		}
	}
	for _, tipo := range s.Tipos {
		if (tipo == contrato.TipoAbrirCajon || tipo == contrato.TipoRaw) && !cli.PermiteOperacion(tipo) {
			return rechazo(http.StatusForbidden, ErrOperacionNoPermitida,
				fmt.Sprintf("La operación %q está deshabilitada para esta aplicación.", tipo),
				"Las operaciones sensibles (abrir_cajon, raw) se habilitan por aplicación en el panel del puente.")
		}
	}
	return nil
}
