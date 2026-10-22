package servicio

import (
	"bytes"
	"encoding/xml"
	"strings"
	"text/template"
)

// Nombres del servicio en cada sistema.
const (
	NombreWindows = "PRISMA"
	NombreLinux   = "prisma"
	NombreMacOS   = "com.github.bayalad7.prisma"
	NombreVisible = "PRISMA - Puente de Impresión Térmica"
	Descripcion   = "Recibe tickets de aplicaciones autorizadas por HTTPS y los imprime en impresoras térmicas."
	UsuarioLinux  = "prisma"
	UsuarioMacOS  = "_prisma"
)

// Comando es el ejecutable y los argumentos con que el sistema arranca el puente.
type Comando struct {
	Ejecutable string
	Argumentos []string
}

// Argumentos del servicio: «servicio ejecutar», y la configuración si no es la del sistema.
func Argumentos(rutaConfig string) []string {
	args := []string{"servicio", "ejecutar"}
	if rutaConfig != "" {
		args = append(args, "--config", rutaConfig)
	}
	return args
}

// La unidad no crea usuario ni grupos: lo hacen «prisma servicio instalar» o el paquete .deb.
// StateDirectory y LogsDirectory hacen que systemd cree /var/lib/prisma y /var/log/prisma con el
// dueño correcto aunque alguien los borre.
var plantillaSystemd = template.Must(template.New("systemd").Funcs(template.FuncMap{"comando": comandoSystemd}).Parse(`[Unit]
Description=` + NombreVisible + `
Documentation=https://github.com/bayalad7/prisma/blob/main/docs/instalacion/README.md
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=300
StartLimitBurst=10

[Service]
Type=simple
User=` + UsuarioLinux + `
Group=` + UsuarioLinux + `
ExecStart={{comando .}}
Restart=on-failure
RestartSec=5
StateDirectory=prisma
StateDirectoryMode=0750
LogsDirectory=prisma
LogsDirectoryMode=0750
AmbientCapabilities=CAP_NET_BIND_SERVICE
NoNewPrivileges=true
ProtectSystem=full
ProtectHome=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
`))

// UnidadSystemd devuelve el contenido de prisma.service.
func UnidadSystemd(c Comando) string {
	var b bytes.Buffer
	if err := plantillaSystemd.Execute(&b, c); err != nil {
		panic(err) // la plantilla es fija: un error aquí es un error de programación
	}
	return b.String()
}

// comandoSystemd escribe ExecStart con comillas donde haga falta (sintaxis de systemd).
func comandoSystemd(c Comando) string {
	partes := []string{citarSystemd(c.Ejecutable)}
	for _, a := range c.Argumentos {
		partes = append(partes, citarSystemd(a))
	}
	return strings.Join(partes, " ")
}

func citarSystemd(s string) string {
	if s != "" && !strings.ContainsAny(s, " \t\"'\\$%;") {
		return s
	}
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `$$`, `%`, `%%`)
	return `"` + r.Replace(s) + `"`
}

// KeepAlive con SuccessfulExit=false: launchd reinicia el puente si termina con error, pero no si
// se detuvo a propósito.
var plantillaLaunchd = template.Must(template.New("launchd").Funcs(template.FuncMap{"xml": escaparXML}).Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>` + NombreMacOS + `</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{xml .Comando.Ejecutable}}</string>{{range .Comando.Argumentos}}
		<string>{{xml .}}</string>{{end}}
	</array>
	<key>UserName</key>
	<string>` + UsuarioMacOS + `</string>
	<key>GroupName</key>
	<string>` + UsuarioMacOS + `</string>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<dict>
		<key>SuccessfulExit</key>
		<false/>
	</dict>
	<key>ThrottleInterval</key>
	<integer>10</integer>
	<key>StandardErrorPath</key>
	<string>{{xml .Logs}}/launchd.err</string>
</dict>
</plist>
`))

// PlistLaunchd devuelve el LaunchDaemon. logs es el directorio de bitácoras: launchd deja ahí la
// salida de error que el puente no alcance a escribir en su bitácora (por ejemplo, un pánico).
func PlistLaunchd(c Comando, logs string) string {
	var b bytes.Buffer
	if err := plantillaLaunchd.Execute(&b, struct {
		Comando Comando
		Logs    string
	}{c, logs}); err != nil {
		panic(err)
	}
	return b.String()
}

func escaparXML(s string) string {
	var b strings.Builder
	xml.EscapeText(&b, []byte(s))
	return b.String()
}
