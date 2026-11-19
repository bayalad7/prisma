#!/bin/sh
# Construye dist/PRISMA-<versión>-macos.pkg (binario universal Intel + Apple Silicon). Solo en macOS.
# Instala también /Applications/PRISMA.app: el acceso con el logo de la marca que abre el panel, para
# encontrarlo con Spotlight o en Launchpad (icono: empaquetado/marca/prisma.icns, de internal/marca).
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

APP="$TRABAJO/raiz/Applications/PRISMA.app"
mkdir -p "$TRABAJO/raiz/Applications"
cp -R "$RAIZ/empaquetado/macos/PRISMA.app" "$APP"
mkdir -p "$APP/Contents/Resources"
cp "$RAIZ/empaquetado/marca/prisma.icns" "$APP/Contents/Resources/prisma.icns"
sed -i '' "s/@VERSION@/$VERSION/g" "$APP/Contents/Info.plist"
chmod 0755 "$APP/Contents/MacOS/abrir-panel"

if [ -n "${PRISMA_FIRMA_APP:-}" ]; then
	codesign --force --options runtime --timestamp --sign "$PRISMA_FIRMA_APP" "$TRABAJO/raiz/usr/local/bin/prisma"
	codesign --force --options runtime --timestamp --sign "$PRISMA_FIRMA_APP" "$APP"
fi

# PRISMA.app no es reubicable: si alguien la movió, el instalador la deja de nuevo en /Applications.
pkgbuild --analyze --root "$TRABAJO/raiz" "$TRABAJO/componentes.plist"
/usr/libexec/PlistBuddy -c "Set :0:BundleIsRelocatable false" "$TRABAJO/componentes.plist"

chmod 0755 "$RAIZ/empaquetado/macos/scripts/preinstall" "$RAIZ/empaquetado/macos/scripts/postinstall" # pkgbuild los exige ejecutables
set -- --root "$TRABAJO/raiz" --identifier com.github.bayalad7.prisma --version "$VERSION" \
	--scripts "$RAIZ/empaquetado/macos/scripts" --install-location / --component-plist "$TRABAJO/componentes.plist"
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
