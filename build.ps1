# PRISMA — tareas de desarrollo en Windows (equivalente al Makefile).
# Uso: .\build.ps1 [test|test-sdk|build|build-todo|lint|limpiar]
param(
    [ValidateSet('test', 'test-sdk', 'build', 'build-todo', 'lint', 'limpiar')]
    [string]$Tarea = 'build'
)

$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

$Modulo = 'github.com/bayalad7/prisma'
$Staticcheck = 'honnef.co/go/tools/cmd/staticcheck@v0.8.2'
$Plataformas = @('windows/amd64', 'windows/386', 'linux/amd64', 'linux/arm64', 'darwin/amd64', 'darwin/arm64')

function Invocar([string]$Programa, [string[]]$Argumentos) {
    & $Programa @Argumentos
    if ($LASTEXITCODE -ne 0) { throw "$Programa $($Argumentos -join ' ') terminó con código $LASTEXITCODE" }
}

function Obtener-Git([string[]]$Argumentos, [string]$PorDefecto) {
    try {
        $salida = & git @Argumentos 2>$null
        if ($LASTEXITCODE -eq 0 -and $salida) { return "$salida".Trim() }
    } catch { }
    return $PorDefecto
}

function Obtener-LdFlags {
    $version = if ($env:VERSION) { $env:VERSION } else { Obtener-Git @('describe', '--tags', '--always', '--dirty') '0.0.0-dev' }
    $commit = Obtener-Git @('rev-parse', '--short', 'HEAD') 'desconocido'
    $fecha = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
    return "-s -w -X $Modulo/internal/version.Version=$version -X $Modulo/internal/version.Commit=$commit -X $Modulo/internal/version.Fecha=$fecha"
}

$env:CGO_ENABLED = '0'

switch ($Tarea) {
    'test' {
        Invocar 'go' @('test', './...')
    }
    'test-sdk' {
        # SDK JavaScript y firmadores de referencia: requieren Node.js, PHP 8.2 (sodium) y Python con cryptography.
        Push-Location sdk/js
        try { Invocar 'npm' @('test') } finally { Pop-Location }
        Invocar 'node' @('--test', 'sdk/firmadores/node/*.test.mjs')
        Invocar 'php' @('sdk/firmadores/php/prueba.php')
        Invocar 'python' @('-m', 'unittest', 'discover', '-s', 'sdk/firmadores/python')
    }
    'build' {
        Invocar 'go' @('build', '-trimpath', '-ldflags', (Obtener-LdFlags), '-o', 'dist/prisma.exe', './cmd/prisma')
        Write-Host 'Binario: dist\prisma.exe'
    }
    'build-todo' {
        $ldflags = Obtener-LdFlags
        $goosAnterior = $env:GOOS; $goarchAnterior = $env:GOARCH
        try {
            foreach ($p in $Plataformas) {
                $os, $arch = $p.Split('/')
                $ext = if ($os -eq 'windows') { '.exe' } else { '' }
                Write-Host "==> $os/$arch"
                $env:GOOS = $os; $env:GOARCH = $arch
                Invocar 'go' @('build', '-trimpath', '-ldflags', $ldflags, '-o', "dist/prisma-$os-$arch$ext", './cmd/prisma')
            }
        } finally {
            $env:GOOS = $goosAnterior; $env:GOARCH = $goarchAnterior
        }
    }
    'lint' {
        Invocar 'go' @('vet', './...')
        Invocar 'go' @('run', $Staticcheck, './...')
    }
    'limpiar' {
        if (Test-Path dist) { Remove-Item -Recurse -Force dist }
    }
}
