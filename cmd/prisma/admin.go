package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/bayalad7/prisma/internal/admin"
	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/config"
	"github.com/bayalad7/prisma/internal/seguridad"
)

func nuevoComandoAdmin(op *opciones) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Tareas de administración local",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "restablecer-contrasena",
		Short: "Genera una contraseña nueva para el panel de administración",
		Long: "Genera una contraseña aleatoria para el panel, la muestra una sola vez y obliga a cambiarla\n" +
			"al entrar. Solo funciona en el equipo donde está instalado PRISMA, porque escribe directamente\n" +
			"en su base de datos (requiere permisos de administrador del sistema).",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			cfg, origen, err := config.Cargar(op.rutaConfig)
			if err != nil {
				return fmt.Errorf("%s: %w", origen.Ruta, err)
			}
			alm, err := almacen.Abrir(c.Context(), cfg.RutaBaseDatos())
			if err != nil {
				return err
			}
			defer alm.Cerrar()

			contrasena, err := seguridad.GenerarContrasena()
			if err != nil {
				return err
			}
			hash, err := seguridad.HashContrasena(contrasena)
			if err != nil {
				return err
			}
			err = alm.Ajustes.EstablecerVarios(c.Context(), map[string]string{
				almacen.AjusteContrasenaAdmin: hash,
				almacen.AjusteDebeCambiar:     "1",
			})
			if err != nil {
				return err
			}
			w := c.OutOrStdout()
			fmt.Fprintln(w, "Contraseña del panel restablecida. Anótala; no se volverá a mostrar:")
			fmt.Fprintln(w)
			fmt.Fprintln(w, "    "+contrasena)
			fmt.Fprintln(w)
			fmt.Fprintln(w, "El panel pedirá cambiarla en el próximo inicio de sesión.")
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "codigo-inicial",
		Short: "Genera el código para configurar el panel desde otro equipo",
		Long: "La primera vez que se abre el panel hay que crear la contraseña. Desde el propio equipo del\n" +
			"puente basta con abrir https://localhost:<puerto>/admin; desde otro equipo de la red el panel\n" +
			"pide este código de un solo uso, que vence a los 30 minutos.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			cfg, origen, err := config.Cargar(op.rutaConfig)
			if err != nil {
				return fmt.Errorf("%s: %w", origen.Ruta, err)
			}
			alm, err := almacen.Abrir(c.Context(), cfg.RutaBaseDatos())
			if err != nil {
				return err
			}
			defer alm.Cerrar()
			if hash, ok, err := alm.Ajustes.Obtener(c.Context(), almacen.AjusteContrasenaAdmin); err != nil {
				return err
			} else if ok && hash != "" {
				return fmt.Errorf("el panel ya tiene contraseña; si la olvidaste, usa «prisma admin restablecer-contrasena»")
			}
			codigo, vence, err := admin.GuardarCodigoInicial(c.Context(), alm, time.Now())
			if err != nil {
				return err
			}
			w := c.OutOrStdout()
			fmt.Fprintln(w, "Código de configuración del panel (un solo uso):")
			fmt.Fprintln(w)
			fmt.Fprintln(w, "    "+codigo)
			fmt.Fprintln(w)
			fmt.Fprintf(w, "Escríbelo en el panel antes de las %s.\n", vence.Format("15:04"))
			return nil
		},
	})
	return cmd
}
