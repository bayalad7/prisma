package conector

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	"go.bug.st/serial"
)

// Detectada es una impresora o puerto encontrado en este equipo o en la red local.
type Detectada struct {
	Tipo    string `json:"tipo"`    // red | usb | bluetooth
	Destino string `json:"destino"` // lo que va en impresoras.destino
	Detalle string `json:"detalle,omitempty"`
}

// OpcionesDeteccion controla qué se busca. El escaneo de red solo se hace si se pide.
type OpcionesDeteccion struct {
	Red          bool
	Subredes     []*net.IPNet  // vacío = subredes privadas de este equipo
	Puerto       int           // 9100 por defecto
	Concurrencia int           // conexiones simultáneas, 64 por defecto
	TiempoHost   time.Duration // por dirección, 400 ms por defecto
	TiempoTotal  time.Duration // del escaneo completo, 15 s por defecto
}

// Detectar lista impresoras instaladas en el sistema, puertos serie y, si se pide, equipos de
// la red local que aceptan conexiones en el puerto 9100.
func Detectar(ctx context.Context, op OpcionesDeteccion) []Detectada {
	ds := impresorasSistema(ctx)
	if puertos, err := serial.GetPortsList(); err == nil {
		for _, p := range puertos {
			ds = append(ds, Detectada{Tipo: "bluetooth", Destino: p, Detalle: "puerto serie (Bluetooth SPP o serial)"})
		}
	}
	if op.Red {
		ds = append(ds, EscanearRed(ctx, op)...)
	}
	return ds
}

// SubredesLocales devuelve las subredes IPv4 privadas de las interfaces activas. Una subred mayor
// que /24 se reduce al /24 de este equipo, para que el escaneo sea acotado.
func SubredesLocales() []*net.IPNet {
	var redes []*net.IPNet
	vistas := map[string]bool{}
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		dirs, _ := iface.Addrs()
		for _, d := range dirs {
			ipn, ok := d.(*net.IPNet)
			if !ok || ipn.IP.To4() == nil || !ipn.IP.IsPrivate() {
				continue
			}
			unos, _ := ipn.Mask.Size()
			if unos < 24 {
				ipn = &net.IPNet{IP: ipn.IP.To4().Mask(net.CIDRMask(24, 32)), Mask: net.CIDRMask(24, 32)}
			} else {
				ipn = &net.IPNet{IP: ipn.IP.To4().Mask(ipn.Mask), Mask: ipn.Mask}
			}
			if !vistas[ipn.String()] {
				vistas[ipn.String()] = true
				redes = append(redes, ipn)
			}
		}
	}
	return redes
}

// hosts enumera las direcciones utilizables de una subred IPv4 (máximo un /24).
func hosts(red *net.IPNet) []net.IP {
	unos, bits := red.Mask.Size()
	if bits != 32 || unos < 24 {
		return nil
	}
	base := binary.BigEndian.Uint32(red.IP.To4())
	total := uint32(1) << (32 - unos)
	var ips []net.IP
	for i := uint32(1); i+1 < total; i++ { // sin dirección de red ni de difusión
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, base+i)
		ips = append(ips, ip)
	}
	return ips
}

// EscanearRed prueba el puerto (9100) de cada dirección de las subredes, con concurrencia y
// tiempo limitados.
func EscanearRed(ctx context.Context, op OpcionesDeteccion) []Detectada {
	if op.Puerto == 0 {
		op.Puerto = 9100
	}
	if op.Concurrencia <= 0 {
		op.Concurrencia = 64
	}
	if op.TiempoHost <= 0 {
		op.TiempoHost = 400 * time.Millisecond
	}
	if op.TiempoTotal <= 0 {
		op.TiempoTotal = 15 * time.Second
	}
	subredes := op.Subredes
	if len(subredes) == 0 {
		subredes = SubredesLocales()
	}
	ctx, fin := context.WithTimeout(ctx, op.TiempoTotal)
	defer fin()

	var (
		mu          sync.Mutex
		encontradas []Detectada
		wg          sync.WaitGroup
		turnos      = make(chan struct{}, op.Concurrencia)
	)
	for _, red := range subredes {
		for _, ip := range hosts(red) {
			select {
			case <-ctx.Done():
			case turnos <- struct{}{}:
				wg.Add(1)
				go func(ip net.IP) {
					defer func() { <-turnos; wg.Done() }()
					dir := net.JoinHostPort(ip.String(), strconv.Itoa(op.Puerto))
					d := net.Dialer{Timeout: op.TiempoHost}
					c, err := d.DialContext(ctx, "tcp", dir)
					if err != nil {
						return
					}
					c.Close()
					mu.Lock()
					encontradas = append(encontradas, Detectada{Tipo: "red", Destino: dir,
						Detalle: fmt.Sprintf("responde en el puerto %d", op.Puerto)})
					mu.Unlock()
				}(ip)
				continue
			}
			break
		}
	}
	wg.Wait()
	sort.Slice(encontradas, func(i, j int) bool {
		a, _, _ := net.SplitHostPort(encontradas[i].Destino)
		b, _, _ := net.SplitHostPort(encontradas[j].Destino)
		return binary.BigEndian.Uint32(net.ParseIP(a).To4()) < binary.BigEndian.Uint32(net.ParseIP(b).To4())
	})
	return encontradas
}
