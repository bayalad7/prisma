package almacen

import (
	"context"
	"time"
)

// MaxBytesRecurso es el tamaño máximo de un PNG guardado en el puente.
const MaxBytesRecurso = 512 * 1024

// Recurso es una imagen guardada en el puente, referenciable por nombre desde la operación imagen.
type Recurso struct {
	Nombre  string
	PNG     []byte // vacío en los listados
	AnchoPx int
	AltoPx  int
	Creado  time.Time
}

type RepoRecursos struct{ a *Almacen }

// Guardar crea el recurso o reemplaza el existente con el mismo nombre.
// La validación del PNG (formato, dimensiones) corresponde a quien llama.
func (r *RepoRecursos) Guardar(ctx context.Context, rec *Recurso) error {
	ahora := r.a.ahora()
	_, err := r.a.bd().ExecContext(ctx, `INSERT INTO recursos (nombre, png, ancho_px, alto_px, creado) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (nombre) DO UPDATE SET png = excluded.png, ancho_px = excluded.ancho_px,
		alto_px = excluded.alto_px, creado = excluded.creado`,
		rec.Nombre, rec.PNG, rec.AnchoPx, rec.AltoPx, ms(ahora))
	if err != nil {
		return traducirError(err, "recurso "+rec.Nombre)
	}
	rec.Creado = desdeMs(ms(ahora))
	return nil
}

func (r *RepoRecursos) Obtener(ctx context.Context, nombre string) (*Recurso, error) {
	var rec Recurso
	var creado int64
	err := r.a.bd().QueryRowContext(ctx, `SELECT nombre, png, ancho_px, alto_px, creado FROM recursos WHERE nombre = ?`, nombre).
		Scan(&rec.Nombre, &rec.PNG, &rec.AnchoPx, &rec.AltoPx, &creado)
	if err != nil {
		return nil, traducirError(err, "recurso "+nombre)
	}
	rec.Creado = desdeMs(creado)
	return &rec, nil
}

// Listar devuelve los recursos sin el contenido PNG.
func (r *RepoRecursos) Listar(ctx context.Context) ([]Recurso, error) {
	filas, err := r.a.bd().QueryContext(ctx, `SELECT nombre, ancho_px, alto_px, creado FROM recursos ORDER BY nombre`)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	var lista []Recurso
	for filas.Next() {
		var rec Recurso
		var creado int64
		if err := filas.Scan(&rec.Nombre, &rec.AnchoPx, &rec.AltoPx, &creado); err != nil {
			return nil, err
		}
		rec.Creado = desdeMs(creado)
		lista = append(lista, rec)
	}
	return lista, filas.Err()
}

func (r *RepoRecursos) Eliminar(ctx context.Context, nombre string) error {
	res, err := r.a.bd().ExecContext(ctx, `DELETE FROM recursos WHERE nombre = ?`, nombre)
	return exigirUnaFila(res, err, "recurso "+nombre)
}
