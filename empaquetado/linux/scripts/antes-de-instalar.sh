#!/bin/sh
# preinst: crea el usuario de sistema «prisma» antes de copiar los archivos.
set -e

if ! getent passwd prisma >/dev/null; then
	useradd --system --user-group --home-dir /var/lib/prisma --no-create-home \
		--shell /usr/sbin/nologin --comment "PRISMA - Puente de Impresion Termica" prisma
fi

# lp: impresoras USB (/dev/usb/lp*); dialout: puertos serie y Bluetooth (/dev/rfcomm*).
for grupo in lp dialout; do
	if getent group "$grupo" >/dev/null; then
		usermod -aG "$grupo" prisma
	fi
done
