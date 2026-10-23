package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/cola"
	"github.com/bayalad7/prisma/internal/conector"
	"github.com/bayalad7/prisma/internal/contrato"
	"github.com/bayalad7/prisma/internal/dev"
	"github.com/bayalad7/prisma/internal/escpos"
)

// clienteConsola identifica en el historial los trabajos enviados desde la CLI.
const clienteConsola = "cli_consola"

func nuevoComandoImprimir(op *opciones) *cobra.Command {
	var alias string
	cmd := &cobra.Command{
		Use:   "imprimir documento.json [--impresora alias]",
		Short: "Imprime un documento directamente (solo en modo desarrollo)",
		Long: "Valida, renderiza e imprime un documento en una impresora configurada, pasando por la\n" +
			"misma cola que usará el servidor. Solo funciona con [dev] modo_desarrollo = true: en un\n" +
			"negocio, las aplicaciones imprimen por la API con su autorización.",
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			alm, cfg, err := abrirAlmacen(c.Context(), op)
			if err != nil {
				return err
			}
			defer alm.Cerrar()
			if !cfg.Dev.ModoDesarrollo {
				return errors.New("«prisma imprimir» solo está disponible con [dev] modo_desarrollo = true en config.toml; " +
					"en producción las aplicaciones imprimen por la API")
			}
			return imprimir(c.Context(), c, alm, args[0], alias)
		},
	}
	cmd.Flags().StringVar(&alias, "impresora", "", "alias de la impresora (por defecto, el campo impresora del documento)")
	return cmd
}

func imprimir(ctx context.Context, c *cobra.Command, alm *almacen.Almacen, ruta, alias string) error {
	w := c.OutOrStdout()
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return err
	}
	doc, err := contrato.Decodificar(datos)
	if err != nil {
		return err
	}
	if alias == "" {
		alias = doc.Impresora
	}
	imp, err := obtenerImpresora(ctx, alm, alias)
	if err != nil {
		return err
	}
	if !imp.Activa {
		return fmt.Errorf("la impresora %q está desactivada", alias)
	}
	bytesTicket, advs, err := escpos.Renderizar(*doc, *imp, escpos.RecursosAlmacen{Ctx: ctx, Repo: alm.Recursos})
	if err != nil {
		return err
	}
	con, err := conector.Nuevo(*imp)
	if err != nil {
		return err
	}

	b := make([]byte, 6)
	rand.Read(b)
	t := &almacen.Trabajo{ID: "trb_" + hex.EncodeToString(b), DocumentoID: doc.ID, ClienteID: clienteConsola,
		ImpresoraAlias: alias, Virtual: imp.Tipo == almacen.ConexionVirtual, Referencia: doc.Referencia}
	if err := alm.Trabajos.Crear(ctx, t); err != nil {
		return err
	}

	q := cola.Nueva(cola.Opciones{AlIniciar: func(string, string) { alm.Trabajos.MarcarImprimiendo(ctx, t.ID) }})
	defer q.Cerrar()
	q.Registrar(alias, con)
	resultado, err := q.Encolar(alias, t.ID, bytesTicket)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Enviando %d bytes a %q (%s → %s)…\n", len(bytesTicket), alias, imp.Tipo, imp.Destino)
	r := <-resultado

	var codigos []string
	for _, a := range advs {
		codigos = append(codigos, a.Codigo)
		fmt.Fprintf(w, "Advertencia %s: %s\n", a.Codigo, a.Mensaje)
	}
	estado, codigoError := almacen.TrabajoCompletado, ""
	if !r.OK {
		estado, codigoError = almacen.TrabajoFallido, conector.Codigo(r.Err)
	}
	if err := alm.Trabajos.Finalizar(ctx, t.ID, estado, codigoError, codigos, r.Duracion); err != nil {
		return err
	}
	if !r.OK {
		return r.Err
	}
	fmt.Fprintf(w, "Impreso en %s (trabajo %s).\n", r.Duracion.Round(time.Millisecond), t.ID)
	// El rollo de una impresora virtual vive en la memoria de este proceso, que termina aquí: el
	// ticket se muestra en la terminal. Lo que llega por la API del puente se ve en el panel.
	if imp.Tipo == almacen.ConexionVirtual {
		fmt.Fprintf(w, "\n%q es una impresora virtual; así se vería el ticket:\n%s", alias, dev.VistaTexto(dev.Simular(bytesTicket, *imp), false))
	}
	return nil
}
