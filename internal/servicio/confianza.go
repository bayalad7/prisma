package servicio

import (
	"fmt"
	"os/exec"
	"strings"
)

// Confianza en el certificado del puente en ESTE equipo (ADR 0020): el instalador lo agrega al
// almacén de confianza del sistema para que el navegador abra el panel sin advertencia, y lo quita
// al desinstalar. Solo se usa con el certificado autofirmado limitado a la red local; los demás
// equipos siguen instalándolo a mano (docs/instalacion/https-y-certificados.md).

// OrganizacionCertificado identifica en el almacén los certificados que puso PRISMA, para quitar
// los anteriores al confiar en uno nuevo y todos al desinstalar.
const OrganizacionCertificado = "PRISMA (autofirmado)"

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

// ConfiarCertificado agrega el certificado (PEM) al almacén de confianza del sistema, después de
// quitar los que PRISMA haya puesto antes. Requiere permisos de administrador.
func ConfiarCertificado(rutaPEM string) error {
	if err := QuitarConfianza(); err != nil {
		return err
	}
	return confiarEnSistema(rutaPEM)
}

// QuitarConfianza quita del almacén de confianza del sistema los certificados que puso PRISMA. No
// falla si no había ninguno.
func QuitarConfianza() error { return quitarDeSistema() }
