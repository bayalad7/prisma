#!/bin/sh
# Instalador de PRISMA para Debian, Ubuntu y Raspberry Pi OS (64 bits).
#
# En una línea:
#   curl -fsSL https://raw.githubusercontent.com/bayalad7/prisma/main/empaquetado/linux/instalar.sh | sh
#
# Para revisarlo antes de ejecutarlo (recomendado):
#   curl -fsSLO https://raw.githubusercontent.com/bayalad7/prisma/main/empaquetado/linux/instalar.sh
#   less instalar.sh
#   sh instalar.sh
#
# Qué hace, en orden:
#   1. Comprueba que el sistema sea Debian/Ubuntu de 64 bits (amd64 o arm64).
#   2. Descarga de GitHub Releases el paquete .deb, checksums.txt y su firma.
#   3. Verifica la firma de checksums.txt con la llave pública de abajo y el SHA-256 del .deb.
#   4. Instala el paquete con apt (pide la contraseña de sudo). El paquete crea el usuario
#      «prisma» y el servicio de systemd, que arranca con el equipo.
#
# Variables opcionales:
#   PRISMA_VERSION=1.2.3   instala esa versión en lugar de la última.
set -eu

REPO="bayalad7/prisma"
# Llave pública Ed25519 de las versiones (la misma que empaquetado/llave-releases.pub).
LLAVE_RELEASES=""
URL_API="${PRISMA_URL_API:-https://api.github.com/repos/$REPO/releases}"
URL_DESCARGAS="${PRISMA_URL_DESCARGAS:-https://github.com/$REPO/releases/download}"

error() {
	printf '\nError: %s\n' "$1" >&2
	exit 1
}
paso() { printf '==> %s\n' "$1"; }

# 1. Sistema
command -v dpkg >/dev/null 2>&1 || error "este instalador es para Debian, Ubuntu o Raspberry Pi OS (no se encontró dpkg)."
ARQ=$(dpkg --print-architecture)
case "$ARQ" in
amd64 | arm64) ;;
armhf | armel)
	error "este sistema es de 32 bits ($ARQ). PRISMA necesita Raspberry Pi OS de 64 bits (Pi 3, 4, 5 o Zero 2): instálalo con Raspberry Pi Imager y vuelve a intentarlo."
	;;
*) error "arquitectura no soportada: $ARQ (se necesita amd64 o arm64)." ;;
esac
command -v curl >/dev/null 2>&1 || error "falta curl: instálalo con «sudo apt install curl»."

SUDO=""
if [ "$(id -u)" -ne 0 ]; then
	command -v sudo >/dev/null 2>&1 || error "ejecútalo como root o instala sudo."
	SUDO="sudo"
fi

# 2. Versión y descargas
if [ -n "${PRISMA_VERSION:-}" ]; then
	VERSION="${PRISMA_VERSION#v}"
else
	paso "Buscando la última versión"
	VERSION=$(curl -fsSL "$URL_API/latest" | sed -n 's/.*"tag_name": *"v\{0,1\}\([^"]*\)".*/\1/p' | head -n 1)
	[ -n "$VERSION" ] || error "no se pudo averiguar la última versión de PRISMA."
fi
DEB="prisma_${VERSION}_${ARQ}.deb"
DIR=$(mktemp -d)
trap 'rm -rf "$DIR"' EXIT

paso "Descargando PRISMA $VERSION ($ARQ)"
for archivo in "$DEB" checksums.txt checksums.txt.sig; do
	curl -fsSL -o "$DIR/$archivo" "$URL_DESCARGAS/v$VERSION/$archivo" || error "no se pudo descargar $archivo de la versión $VERSION."
done

# 3. Verificación
if [ -n "$LLAVE_RELEASES" ]; then
	command -v openssl >/dev/null 2>&1 || error "falta openssl para verificar la firma: instálalo con «sudo apt install openssl»."
	printf -- '-----BEGIN PUBLIC KEY-----\nMCowBQYDK2VwAyEA%s\n-----END PUBLIC KEY-----\n' "$LLAVE_RELEASES" >"$DIR/llave.pem"
	base64 -d "$DIR/checksums.txt.sig" >"$DIR/firma.bin" 2>/dev/null || error "la firma de checksums.txt está dañada."
	openssl pkeyutl -verify -pubin -inkey "$DIR/llave.pem" -rawin -in "$DIR/checksums.txt" -sigfile "$DIR/firma.bin" >/dev/null 2>&1 ||
		error "la firma de checksums.txt NO es válida: no se instaló nada. Avisa en https://github.com/$REPO/issues"
	paso "Firma verificada"
else
	printf 'Aviso: este instalador no trae la llave de las versiones; solo se verificará el SHA-256.\n' >&2
fi
(cd "$DIR" && grep -E " [*]?$DEB\$" checksums.txt | sha256sum -c --status) ||
	error "el SHA-256 de $DEB no coincide: la descarga está dañada o fue alterada. No se instaló nada."
paso "SHA-256 verificado"

# 4. Instalación
paso "Instalando (puede pedir tu contraseña)"
chmod 0644 "$DIR/$DEB"
$SUDO apt-get install -y "$DIR/$DEB"
