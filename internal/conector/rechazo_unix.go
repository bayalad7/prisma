//go:build !windows

package conector

import (
	"errors"
	"syscall"
)

func conexionRechazada(err error) bool {
	return errors.Is(err, syscall.ECONNREFUSED)
}
