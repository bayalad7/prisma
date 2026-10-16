package servidor

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/cola"
	"github.com/bayalad7/prisma/internal/config"
	"github.com/bayalad7/prisma/internal/seguridad"
)

// clientePrueba es una aplicación registrada en el puente de prueba.
type clientePrueba struct {
	id, origen, llave string
}

type entorno struct {
	t         *testing.T
	alm       *almacen.Almacen
	srv       *Servidor
	http      *httptest.Server
	dirCaja   string
	dirCocina string
	privada   ed25519.PrivateKey
	niveles   map[int]clientePrueba
}

func nuevoEntorno(t *testing.T, limites config.Limites) *entorno {
	t.Helper()
	ctx := context.Background()
	alm, err := almacen.AbrirMemoria(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { alm.Cerrar() })
	e := &entorno{t: t, alm: alm, dirCaja: t.TempDir(), dirCocina: t.TempDir(), niveles: map[int]clientePrueba{}}

	impresora := func(alias, tipo, destino string) {
		imp := &almacen.Impresora{Alias: alias, Nombre: "Impresora " + alias, Tipo: tipo, Destino: destino, AnchoMM: 80,
			Columnas: 48, ColumnasFuenteB: 64, AnchoPx: 576, PaginaCodigos: "CP858", ComandoPaginaCodigos: 19,
			SoportaQRNativo: true, SoportaCorte: true, SoportaCajon: true, SoportaImagen: true, SoportaCodigoBarrasNativo: true, Activa: true}
		if err := alm.Impresoras.Crear(ctx, imp); err != nil {
			t.Fatal(err)
		}
	}
	impresora("caja", almacen.ConexionArchivo, e.dirCaja)
	impresora("cocina", almacen.ConexionArchivo, e.dirCocina)
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	impresora("apagada", almacen.ConexionRed, ln.Addr().String()) // nadie escucha
	ln.Close()

	publica, privada, _ := ed25519.GenerateKey(rand.Reader)
	e.privada = privada
	for nivel := 1; nivel <= 3; nivel++ {
		llave, prefijo, _ := seguridad.GenerarLlave()
		hash, _ := seguridad.HashLlave(llave)
		c := clientePrueba{id: fmt.Sprintf("cli_nivel%d00000", nivel), origen: fmt.Sprintf("https://n%d.example", nivel), llave: llave}
		cli := &almacen.Cliente{ID: c.id, Nombre: fmt.Sprintf("App nivel %d", nivel), Nivel: nivel, LlaveHash: hash,
			LlavePrefijo: prefijo, TodasLasImpresoras: true, Activo: true}
		if nivel == 3 {
			cli.LlavePublica = publica
		}
		if err := alm.Clientes.Crear(ctx, cli); err != nil {
			t.Fatal(err)
		}
		if err := alm.Clientes.AgregarOrigen(ctx, c.id, c.origen); err != nil {
			t.Fatal(err)
		}
		e.niveles[nivel] = c
	}

	q := cola.Nueva(cola.Opciones{IntervaloPrueba: time.Hour})
	t.Cleanup(q.Cerrar)
	e.srv = Nuevo(Opciones{Almacen: alm, Cola: q, Limites: limites, Version: "prueba", TiempoRespuesta: 10 * time.Second})
	e.http = httptest.NewServer(e.srv.Handler())
	t.Cleanup(e.http.Close)
	return e
}

func uuid4() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// documento devuelve un documento v1 para la impresora indicada con un id nuevo.
func documento(impresora string, ops ...map[string]any) map[string]any {
	if len(ops) == 0 {
		ops = []map[string]any{{"tipo": "texto", "valor": "Hola PRISMA"}, {"tipo": "cortar"}}
	}
	return map[string]any{"version": "1", "id": uuid4(), "impresora": impresora, "operaciones": ops}
}

func cuerpo(t *testing.T, doc map[string]any) []byte {
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// firmar firma el documento para el cliente; con emitido en el pasado se obtiene uno expirado.
func (e *entorno) firmar(doc map[string]any, clienteID string, emitido time.Time) []byte {
	firmado, err := seguridad.FirmarDocumento(cuerpo(e.t, doc), e.privada, seguridad.OpcionesFirma{ClienteID: clienteID, Emitido: emitido})
	if err != nil {
		e.t.Fatal(err)
	}
	return firmado
}

type respuesta struct {
	estado int
	cuerpo map[string]any
	crudo  string
	cab    http.Header
}

func (r respuesta) codigo() string {
	c, _ := r.cuerpo["codigo"].(string)
	return c
}

func (e *entorno) pedir(metodo, ruta string, cuerpo []byte, cabeceras map[string]string) respuesta {
	e.t.Helper()
	req, _ := http.NewRequest(metodo, e.http.URL+ruta, bytes.NewReader(cuerpo))
	for k, v := range cabeceras {
		if v != "" {
			req.Header.Set(k, v)
		}
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	crudo, _ := io.ReadAll(res.Body)
	r := respuesta{estado: res.StatusCode, crudo: string(crudo), cab: res.Header}
	json.Unmarshal(crudo, &r.cuerpo)
	return r
}

func (e *entorno) imprimir(cuerpo []byte, origen, llave string) respuesta {
	e.t.Helper()
	cab := map[string]string{"Content-Type": "application/json", "Origin": origen}
	if llave != "" {
		cab["Authorization"] = "Bearer " + llave
	}
	return e.pedir(http.MethodPost, "/v1/imprimir", cuerpo, cab)
}

func contarImpresos(t *testing.T, dir string) int {
	t.Helper()
	bins, _ := filepath.Glob(filepath.Join(dir, "*.bin"))
	return len(bins)
}

// esperado es el oráculo de la matriz: el resultado que exige la especificación para cada caso.
func esperado(nivel int, origen, llave, firma string) (estado int, codigo string) {
	switch {
	case origen == "invalido":
		return 403, "ORIGEN_NO_AUTORIZADO"
	case llave == "invalida":
		return 401, "LLAVE_INVALIDA"
	case origen == "ausente" && llave == "valida" && nivel == 1:
		return 403, "ORIGEN_REQUERIDO"
	case origen == "ausente" && llave == "ausente" && nivel != 3:
		return 401, "CREDENCIALES_REQUERIDAS"
	case origen == "valido" && llave == "ausente" && nivel == 2:
		return 401, "LLAVE_REQUERIDA"
	case firma == "alterada" && nivel == 3:
		return 401, "FIRMA_INVALIDA"
	case firma == "expirada":
		return 401, "FIRMA_EXPIRADA"
	}
	return 200, ""
}

// TestMatrizDeAutorizacion recorre {nivel 1, 2, 3} × {origen válido, inválido, ausente} ×
// {llave válida, inválida, ausente} × {firma válida, alterada, expirada, repetida}.
func TestMatrizDeAutorizacion(t *testing.T) {
	e := nuevoEntorno(t, config.Limites{})
	impresos := 0
	for nivel := 1; nivel <= 3; nivel++ {
		c := e.niveles[nivel]
		for _, origen := range []string{"valido", "invalido", "ausente"} {
			for _, llave := range []string{"valida", "invalida", "ausente"} {
				for _, firma := range []string{"valida", "alterada", "expirada", "repetida"} {
					nombre := fmt.Sprintf("nivel%d/origen_%s/llave_%s/firma_%s", nivel, origen, llave, firma)
					t.Run(nombre, func(t *testing.T) {
						cabOrigen := map[string]string{"valido": c.origen, "invalido": "https://malicioso.example", "ausente": ""}[origen]
						cabLlave := map[string]string{"valida": c.llave, "invalida": c.llave[:len(c.llave)-4] + "AAAA", "ausente": ""}[llave]
						emitido := time.Now()
						if firma == "expirada" {
							emitido = time.Now().Add(-200 * time.Second)
						}
						datos := e.firmar(documento("caja"), c.id, emitido)
						if firma == "alterada" {
							var doc map[string]any
							json.Unmarshal(datos, &doc)
							doc["operaciones"].([]any)[0].(map[string]any)["valor"] = "Total alterado: $1"
							datos = cuerpo(t, doc)
						}

						estado, codigo := esperado(nivel, origen, llave, firma)
						r := e.imprimir(datos, cabOrigen, cabLlave)
						if r.estado != estado || r.codigo() != codigo {
							t.Fatalf("respuesta %d %s, quiero %d %s\n%s", r.estado, r.codigo(), estado, codigo, r.crudo)
						}
						if estado == 200 {
							impresos++
							if r.cuerpo["ok"] != true || r.cuerpo["trabajo_id"] == "" || r.cuerpo["duracion_ms"] == nil {
								t.Fatalf("respuesta exitosa incompleta: %s", r.crudo)
							}
						} else if r.cuerpo["ok"] != false || r.cuerpo["mensaje"] == "" || r.cuerpo["sugerencia"] == "" {
							t.Fatalf("error sin el formato acordado: %s", r.crudo)
						}
						if firma == "repetida" {
							// El mismo documento otra vez: nunca se imprime dos veces.
							r2 := e.imprimir(datos, cabOrigen, cabLlave)
							if estado == 200 {
								if r2.estado != 200 || r2.cuerpo["duplicado"] != true || r2.cuerpo["trabajo_id"] != r.cuerpo["trabajo_id"] {
									t.Fatalf("repetido: %d %s", r2.estado, r2.crudo)
								}
							} else if r2.estado != estado || r2.codigo() != codigo {
								t.Fatalf("repetido: %d %s, quiero %d %s", r2.estado, r2.codigo(), estado, codigo)
							}
						}
						// CORS: eco exacto solo para el origen autorizado y si la petición pasó la identificación.
						if allow := r.cab.Get("Access-Control-Allow-Origin"); allow == "*" || (allow != "" && allow != c.origen) ||
							(origen == "invalido" && allow != "") {
							t.Fatalf("Access-Control-Allow-Origin = %q", allow)
						}
					})
				}
			}
		}
	}
	if got := contarImpresos(t, e.dirCaja); got != impresos {
		t.Fatalf("se imprimieron %d tickets, se esperaban %d", got, impresos)
	}
	// Los orígenes desconocidos quedaron como solicitud pendiente.
	if s, err := e.alm.Solicitudes.Obtener(context.Background(), "https://malicioso.example"); err != nil || s.Estado != almacen.SolicitudPendiente {
		t.Fatalf("solicitud: %+v %v", s, err)
	}
}

func TestNivel3PuedeExigirLlave(t *testing.T) {
	e := nuevoEntorno(t, config.Limites{})
	ctx := context.Background()
	cli, _ := e.alm.Clientes.Obtener(ctx, e.niveles[3].id)
	cli.ExigirLlave = true
	if err := e.alm.Clientes.Actualizar(ctx, cli); err != nil {
		t.Fatal(err)
	}
	datos := e.firmar(documento("caja"), cli.ID, time.Now())
	if r := e.imprimir(datos, "", ""); r.codigo() != "LLAVE_REQUERIDA" {
		t.Fatalf("sin llave: %d %s", r.estado, r.crudo)
	}
	if r := e.imprimir(datos, e.niveles[3].origen, ""); r.codigo() != "LLAVE_REQUERIDA" {
		t.Fatalf("con origen y sin llave: %d %s", r.estado, r.crudo)
	}
	if r := e.imprimir(datos, "", e.niveles[3].llave); r.estado != 200 {
		t.Fatalf("con llave y firma: %d %s", r.estado, r.crudo)
	}
}

func TestNivel3SinFirma(t *testing.T) {
	e := nuevoEntorno(t, config.Limites{})
	r := e.imprimir(cuerpo(t, documento("caja")), e.niveles[3].origen, "")
	if r.estado != 401 || r.codigo() != "FIRMA_REQUERIDA" {
		t.Fatalf("%d %s", r.estado, r.crudo)
	}
}

func TestPermisos(t *testing.T) {
	e := nuevoEntorno(t, config.Limites{})
	ctx := context.Background()
	c := e.niveles[2]
	if err := e.alm.Clientes.EstablecerImpresoras(ctx, c.id, false, []string{"caja"}); err != nil {
		t.Fatal(err)
	}
	enviar := func(doc map[string]any) respuesta {
		return e.imprimir(cuerpo(t, doc), c.origen, c.llave)
	}
	if r := enviar(documento("cocina")); r.estado != 403 || r.codigo() != "IMPRESORA_NO_PERMITIDA" {
		t.Fatalf("impresora no permitida: %d %s", r.estado, r.crudo)
	}
	cajon := []map[string]any{{"tipo": "texto", "valor": "Venta"}, {"tipo": "abrir_cajon"}}
	if r := enviar(documento("caja", cajon...)); r.estado != 403 || r.codigo() != "OPERACION_NO_PERMITIDA" {
		t.Fatalf("abrir_cajon sin permiso: %d %s", r.estado, r.crudo)
	}
	raw := []map[string]any{{"tipo": "raw", "base64": "G0A="}}
	if r := enviar(documento("caja", raw...)); r.codigo() != "OPERACION_NO_PERMITIDA" {
		t.Fatalf("raw sin permiso: %d %s", r.estado, r.crudo)
	}
	if contarImpresos(t, e.dirCaja)+contarImpresos(t, e.dirCocina) != 0 {
		t.Fatal("nada debía imprimirse")
	}

	cli, _ := e.alm.Clientes.Obtener(ctx, c.id)
	cli.OperacionesSensibles = []string{"abrir_cajon"}
	e.alm.Clientes.Actualizar(ctx, cli)
	if r := enviar(documento("caja", cajon...)); r.estado != 200 {
		t.Fatalf("abrir_cajon con permiso: %d %s", r.estado, r.crudo)
	}
	if r := enviar(documento("caja", raw...)); r.codigo() != "OPERACION_NO_PERMITIDA" {
		t.Fatalf("raw sigue sin permiso: %d %s", r.estado, r.crudo)
	}

	// GET /v1/impresoras solo muestra las permitidas.
	r := e.pedir(http.MethodGet, "/v1/impresoras", nil, map[string]string{"Origin": c.origen, "Authorization": "Bearer " + c.llave})
	var lista []map[string]any
	json.Unmarshal([]byte(r.crudo), &lista)
	if r.estado != 200 || len(lista) != 1 || lista[0]["alias"] != "caja" || lista[0]["estado"] == nil {
		t.Fatalf("impresoras: %d %s", r.estado, r.crudo)
	}
}

func TestImpresoraInexistenteYDocumentoInvalido(t *testing.T) {
	e := nuevoEntorno(t, config.Limites{})
	c := e.niveles[2]
	if r := e.imprimir(cuerpo(t, documento("bodega")), c.origen, c.llave); r.estado != 404 || r.codigo() != "IMPRESORA_NO_ENCONTRADA" {
		t.Fatalf("%d %s", r.estado, r.crudo)
	}
	doc := documento("caja")
	doc["copias"] = 9
	r := e.imprimir(cuerpo(t, doc), c.origen, c.llave)
	if r.estado != 400 || r.codigo() != "DOCUMENTO_INVALIDO" || r.cuerpo["detalles"] == nil {
		t.Fatalf("%d %s", r.estado, r.crudo)
	}
	doc = documento("caja")
	doc["version"] = "2"
	if r := e.imprimir(cuerpo(t, doc), c.origen, c.llave); r.codigo() != "VERSION_NO_SOPORTADA" || r.cuerpo["versiones_soportadas"] == nil {
		t.Fatalf("%d %s", r.estado, r.crudo)
	}
}

func TestIdempotencia(t *testing.T) {
	e := nuevoEntorno(t, config.Limites{})
	c := e.niveles[2]
	doc := documento("caja")
	primero := e.imprimir(cuerpo(t, doc), c.origen, c.llave)
	if primero.estado != 200 {
		t.Fatalf("%d %s", primero.estado, primero.crudo)
	}
	// Mismo id con otro contenido: ID_REUTILIZADO.
	doc["operaciones"] = []map[string]any{{"tipo": "texto", "valor": "Otro ticket"}}
	if r := e.imprimir(cuerpo(t, doc), c.origen, c.llave); r.estado != 409 || r.codigo() != "ID_REUTILIZADO" {
		t.Fatalf("%d %s", r.estado, r.crudo)
	}
	// Otro cliente puede usar el mismo id.
	n1 := e.niveles[1]
	if r := e.imprimir(cuerpo(t, doc), n1.origen, ""); r.estado != 200 {
		t.Fatalf("otro cliente: %d %s", r.estado, r.crudo)
	}
}

func TestTrabajoFallidoLiberaElID(t *testing.T) {
	e := nuevoEntorno(t, config.Limites{})
	c := e.niveles[2]
	datos := cuerpo(t, documento("apagada"))
	r := e.imprimir(datos, c.origen, c.llave)
	if r.estado != 503 || r.codigo() != "IMPRESORA_INALCANZABLE" || r.cuerpo["trabajo_id"] == nil {
		t.Fatalf("%d %s", r.estado, r.crudo)
	}
	// Reintentar el mismo documento vuelve a intentar imprimir (no es un repetido).
	r2 := e.imprimir(datos, c.origen, c.llave)
	if r2.codigo() != "IMPRESORA_INALCANZABLE" || r2.cuerpo["trabajo_id"] == r.cuerpo["trabajo_id"] {
		t.Fatalf("reintento: %d %s", r2.estado, r2.crudo)
	}
	t1, _ := e.alm.Trabajos.Obtener(context.Background(), r.cuerpo["trabajo_id"].(string))
	if t1.Estado != almacen.TrabajoFallido || t1.CodigoError != "IMPRESORA_INALCANZABLE" {
		t.Fatalf("trabajo: %+v", t1)
	}
}

func TestPreflight(t *testing.T) {
	e := nuevoEntorno(t, config.Limites{})
	c := e.niveles[2]
	r := e.pedir(http.MethodOptions, "/v1/imprimir", nil, map[string]string{
		"Origin": c.origen, "Access-Control-Request-Method": "POST",
		"Access-Control-Request-Headers": "content-type, authorization", "Access-Control-Request-Private-Network": "true"})
	cab := r.cab
	if r.estado != 204 || cab.Get("Access-Control-Allow-Origin") != c.origen || cab.Get("Access-Control-Allow-Private-Network") != "true" ||
		!strings.Contains(cab.Get("Access-Control-Allow-Methods"), "POST") || !strings.Contains(cab.Get("Access-Control-Allow-Headers"), "Authorization") ||
		!strings.Contains(strings.Join(cab.Values("Vary"), ","), "Origin") {
		t.Fatalf("preflight autorizado: %d %v", r.estado, cab)
	}
	// Sin la cabecera de red privada no se agrega Allow-Private-Network.
	r = e.pedir(http.MethodOptions, "/v1/imprimir", nil, map[string]string{"Origin": c.origen, "Access-Control-Request-Method": "POST"})
	if r.cab.Get("Access-Control-Allow-Private-Network") != "" {
		t.Fatal("Allow-Private-Network solo cuando se solicita")
	}
	// Origen desconocido: sin cabeceras CORS y queda como solicitud pendiente.
	r = e.pedir(http.MethodOptions, "/v1/imprimir", nil, map[string]string{"Origin": "https://nueva.example",
		"Access-Control-Request-Method": "POST", "Access-Control-Request-Private-Network": "true"})
	if r.estado != 403 || r.cab.Get("Access-Control-Allow-Origin") != "" || r.cab.Get("Access-Control-Allow-Private-Network") != "" {
		t.Fatalf("preflight desconocido: %d %v", r.estado, r.cab)
	}
	if _, err := e.alm.Solicitudes.Obtener(context.Background(), "https://nueva.example"); err != nil {
		t.Fatal("el preflight debía registrar la solicitud")
	}
}

func TestLimitePorIPYPorCliente(t *testing.T) {
	e := nuevoEntorno(t, config.Limites{PorIP: 3})
	for i := range 3 {
		if r := e.pedir(http.MethodGet, "/v1/salud", nil, nil); r.estado != 200 {
			t.Fatalf("petición %d: %d", i, r.estado)
		}
	}
	r := e.pedir(http.MethodGet, "/v1/salud", nil, nil)
	if r.estado != 429 || r.codigo() != "DEMASIADAS_PETICIONES" || r.cab.Get("Retry-After") == "" {
		t.Fatalf("límite por IP: %d %s %v", r.estado, r.crudo, r.cab)
	}

	e = nuevoEntorno(t, config.Limites{PorCliente: 2})
	c := e.niveles[2]
	cab := map[string]string{"Origin": c.origen, "Authorization": "Bearer " + c.llave}
	e.pedir(http.MethodGet, "/v1/impresoras", nil, cab)
	e.pedir(http.MethodGet, "/v1/impresoras", nil, cab)
	if r := e.pedir(http.MethodGet, "/v1/impresoras", nil, cab); r.estado != 429 {
		t.Fatalf("límite por cliente: %d %s", r.estado, r.crudo)
	}
	// Otro cliente no se ve afectado.
	n1 := e.niveles[1]
	if r := e.pedir(http.MethodGet, "/v1/impresoras", nil, map[string]string{"Origin": n1.origen}); r.estado != 200 {
		t.Fatalf("otro cliente: %d %s", r.estado, r.crudo)
	}
}

func TestSaludYCabecerasDeSeguridad(t *testing.T) {
	e := nuevoEntorno(t, config.Limites{})
	e.alm.Ajustes.Establecer(context.Background(), AjusteNombrePuente, "Café Prisma")
	r := e.pedir(http.MethodGet, "/v1/salud", nil, nil)
	if r.estado != 200 || r.cuerpo["ok"] != true || r.cuerpo["version"] != "prueba" || r.cuerpo["nombre_puente"] != "Café Prisma" {
		t.Fatalf("salud: %d %s", r.estado, r.crudo)
	}
	if len(r.cuerpo) != 3 {
		t.Fatalf("salud no debe exponer más datos: %s", r.crudo)
	}
	for k, v := range map[string]string{"X-Content-Type-Options": "nosniff", "X-Frame-Options": "DENY", "Cache-Control": "no-store"} {
		if r.cab.Get(k) != v {
			t.Errorf("%s = %q", k, r.cab.Get(k))
		}
	}
	if !strings.Contains(r.cab.Get("Content-Security-Policy"), "default-src 'none'") {
		t.Errorf("CSP: %q", r.cab.Get("Content-Security-Policy"))
	}
	// Origen autorizado recibe CORS en salud; uno desconocido no.
	if r := e.pedir(http.MethodGet, "/v1/salud", nil, map[string]string{"Origin": e.niveles[1].origen}); r.cab.Get("Access-Control-Allow-Origin") != e.niveles[1].origen {
		t.Error("salud debe permitir CORS a un origen autorizado")
	}
	if r := e.pedir(http.MethodGet, "/v1/salud", nil, map[string]string{"Origin": "https://otra.example"}); r.cab.Get("Access-Control-Allow-Origin") != "" {
		t.Error("salud no debe permitir CORS a un origen desconocido")
	}
}

func TestRutasMetodosYCuerpo(t *testing.T) {
	e := nuevoEntorno(t, config.Limites{})
	if r := e.pedir(http.MethodGet, "/v1/nada", nil, nil); r.estado != 404 || r.codigo() != "RUTA_NO_ENCONTRADA" {
		t.Fatalf("%d %s", r.estado, r.crudo)
	}
	if r := e.pedir(http.MethodGet, "/v1/imprimir", nil, nil); r.estado != 405 || r.codigo() != "METODO_NO_PERMITIDO" || r.cab.Get("Allow") == "" {
		t.Fatalf("%d %s", r.estado, r.crudo)
	}
	c := e.niveles[2]
	grande := bytes.Repeat([]byte("x"), MaxBytesCuerpo+1)
	if r := e.imprimir(grande, c.origen, c.llave); r.estado != 413 || r.codigo() != "DOCUMENTO_DEMASIADO_GRANDE" {
		t.Fatalf("cuerpo grande: %d %s", r.estado, r.crudo)
	}
	if r := e.imprimir([]byte("{no es json"), c.origen, c.llave); r.estado != 400 || r.codigo() != "JSON_INVALIDO" {
		t.Fatalf("JSON inválido: %d %s", r.estado, r.crudo)
	}
	// Sin credenciales, GET /v1/impresoras no revela nada.
	if r := e.pedir(http.MethodGet, "/v1/impresoras", nil, nil); r.estado != 401 || r.codigo() != "CREDENCIALES_REQUERIDAS" {
		t.Fatalf("%d %s", r.estado, r.crudo)
	}
}

func TestPrevisualizar(t *testing.T) {
	e := nuevoEntorno(t, config.Limites{})
	c := e.niveles[2]
	cab := map[string]string{"Origin": c.origen, "Authorization": "Bearer " + c.llave}
	doc := documento("caja", map[string]any{"tipo": "texto", "valor": "<b>Hola</b>", "alineacion": "centro"})
	r := e.pedir(http.MethodPost, "/v1/previsualizar", cuerpo(t, doc), cab)
	if r.estado != 200 || !strings.HasPrefix(r.cab.Get("Content-Type"), "text/html") || !strings.Contains(r.crudo, "prisma-ticket") ||
		strings.Contains(r.crudo, "<b>") || !strings.Contains(r.cab.Get("Content-Security-Policy"), "img-src data:") {
		t.Fatalf("previsualizar: %d %v\n%s", r.estado, r.cab, r.crudo)
	}
	if contarImpresos(t, e.dirCaja) != 0 {
		t.Fatal("previsualizar no debe imprimir")
	}
	// Misma autorización que imprimir.
	if r := e.pedir(http.MethodPost, "/v1/previsualizar", cuerpo(t, doc), map[string]string{"Origin": c.origen}); r.codigo() != "LLAVE_REQUERIDA" {
		t.Fatalf("sin llave: %d %s", r.estado, r.crudo)
	}
}

func TestClienteDesactivado(t *testing.T) {
	e := nuevoEntorno(t, config.Limites{})
	ctx := context.Background()
	cli, _ := e.alm.Clientes.Obtener(ctx, e.niveles[2].id)
	cli.Activo = false
	e.alm.Clientes.Actualizar(ctx, cli)
	if r := e.imprimir(cuerpo(t, documento("caja")), e.niveles[2].origen, e.niveles[2].llave); r.estado != 403 || r.codigo() != "CLIENTE_DESACTIVADO" {
		t.Fatalf("%d %s", r.estado, r.crudo)
	}
}

func TestRegistrarImpresorasYPerfilActualizado(t *testing.T) {
	e := nuevoEntorno(t, config.Limites{})
	ctx := context.Background()
	if err := e.srv.RegistrarImpresoras(ctx); err != nil {
		t.Fatal(err)
	}
	// Cambiar el destino con el puente en marcha: el siguiente trabajo usa el nuevo conector.
	imp, _ := e.alm.Impresoras.Obtener(ctx, "caja")
	nuevoDir := t.TempDir()
	imp.Destino = nuevoDir
	time.Sleep(2 * time.Millisecond)
	if err := e.alm.Impresoras.Actualizar(ctx, imp); err != nil {
		t.Fatal(err)
	}
	c := e.niveles[2]
	if r := e.imprimir(cuerpo(t, documento("caja")), c.origen, c.llave); r.estado != 200 {
		t.Fatalf("%d %s", r.estado, r.crudo)
	}
	if contarImpresos(t, nuevoDir) != 1 {
		t.Fatal("el trabajo debía ir al destino nuevo")
	}
}

func TestErrorInternoNoFiltraDetalles(t *testing.T) {
	e := nuevoEntorno(t, config.Limites{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/salud", nil)
	e.srv.responderError(rec, req, fmt.Errorf("sql: base dañada en C:\\secreto\\prisma.db"))
	if rec.Code != 500 || strings.Contains(rec.Body.String(), "secreto") || !strings.Contains(rec.Body.String(), "ERROR_INTERNO") {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}

func TestAutofirmadoYArchivos(t *testing.T) {
	dir := t.TempDir()
	prov, err := CargarAutofirmado(dir)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := LeerCertificado(prov.Certificado)
	if err != nil {
		t.Fatal(err)
	}
	tieneLocalhost := false
	for _, n := range cert.DNSNames {
		tieneLocalhost = tieneLocalhost || n == "localhost"
	}
	if !tieneLocalhost || len(IPsSinCubrir(cert)) != 0 || cert.NotAfter.Sub(cert.NotBefore) > 826*24*time.Hour {
		t.Fatalf("certificado: dns %v ips %v vence %v", cert.DNSNames, cert.IPAddresses, cert.NotAfter)
	}
	if info, _ := os.Stat(filepath.Join(dir, "tls", "llave.pem")); info == nil {
		t.Fatal("falta la llave")
	}
	cfg := ConfigTLS(prov)
	if cfg.MinVersion != 0x0303 {
		t.Fatal("TLS 1.2 mínimo")
	}
	primero, _ := prov.ObtenerCertificado(nil)

	// Al regenerar, el proveedor recarga los archivos sin reiniciar.
	prov.revisado = time.Time{}
	time.Sleep(20 * time.Millisecond)
	if _, err := GenerarAutofirmado(dir); err != nil {
		t.Fatal(err)
	}
	os.Chtimes(prov.Certificado, time.Now().Add(time.Second), time.Now().Add(time.Second))
	segundo, _ := prov.ObtenerCertificado(nil)
	if bytes.Equal(primero.Certificate[0], segundo.Certificate[0]) {
		t.Fatal("no recargó el certificado regenerado")
	}

	// Archivos inválidos: error claro.
	os.WriteFile(filepath.Join(dir, "malo.pem"), []byte("x"), 0o600)
	if _, err := NuevoArchivos(filepath.Join(dir, "malo.pem"), filepath.Join(dir, "malo.pem")); err == nil {
		t.Fatal("esperaba error con archivos inválidos")
	}
}

func TestEscucharHTTPSYLocalhost(t *testing.T) {
	e := nuevoEntorno(t, config.Limites{})
	for _, modo := range []string{config.TLSAutofirmado, config.TLSSoloLocalhostHTTP} {
		t.Run(modo, func(t *testing.T) {
			cfg := config.PorDefecto()
			cfg.Servidor.Escuchar, cfg.Servidor.Puerto, cfg.Servidor.ModoTLS = "127.0.0.1", puertoLibre(t), modo
			cfg.Datos.Ruta = t.TempDir()
			ctx, cancelar := context.WithCancel(context.Background())
			dir := make(chan string, 1)
			fin := make(chan error, 1)
			go func() { fin <- e.srv.Escuchar(ctx, cfg, func(d string) { dir <- d }) }()
			d := <-dir
			cliente := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{TLSClientConfig: insecuro()}}
			esquema := "https"
			if modo == config.TLSSoloLocalhostHTTP {
				esquema = "http"
			}
			res, err := cliente.Get(esquema + "://" + d + "/v1/salud")
			if err != nil {
				t.Fatal(err)
			}
			res.Body.Close()
			if res.StatusCode != 200 || (esquema == "https" && res.TLS == nil) {
				t.Fatalf("estado %d, tls %v", res.StatusCode, res.TLS != nil)
			}
			cancelar()
			if err := <-fin; err != nil {
				t.Fatal(err)
			}
		})
	}
}

func puertoLibre(t *testing.T) int {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}
