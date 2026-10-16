package servidor

import "crypto/tls"

// insecuro acepta el certificado autofirmado del puente de prueba.
func insecuro() *tls.Config { return &tls.Config{InsecureSkipVerify: true} } //nolint:gosec // solo en pruebas
