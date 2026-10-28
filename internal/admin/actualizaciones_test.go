package admin

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bayalad7/prisma/internal/actualizacion"
)

// TestComprobarActualizacionesAhora: Ajustes muestra la última comprobación y «Comprobar ahora»
// consulta GitHub en el momento. Sin red, dice «sin conexión a Internet»; cuando responde, avisa la
// versión nueva también en Inicio. Una compilación de desarrollo no consulta.
func TestComprobarActualizacionesAhora(t *testing.T) {
	github := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"tag_name": "v0.3.3", "html_url": "https://github.com/bayalad7/prisma/releases/tag/v0.3.3",
			"published_at": "2026-10-28T12:00:00Z"})
	}))
	t.Cleanup(github.Close)

	b := nuevoBanco(t)
	s := b.configurado()
	if r := s.exigir("GET", "/admin/api/ajustes", nil, 200); r.datos["actualizaciones"].(map[string]any)["ultima"] != "" {
		t.Fatalf("sin comprobaciones: %v", r.datos["actualizaciones"])
	}
	s.exigirError("POST", "/admin/api/actualizaciones/comprobar", nil, 409, "SIN_COMPROBADOR")

	cliente := actualizacion.NuevoCliente("0.3.2")
	// Como al salir de una suspensión, antes de que vuelva la red: el nombre no resuelve (.invalid
	// nunca existe), igual que «lookup api.github.com: no such host».
	cliente.URLAPI = "https://api.github.invalid/latest"
	b.adm.op.Version = "0.3.2"
	b.adm.op.Actualizaciones = &actualizacion.Comprobador{Cliente: cliente, Actual: "0.3.2", Aviso: b.aviso, Log: slog.New(slog.DiscardHandler)}

	r := s.exigir("POST", "/admin/api/actualizaciones/comprobar", nil, 200)
	if r.datos["ultima"] == "" || r.datos["error"] != "sin conexión a Internet" || r.datos["disponible"] != nil {
		t.Fatalf("sin red: %v", r.datos)
	}
	cliente.URLAPI = github.URL
	r = s.exigir("POST", "/admin/api/actualizaciones/comprobar", nil, 200)
	if d, _ := r.datos["disponible"].(map[string]any); r.datos["error"] != "" || d["version"] != "0.3.3" {
		t.Fatalf("con red: %v", r.datos)
	}
	if v, _ := s.exigir("GET", "/admin/api/resumen", nil, 200).datos["actualizacion"].(map[string]any); v["version"] != "0.3.3" {
		t.Fatal("Inicio debe avisar la versión que encontró «Comprobar ahora»")
	}
}
