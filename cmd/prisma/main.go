// Comando prisma: CLI y punto de entrada del puente de impresión.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/bayalad7/prisma/internal/actualizacion"
	"github.com/bayalad7/prisma/internal/conector"
	"github.com/bayalad7/prisma/internal/contrato"
)

func main() {
	actualizacion.LimpiarAnterior() // el .exe anterior que dejó «prisma actualizar» en Windows
	if err := nuevoComandoRaiz().Execute(); err != nil {
		imprimirError(os.Stderr, err)
		os.Exit(1)
	}
}

// imprimirError muestra el error y, si trae una sugerencia, también la sugerencia.
func imprimirError(w io.Writer, err error) {
	fmt.Fprintln(w, "Error:", err)
	var sugerencia string
	var ce *conector.Error
	var de *contrato.Error
	switch {
	case errors.As(err, &ce):
		sugerencia = ce.Sugerencia
	case errors.As(err, &de):
		sugerencia = de.Sugerencia
	}
	if sugerencia != "" {
		fmt.Fprintln(w, "Sugerencia:", sugerencia)
	}
}
