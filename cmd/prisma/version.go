package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bayalad7/prisma/internal/version"
)

func nuevoComandoVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Muestra la versión de PRISMA",
		Args:  cobra.NoArgs,
		Run: func(c *cobra.Command, _ []string) {
			fmt.Fprintln(c.OutOrStdout(), version.Texto())
		},
	}
}
