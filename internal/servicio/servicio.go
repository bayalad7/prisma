// Package servicio instala PRISMA como servicio del sistema (kardianos/service): servicio de
// Windows con inicio automático y reinicio ante fallos, unidad de systemd con un usuario propio en
// Linux y LaunchDaemon en macOS.
package servicio

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"time"

	"github.com/kardianos/service"
)

// Opciones de instalación.
type Opciones struct {
	// RutaConfig es la config.toml que usará el servicio; vacía = la ruta del sistema.
	RutaConfig string
	// Datos y Logs son los directorios de datos y bitácoras: el instalador los crea con permisos
	// para el usuario del servicio.
	Datos string
	Logs  string
	// Puerto es el puerto HTTPS del puente (para la regla del firewall de Windows).
	Puerto int
}

// Nombre del servicio en el sistema actual.
func Nombre() string {
	switch runtime.GOOS {
	case "windows":
		return NombreWindows
	case "darwin":
		return NombreMacOS
	}
	return NombreLinux
}

func configuracion(op Opciones) (*service.Config, error) {
	exe, err := ejecutable()
	if err != nil {
		return nil, err
	}
	c := Comando{Ejecutable: exe, Argumentos: Argumentos(op.RutaConfig)}
	cfg := &service.Config{
		Name:        Nombre(),
		DisplayName: NombreVisible,
		Description: Descripcion,
		Executable:  c.Ejecutable,
		Arguments:   c.Argumentos,
		Option:      service.KeyValue{},
	}
	switch runtime.GOOS {
	case "windows":
		// Inicio automático; si el proceso falla, Windows lo reinicia a los 5 s (y el contador de
		// fallos vuelve a cero cada día).
		cfg.Option["StartType"] = "automatic"
		cfg.Option["OnFailure"] = "restart"
		cfg.Option["OnFailureDelayDuration"] = "5s"
		cfg.Option["OnFailureResetPeriod"] = 86400
	case "darwin":
		cfg.UserName = UsuarioMacOS
		cfg.Option["LaunchdConfig"] = PlistLaunchd(c, op.Logs)
	default:
		cfg.UserName = UsuarioLinux
		cfg.Option["SystemdScript"] = UnidadSystemd(c)
	}
	return cfg, nil
}

// ejecutable es la ruta absoluta y sin enlaces simbólicos del binario actual.
func ejecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	return filepath.Abs(exe)
}

func nuevo(op Opciones, p service.Interface) (service.Service, error) {
	cfg, err := configuracion(op)
	if err != nil {
		return nil, err
	}
	return service.New(p, cfg)
}

// sinPrograma permite controlar el servicio (instalar, iniciar…) sin ejecutarlo.
type sinPrograma struct{}

func (sinPrograma) Start(service.Service) error { return nil }
func (sinPrograma) Stop(service.Service) error  { return nil }

// Estado del servicio.
type Estado struct {
	Instalado bool
	EnMarcha  bool
}

func (e Estado) String() string {
	switch {
	case !e.Instalado:
		return "no instalado"
	case e.EnMarcha:
		return "en marcha"
	}
	return "detenido"
}

// ConsultarEstado no requiere permisos de administrador.
func ConsultarEstado(op Opciones) (Estado, error) {
	s, err := nuevo(op, sinPrograma{})
	if err != nil {
		return Estado{}, err
	}
	st, err := s.Status()
	if errors.Is(err, service.ErrNotInstalled) {
		return Estado{}, nil
	}
	if err != nil {
		return Estado{}, err
	}
	return estadoSistema(Estado{Instalado: true, EnMarcha: st == service.StatusRunning}), nil
}

var reEstadoLaunchd = regexp.MustCompile(`(?m)^\s*state = running\s*$`)

// launchdEnMarcha interpreta la salida de «launchctl print system/<label>».
func launchdEnMarcha(salida string) bool { return reEstadoLaunchd.MatchString(salida) }

// Instalar prepara el sistema (usuario, carpetas, firewall) y registra el servicio. Si ya estaba
// instalado, lo vuelve a registrar con la configuración actual (por ejemplo, tras mover el binario).
func Instalar(op Opciones) error {
	if err := prepararSistema(op); err != nil {
		return err
	}
	s, err := nuevo(op, sinPrograma{})
	if err != nil {
		return err
	}
	if st, err := s.Status(); err == nil {
		if st == service.StatusRunning {
			if err := s.Stop(); err != nil {
				return traducir("detener", err)
			}
		}
		if err := s.Uninstall(); err != nil {
			return traducir("reinstalar", err)
		}
	}
	return traducir("instalar", s.Install())
}

// Desinstalar detiene el servicio y lo quita del sistema. Conserva datos, configuración y bitácoras.
func Desinstalar(op Opciones) error {
	s, err := nuevo(op, sinPrograma{})
	if err != nil {
		return err
	}
	if st, err := s.Status(); err == nil && st == service.StatusRunning {
		if err := s.Stop(); err != nil {
			return traducir("detener", err)
		}
	}
	if err := traducir("desinstalar", s.Uninstall()); err != nil {
		return err
	}
	limpiarSistema(op)
	return nil
}

// Controlar ejecuta iniciar, detener o reiniciar.
func Controlar(op Opciones, accion string) error {
	s, err := nuevo(op, sinPrograma{})
	if err != nil {
		return err
	}
	if _, err := s.Status(); errors.Is(err, service.ErrNotInstalled) {
		return errors.New("el servicio no está instalado; instálalo con «prisma servicio instalar»")
	}
	return traducir(accion, service.Control(s, map[string]string{"iniciar": "start", "detener": "stop", "reiniciar": "restart"}[accion]))
}

func traducir(accion string, err error) error {
	if err == nil {
		return nil
	}
	if os.IsPermission(err) || errors.Is(err, os.ErrPermission) || esAccesoDenegado(err) {
		return fmt.Errorf("no se pudo %s el servicio: %w\n%s", accion, err, sugerenciaPermisos())
	}
	return fmt.Errorf("no se pudo %s el servicio: %w", accion, err)
}

func sugerenciaPermisos() string {
	if runtime.GOOS == "windows" {
		return "Abre PowerShell como administrador (clic derecho → «Ejecutar como administrador») y vuelve a intentarlo."
	}
	return "Ejecuta el comando con sudo."
}

// Interactivo indica si el proceso corre desde una terminal y no bajo el gestor de servicios.
func Interactivo() bool { return service.Interactive() }

// Ejecutar corre el puente bajo el gestor de servicios: llama a correr con un contexto que se
// cancela cuando el sistema pide detener el servicio. Si correr termina con error sin que se lo
// hayan pedido, el proceso sale con código 1 para que el sistema lo reinicie.
func Ejecutar(op Opciones, correr func(context.Context) error, alFallar func(error)) error {
	p := &programa{correr: correr, alFallar: alFallar}
	s, err := nuevo(op, p)
	if err != nil {
		return err
	}
	return s.Run()
}

type programa struct {
	correr   func(context.Context) error
	alFallar func(error)
	cancelar context.CancelFunc
	hecho    chan error
}

func (p *programa) Start(service.Service) error {
	ctx, cancelar := context.WithCancel(context.Background())
	p.cancelar, p.hecho = cancelar, make(chan error, 1)
	go func() {
		err := p.correr(ctx)
		if ctx.Err() == nil { // terminó solo, sin que el sistema lo pidiera
			if err == nil {
				err = errors.New("el puente se detuvo inesperadamente")
			}
			p.alFallar(err)
			os.Exit(1)
		}
		p.hecho <- err
	}()
	return nil
}

func (p *programa) Stop(service.Service) error {
	p.cancelar()
	select {
	case err := <-p.hecho:
		return err
	case <-time.After(20 * time.Second):
		return errors.New("el puente no se detuvo en 20 s")
	}
}
