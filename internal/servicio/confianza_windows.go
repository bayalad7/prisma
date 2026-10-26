package servicio

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"slices"
	"unsafe"

	"golang.org/x/sys/windows"
)

// En Windows, el almacén «Entidades de certificación raíz de confianza» del equipo: lo usan Edge,
// Chrome y Firefox (que importa las raíces del sistema). Se usa la API de certificados de Windows
// directamente: no depende de PowerShell ni de la salida de certutil, que cambia con el idioma.

const codificacion = windows.X509_ASN_ENCODING | windows.PKCS_7_ASN_ENCODING

// abrirRaices abre el almacén raíz del equipo (requiere administrador para modificarlo).
func abrirRaices() (windows.Handle, error) {
	nombre, _ := windows.UTF16PtrFromString("ROOT")
	return windows.CertOpenStore(windows.CERT_STORE_PROV_SYSTEM, 0, 0,
		windows.CERT_SYSTEM_STORE_LOCAL_MACHINE, uintptr(unsafe.Pointer(nombre)))
}

// esDePrisma dice si un certificado (DER) es un autofirmado del puente.
func esDePrisma(der []byte) bool {
	c, err := x509.ParseCertificate(der)
	return err == nil && slices.Contains(c.Subject.Organization, OrganizacionCertificado)
}

func confiarEnSistema(rutaPEM string) error {
	datos, err := os.ReadFile(rutaPEM)
	if err != nil {
		return err
	}
	bloque, _ := pem.Decode(datos)
	if bloque == nil || bloque.Type != "CERTIFICATE" {
		return fmt.Errorf("%s no es un certificado PEM", rutaPEM)
	}
	raices, err := abrirRaices()
	if err != nil {
		return fmt.Errorf("no se pudo abrir el almacén raíz del equipo: %w", err)
	}
	defer windows.CertCloseStore(raices, 0)
	ctx, err := windows.CertCreateCertificateContext(codificacion, &bloque.Bytes[0], uint32(len(bloque.Bytes)))
	if err != nil {
		return err
	}
	defer windows.CertFreeCertificateContext(ctx)
	if err := windows.CertAddCertificateContextToStore(raices, ctx, windows.CERT_STORE_ADD_REPLACE_EXISTING, nil); err != nil {
		return fmt.Errorf("no se pudo agregar el certificado al almacén raíz del equipo: %w", err)
	}
	return nil
}

func quitarDeSistema() error {
	raices, err := abrirRaices()
	if err != nil {
		return fmt.Errorf("no se pudo abrir el almacén raíz del equipo: %w", err)
	}
	defer windows.CertCloseStore(raices, 0)
	// Primero se juntan (duplicados) y después se borran: borrar libera el contexto, y la
	// enumeración lo necesita para seguir.
	var dePrisma []*windows.CertContext
	var ctx *windows.CertContext
	for {
		ctx, err = windows.CertEnumCertificatesInStore(raices, ctx)
		if err != nil {
			break
		}
		if esDePrisma(unsafe.Slice(ctx.EncodedCert, ctx.Length)) {
			dePrisma = append(dePrisma, windows.CertDuplicateCertificateContext(ctx))
		}
	}
	var errores []error
	for _, c := range dePrisma {
		if err := windows.CertDeleteCertificateFromStore(c); err != nil {
			errores = append(errores, err)
		}
	}
	return errors.Join(errores...)
}
