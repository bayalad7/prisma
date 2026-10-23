#!/usr/bin/env bash
# Compila cada diagrama Mermaid de los Markdown del repositorio con @mermaid-js/mermaid-cli.
# Sale con error si alguno no compila. Requiere Node.js 18+ (npx descarga mermaid-cli).
#
# Uso: scripts/validar-diagramas.sh
set -euo pipefail

MERMAID_CLI="@mermaid-js/mermaid-cli@11.4.2"
raiz="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

node "$raiz/scripts/extraer-diagramas.mjs" "$tmp"

fallas=0
while IFS=$'\t' read -r mmd origen; do
  [ -n "$mmd" ] || continue
  if npx --yes -p "$MERMAID_CLI" mmdc -q -p "$raiz/scripts/puppeteer.json" \
       -i "$tmp/$mmd" -o "$tmp/${mmd%.mmd}.svg" >"$tmp/${mmd%.mmd}.log" 2>&1; then
    echo "ok     $origen"
  else
    echo "FALLA  $origen"
    grep -m 3 -E "Error|error|Expecting|line" "$tmp/${mmd%.mmd}.log" | sed 's/^/       /' || true
    fallas=$((fallas + 1))
  fi
done <"$tmp/indice.tsv"

if [ "$fallas" -gt 0 ]; then
  echo "$fallas diagrama(s) no compilan."
  exit 1
fi
echo "Todos los diagramas compilan."
