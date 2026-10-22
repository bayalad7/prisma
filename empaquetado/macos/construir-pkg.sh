#!/bin/sh
# Construye dist/PRISMA-<versión>-macos.pkg (binario universal Intel + Apple Silicon). Solo en macOS.
#   empaquetado/macos/construir-pkg.sh 1.2.3
# Requiere dist/prisma-darwin-amd64 y dist/prisma-darwin-arm64 (make build-todo).
#
# Firma y notarización opcionales, por variables de entorno:
#   PRISMA_FIRMA_APP          identidad «Developer ID Application: …» para firmar el binario
#   PRISMA_FIRMA_INSTALADOR   identidad «Developer ID Installer: …» para firmar el .pkg
#   PRISMA_APPLE_ID, PRISMA_APPLE_EQUIPO, PRISMA_APPLE_CLAVE
#                             Apple ID, Team ID y contraseña de app para notarizar el .pkg
set -eu

VERSION="${1:?uso: construir-pkg.sh <versión>}"
RAIZ=$(cd "$(dirname "$0")/../.." && pwd)
TRABAJO=$(mktemp -d)
trap 'rm -rf "$TRABAJO"' EXIT
SALIDA="$RAIZ/dist/PRISMA-$VERSION-macos.pkg"

mkdir -p "$TRABAJO/raiz/usr/local/bin"
lipo -create -output "$TRABAJO/raiz/usr/local/bin/prisma" \
	"$RAIZ/dist/prisma-darwin-amd64" "$RAIZ/dist/prisma-darwin-arm64"
chmod 0755 "$TRABAJO/raiz/usr/local/bin/prisma"

if [ -n "${PRISMA_FIRMA_APP:-}" ]; then
	codesign --force --options runtime --timestamp --sign "$PRISMA_FIRMA_APP" "$TRABAJO/raiz/usr/local/bin/prisma"
fi

chmod 0755 "$RAIZ/empaquetado/macos/scripts/preinstall" "$RAIZ/empaquetado/macos/scripts/postinstall" # pkgbuild los exige ejecutables
set -- --root "$TRABAJO/raiz" --identifier com.github.bayalad7.prisma --version "$VERSION" \
	--scripts "$RAIZ/empaquetado/macos/scripts" --install-location /
if [ -n "${PRISMA_FIRMA_INSTALADOR:-}" ]; then
	set -- "$@" --sign "$PRISMA_FIRMA_INSTALADOR"
fi
pkgbuild "$@" "$SALIDA"

if [ -n "${PRISMA_APPLE_ID:-}" ]; then
	xcrun notarytool submit "$SALIDA" --apple-id "$PRISMA_APPLE_ID" --team-id "$PRISMA_APPLE_EQUIPO" \
		--password "$PRISMA_APPLE_CLAVE" --wait
	xcrun stapler staple "$SALIDA"
fi
echo "Paquete: $SALIDA"
