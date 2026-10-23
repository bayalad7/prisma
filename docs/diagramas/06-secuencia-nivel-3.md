---
titulo: "Imprimir en el nivel 3 (firma)"
descripcion: "Cómo imprime una aplicación que firma cada ticket en su servidor"
audiencia: [integrador]
pantallas: [aplicaciones]
secciones_manual: [integradores/firmar-documentos]
---
# Imprimir en el nivel 3 (firma)

¿Quién firma qué, y qué comprueba el puente? Para integradores con backend que necesitan que el ticket no se pueda alterar ni repetir.

```mermaid
sequenceDiagram
    autonumber
    box rgb(232,240,254) aplicación
        participant back as backend (llave privada)
        participant nav as navegador o servidor
    end
    box rgb(230,244,234) PRISMA
        participant srv as internal/servidor
    end
    box rgb(253,232,232) seguridad
        participant seg as internal/seguridad
    end

    back->>back: documento con cliente_id, emitido y expira
    back->>back: firma Ed25519 de la forma canónica JCS sin el campo firma
    back->>nav: documento firmado
    nav->>srv: POST /v1/imprimir, con o sin Origin
    srv->>seg: Identificar (origen y llave, si vienen)
    srv->>seg: VerificarDocumento
    seg->>seg: cliente_id coincide y es de nivel 3
    seg->>seg: firma válida con la llave pública
    seg->>seg: vigencia entre 0 y 120 s, emitido no en el futuro (30 s de tolerancia)
    seg->>seg: expira no pasó (30 s de tolerancia)
    alt todo correcto
        srv-->>nav: 200 impreso
    else falta la firma
        srv-->>nav: 401 FIRMA_REQUERIDA
    else no verifica o cliente_id ajeno
        srv-->>nav: 401 FIRMA_INVALIDA
    else venció
        srv-->>nav: 401 FIRMA_EXPIRADA
    end
```

- El puente solo guarda la llave **pública**: una base filtrada no permite falsificar documentos.
- Sin `Origin` ni llave, el cliente se identifica por el `cliente_id` del documento firmado.
- Un cliente de nivel 3 puede exigir también su llave `pk_` (`exigir_llave`); sin ella responde `LLAVE_REQUERIDA`.
- La repetición la frena la idempotencia: el mismo `id` no se imprime dos veces en 10 minutos ([24-idempotencia.md](24-idempotencia.md)).
- La guía para implementarlo: [integradores/firmar-documentos.md](../integradores/firmar-documentos.md).

**Fuentes en el código:**

- internal/seguridad/autorizacion.go (`VerificarDocumento`)
- internal/seguridad/firma.go (`VerificarFirma`, JCS)
- internal/contrato/reglas.go (`VigenciaMaxima`)
- sdk/firmadores/
- testdata/vector-firma/

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
