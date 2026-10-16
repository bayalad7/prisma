package seguridad

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"regexp"
	"sync"
	"time"
)

// Una llave de cliente tiene la forma pk_<id>_<secreto>:
//   - pk_<id> es el prefijo público: se guarda en claro para encontrar al cliente y mostrarlo en el panel;
//   - <secreto> son 32 bytes aleatorios en base64url.
//
// Solo se guarda el hash argon2id de la llave completa.
var formatoLlave = regexp.MustCompile(`^(pk_[a-z2-7]{10})_[A-Za-z0-9_-]{43}$`)

// Parámetros argon2id para llaves (OWASP: 19 MiB, 2 pasadas, 1 hilo). Son más livianos que los de
// la contraseña del panel porque se verifican en las peticiones de impresión; la caché evita
// repetir el cálculo para la misma llave.
const (
	llaveMemoriaKiB = 19 * 1024
	llavePasadas    = 2
	llaveHilos      = 1
)

const alfabetoPrefijo = "abcdefghijklmnopqrstuvwxyz234567"

// GenerarLlave crea una llave de cliente nueva y su prefijo público.
func GenerarLlave() (llave, prefijo string, err error) {
	id := make([]byte, 10)
	if _, err := rand.Read(id); err != nil {
		return "", "", err
	}
	for i, b := range id {
		id[i] = alfabetoPrefijo[int(b)%len(alfabetoPrefijo)] // 256 es múltiplo de 32: sin sesgo
	}
	secreto := make([]byte, 32)
	if _, err := rand.Read(secreto); err != nil {
		return "", "", err
	}
	prefijo = "pk_" + string(id)
	return prefijo + "_" + base64.RawURLEncoding.EncodeToString(secreto), prefijo, nil
}

// PrefijoDeLlave valida el formato y devuelve el prefijo público de la llave.
func PrefijoDeLlave(llave string) (string, bool) {
	m := formatoLlave.FindStringSubmatch(llave)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// HashLlave devuelve el hash argon2id (PHC) de una llave.
func HashLlave(llave string) (string, error) {
	return hashArgon2id(llave, llaveMemoriaKiB, llavePasadas, llaveHilos)
}

// cacheLlaves recuerda el resultado de verificar (llave, hash) para no recalcular argon2id en cada
// petición. La clave incluye el hash guardado, así que regenerar la llave de un cliente invalida
// las entradas viejas sin hacer nada más. Los resultados negativos también se recuerdan: repetir
// una llave falsa no vuelve a costar un cálculo.
type cacheLlaves struct {
	mu       sync.Mutex
	entradas map[[32]byte]entradaLlave
	max      int
	ttl      time.Duration
	ahora    func() time.Time
}

type entradaLlave struct {
	valida bool
	vence  time.Time
}

func nuevaCacheLlaves(max int, ttl time.Duration, ahora func() time.Time) *cacheLlaves {
	return &cacheLlaves{entradas: map[[32]byte]entradaLlave{}, max: max, ttl: ttl, ahora: ahora}
}

func claveCache(llave, phc string) [32]byte {
	return sha256.Sum256([]byte(llave + "\x00" + phc))
}

func (c *cacheLlaves) buscar(llave, phc string) (valida, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entradas[claveCache(llave, phc)]
	if !ok || c.ahora().After(e.vence) {
		return false, false
	}
	return e.valida, true
}

func (c *cacheLlaves) guardar(llave, phc string, valida bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entradas) >= c.max {
		// Sin LRU: al llenarse se descartan las vencidas y, si no alcanza, todo. Es raro y barato.
		ahora := c.ahora()
		for k, e := range c.entradas {
			if ahora.After(e.vence) {
				delete(c.entradas, k)
			}
		}
		if len(c.entradas) >= c.max {
			clear(c.entradas)
		}
	}
	c.entradas[claveCache(llave, phc)] = entradaLlave{valida: valida, vence: c.ahora().Add(c.ttl)}
}
