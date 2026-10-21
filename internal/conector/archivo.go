package conector

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/dev"
)

// Archivo guarda cada trabajo como <destino>/<fecha>-<id>.bin junto con un .txt que muestra
// cómo se vería impreso. Sirve para desarrollar y para auditar sin impresora.
type Archivo struct {
	Directorio string
	Perfil     almacen.Impresora
	ahora      func() time.Time
}

func NuevoArchivo(directorio string, perfil almacen.Impresora) *Archivo {
	return &Archivo{Directorio: directorio, Perfil: perfil, ahora: time.Now}
}

func (a *Archivo) Enviar(ctx context.Context, datos []byte) error {
	if err := os.MkdirAll(a.Directorio, 0o755); err != nil {
		return a.traducir(err)
	}
	id := trabajoDe(ctx)
	if id == "" {
		b := make([]byte, 4)
		rand.Read(b)
		id = hex.EncodeToString(b)
	}
	base := filepath.Join(a.Directorio, a.ahora().Format("20060102-150405.000")+"-"+id)
	if err := os.WriteFile(base+".bin", datos, 0o640); err != nil {
		return a.traducir(err)
	}
	vista := dev.VistaTexto(dev.Simular(datos, a.Perfil), false)
	if err := os.WriteFile(base+".txt", []byte(vista), 0o640); err != nil {
		return a.traducir(err)
	}
	return nil
}

// Probar comprueba que el directorio existe (o se puede crear) y admite escritura.
func (a *Archivo) Probar(context.Context) error {
	if err := os.MkdirAll(a.Directorio, 0o755); err != nil {
		return a.traducir(err)
	}
	f, err := os.CreateTemp(a.Directorio, ".prueba-*")
	if err != nil {
		return a.traducir(err)
	}
	f.Close()
	return os.Remove(f.Name())
}

func (a *Archivo) traducir(err error) error {
	if errors.Is(err, fs.ErrPermission) {
		return nuevoError(ErrPermisoDenegado, fmt.Sprintf("No hay permiso para escribir en %s.", a.Directorio),
			"Elige una carpeta en la que pueda escribir el usuario del servicio PRISMA.", err)
	}
	return nuevoError(ErrImpresoraInalcanzable, fmt.Sprintf("No se pudo escribir en %s.", a.Directorio),
		"Revisa que la carpeta exista y que haya espacio en disco.", err)
}
