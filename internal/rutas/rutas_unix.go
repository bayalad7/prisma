//go:build !windows && !darwin

package rutas

// Linux y cualquier otro Unix usan la jerarquía FHS.
func porDefecto() Rutas {
	return Rutas{
		Config: "/etc/prisma/config.toml",
		Datos:  "/var/lib/prisma",
		Logs:   "/var/log/prisma",
	}
}
