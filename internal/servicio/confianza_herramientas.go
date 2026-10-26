//go:build !windows

package servicio

import (
	"fmt"
	"os/exec"
	"strings"
)

// ejecutar corre una herramienta del sistema y devuelve su salida; las pruebas la reemplazan.
var ejecutar = func(nombre string, args ...string) ([]byte, error) {
	return exec.Command(nombre, args...).CombinedOutput()
}

// correr ejecuta un comando y convierte su falla en un error con la salida.
func correr(nombre string, args ...string) error {
	if salida, err := ejecutar(nombre, args...); err != nil {
		return fmt.Errorf("%s %s: %v: %s", nombre, strings.Join(args, " "), err, strings.TrimSpace(string(salida)))
	}
	return nil
}
