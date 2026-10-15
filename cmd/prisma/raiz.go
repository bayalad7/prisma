package main

import (
	"github.com/spf13/cobra"
)

// opciones globales compartidas por los subcomandos.
type opciones struct {
	rutaConfig string
}

const plantillaUso = `Uso:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [comando]{{end}}{{if gt (len .Aliases) 0}}

Alias:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Ejemplos:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Comandos:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Opciones:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Opciones globales:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}

Usa "{{.CommandPath}} [comando] --help" para más información sobre un comando.{{end}}
`

func nuevoComandoRaiz() *cobra.Command {
	op := &opciones{}
	raiz := &cobra.Command{
		Use:   "prisma",
		Short: "PRISMA — Puente Remoto de Impresión Sobre Múltiples Arquitecturas",
		Long: "PRISMA recibe documentos JSON por HTTPS desde aplicaciones autorizadas y los imprime\n" +
			"en impresoras de tickets por USB, red o Bluetooth.",
		SilenceUsage:      true,
		SilenceErrors:     true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}
	raiz.SetUsageTemplate(plantillaUso)
	raiz.PersistentFlags().StringVar(&op.rutaConfig, "config", "", "ruta de config.toml (por defecto, la ruta del sistema; ver \"prisma config ruta\")")
	raiz.PersistentFlags().BoolP("help", "h", false, "muestra esta ayuda")
	raiz.SetHelpCommand(&cobra.Command{
		Use:   "help [comando]",
		Short: "Muestra la ayuda de un comando",
		Run: func(c *cobra.Command, args []string) {
			destino, _, err := c.Root().Find(args)
			if destino == nil || err != nil {
				destino = c.Root()
			}
			_ = destino.Help()
		},
	})

	raiz.AddCommand(
		nuevoComandoVersion(),
		nuevoComandoConfig(op),
		nuevoComandoEjecutar(op),
		nuevoComandoAdmin(op),
		nuevoComandoImpresoras(op),
		nuevoComandoImprimir(op),
		nuevoComandoDev(),
	)
	return raiz
}
