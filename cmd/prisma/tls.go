package main

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/bayalad7/prisma/internal/config"
	"github.com/bayalad7/prisma/internal/servicio"
	"github.com/bayalad7/prisma/internal/servidor"
)

func nuevoComandoTLS(op *opciones) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tls",
		Short: "Consulta o regenera el certificado HTTPS del puente",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "info",
			Short: "Muestra el certificado en uso: archivo, nombres, IPs, vencimiento y huella",
			Args:  cobra.NoArgs,
			RunE: func(c *cobra.Command, _ []string) error {
				cfg, origen, err := config.Cargar(op.rutaConfig)
				if err != nil {
					return fmt.Errorf("%s: %w", origen.Ruta, err)
				}
				w := c.OutOrStdout()
				switch cfg.Servidor.ModoTLS {
				case config.TLSSoloLocalhostHTTP:
					fmt.Fprintln(w, "modo_tls = solo_localhost_http: el puente sirve HTTP sin certificado, solo en este equipo.")
					return nil
				case config.TLSArchivos:
					return mostrarCertificado(w, cfg.Servidor.Certificado)
				}
				ruta, _ := servidor.RutasAutofirmado(cfg.DirDatos())
				if err := mostrarCertificado(w, ruta); err != nil {
					return fmt.Errorf("%w (se genera al primer arranque o con «prisma tls regenerar»)", err)
				}
				mostrarAutoridad(w, cfg.DirDatos())
				return nil
			},
		},
		nuevoComandoRegenerar(op),
		nuevoComandoConfiar(op),
	)
	return cmd
}

// nuevoComandoRegenerar: «prisma tls regenerar» renueva el certificado del servidor con la autoridad
// local que ya hay (los equipos que confían en ella no tienen que hacer nada); crea una autoridad
// nueva si no hay, si ya no cubre los nombres del equipo o con --nueva-autoridad (ADR 0021).
func nuevoComandoRegenerar(op *opciones) *cobra.Command {
	var nuevaAutoridad bool
	cmd := &cobra.Command{
		Use:   "regenerar",
		Short: "Genera un certificado del servidor nuevo con el hostname y las IPs actuales",
		Long: "Úsalo si cambió la IP o el nombre del equipo. El puente en marcha toma el certificado nuevo en\n" +
			"unos segundos, sin reiniciar. Lo firma la misma autoridad local, así que los equipos que ya\n" +
			"confían en ella no tienen que hacer nada; si hace falta una autoridad nueva (no había, cambió el\n" +
			"nombre del equipo o --nueva-autoridad), los equipos deben volver a confiar en ella.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			cfg, origen, err := config.Cargar(op.rutaConfig)
			if err != nil {
				return fmt.Errorf("%s: %w", origen.Ruta, err)
			}
			if cfg.Servidor.ModoTLS != config.TLSAutofirmado {
				return fmt.Errorf("modo_tls es %q: solo se regenera el certificado autofirmado", cfg.Servidor.ModoTLS)
			}
			gen, err := servidor.GenerarAutofirmado(cfg.DirDatos(), nuevaAutoridad)
			if err != nil {
				return err
			}
			w := c.OutOrStdout()
			fmt.Fprintln(w, "Certificado del servidor regenerado.")
			ruta, _ := servidor.RutasAutofirmado(cfg.DirDatos())
			if err := mostrarCertificado(w, ruta); err != nil {
				return err
			}
			mostrarAutoridad(w, cfg.DirDatos())
			if gen.AutoridadNueva {
				fmt.Fprintln(w, "\nSe creó una autoridad local nueva. Para que este equipo confíe en ella, ejecuta como")
				fmt.Fprintln(w, "administrador «prisma tls confiar»; en los demás equipos, vuelve a instalarla como explica")
				fmt.Fprintln(w, "docs/instalacion/https-y-certificados.md.")
			} else {
				fmt.Fprintln(w, "\nLo firmó la misma autoridad local: los equipos que ya confían en ella no tienen que hacer nada.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&nuevaAutoridad, "nueva-autoridad", false, "crea también una autoridad local nueva (hay que volver a confiar en ella en cada equipo)")
	return cmd
}

// nuevoComandoConfiar: «prisma tls confiar» agrega la autoridad local al almacén de confianza de
// este equipo (lo usan los instaladores); con --quitar, la quita (ADR 0020 y 0021).
func nuevoComandoConfiar(op *opciones) *cobra.Command {
	var quitar bool
	var esperar time.Duration
	cmd := &cobra.Command{
		Use:   "confiar",
		Short: "Hace que este equipo confíe en el certificado del puente (requiere administrador)",
		Long: "Agrega la autoridad local del puente al almacén de confianza del sistema, para que el navegador\n" +
			"de este equipo abra el panel sin advertencia. La autoridad está limitada a la red local (desde\n" +
			"PRISMA 0.3.1). Con --quitar, quita los certificados que PRISMA haya puesto.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			w := c.OutOrStdout()
			if quitar {
				if err := servicio.QuitarConfianza(); err != nil {
					return fmt.Errorf("no se pudo quitar el certificado del almacén de confianza (¿como administrador?): %w", err)
				}
				fmt.Fprintln(w, "Este equipo ya no confía en el certificado del puente.")
				return nil
			}
			cfg, origen, err := config.Cargar(op.rutaConfig)
			if err != nil {
				return fmt.Errorf("%s: %w", origen.Ruta, err)
			}
			if cfg.Servidor.ModoTLS != config.TLSAutofirmado {
				return fmt.Errorf("modo_tls es %q: solo se confía así en el certificado autofirmado", cfg.Servidor.ModoTLS)
			}
			// El servicio genera el certificado al arrancar: el instalador espera a que exista.
			rutaServidor, _ := servidor.RutasAutofirmado(cfg.DirDatos())
			_, err = servidor.LeerCertificado(rutaServidor)
			for limite := time.Now().Add(esperar); err != nil && time.Now().Before(limite); {
				time.Sleep(500 * time.Millisecond)
				_, err = servidor.LeerCertificado(rutaServidor)
			}
			if err != nil {
				return fmt.Errorf("%w (se genera al primer arranque del puente o con «prisma tls regenerar»)", err)
			}
			ca, err := servidor.AutoridadLocal(cfg.DirDatos())
			if err != nil {
				return fmt.Errorf("este puente todavía no tiene autoridad local: su certificado es de una versión anterior. " +
					"Genera uno nuevo con «prisma tls regenerar» (después, los demás equipos deben volver a confiar en él)")
			}
			ruta, _ := servidor.RutasAutoridad(cfg.DirDatos())
			if err := servicio.ConfiarCertificado(ruta); err != nil {
				return fmt.Errorf("no se pudo agregar la autoridad al almacén de confianza (¿como administrador?): %w", err)
			}
			fmt.Fprintf(w, "Este equipo confía en la autoridad local del puente (huella %s).\n", servidor.Huella(ca))
			return nil
		},
	}
	cmd.Flags().BoolVar(&quitar, "quitar", false, "quita del almacén los certificados que puso PRISMA")
	cmd.Flags().DurationVar(&esperar, "esperar", 0, "cuánto esperar a que el puente genere el certificado (por ejemplo 60s)")
	return cmd
}

// mostrarAutoridad dice qué autoridad local firma el certificado (la que se instala en los equipos),
// o que el certificado es de antes de 0.3.1 y no tiene.
func mostrarAutoridad(w io.Writer, dirDatos string) {
	ca, err := servidor.AutoridadLocal(dirDatos)
	if err != nil {
		fmt.Fprintln(w, "\nautoridad local: no tiene (certificado de una versión anterior); regenéralo con «prisma tls regenerar»")
		fmt.Fprintln(w, "  para que este equipo pueda confiar en él con «prisma tls confiar».")
		return
	}
	ruta, _ := servidor.RutasAutoridad(dirDatos)
	fmt.Fprintln(w, "\nautoridad local (la que se instala en los equipos):")
	fmt.Fprintf(w, "archivo:      %s\n", ruta)
	fmt.Fprintf(w, "válida hasta: %s\n", ca.NotAfter.Local().Format(time.DateOnly))
	fmt.Fprintf(w, "huella SHA-256:\n  %s\n", servidor.Huella(ca))
	fmt.Fprintln(w, "limitada a la red local: sí")
}

func mostrarCertificado(w io.Writer, ruta string) error {
	cert, err := servidor.LeerCertificado(ruta)
	if err != nil {
		return err
	}
	ips := make([]string, len(cert.IPAddresses))
	for i, ip := range cert.IPAddresses {
		ips[i] = ip.String()
	}
	fmt.Fprintf(w, "archivo:      %s\n", ruta)
	fmt.Fprintf(w, "sujeto:       %s\n", cert.Subject.CommonName)
	fmt.Fprintf(w, "nombres:      %s\n", strings.Join(cert.DNSNames, ", "))
	fmt.Fprintf(w, "IPs:          %s\n", strings.Join(ips, ", "))
	fmt.Fprintf(w, "válido hasta: %s\n", cert.NotAfter.Local().Format(time.DateOnly))
	fmt.Fprintf(w, "huella SHA-256:\n  %s\n", servidor.Huella(cert))
	if faltan := servidor.IPsSinCubrir(cert); len(faltan) > 0 {
		var f []string
		for _, ip := range faltan {
			if !ip.Equal(net.IPv6loopback) {
				f = append(f, ip.String())
			}
		}
		if len(f) > 0 {
			fmt.Fprintf(w, "\nAtención: no incluye las IPs actuales %s; regenéralo con «prisma tls regenerar».\n", strings.Join(f, ", "))
		}
	}
	return nil
}
