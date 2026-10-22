package servicio

import (
	"encoding/xml"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var actualizar = flag.Bool("actualizar", false, "regenera empaquetado/linux/prisma.service")

// La unidad que instala el paquete .deb debe ser la misma que genera «prisma servicio instalar»
// con el binario en /usr/bin. Para regenerarla: go test ./internal/servicio -run Deb -actualizar
func TestUnidadDelPaqueteDeb(t *testing.T) {
	const ruta = "../../empaquetado/linux/prisma.service"
	esperada := UnidadSystemd(Comando{Ejecutable: "/usr/bin/prisma", Argumentos: Argumentos("")})
	if *actualizar {
		if err := os.WriteFile(ruta, []byte(esperada), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	archivo, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ReplaceAll(string(archivo), "\r\n", "\n") != esperada {
		t.Fatalf("empaquetado/linux/prisma.service no coincide con UnidadSystemd; regenéralo con:\n%s", esperada)
	}
}

func TestUnidadSystemd(t *testing.T) {
	u := UnidadSystemd(Comando{Ejecutable: "/opt/mi prisma/prisma", Argumentos: Argumentos("/etc/prisma/otra config.toml")})
	for _, want := range []string{
		`ExecStart="/opt/mi prisma/prisma" servicio ejecutar --config "/etc/prisma/otra config.toml"`,
		"User=prisma", "Restart=on-failure", "StateDirectory=prisma", "LogsDirectory=prisma", "WantedBy=multi-user.target",
	} {
		if !strings.Contains(u, want) {
			t.Errorf("falta %q en:\n%s", want, u)
		}
	}
	if strings.Contains(u, "{{") {
		t.Error("la unidad no debe llevar directivas de plantilla: kardianos la interpretaría")
	}
	if got := citarSystemd(`a"b$c%d`); got != `"a\"b$$c%%d"` {
		t.Errorf("citarSystemd = %s", got)
	}
}

func TestPlistLaunchd(t *testing.T) {
	p := PlistLaunchd(Comando{Ejecutable: "/usr/local/bin/prisma", Argumentos: Argumentos("/Users/a&b/config.toml")}, "/Library/Logs/PRISMA")
	// Debe ser XML válido y llevar los argumentos escapados.
	if err := xml.Unmarshal([]byte(p), new(any)); err != nil {
		t.Fatalf("plist inválido: %v\n%s", err, p)
	}
	for _, want := range []string{
		"<string>" + NombreMacOS + "</string>", "<string>/usr/local/bin/prisma</string>", "<string>servicio</string>",
		"<string>/Users/a&amp;b/config.toml</string>", "<string>_prisma</string>", "<key>RunAtLoad</key>\n\t<true/>",
		"<key>SuccessfulExit</key>\n\t\t<false/>", "/Library/Logs/PRISMA/launchd.err",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("falta %q en:\n%s", want, p)
		}
	}
}

func TestLaunchdEnMarcha(t *testing.T) {
	// Salida real de «launchctl print system/com.github.bayalad7.prisma» en el CI de macOS.
	enMarcha := "system/com.github.bayalad7.prisma = {\n\tactive count = 1\n\tpath = /Library/LaunchDaemons/" +
		"com.github.bayalad7.prisma.plist\n\ttype = LaunchDaemon\n\tstate = running\n\n\tprogram = /usr/local/bin/prisma\n"
	if !launchdEnMarcha(enMarcha) {
		t.Error("no reconoce el servicio en marcha")
	}
	for _, salida := range []string{"", strings.Replace(enMarcha, "state = running", "state = not running", 1),
		strings.Replace(enMarcha, "state = running", "state = spawn scheduled", 1)} {
		if launchdEnMarcha(salida) {
			t.Errorf("no está en marcha: %q", salida)
		}
	}
}

func TestArgumentosYEstado(t *testing.T) {
	if got := Argumentos(""); strings.Join(got, " ") != "servicio ejecutar" {
		t.Errorf("Argumentos = %v", got)
	}
	for e, want := range map[Estado]string{{}: "no instalado", {Instalado: true}: "detenido", {Instalado: true, EnMarcha: true}: "en marcha"} {
		if e.String() != want {
			t.Errorf("%+v = %q", e, e.String())
		}
	}
}

func TestRotativo(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "logs", "prisma.log")
	r, err := AbrirRotativo(ruta, 100, 2)
	if err != nil {
		t.Fatal(err)
	}
	linea := strings.Repeat("x", 39) + "\n" // 40 bytes: caben 2 por archivo
	for i := range 7 {
		if _, err := fmt.Fprint(r, fmt.Sprintf("%d", i%10)+linea[1:]); err != nil {
			t.Fatal(err)
		}
	}
	r.Close()
	leer := func(n string) string { b, _ := os.ReadFile(n); return string(b) }
	// 7 líneas de 40 bytes en archivos de 100: [0,1] [2,3] [4,5] [6] → se conservan 2 copias.
	if got := leer(ruta); !strings.HasPrefix(got, "6") || len(got) != 40 {
		t.Errorf("actual: %q", got)
	}
	if got := leer(ruta + ".1"); !strings.HasPrefix(got, "4") || len(got) != 80 {
		t.Errorf(".1: %q", got)
	}
	if got := leer(ruta + ".2"); !strings.HasPrefix(got, "2") {
		t.Errorf(".2: %q", got)
	}
	if _, err := os.Stat(ruta + ".3"); !os.IsNotExist(err) {
		t.Error("no debe haber más copias que las configuradas")
	}

	// Al reabrir continúa el archivo existente y no se escribe después de cerrar.
	r, _ = AbrirRotativo(ruta, 100, 2)
	fmt.Fprint(r, "7"+linea[1:])
	r.Close()
	if got := leer(ruta); len(got) != 80 {
		t.Errorf("tras reabrir: %d bytes", len(got))
	}
	if _, err := r.Write([]byte("x")); err == nil {
		t.Error("escribir después de cerrar debe fallar")
	}
}

func TestRotativoConcurrente(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "prisma.log")
	r, _ := AbrirRotativo(ruta, 1000, 3)
	defer r.Close()
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				fmt.Fprintf(r, "goroutine %d: una línea de bitácora\n", g)
			}
		}()
	}
	wg.Wait()
	// Ninguna línea se parte entre archivos ni se mezcla con otra.
	for _, n := range []string{ruta, ruta + ".1", ruta + ".2", ruta + ".3"} {
		b, _ := os.ReadFile(n)
		for _, l := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
			if l != "" && !strings.HasSuffix(l, ": una línea de bitácora") {
				t.Fatalf("línea dañada en %s: %q", n, l)
			}
		}
	}
}
