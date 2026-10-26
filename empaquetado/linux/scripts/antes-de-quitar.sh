#!/bin/sh
# prerm: al desinstalar, detiene y deshabilita el servicio. En una actualización no hace nada: el
# postinst de la versión nueva lo reinicia.
set -e

if [ "$1" = "remove" ] && [ -d /run/systemd/system ]; then
	systemctl disable --now prisma.service >/dev/null 2>&1 || true
fi
if [ "$1" = "remove" ]; then
	prisma tls confiar --quitar >/dev/null 2>&1 || true
fi
