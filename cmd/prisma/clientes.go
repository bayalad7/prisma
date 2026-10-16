package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/seguridad"
)

func nuevoComandoClientes(op *opciones) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clientes",
		Short: "Administra las aplicaciones autorizadas a imprimir",
	}
	cmd.AddCommand(nuevoComandoAgregarCliente(op), nuevoComandoListarClientes(op), nuevoComandoEliminarCliente(op),
		nuevoComandoRegenerarLlave(op))
	return cmd
}

const alfabetoID = "abcdefghijklmnopqrstuvwxyz0123456789"

func nuevoIDCliente() string {
	b := make([]byte, 16)
	rand.Read(b)
	for i := range b {
		b[i] = alfabetoID[int(b[i])%len(alfabetoID)]
	}
	return "cli_" + string(b)
}

// leerLlavePublica acepta la ruta de un archivo PEM o la llave en base64.
func leerLlavePublica(valor string) ([]byte, error) {
	datos := []byte(valor)
	if contenido, err := os.ReadFile(valor); err == nil {
		datos = contenido
	}
	k, err := seguridad.LeerLlavePublica(datos)
	if err != nil {
		return nil, fmt.Errorf("--llave-publica: %w", err)
	}
	return k, nil
}

func mostrarLlave(w io.Writer, llave string) {
	fmt.Fprintln(w, "\nLlave de cliente (anótala; no se volverá a mostrar):")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "    "+llave)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "La aplicación la envía en cada petición como «Authorization: Bearer <llave>».")
}

func nuevoComandoAgregarCliente(op *opciones) *cobra.Command {
	var (
		cli                  almacen.Cliente
		origenes, impresoras []string
		operaciones          []string
		llavePublica         string
		conLlave, inactivo   bool
	)
	cmd := &cobra.Command{
		Use:   "agregar --nombre \"Mi app\" --nivel 2 --origen https://miapp.com",
		Short: "Registra una aplicación y, si corresponde, genera su llave",
		Example: "  prisma clientes agregar --nombre \"Tienda web\" --nivel 1 --origen https://tienda.example\n" +
			"  prisma clientes agregar --nombre \"Caja web\" --nivel 2 --origen https://caja.example --impresoras caja\n" +
			"  prisma clientes agregar --nombre \"Backend\" --nivel 3 --llave-publica publica.pem --operaciones abrir_cajon",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			if cli.Nivel < 1 || cli.Nivel > 3 {
				return fmt.Errorf("--nivel debe ser 1, 2 o 3")
			}
			if cli.Nivel == almacen.NivelOrigen && len(origenes) == 0 {
				return fmt.Errorf("el nivel 1 se identifica solo por el origen: indica al menos un --origen")
			}
			if cli.Nivel == almacen.NivelFirma {
				if llavePublica == "" {
					return fmt.Errorf("el nivel 3 necesita --llave-publica (genera un par de prueba con «prisma dev llaves»)")
				}
				k, err := leerLlavePublica(llavePublica)
				if err != nil {
					return err
				}
				cli.LlavePublica = k
			}
			for _, o := range operaciones {
				if o != almacen.OperacionAbrirCajon && o != almacen.OperacionRaw {
					return fmt.Errorf("operación sensible %q desconocida; usa abrir_cajon o raw", o)
				}
			}
			cli.ID, cli.OperacionesSensibles = nuevoIDCliente(), operaciones
			cli.TodasLasImpresoras, cli.Activo = len(impresoras) == 0, !inactivo

			var llave string
			if cli.Nivel == almacen.NivelLlave || cli.ExigirLlave || conLlave {
				var err error
				if llave, cli.LlavePrefijo, err = seguridad.GenerarLlave(); err != nil {
					return err
				}
				if cli.LlaveHash, err = seguridad.HashLlave(llave); err != nil {
					return err
				}
			}

			alm, _, err := abrirAlmacen(c.Context(), op)
			if err != nil {
				return err
			}
			defer alm.Cerrar()
			if err := alm.Clientes.Crear(c.Context(), &cli); err != nil {
				return err
			}
			for _, o := range origenes {
				if err := alm.Clientes.AgregarOrigen(c.Context(), cli.ID, o); err != nil {
					alm.Clientes.Eliminar(c.Context(), cli.ID)
					if errors.Is(err, almacen.ErrDuplicado) {
						return fmt.Errorf("el origen %s ya pertenece a otra aplicación", o)
					}
					return err
				}
				alm.Solicitudes.Eliminar(c.Context(), o) // si estaba pendiente, ya no lo está
			}
			if len(impresoras) > 0 {
				if err := alm.Clientes.EstablecerImpresoras(c.Context(), cli.ID, false, impresoras); err != nil {
					alm.Clientes.Eliminar(c.Context(), cli.ID)
					return err
				}
			}
			w := c.OutOrStdout()
			fmt.Fprintf(w, "Aplicación %q registrada con id %s (nivel %d).\n", cli.Nombre, cli.ID, cli.Nivel)
			if llave != "" {
				mostrarLlave(w, llave)
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&cli.Nombre, "nombre", "", "nombre de la aplicación (obligatorio)")
	f.IntVar(&cli.Nivel, "nivel", 2, "nivel de seguridad: 1 (origen), 2 (llave) o 3 (firma)")
	f.StringSliceVar(&origenes, "origen", nil, "origen permitido, exacto (ej. https://miapp.com); se puede repetir")
	f.StringSliceVar(&impresoras, "impresoras", nil, "alias permitidos separados por coma (por defecto, todas)")
	f.StringSliceVar(&operaciones, "operaciones", nil, "operaciones sensibles habilitadas: abrir_cajon, raw")
	f.StringVar(&llavePublica, "llave-publica", "", "nivel 3: archivo PEM o base64 de la llave pública Ed25519")
	f.BoolVar(&cli.ExigirLlave, "exigir-llave", false, "nivel 3: exige además la llave de cliente (Bearer pk_…)")
	f.BoolVar(&conLlave, "con-llave", false, "genera una llave aunque el nivel no la exija (para llamadas sin origen)")
	f.BoolVar(&inactivo, "inactivo", false, "lo registra desactivado")
	cmd.MarkFlagRequired("nombre")
	return cmd
}

func nuevoComandoListarClientes(op *opciones) *cobra.Command {
	return &cobra.Command{
		Use:   "listar",
		Short: "Lista las aplicaciones registradas",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			alm, _, err := abrirAlmacen(c.Context(), op)
			if err != nil {
				return err
			}
			defer alm.Cerrar()
			lista, err := alm.Clientes.Listar(c.Context())
			if err != nil {
				return err
			}
			w := c.OutOrStdout()
			if len(lista) == 0 {
				fmt.Fprintln(w, "No hay aplicaciones registradas. Agrega una con «prisma clientes agregar».")
				return nil
			}
			tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tNOMBRE\tNIVEL\tLLAVE\tORÍGENES\tIMPRESORAS\tSENSIBLES\tACTIVO")
			for _, cli := range lista {
				origenes, _ := alm.Clientes.Origenes(c.Context(), cli.ID)
				impresoras := "todas"
				if !cli.TodasLasImpresoras {
					permitidas, _ := alm.Clientes.ImpresorasPermitidas(c.Context(), cli.ID)
					impresoras = strings.Join(permitidas, ",")
				}
				llave := "—"
				if cli.LlavePrefijo != "" {
					llave = cli.LlavePrefijo + "_…"
				}
				fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%s\t%s\t%s\t%s\n", cli.ID, cli.Nombre, cli.Nivel, llave, strings.Join(origenes, " "),
					impresoras, strings.Join(cli.OperacionesSensibles, ","), map[bool]string{true: "sí", false: "no"}[cli.Activo])
			}
			return tw.Flush()
		},
	}
}

func nuevoComandoEliminarCliente(op *opciones) *cobra.Command {
	return &cobra.Command{
		Use:   "eliminar ID",
		Short: "Elimina una aplicación con sus orígenes y permisos",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			alm, _, err := abrirAlmacen(c.Context(), op)
			if err != nil {
				return err
			}
			defer alm.Cerrar()
			if err := alm.Clientes.Eliminar(c.Context(), args[0]); errors.Is(err, almacen.ErrNoEncontrado) {
				return fmt.Errorf("no existe ninguna aplicación con el id %q", args[0])
			} else if err != nil {
				return err
			}
			fmt.Fprintf(c.OutOrStdout(), "Aplicación %s eliminada.\n", args[0])
			return nil
		},
	}
}

func nuevoComandoRegenerarLlave(op *opciones) *cobra.Command {
	return &cobra.Command{
		Use:   "regenerar-llave ID",
		Short: "Genera una llave nueva para la aplicación (la anterior deja de funcionar)",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			alm, _, err := abrirAlmacen(c.Context(), op)
			if err != nil {
				return err
			}
			defer alm.Cerrar()
			cli, err := alm.Clientes.Obtener(c.Context(), args[0])
			if errors.Is(err, almacen.ErrNoEncontrado) {
				return fmt.Errorf("no existe ninguna aplicación con el id %q", args[0])
			} else if err != nil {
				return err
			}
			llave, prefijo, err := seguridad.GenerarLlave()
			if err != nil {
				return err
			}
			if cli.LlaveHash, err = seguridad.HashLlave(llave); err != nil {
				return err
			}
			cli.LlavePrefijo = prefijo
			if err := alm.Clientes.Actualizar(c.Context(), cli); err != nil {
				return err
			}
			fmt.Fprintf(c.OutOrStdout(), "Llave regenerada para %q.\n", cli.Nombre)
			mostrarLlave(c.OutOrStdout(), llave)
			return nil
		},
	}
}
