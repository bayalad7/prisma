package seguridad

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"testing"
)

// Llave del vector de prueba 1 de RFC 8032 §7.1, publicada en la especificación del contrato.
var publicaPrueba, _ = hex.DecodeString("d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a")

func leerEjemplo(t *testing.T, ruta string) []byte {
	t.Helper()
	datos, err := os.ReadFile("../../ejemplos/" + ruta)
	if err != nil {
		t.Fatal(err)
	}
	return datos
}

func TestFormaCanonicaDeLaEspecificacion(t *testing.T) {
	// Es la forma canónica publicada en docs/contrato-documento-v1.md, §7.
	const esperada = `{"cliente_id":"cli_7Hq2MZp4XwR9","emitido":"2026-10-14T18:30:00Z","expira":"2026-10-14T18:31:30Z","id":"780bf0ea-9a33-4add-86f3-9f4f23000fc4","impresora":"caja","operaciones":[{"negrita":true,"tipo":"texto","valor":"Pedido en línea #812"},{"derecha":"$249.00","izquierda":"TOTAL","tipo":"par"},{"tipo":"cortar"}],"referencia":"PED-000812","version":"1"}`
	c, err := CanonicoSinFirma(leerEjemplo(t, "validos/firmado.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(c) != esperada {
		t.Fatalf("\n got %s\nwant %s", c, esperada)
	}
}

func TestVerificarFirmaDeLosEjemplos(t *testing.T) {
	firmado := leerEjemplo(t, "validos/firmado.json")
	var doc struct{ Firma string }
	json.Unmarshal(firmado, &doc)
	if err := VerificarFirma(firmado, doc.Firma, publicaPrueba); err != nil {
		t.Fatalf("firmado.json debe verificar: %v", err)
	}

	// Los inválidos del índice con regla firma_ed25519 deben fallar (FIRMA_INVALIDA).
	var indice []struct{ Archivo, Codigo, Regla string }
	json.Unmarshal(leerEjemplo(t, "invalidos/indice.json"), &indice)
	probados := 0
	for _, e := range indice {
		if e.Regla != "firma_ed25519" {
			continue
		}
		probados++
		datos := leerEjemplo(t, "invalidos/"+e.Archivo)
		var d struct{ Firma string }
		json.Unmarshal(datos, &d)
		if err := VerificarFirma(datos, d.Firma, publicaPrueba); !errors.Is(err, ErrFirmaInvalida) {
			t.Errorf("%s: esperaba ErrFirmaInvalida (%s), obtuve %v", e.Archivo, e.Codigo, err)
		}
	}
	if probados == 0 {
		t.Fatal("el índice no tiene casos de firma")
	}
}

func TestFirmaIgnoraFormatoPeroNoContenido(t *testing.T) {
	publica, privada, _ := ed25519.GenerateKey(nil)
	original := []byte(`{"version":"1","impresora":"caja","operaciones":[{"tipo":"cortar"}]}`)
	c, _ := CanonicoSinFirma(original)
	firma := ed25519.Sign(privada, c)

	// Mismo contenido con otro orden, espacios y la firma incluida: verifica.
	reformateado := []byte(`{ "operaciones" : [ { "tipo":"cortar" } ], "firma":"x", "impresora":"caja", "version":"1" }`)
	if err := VerificarFirma(reformateado, b64(firma), publica); err != nil {
		t.Fatalf("el formato no debe afectar la firma: %v", err)
	}
	// Otro contenido: no verifica.
	if err := VerificarFirma(bytes.Replace(original, []byte("caja"), []byte("cocina"), 1), b64(firma), publica); !errors.Is(err, ErrFirmaInvalida) {
		t.Fatalf("contenido alterado: %v", err)
	}
	if err := VerificarFirma(original, "no-es-base64", publica); !errors.Is(err, ErrFirmaInvalida) {
		t.Fatalf("firma mal codificada: %v", err)
	}
	if err := VerificarFirma(original, b64(firma), publica[:10]); err == nil {
		t.Fatal("llave pública corta debe fallar")
	}
}

func TestHashContenidoNoDependeDeLaFirmaNiDelFormato(t *testing.T) {
	a, _ := HashContenido([]byte(`{"id":"1","version":"1","firma":"AAAA"}`))
	b, _ := HashContenido([]byte(`{ "version" : "1", "id" : "1" }`))
	c, _ := HashContenido([]byte(`{"version":"1","id":"2"}`))
	if !bytes.Equal(a, b) || bytes.Equal(a, c) || len(a) != 32 {
		t.Fatal("hash de contenido incorrecto")
	}
}

func b64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
