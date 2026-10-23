; Instalador de PRISMA para Windows (Inno Setup 6).
; Desde la raíz del repositorio, con los binarios de «build.ps1 build-todo» en dist\:
;   ISCC /DVersion=1.2.3 /DArq=amd64 empaquetado\windows\prisma.iss
; Firma opcional: ISCC /DFirmar /Sfirma="signtool sign /fd sha256 /tr http://timestamp.digicert.com /td sha256 /f cert.pfx /p clave $f" …
;
; Qué hace:
;   - copia prisma.exe a Archivos de programa\PRISMA y lo agrega al PATH del sistema;
;   - crea C:\ProgramData\PRISMA con un config.toml de ejemplo (nunca sobrescribe el existente);
;   - «prisma servicio instalar»: carpetas de datos y bitácoras, regla del firewall para el puerto
;     del puente (solo subred local) y servicio de Windows con inicio automático y reinicio ante fallos;
;   - inicia el servicio y, al terminar, abre https://localhost:8443/admin.
; Al actualizar detiene el servicio antes de copiar; al desinstalar quita servicio, regla y PATH,
; y conserva los datos de C:\ProgramData\PRISMA.

#ifndef Version
  #define Version "0.0.0"
#endif
#ifndef Arq
  #define Arq "amd64"
#endif

[Setup]
AppId={{D743612F-A050-4FDC-BC67-826BA98B6250}
AppName=PRISMA
AppVersion={#Version}
AppVerName=PRISMA {#Version}
AppPublisher=PRISMA
AppPublisherURL=https://github.com/bayalad7/prisma
AppSupportURL=https://github.com/bayalad7/prisma/blob/main/docs/instalacion/README.md
VersionInfoVersion={#Version}
VersionInfoDescription=PRISMA - Puente de Impresión Térmica
DefaultDirName={autopf}\PRISMA
DisableProgramGroupPage=yes
DisableDirPage=auto
PrivilegesRequired=admin
#if Arq == "amd64"
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
#endif
OutputDir=..\..\dist
OutputBaseFilename=PRISMA-{#Version}-windows-{#Arq}-setup
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
UninstallDisplayIcon={app}\prisma.exe
UninstallDisplayName=PRISMA - Puente de Impresión Térmica
ChangesEnvironment=yes
CloseApplications=no
SetupLogging=yes
#ifdef Firmar
SignTool=firma
SignedUninstaller=yes
#endif

[Languages]
Name: "es"; MessagesFile: "compiler:Languages\Spanish.isl"

[Messages]
es.FinishedLabel=PRISMA quedó instalado como servicio de Windows y arrancará solo con el equipo.%n%nEn el panel de administración vas a crear tu contraseña y agregar las impresoras. El navegador avisará que el certificado no es de confianza: es normal la primera vez (el puente usa uno propio). Elige «Configuración avanzada» → «Continuar».

[Files]
Source: "..\..\dist\prisma-windows-{#Arq}.exe"; DestDir: "{app}"; DestName: "prisma.exe"; Flags: ignoreversion
Source: "..\config.toml"; DestDir: "{commonappdata}\PRISMA"; Flags: onlyifdoesntexist uninsneveruninstall

[Run]
Filename: "{app}\prisma.exe"; Parameters: "servicio instalar"; StatusMsg: "Instalando el servicio de Windows…"; Flags: runhidden waituntilterminated
Filename: "{app}\prisma.exe"; Parameters: "servicio iniciar"; StatusMsg: "Iniciando el servicio…"; Flags: runhidden waituntilterminated
Filename: "https://localhost:8443/admin"; Description: "Abrir el panel de administración"; Flags: postinstall shellexec nowait skipifsilent

[UninstallRun]
Filename: "{app}\prisma.exe"; Parameters: "servicio desinstalar"; RunOnceId: "QuitarServicio"; Flags: runhidden waituntilterminated

[Code]
const
  ClaveEntorno = 'SYSTEM\CurrentControlSet\Control\Session Manager\Environment';

// Al actualizar, el servicio en marcha tiene prisma.exe abierto: se detiene antes de copiar.
function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  Codigo: Integer;
begin
  Result := '';
  if FileExists(ExpandConstant('{app}\prisma.exe')) then
    Exec(ExpandConstant('{app}\prisma.exe'), 'servicio detener', '', SW_HIDE, ewWaitUntilTerminated, Codigo);
end;

function EnPath(Dir: String): Boolean;
var
  Actual: String;
begin
  Result := RegQueryStringValue(HKLM, ClaveEntorno, 'Path', Actual) and
    (Pos(';' + Uppercase(Dir) + ';', ';' + Uppercase(Actual) + ';') > 0);
end;

procedure CurStepChanged(CurStep: TSetupStep);
var
  Actual: String;
begin
  // Agrega la carpeta al PATH del sistema para usar «prisma» desde PowerShell.
  if (CurStep = ssPostInstall) and not EnPath(ExpandConstant('{app}')) then
    if RegQueryStringValue(HKLM, ClaveEntorno, 'Path', Actual) then
      RegWriteExpandStringValue(HKLM, ClaveEntorno, 'Path', Actual + ';' + ExpandConstant('{app}'));
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  Actual, Dir: String;
  P: Integer;
begin
  if CurUninstallStep <> usPostUninstall then
    exit;
  Dir := ExpandConstant('{app}');
  if RegQueryStringValue(HKLM, ClaveEntorno, 'Path', Actual) then
  begin
    P := Pos(';' + Uppercase(Dir), Uppercase(Actual));
    if P > 0 then
    begin
      Delete(Actual, P, Length(Dir) + 1);
      RegWriteExpandStringValue(HKLM, ClaveEntorno, 'Path', Actual);
    end;
  end;
end;
