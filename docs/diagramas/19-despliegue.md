---
titulo: "Despliegue por plataforma"
descripcion: "Qué se instala en cada sistema operativo"
audiencia: [instalador, desarrollador]
pantallas: []
secciones_manual: [instalacion/README]
---
# Despliegue por plataforma

¿Qué se publica para cada sistema y cómo queda instalado el servicio? Para instaladores y para quien mantiene los paquetes.

```mermaid
flowchart LR
    bin["binarios de prisma: 6 plataformas<br/>sin CGO"]:::prisma
    bin --> winst["PRISMA-v-windows-amd64 y 386-setup.exe<br/>(Inno Setup)"]:::externo
    bin --> deb["prisma_v_amd64 y arm64.deb<br/>(nfpm)"]:::externo
    bin --> pkg["PRISMA-v-macos.pkg<br/>(universal)"]:::externo
    deb --> sh["instalar.sh: una línea,<br/>verifica firma y SHA-256"]:::seguridad
    winst --> wsvc["servicio PRISMA, LocalSystem<br/>C:\ProgramData\PRISMA, firewall 8443"]:::prisma
    deb --> lsvc["prisma.service, usuario prisma<br/>/etc/prisma, /var/lib/prisma"]:::prisma
    sh --> lsvc
    pkg --> msvc["LaunchDaemon, usuario _prisma<br/>/Library/Application Support/PRISMA"]:::prisma
    bin --> man["instalación manual:<br/>prisma servicio instalar"]:::prisma
    man --> wsvc & lsvc & msvc
    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
```

- Los tres instaladores terminan en el mismo `internal/servicio`; `prisma servicio instalar` hace lo mismo sin instalador.
- La unidad de systemd del `.deb` es idéntica a la plantilla de `internal/servicio` (lo verifica una prueba).
- En Windows, las carpetas de datos y bitácoras quedan solo para SYSTEM y Administradores.
- Las rutas completas por sistema están en [operacion/README.md](../operacion/README.md#dónde-está-cada-cosa).

**Fuentes en el código:**

- internal/servicio/servicio.go
- internal/servicio/plantillas.go
- internal/servicio/sistema_windows.go
- internal/servicio/sistema_linux.go
- internal/servicio/sistema_darwin.go
- internal/rutas/
- empaquetado/windows/prisma.iss
- empaquetado/linux/
- empaquetado/macos/
- Makefile

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
