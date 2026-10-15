// Comando prisma: CLI y punto de entrada del puente de impresión.
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := nuevoComandoRaiz().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
