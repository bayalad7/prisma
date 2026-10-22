package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/bayalad7/prisma/internal/actualizacion"
	"github.com/bayalad7/prisma/internal/servicio"
	"github.com/bayalad7/prisma/internal/version"
)

func nuevoComandoActualizar(op *opciones) *cobra.Command {
	var soloComprobar, si, forzar bool
	cmd := &cobra.Command{
		Use:   "actualizar",
		Short: "Instala la última versión de PRISMA",
		Long: "Busca la última versión en GitHub Releases, verifica la firma de sus checksums y el SHA-256\n" +
			"del archivo, reemplaza el programa y reinicia el servicio. La base de datos se respalda sola\n" +
			"antes de migrarla al siguiente arranque. Requiere permisos de administrador.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			ctx, cancelar := context.WithTimeout(c.Context(), 10*time.Minute)
			defer cancelar()
			w := c.OutOrStdout()
			cliente := actualizacion.NuevoCliente(version.Version)
			v, err := cliente.Ultima(ctx)
			if err != nil {
				return err
			}
			nueva := actualizacion.EsNueva(version.Version, v.Numero)
			fmt.Fprintf(w, "Versión instalada: %s\nÚltima versión:    %s (%s)\n", version.Version, v.Numero, v.URL)
			switch {
			case soloComprobar:
				if nueva {
					fmt.Fprintln(w, "Hay una versión nueva. Instálala con «prisma actualizar».")
				} else {
					fmt.Fprintln(w, "PRISMA está al día.")
				}
				return nil
			case actualizacion.Desarrollo(version.Version) && !forzar:
				return errors.New("esta es una compilación de desarrollo; usa --forzar para reemplazarla por la versión publicada")
			case !nueva && !forzar:
				fmt.Fprintln(w, "PRISMA está al día.")
				return nil
			}
			if !si && !confirmar(c.InOrStdin(), w, fmt.Sprintf("¿Instalar la versión %s? [s/N] ", v.Numero)) {
				return errors.New("actualización cancelada")
			}
			sumas, err := cliente.Checksums(ctx, v)
			if err != nil {
				return err
			}
			fmt.Fprintln(w, "Firma de checksums.txt verificada.")
			exe, err := os.Executable()
			if err != nil {
				return err
			}
			if real, err := filepath.EvalSymlinks(exe); err == nil {
				exe = real
			}
			if actualizacion.InstaladoConPaquete(exe) {
				return actualizarDeb(ctx, w, cliente, v, sumas)
			}
			return actualizarBinario(ctx, w, op, cliente, v, sumas, exe)
		},
	}
	f := cmd.Flags()
	f.BoolVar(&soloComprobar, "comprobar", false, "solo dice si hay una versión nueva, sin instalar nada")
	f.BoolVarP(&si, "si", "y", false, "no pide confirmación")
	f.BoolVar(&forzar, "forzar", false, "reinstala aunque no haya versión nueva (o reemplaza una compilación de desarrollo)")
	return cmd
}

func confirmar(r io.Reader, w io.Writer, pregunta string) bool {
	fmt.Fprint(w, pregunta)
	linea, _ := bufio.NewReader(r).ReadString('\n')
	return strings.EqualFold(strings.TrimSpace(linea), "s") || strings.EqualFold(strings.TrimSpace(linea), "si") ||
		strings.EqualFold(strings.TrimSpace(linea), "sí")
}

// actualizarBinario reemplaza el ejecutable y reinicia el servicio si estaba en marcha.
func actualizarBinario(ctx context.Context, w io.Writer, op *opciones, cliente *actualizacion.Cliente, v *actualizacion.Version, sumas map[string][]byte, exe string) error {
	dir := filepath.Dir(exe)
	if err := actualizacion.Escribible(dir); err != nil {
		if errors.Is(err, actualizacion.ErrSinPermiso) {
			if runtime.GOOS == "windows" {
				return fmt.Errorf("%w en %s: abre PowerShell como administrador y vuelve a intentarlo", err, dir)
			}
			return fmt.Errorf("%w en %s: ejecuta «sudo prisma actualizar»", err, dir)
		}
		return err
	}
	nombre := actualizacion.ArchivoBinario(runtime.GOOS, runtime.GOARCH)
	nuevo := filepath.Join(dir, ".prisma-"+v.Numero+".descarga")
	fmt.Fprintf(w, "Descargando %s…\n", nombre)
	if err := cliente.Descargar(ctx, v, nombre, sumas, nuevo); err != nil {
		return err
	}
	defer os.Remove(nuevo)
	fmt.Fprintln(w, "SHA-256 verificado.")

	so, err := opcionesServicio(op)
	if err != nil {
		return err
	}
	estado, _ := servicio.ConsultarEstado(so)
	if estado.EnMarcha {
		fmt.Fprintln(w, "Deteniendo el servicio…")
		if err := servicio.Controlar(so, "detener"); err != nil {
			return err
		}
	}
	if err := actualizacion.Reemplazar(nuevo, exe); err != nil {
		if estado.EnMarcha {
			servicio.Controlar(so, "iniciar")
		}
		return fmt.Errorf("no se pudo reemplazar %s: %w", exe, err)
	}
	fmt.Fprintf(w, "PRISMA %s instalado en %s.\n", v.Numero, exe)
	if estado.EnMarcha {
		if err := servicio.Controlar(so, "iniciar"); err != nil {
			return err
		}
		fmt.Fprintln(w, "Servicio iniciado. Antes de migrar la base de datos, el puente guarda una copia en datos/respaldos.")
	}
	return nil
}

// actualizarDeb instala el .deb de la versión con dpkg; su script reinicia el servicio.
func actualizarDeb(ctx context.Context, w io.Writer, cliente *actualizacion.Cliente, v *actualizacion.Version, sumas map[string][]byte) error {
	if os.Geteuid() != 0 {
		return errors.New("PRISMA se instaló con el paquete .deb: ejecuta «sudo prisma actualizar»")
	}
	dir, err := os.MkdirTemp("", "prisma-actualizar-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	nombre := actualizacion.ArchivoDeb(v.Numero, runtime.GOARCH)
	deb := filepath.Join(dir, nombre)
	fmt.Fprintf(w, "Descargando %s…\n", nombre)
	if err := cliente.Descargar(ctx, v, nombre, sumas, deb); err != nil {
		return err
	}
	fmt.Fprintln(w, "SHA-256 verificado. Instalando con dpkg…")
	cmd := exec.CommandContext(ctx, "dpkg", "-i", deb)
	cmd.Stdout, cmd.Stderr = w, w
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("dpkg no pudo instalar %s: %w", nombre, err)
	}
	fmt.Fprintf(w, "PRISMA %s instalado.\n", v.Numero)
	return nil
}
