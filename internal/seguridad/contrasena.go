package seguridad

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Parámetros argon2id (RFC 9106, segunda recomendación): 64 MiB, 3 pasadas, 4 hilos.
// Solo se usan para la contraseña del panel, que se verifica pocas veces.
const (
	argonMemoriaKiB = 64 * 1024
	argonPasadas    = 3
	argonHilos      = 4
	argonSal        = 16
	argonLargo      = 32
)

var errHashInvalido = errors.New("hash de contraseña con formato inválido")

// HashContrasena devuelve el hash argon2id en formato PHC:
// $argon2id$v=19$m=65536,t=3,p=4$<sal>$<hash>
func HashContrasena(contrasena string) (string, error) {
	return hashArgon2id(contrasena, argonMemoriaKiB, argonPasadas, argonHilos)
}

func hashArgon2id(secreto string, memoriaKiB, pasadas uint32, hilos uint8) (string, error) {
	sal := make([]byte, argonSal)
	if _, err := rand.Read(sal); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(secreto), sal, pasadas, memoriaKiB, hilos, argonLargo)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memoriaKiB, pasadas, hilos, b64.EncodeToString(sal), b64.EncodeToString(hash)), nil
}

// VerificarContrasena compara en tiempo constante. Respeta los parámetros guardados en el hash,
// así que los hashes antiguos siguen funcionando si cambian los valores por defecto.
func VerificarContrasena(contrasena, phc string) (bool, error) {
	partes := strings.Split(phc, "$")
	if len(partes) != 6 || partes[1] != "argon2id" {
		return false, errHashInvalido
	}
	var version int
	if _, err := fmt.Sscanf(partes[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, errHashInvalido
	}
	var memoria, pasadas uint32
	var hilos uint8
	if _, err := fmt.Sscanf(partes[3], "m=%d,t=%d,p=%d", &memoria, &pasadas, &hilos); err != nil ||
		memoria == 0 || pasadas == 0 || hilos == 0 {
		return false, errHashInvalido
	}
	b64 := base64.RawStdEncoding
	sal, err := b64.DecodeString(partes[4])
	if err != nil {
		return false, errHashInvalido
	}
	esperado, err := b64.DecodeString(partes[5])
	if err != nil || len(esperado) == 0 {
		return false, errHashInvalido
	}
	obtenido := argon2.IDKey([]byte(contrasena), sal, pasadas, memoria, hilos, uint32(len(esperado)))
	return subtle.ConstantTimeCompare(obtenido, esperado) == 1, nil
}

// alfabetoContrasena evita caracteres ambiguos (0/O, 1/l/I).
const alfabetoContrasena = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// GenerarContrasena devuelve una contraseña aleatoria de 4 grupos de 5 caracteres (~117 bits),
// fácil de dictar, por ejemplo "k7Rmq-Xa3Vt-9pLwe-Hc2Nf".
func GenerarContrasena() (string, error) {
	const grupos, porGrupo = 4, 5
	var b strings.Builder
	aleatorio := make([]byte, grupos*porGrupo)
	if _, err := rand.Read(aleatorio); err != nil {
		return "", err
	}
	// Muestreo por rechazo para no sesgar el alfabeto.
	limite := byte(256 - 256%len(alfabetoContrasena))
	for i := 0; i < grupos*porGrupo; i++ {
		for aleatorio[i] >= limite {
			if _, err := rand.Read(aleatorio[i : i+1]); err != nil {
				return "", err
			}
		}
		if i > 0 && i%porGrupo == 0 {
			b.WriteByte('-')
		}
		b.WriteByte(alfabetoContrasena[int(aleatorio[i])%len(alfabetoContrasena)])
	}
	return b.String(), nil
}
