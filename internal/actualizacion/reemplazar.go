package actualizacion

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Reemplazar instala nuevo en lugar de actual. En Linux y macOS basta con renombrar (el proceso en
// marcha conserva el archivo anterior). Windows no deja sobrescribir un .exe en uso, pero sí
// renombrarlo: el anterior queda como .old y se borra en el siguiente arranque.
func Reemplazar(nuevo, actual string) error {
	if runtime.GOOS != "windows" {
		return os.Rename(nuevo, actual)
	}
	viejo := actual + ".old"
	os.Remove(viejo)
	if err := os.Rename(actual, viejo); err != nil {
		return err
	}
	if err := os.Rename(nuevo, actual); err != nil {
		os.Rename(viejo, actual)
		return err
	}
	return nil
}

// LimpiarAnterior borra el .old que dejó una actualización en Windows.
func LimpiarAnterior() {
	if runtime.GOOS != "windows" {
		return
	}
	if exe, err := os.Executable(); err == nil {
		os.Remove(exe + ".old")
	}
}

// InstaladoConPaquete indica si el binario lo instaló el paquete .deb: entonces se actualiza con
// dpkg para que el sistema de paquetes siga sabiendo qué versión hay.
func InstaladoConPaquete(exe string) bool {
	if runtime.GOOS != "linux" || filepath.Dir(exe) != "/usr/bin" {
		return false
	}
	return exec.Command("dpkg", "-S", exe).Run() == nil
}

// ErrSinPermiso: el usuario no puede escribir donde está instalado PRISMA.
var ErrSinPermiso = errors.New("no hay permiso para reemplazar el programa")

// Escribible comprueba que se pueda crear un archivo junto al ejecutable.
func Escribible(dir string) error {
	f, err := os.CreateTemp(dir, ".prisma-prueba-*")
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return ErrSinPermiso
		}
		return err
	}
	f.Close()
	return os.Remove(f.Name())
}
