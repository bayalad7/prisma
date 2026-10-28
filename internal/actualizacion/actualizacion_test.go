package actualizacion

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// publicacion simula GitHub Releases: la API y los archivos de una versión.
type publicacion struct {
	srv       *httptest.Server
	llave     string
	archivos  map[string][]byte
	etiqueta  string
	consultas atomic.Int32
	fallar    atomic.Int32 // cuántas consultas más responden 503 (sin red, GitHub caído)
}

func nuevaPublicacion(t *testing.T, etiqueta string) *publicacion {
	t.Helper()
	publica, privada, _ := ed25519.GenerateKey(rand.Reader)
	binario := []byte("binario nuevo de PRISMA " + etiqueta)
	nombre := ArchivoBinario("linux", "amd64")
	suma := sha256.Sum256(binario)
	checksums := []byte(fmt.Sprintf("%s  %s\n%s  otro-archivo\n", hex.EncodeToString(suma[:]), nombre, strings.Repeat("ab", 32)))
	p := &publicacion{
		llave:    base64.StdEncoding.EncodeToString(publica),
		etiqueta: etiqueta,
		archivos: map[string][]byte{
			nombre:           binario,
			ArchivoChecksums: checksums,
			ArchivoFirma:     []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(privada, checksums)) + "\n"),
		},
	}
	p.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest" {
			p.consultas.Add(1)
			if p.fallar.Load() > 0 {
				p.fallar.Add(-1)
				http.Error(w, "no disponible", http.StatusServiceUnavailable)
				return
			}
			if !strings.HasPrefix(r.Header.Get("User-Agent"), "prisma/") {
				http.Error(w, "sin user agent", http.StatusForbidden)
				return
			}
			var assets []map[string]string
			for n := range p.archivos {
				assets = append(assets, map[string]string{"name": n, "browser_download_url": p.srv.URL + "/descargas/" + n})
			}
			json.NewEncoder(w).Encode(map[string]any{"tag_name": p.etiqueta, "html_url": "https://github.com/x/releases/" + p.etiqueta,
				"published_at": "2026-10-01T12:00:00Z", "assets": assets})
			return
		}
		datos, ok := p.archivos[strings.TrimPrefix(r.URL.Path, "/descargas/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(datos)
	}))
	t.Cleanup(p.srv.Close)
	return p
}

func (p *publicacion) cliente() *Cliente {
	c := NuevoCliente("1.0.0")
	c.URLAPI, c.Llave = p.srv.URL+"/latest", p.llave
	return c
}

func TestDescargaVerificada(t *testing.T) {
	p := nuevaPublicacion(t, "v1.2.0")
	ctx := context.Background()
	c := p.cliente()
	v, err := c.Ultima(ctx)
	if err != nil || v.Numero != "1.2.0" || v.URL == "" || v.Fecha.IsZero() {
		t.Fatalf("Ultima: %+v, %v", v, err)
	}
	sumas, err := c.Checksums(ctx, v)
	if err != nil || len(sumas) != 2 {
		t.Fatalf("Checksums: %v, %v", sumas, err)
	}
	destino := filepath.Join(t.TempDir(), "prisma.nuevo")
	nombre := ArchivoBinario("linux", "amd64")
	if err := c.Descargar(ctx, v, nombre, sumas, destino); err != nil {
		t.Fatal(err)
	}
	if datos, _ := os.ReadFile(destino); string(datos) != "binario nuevo de PRISMA v1.2.0" {
		t.Fatalf("contenido %q", datos)
	}

	// Un binario alterado no se acepta y no queda en disco.
	p.archivos[nombre] = []byte("binario alterado")
	if err := c.Descargar(ctx, v, nombre, sumas, destino+"2"); err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("binario alterado: %v", err)
	}
	if _, err := os.Stat(destino + "2"); !os.IsNotExist(err) {
		t.Fatal("el archivo alterado quedó en disco")
	}
	if err := c.Descargar(ctx, v, "no-publicado", sumas, destino+"3"); err == nil {
		t.Fatal("un archivo sin checksum no debe descargarse")
	}
}

func TestFirmaDeChecksums(t *testing.T) {
	p := nuevaPublicacion(t, "v1.2.0")
	ctx := context.Background()
	c := p.cliente()
	v, _ := c.Ultima(ctx)

	// checksums.txt alterado después de firmarlo.
	original := p.archivos[ArchivoChecksums]
	p.archivos[ArchivoChecksums] = append([]byte(strings.Repeat("00", 32)+"  prisma-linux-amd64\n"), original...)
	if _, err := c.Checksums(ctx, v); err == nil || !strings.Contains(err.Error(), "firma") {
		t.Fatalf("checksums alterados: %v", err)
	}
	p.archivos[ArchivoChecksums] = original

	// Firmado con otra llave.
	otra, _, _ := ed25519.GenerateKey(rand.Reader)
	c.Llave = base64.StdEncoding.EncodeToString(otra)
	if _, err := c.Checksums(ctx, v); err == nil {
		t.Fatal("una firma de otra llave no debe aceptarse")
	}

	// Sin versiones publicadas, GitHub responde 404.
	sinVersiones := NuevoCliente("1.0.0")
	sinVersiones.URLAPI = p.srv.URL + "/no-hay"
	if _, err := sinVersiones.Ultima(ctx); err != ErrSinVersiones {
		t.Fatalf("sin versiones: %v", err)
	}

	// Una compilación sin llave no instala nada.
	c.Llave = ""
	if _, err := c.Checksums(ctx, v); err != ErrSinLlave {
		t.Fatalf("sin llave: %v", err)
	}
}

func TestEsNueva(t *testing.T) {
	casos := []struct {
		actual, disponible string
		nueva              bool
	}{
		{"1.0.0", "1.0.1", true},
		{"1.0.0", "1.1.0", true},
		{"1.9.9", "2.0.0", true},
		{"1.10.0", "1.9.0", false},
		{"1.0.0", "1.0.0", false},
		{"v1.0.0", "v1.0.1", true},
		{"1.0.0-rc.1", "1.0.0", true},
		{"1.0.0", "1.0.0-rc.1", false},
		{"0.0.0-dev", "9.9.9", false}, // desarrollo
		{"dd98d46-dirty", "1.0.0", false},
		{"1.0.0", "basura", false},
	}
	for _, c := range casos {
		if got := EsNueva(c.actual, c.disponible); got != c.nueva {
			t.Errorf("EsNueva(%q, %q) = %v", c.actual, c.disponible, got)
		}
	}
	if !Desarrollo("0.0.0-dev") || !Desarrollo("dd98d46") || Desarrollo("1.0.0") {
		t.Error("Desarrollo")
	}
}

func TestLeerChecksums(t *testing.T) {
	s := strings.Repeat("ab", 32)
	sumas, err := LeerChecksums([]byte(s + "  a.deb\n" + s + " *b.exe\n\n"))
	if err != nil || len(sumas["a.deb"]) != 32 || len(sumas["b.exe"]) != 32 {
		t.Fatalf("%v, %v", sumas, err)
	}
	if _, err := LeerChecksums([]byte("zz  a\n")); err == nil {
		t.Fatal("hex inválido")
	}
}

func TestComprobador(t *testing.T) {
	p := nuevaPublicacion(t, "v1.2.0")
	aviso := &Aviso{}
	var activo atomic.Bool
	activo.Store(true)
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	c := &Comprobador{Cliente: p.cliente(), Actual: "1.0.0", Aviso: aviso, Activo: func(context.Context) bool { return activo.Load() },
		Inicial: time.Millisecond, Revisar: time.Millisecond, Cada: 20 * time.Millisecond, Log: slog.New(slog.DiscardHandler)}
	go c.Correr(ctx)
	esperar(t, func() bool { return aviso.Disponible() != nil })
	if aviso.Disponible().Numero != "1.2.0" {
		t.Fatalf("aviso %+v", aviso.Disponible())
	}
	// Desactivado en el panel: deja de consultar y quita el aviso.
	activo.Store(false)
	esperar(t, func() bool { return aviso.Disponible() == nil })
	antes := p.consultas.Load()
	time.Sleep(80 * time.Millisecond)
	if p.consultas.Load() != antes {
		t.Fatal("siguió consultando con el aviso desactivado")
	}

	// Una compilación de desarrollo nunca consulta.
	var nilAviso *Aviso
	if nilAviso.Disponible() != nil {
		t.Fatal("aviso nil")
	}
	dev := &Comprobador{Cliente: p.cliente(), Actual: "0.0.0-dev", Aviso: &Aviso{}, Activo: func(context.Context) bool { return true }, Inicial: time.Millisecond, Cada: time.Millisecond, Log: slog.New(slog.DiscardHandler)}
	antes = p.consultas.Load()
	dev.Correr(ctx) // vuelve de inmediato
	if p.consultas.Load() != antes {
		t.Fatal("una compilación de desarrollo consultó GitHub")
	}
}

func esperar(t *testing.T, cond func() bool) {
	t.Helper()
	limite := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(limite) {
			t.Fatal("tiempo agotado")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestReemplazar(t *testing.T) {
	dir := t.TempDir()
	actual, nuevo := filepath.Join(dir, "prisma.exe"), filepath.Join(dir, "prisma.nuevo")
	os.WriteFile(actual, []byte("viejo"), 0o755)
	os.WriteFile(nuevo, []byte("nuevo"), 0o755)
	if err := Reemplazar(nuevo, actual); err != nil {
		t.Fatal(err)
	}
	if datos, _ := os.ReadFile(actual); string(datos) != "nuevo" {
		t.Fatalf("contenido %q", datos)
	}
	if _, err := os.Stat(nuevo); !os.IsNotExist(err) {
		t.Fatal("el archivo nuevo debe moverse, no copiarse")
	}
	if err := Escribible(dir); err != nil {
		t.Fatal(err)
	}
}

// La llave del instalador de Linux debe ser la misma que se embebe en el binario.
func TestLlaveDelInstaladorLinux(t *testing.T) {
	pub, err := os.ReadFile("../../empaquetado/llave-releases.pub")
	if err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile("../../empaquetado/linux/instalar.sh")
	if err != nil {
		t.Fatal(err)
	}
	if want := `LLAVE_RELEASES="` + strings.TrimSpace(string(pub)) + `"`; !strings.Contains(string(script), want) {
		t.Fatalf("empaquetado/linux/instalar.sh debe contener %s", want)
	}
}
