package servicio

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"golang.org/x/sys/windows"
)

// ReglaFirewall es el nombre de la regla de entrada que abre el puerto del puente.
const ReglaFirewall = "PRISMA"

func prepararSistema(op Opciones) error {
	if !windows.GetCurrentProcessToken().IsElevated() {
		return errors.New("instalar el servicio requiere permisos de administrador.\n" + sugerenciaPermisos())
	}
	for _, dir := range []string{op.Datos, op.Logs} {
		if dir != "" {
			if err := os.MkdirAll(dir, 0o750); err != nil {
				return err
			}
		}
	}
	// ProgramData deja leer a todos los usuarios. Los datos (hashes de llaves, llave privada del
	// certificado) y las bitácoras (IPs, aplicaciones) quedan solo para SYSTEM (S-1-5-18) y
	// Administradores (S-1-5-32-544).
	for _, dir := range []string{op.Datos, op.Logs} {
		if dir == "" {
			continue
		}
		if out, err := exec.Command("icacls", dir, "/inheritance:r", "/grant:r",
			"*S-1-5-18:(OI)(CI)F", "*S-1-5-32-544:(OI)(CI)F").CombinedOutput(); err != nil {
			return fmt.Errorf("no se pudieron restringir los permisos de %s: %v: %s", dir, err, out)
		}
	}
	if op.Puerto > 0 {
		exe, err := ejecutable()
		if err != nil {
			return err
		}
		// Solo desde la subred local, en cualquier perfil: Windows suele marcar como «pública» la
		// red de un negocio, y una regla solo para redes privadas dejaría fuera a las cajas.
		exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", "name="+ReglaFirewall).Run()
		if out, err := exec.Command("netsh", "advfirewall", "firewall", "add", "rule", "name="+ReglaFirewall,
			"dir=in", "action=allow", "protocol=TCP", "localport="+strconv.Itoa(op.Puerto), "remoteip=localsubnet",
			"profile=any", "program="+exe, "description="+NombreVisible).CombinedOutput(); err != nil {
			return fmt.Errorf("no se pudo crear la regla del firewall: %v: %s", err, out)
		}
	}
	return nil
}

// estadoSistema: el estado de kardianos ya es correcto sin permisos de administrador.
func estadoSistema(e Estado) Estado { return e }

func limpiarSistema(Opciones) {
	exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", "name="+ReglaFirewall).Run()
}

func esAccesoDenegado(err error) bool { return errors.Is(err, windows.ERROR_ACCESS_DENIED) }
