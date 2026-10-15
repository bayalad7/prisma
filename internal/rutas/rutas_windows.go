//go:build windows

package rutas

import (
	"os"
	"path/filepath"
)

func porDefecto() Rutas {
	base := os.Getenv("ProgramData")
	if base == "" {
		base = `C:\ProgramData`
	}
	raiz := filepath.Join(base, "PRISMA")
	return Rutas{
		Config: filepath.Join(raiz, "config.toml"),
		Datos:  filepath.Join(raiz, "datos"),
		Logs:   filepath.Join(raiz, "logs"),
	}
}
