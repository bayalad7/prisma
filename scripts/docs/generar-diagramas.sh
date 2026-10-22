#!/bin/sh
# Genera los SVG de los diagramas para el panel (ver generar-diagramas.mjs). Requiere Node.js 18+.
set -e
cd "$(dirname "$0")/../.."
exec node scripts/docs/generar-diagramas.mjs
