//go:build !windows

package servicio

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// limiteHerramienta: ninguna herramienta del sistema puede dejar colgado al instalador. En macOS,
// quitar una confianza de administrador espera una autorización que sin pantalla nunca llega.
const limiteHerramienta = 2 * time.Minute

// ejecutar corre una herramienta del sistema y devuelve su salida; las pruebas la reemplazan.
var ejecutar = func(nombre string, args ...string) ([]byte, error) {
	ctx, cancelar := context.WithTimeout(context.Background(), limiteHerramienta)
	defer cancelar()
	salida, err := exec.CommandContext(ctx, nombre, args...).CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return salida, fmt.Errorf("no respondió en %v (¿espera una autorización?)", limiteHerramienta)
	}
	return salida, err
}

// correr ejecuta un comando y convierte su falla en un error con la salida.
func correr(nombre string, args ...string) error {
	if salida, err := ejecutar(nombre, args...); err != nil {
		return fmt.Errorf("%s %s: %v: %s", nombre, strings.Join(args, " "), err, strings.TrimSpace(string(salida)))
	}
	return nil
}
