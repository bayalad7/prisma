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
// a la vez, como las impresoras reales.
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
		datos = append(datos, buf[:n]...)
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
