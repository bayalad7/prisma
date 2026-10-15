// Package version expone la versión del binario. Los valores se inyectan al compilar con
// -ldflags "-X github.com/bayalad7/prisma/internal/version.Version=..." (ver Makefile y build.ps1).
package version

import (
	"fmt"
	"runtime"
)

var (
	Version = "0.0.0-dev"
	Commit  = "desconocido"
	Fecha   = "desconocida"
)

// VersionesDocumento son las versiones del contrato del documento que este puente acepta.
var VersionesDocumento = []string{"1"}

// Texto devuelve la descripción completa que muestra `prisma version`.
func Texto() string {
	return fmt.Sprintf("PRISMA %s\ncommit: %s\ncompilado: %s\nplataforma: %s/%s (%s)\ncontrato de documento: %v",
		Version, Commit, Fecha, runtime.GOOS, runtime.GOARCH, runtime.Version(), VersionesDocumento)
}
