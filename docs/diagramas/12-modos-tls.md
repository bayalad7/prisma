---
titulo: "Modos de TLS y ciclo del certificado"
descripcion: "Cómo funciona el certificado HTTPS del puente"
audiencia: [instalador]
pantallas: [ajustes]
secciones_manual: [instalacion/https-y-certificados]
---
# Modos de TLS y ciclo del certificado

¿Qué hace el puente con HTTPS según `modo_tls`, y qué pasa con su certificado al arrancar y mientras corre? Para instaladores y administradores.

```mermaid
flowchart TD
    a["arranque"]:::prisma --> m{"servidor.modo_tls"}:::prisma
    m -- autofirmado --> e{"¿tls/certificado.pem existe y no venció?"}:::seguridad
    e -- no --> g["generar ECDSA P-256, 825 días,<br/>localhost, nombre del equipo e IPs"]:::seguridad
    e -- sí --> u["usarlo"]:::seguridad
    g --> u
    m -- archivos --> f["cargar certificado y llave<br/>de config.toml"]:::datos
    m -- solo_localhost_http --> h["HTTP sin cifrar,<br/>solo en 127.0.0.1 o ::1"]:::externo
    u --> v{"días para vencer"}:::seguridad
    f --> v
    v -- "vencido" --> err["bitácora ERROR y aviso rojo en el panel"]:::seguridad
    v -- "menos de 60" --> warn["bitácora WARN y aviso en Inicio"]:::seguridad
    u --> ip["si faltan IPs actuales: WARN, regenerarlo"]:::seguridad
    r["regenerar: panel o prisma tls regenerar"]:::prisma --> rc["se recarga en unos segundos, sin reiniciar"]:::prisma
    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
```

- **Solo existen tres modos.** No hay un modo «remoto» ni descarga de certificados desde un servidor en PRISMA 0.1.0: `config.Validar` rechaza cualquier otro valor de `modo_tls`. Por eso no hay diagrama de certificado remoto.
- Con `autofirmado`, un certificado vencido se **regenera al arrancar**; con `archivos`, el puente no puede renovarlo y solo avisa.
- `archivos` también se recarga en caliente: el puente vuelve a leer los archivos cada 5 s si cambian.
- Después de regenerar, hay que volver a instalar el certificado en cada equipo que imprime.

**Fuentes en el código:**

- internal/config/config.go (`TLSAutofirmado`, `TLSArchivos`, `TLSSoloLocalhostHTTP`, validación de `modo_tls`)
- internal/servidor/escuchar.go (`Escuchar`, `avisarVencimiento`)
- internal/servidor/tls.go (`CargarAutofirmado`, `NuevoArchivos`, `Vencimiento`, `IPsSinCubrir`)
- internal/admin/ajustes.go (`avisoCertificado`, `regenerarTLS`)
- cmd/prisma/tls.go

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
