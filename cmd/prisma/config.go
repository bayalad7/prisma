package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/bayalad7/prisma/internal/config"
	"github.com/bayalad7/prisma/internal/rutas"
)

func nuevoComandoConfig(op *opciones) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Consulta y valida config.toml",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "validar",
			Short: "Valida config.toml y muestra la configuración efectiva",
			Args:  cobra.NoArgs,
			RunE: func(c *cobra.Command, _ []string) error {
				cfg, origen, err := config.Cargar(op.rutaConfig)
				if err != nil {
					return fmt.Errorf("%s: %w", origen.Ruta, err)
				}
				w := c.OutOrStdout()
				if origen.Existe {
					fmt.Fprintf(w, "La configuración es válida: %s\n\n", origen.Ruta)
				} else {
					fmt.Fprintf(w, "No existe %s; se usarán los valores por defecto, que son válidos.\n\n", origen.Ruta)
				}
				imprimirResumen(w, cfg)
				return nil
			},
		},
		&cobra.Command{
			Use:   "ruta",
			Short: "Muestra dónde busca PRISMA su configuración y sus datos",
			Args:  cobra.NoArgs,
			RunE: func(c *cobra.Command, _ []string) error {
				w := c.OutOrStdout()
				cfg, origen, err := config.Cargar(op.rutaConfig)
				estado := ""
				if !origen.Existe {
					estado = " (no existe; se usan los valores por defecto)"
				}
				fmt.Fprintf(w, "config:         %s%s\n", origen.Ruta, estado)
				if err != nil {
					// Aun con una configuración inválida, mostramos las rutas por defecto.
					cfg = config.PorDefecto()
					fmt.Fprintf(w, "                (la configuración tiene errores; ejecuta \"prisma config validar\")\n")
				}
				fmt.Fprintf(w, "datos:          %s\n", cfg.DirDatos())
				fmt.Fprintf(w, "base de datos:  %s\n", cfg.RutaBaseDatos())
				fmt.Fprintf(w, "bitácoras:      %s\n", rutas.PorDefecto().Logs)
				return nil
			},
		},
	)
	return cmd
}

func imprimirResumen(w io.Writer, cfg config.Config) {
	s := cfg.Servidor
	fmt.Fprintf(w, "servidor.escuchar    = %s\n", s.Escuchar)
	fmt.Fprintf(w, "servidor.puerto      = %d\n", s.Puerto)
	fmt.Fprintf(w, "servidor.modo_tls    = %s\n", s.ModoTLS)
	if s.ModoTLS == config.TLSArchivos {
		fmt.Fprintf(w, "servidor.certificado = %s\n", s.Certificado)
		fmt.Fprintf(w, "servidor.llave       = %s\n", s.Llave)
	}
	fmt.Fprintf(w, "datos.ruta           = %s\n", cfg.DirDatos())
	fmt.Fprintf(w, "dev.modo_desarrollo  = %t\n", cfg.Dev.ModoDesarrollo)
}
