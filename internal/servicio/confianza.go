package servicio

// Confianza en el certificado del puente en ESTE equipo (ADR 0020): el instalador lo agrega al
// almacén de confianza del sistema para que el navegador abra el panel sin advertencia, y lo quita
// al desinstalar. Solo se usa con el certificado autofirmado limitado a la red local; los demás
// equipos siguen instalándolo a mano (docs/instalacion/https-y-certificados.md).

// OrganizacionCertificado identifica en el almacén los certificados que puso PRISMA, para quitar
// los anteriores al confiar en uno nuevo y todos al desinstalar.
const OrganizacionCertificado = "PRISMA (autofirmado)"

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
