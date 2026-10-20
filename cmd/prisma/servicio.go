package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/bayalad7/prisma/internal/config"
	"github.com/bayalad7/prisma/internal/rutas"
	"github.com/bayalad7/prisma/internal/servicio"
)

func nuevoComandoServicio(op *opciones) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "servicio",
		Short: "Instala y controla PRISMA como servicio del sistema",
		Long: "Instala PRISMA como servicio para que arranque solo con el equipo y se reinicie si falla:\n" +
			"servicio de Windows, unidad de systemd (usuario «prisma») o LaunchDaemon de macOS (usuario «_prisma»).\n" +
			"Instalar, desinstalar, iniciar, detener y reiniciar requieren permisos de administrador.",
	}
	accion := func(uso, corto string, f func(servicio.Opciones) error, listo string) *cobra.Command {
		return &cobra.Command{
			Use: uso, Short: corto, Args: cobra.NoArgs,
			RunE: func(c *cobra.Command, _ []string) error {
				so, err := opcionesServicio(op)
				if err != nil {
					return err
				}
				if err := f(so); err != nil {
					return err
				}
				fmt.Fprintln(c.OutOrStdout(), listo)
				return nil
			},
		}
	}
	controlar := func(a string) func(servicio.Opciones) error {
		return func(so servicio.Opciones) error { return servicio.Controlar(so, a) }
	}
	cmd.AddCommand(
		accion("instalar", "Instala el servicio con inicio automático (no lo inicia)", servicio.Instalar,
			"Servicio instalado: arrancará solo con el equipo. Inícialo ahora con «prisma servicio iniciar»."),
		accion("desinstalar", "Detiene y quita el servicio (conserva datos y configuración)", servicio.Desinstalar,
			"Servicio desinstalado. Los datos, la configuración y las bitácoras se conservan."),
		accion("iniciar", "Inicia el servicio", controlar("iniciar"), "Servicio iniciado."),
		accion("detener", "Detiene el servicio", controlar("detener"), "Servicio detenido."),
		accion("reiniciar", "Reinicia el servicio", controlar("reiniciar"), "Servicio reiniciado."),
		&cobra.Command{
			Use: "estado", Short: "Muestra si el servicio está instalado y en marcha", Args: cobra.NoArgs,
			RunE: func(c *cobra.Command, _ []string) error {
				so, err := opcionesServicio(op)
				if err != nil {
					return err
				}
				e, err := servicio.ConsultarEstado(so)
				if err != nil {
					return err
				}
				fmt.Fprintf(c.OutOrStdout(), "Servicio %s (%s): %s\n", servicio.Nombre(), runtime.GOOS, e)
				if e.Instalado {
					fmt.Fprintf(c.OutOrStdout(), "Bitácora: %s\n", filepath.Join(so.Logs, "prisma.log"))
				}
				return nil
			},
		},
		nuevoComandoServicioEjecutar(op),
	)
	return cmd
}

// opcionesServicio arma las opciones desde config.toml: la ruta absoluta de la configuración
// (solo si no es la del sistema), el puerto y las carpetas.
func opcionesServicio(op *opciones) (servicio.Opciones, error) {
	cfg, origen, err := config.Cargar(op.rutaConfig)
	if err != nil {
		return servicio.Opciones{}, fmt.Errorf("%s: %w", origen.Ruta, err)
	}
	so := servicio.Opciones{Datos: cfg.DirDatos(), Logs: rutas.PorDefecto().Logs, Puerto: cfg.Servidor.Puerto}
	if op.rutaConfig != "" {
		if so.RutaConfig, err = filepath.Abs(op.rutaConfig); err != nil {
			return so, err
		}
	}
	return so, nil
}

// nuevoComandoServicioEjecutar es lo que ejecuta el sistema: el puente con su bitácora rotativa.
func nuevoComandoServicioEjecutar(op *opciones) *cobra.Command {
	return &cobra.Command{
		Use:    "ejecutar",
		Short:  "Ejecuta el puente como servicio (lo usa el sistema)",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			so, err := opcionesServicio(op)
			if err != nil {
				return err
			}
			bitacora, err := servicio.AbrirRotativo(filepath.Join(so.Logs, "prisma.log"), servicio.MaxBytesBitacora, servicio.CopiasBitacora)
			if err != nil {
				return fmt.Errorf("no se pudo abrir la bitácora en %s: %w", so.Logs, err)
			}
			defer bitacora.Close()
			var w io.Writer = bitacora
			if runtime.GOOS == "linux" {
				w = io.MultiWriter(bitacora, os.Stderr) // también en journalctl -u prisma
			}
			log := slog.New(slog.NewJSONHandler(w, nil))
			return servicio.Ejecutar(so,
				func(ctx context.Context) error { return ejecutar(ctx, op.rutaConfig, log) },
				func(err error) {
					log.Error("el puente se detuvo por un error; el sistema lo reiniciará", "error", err.Error())
				})
		},
	}
}
