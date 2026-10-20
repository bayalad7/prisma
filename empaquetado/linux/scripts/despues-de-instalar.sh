#!/bin/sh
# postinst: carpetas con sus permisos, y el servicio habilitado (arranca con el equipo) y en marcha.
set -e

install -d -o prisma -g prisma -m 0750 /var/lib/prisma /var/log/prisma

if [ -d /run/systemd/system ]; then
	systemctl daemon-reload
	systemctl enable prisma.service >/dev/null 2>&1
	# En una actualización, reinicia con el binario nuevo; la base se respalda sola antes de migrar.
	if ! systemctl restart prisma.service; then
		echo "Aviso: el servicio no arrancó. Revisa: journalctl -u prisma -n 50" >&2
	fi
fi

ip=$(hostname -I 2>/dev/null | awk '{print $1}')
cat <<MENSAJE

PRISMA está instalado y en marcha.

  Panel de administración:  https://${ip:-<ip-de-este-equipo>}:8443/admin
  Desde otro equipo, el panel pide un código; genéralo aquí con:
      sudo -u prisma prisma admin codigo-inicial

  Estado del servicio:  systemctl status prisma
  Bitácora:             journalctl -u prisma -f

MENSAJE
