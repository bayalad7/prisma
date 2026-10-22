package main

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/bayalad7/prisma/internal/config"
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
				fmt.Fprintln(w, "\nPara que los navegadores confíen en él, sigue docs/instalacion/https-y-certificados.md.")
				return nil
			},
		},
	)
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
