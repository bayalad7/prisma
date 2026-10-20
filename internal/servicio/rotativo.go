package servicio

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Tamaño y cantidad de las bitácoras del servicio: prisma.log y prisma.log.1 … prisma.log.5.
const (
	MaxBytesBitacora = 10 << 20
	CopiasBitacora   = 5
)

// Rotativo es un io.Writer que escribe en un archivo y lo rota al llegar a MaxBytes, conservando
// Copias archivos anteriores. Es seguro para uso concurrente.
type Rotativo struct {
	Ruta     string
	MaxBytes int64
	Copias   int

	mu     sync.Mutex
	f      *os.File
	tamano int64
}

// AbrirRotativo crea el directorio si hace falta y abre (o continúa) la bitácora.
func AbrirRotativo(ruta string, maxBytes int64, copias int) (*Rotativo, error) {
	r := &Rotativo{Ruta: ruta, MaxBytes: maxBytes, Copias: copias}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o750); err != nil {
		return nil, err
	}
	if err := r.abrir(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Rotativo) abrir() error {
	f, err := os.OpenFile(r.Ruta, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	r.f, r.tamano = f, info.Size()
	return nil
}

func (r *Rotativo) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		return 0, os.ErrClosed
	}
	// Una línea nunca se parte entre dos archivos: se rota antes de escribirla.
	if r.tamano > 0 && r.tamano+int64(len(p)) > r.MaxBytes {
		if err := r.rotar(); err != nil {
			return 0, err
		}
	}
	n, err := r.f.Write(p)
	r.tamano += int64(n)
	return n, err
}

// rotar: prisma.log.(N-1) → .N, …, prisma.log → .1, y abre uno nuevo.
func (r *Rotativo) rotar() error {
	if err := r.f.Close(); err != nil {
		return err
	}
	r.f = nil
	os.Remove(fmt.Sprintf("%s.%d", r.Ruta, r.Copias))
	for i := r.Copias - 1; i >= 1; i-- {
		os.Rename(fmt.Sprintf("%s.%d", r.Ruta, i), fmt.Sprintf("%s.%d", r.Ruta, i+1))
	}
	if r.Copias > 0 {
		if err := os.Rename(r.Ruta, r.Ruta+".1"); err != nil {
			return err
		}
	} else if err := os.Remove(r.Ruta); err != nil {
		return err
	}
	return r.abrir()
}

func (r *Rotativo) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		return nil
	}
	err := r.f.Close()
	r.f = nil
	return err
}
