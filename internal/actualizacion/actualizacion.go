// Package actualizacion busca versiones nuevas de PRISMA en GitHub Releases, descarga el archivo
// de esta plataforma y lo verifica (firma Ed25519 de checksums.txt y SHA-256) antes de instalarlo.
package actualizacion

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// LlavePublica es la llave Ed25519 (base64 de 32 bytes) con la que se firman los checksums de
// cada versión. Se inyecta al compilar desde empaquetado/llave-releases.pub; sin ella,
// «prisma actualizar» se niega a instalar nada.
var LlavePublica = ""

// URLAPI es la consulta de la última versión publicada (sin versiones preliminares). Es una
// variable para que las pruebas de instalación puedan compilar un binario contra un servidor propio.
var URLAPI = "https://api.github.com/repos/bayalad7/prisma/releases/latest"

// Nombres de los archivos de cada versión.
const (
	ArchivoChecksums = "checksums.txt"
	ArchivoFirma     = "checksums.txt.sig"
	MaxBytesBinario  = 200 << 20
)

// Version es una versión publicada.
type Version struct {
	Etiqueta string    // v1.2.3
	Numero   string    // 1.2.3
	URL      string    // página de la versión
	Fecha    time.Time // publicación
	Archivos map[string]string
}

// Cliente consulta y descarga versiones.
type Cliente struct {
	HTTP   *http.Client
	URLAPI string
	Llave  string // base64; por defecto LlavePublica
	Actual string // versión instalada, para el User-Agent
}

// NuevoCliente con los valores por defecto.
func NuevoCliente(actual string) *Cliente {
	return &Cliente{HTTP: &http.Client{Timeout: 60 * time.Second}, URLAPI: URLAPI, Llave: LlavePublica, Actual: actual}
}

func (c *Cliente) pedir(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "prisma/"+c.Actual)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("no se pudo consultar %s: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound && url == c.URLAPI {
			return nil, ErrSinVersiones
		}
		return nil, fmt.Errorf("%s respondió %s", url, resp.Status)
	}
	return resp, nil
}

// Ultima devuelve la última versión publicada.
func (c *Cliente) Ultima(ctx context.Context) (*Version, error) {
	resp, err := c.pedir(ctx, c.URLAPI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var r struct {
		TagName     string    `json:"tag_name"`
		HTMLURL     string    `json:"html_url"`
		PublishedAt time.Time `json:"published_at"`
		Assets      []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&r); err != nil {
		return nil, fmt.Errorf("respuesta de GitHub inválida: %w", err)
	}
	v := &Version{Etiqueta: r.TagName, Numero: strings.TrimPrefix(r.TagName, "v"), URL: r.HTMLURL, Fecha: r.PublishedAt, Archivos: map[string]string{}}
	if _, ok := semver(v.Numero); !ok {
		return nil, fmt.Errorf("la última versión publicada tiene una etiqueta no válida: %q", r.TagName)
	}
	for _, a := range r.Assets {
		v.Archivos[a.Name] = a.URL
	}
	return v, nil
}

// semver lee «1.2.3» (con un sufijo -algo opcional, que cuenta como anterior a la versión sin él).
func semver(s string) ([4]int, bool) {
	var v [4]int
	base, pre, hayPre := strings.Cut(s, "-")
	partes := strings.Split(base, ".")
	if len(partes) != 3 {
		return v, false
	}
	for i, p := range partes {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return v, false
		}
		v[i] = n
	}
	v[3] = 1
	if hayPre && pre != "" {
		v[3] = 0
	}
	return v, true
}

// EsNueva indica si disponible es posterior a actual. Una compilación de desarrollo (que no es
// una versión, como 0.0.0-dev o un hash de git) nunca se considera desactualizada.
func EsNueva(actual, disponible string) bool {
	a, okA := semver(strings.TrimPrefix(actual, "v"))
	d, okD := semver(strings.TrimPrefix(disponible, "v"))
	if !okA || !okD || a == ([4]int{0, 0, 0, 0}) {
		return false
	}
	for i := range a {
		if a[i] != d[i] {
			return d[i] > a[i]
		}
	}
	return false
}

// Desarrollo indica si la versión actual es una compilación de desarrollo.
func Desarrollo(actual string) bool {
	v, ok := semver(strings.TrimPrefix(actual, "v"))
	return !ok || v[0] == 0 && v[1] == 0 && v[2] == 0
}

// ArchivoBinario es el nombre del binario de esta plataforma en cada versión.
func ArchivoBinario(goos, goarch string) string {
	if goos == "windows" {
		return fmt.Sprintf("prisma-%s-%s.exe", goos, goarch)
	}
	return fmt.Sprintf("prisma-%s-%s", goos, goarch)
}

// ArchivoDeb es el nombre del paquete .deb de una versión.
func ArchivoDeb(numero, goarch string) string {
	return fmt.Sprintf("prisma_%s_%s.deb", numero, goarch)
}

// ErrSinVersiones: el proyecto todavía no publicó ninguna versión.
var ErrSinVersiones = errors.New("todavía no hay versiones publicadas de PRISMA")

// ErrSinLlave: esta compilación no puede verificar versiones.
var ErrSinLlave = errors.New("esta compilación de PRISMA no tiene la llave para verificar actualizaciones; " +
	"descarga el instalador desde la página de versiones")

// Checksums descarga checksums.txt, verifica su firma y devuelve nombre → SHA-256.
func (c *Cliente) Checksums(ctx context.Context, v *Version) (map[string][]byte, error) {
	llave, err := base64.StdEncoding.DecodeString(c.Llave)
	if c.Llave == "" || err != nil || len(llave) != ed25519.PublicKeySize {
		return nil, ErrSinLlave
	}
	contenido, err := c.descargarTexto(ctx, v, ArchivoChecksums, 1<<20)
	if err != nil {
		return nil, err
	}
	firmaB64, err := c.descargarTexto(ctx, v, ArchivoFirma, 4096)
	if err != nil {
		return nil, err
	}
	firma, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(firmaB64)))
	if err != nil || !ed25519.Verify(llave, contenido, firma) {
		return nil, fmt.Errorf("la firma de %s no es válida: no se instalará nada", ArchivoChecksums)
	}
	return LeerChecksums(contenido)
}

// LeerChecksums interpreta el formato de sha256sum: «<hex>  <nombre>» por línea.
func LeerChecksums(contenido []byte) (map[string][]byte, error) {
	sumas := map[string][]byte{}
	s := bufio.NewScanner(bytes.NewReader(contenido))
	for s.Scan() {
		linea := strings.TrimSpace(s.Text())
		if linea == "" {
			continue
		}
		hexa, nombre, ok := strings.Cut(linea, " ")
		suma, err := hex.DecodeString(hexa)
		if !ok || err != nil || len(suma) != sha256.Size {
			return nil, fmt.Errorf("%s tiene una línea inválida: %q", ArchivoChecksums, linea)
		}
		sumas[strings.TrimLeft(strings.TrimSpace(nombre), "*")] = suma
	}
	return sumas, s.Err()
}

func (c *Cliente) descargarTexto(ctx context.Context, v *Version, nombre string, max int64) ([]byte, error) {
	url, ok := v.Archivos[nombre]
	if !ok {
		return nil, fmt.Errorf("la versión %s no publica %s", v.Etiqueta, nombre)
	}
	resp, err := c.pedir(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, max))
}

// Descargar guarda el archivo nombre de la versión en destino y comprueba su SHA-256. Si no
// coincide, borra el archivo.
func (c *Cliente) Descargar(ctx context.Context, v *Version, nombre string, sumas map[string][]byte, destino string) error {
	esperada, ok := sumas[nombre]
	if !ok {
		return fmt.Errorf("%s no incluye %s", ArchivoChecksums, nombre)
	}
	url, ok := v.Archivos[nombre]
	if !ok {
		return fmt.Errorf("la versión %s no publica %s para esta plataforma (%s/%s)", v.Etiqueta, nombre, runtime.GOOS, runtime.GOARCH)
	}
	resp, err := c.pedir(ctx, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	f, err := os.OpenFile(destino, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	h := sha256.New()
	_, err = io.Copy(io.MultiWriter(f, h), io.LimitReader(resp.Body, MaxBytesBinario))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && !bytes.Equal(h.Sum(nil), esperada) {
		err = fmt.Errorf("el SHA-256 de %s no coincide con %s: la descarga está dañada o fue alterada", nombre, ArchivoChecksums)
	}
	if err != nil {
		os.Remove(destino)
	}
	return err
}
