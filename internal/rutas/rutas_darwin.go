//go:build darwin

package rutas

const raizDarwin = "/Library/Application Support/PRISMA"

func porDefecto() Rutas {
	return Rutas{
		Config: raizDarwin + "/config.toml",
		Datos:  raizDarwin + "/datos",
		Logs:   "/Library/Logs/PRISMA",
	}
}
