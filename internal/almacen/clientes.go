package almacen

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Niveles de seguridad de un cliente.
const (
	NivelOrigen = 1
	NivelLlave  = 2
	NivelFirma  = 3
)

// Operaciones sensibles que se habilitan por cliente.
const (
	OperacionAbrirCajon = "abrir_cajon"
	OperacionRaw        = "raw"
)

// Cliente es una aplicación autorizada a imprimir.
type Cliente struct {
	ID                   string // cli_...
	Nombre               string
	Nivel                int
	LlaveHash            string // hash argon2id (PHC) de la llave pk_; obligatoria en nivel 2
	LlavePrefijo         string // parte pública de la llave, para buscarla y mostrarla
	LlavePublica         []byte // nivel 3: llave pública Ed25519
	ExigirLlave          bool   // nivel 3: exige además la llave pk_
	OperacionesSensibles []string
	TodasLasImpresoras   bool // false = solo las de ImpresorasPermitidas
	Activo               bool
	Creado               time.Time
	Actualizado          time.Time
}

// PermiteOperacion indica si el cliente tiene habilitada una operación sensible.
func (c *Cliente) PermiteOperacion(op string) bool {
	for _, o := range c.OperacionesSensibles {
		if o == op {
			return true
		}
	}
	return false
}

type RepoClientes struct{ a *Almacen }

const columnasCliente = `id, nombre, nivel, llave_hash, llave_prefijo, llave_publica, exigir_llave,
	operaciones_sensibles, todas_las_impresoras, activo, creado, actualizado`

func (r *RepoClientes) Crear(ctx context.Context, c *Cliente) error {
	ops, err := codificarLista(c.OperacionesSensibles)
	if err != nil {
		return err
	}
	ahora := r.a.ahora()
	_, err = r.a.bd().ExecContext(ctx, `INSERT INTO clientes (`+columnasCliente+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Nombre, c.Nivel, textoNulo(c.LlaveHash), textoNulo(c.LlavePrefijo), c.LlavePublica, bit(c.ExigirLlave), ops,
		bit(c.TodasLasImpresoras), bit(c.Activo), ms(ahora), ms(ahora))
	if err != nil {
		return traducirError(err, "cliente "+c.ID)
	}
	c.Creado, c.Actualizado = desdeMs(ms(ahora)), desdeMs(ms(ahora))
	return nil
}

func (r *RepoClientes) Obtener(ctx context.Context, id string) (*Cliente, error) {
	return r.uno(ctx, `WHERE id = ?`, id, "cliente "+id)
}

// ObtenerPorPrefijo busca al cliente dueño de una llave pk_ por su prefijo público.
func (r *RepoClientes) ObtenerPorPrefijo(ctx context.Context, prefijo string) (*Cliente, error) {
	return r.uno(ctx, `WHERE llave_prefijo = ?`, prefijo, "cliente con llave "+prefijo)
}

// ObtenerPorOrigen busca al cliente al que pertenece un origen (se normaliza antes de buscar).
func (r *RepoClientes) ObtenerPorOrigen(ctx context.Context, origen string) (*Cliente, error) {
	norm, err := NormalizarOrigen(origen)
	if err != nil {
		return nil, err
	}
	return r.uno(ctx, `WHERE id = (SELECT cliente_id FROM clientes_origenes WHERE origen = ?)`, norm, "cliente del origen "+norm)
}

func (r *RepoClientes) uno(ctx context.Context, where string, arg any, que string) (*Cliente, error) {
	c, err := escanearCliente(r.a.bd().QueryRowContext(ctx, `SELECT `+columnasCliente+` FROM clientes `+where, arg))
	return c, traducirError(err, que)
}

func (r *RepoClientes) Listar(ctx context.Context) ([]Cliente, error) {
	filas, err := r.a.bd().QueryContext(ctx, `SELECT `+columnasCliente+` FROM clientes ORDER BY nombre, id`)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	var lista []Cliente
	for filas.Next() {
		c, err := escanearCliente(filas)
		if err != nil {
			return nil, err
		}
		lista = append(lista, *c)
	}
	return lista, filas.Err()
}

func (r *RepoClientes) Actualizar(ctx context.Context, c *Cliente) error {
	ops, err := codificarLista(c.OperacionesSensibles)
	if err != nil {
		return err
	}
	ahora := r.a.ahora()
	res, err := r.a.bd().ExecContext(ctx, `UPDATE clientes SET nombre = ?, nivel = ?, llave_hash = ?, llave_prefijo = ?,
		llave_publica = ?, exigir_llave = ?, operaciones_sensibles = ?, todas_las_impresoras = ?, activo = ?,
		actualizado = ? WHERE id = ?`,
		c.Nombre, c.Nivel, textoNulo(c.LlaveHash), textoNulo(c.LlavePrefijo), c.LlavePublica, bit(c.ExigirLlave), ops,
		bit(c.TodasLasImpresoras), bit(c.Activo), ms(ahora), c.ID)
	if err := exigirUnaFila(res, err, "cliente "+c.ID); err != nil {
		return err
	}
	c.Actualizado = desdeMs(ms(ahora))
	return nil
}

// Eliminar borra el cliente con sus orígenes y permisos. Su historial de trabajos se conserva.
func (r *RepoClientes) Eliminar(ctx context.Context, id string) error {
	res, err := r.a.bd().ExecContext(ctx, `DELETE FROM clientes WHERE id = ?`, id)
	return exigirUnaFila(res, err, "cliente "+id)
}

// EstablecerImpresoras define a qué impresoras puede imprimir el cliente. Con todas = true se
// ignoran los alias. Con todas = false, una lista vacía significa que no puede imprimir en ninguna.
func (r *RepoClientes) EstablecerImpresoras(ctx context.Context, clienteID string, todas bool, alias []string) error {
	return r.a.enTransaccion(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE clientes SET todas_las_impresoras = ?, actualizado = ? WHERE id = ?`,
			bit(todas), ms(r.a.ahora()), clienteID)
		if err := exigirUnaFila(res, err, "cliente "+clienteID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM clientes_impresoras WHERE cliente_id = ?`, clienteID); err != nil {
			return err
		}
		if todas {
			return nil
		}
		for _, a := range alias {
			res, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO clientes_impresoras (cliente_id, impresora_id)
				SELECT ?, id FROM impresoras WHERE alias = ?`, clienteID, a)
			if err != nil {
				return traducirError(err, "impresora "+a)
			}
			if n, _ := res.RowsAffected(); n == 0 {
				var existe int
				if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM impresoras WHERE alias = ?`, a).Scan(&existe); err != nil {
					return err
				}
				if existe == 0 {
					return traducirError(sql.ErrNoRows, "impresora "+a)
				}
			}
		}
		return nil
	})
}

// ImpresorasPermitidas devuelve los alias permitidos explícitamente (solo aplica si TodasLasImpresoras = false).
func (r *RepoClientes) ImpresorasPermitidas(ctx context.Context, clienteID string) ([]string, error) {
	filas, err := r.a.bd().QueryContext(ctx, `SELECT i.alias FROM clientes_impresoras ci
		JOIN impresoras i ON i.id = ci.impresora_id WHERE ci.cliente_id = ? ORDER BY i.alias`, clienteID)
	if err != nil {
		return nil, err
	}
	return escanearTextos(filas)
}

// PuedeImprimirEn indica si el cliente tiene permiso sobre la impresora (no comprueba si está activa).
func (r *RepoClientes) PuedeImprimirEn(ctx context.Context, clienteID, alias string) (bool, error) {
	var permitido bool
	err := r.a.bd().QueryRowContext(ctx, `SELECT c.todas_las_impresoras = 1 OR EXISTS (
			SELECT 1 FROM clientes_impresoras ci JOIN impresoras i ON i.id = ci.impresora_id
			WHERE ci.cliente_id = c.id AND i.alias = ?)
		FROM clientes c WHERE c.id = ?`, alias, clienteID).Scan(&permitido)
	return permitido, traducirError(err, "cliente "+clienteID)
}

// AgregarOrigen aprueba un origen para el cliente. Falla con ErrDuplicado si ya pertenece a cualquier cliente.
func (r *RepoClientes) AgregarOrigen(ctx context.Context, clienteID, origen string) error {
	norm, err := NormalizarOrigen(origen)
	if err != nil {
		return err
	}
	_, err = r.a.bd().ExecContext(ctx, `INSERT INTO clientes_origenes (cliente_id, origen) VALUES (?, ?)`, clienteID, norm)
	return traducirError(err, "origen "+norm)
}

func (r *RepoClientes) QuitarOrigen(ctx context.Context, clienteID, origen string) error {
	norm, err := NormalizarOrigen(origen)
	if err != nil {
		return err
	}
	res, err := r.a.bd().ExecContext(ctx, `DELETE FROM clientes_origenes WHERE cliente_id = ? AND origen = ?`, clienteID, norm)
	return exigirUnaFila(res, err, "origen "+norm)
}

func (r *RepoClientes) Origenes(ctx context.Context, clienteID string) ([]string, error) {
	filas, err := r.a.bd().QueryContext(ctx, `SELECT origen FROM clientes_origenes WHERE cliente_id = ? ORDER BY origen`, clienteID)
	if err != nil {
		return nil, err
	}
	return escanearTextos(filas)
}

func escanearCliente(f filaEscaneable) (*Cliente, error) {
	var c Cliente
	var hash, prefijo sql.NullString
	var ops string
	var exigir, todas, activo int
	var creado, actualizado int64
	err := f.Scan(&c.ID, &c.Nombre, &c.Nivel, &hash, &prefijo, &c.LlavePublica, &exigir, &ops, &todas, &activo,
		&creado, &actualizado)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(ops), &c.OperacionesSensibles); err != nil {
		return nil, fmt.Errorf("operaciones_sensibles del cliente %s: %w", c.ID, err)
	}
	c.LlaveHash, c.LlavePrefijo, c.ExigirLlave = hash.String, prefijo.String, exigir == 1
	c.TodasLasImpresoras, c.Activo = todas == 1, activo == 1
	c.Creado, c.Actualizado = desdeMs(creado), desdeMs(actualizado)
	return &c, nil
}

func codificarLista(l []string) (string, error) {
	if l == nil {
		l = []string{}
	}
	b, err := json.Marshal(l)
	return string(b), err
}

func escanearTextos(filas *sql.Rows) ([]string, error) {
	defer filas.Close()
	var lista []string
	for filas.Next() {
		var s string
		if err := filas.Scan(&s); err != nil {
			return nil, err
		}
		lista = append(lista, s)
	}
	return lista, filas.Err()
}
