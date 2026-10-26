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
				return nil
			},
		},
		&cobra.Command{
			Use:   "regenerar",
			Short: "Genera un certificado autofirmado nuevo con el hostname y las IPs actuales",
			Long: "Úsalo si cambió la IP o el nombre del equipo. El puente en marcha toma el certificado nuevo\n" +
				"en unos segundos, sin reiniciar. Los equipos que confiaban en el anterior deben confiar en el nuevo.",
			Args: cobra.NoArgs,
			RunE: func(c *cobra.Command, _ []string) error {
				cfg, origen, err := config.Cargar(op.rutaConfig)
				if err != nil {
					return fmt.Errorf("%s: %w", origen.Ruta, err)
				}
				if cfg.Servidor.ModoTLS != config.TLSAutofirmado {
					return fmt.Errorf("modo_tls es %q: solo se regenera el certificado autofirmado", cfg.Servidor.ModoTLS)
				}
				if _, err := servidor.GenerarAutofirmado(cfg.DirDatos()); err != nil {
					return err
				}
				w := c.OutOrStdout()
				fmt.Fprintln(w, "Certificado autofirmado regenerado.")
				ruta, _ := servidor.RutasAutofirmado(cfg.DirDatos())
				if err := mostrarCertificado(w, ruta); err != nil {
					return err
				}
				fmt.Fprintln(w, "\nPara que este equipo confíe en él, ejecuta como administrador «prisma tls confiar»; para los")
				fmt.Fprintln(w, "demás equipos, sigue docs/instalacion/https-y-certificados.md.")
				return nil
			},
		},
		nuevoComandoConfiar(op),
	)
	return cmd
}

// nuevoComandoConfiar: «prisma tls confiar» agrega el certificado autofirmado al almacén de
// confianza de este equipo (lo usan los instaladores); con --quitar, lo quita (ADR 0020).
func nuevoComandoConfiar(op *opciones) *cobra.Command {
	var quitar bool
	var esperar time.Duration
	cmd := &cobra.Command{
		Use:   "confiar",
		Short: "Hace que este equipo confíe en el certificado del puente (requiere administrador)",
		Long: "Agrega el certificado autofirmado al almacén de confianza del sistema, para que el navegador de\n" +
			"este equipo abra el panel sin advertencia. Solo acepta certificados limitados a la red local\n" +
			"(los de PRISMA 0.3.0 o posterior). Con --quitar, quita los que PRISMA haya puesto.",
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
			ruta, _ := servidor.RutasAutofirmado(cfg.DirDatos())
			// El servicio genera el certificado al arrancar: el instalador espera a que exista.
			cert, err := servidor.LeerCertificado(ruta)
			for limite := time.Now().Add(esperar); err != nil && time.Now().Before(limite); {
				time.Sleep(500 * time.Millisecond)
				cert, err = servidor.LeerCertificado(ruta)
			}
			if err != nil {
				return fmt.Errorf("%w (se genera al primer arranque del puente o con «prisma tls regenerar»)", err)
			}
			if !servidor.LimitadoALaRedLocal(cert) {
				return fmt.Errorf("el certificado es de una versión anterior y no está limitado a la red local: " +
					"genera uno nuevo con «prisma tls regenerar» (después, los demás equipos deben volver a confiar en él)")
			}
			if err := servicio.ConfiarCertificado(ruta); err != nil {
				return fmt.Errorf("no se pudo agregar el certificado al almacén de confianza (¿como administrador?): %w", err)
			}
			fmt.Fprintf(w, "Este equipo confía en el certificado del puente (huella %s).\n", servidor.Huella(cert))
			return nil
		},
	}
	cmd.Flags().BoolVar(&quitar, "quitar", false, "quita del almacén los certificados que puso PRISMA")
	cmd.Flags().DurationVar(&esperar, "esperar", 0, "cuánto esperar a que el puente genere el certificado (por ejemplo 60s)")
	return cmd
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
	if cert.IsCA && cert.Subject.Organization != nil && cert.Subject.Organization[0] == servicio.OrganizacionCertificado {
		limitado := "sí (se puede confiar en él con «prisma tls confiar»)"
		if !servidor.LimitadoALaRedLocal(cert) {
			limitado = "no: es de una versión anterior; regenéralo para que este equipo pueda confiar en él"
		}
		fmt.Fprintf(w, "limitado a la red local: %s\n", limitado)
	}
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
