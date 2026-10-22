# Cómo decide el puente si acepta una petición

¿Se acepta esta petición y con qué código se rechaza?

```mermaid
flowchart TD
    P["Petición a /v1/imprimir"] --> IP{"¿Supera el límite por IP?"}
    IP -->|Sí| R429["429 DEMASIADAS_PETICIONES"]
    IP -->|No| O{"¿Trae Origin?"}
    O -->|Sí| OA{"¿Pertenece a una<br/>aplicación activa?"}
    OA -->|No| R403O["403 ORIGEN_NO_AUTORIZADO<br/>queda como solicitud pendiente"]
    OA -->|"Sí, desactivada"| R403D["403 CLIENTE_DESACTIVADO"]
    OA -->|Sí| L{"¿Trae llave pk_?"}
    O -->|No| LS{"¿Trae llave pk_?"}
    LS -->|No| F0{"¿Documento firmado<br/>con cliente_id?"}
    F0 -->|No| R401C["401 CREDENCIALES_REQUERIDAS"]
    F0 -->|Sí| N3["Cliente de nivel 3<br/>identificado por la firma"]
    LS -->|Sí| LV
    L -->|Sí| LV{"¿Válida y de esta aplicación?"}
    LV -->|No| R401L["401 LLAVE_INVALIDA"]
    LV -->|Sí| NIV
    L -->|No| NIV{"Nivel de la aplicación"}
    NIV -->|"1 desde fuera del navegador"| R403R["403 ORIGEN_REQUERIDO"]
    NIV -->|"2, o 3 que exige llave, sin llave"| R401R["401 LLAVE_REQUERIDA"]
    NIV -->|"1 con Origin, o 2 con llave"| PER
    NIV -->|3| N3
    N3 --> FV{"¿Firma válida, vigente<br/>y del mismo cliente?"}
    FV -->|"Falta"| R401F["401 FIRMA_REQUERIDA"]
    FV -->|"No verifica"| R401FI["401 FIRMA_INVALIDA"]
    FV -->|"Venció"| R401FE["401 FIRMA_EXPIRADA"]
    FV -->|Sí| PER{"¿Impresora y operaciones<br/>permitidas?"}
    PER -->|No| R403P["403 IMPRESORA_NO_PERMITIDA<br/>u OPERACION_NO_PERMITIDA"]
    PER -->|Sí| OK["Validación completa,<br/>render e impresión"]
```

Entre la identificación y la firma también se aplican el límite por cliente y las comprobaciones de tamaño, JSON y versión; ver [secuencia-imprimir.md](secuencia-imprimir.md).

**Basado en:**

- internal/seguridad/autorizacion.go (`Identificar`, `VerificarDocumento`, `VerificarPermisos`)
- internal/seguridad/firma.go
- internal/servidor/api.go (`identificar`, `autorizarDocumento`)
