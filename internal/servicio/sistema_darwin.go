package servicio

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

func prepararSistema(op Opciones) error {
	if os.Geteuid() != 0 {
		return errors.New("instalar el servicio requiere permisos de administrador: ejecuta el comando con sudo")
	}
	if _, err := user.Lookup(UsuarioMacOS); err != nil {
		if err := crearUsuario(); err != nil {
			return fmt.Errorf("no se pudo crear el usuario %s: %w", UsuarioMacOS, err)
		}
	}
	u, err := user.Lookup(UsuarioMacOS)
	if err != nil {
		return err
	}
	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(u.Gid)
	// La carpeta que contiene a datos (y a config.toml) debe poder recorrerla _prisma: MkdirAll
	// la crearía con los permisos de datos (0750, de root) y el servicio no podría ni leer su
	// configuración. Se corrige también si una instalación anterior la dejó así.
	if op.Datos != "" {
		padre := filepath.Dir(op.Datos)
		if err := os.MkdirAll(padre, 0o755); err != nil {
			return err
		}
		if err := os.Chmod(padre, 0o755); err != nil {
			return err
		}
	}
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

// crearUsuario crea el grupo y el usuario de sistema ocultos _prisma con un id libre por debajo de
// 500, como los demás demonios de macOS (_www, _lp…).
func crearUsuario() error {
	id, err := idLibre()
	if err != nil {
		return err
	}
	n := strconv.Itoa(id)
	pasos := [][]string{
		{"-create", "/Groups/" + UsuarioMacOS},
		{"-create", "/Groups/" + UsuarioMacOS, "PrimaryGroupID", n},
		{"-create", "/Groups/" + UsuarioMacOS, "RealName", "PRISMA"},
		{"-create", "/Groups/" + UsuarioMacOS, "Password", "*"},
		{"-create", "/Users/" + UsuarioMacOS},
		{"-create", "/Users/" + UsuarioMacOS, "UniqueID", n},
		{"-create", "/Users/" + UsuarioMacOS, "PrimaryGroupID", n},
		{"-create", "/Users/" + UsuarioMacOS, "RealName", "PRISMA"},
		{"-create", "/Users/" + UsuarioMacOS, "UserShell", "/usr/bin/false"},
		{"-create", "/Users/" + UsuarioMacOS, "NFSHomeDirectory", "/var/empty"},
		{"-create", "/Users/" + UsuarioMacOS, "Password", "*"},
		{"-create", "/Users/" + UsuarioMacOS, "IsHidden", "1"},
	}
	for _, p := range pasos {
		if out, err := exec.Command("dscl", append([]string{"."}, p...)...).CombinedOutput(); err != nil {
			return fmt.Errorf("dscl %s: %v: %s", strings.Join(p, " "), err, out)
		}
	}
	return nil
}

// idLibre busca un número que no usen ni usuarios ni grupos, entre 300 y 499.
func idLibre() (int, error) {
	usados := map[int]bool{}
	for _, consulta := range [][]string{{".", "-list", "/Users", "UniqueID"}, {".", "-list", "/Groups", "PrimaryGroupID"}} {
		out, err := exec.Command("dscl", consulta...).Output()
		if err != nil {
			return 0, err
		}
		for _, linea := range strings.Split(string(out), "\n") {
			campos := strings.Fields(linea)
			if len(campos) == 2 {
				if n, err := strconv.Atoi(campos[1]); err == nil {
					usados[n] = true
				}
			}
		}
	}
	for id := 300; id < 500; id++ {
		if !usados[id] {
			return id, nil
		}
	}
	return 0, errors.New("no hay ids de sistema libres entre 300 y 499")
}

func limpiarSistema(Opciones) {}

// estadoSistema corrige el estado sin permisos de administrador. kardianos consulta
// «launchctl list», que sin root mira el dominio del usuario y no ve el LaunchDaemon del sistema:
// el servicio en marcha aparecía como detenido. «launchctl print system/…» sí lo ve.
func estadoSistema(e Estado) Estado {
	if !e.Instalado || e.EnMarcha || os.Geteuid() == 0 {
		return e
	}
	salida, err := exec.Command("launchctl", "print", "system/"+NombreMacOS).Output()
	if err == nil && launchdEnMarcha(string(salida)) {
		e.EnMarcha = true
	}
	return e
}

func esAccesoDenegado(error) bool { return false }

// marcadoParaEliminacion solo pasa en Windows (ver reinstalar).
func marcadoParaEliminacion(error) bool { return false }
