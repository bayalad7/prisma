package admin

import (
	"context"
	"testing"

	"github.com/bayalad7/prisma/internal/almacen"
	"github.com/bayalad7/prisma/internal/conector"
)

// TestPrimerosPasosDelResumen: el resumen dice qué pasos de «Primeros pasos» faltan, y «Hoy» cuenta
// las pruebas del panel aparte de los tickets de las aplicaciones.
func TestPrimerosPasosDelResumen(t *testing.T) {
	b := nuevoBanco(t)
	s := b.configurado()
	conector.RollosVirtuales.Vaciar("prueba")
	t.Cleanup(func() { conector.RollosVirtuales.Vaciar("prueba") })

	pasos := func() map[string]any {
		t.Helper()
		return s.exigir("GET", "/admin/api/resumen", nil, 200).datos["primeros_pasos"].(map[string]any)
	}
	p := pasos()
	if p["impresoras_fisicas"] != float64(0) || p["impresoras_virtuales"] != float64(0) || p["aplicaciones"] != float64(0) ||
		p["prueba_impresa"] != false || p["acentos_probados"] != false {
		t.Fatalf("puente recién configurado: %v", p)
	}

	s.exigir("POST", "/admin/api/impresoras", perfilVirtual(nil), 200)
	s.exigir("POST", "/admin/api/impresoras/prueba/diagnostico", nil, 200)
	s.exigir("POST", "/admin/api/impresoras/prueba/paginas-codigos", nil, 200)
	p = pasos()
	if p["impresoras_virtuales"] != float64(1) || p["prueba_impresa"] != true {
		t.Fatalf("tras el diagnóstico en la virtual: %v", p)
	}
	if p["acentos_probados"] != false {
		t.Fatal("probar páginas de códigos en una impresora virtual no prueba los acentos de una física")
	}

	hoy := s.exigir("GET", "/admin/api/resumen", nil, 200).datos["trabajos_hoy"].(map[string]any)
	if hoy["completados"] != float64(0) || hoy["pruebas"] != float64(2) {
		t.Fatalf("«Hoy» debe contar las pruebas del panel aparte: %v", hoy)
	}
}

// TestPrimerosPasosDeUnPuenteActualizado: un puente que viene de 0.1.0 no tiene las marcas, pero sí
// pruebas en el historial; el resumen las deduce.
func TestPrimerosPasosDeUnPuenteActualizado(t *testing.T) {
	b := nuevoBanco(t)
	s := b.configurado()
	s.exigir("POST", "/admin/api/impresoras", map[string]any{"alias": "caja", "tipo": "archivo", "destino": t.TempDir(), "ancho_mm": 80,
		"columnas": 48, "columnas_fuente_b": 64, "ancho_px": 576, "pagina_codigos": "CP858", "comando_pagina_codigos": 19, "activa": true}, 200)
	s.exigir("POST", "/admin/api/impresoras/caja/diagnostico", nil, 200)
	s.exigir("POST", "/admin/api/impresoras/caja/paginas-codigos", nil, 200)
	for _, clave := range []string{almacen.AjustePrimeraPrueba, almacen.AjusteAcentosProbados} {
		if err := b.alm.Ajustes.Eliminar(context.Background(), clave); err != nil {
			t.Fatal(err)
		}
	}
	p := s.exigir("GET", "/admin/api/resumen", nil, 200).datos["primeros_pasos"].(map[string]any)
	if p["prueba_impresa"] != true || p["acentos_probados"] != true {
		t.Fatalf("los pasos deben deducirse del historial: %v", p)
	}
	if v, _, _ := b.alm.Ajustes.Obtener(context.Background(), almacen.AjustePrimeraPrueba); v != "1" {
		t.Fatal("la marca deducida debe guardarse")
	}
}
