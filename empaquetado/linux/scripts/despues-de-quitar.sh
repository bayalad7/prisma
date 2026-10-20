#!/bin/sh
# postrm: «apt remove» conserva datos, configuración y usuario; «apt purge» los borra.
set -e

if [ -d /run/systemd/system ]; then
	systemctl daemon-reload || true
fi

if [ "$1" = "purge" ]; then
	rm -rf /var/lib/prisma /var/log/prisma /etc/prisma
	if getent passwd prisma >/dev/null; then
		userdel prisma || true
	fi
fi
