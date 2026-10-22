# Diagnóstico: «no imprime»

¿Por qué no salió el ticket y qué hacer? Árbol de decisión para soporte y para el dueño del negocio.

```mermaid
flowchart TD
    s["no imprime"]:::externo --> r{"¿la aplicación recibe<br/>una respuesta del puente?"}:::prisma
    r -- "no: error de red" --> red["¿abre https://ip:8443/v1/salud<br/>sin aviso de certificado?"]:::externo
    red -- "aviso de certificado" --> c1["instalar el certificado<br/>en ese equipo"]:::seguridad
    red -- "no carga" --> c2["puente apagado, otra red o firewall:<br/>prisma servicio estado"]:::prisma
    red -- "sí carga" --> c3["origen no autorizado:<br/>aprobarlo en Solicitudes"]:::seguridad
    r -- "sí, con error" --> e{"código"}:::prisma
    e -- "401 · 403" --> c4["llave, origen o permisos de la aplicación:<br/>revisar en Aplicaciones"]:::seguridad
    e -- "503 IMPRESORA_INALCANZABLE · 504 TIEMPO_AGOTADO" --> c5["impresora apagada, sin papel, tapa abierta,<br/>Bluetooth fuera de alcance"]:::externo
    e -- "503 PUERTO_OCUPADO · PERMISO_DENEGADO" --> c6["otro programa usa el puerto,<br/>o faltan grupos lp/dialout"]:::prisma
    e -- "503 PUENTE_OCUPADO" --> c7["otro programa abrió prisma.db"]:::datos
    r -- "sí, ok, pero no salió" --> ok{"¿impresora de red?"}:::prisma
    ok -- sí --> c8["activar Confirmar impresión"]:::seguridad
    ok -- "no, USB en Windows" --> c9["revisar la cola de Windows:<br/>trabajos atascados o sin conexión"]:::externo
    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
```

- La primera pregunta separa los problemas de **llegar** al puente (red, certificado, origen) de los de **imprimir** (impresora, permisos, base).
- Desde un navegador, un origen no aprobado se ve como error de red: si `/v1/salud` carga, casi siempre es eso.
- Con la advertencia `IMPRESORA_FUERA_DE_LINEA`, el ticket ya está en la impresora: se imprime al cerrar la tapa o poner papel. **No reenviar.**
- Las soluciones completas: [administracion/problemas-comunes.md](../administracion/problemas-comunes.md), [instalacion/problemas-comunes.md](../instalacion/problemas-comunes.md) e [integradores/problemas-comunes.md](../integradores/problemas-comunes.md).

**Fuentes en el código:**

- internal/conector/conector.go (códigos `Err…`)
- internal/conector/red.go
- internal/conector/usb_windows.go
- internal/servidor/servidor.go (`ResponderError`)
- internal/seguridad/autorizacion.go
- sdk/js/src/prisma.mjs (`_diagnosticarNavegador`)
- docs/referencia/codigos-de-error.md

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
