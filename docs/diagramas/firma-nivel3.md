# Firma de nivel 3

¿Quién firma qué en el nivel 3, y qué comprueba el puente?

```mermaid
sequenceDiagram
    participant B as Backend de la aplicación
    participant N as Navegador o servidor
    participant P as Puente

    Note over B: Guarda la llave privada Ed25519
    Note over P: Guarda solo la llave pública
    B->>B: arma el documento con cliente_id, emitido y expira
    B->>B: forma canónica JCS (RFC 8785) sin el campo firma
    B->>B: firma Ed25519 de esos bytes UTF-8
    B->>N: documento firmado
    N->>P: POST /v1/imprimir
    P->>P: cliente_id coincide con el cliente
    P->>P: verifica la firma con la llave pública
    P->>P: vigencia de 0 a 120 s, 30 s de tolerancia de reloj
    P->>P: el id no se usó con otro contenido (10 min)
    P-->>N: 200, o FIRMA_REQUERIDA, FIRMA_INVALIDA, FIRMA_EXPIRADA
```

La guía para implementarlo está en [integradores/firmar-documentos.md](../integradores/firmar-documentos.md).

**Basado en:**

- internal/seguridad/firma.go (forma canónica con `gowebpki/jcs`)
- internal/contrato/reglas.go (`VigenciaMaxima`)
- sdk/firmadores/*
- testdata/vector-firma/
