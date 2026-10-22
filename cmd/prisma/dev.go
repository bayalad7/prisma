package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/bayalad7/prisma/internal/contrato"
	"github.com/bayalad7/prisma/internal/dev"
	"github.com/bayalad7/prisma/internal/escpos"
	"github.com/bayalad7/prisma/internal/seguridad"
)

func nuevoComandoDev() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Herramientas de desarrollo (decodificar y previsualizar tickets)",
	}
	cmd.AddCommand(nuevoComandoDecodificar(), nuevoComandoPrevisualizar(), nuevoComandoRenderizar(), nuevoComandoImpresoraVirtual(),
		nuevoComandoLlaves(), nuevoComandoFirmar())
	return cmd
}

func nuevoComandoLlaves() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "llaves --salida carpeta",
		Short: "Genera un par de llaves Ed25519 de prueba para un cliente de nivel 3",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			publica, privada, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				return err
			}
			pPEM, err := seguridad.LlavePrivadaPEM(privada)
			if err != nil {
				return err
			}
			uPEM, err := seguridad.LlavePublicaPEM(publica)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return err
			}
			rutaPrivada, rutaPublica := filepath.Join(dir, "privada.pem"), filepath.Join(dir, "publica.pem")
			for _, a := range []string{rutaPrivada, rutaPublica} {
				if _, err := os.Stat(a); err == nil {
					return fmt.Errorf("%s ya existe; elige otra carpeta para no sobrescribir llaves", a)
				}
			}
			if err := os.WriteFile(rutaPrivada, pPEM, 0o600); err != nil {
				return err
			}
			if err := os.WriteFile(rutaPublica, uPEM, 0o644); err != nil {
				return err
			}
			w := c.OutOrStdout()
			fmt.Fprintf(w, "Llave privada: %s (guárdala solo en el backend que firma)\n", rutaPrivada)
			fmt.Fprintf(w, "Llave pública: %s\n", rutaPublica)
			fmt.Fprintf(w, "Pública en base64: %s\n\n", base64.StdEncoding.EncodeToString(publica))
			fmt.Fprintf(w, "Registra el cliente con: prisma clientes agregar --nombre \"Backend\" --nivel 3 --llave-publica %s\n", rutaPublica)
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "salida", ".", "carpeta donde guardar privada.pem y publica.pem")
	return cmd
}

func nuevoComandoFirmar() *cobra.Command {
	var llave, cliente, salida string
	var nuevoID bool
	var vigencia time.Duration
	cmd := &cobra.Command{
		Use:   "firmar documento.json --llave privada.pem --cliente cli_xxx",
		Short: "Firma un documento como lo haría el backend de un cliente de nivel 3",
		Long: "Agrega cliente_id, emitido (ahora), expira y firma (Ed25519 sobre la forma canónica JCS).\n" +
			"El resultado se puede enviar con curl:\n" +
			"  prisma dev firmar doc.json --llave privada.pem --cliente cli_xxx --nuevo-id -o firmado.json\n" +
			"  curl -k https://localhost:8443/v1/imprimir -H \"Content-Type: application/json\" --data-binary @firmado.json",
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			doc, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			pem, err := os.ReadFile(llave)
			if err != nil {
				return err
			}
			privada, err := seguridad.LeerLlavePrivada(pem)
			if err != nil {
				return fmt.Errorf("%s: %w", llave, err)
			}
			if vigencia <= 0 || vigencia > contrato.VigenciaMaxima {
				return fmt.Errorf("--vigencia debe estar entre 1s y %s", contrato.VigenciaMaxima)
			}
			op := seguridad.OpcionesFirma{ClienteID: cliente, Vigencia: vigencia}
			if nuevoID {
				op.NuevoID = uuidV4()
			}
			firmado, err := seguridad.FirmarDocumento(doc, privada, op)
			if err != nil {
				return err
			}
			if salida == "" {
				_, err = c.OutOrStdout().Write(firmado)
				return err
			}
			if err := os.WriteFile(salida, firmado, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(c.ErrOrStderr(), "Documento firmado en %s (vence en %s).\n", salida, vigencia)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&llave, "llave", "", "llave privada Ed25519 en PEM (obligatoria)")
	f.StringVar(&cliente, "cliente", "", "id del cliente de nivel 3, cli_… (obligatorio)")
	f.BoolVar(&nuevoID, "nuevo-id", false, "reemplaza el id del documento por un UUID v4 nuevo (para no repetir)")
	f.DurationVar(&vigencia, "vigencia", 90*time.Second, "tiempo hasta expira (máximo 120s)")
	f.StringVarP(&salida, "salida", "o", "", "archivo de salida (por defecto, la salida estándar)")
	cmd.MarkFlagRequired("llave")
	cmd.MarkFlagRequired("cliente")
	return cmd
}

func uuidV4() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func nuevoComandoImpresoraVirtual() *cobra.Command {
	var puerto, ancho int
	var escuchar, directorio string
	var sinTraza, sinColor bool
	cmd := &cobra.Command{
		Use:   "impresora-virtual [--puerto 9100] [--ancho 80]",
		Short: "Simula una impresora de red: guarda cada trabajo y muestra su traza y previsualización",
		Long: "Escucha por TCP como una impresora de red. Configúrala en el puente con\n" +
			"  prisma impresoras agregar --alias virtual --tipo red --destino 127.0.0.1:9100\n" +
			"para desarrollar sin hardware. Ctrl+C la detiene.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			perfil, err := dev.PerfilPrueba(ancho)
			if err != nil {
				return err
			}
			if directorio == "" {
				directorio = filepath.Join(os.TempDir(), "prisma-impresora-virtual")
			}
			ln, err := net.Listen("tcp", net.JoinHostPort(escuchar, strconv.Itoa(puerto)))
			if err != nil {
				return fmt.Errorf("no se pudo escuchar en %s:%d (¿otro programa usa el puerto?): %w", escuchar, puerto, err)
			}
			w := c.OutOrStdout()
			fmt.Fprintf(w, "Impresora virtual de %d mm escuchando en %s. Trabajos en %s. Ctrl+C para salir.\n",
				ancho, ln.Addr(), directorio)
			ctx, detener := signal.NotifyContext(c.Context(), os.Interrupt, syscall.SIGTERM)
			defer detener()
			v := &dev.ImpresoraVirtual{Perfil: perfil, Directorio: directorio, Salida: w, SinTraza: sinTraza,
				Color: !sinColor && os.Getenv("NO_COLOR") == ""}
			return v.Servir(ctx, ln)
		},
	}
	f := cmd.Flags()
	f.IntVar(&puerto, "puerto", 9100, "puerto TCP")
	f.StringVar(&escuchar, "escuchar", "127.0.0.1", "dirección en la que escucha (0.0.0.0 para recibir desde otros equipos)")
	f.IntVar(&ancho, "ancho", 80, "ancho del papel en mm (58 u 80)")
	f.StringVar(&directorio, "salida", "", "carpeta donde guardar los .bin (por defecto, una carpeta temporal)")
	f.BoolVar(&sinTraza, "sin-traza", false, "muestra solo la previsualización")
	f.BoolVar(&sinColor, "sin-color", false, "no usa secuencias de color ANSI")
	return cmd
}

func nuevoComandoDecodificar() *cobra.Command {
	return &cobra.Command{
		Use:   "decodificar archivo.bin",
		Short: "Muestra una traza legible de los comandos ESC/POS de un archivo",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			datos, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			fmt.Fprint(c.OutOrStdout(), dev.Traza(datos))
			return nil
		},
	}
}

// opcionesTicket son las opciones comunes para renderizar un documento con un perfil de prueba.
type opcionesTicket struct {
	ancho    int
	recursos string
}

func (o *opcionesTicket) registrar(cmd *cobra.Command) {
	cmd.Flags().IntVar(&o.ancho, "ancho", 80, "ancho del papel en mm (58 u 80)")
	cmd.Flags().StringVar(&o.recursos, "recursos", "", "carpeta con las imágenes guardadas (<nombre>.png) para la operación imagen con recurso")
}

func (o *opcionesTicket) cargar(ruta string) (*contrato.Documento, escpos.Recursos, error) {
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return nil, nil, err
	}
	doc, err := contrato.Decodificar(datos)
	if err != nil {
		return nil, nil, err
	}
	var recursos escpos.Recursos = escpos.RecursosMapa{}
	if o.recursos != "" {
		recursos = escpos.RecursosDirectorio(o.recursos)
	}
	return doc, recursos, nil
}

func imprimirAdvertencias(w io.Writer, advs []escpos.Advertencia) {
	for _, a := range advs {
		fmt.Fprintf(w, "Advertencia %s: %s\n", a.Codigo, a.Mensaje)
	}
}

func nuevoComandoPrevisualizar() *cobra.Command {
	var o opcionesTicket
	var salidaHTML string
	var sinColor bool
	cmd := &cobra.Command{
		Use:   "previsualizar documento.json",
		Short: "Simula en la terminal (o en HTML) cómo se imprimirá un documento",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			doc, recursos, err := o.cargar(args[0])
			if err != nil {
				return err
			}
			perfil, err := dev.PerfilPrueba(o.ancho)
			if err != nil {
				return err
			}
			w := c.OutOrStdout()
			if salidaHTML != "" {
				fragmento, advs, err := dev.Previsualizar(*doc, perfil, recursos)
				if err != nil {
					return err
				}
				titulo := fmt.Sprintf("%s — %d mm", filepath.Base(args[0]), o.ancho)
				if err := os.WriteFile(salidaHTML, []byte(dev.DocumentoHTML(fragmento, titulo)), 0o644); err != nil {
					return err
				}
				fmt.Fprintf(w, "Previsualización guardada en %s\n", salidaHTML)
				imprimirAdvertencias(w, advs)
				return nil
			}
			datos, advs, err := escpos.Renderizar(*doc, perfil, recursos)
			if err != nil {
				return err
			}
			color := !sinColor && os.Getenv("NO_COLOR") == ""
			fmt.Fprint(w, dev.VistaTexto(dev.Simular(datos, perfil), color))
			imprimirAdvertencias(w, advs)
			return nil
		},
	}
	o.registrar(cmd)
	cmd.Flags().StringVar(&salidaHTML, "html", "", "guarda la previsualización como página HTML en este archivo")
	cmd.Flags().BoolVar(&sinColor, "sin-color", false, "no usa secuencias de color ANSI")
	return cmd
}

func nuevoComandoRenderizar() *cobra.Command {
	var o opcionesTicket
	var salida string
	cmd := &cobra.Command{
		Use:   "renderizar documento.json -o salida.bin",
		Short: "Genera los bytes ESC/POS de un documento (para decodificarlos o enviarlos a mano)",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if salida == "" {
				return errors.New("indica el archivo de salida con -o")
			}
			doc, recursos, err := o.cargar(args[0])
			if err != nil {
				return err
			}
			perfil, err := dev.PerfilPrueba(o.ancho)
			if err != nil {
				return err
			}
			datos, advs, err := escpos.Renderizar(*doc, perfil, recursos)
			if err != nil {
				return err
			}
			if err := os.WriteFile(salida, datos, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(c.OutOrStdout(), "%d bytes escritos en %s\n", len(datos), salida)
			imprimirAdvertencias(c.OutOrStdout(), advs)
			return nil
		},
	}
	o.registrar(cmd)
	cmd.Flags().StringVarP(&salida, "salida", "o", "", "archivo de salida")
	return cmd
}
