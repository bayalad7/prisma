package main

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/conector"
	"github.com/bayalad7/prisma/internal/dev"
)

// configDesarrollo es como configDePrueba pero con modo_desarrollo activo.
func configDesarrollo(t *testing.T) (rutaConfig, dirDatos string) {
	t.Helper()
	rutaConfig, dirDatos = configDePrueba(t)
	f, err := os.OpenFile(rutaConfig, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("[dev]\nmodo_desarrollo = true\n")
	f.Close()
	return rutaConfig, dirDatos
}

func TestImpresorasAgregarListarProbarEliminar(t *testing.T) {
	ruta, _ := configDePrueba(t)
	dirSalida := filepath.Join(t.TempDir(), "tickets")
	salida, err := correr(t, "impresoras", "agregar", "--config", ruta, "--alias", "archivo", "--tipo", "archivo",
		"--destino", dirSalida, "--ancho", "58", "--cajon")
	if err != nil || !strings.Contains(salida, "32 columnas") {
		t.Fatalf("agregar: %v\n%s", err, salida)
	}
	if _, err := correr(t, "impresoras", "agregar", "--config", ruta, "--alias", "archivo", "--tipo", "archivo", "--destino", "x"); err == nil ||
		!strings.Contains(err.Error(), "ya existe") {
		t.Fatalf("alias repetido: %v", err)
	}
	for _, malo := range [][]string{{"--alias", "Caja!"}, {"--tipo", "wifi"}, {"--pagina", "UTF8"}, {"--ancho", "76"}} {
		args := []string{"impresoras", "agregar", "--config", ruta, "--alias", "otra", "--tipo", "red", "--destino", "1.2.3.4"}
		if _, err := correr(t, append(args, malo...)...); err == nil {
			t.Errorf("%v debía fallar", malo)
		}
	}

	salida, err = correr(t, "impresoras", "listar", "--config", ruta)
	if err != nil || !strings.Contains(salida, "archivo") || !strings.Contains(salida, "32 / 42") {
		t.Fatalf("listar: %v\n%s", err, salida)
	}
	if salida, err = correr(t, "impresoras", "probar", "archivo", "--config", ruta); err != nil || !strings.Contains(salida, "en línea") {
		t.Fatalf("probar: %v\n%s", err, salida)
	}
	if _, err := correr(t, "impresoras", "probar", "nadie", "--config", ruta); conector.Codigo(err) != conector.ErrImpresoraNoEncontrada {
		t.Fatalf("probar inexistente: %v", err)
	}
	if _, err := correr(t, "impresoras", "eliminar", "archivo", "--config", ruta); err != nil {
		t.Fatal(err)
	}
	if _, err := correr(t, "impresoras", "eliminar", "archivo", "--config", ruta); err == nil {
		t.Fatal("eliminar dos veces debe fallar")
	}
}

func TestDetectarJSON(t *testing.T) {
	salida, err := correr(t, "impresoras", "detectar", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var ds []conector.Detectada
	if err := json.Unmarshal([]byte(salida), &ds); err != nil {
		t.Fatalf("no es JSON: %v\n%s", err, salida)
	}
	if _, err := correr(t, "impresoras", "detectar", "--subred", "no-es-subred"); err == nil {
		t.Fatal("una subred inválida debe fallar")
	}
}

func TestImprimirSoloEnModoDesarrollo(t *testing.T) {
	ruta, _ := configDePrueba(t)
	_, err := correr(t, "imprimir", "--config", ruta, "../../ejemplos/validos/venta-simple.json")
	if err == nil || !strings.Contains(err.Error(), "modo_desarrollo") {
		t.Fatalf("esperaba el rechazo fuera de modo desarrollo: %v", err)
	}
}

func TestImprimirEnLaImpresoraVirtual(t *testing.T) {
	ruta, dirDatos := configDesarrollo(t)
	perfil, _ := dev.PerfilPrueba(80)
	v := &dev.ImpresoraVirtual{Perfil: perfil, Directorio: t.TempDir(), Salida: io.Discard}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	go v.Servir(ctx, ln)

	if _, err := correr(t, "impresoras", "agregar", "--config", ruta, "--alias", "caja", "--tipo", "red", "--destino", ln.Addr().String()); err != nil {
		t.Fatal(err)
	}
	salida, err := correr(t, "imprimir", "--config", ruta, "--impresora", "caja", "../../ejemplos/validos/venta-simple.json")
	if err != nil || !strings.Contains(salida, "Impreso en") {
		t.Fatalf("imprimir: %v\n%s", err, salida)
	}
	limite := time.Now().Add(5 * time.Second)
	for v.Recibidos() < 1 && time.Now().Before(limite) {
		time.Sleep(10 * time.Millisecond)
	}
	if v.Recibidos() != 1 {
		t.Fatal("la impresora virtual no recibió el trabajo")
	}

	// El trabajo quedó registrado como completado.
	alm, err := almacen.Abrir(context.Background(), filepath.Join(dirDatos, "prisma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer alm.Cerrar()
	ts, _ := alm.Trabajos.Listar(context.Background(), almacen.FiltroTrabajos{})
	if len(ts) != 1 || ts[0].Estado != almacen.TrabajoCompletado || ts[0].ImpresoraAlias != "caja" || ts[0].ClienteID != clienteConsola {
		t.Fatalf("trabajos: %+v", ts)
	}
}

func TestImprimirEnImpresoraApagada(t *testing.T) {
	ruta, _ := configDesarrollo(t)
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	dir := ln.Addr().String()
	ln.Close() // nadie escucha
	if _, err := correr(t, "impresoras", "agregar", "--config", ruta, "--alias", "caja", "--tipo", "red", "--destino", dir); err != nil {
		t.Fatal(err)
	}
	_, err := correr(t, "imprimir", "--config", ruta, "../../ejemplos/validos/venta-simple.json")
	if conector.Codigo(err) != conector.ErrImpresoraInalcanzable {
		t.Fatalf("esperaba IMPRESORA_INALCANZABLE, obtuve %v", err)
	}
	var b strings.Builder
	imprimirError(&b, err)
	if !strings.Contains(b.String(), "Sugerencia:") {
		t.Errorf("el error debe mostrar la sugerencia:\n%s", b.String())
	}
}
