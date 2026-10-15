package main

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/seguridad"
)

// configDePrueba escribe un config.toml que guarda los datos en un directorio temporal.
func configDePrueba(t *testing.T) (rutaConfig, dirDatos string) {
	t.Helper()
	dir := t.TempDir()
	dirDatos = filepath.Join(dir, "datos")
	rutaConfig = filepath.Join(dir, "config.toml")
	contenido := "[servidor]\npuerto = 9443\n[datos]\nruta = '" + dirDatos + "'\n"
	if err := os.WriteFile(rutaConfig, []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}
	return rutaConfig, dirDatos
}

// bufferSeguro permite leer la bitácora mientras ejecutar escribe desde otra goroutine.
type bufferSeguro struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *bufferSeguro) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *bufferSeguro) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

func correr(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := nuevoComandoRaiz()
	var salida bytes.Buffer
	cmd.SetOut(&salida)
	cmd.SetErr(&salida)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return salida.String(), err
}

func TestComandoVersion(t *testing.T) {
	salida, err := correr(t, "version")
	if err != nil || !strings.HasPrefix(salida, "PRISMA ") || !strings.Contains(salida, "contrato de documento: [1]") {
		t.Fatalf("salida %q, error %v", salida, err)
	}
}

func TestComandoConfigValidar(t *testing.T) {
	ruta, _ := configDePrueba(t)
	salida, err := correr(t, "config", "validar", "--config", ruta)
	if err != nil || !strings.Contains(salida, "La configuración es válida") || !strings.Contains(salida, "9443") {
		t.Fatalf("salida %q, error %v", salida, err)
	}

	mala := filepath.Join(t.TempDir(), "config.toml")
	os.WriteFile(mala, []byte("[servidor]\npuerto = 0\n"), 0o600)
	if _, err := correr(t, "config", "validar", "--config", mala); err == nil || !strings.Contains(err.Error(), "servidor.puerto") {
		t.Fatalf("esperaba error de servidor.puerto, obtuve %v", err)
	}
}

func TestEjecutarCreaLaBaseDeDatos(t *testing.T) {
	ruta, dirDatos := configDePrueba(t)
	ctx, cancelar := context.WithCancel(context.Background())
	var bitacora bufferSeguro
	listo := make(chan error, 1)
	go func() { listo <- ejecutar(ctx, ruta, slog.New(slog.NewJSONHandler(&bitacora, nil))) }()

	rutaBD := filepath.Join(dirDatos, "prisma.db")
	limite := time.Now().Add(10 * time.Second)
	for !strings.Contains(bitacora.String(), "todavía no está implementado") {
		if time.Now().After(limite) {
			t.Fatalf("ejecutar no terminó de arrancar; bitácora:\n%s", bitacora.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancelar()
	if err := <-listo; err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(rutaBD); err != nil {
		t.Fatalf("no se creó la base de datos: %v", err)
	}
	if !strings.Contains(bitacora.String(), `"msg":"PRISMA detenido"`) {
		t.Errorf("falta el mensaje de cierre en la bitácora:\n%s", bitacora.String())
	}
}

func TestRestablecerContrasena(t *testing.T) {
	ruta, dirDatos := configDePrueba(t)
	salida, err := correr(t, "admin", "restablecer-contrasena", "--config", ruta)
	if err != nil {
		t.Fatal(err)
	}
	var contrasena string
	for _, linea := range strings.Split(salida, "\n") {
		if strings.HasPrefix(linea, "    ") {
			contrasena = strings.TrimSpace(linea)
		}
	}
	if contrasena == "" {
		t.Fatalf("no se mostró la contraseña:\n%s", salida)
	}

	alm, err := almacen.Abrir(context.Background(), filepath.Join(dirDatos, "prisma.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer alm.Cerrar()
	hash, _, _ := alm.Ajustes.Obtener(context.Background(), almacen.AjusteContrasenaAdmin)
	if ok, err := seguridad.VerificarContrasena(contrasena, hash); err != nil || !ok {
		t.Fatalf("la contraseña mostrada no coincide con el hash guardado (%v)", err)
	}
	if v, _, _ := alm.Ajustes.Obtener(context.Background(), almacen.AjusteDebeCambiar); v != "1" {
		t.Error("debe_cambiar no quedó activo")
	}
	if strings.Contains(salida, hash) {
		t.Error("la salida no debe incluir el hash")
	}
}
