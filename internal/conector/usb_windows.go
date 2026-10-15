//go:build windows

package conector

import (
	"context"
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	winspool             = windows.NewLazySystemDLL("winspool.drv")
	procOpenPrinterW     = winspool.NewProc("OpenPrinterW")
	procClosePrinter     = winspool.NewProc("ClosePrinter")
	procStartDocPrinterW = winspool.NewProc("StartDocPrinterW")
	procStartPagePrinter = winspool.NewProc("StartPagePrinter")
	procWritePrinter     = winspool.NewProc("WritePrinter")
	procEndPagePrinter   = winspool.NewProc("EndPagePrinter")
	procEndDocPrinter    = winspool.NewProc("EndDocPrinter")
	procGetPrinterW      = winspool.NewProc("GetPrinterW")
	procEnumPrintersW    = winspool.NewProc("EnumPrintersW")
)

// docInfo1 es DOC_INFO_1W.
type docInfo1 struct {
	nombreDoc, archivoSalida, tipoDatos *uint16
}

// printerInfo2 es PRINTER_INFO_2W.
type printerInfo2 struct {
	servidor, nombre, compartido, puerto, controlador, comentario, ubicacion      *uint16
	devMode                                                                       uintptr
	separador, procesador, tipoDatos, parametros                                  *uint16
	descriptorSeguridad                                                           uintptr
	atributos, prioridad, prioridadPorDefecto, inicio, fin, estado, trabajos, ppm uint32
}

const (
	estadoPausada               = 0x00000001
	estadoError                 = 0x00000002
	estadoSinPapel              = 0x00000010
	estadoFueraDeLinea          = 0x00000080
	atributoTrabajarSinConexion = 0x00000400
)

// Winspool envía datos RAW a una impresora instalada en Windows, por su nombre.
type Winspool struct {
	Nombre string
}

func nuevoUSB(destino string) Conector { return &Winspool{Nombre: destino} }

func (w *Winspool) abrir() (windows.Handle, error) {
	nombre, err := windows.UTF16PtrFromString(w.Nombre)
	if err != nil {
		return 0, err
	}
	var h windows.Handle
	r, _, e := procOpenPrinterW.Call(uintptr(unsafe.Pointer(nombre)), uintptr(unsafe.Pointer(&h)), 0)
	if r == 0 {
		return 0, w.traducir(e)
	}
	return h, nil
}

func cerrarImpresora(h windows.Handle) { procClosePrinter.Call(uintptr(h)) }

// Enviar crea un documento RAW en la cola de impresión de Windows. El controlador no
// interpreta los bytes: llegan tal cual a la impresora.
func (w *Winspool) Enviar(ctx context.Context, datos []byte) error {
	if len(datos) == 0 {
		return nil
	}
	h, err := w.abrir()
	if err != nil {
		return err
	}
	defer cerrarImpresora(h)

	nombreDoc := "PRISMA"
	if id := trabajoDe(ctx); id != "" {
		nombreDoc += " " + id
	}
	doc := docInfo1{nombreDoc: windows.StringToUTF16Ptr(nombreDoc), tipoDatos: windows.StringToUTF16Ptr("RAW")}
	if r, _, e := procStartDocPrinterW.Call(uintptr(h), 1, uintptr(unsafe.Pointer(&doc))); r == 0 {
		return w.traducir(e)
	}
	defer procEndDocPrinter.Call(uintptr(h))
	if r, _, e := procStartPagePrinter.Call(uintptr(h)); r == 0 {
		return w.traducir(e)
	}
	defer procEndPagePrinter.Call(uintptr(h))

	return conPlazo(ctx, TiempoEscritura, func() error {
		for len(datos) > 0 {
			var escritos uint32
			r, _, e := procWritePrinter.Call(uintptr(h), uintptr(unsafe.Pointer(&datos[0])), uintptr(len(datos)), uintptr(unsafe.Pointer(&escritos)))
			if r == 0 {
				return w.traducir(e)
			}
			datos = datos[escritos:]
		}
		return nil
	}, nil)
}

// Probar abre la impresora y revisa su estado en la cola de Windows.
func (w *Winspool) Probar(context.Context) error {
	h, err := w.abrir()
	if err != nil {
		return err
	}
	defer cerrarImpresora(h)
	var necesario uint32
	procGetPrinterW.Call(uintptr(h), 2, 0, 0, uintptr(unsafe.Pointer(&necesario)))
	if necesario == 0 {
		return nil
	}
	buf := make([]byte, necesario)
	if r, _, e := procGetPrinterW.Call(uintptr(h), 2, uintptr(unsafe.Pointer(&buf[0])), uintptr(necesario), uintptr(unsafe.Pointer(&necesario))); r == 0 {
		return w.traducir(e)
	}
	info := (*printerInfo2)(unsafe.Pointer(&buf[0]))
	switch {
	case info.atributos&atributoTrabajarSinConexion != 0 || info.estado&estadoFueraDeLinea != 0:
		return nuevoError(ErrImpresoraInalcanzable, fmt.Sprintf("Windows marca la impresora %q como sin conexión.", w.Nombre),
			"Revisa el cable USB y que esté encendida; en Configuración > Impresoras desmarca «Usar impresora sin conexión».", nil)
	case info.estado&estadoSinPapel != 0:
		return nuevoError(ErrImpresoraInalcanzable, fmt.Sprintf("La impresora %q no tiene papel.", w.Nombre), "Cambia el rollo de papel.", nil)
	case info.estado&(estadoError|estadoPausada) != 0:
		return nuevoError(ErrImpresoraInalcanzable, fmt.Sprintf("La impresora %q está en pausa o con error en Windows.", w.Nombre),
			"Abre su cola de impresión en Windows, reanúdala y elimina los trabajos atascados.", nil)
	}
	return nil
}

func (w *Winspool) traducir(e error) error {
	var errno windows.Errno
	if errors.As(e, &errno) {
		switch errno {
		case windows.ERROR_INVALID_PRINTER_NAME:
			return nuevoError(ErrImpresoraNoEncontrada, fmt.Sprintf("Windows no tiene ninguna impresora llamada %q.", w.Nombre),
				"Usa el nombre exacto que aparece en Configuración > Impresoras (ver «prisma impresoras detectar»).", e)
		case windows.ERROR_ACCESS_DENIED:
			return nuevoError(ErrPermisoDenegado, fmt.Sprintf("No hay permiso para imprimir en %q.", w.Nombre),
				"Da permiso de impresión al usuario del servicio PRISMA en las propiedades de la impresora.", e)
		}
	}
	return nuevoError(ErrImpresoraInalcanzable, fmt.Sprintf("Windows no pudo enviar el trabajo a %q.", w.Nombre),
		"Revisa la cola de impresión de Windows y que la impresora esté encendida.", e)
}

func impresorasSistema(context.Context) []Detectada {
	ds, _ := impresorasWindows()
	return ds
}

// impresorasWindows lista las impresoras locales y conectadas (EnumPrinters nivel 2).
func impresorasWindows() ([]Detectada, error) {
	const flags = 0x2 | 0x4 // PRINTER_ENUM_LOCAL | PRINTER_ENUM_CONNECTIONS
	var necesario, cantidad uint32
	procEnumPrintersW.Call(flags, 0, 2, 0, 0, uintptr(unsafe.Pointer(&necesario)), uintptr(unsafe.Pointer(&cantidad)))
	if necesario == 0 {
		return nil, nil
	}
	buf := make([]byte, necesario)
	r, _, e := procEnumPrintersW.Call(flags, 0, 2, uintptr(unsafe.Pointer(&buf[0])), uintptr(necesario),
		uintptr(unsafe.Pointer(&necesario)), uintptr(unsafe.Pointer(&cantidad)))
	if r == 0 {
		return nil, e
	}
	infos := unsafe.Slice((*printerInfo2)(unsafe.Pointer(&buf[0])), cantidad)
	var ds []Detectada
	for _, info := range infos {
		ds = append(ds, Detectada{
			Tipo:    "usb",
			Destino: windows.UTF16PtrToString(info.nombre),
			Detalle: fmt.Sprintf("puerto %s, controlador %s", windows.UTF16PtrToString(info.puerto), windows.UTF16PtrToString(info.controlador)),
		})
	}
	return ds, nil
}
