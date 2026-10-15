package main

import (
	"fmt"

	"github.com/spf13/cobra"

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
	return cmd
}
