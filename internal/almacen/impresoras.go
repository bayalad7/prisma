package almacen

import (
	"context"
	"database/sql"
	"time"
)

// Tipos de conexión de una impresora.
const (
	ConexionRed       = "red"
	ConexionUSB       = "usb"
	ConexionBluetooth = "bluetooth"
	ConexionArchivo   = "archivo"
)

// Impresora es un dispositivo configurado en el puente, con su perfil.
type Impresora struct {
	ID                        int64
	Alias                     string
	Nombre                    string
	Tipo                      string // red | usb | bluetooth | archivo
	Destino                   string // ip:puerto, dispositivo USB, puerto serie o ruta de archivo
	AnchoMM                   int
	Columnas                  int // columnas con la fuente A
	ColumnasFuenteB           int
	AnchoPx                   int // puntos imprimibles por línea
	PaginaCodigos             string
	ComandoPaginaCodigos      int // n de ESC t n
	SoportaQRNativo           bool
	SoportaCorte              bool
	SoportaCajon              bool
	SoportaImagen             bool
	SoportaCodigoBarrasNativo bool
	Baudios                   int // 0 = no aplica (solo bluetooth / serie)
	Activa                    bool
	Creada                    time.Time
	Actualizada               time.Time
}

type RepoImpresoras struct{ a *Almacen }

const columnasImpresora = `id, alias, nombre, tipo, destino, ancho_mm, columnas, columnas_fuente_b, ancho_px,
	pagina_codigos, comando_pagina_codigos, soporta_qr_nativo, soporta_corte, soporta_cajon, soporta_imagen,
	soporta_codigo_barras_nativo, baudios, activa, creada, actualizada`

// Crear inserta la impresora y completa ID, Creada y Actualizada.
func (r *RepoImpresoras) Crear(ctx context.Context, imp *Impresora) error {
	ahora := r.a.ahora()
	res, err := r.a.bd().ExecContext(ctx, `INSERT INTO impresoras (alias, nombre, tipo, destino, ancho_mm, columnas,
		columnas_fuente_b, ancho_px, pagina_codigos, comando_pagina_codigos, soporta_qr_nativo, soporta_corte,
		soporta_cajon, soporta_imagen, soporta_codigo_barras_nativo, baudios, activa, creada, actualizada)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		imp.Alias, imp.Nombre, imp.Tipo, imp.Destino, imp.AnchoMM, imp.Columnas, imp.ColumnasFuenteB, imp.AnchoPx,
		imp.PaginaCodigos, imp.ComandoPaginaCodigos, bit(imp.SoportaQRNativo), bit(imp.SoportaCorte),
		bit(imp.SoportaCajon), bit(imp.SoportaImagen), bit(imp.SoportaCodigoBarrasNativo), baudiosNulo(imp.Baudios),
		bit(imp.Activa), ms(ahora), ms(ahora))
	if err != nil {
		return traducirError(err, "impresora "+imp.Alias)
	}
	imp.ID, _ = res.LastInsertId()
	imp.Creada, imp.Actualizada = desdeMs(ms(ahora)), desdeMs(ms(ahora))
	return nil
}

// Obtener busca una impresora por su alias.
func (r *RepoImpresoras) Obtener(ctx context.Context, alias string) (*Impresora, error) {
	fila := r.a.bd().QueryRowContext(ctx, `SELECT `+columnasImpresora+` FROM impresoras WHERE alias = ?`, alias)
	imp, err := escanearImpresora(fila)
	return imp, traducirError(err, "impresora "+alias)
}

// Listar devuelve todas las impresoras ordenadas por alias.
func (r *RepoImpresoras) Listar(ctx context.Context) ([]Impresora, error) {
	filas, err := r.a.bd().QueryContext(ctx, `SELECT `+columnasImpresora+` FROM impresoras ORDER BY alias`)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	var lista []Impresora
	for filas.Next() {
		imp, err := escanearImpresora(filas)
		if err != nil {
			return nil, err
		}
		lista = append(lista, *imp)
	}
	return lista, filas.Err()
}

// Actualizar guarda todos los campos de la impresora identificada por ID (incluido un cambio de alias).
func (r *RepoImpresoras) Actualizar(ctx context.Context, imp *Impresora) error {
	ahora := r.a.ahora()
	res, err := r.a.bd().ExecContext(ctx, `UPDATE impresoras SET alias = ?, nombre = ?, tipo = ?, destino = ?,
		ancho_mm = ?, columnas = ?, columnas_fuente_b = ?, ancho_px = ?, pagina_codigos = ?,
		comando_pagina_codigos = ?, soporta_qr_nativo = ?, soporta_corte = ?, soporta_cajon = ?,
		soporta_imagen = ?, soporta_codigo_barras_nativo = ?, baudios = ?, activa = ?, actualizada = ?
		WHERE id = ?`,
		imp.Alias, imp.Nombre, imp.Tipo, imp.Destino, imp.AnchoMM, imp.Columnas, imp.ColumnasFuenteB, imp.AnchoPx,
		imp.PaginaCodigos, imp.ComandoPaginaCodigos, bit(imp.SoportaQRNativo), bit(imp.SoportaCorte),
		bit(imp.SoportaCajon), bit(imp.SoportaImagen), bit(imp.SoportaCodigoBarrasNativo), baudiosNulo(imp.Baudios),
		bit(imp.Activa), ms(ahora), imp.ID)
	if err := exigirUnaFila(res, err, "impresora "+imp.Alias); err != nil {
		return err
	}
	imp.Actualizada = desdeMs(ms(ahora))
	return nil
}

// Eliminar borra la impresora; los permisos de clientes sobre ella se borran en cascada.
func (r *RepoImpresoras) Eliminar(ctx context.Context, alias string) error {
	res, err := r.a.bd().ExecContext(ctx, `DELETE FROM impresoras WHERE alias = ?`, alias)
	return exigirUnaFila(res, err, "impresora "+alias)
}

func escanearImpresora(f filaEscaneable) (*Impresora, error) {
	var imp Impresora
	var qr, corte, cajon, imagen, barras, activa int
	var baudios sql.NullInt64
	var creada, actualizada int64
	err := f.Scan(&imp.ID, &imp.Alias, &imp.Nombre, &imp.Tipo, &imp.Destino, &imp.AnchoMM, &imp.Columnas,
		&imp.ColumnasFuenteB, &imp.AnchoPx, &imp.PaginaCodigos, &imp.ComandoPaginaCodigos, &qr, &corte, &cajon,
		&imagen, &barras, &baudios, &activa, &creada, &actualizada)
	if err != nil {
		return nil, err
	}
	imp.SoportaQRNativo, imp.SoportaCorte, imp.SoportaCajon = qr == 1, corte == 1, cajon == 1
	imp.SoportaImagen, imp.SoportaCodigoBarrasNativo, imp.Activa = imagen == 1, barras == 1, activa == 1
	imp.Baudios = int(baudios.Int64)
	imp.Creada, imp.Actualizada = desdeMs(creada), desdeMs(actualizada)
	return &imp, nil
}

func baudiosNulo(b int) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(b), Valid: b != 0}
}

// exigirUnaFila traduce el error de un UPDATE/DELETE y devuelve ErrNoEncontrado si no afectó ninguna fila.
func exigirUnaFila(res sql.Result, err error, que string) error {
	if err != nil {
		return traducirError(err, que)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return traducirError(sql.ErrNoRows, que)
	}
	return nil
}
