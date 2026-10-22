# Ciclo del certificado

¿Qué hace el puente con su certificado al arrancar y mientras corre?

```mermaid
flowchart TD
    A["Arranque"] --> M{"modo_tls"}
    M -->|autofirmado| E{"¿Existe tls/certificado.pem?"}
    E -->|No| G["Generar: ECDSA P-256, 825 días,<br/>localhost, nombre del equipo e IPs"]
    E -->|Sí| V{"¿Venció?"}
    V -->|Sí| G
    V -->|No| U["Usarlo"]
    G --> U
    M -->|archivos| F["Cargar certificado y llave<br/>de config.toml"]
    M -->|solo_localhost_http| H["HTTP solo en localhost"]
    U --> C{"Días para vencer"}
    F --> C
    C -->|"vencido"| ERR["Bitácora ERROR y aviso rojo en el panel"]
    C -->|"menos de 60"| WARN["Bitácora WARN y aviso en Inicio"]
    C -->|"60 o más"| OK["Sin aviso"]
    U --> IP{"¿Faltan IPs actuales<br/>en el certificado?"}
    IP -->|Sí| WIP["Bitácora WARN: regenerarlo"]
    R["Regenerar desde el panel o<br/>prisma tls regenerar"] --> G2["Certificado nuevo"] --> RC["Se recarga en unos segundos,<br/>sin reiniciar"]
```

Después de regenerarlo, hay que volver a instalarlo en cada equipo que imprime.

**Basado en:**

- internal/servidor/tls.go (`CargarAutofirmado`, `Vencimiento`, `AvisoVencimiento`, `vigenciaAutofirmado`)
- internal/servidor/escuchar.go (`avisarVencimiento`)
- internal/admin/ajustes.go (`avisoCertificado`, `regenerarTLS`)
- cmd/prisma/tls.go
