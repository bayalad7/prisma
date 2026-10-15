// Package rutas define dónde vive PRISMA en cada sistema operativo.
// Los valores concretos están en rutas_<sistema>.go (seleccionados por build tags).
package rutas

import "path/filepath"

// NombreBaseDatos es el archivo SQLite dentro del directorio de datos.
const NombreBaseDatos = "prisma.db"

// Rutas agrupa las ubicaciones por defecto del sistema.
type Rutas struct {
	Config string // archivo config.toml
	Datos  string // directorio de datos (base de datos, certificados autofirmados)
	Logs   string // directorio de bitácoras
}

// PorDefecto devuelve las rutas del sistema actual.
func PorDefecto() Rutas {
	return porDefecto()
}

// BaseDatos devuelve la ruta del archivo SQLite dentro de un directorio de datos.
func BaseDatos(dirDatos string) string {
	return filepath.Join(dirDatos, NombreBaseDatos)
}
