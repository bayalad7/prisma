package servicio

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
)

// Unidades que instala el paquete .deb: si existen, el servicio lo administra el paquete.
var unidadesDelPaquete = []string{"/lib/systemd/system/prisma.service", "/usr/lib/systemd/system/prisma.service"}

func prepararSistema(op Opciones) error {
	if os.Geteuid() != 0 {
		return errors.New("instalar el servicio requiere permisos de administrador: ejecuta el comando con sudo")
	}
	for _, u := range unidadesDelPaquete {
		if _, err := os.Stat(u); err == nil {
			return fmt.Errorf("el servicio ya lo instaló el paquete .deb (%s); adminístralo con «prisma servicio iniciar|detener|estado»", u)
		}
	}
	exe, err := ejecutable()
	if err != nil {
		return err
	}
	// El usuario del servicio no puede entrar a las carpetas personales (ProtectHome en la unidad).
	for _, prefijo := range []string{"/home/", "/root/", "/tmp/", "/var/tmp/", "/run/user/"} {
		if strings.HasPrefix(exe, prefijo) {
			return fmt.Errorf("el binario está en %s, fuera del alcance del servicio; muévelo antes de instalar:\n  sudo install -m 0755 %s /usr/local/bin/prisma\n  sudo /usr/local/bin/prisma servicio instalar", exe, exe)
		}
	}
	if _, err := user.Lookup(UsuarioLinux); err != nil {
		if out, err := exec.Command("useradd", "--system", "--user-group", "--home-dir", "/var/lib/prisma",
			"--no-create-home", "--shell", "/usr/sbin/nologin", UsuarioLinux).CombinedOutput(); err != nil {
			return fmt.Errorf("no se pudo crear el usuario %s: %v: %s", UsuarioLinux, err, out)
		}
	}
	// lp: impresoras USB (/dev/usb/lp*); dialout: puertos serie y Bluetooth (/dev/rfcomm*).
	for _, g := range []string{"lp", "dialout"} {
		if _, err := user.LookupGroup(g); err == nil {
			if out, err := exec.Command("usermod", "-aG", g, UsuarioLinux).CombinedOutput(); err != nil {
				return fmt.Errorf("no se pudo agregar %s al grupo %s: %v: %s", UsuarioLinux, g, err, out)
			}
		}
	}
	return crearCarpetas(op)
}

// crearCarpetas deja datos y bitácoras con dueño prisma y permisos 0750. systemd también crea los
// de por defecto (StateDirectory, LogsDirectory); esto cubre rutas propias de config.toml.
func crearCarpetas(op Opciones) error {
	u, err := user.Lookup(UsuarioLinux)
	if err != nil {
		return err
	}
	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(u.Gid)
	for _, dir := range []string{op.Datos, op.Logs} {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return err
		}
		if err := os.Chown(dir, uid, gid); err != nil {
			return err
		}
	}
	return nil
}

func limpiarSistema(Opciones) {}

func esAccesoDenegado(error) bool { return false }
