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
	ConexionVirtual   = "virtual" // dentro del puente: muestra el ticket en el panel, sin papel
)

// Simulaciones de una impresora virtual: solo fallas que la API v1 ya sabe informar.
const (
	SimulacionNormal         = "normal"
	SimulacionFueraDeLinea   = "fuera_de_linea"  // responde IMPRESORA_INALCANZABLE
	SimulacionLenta          = "lenta"           // tarda SimulacionRetrasoMS en recibir cada ticket
	SimulacionFallaAleatoria = "falla_aleatoria" // falla SimulacionFallaPct de cada 100 tickets
)

// Impresora es un dispositivo configurado en el puente, con su perfil.
type Impresora struct {
	ID                        int64
	Alias                     string
	Nombre                    string
	Tipo                      string // red | usb | bluetooth | archivo | virtual
	Destino                   string // ip:puerto, dispositivo USB, puerto serie, ruta de archivo o "memoria"
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
	ConfirmarImpresion        bool // consultar el estado tras cada ticket (solo red y bluetooth)
	Baudios                   int  // 0 = no aplica (solo bluetooth / serie)
	Activa                    bool
	// Solo impresoras virtuales.
	Simulacion          string // SimulacionNormal, SimulacionFueraDeLinea…
	SimulacionRetrasoMS int
	SimulacionFallaPct  int
	TicketsEnMemoria    int // cuántos tickets recuerda (en memoria, nunca en la base)
	Creada              time.Time
	Actualizada         time.Time
}

type RepoImpresoras struct{ a *Almacen }

const columnasImpresora = `id, alias, nombre, tipo, destino, ancho_mm, columnas, columnas_fuente_b, ancho_px,
	pagina_codigos, comando_pagina_codigos, soporta_qr_nativo, soporta_corte, soporta_cajon, soporta_imagen,
	soporta_codigo_barras_nativo, confirmar_impresion, baudios, activa, simulacion, simulacion_retraso_ms,
	simulacion_falla_pct, tickets_en_memoria, creada, actualizada`

// Crear inserta la impresora y completa ID, Creada y Actualizada.
func (r *RepoImpresoras) Crear(ctx context.Context, imp *Impresora) error {
	completarSimulacion(imp)
	ahora := r.a.ahora()
	res, err := r.a.bd().ExecContext(ctx, `INSERT INTO impresoras (alias, nombre, tipo, destino, ancho_mm, columnas,
		columnas_fuente_b, ancho_px, pagina_codigos, comando_pagina_codigos, soporta_qr_nativo, soporta_corte,
		soporta_cajon, soporta_imagen, soporta_codigo_barras_nativo, confirmar_impresion, baudios, activa,
		simulacion, simulacion_retraso_ms, simulacion_falla_pct, tickets_en_memoria, creada,
		actualizada) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		imp.Alias, imp.Nombre, imp.Tipo, imp.Destino, imp.AnchoMM, imp.Columnas, imp.ColumnasFuenteB, imp.AnchoPx,
		imp.PaginaCodigos, imp.ComandoPaginaCodigos, bit(imp.SoportaQRNativo), bit(imp.SoportaCorte),
		bit(imp.SoportaCajon), bit(imp.SoportaImagen), bit(imp.SoportaCodigoBarrasNativo), bit(imp.ConfirmarImpresion),
		baudiosNulo(imp.Baudios),
		bit(imp.Activa), imp.Simulacion, imp.SimulacionRetrasoMS, imp.SimulacionFallaPct, imp.TicketsEnMemoria,
		ms(ahora), ms(ahora))
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
	completarSimulacion(imp)
	ahora := r.a.ahora()
	res, err := r.a.bd().ExecContext(ctx, `UPDATE impresoras SET alias = ?, nombre = ?, tipo = ?, destino = ?,
		ancho_mm = ?, columnas = ?, columnas_fuente_b = ?, ancho_px = ?, pagina_codigos = ?,
		comando_pagina_codigos = ?, soporta_qr_nativo = ?, soporta_corte = ?, soporta_cajon = ?,
		soporta_imagen = ?, soporta_codigo_barras_nativo = ?, confirmar_impresion = ?, baudios = ?, activa = ?,
		simulacion = ?, simulacion_retraso_ms = ?, simulacion_falla_pct = ?, tickets_en_memoria = ?,
		actualizada = ?
		WHERE id = ?`,
		imp.Alias, imp.Nombre, imp.Tipo, imp.Destino, imp.AnchoMM, imp.Columnas, imp.ColumnasFuenteB, imp.AnchoPx,
		imp.PaginaCodigos, imp.ComandoPaginaCodigos, bit(imp.SoportaQRNativo), bit(imp.SoportaCorte),
		bit(imp.SoportaCajon), bit(imp.SoportaImagen), bit(imp.SoportaCodigoBarrasNativo), bit(imp.ConfirmarImpresion),
		baudiosNulo(imp.Baudios),
		bit(imp.Activa), imp.Simulacion, imp.SimulacionRetrasoMS, imp.SimulacionFallaPct, imp.TicketsEnMemoria,
		ms(ahora), imp.ID)
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
	var qr, corte, cajon, imagen, barras, confirmar, activa int
	var baudios sql.NullInt64
	var creada, actualizada int64
	err := f.Scan(&imp.ID, &imp.Alias, &imp.Nombre, &imp.Tipo, &imp.Destino, &imp.AnchoMM, &imp.Columnas,
		&imp.ColumnasFuenteB, &imp.AnchoPx, &imp.PaginaCodigos, &imp.ComandoPaginaCodigos, &qr, &corte, &cajon,
		&imagen, &barras, &confirmar, &baudios, &activa, &imp.Simulacion, &imp.SimulacionRetrasoMS,
		&imp.SimulacionFallaPct, &imp.TicketsEnMemoria, &creada, &actualizada)
	if err != nil {
		return nil, err
	}
	imp.SoportaQRNativo, imp.SoportaCorte, imp.SoportaCajon = qr == 1, corte == 1, cajon == 1
	imp.SoportaImagen, imp.SoportaCodigoBarrasNativo, imp.Activa = imagen == 1, barras == 1, activa == 1
	imp.ConfirmarImpresion = confirmar == 1
	imp.Baudios = int(baudios.Int64)
	imp.Creada, imp.Actualizada = desdeMs(creada), desdeMs(actualizada)
	return &imp, nil
}

// completarSimulacion pone los valores por defecto a una impresora sin simulación configurada.
func completarSimulacion(imp *Impresora) {
	if imp.Simulacion == "" {
		imp.Simulacion = SimulacionNormal
	}
	if imp.TicketsEnMemoria == 0 {
		imp.TicketsEnMemoria = 20
	}
	// Un retraso de 0 ms o un 0 % de fallas no simulan nada: se usan los valores típicos.
	if imp.SimulacionRetrasoMS == 0 {
		imp.SimulacionRetrasoMS = 3000
	}
	if imp.SimulacionFallaPct == 0 {
		imp.SimulacionFallaPct = 30
	}
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
