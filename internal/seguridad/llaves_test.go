package seguridad

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestGenerarYVerificarLlave(t *testing.T) {
	llave, prefijo, err := GenerarLlave()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(llave, prefijo+"_") || len(prefijo) != 13 {
		t.Fatalf("llave %q, prefijo %q", llave, prefijo)
	}
	if p, ok := PrefijoDeLlave(llave); !ok || p != prefijo {
		t.Fatalf("PrefijoDeLlave = %q, %v", p, ok)
	}
	otra, _, _ := GenerarLlave()
	if otra == llave {
		t.Fatal("dos llaves iguales")
	}
	hash, err := HashLlave(llave)
	if err != nil || !strings.HasPrefix(hash, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("hash %q, %v", hash, err)
	}
	if ok, _ := VerificarContrasena(llave, hash); !ok {
		t.Fatal("la llave correcta no verificó")
	}
	if ok, _ := VerificarContrasena(otra, hash); ok {
		t.Fatal("otra llave verificó")
	}
	for _, mala := range []string{"", "pk_", "pk_abc_def", "sk_aaaaaaaaaa_" + strings.Repeat("A", 43), prefijo + "_corta"} {
		if _, ok := PrefijoDeLlave(mala); ok {
			t.Errorf("%q no debería ser una llave válida", mala)
		}
	}
}

func TestCacheLlaves(t *testing.T) {
	ahora := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	c := nuevaCacheLlaves(3, time.Minute, func() time.Time { return ahora })
	c.guardar("a", "h1", true)
	c.guardar("b", "h1", false)
	if v, ok := c.buscar("a", "h1"); !ok || !v {
		t.Fatal("positivo")
	}
	if v, ok := c.buscar("b", "h1"); !ok || v {
		t.Fatal("negativo")
	}
	// Otro hash (llave regenerada) no usa la entrada vieja.
	if _, ok := c.buscar("a", "h2"); ok {
		t.Fatal("la entrada depende del hash guardado")
	}
	ahora = ahora.Add(2 * time.Minute)
	if _, ok := c.buscar("a", "h1"); ok {
		t.Fatal("la entrada debía vencer")
	}
	c.guardar("c", "h", true)
	c.guardar("d", "h", true) // llena: descarta las vencidas
	if len(c.entradas) > 3 {
		t.Fatalf("la caché creció a %d", len(c.entradas))
	}
}

func TestLimitador(t *testing.T) {
	ahora := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	l := NuevoLimitador(60) // 1 por segundo, ráfaga de 60
	l.Ahora = func() time.Time { return ahora }
	for i := range 60 {
		if ok, _ := l.Permitir("ip"); !ok {
			t.Fatalf("la petición %d debía pasar", i)
		}
	}
	ok, espera := l.Permitir("ip")
	if ok || espera <= 0 || espera > time.Second {
		t.Fatalf("la 61 debía esperar ~1 s: %v %v", ok, espera)
	}
	if ok, _ := l.Permitir("otra-ip"); !ok {
		t.Fatal("cada clave tiene su cubeta")
	}
	ahora = ahora.Add(1 * time.Second)
	if ok, _ := l.Permitir("ip"); !ok {
		t.Fatal("tras un segundo hay una ficha nueva")
	}
	if ok, _ := NuevoLimitador(0).Permitir("x"); !ok {
		t.Fatal("0 desactiva el límite")
	}
}

func TestLlavesPEM(t *testing.T) {
	publica, privada, _ := ed25519.GenerateKey(nil)
	pp, _ := LlavePrivadaPEM(privada)
	up, _ := LlavePublicaPEM(publica)
	p2, err := LeerLlavePrivada(pp)
	if err != nil || !p2.Equal(privada) {
		t.Fatalf("privada: %v", err)
	}
	u2, err := LeerLlavePublica(up)
	if err != nil || !u2.Equal(publica) {
		t.Fatalf("pública PEM: %v", err)
	}
	u3, err := LeerLlavePublica([]byte(b64(publica) + "\n"))
	if err != nil || !u3.Equal(publica) {
		t.Fatalf("pública base64: %v", err)
	}
	if _, err := LeerLlavePublica([]byte("abc")); err == nil {
		t.Fatal("esperaba error")
	}
	if _, err := LeerLlavePrivada(up); err == nil {
		t.Fatal("una pública no es una privada")
	}
}

func TestFirmarDocumento(t *testing.T) {
	publica, privada, _ := ed25519.GenerateKey(nil)
	doc := []byte(`{"version":"1","id":"x","impresora":"caja","operaciones":[{"tipo":"cortar"}],"firma":"vieja"}`)
	emitido := time.Date(2026, 10, 15, 12, 0, 0, 0, time.FixedZone("CST", -6*3600))
	firmado, err := FirmarDocumento(doc, privada, OpcionesFirma{ClienteID: "cli_prueba01", Emitido: emitido, NuevoID: "nuevo"})
	if err != nil {
		t.Fatal(err)
	}
	var crudo map[string]json.RawMessage
	json.Unmarshal(firmado, &crudo)
	d := map[string]string{}
	for _, k := range []string{"cliente_id", "emitido", "expira", "firma", "id"} {
		var s string
		json.Unmarshal(crudo[k], &s)
		d[k] = s
	}
	if d["cliente_id"] != "cli_prueba01" || d["emitido"] != "2026-10-15T18:00:00Z" || d["expira"] != "2026-10-15T18:01:30Z" || d["id"] != "nuevo" {
		t.Fatalf("campos: %v", d)
	}
	if err := VerificarFirma(firmado, d["firma"], publica); err != nil {
		t.Fatalf("la firma no verifica: %v", err)
	}
}

// TestVectorDeFirma comprueba testdata/vector-firma, que se publica para los integradores.
func TestVectorDeFirma(t *testing.T) {
	leer := func(n string) []byte {
		b, err := os.ReadFile("../../testdata/vector-firma/" + n)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	privada, err := LeerLlavePrivada(leer("llave-privada.pem"))
	if err != nil {
		t.Fatal(err)
	}
	publica, err := LeerLlavePublica(leer("llave-publica.pem"))
	if err != nil || !publica.Equal(privada.Public()) {
		t.Fatalf("la llave pública no corresponde: %v", err)
	}
	if !bytes.Equal(publica, publicaPrueba) {
		t.Fatal("el vector debe usar la llave de RFC 8032")
	}
	firmado := leer("firmado.json")
	canonico, _ := CanonicoSinFirma(firmado)
	if !bytes.Equal(canonico, leer("canonico.txt")) {
		t.Fatalf("forma canónica:\n got %s\nwant %s", canonico, leer("canonico.txt"))
	}
	firma := string(leer("firma.txt"))
	if b64(ed25519.Sign(privada, canonico)) != firma {
		t.Fatal("la firma de canonico.txt no coincide con firma.txt")
	}
	if err := VerificarFirma(firmado, firma, publica); err != nil {
		t.Fatal(err)
	}
	// El vector y el ejemplo de la especificación son el mismo documento firmado.
	ejemplo, _ := CanonicoSinFirma(leerEjemplo(t, "validos/firmado.json"))
	if !bytes.Equal(ejemplo, canonico) {
		t.Fatal("el vector debe coincidir con ejemplos/validos/firmado.json")
	}
	// Firmar documento.json con los mismos datos reproduce firmado.json.
	emitido, _ := time.Parse(time.RFC3339, "2026-10-14T18:30:00Z")
	otra, err := FirmarDocumento(leer("documento.json"), privada, OpcionesFirma{ClienteID: "cli_7Hq2MZp4XwR9", Emitido: emitido})
	if err != nil {
		t.Fatal(err)
	}
	if c2, _ := CanonicoSinFirma(otra); !bytes.Equal(c2, canonico) || !bytes.Contains(otra, []byte(firma)) {
		t.Fatal("FirmarDocumento no reproduce el vector")
	}
}
