# Genera los SVG de los diagramas para el panel (ver generar-diagramas.mjs). Requiere Node.js 18+.
$ErrorActionPreference = "Stop"
Set-Location (Join-Path $PSScriptRoot "../..")
node scripts/docs/generar-diagramas.mjs
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
