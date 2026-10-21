package dev

import (
	"bytes"
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type salidaSegura struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *salidaSegura) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *salidaSegura) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func TestImpresoraVirtual(t *testing.T) {
	perfil, _ := PerfilPrueba(58)
	var salida salidaSegura
	v := &ImpresoraVirtual{Perfil: perfil, Directorio: t.TempDir(), Salida: &salida}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	terminado := make(chan error, 1)
	go func() { terminado <- v.Servir(ctx, ln) }()

	enviar := func(datos []byte) {
		c, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		c.Write(datos)
		c.Close()
	}
	enviar(nil) // prueba de conexión: no cuenta como trabajo
	enviar([]byte("\x1b@\x1ba\x01Hola virtual\n\x1dVA\x00"))

	limite := time.Now().Add(5 * time.Second)
	for v.Recibidos() < 1 || !strings.Contains(salida.String(), "corte total") {
		if time.Now().After(limite) {
			t.Fatalf("no se recibió el trabajo:\n%s", salida.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancelar()
	if err := <-terminado; err != nil {
		t.Fatal(err)
	}

	out := salida.String()
	for _, esperado := range []string{"Trabajo 1", "inicializar impresora", "alineación centro", "Hola virtual"} {
		if !strings.Contains(out, esperado) {
			t.Errorf("falta %q en la salida:\n%s", esperado, out)
		}
	}
	if v.Recibidos() != 1 {
		t.Errorf("la conexión vacía no debe contar: %d", v.Recibidos())
	}
	bins, _ := filepath.Glob(filepath.Join(v.Directorio, "*.bin"))
	if len(bins) != 1 {
		t.Fatalf("archivos: %v", bins)
	}
	if datos, _ := os.ReadFile(bins[0]); !bytes.Contains(datos, []byte("Hola virtual")) {
		t.Errorf("el .bin no tiene los datos: %q", datos)
	}
}

func TestConsultasEstado(t *testing.T) {
	casos := map[string]int{"": 0, "\x10\x04": 0, "\x10\x04\x01": 1, "a\x10\x04\x04b\x10\x04\x02": 2, "\x10\x04\x05": 0}
	for d, quiero := range casos {
		if got := consultasEstado([]byte(d)); got != quiero {
			t.Errorf("%q: %d, quiero %d", d, got, quiero)
		}
	}
	if traza := Traza([]byte("\x1b@\x10\x04\x01")); !strings.Contains(traza, "consultar estado 1") {
		t.Errorf("la traza no reconoce DLE EOT:\n%s", traza)
	}
}
