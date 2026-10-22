# Orden de autorización y validación

¿En qué orden comprueba el puente una petición, y con qué código rechaza en cada paso? Para integradores que depuran un error y para auditores.

```mermaid
flowchart TD
    p["POST /v1/imprimir"]:::externo --> ip{"límite por IP"}:::seguridad
    ip -- "429 DEMASIADAS_PETICIONES" --> x(["rechazo"]):::externo
    ip --> id{"identificar: origen y llave"}:::seguridad
    id -- "403 ORIGEN_NO_AUTORIZADO, CLIENTE_DESACTIVADO, ORIGEN_REQUERIDO<br/>401 LLAVE_REQUERIDA, LLAVE_INVALIDA" --> x
    id --> lc{"límite por cliente"}:::seguridad
    lc -- "429 DEMASIADAS_PETICIONES" --> x
    lc --> cu{"cuerpo, JSON y versión"}:::prisma
    cu -- "413 DOCUMENTO_DEMASIADO_GRANDE<br/>400 JSON_INVALIDO, VERSION_NO_SOPORTADA" --> x
    cu --> fi{"firma"}:::seguridad
    fi -- "401 CREDENCIALES_REQUERIDAS, FIRMA_REQUERIDA,<br/>FIRMA_INVALIDA, FIRMA_EXPIRADA" --> x
    fi --> rep{"¿id repetido?"}:::datos
    rep -- "200 duplicado · 409 ID_REUTILIZADO" --> x
    rep --> pe{"permisos"}:::seguridad
    pe -- "403 IMPRESORA_NO_PERMITIDA, OPERACION_NO_PERMITIDA" --> x
    pe --> va{"esquema, reglas y perfil"}:::prisma
    va -- "400 DOCUMENTO_INVALIDO y otros · 404 IMPRESORA_NO_ENCONTRADA<br/>422 FILA_NO_CABE, OPERACION_NO_SOPORTADA" --> x
    va --> co["render, cola e impresión"]:::prisma
    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
```

- El puente se detiene en el primer paso que falla. El orden garantiza que nada costoso ocurra antes de saber quién envía el documento.
- *Rechazo* agrupa todas las salidas; el borde dice el estado HTTP y el código. «200 duplicado» no es un error: responde el trabajo original sin imprimir.
- `CREDENCIALES_REQUERIDAS` sale en el paso de firma: si no hubo origen ni llave, solo un documento firmado puede identificar al cliente.
- El detalle de la identificación está en [07-orden-autorizacion-detalle.md](07-orden-autorizacion-detalle.md). Los errores de impresión (503, 504) están en [23-diagnostico-problemas.md](23-diagnostico-problemas.md).
- Todos los códigos: [referencia/codigos-de-error.md](../referencia/codigos-de-error.md).

**Fuentes en el código:**

- internal/servidor/api.go (`autorizarDocumento`, `imprimir`, `responderSiRepetido`)
- internal/servidor/servidor.go (`limitarIP`, `ResponderError`)
- internal/seguridad/autorizacion.go
- internal/contrato/validar.go (`LeerSobre`, `Decodificar`)

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
