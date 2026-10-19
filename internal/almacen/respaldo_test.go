package almacen

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRespaldarYRestaurar(t *testing.T) {
	dir := t.TempDir()
	a, err := Abrir(ctx, filepath.Join(dir, "prisma.db"))
	exigirSinError(t, err)
	defer a.Cerrar()
	exigirSinError(t, a.Ajustes.Establecer(ctx, "x", "antes"))
	exigirSinError(t, a.Impresoras.Crear(ctx, impresoraDePrueba("caja")))

	respaldo := filepath.Join(dir, "respaldo.db")
	exigirSinError(t, a.Respaldar(ctx, respaldo))
	exigirSinError(t, ValidarRespaldo(ctx, respaldo))

	// Cambios después del respaldo: la restauración los deshace.
	exigirSinError(t, a.Ajustes.Establecer(ctx, "x", "despues"))
	exigirSinError(t, a.Impresoras.Crear(ctx, impresoraDePrueba("cocina")))
	exigirSinError(t, a.Restaurar(ctx, respaldo))
	if v, _, _ := a.Ajustes.Obtener(ctx, "x"); v != "antes" {
		t.Fatalf("ajuste tras restaurar: %q", v)
	}
	lista, _ := a.Impresoras.Listar(ctx)
	if len(lista) != 1 || lista[0].Alias != "caja" {
		t.Fatalf("impresoras tras restaurar: %+v", lista)
	}
	// Sigue funcionando para escribir.
	exigirSinError(t, a.Ajustes.Establecer(ctx, "y", "1"))
}

func TestRestaurarRechazaArchivosInvalidos(t *testing.T) {
	dir := t.TempDir()
	a, err := Abrir(ctx, filepath.Join(dir, "prisma.db"))
	exigirSinError(t, err)
	defer a.Cerrar()
	exigirSinError(t, a.Ajustes.Establecer(ctx, "x", "intacto"))

	texto := filepath.Join(dir, "no-es-db.db")
	os.WriteFile(texto, []byte("hola"), 0o600)
	if err := a.Restaurar(ctx, texto); !errors.Is(err, ErrRespaldoInvalido) {
		t.Fatalf("archivo de texto: %v", err)
	}
	// Otra base SQLite que no es de PRISMA.
	ajena, err := Abrir(ctx, filepath.Join(dir, "otra", "x.db"))
	exigirSinError(t, err)
	ajena.bd().ExecContext(ctx, "PRAGMA user_version = 99")
	ajena.Cerrar()
	if err := a.Restaurar(ctx, filepath.Join(dir, "otra", "x.db")); !errors.Is(err, ErrRespaldoInvalido) {
		t.Fatalf("esquema desconocido: %v", err)
	}
	if v, _, _ := a.Ajustes.Obtener(ctx, "x"); v != "intacto" {
		t.Fatal("un respaldo inválido no debe tocar la base en uso")
	}
	m, _ := AbrirMemoria(ctx)
	defer m.Cerrar()
	if err := m.Restaurar(ctx, texto); err == nil {
		t.Fatal("una base en memoria no se restaura")
	}
}

func TestTrabajosPorFechaYConteo(t *testing.T) {
	a, r := nuevoAlmacen(t)
	inicio := r.t
	for i, est := range []string{TrabajoCompletado, TrabajoFallido, TrabajoCompletado} {
		tr := trabajoDePrueba(string(rune('a'+i)), "cli_a", "caja")
		exigirSinError(t, a.Trabajos.Crear(ctx, tr))
		exigirSinError(t, a.Trabajos.Finalizar(ctx, tr.ID, est, "", nil, 0))
		r.avanzar(time.Hour)
	}
	lista, _ := a.Trabajos.Listar(ctx, FiltroTrabajos{Desde: inicio.Add(30 * time.Minute), Hasta: inicio.Add(90 * time.Minute)})
	if len(lista) != 1 || lista[0].ID != "b" {
		t.Fatalf("filtro por fecha: %+v", lista)
	}
	conteo, err := a.Trabajos.Contar(ctx, inicio)
	exigirSinError(t, err)
	if conteo[TrabajoCompletado] != 2 || conteo[TrabajoFallido] != 1 {
		t.Fatalf("conteo: %v", conteo)
	}
}
