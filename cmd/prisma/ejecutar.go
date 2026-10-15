package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/config"
	"github.com/bayalad7/prisma/internal/version"
)

func nuevoComandoEjecutar(op *opciones) *cobra.Command {
	return &cobra.Command{
		Use:   "ejecutar",
		Short: "Ejecuta el puente en primer plano (Ctrl+C para detenerlo)",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			ctx, detener := signal.NotifyContext(c.Context(), os.Interrupt, syscall.SIGTERM)
			defer detener()
			return ejecutar(ctx, op.rutaConfig, slog.New(slog.NewJSONHandler(c.ErrOrStderr(), nil)))
		},
	}
}

func ejecutar(ctx context.Context, rutaConfig string, log *slog.Logger) error {
	cfg, origen, err := config.Cargar(rutaConfig)
	if err != nil {
		return fmt.Errorf("%s: %w", origen.Ruta, err)
	}
	log.Info("iniciando PRISMA", "version", version.Version, "config", origen.Ruta, "config_existe", origen.Existe,
		"modo_desarrollo", cfg.Dev.ModoDesarrollo)

	ruta := cfg.RutaBaseDatos()
	alm, err := almacen.Abrir(ctx, ruta)
	if err != nil {
		return err
	}
	defer alm.Cerrar()
	esquema, err := alm.VersionEsquema(ctx)
	if err != nil {
		return err
	}
	log.Info("base de datos lista", "ruta", ruta, "esquema", esquema)

	// Al arrancar no hay nada en la cola: lo que quedó pendiente se interrumpió con el puente.
	if n, err := alm.Trabajos.MarcarInterrumpidos(ctx); err != nil {
		return err
	} else if n > 0 {
		log.Warn("trabajos interrumpidos por el reinicio del puente", "cantidad", n)
	}
	if err := mantenimiento(ctx, alm, log); err != nil {
		return err
	}
	if _, ok, err := alm.Ajustes.Obtener(ctx, almacen.AjusteContrasenaAdmin); err != nil {
		return err
	} else if !ok {
		log.Warn("el panel aún no tiene contraseña de administrador; créala con \"prisma admin restablecer-contrasena\"")
	}

	log.Warn("el servidor HTTPS todavía no está implementado; el puente solo mantiene la base de datos",
		"escuchar", cfg.Servidor.Escuchar, "puerto", cfg.Servidor.Puerto, "modo_tls", cfg.Servidor.ModoTLS)

	cada := time.NewTicker(time.Hour)
	defer cada.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info("PRISMA detenido")
			return nil
		case <-cada.C:
			if err := mantenimiento(ctx, alm, log); err != nil {
				log.Error("falló el mantenimiento de la base de datos", "error", err.Error())
			}
		}
	}
}

// mantenimiento purga el historial de trabajos vencido y los ids de idempotencia expirados.
func mantenimiento(ctx context.Context, alm *almacen.Almacen, log *slog.Logger) error {
	dias, err := alm.Ajustes.RetencionDias(ctx)
	if err != nil {
		return err
	}
	trabajos, err := alm.Trabajos.Purgar(ctx, time.Now().AddDate(0, 0, -dias))
	if err != nil {
		return err
	}
	ids, err := alm.IDsRecientes.Purgar(ctx)
	if err != nil {
		return err
	}
	if trabajos+ids > 0 {
		log.Info("mantenimiento de la base de datos", "trabajos_purgados", trabajos, "ids_purgados", ids, "retencion_dias", dias)
	}
	return nil
}
