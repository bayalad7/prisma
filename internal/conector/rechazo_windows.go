//go:build windows

package conector

import (
	"errors"

	"golang.org/x/sys/windows"
)

func conexionRechazada(err error) bool {
	return errors.Is(err, windows.WSAECONNREFUSED)
}
