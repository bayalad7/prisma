package servidor

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/bayalad7/prisma/internal/config"
)

// Escuchar sirve la API según la configuración de arranque hasta que se cancela el contexto.
// listo, si no es nil, recibe la dirección real en la que quedó escuchando.
func (s *Servidor) Escuchar(ctx context.Context, cfg config.Config, listo func(direccion string)) error {
	srv := &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      s.op.TiempoRespuesta + 30*time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    32 << 10,
		ErrorLog:          slog.NewLogLogger(s.op.Log.Handler(), slog.LevelWarn),
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(cfg.Servidor.Escuchar, strconv.Itoa(cfg.Servidor.Puerto)))
	if err != nil {
		return err
	}

	esquema := "https"
	switch cfg.Servidor.ModoTLS {
	case config.TLSSoloLocalhostHTTP:
		// La configuración ya exige que escuchar sea 127.0.0.1, ::1 o localhost.
		esquema = "http"
	case config.TLSArchivos:
		prov, err := NuevoArchivos(cfg.Servidor.Certificado, cfg.Servidor.Llave)
		if err != nil {
			ln.Close()
			return err
		}
		if cert, err := LeerCertificado(prov.Certificado); err == nil {
			s.avisarVencimiento(cert)
		}
		ln = tls.NewListener(ln, ConfigTLS(prov))
	default: // autofirmado
		prov, err := CargarAutofirmado(cfg.DirDatos())
		if err != nil {
			ln.Close()
			return err
		}
		if cert, err := LeerCertificado(prov.Certificado); err == nil {
			s.op.Log.Info("certificado autofirmado", "archivo", prov.Certificado, "huella_sha256", Huella(cert),
				"vence", cert.NotAfter.Format(time.DateOnly))
			s.avisarVencimiento(cert)
			if faltan := IPsSinCubrir(cert); len(faltan) > 0 {
				s.op.Log.Warn("el certificado no incluye algunas IPs actuales de este equipo; regenéralo con «prisma tls regenerar»",
					"ips", faltan)
			}
		}
		ln = tls.NewListener(ln, ConfigTLS(prov))
	}

	s.op.Log.Info("API escuchando", "url", esquema+"://"+ln.Addr().String()+"/v1/salud")
	if listo != nil {
		listo(ln.Addr().String())
	}
	cerrado := make(chan struct{})
	go func() {
		defer close(cerrado)
		<-ctx.Done()
		fin, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelar()
		srv.Shutdown(fin)
	}()
	err = srv.Serve(ln)
	if errors.Is(err, http.ErrServerClosed) {
		<-cerrado
		return nil
	}
	return err
}

// avisarVencimiento deja en la bitácora si el certificado venció o está por vencer: cuando vence,
// ningún cliente puede imprimir.
func (s *Servidor) avisarVencimiento(cert *x509.Certificate) {
	switch estado, dias := Vencimiento(cert, time.Now()); estado {
	case CertificadoVencido:
		s.op.Log.Error("el certificado HTTPS venció: los clientes no pueden conectarse; regenéralo («prisma tls regenerar» o el panel) y vuelve a instalarlo en los equipos",
			"vencio", cert.NotAfter.Format(time.DateOnly))
	case CertificadoPorVencer:
		s.op.Log.Warn("el certificado HTTPS está por vencer; regenéralo antes y vuelve a instalarlo en los equipos",
			"vence", cert.NotAfter.Format(time.DateOnly), "dias", dias)
	}
}
