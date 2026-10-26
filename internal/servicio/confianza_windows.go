package servicio

// En Windows, el almacén «Entidades de certificación raíz de confianza» del equipo: lo usan Edge,
// Chrome y Firefox (que importa las raíces del sistema).

func confiarEnSistema(rutaPEM string) error {
	return correr("certutil", "-addstore", "-f", "Root", rutaPEM)
}

func quitarDeSistema() error {
	return correr("powershell", "-NoProfile", "-NonInteractive", "-Command",
		`Get-ChildItem Cert:\LocalMachine\Root | Where-Object { $_.Subject -like '*O="`+OrganizacionCertificado+`"*' -or $_.Subject -like '*O=`+OrganizacionCertificado+`*' } | Remove-Item`)
}
