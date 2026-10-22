#!/usr/bin/env bash
# Prepara dos puentes de prueba: "vacio" (recién instalado) y "demo" (con datos).
# Uso: copia este archivo y recorrido.mjs a <trabajo>/ux/, compila PRISMA en <trabajo>/prisma.exe y
# ejecuta ux/sembrar.sh. Después arranca las impresoras virtuales (puertos 9100 y 9101) y los dos
# puentes (prisma ejecutar --config ux/demo.toml y ux/vacio.toml), genera trabajos con curl y corre
# node recorrido.mjs <carpeta-de-capturas>. Requiere playwright y @axe-core/playwright.
set -euo pipefail
S="$(cd "$(dirname "$0")/.." && pwd)"
P="$S/prisma.exe"
U="$S/ux"
rm -rf "$U/datos-vacio" "$U/datos-demo" "$U/virtual"
mkdir -p "$U/datos-vacio" "$U/datos-demo" "$U/virtual"
W() { cygpath -m "$1"; }

cat >"$U/vacio.toml" <<EOF
[servidor]
escuchar = "127.0.0.1"
puerto = 8444
[datos]
ruta = '$(W "$U/datos-vacio")'
[dev]
modo_desarrollo = true
EOF
cat >"$U/demo.toml" <<EOF
[servidor]
escuchar = "127.0.0.1"
puerto = 8443
[datos]
ruta = '$(W "$U/datos-demo")'
[dev]
modo_desarrollo = true
EOF

C="--config $(W "$U/demo.toml")"
"$P" impresoras agregar $C --alias caja --nombre "Caja principal" --tipo red --destino 127.0.0.1:9100 --cajon --confirmar
"$P" impresoras agregar $C --alias cocina --nombre "Cocina" --tipo red --destino 127.0.0.1:9101 --ancho 58
"$P" impresoras agregar $C --alias barra --nombre "Barra de bebidas" --tipo red --destino 127.0.0.1:9
"$P" clientes agregar $C --nombre "Punto de venta web" --nivel 2 --origen https://pos.ejemplo.mx --impresoras caja,cocina --operaciones abrir_cajon | tee "$U/cliente-pos.txt"
"$P" clientes agregar $C --nombre "Menú digital" --nivel 1 --origen https://menu.ejemplo.mx --impresoras cocina | tee "$U/cliente-menu.txt"
grep -o 'pk_[A-Za-z0-9_]*' "$U/cliente-pos.txt" | head -1 >"$U/llave-pos.txt"
