# Compila cada diagrama Mermaid de los Markdown del repositorio con @mermaid-js/mermaid-cli.
# Sale con error si alguno no compila. Requiere Node.js 18+ (npx descarga mermaid-cli).
#
# Uso: .\scripts\validar-diagramas.ps1
$ErrorActionPreference = 'Stop'

$MermaidCli = '@mermaid-js/mermaid-cli@11.4.2'
$raiz = Split-Path -Parent $PSScriptRoot
$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("prisma-diagramas-" + [guid]::NewGuid())
New-Item -ItemType Directory -Force $tmp | Out-Null

try {
    node (Join-Path $raiz 'scripts/extraer-diagramas.mjs') $tmp
    if ($LASTEXITCODE -ne 0) { throw 'no se pudieron extraer los diagramas' }

    $fallas = 0
    foreach ($linea in Get-Content (Join-Path $tmp 'indice.tsv')) {
        if (-not $linea) { continue }
        $mmd, $origen = $linea -split "`t", 2
        $entrada = Join-Path $tmp $mmd
        $svg = [System.IO.Path]::ChangeExtension($entrada, '.svg')
        $log = [System.IO.Path]::ChangeExtension($entrada, '.log')
        # La salida va al archivo desde cmd: en Windows PowerShell 5.1, redirigir el stderr de un
        # programa con ErrorActionPreference = Stop lo convierte en un error que corta el script.
        $config = Join-Path $raiz 'scripts/puppeteer.json'
        cmd /c "npx --yes -p $MermaidCli mmdc -q -p `"$config`" -i `"$entrada`" -o `"$svg`" > `"$log`" 2>&1"
        if ($LASTEXITCODE -eq 0) {
            Write-Output "ok     $origen"
        } else {
            Write-Output "FALLA  $origen"
            Select-String -Path $log -Pattern 'Error|Expecting|line' | Select-Object -First 3 | ForEach-Object { Write-Output ("       " + $_.Line) }
            $fallas++
        }
    }

    if ($fallas -gt 0) {
        Write-Output "$fallas diagrama(s) no compilan."
        exit 1
    }
    Write-Output 'Todos los diagramas compilan.'
} finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
