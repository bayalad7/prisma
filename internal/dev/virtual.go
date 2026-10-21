package dev

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/bayalad7/prisma/internal/almacen"
)

// ImpresoraVirtual escucha por TCP como una impresora de red (puerto 9100): guarda cada trabajo
// recibido como .bin y muestra su traza decodificada y la previsualización. Atiende una conexión
// a la vez, como las impresoras reales, y responde a la consulta de estado DLE EOT como una
// impresora en línea (así se puede probar «Confirmar impresión»).
type ImpresoraVirtual struct {
	Perfil     almacen.Impresora
	Directorio string    // dónde guardar los .bin
	Salida     io.Writer // traza y previsualización
	SinTraza   bool      // muestra solo la previsualización
	Color      bool

	mu        sync.Mutex
	recibidos int
}

// Recibidos devuelve cuántos trabajos (conexiones con datos) se recibieron.
func (v *ImpresoraVirtual) Recibidos() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.recibidos
}

// Servir atiende conexiones hasta que se cancela el contexto.
func (v *ImpresoraVirtual) Servir(ctx context.Context, ln net.Listener) error {
	if err := os.MkdirAll(v.Directorio, 0o755); err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		ln.Close()
	}()
	for {
		c, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		v.atender(c)
	}
}

func (v *ImpresoraVirtual) atender(c net.Conn) {
	defer c.Close()
	// Lee hasta que el cliente cierre o deje de enviar durante 5 s.
	var datos []byte
	buf := make([]byte, 32*1024)
	for {
		c.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, err := c.Read(buf)
		previos := len(datos)
		datos = append(datos, buf[:n]...)
		// Se busca desde 2 bytes antes por si la consulta llegó partida entre dos lecturas.
		for range consultasEstado(datos[max(0, previos-2):]) {
			c.Write([]byte{EstadoEnLinea})
		}
		if err != nil {
			var ne net.Error
			if !errors.Is(err, io.EOF) && !(errors.As(err, &ne) && ne.Timeout()) {
				fmt.Fprintf(v.Salida, "conexión de %s interrumpida: %v\n", c.RemoteAddr(), err)
			}
			break
		}
	}
	if len(datos) == 0 {
		return // prueba de conexión (Probar) sin datos
	}

	v.mu.Lock()
	v.recibidos++
	n := v.recibidos
	v.mu.Unlock()
	nombre := fmt.Sprintf("%s-%03d.bin", time.Now().Format("20060102-150405"), n)
	ruta := filepath.Join(v.Directorio, nombre)
	if err := os.WriteFile(ruta, datos, 0o644); err != nil {
		fmt.Fprintf(v.Salida, "no se pudo guardar %s: %v\n", ruta, err)
	}

	fmt.Fprintf(v.Salida, "\n══ Trabajo %d · %d bytes desde %s · %s ══\n", n, len(datos), c.RemoteAddr(), ruta)
	if !v.SinTraza {
		fmt.Fprint(v.Salida, Traza(datos))
		fmt.Fprintln(v.Salida)
	}
	fmt.Fprint(v.Salida, VistaTexto(Simular(datos, v.Perfil), v.Color))
}

// EstadoEnLinea es la respuesta a DLE EOT n de una impresora sin problemas: los bits fijos 1 y 4.
const EstadoEnLinea = 0x12

// consultasEstado cuenta las consultas DLE EOT n (10 04 n, n de 1 a 4) completas en d.
func consultasEstado(d []byte) int {
	n := 0
	for i := 0; i+2 < len(d); i++ {
		if d[i] == 0x10 && d[i+1] == 0x04 && d[i+2] >= 1 && d[i+2] <= 4 {
			n++
			i += 2
		}
	}
	return n
}
