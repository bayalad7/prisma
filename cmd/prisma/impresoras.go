package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/conector"
	"github.com/bayalad7/prisma/internal/config"
	"github.com/bayalad7/prisma/internal/dev"
	"github.com/bayalad7/prisma/internal/escpos"
)

func nuevoComandoImpresoras(op *opciones) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "impresoras",
		Short: "Detecta, configura y prueba impresoras",
	}
	cmd.AddCommand(
		nuevoComandoDetectar(),
		nuevoComandoListarImpresoras(op),
		nuevoComandoAgregarImpresora(op),
		nuevoComandoEliminarImpresora(op),
		nuevoComandoProbarImpresora(op),
	)
	return cmd
}

// abrirAlmacen carga la configuración y abre la base de datos (para los comandos locales).
func abrirAlmacen(ctx context.Context, op *opciones) (*almacen.Almacen, config.Config, error) {
	cfg, origen, err := config.Cargar(op.rutaConfig)
	if err != nil {
		return nil, cfg, fmt.Errorf("%s: %w", origen.Ruta, err)
	}
	alm, err := almacen.Abrir(ctx, cfg.RutaBaseDatos())
	return alm, cfg, err
}

func nuevoComandoDetectar() *cobra.Command {
	var comoJSON, red bool
	var subredes []string
	cmd := &cobra.Command{
		Use:   "detectar",
		Short: "Lista las impresoras y puertos de este equipo (y, con --red, de la red local)",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			op := conector.OpcionesDeteccion{Red: red || len(subredes) > 0}
			for _, s := range subredes {
				_, ipn, err := net.ParseCIDR(s)
				if err != nil {
					return fmt.Errorf("subred %q no válida; usa la forma 192.168.1.0/24", s)
				}
				op.Subredes = append(op.Subredes, ipn)
			}
			w := c.OutOrStdout()
			if op.Red && !comoJSON {
				redes := op.Subredes
				if len(redes) == 0 {
					redes = conector.SubredesLocales()
				}
				fmt.Fprintf(w, "Buscando impresoras en el puerto 9100 de %v (hasta 15 s)…\n", redes)
			}
			ds := conector.Detectar(c.Context(), op)
			if comoJSON {
				e := json.NewEncoder(w)
				e.SetIndent("", "  ")
				if ds == nil {
					ds = []conector.Detectada{}
				}
				return e.Encode(ds)
			}
			if len(ds) == 0 {
				fmt.Fprintln(w, "No se encontraron impresoras. Prueba con --red para buscar en la red local.")
				return nil
			}
			tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "TIPO\tDESTINO\tDETALLE")
			for _, d := range ds {
				fmt.Fprintf(tw, "%s\t%s\t%s\n", d.Tipo, d.Destino, d.Detalle)
			}
			tw.Flush()
			fmt.Fprintln(w, "\nUsa TIPO y DESTINO con «prisma impresoras agregar».")
			return nil
		},
	}
	cmd.Flags().BoolVar(&comoJSON, "json", false, "salida en JSON")
	cmd.Flags().BoolVar(&red, "red", false, "busca también impresoras de red (puerto 9100) en la subred local")
	cmd.Flags().StringSliceVar(&subredes, "subred", nil, "subred a escanear en lugar de las locales, ej. 192.168.1.0/24 (implica --red)")
	return cmd
}

func nuevoComandoListarImpresoras(op *opciones) *cobra.Command {
	return &cobra.Command{
		Use:   "listar",
		Short: "Lista las impresoras configuradas en el puente",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			alm, _, err := abrirAlmacen(c.Context(), op)
			if err != nil {
				return err
			}
			defer alm.Cerrar()
			lista, err := alm.Impresoras.Listar(c.Context())
			if err != nil {
				return err
			}
			w := c.OutOrStdout()
			if len(lista) == 0 {
				fmt.Fprintln(w, "No hay impresoras configuradas. Agrega una con «prisma impresoras agregar».")
				return nil
			}
			tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "ALIAS\tTIPO\tDESTINO\tANCHO\tCOLUMNAS\tPÁGINA\tACTIVA")
			for _, i := range lista {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%d mm\t%d / %d\t%s\t%s\n", i.Alias, i.Tipo, i.Destino, i.AnchoMM,
					i.Columnas, i.ColumnasFuenteB, i.PaginaCodigos, map[bool]string{true: "sí", false: "no"}[i.Activa])
			}
			return tw.Flush()
		},
	}
}

var patronAlias = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

func nuevoComandoAgregarImpresora(op *opciones) *cobra.Command {
	var imp almacen.Impresora
	var ancho int
	var cajon, sinCorte, sinQR, sinBarras, sinImagen, inactiva bool
	cmd := &cobra.Command{
		Use:   "agregar --alias caja --tipo red --destino 192.168.1.50:9100",
		Short: "Agrega una impresora al puente",
		Example: "  prisma impresoras agregar --alias caja --tipo red --destino 192.168.1.50 --cajon\n" +
			"  prisma impresoras agregar --alias cocina --tipo usb --destino \"EPSON TM-T20II Receipt\"\n" +
			"  prisma impresoras agregar --alias barra --tipo bluetooth --destino COM5 --ancho 58\n" +
			"  prisma impresoras agregar --alias virtual --tipo red --destino 127.0.0.1:9100",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			if !patronAlias.MatchString(imp.Alias) {
				return fmt.Errorf("alias %q no válido: usa minúsculas, dígitos, - o _ (hasta 32 caracteres)", imp.Alias)
			}
			tipos := []string{almacen.ConexionRed, almacen.ConexionUSB, almacen.ConexionBluetooth, almacen.ConexionArchivo}
			if !slices.Contains(tipos, imp.Tipo) {
				return fmt.Errorf("tipo %q no válido; usa %s", imp.Tipo, strings.Join(tipos, ", "))
			}
			if !slices.Contains(escpos.PaginasSoportadas(), strings.ToUpper(imp.PaginaCodigos)) {
				return fmt.Errorf("página de códigos %q no soportada; usa %s", imp.PaginaCodigos, strings.Join(escpos.PaginasSoportadas(), ", "))
			}
			base, err := dev.PerfilPrueba(ancho)
			if err != nil {
				return err
			}
			// Las columnas y puntos dependen del ancho; las banderas explícitas los reemplazan.
			f := c.Flags()
			if !f.Changed("columnas") {
				imp.Columnas = base.Columnas
			}
			if !f.Changed("columnas-b") {
				imp.ColumnasFuenteB = base.ColumnasFuenteB
			}
			if !f.Changed("ancho-px") {
				imp.AnchoPx = base.AnchoPx
			}
			if imp.Nombre == "" {
				imp.Nombre = imp.Alias
			}
			imp.AnchoMM, imp.PaginaCodigos = ancho, strings.ToUpper(imp.PaginaCodigos)
			imp.SoportaCajon, imp.SoportaCorte, imp.SoportaQRNativo = cajon, !sinCorte, !sinQR
			imp.SoportaCodigoBarrasNativo, imp.SoportaImagen, imp.Activa = !sinBarras, !sinImagen, !inactiva

			alm, _, err := abrirAlmacen(c.Context(), op)
			if err != nil {
				return err
			}
			defer alm.Cerrar()
			if err := alm.Impresoras.Crear(c.Context(), &imp); err != nil {
				if errors.Is(err, almacen.ErrDuplicado) {
					return fmt.Errorf("ya existe una impresora con el alias %q", imp.Alias)
				}
				return err
			}
			fmt.Fprintf(c.OutOrStdout(), "Impresora %q agregada (%s → %s, %d mm, %d columnas). Pruébala con «prisma impresoras probar %s».\n",
				imp.Alias, imp.Tipo, imp.Destino, imp.AnchoMM, imp.Columnas, imp.Alias)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&imp.Alias, "alias", "", "alias con el que las aplicaciones eligen la impresora (obligatorio)")
	f.StringVar(&imp.Nombre, "nombre", "", "nombre descriptivo (por defecto, el alias)")
	f.StringVar(&imp.Tipo, "tipo", "", "red, usb, bluetooth o archivo (obligatorio)")
	f.StringVar(&imp.Destino, "destino", "", "IP[:puerto], nombre de la impresora o cola, puerto serie o carpeta (obligatorio)")
	f.IntVar(&ancho, "ancho", 80, "ancho del papel en mm: 58 u 80")
	f.IntVar(&imp.Columnas, "columnas", 0, "columnas con la fuente A (por defecto según el ancho: 32 o 48)")
	f.IntVar(&imp.ColumnasFuenteB, "columnas-b", 0, "columnas con la fuente B (por defecto 42 o 64)")
	f.IntVar(&imp.AnchoPx, "ancho-px", 0, "puntos imprimibles por línea (por defecto 384 o 576)")
	f.StringVar(&imp.PaginaCodigos, "pagina", "CP858", "página de códigos: "+strings.Join(escpos.PaginasSoportadas(), ", "))
	f.IntVar(&imp.ComandoPaginaCodigos, "comando-pagina", 19, "n de ESC t n para esa página (Epson: CP437=0, CP850=2, WPC1252=16, CP858=19)")
	f.IntVar(&imp.Baudios, "baudios", 0, "velocidad del puerto serie (bluetooth); por defecto 9600")
	f.BoolVar(&cajon, "cajon", false, "tiene cajón de dinero conectado")
	f.BoolVar(&sinCorte, "sin-corte", false, "no tiene cuchilla")
	f.BoolVar(&sinQR, "sin-qr-nativo", false, "no imprime QR por comando (se envía como imagen)")
	f.BoolVar(&sinBarras, "sin-barras-nativo", false, "no imprime códigos de barras por comando (se envían como imagen)")
	f.BoolVar(&sinImagen, "sin-imagen", false, "no imprime imágenes")
	f.BoolVar(&inactiva, "inactiva", false, "la agrega desactivada")
	for _, obligatoria := range []string{"alias", "tipo", "destino"} {
		cmd.MarkFlagRequired(obligatoria)
	}
	return cmd
}

func nuevoComandoEliminarImpresora(op *opciones) *cobra.Command {
	return &cobra.Command{
		Use:   "eliminar alias",
		Short: "Elimina una impresora del puente",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			alm, _, err := abrirAlmacen(c.Context(), op)
			if err != nil {
				return err
			}
			defer alm.Cerrar()
			if err := alm.Impresoras.Eliminar(c.Context(), args[0]); errors.Is(err, almacen.ErrNoEncontrado) {
				return fmt.Errorf("no existe ninguna impresora con el alias %q", args[0])
			} else if err != nil {
				return err
			}
			fmt.Fprintf(c.OutOrStdout(), "Impresora %q eliminada.\n", args[0])
			return nil
		},
	}
}

func obtenerImpresora(ctx context.Context, alm *almacen.Almacen, alias string) (*almacen.Impresora, error) {
	imp, err := alm.Impresoras.Obtener(ctx, alias)
	if errors.Is(err, almacen.ErrNoEncontrado) {
		return nil, &conector.Error{Codigo: conector.ErrImpresoraNoEncontrada,
			Mensaje:    fmt.Sprintf("No hay ninguna impresora configurada con el alias %q.", alias),
			Sugerencia: "Revisa «prisma impresoras listar» o agrégala con «prisma impresoras agregar»."}
	}
	return imp, err
}

func nuevoComandoProbarImpresora(op *opciones) *cobra.Command {
	return &cobra.Command{
		Use:   "probar alias",
		Short: "Comprueba que la impresora esté alcanzable (sin imprimir)",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			alm, _, err := abrirAlmacen(c.Context(), op)
			if err != nil {
				return err
			}
			defer alm.Cerrar()
			imp, err := obtenerImpresora(c.Context(), alm, args[0])
			if err != nil {
				return err
			}
			con, err := conector.Nuevo(*imp)
			if err != nil {
				return err
			}
			ctx, fin := context.WithTimeout(c.Context(), 10*time.Second)
			defer fin()
			if err := con.Probar(ctx); err != nil {
				return err
			}
			fmt.Fprintf(c.OutOrStdout(), "La impresora %q (%s → %s) está en línea.\n", imp.Alias, imp.Tipo, imp.Destino)
			return nil
		},
	}
}
