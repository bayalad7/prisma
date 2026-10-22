# Despliegue por sistema

¿Cómo queda instalado el puente en cada sistema?

```mermaid
flowchart LR
    subgraph Windows
        W1["Servicio PRISMA<br/>LocalSystem, reinicio a los 5 s"]
        W2["C:\Program Files\PRISMA\prisma.exe"]
        W3["C:\ProgramData\PRISMA<br/>config.toml · datos · logs<br/>datos y logs: SYSTEM y Administradores"]
        W4["Firewall: 8443/TCP, subred local"]
    end
    subgraph Linux["Linux y Raspberry Pi"]
        L1["prisma.service (systemd)<br/>usuario prisma, grupos lp y dialout"]
        L2["/usr/bin/prisma"]
        L3["/etc/prisma/config.toml<br/>/var/lib/prisma · /var/log/prisma"]
    end
    subgraph macOS
        M1["LaunchDaemon com.github.bayalad7.prisma<br/>usuario oculto _prisma"]
        M2["/usr/local/bin/prisma"]
        M3["/Library/Application Support/PRISMA<br/>/Library/Logs/PRISMA"]
    end
    W1 --> W2 --> W3
    L1 --> L2 --> L3
    M1 --> M2 --> M3
```

La tabla completa de rutas está en [operacion/README.md](../operacion/README.md#dónde-está-cada-cosa).

**Basado en:**

- internal/servicio/servicio.go
- internal/servicio/plantillas.go
- internal/servicio/sistema_windows.go
- internal/servicio/sistema_linux.go
- internal/servicio/sistema_darwin.go
- internal/rutas/*
- empaquetado/*
