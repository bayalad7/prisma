package actualizacion

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// relojDePrueba es la fecha y hora reales que ve el comprobador; las pruebas la adelantan como lo
// haría una suspensión del equipo.
type relojDePrueba struct{ t time.Time }

func (r *relojDePrueba) ahora() time.Time      { return r.t }
func (r *relojDePrueba) pasar(d time.Duration) { r.t = r.t.Add(d) }

// TestReintentaPocoDespuesDeUnFallo: lo que pasó con 0.3.1 en Windows. El equipo sale de una
// suspensión, la consulta llega antes de que vuelva la red y falla; antes se esperaba un día entero
// para reintentar (y volvía a caer al despertar). Ahora reintenta a los 15 minutos, duplica la espera
// en cada fallo y, cuando sale bien, avisa la versión nueva y espera un día. Solo el primer fallo de
// una racha va a la bitácora.
func TestReintentaPocoDespuesDeUnFallo(t *testing.T) {
	p := nuevaPublicacion(t, "v0.3.2")
	p.fallar.Store(2)
	reloj := &relojDePrueba{t: time.Date(2026, 10, 26, 9, 22, 0, 0, time.UTC)}
	var bitacora bytes.Buffer
	var guardado Estado
	aviso := &Aviso{}
	c := &Comprobador{Cliente: p.cliente(), Actual: "0.3.1", Aviso: aviso, Reintento: 15 * time.Minute, Cada: 24 * time.Hour,
		Log: slog.New(slog.NewJSONHandler(&bitacora, nil)), Ahora: reloj.ahora, Guardar: func(_ context.Context, e Estado) { guardado = e }}
	ctx := context.Background()

	if !c.toca() {
		t.Fatal("sin consultas previas, toca consultar")
	}
	if e := c.Comprobar(ctx); e.FallosSeguidos != 1 || e.Error == "" || aviso.Disponible() != nil {
		t.Fatalf("primer fallo: %+v", e)
	}
	reloj.pasar(14 * time.Minute)
	if c.toca() {
		t.Fatal("no debe reintentar antes de 15 minutos")
	}
	reloj.pasar(time.Minute)
	if !c.toca() {
		t.Fatal("a los 15 minutos debe reintentar")
	}
	c.Comprobar(ctx) // segundo fallo: la espera se duplica
	reloj.pasar(29 * time.Minute)
	if c.toca() {
		t.Fatal("tras dos fallos espera 30 minutos")
	}
	reloj.pasar(time.Minute)
	if !c.toca() {
		t.Fatal("a los 30 minutos debe reintentar")
	}
	e := c.Comprobar(ctx)
	if e.Error != "" || e.FallosSeguidos != 0 || e.Disponible == nil || e.Disponible.Numero != "0.3.2" {
		t.Fatalf("cuando sale bien: %+v", e)
	}
	if v := aviso.Disponible(); v == nil || v.Numero != "0.3.2" {
		t.Fatalf("el panel debe avisar la 0.3.2: %+v", v)
	}
	if guardado.UltimoExito != reloj.t || guardado.Disponible == nil {
		t.Fatalf("debe guardar el estado: %+v", guardado)
	}
	reloj.pasar(23 * time.Hour)
	if c.toca() {
		t.Fatal("después de una consulta buena, espera un día")
	}
	reloj.pasar(time.Hour)
	if !c.toca() {
		t.Fatal("al día siguiente (fecha y hora reales) debe consultar")
	}
	if n := strings.Count(bitacora.String(), "no se pudo comprobar"); n != 1 {
		t.Fatalf("solo el primer fallo de la racha va a la bitácora, hubo %d:\n%s", n, bitacora.String())
	}
	if !strings.Contains(bitacora.String(), "se volvió a comprobar") || !strings.Contains(bitacora.String(), "hay una versión nueva") {
		t.Fatalf("bitácora:\n%s", bitacora.String())
	}
}

// TestLaEsperaNoPasaDeUnDia: tras muchos fallos seguidos (un negocio sin Internet), reintenta como
// máximo una vez al día.
func TestLaEsperaNoPasaDeUnDia(t *testing.T) {
	reloj := &relojDePrueba{t: time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)}
	c := &Comprobador{Reintento: 15 * time.Minute, Cada: 24 * time.Hour, Ahora: reloj.ahora}
	c.estado = Estado{UltimoIntento: reloj.t, FallosSeguidos: 40, Error: "sin red"}
	reloj.pasar(24*time.Hour - time.Minute)
	if c.toca() {
		t.Fatal("no antes de un día")
	}
	reloj.pasar(time.Minute)
	if !c.toca() {
		t.Fatal("a lo más un día")
	}
}

// TestElAvisoSobreviveAUnReinicio: la versión nueva vista se guarda y, al arrancar otra vez, se
// avisa de inmediato, sin esperar a la próxima consulta. Después de actualizar, ya no se avisa.
func TestElAvisoSobreviveAUnReinicio(t *testing.T) {
	guardado := Estado{UltimoIntento: time.Now(), UltimoExito: time.Now(),
		Disponible: &VersionVista{Numero: "0.3.2", URL: "https://github.com/bayalad7/prisma/releases/tag/v0.3.2"}}
	cargar := func(context.Context) Estado { return guardado }

	aviso := &Aviso{}
	c := &Comprobador{Actual: "0.3.1", Aviso: aviso, Cargar: cargar}
	c.recuperar(context.Background())
	if v := aviso.Disponible(); v == nil || v.Numero != "0.3.2" || c.Estado().Disponible == nil {
		t.Fatalf("debe avisar al arrancar: %+v", v)
	}

	actualizado := &Aviso{}
	(&Comprobador{Actual: "0.3.2", Aviso: actualizado, Cargar: cargar}).recuperar(context.Background())
	if actualizado.Disponible() != nil {
		t.Fatal("con la versión ya instalada, no hay aviso")
	}
}
