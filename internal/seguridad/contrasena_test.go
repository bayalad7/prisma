package seguridad

import (
	"encoding/base64"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/crypto/argon2"
)

func TestHashYVerificarContrasena(t *testing.T) {
	hash, err := HashContrasena("correcta horse")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=65536,t=3,p=4$") {
		t.Errorf("formato PHC inesperado: %s", hash)
	}
	if ok, err := VerificarContrasena("correcta horse", hash); err != nil || !ok {
		t.Errorf("la contraseña correcta no verificó: %v", err)
	}
	if ok, _ := VerificarContrasena("incorrecta", hash); ok {
		t.Error("una contraseña incorrecta verificó")
	}
	otro, _ := HashContrasena("correcta horse")
	if otro == hash {
		t.Error("dos hashes de la misma contraseña deberían tener sal distinta")
	}
}

func TestVerificarRespetaParametrosGuardados(t *testing.T) {
	// Un hash creado con parámetros distintos a los actuales (m=8, t=1, p=1) debe seguir verificando.
	sal := []byte("sal-de-prueba-16")
	b64 := base64.RawStdEncoding
	hash := "$argon2id$v=19$m=8,t=1,p=1$" + b64.EncodeToString(sal) + "$" +
		b64.EncodeToString(argon2.IDKey([]byte("x"), sal, 1, 8, 1, 32))
	if ok, err := VerificarContrasena("x", hash); err != nil || !ok {
		t.Errorf("no verificó un hash con parámetros guardados distintos (ok=%v, err=%v)", ok, err)
	}
	// Cambiar los parámetros de un hash existente hace que deje de coincidir.
	propio, err := HashContrasena("x")
	if err != nil {
		t.Fatal(err)
	}
	partes := strings.Split(propio, "$")
	partes[3] = "m=8,t=1,p=1"
	if ok, err := VerificarContrasena("x", strings.Join(partes, "$")); err != nil || ok {
		t.Errorf("con otros parámetros el hash no debe coincidir (ok=%v, err=%v)", ok, err)
	}
}

func TestVerificarHashInvalido(t *testing.T) {
	for _, h := range []string{"", "texto", "$argon2i$v=19$m=8,t=1,p=1$YQ$YQ", "$argon2id$v=19$m=0,t=1,p=1$YQ$YQ",
		"$argon2id$v=19$m=8,t=1,p=1$***$YQ", "$argon2id$v=18$m=8,t=1,p=1$YQ$YQ"} {
		if _, err := VerificarContrasena("x", h); err == nil {
			t.Errorf("se esperaba error para %q", h)
		}
	}
}

func TestGenerarContrasena(t *testing.T) {
	formato := regexp.MustCompile(`^[` + alfabetoContrasena + `]{5}(-[` + alfabetoContrasena + `]{5}){3}$`)
	vistas := map[string]bool{}
	for i := 0; i < 50; i++ {
		c, err := GenerarContrasena()
		if err != nil {
			t.Fatal(err)
		}
		if !formato.MatchString(c) {
			t.Fatalf("formato inesperado: %q", c)
		}
		if vistas[c] {
			t.Fatalf("contraseña repetida: %q", c)
		}
		vistas[c] = true
	}
}
