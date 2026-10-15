//go:build !windows

package conector

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// nuevoUSB usa el dispositivo directamente si el destino es una ruta /dev/… (Linux: /dev/usb/lpN);
// si no, lo trata como el nombre de una cola de CUPS.
func nuevoUSB(destino string) Conector {
	if strings.HasPrefix(destino, "/dev/") {
		return &Dispositivo{Ruta: destino}
	}
	return &CUPS{Cola: destino}
}

// Dispositivo escribe directamente en el archivo de dispositivo de la impresora.
type Dispositivo struct {
	Ruta string
}

func (d *Dispositivo) abrir() (*os.File, error) {
	f, err := os.OpenFile(d.Ruta, os.O_WRONLY, 0)
	if err != nil {
		return nil, d.traducir(err)
	}
	return f, nil
}

func (d *Dispositivo) Enviar(ctx context.Context, datos []byte) error {
	f, err := d.abrir()
	if err != nil {
		return err
	}
	defer f.Close()
	err = conPlazo(ctx, TiempoEscritura, func() error {
		_, err := f.Write(datos)
		return err
	}, func() { f.Close() })
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled):
		return errorTiempo("recibir los datos", err)
	}
	return d.traducir(err)
}

func (d *Dispositivo) Probar(context.Context) error {
	f, err := d.abrir()
	if err != nil {
		return err
	}
	return f.Close()
}

func (d *Dispositivo) traducir(err error) error {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nuevoError(ErrImpresoraNoEncontrada, fmt.Sprintf("No existe el dispositivo %s.", d.Ruta),
			"Conecta la impresora por USB y revisa su ruta con «prisma impresoras detectar».", err)
	case errors.Is(err, fs.ErrPermission):
		return nuevoError(ErrPermisoDenegado, fmt.Sprintf("No hay permiso para escribir en %s.", d.Ruta),
			"Agrega el usuario del servicio PRISMA al grupo lp.", err)
	case errors.Is(err, syscall.EBUSY):
		return nuevoError(ErrPuertoOcupado, fmt.Sprintf("El dispositivo %s está en uso.", d.Ruta),
			"Otro programa (por ejemplo CUPS) tiene la impresora abierta; usa la cola de CUPS como destino o detén ese programa.", err)
	}
	return nuevoError(ErrImpresoraInalcanzable, fmt.Sprintf("No se pudo escribir en %s.", d.Ruta),
		"Revisa que la impresora esté encendida y conectada.", err)
}

// CUPS envía con `lp -d <cola> -o raw`, sin que CUPS interprete los bytes.
type CUPS struct {
	Cola string
}

func comandoC(ctx context.Context, nombre string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, nombre, args...)
	cmd.Env = append(os.Environ(), "LC_ALL=C") // mensajes en inglés, estables para interpretarlos
	return cmd
}

func (c *CUPS) Enviar(ctx context.Context, datos []byte) error {
	ctx, fin := context.WithTimeout(ctx, TiempoEscritura)
	defer fin()
	cmd := comandoC(ctx, "lp", "-d", c.Cola, "-o", "raw")
	cmd.Stdin = bytes.NewReader(datos)
	salida, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return errorTiempo("entregar el trabajo a CUPS", ctx.Err())
		}
		return c.traducir(err, string(salida))
	}
	return nil
}

func (c *CUPS) Probar(ctx context.Context) error {
	ctx, fin := context.WithTimeout(ctx, TiempoConexion)
	defer fin()
	salida, err := comandoC(ctx, "lpstat", "-p", c.Cola).CombinedOutput()
	if err != nil {
		return c.traducir(err, string(salida))
	}
	if strings.Contains(string(salida), "disabled") {
		return nuevoError(ErrImpresoraInalcanzable, fmt.Sprintf("La cola de CUPS %q está deshabilitada.", c.Cola),
			fmt.Sprintf("Habilítala con «cupsenable %s» y revisa por qué se detuvo en http://localhost:631.", c.Cola), nil)
	}
	return nil
}

func (c *CUPS) traducir(err error, salida string) error {
	if errors.Is(err, exec.ErrNotFound) {
		return nuevoError(ErrImpresoraNoEncontrada, "No se encontró el comando lp de CUPS.",
			"Instala CUPS (paquete cups-client) o usa la ruta del dispositivo, por ejemplo /dev/usb/lp0.", err)
	}
	if strings.Contains(salida, "does not exist") || strings.Contains(salida, "Unknown destination") || strings.Contains(salida, "Invalid destination") {
		return nuevoError(ErrImpresoraNoEncontrada, fmt.Sprintf("CUPS no tiene ninguna cola llamada %q.", c.Cola),
			"Usa el nombre exacto que muestra «lpstat -e» o «prisma impresoras detectar».", err)
	}
	if strings.Contains(salida, "Forbidden") || strings.Contains(salida, "not authorized") {
		return nuevoError(ErrPermisoDenegado, fmt.Sprintf("CUPS no permite imprimir en %q.", c.Cola),
			"Revisa las políticas de la cola en http://localhost:631.", err)
	}
	return nuevoError(ErrImpresoraInalcanzable, fmt.Sprintf("CUPS no aceptó el trabajo para %q: %s", c.Cola, strings.TrimSpace(salida)),
		"Revisa el estado de la cola en http://localhost:631.", err)
}

// impresorasSistema lista colas de CUPS (lpstat -e / -v) y dispositivos /dev/usb/lp*.
func impresorasSistema(ctx context.Context) []Detectada {
	var ds []Detectada
	uris := map[string]string{}
	if salida, err := comandoC(ctx, "lpstat", "-v").Output(); err == nil {
		// "device for COLA: usb://Fabricante/Modelo?serial=…"
		for _, l := range strings.Split(string(salida), "\n") {
			if resto, ok := strings.CutPrefix(l, "device for "); ok {
				if nombre, uri, ok := strings.Cut(resto, ": "); ok {
					uris[nombre] = uri
				}
			}
		}
	}
	if salida, err := comandoC(ctx, "lpstat", "-e").Output(); err == nil {
		for _, nombre := range strings.Fields(string(salida)) {
			ds = append(ds, Detectada{Tipo: "usb", Destino: nombre, Detalle: "cola de CUPS " + uris[nombre]})
		}
	}
	dispositivos, _ := filepath.Glob("/dev/usb/lp*")
	for _, d := range dispositivos {
		ds = append(ds, Detectada{Tipo: "usb", Destino: d, Detalle: "dispositivo USB directo"})
	}
	return ds
}
