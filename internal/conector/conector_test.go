package conector

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/dev"
)

// servidorTCP acepta conexiones y entrega lo recibido en cada una por el canal.
func servidorTCP(t *testing.T) (string, <-chan []byte) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	recibido := make(chan []byte, 10)
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				datos, _ := io.ReadAll(c)
				recibido <- datos
			}()
		}
	}()
	return ln.Addr().String(), recibido
}

// puertoCerrado devuelve una dirección local en la que nadie escucha.
func puertoCerrado(t *testing.T) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	dir := ln.Addr().String()
	ln.Close()
	return dir
}

func exigirCodigo(t *testing.T, err error, codigo string) {
	t.Helper()
	if Codigo(err) != codigo {
		t.Fatalf("esperaba %s, obtuve %v", codigo, err)
	}
	var e *Error
	if e, _ = err.(*Error); e == nil || e.Mensaje == "" || e.Sugerencia == "" {
		t.Fatalf("el error debe tener mensaje y sugerencia: %#v", err)
	}
}

func TestRedEnviaTodoYCierra(t *testing.T) {
	dir, recibido := servidorTCP(t)
	datos := bytes.Repeat([]byte("ticket "), 10000)
	r := NuevoRed(dir)
	if err := r.Probar(context.Background()); err != nil {
		t.Fatal(err)
	}
	<-recibido // la conexión de Probar no envía nada
	if err := r.Enviar(context.Background(), datos); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-recibido:
		if !bytes.Equal(got, datos) {
			t.Fatalf("recibidos %d bytes, enviados %d", len(got), len(datos))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("la impresora no recibió los datos")
	}
}

func TestRedPuertoPorDefecto(t *testing.T) {
	if r := NuevoRed("192.168.1.50"); r.Direccion != "192.168.1.50:9100" {
		t.Fatalf("dirección %q", r.Direccion)
	}
}

func TestRedInalcanzable(t *testing.T) {
	r := NuevoRed(puertoCerrado(t))
	exigirCodigo(t, r.Enviar(context.Background(), []byte("x")), ErrImpresoraInalcanzable)
	exigirCodigo(t, r.Probar(context.Background()), ErrImpresoraInalcanzable)
}

func TestRedTiempoAgotadoAlEscribir(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	var conexiones []net.Conn
	go func() {
		for {
			c, err := ln.Accept() // acepta pero nunca lee: los búferes se llenan
			if err != nil {
				return
			}
			conexiones = append(conexiones, c)
		}
	}()
	r := NuevoRed(ln.Addr().String())
	r.TiempoEscritura = 300 * time.Millisecond
	err = r.Enviar(context.Background(), make([]byte, 64<<20))
	if err == nil {
		t.Skip("el sistema aceptó 64 MB en los búferes de loopback sin que nadie leyera")
	}
	exigirCodigo(t, err, ErrTiempoAgotado)
}

func TestRedContextoVencido(t *testing.T) {
	dir, _ := servidorTCP(t)
	ctx, fin := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer fin()
	exigirCodigo(t, NuevoRed(dir).Enviar(ctx, []byte("x")), ErrTiempoAgotado)
}

func TestArchivoGuardaBinarioYVista(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "salida")
	perfil := almacen.Impresora{Columnas: 32, ColumnasFuenteB: 42, AnchoPx: 384}
	a := NuevoArchivo(dir, perfil)
	a.ahora = func() time.Time { return time.Date(2026, 10, 15, 9, 30, 0, 0, time.UTC) }
	if err := a.Probar(context.Background()); err != nil {
		t.Fatal(err)
	}
	datos := []byte("\x1b@Hola\n")
	if err := a.Enviar(ConTrabajo(context.Background(), "trb_1"), datos); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(dir, "20261015-093000.000-trb_1")
	if got, _ := os.ReadFile(base + ".bin"); !bytes.Equal(got, datos) {
		t.Fatalf(".bin: %q", got)
	}
	vista, _ := os.ReadFile(base + ".txt")
	if !strings.Contains(string(vista), "│Hola") {
		t.Fatalf(".txt sin previsualización:\n%s", vista)
	}
	if err := a.Enviar(context.Background(), datos); err != nil {
		t.Fatal(err)
	}
	if bins, _ := filepath.Glob(filepath.Join(dir, "*.bin")); len(bins) != 2 {
		t.Fatalf("sin id de trabajo debe usar uno aleatorio: %v", bins)
	}
}

func TestSerieInexistente(t *testing.T) {
	puerto := "/dev/ttyPRISMANOEXISTE"
	if runtime.GOOS == "windows" {
		puerto = "COM250"
	}
	s := NuevoSerie(puerto, 9600)
	exigirCodigo(t, s.Probar(context.Background()), ErrImpresoraNoEncontrada)
	exigirCodigo(t, s.Enviar(context.Background(), []byte("x")), ErrImpresoraNoEncontrada)
}

func TestNuevoSegunTipo(t *testing.T) {
	casos := map[string]string{
		almacen.ConexionRed: "*conector.Red", almacen.ConexionBluetooth: "*conector.Serie", almacen.ConexionArchivo: "*conector.Archivo",
	}
	for tipo, want := range casos {
		c, err := Nuevo(almacen.Impresora{Alias: "x", Tipo: tipo, Destino: "d"})
		if err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%T", c); got != want {
			t.Errorf("%s: %s, quiero %s", tipo, got, want)
		}
	}
	if s, _ := Nuevo(almacen.Impresora{Tipo: almacen.ConexionBluetooth, Destino: "COM3"}); s.(*Serie).Baudios != BaudiosPorDefecto {
		t.Error("bluetooth sin baudios debe usar 9600")
	}
	if _, err := Nuevo(almacen.Impresora{Alias: "x", Tipo: almacen.ConexionRed}); Codigo(err) != ErrImpresoraNoEncontrada {
		t.Errorf("sin destino: %v", err)
	}
	if _, err := Nuevo(almacen.Impresora{Tipo: "wifi", Destino: "d"}); err == nil {
		t.Error("tipo desconocido debe fallar")
	}
}

func TestHostsYEscaneo(t *testing.T) {
	_, red24, _ := net.ParseCIDR("192.168.1.0/24")
	if n := len(hosts(red24)); n != 254 {
		t.Errorf("/24: %d direcciones", n)
	}
	_, red16, _ := net.ParseCIDR("10.0.0.0/16")
	if hosts(red16) != nil {
		t.Error("un rango mayor que /24 no se escanea completo")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()
	_, puertoTxt, _ := net.SplitHostPort(ln.Addr().String())
	puerto, _ := strconv.Atoi(puertoTxt)
	_, red, _ := net.ParseCIDR("127.0.0.0/30") // 127.0.0.1 y 127.0.0.2
	ds := EscanearRed(context.Background(), OpcionesDeteccion{Subredes: []*net.IPNet{red}, Puerto: puerto, TiempoHost: time.Second})
	if len(ds) != 1 || ds[0].Destino != ln.Addr().String() || ds[0].Tipo != "red" {
		t.Fatalf("escaneo: %+v", ds)
	}
}

// Una escritura que no se puede interrumpir (por ejemplo, WritePrinter con el spooler colgado)
// no bloquea para siempre: conPlazo la abandona poco después de cancelarla.
func TestConPlazoAbandonaLoQueNoSeDesbloquea(t *testing.T) {
	anterior := esperaTrasCancelar
	esperaTrasCancelar = 100 * time.Millisecond
	defer func() { esperaTrasCancelar = anterior }()
	colgada := make(chan struct{})
	defer close(colgada)
	cancelada := false
	inicio := time.Now()
	err := conPlazo(context.Background(), 50*time.Millisecond, func() error { <-colgada; return nil }, func() { cancelada = true })
	if !errors.Is(err, context.DeadlineExceeded) || !cancelada {
		t.Fatalf("err %v, cancelada %v", err, cancelada)
	}
	if d := time.Since(inicio); d > 2*time.Second {
		t.Fatalf("tardó %s", d)
	}
}

// impresoraConEstado lee hasta la consulta DLE EOT 1 y responde con estado (o no responde si
// estado es negativo). Entrega por el canal lo que recibió antes de la consulta.
func impresoraConEstado(t *testing.T, estado int) (string, <-chan []byte) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	recibido := make(chan []byte, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		var datos []byte
		buf := make([]byte, 4096)
		for !bytes.HasSuffix(datos, consultaEstado) {
			n, err := c.Read(buf)
			datos = append(datos, buf[:n]...)
			if err != nil {
				return
			}
		}
		recibido <- bytes.TrimSuffix(datos, consultaEstado)
		if estado >= 0 {
			c.Write([]byte{byte(estado)})
		}
		io.Copy(io.Discard, c)
	}()
	return ln.Addr().String(), recibido
}

func TestRedConfirmarImpresion(t *testing.T) {
	ticket := []byte("\x1b@Hola\n\x1dVA\x00")
	t.Run("en línea", func(t *testing.T) {
		dir, recibido := impresoraConEstado(t, 0x12)
		r := NuevoRed(dir)
		r.Confirmar = true
		if err := r.Enviar(context.Background(), ticket); err != nil {
			t.Fatal(err)
		}
		if got := <-recibido; !bytes.Equal(got, ticket) {
			t.Fatalf("recibido %q", got)
		}
	})
	t.Run("fuera de línea", func(t *testing.T) {
		dir, _ := impresoraConEstado(t, 0x1A) // bit 3: tapa abierta o sin papel
		r := NuevoRed(dir)
		r.Confirmar = true
		err := r.Enviar(context.Background(), ticket)
		var aviso *Aviso
		if !errors.As(err, &aviso) || aviso.Codigo != AvisoFueraDeLinea || Codigo(err) != AvisoFueraDeLinea {
			t.Fatalf("esperaba el aviso %s, obtuve %v", AvisoFueraDeLinea, err)
		}
	})
	t.Run("sin respuesta", func(t *testing.T) {
		dir, recibido := impresoraConEstado(t, -1)
		r := NuevoRed(dir)
		r.Confirmar, r.TiempoConfirmacion = true, 200*time.Millisecond
		inicio := time.Now()
		exigirCodigo(t, r.Enviar(context.Background(), ticket), ErrTiempoAgotado)
		if d := time.Since(inicio); d > 3*time.Second {
			t.Fatalf("tardó %v: no debe reintentar un envío que ya escribió", d)
		}
		<-recibido
	})
	t.Run("cierra sin responder", func(t *testing.T) {
		dir, recibido := servidorTCP(t) // lee hasta el final y cierra: una impresora sin DLE EOT
		r := NuevoRed(dir)
		r.Confirmar, r.TiempoConfirmacion = true, 200*time.Millisecond
		err := r.Enviar(context.Background(), ticket)
		if c := Codigo(err); c != ErrTiempoAgotado && c != ErrImpresoraInalcanzable {
			t.Fatalf("esperaba un fallo sin confirmación, obtuve %v", err)
		}
		<-recibido
	})
}

func TestInterpretarEstado(t *testing.T) {
	for b, quiero := range map[byte]string{0x12: "", 0x16: "", 0x1A: AvisoFueraDeLinea, 0x1E: AvisoFueraDeLinea, 0xFF: "", 0x00: ""} {
		if got := Codigo(interpretarEstado(b)); got != quiero {
			t.Errorf("0x%02X: %q, quiero %q", b, got, quiero)
		}
	}
}

func TestNuevoPasaConfirmar(t *testing.T) {
	c, _ := Nuevo(almacen.Impresora{Tipo: almacen.ConexionRed, Destino: "10.0.0.1", ConfirmarImpresion: true})
	if !c.(*Red).Confirmar {
		t.Error("red: no pasó ConfirmarImpresion")
	}
	c, _ = Nuevo(almacen.Impresora{Tipo: almacen.ConexionBluetooth, Destino: "COM3", ConfirmarImpresion: true})
	if !c.(*Serie).Confirmar {
		t.Error("bluetooth: no pasó ConfirmarImpresion")
	}
}

func TestConfirmarConImpresoraVirtual(t *testing.T) {
	perfil, _ := dev.PerfilPrueba(80)
	v := &dev.ImpresoraVirtual{Perfil: perfil, Directorio: t.TempDir(), Salida: io.Discard}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	terminado := make(chan error, 1)
	go func() { terminado <- v.Servir(ctx, ln) }()
	r := NuevoRed(ln.Addr().String())
	r.Confirmar, r.TiempoConfirmacion = true, 2*time.Second
	err = r.Enviar(context.Background(), []byte("\x1b@Hola\n"))
	// La impresora guarda el trabajo después de confirmar: hay que esperarla antes de que se
	// borre su carpeta temporal.
	limite := time.Now().Add(5 * time.Second)
	for err == nil && v.Recibidos() < 1 && time.Now().Before(limite) {
		time.Sleep(10 * time.Millisecond)
	}
	cancelar()
	if e := <-terminado; e != nil {
		t.Fatal(e)
	}
	if err != nil {
		t.Fatalf("la impresora virtual debe confirmar: %v", err)
	}
	if v.Recibidos() != 1 {
		t.Fatalf("la impresora virtual no guardó el trabajo: %d", v.Recibidos())
	}
}
