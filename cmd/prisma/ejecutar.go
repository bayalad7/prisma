package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/bayalad7/prisma/internal/actualizacion"
	"github.com/bayalad7/prisma/internal/admin"
	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/cola"
	"github.com/bayalad7/prisma/internal/conector"
	"github.com/bayalad7/prisma/internal/config"
	"github.com/bayalad7/prisma/internal/servidor"
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
	// Después de una actualización, la base se copia antes de migrarla al esquema nuevo.
	if copia, err := almacen.RespaldarSiHayMigraciones(ctx, ruta, filepath.Join(cfg.DirDatos(), "respaldos"), version.Version); err != nil {
		return err
	} else if copia != "" {
		log.Info("base de datos respaldada antes de migrarla", "respaldo", copia)
	}
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
		log.Warn("el panel aún no tiene contraseña: ábrelo en este equipo (/admin) o usa \"prisma admin codigo-inicial\" para configurarlo desde otro")
	}

	// Solo se registran los cambios entre en línea y fuera de línea; «ocupada» ocurre en cada trabajo.
	var muEstados sync.Mutex
	ultimoEstado := map[string]cola.Estado{}
	q := cola.Nueva(cola.Opciones{AlCambiarEstado: func(alias string, e cola.Estado, err error) {
		muEstados.Lock()
		defer muEstados.Unlock()
		if e == cola.Ocupada || ultimoEstado[alias] == e {
			return
		}
		ultimoEstado[alias] = e
		if err != nil {
			log.Warn("estado de impresora", "alias", alias, "estado", e, "codigo", conector.Codigo(err))
		} else {
			log.Info("estado de impresora", "alias", alias, "estado", e)
		}
	}})
	defer q.Cerrar()
	srv := servidor.Nuevo(servidor.Opciones{Almacen: alm, Cola: q, Limites: cfg.Limites, Version: version.Version, Log: log})
	if err := srv.CargarAjustes(ctx); err != nil {
		return err
	}
	if err := srv.RegistrarImpresoras(ctx); err != nil {
		return err
	}
	// Aviso de versiones nuevas en el panel: una consulta diaria a GitHub, desactivable en Ajustes.
	aviso := &actualizacion.Aviso{}
	comprobador := &actualizacion.Comprobador{
		Cliente: actualizacion.NuevoCliente(version.Version), Actual: version.Version, Aviso: aviso,
		Activo: func(ctx context.Context) bool {
			v, _, _ := alm.Ajustes.Obtener(ctx, almacen.AjusteBuscarActualizaciones)
			return v != "0"
		},
		Inicial: time.Hour, Cada: 24 * time.Hour, Log: log,
	}
	go comprobador.Correr(ctx)
	panel := admin.Nuevo(admin.Opciones{Almacen: alm, Servidor: srv, Cola: q, Config: cfg, Version: version.Version, Log: log, Aviso: aviso})
	srv.MontarPanel(panel.Handler())
	errServidor := make(chan error, 1)
	go func() { errServidor <- srv.Escuchar(ctx, cfg, nil) }()

	cada := time.NewTicker(time.Hour)
	defer cada.Stop()
	for {
		select {
		case err := <-errServidor:
			if err != nil {
				return fmt.Errorf("el servidor se detuvo: %w", err)
			}
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
	solicitudes, err := alm.Solicitudes.Purgar(ctx)
	if err != nil {
		return err
	}
	if trabajos+ids+solicitudes > 0 {
		log.Info("mantenimiento de la base de datos", "trabajos_purgados", trabajos, "ids_purgados", ids,
			"solicitudes_purgadas", solicitudes, "retencion_dias", dias)
	}
	return nil
}
