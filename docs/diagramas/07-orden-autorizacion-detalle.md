---
titulo: "Identificación del cliente, en detalle"
descripcion: "Cómo sabe el puente qué aplicación envía un ticket"
audiencia: [integrador, desarrollador]
pantallas: [permisos]
secciones_manual: [integradores/niveles-de-seguridad]
---
# Identificación del cliente, en detalle

¿Cómo combina el puente `Origin` y la llave para saber qué aplicación envía la petición? Detalle del segundo paso de [07-orden-autorizacion.md](07-orden-autorizacion.md), para integradores y auditores.

```mermaid
flowchart TD
    o{"¿trae Origin?"}:::externo -- sí --> oa{"¿origen de una<br/>aplicación activa?"}:::seguridad
    oa -- "no: queda como solicitud" --> r1(["403 ORIGEN_NO_AUTORIZADO<br/>o CLIENTE_DESACTIVADO"]):::externo
    oa -- sí --> l{"¿trae llave?"}:::seguridad
    o -- no --> ls{"¿trae llave?"}:::seguridad
    l -- no --> ex{"¿nivel 2, o 3 que exige llave?"}:::seguridad
    ex -- sí --> r2(["401 LLAVE_REQUERIDA"]):::externo
    ex -- no --> ok["cliente del origen"]:::prisma
    l -- sí --> lv{"¿llave válida y de esa misma aplicación?"}:::seguridad
    lv -- no --> r3(["401 LLAVE_INVALIDA"]):::externo
    lv -- sí --> ok
    ls -- no --> sin["sin cliente: solo un documento<br/>firmado puede identificarlo"]:::prisma
    ls -- sí --> lv2{"¿llave válida, aplicación activa<br/>y de nivel 2 o 3?"}:::seguridad
    lv2 -- "no" --> r4(["401 LLAVE_INVALIDA · 403 CLIENTE_DESACTIVADO<br/>· 403 ORIGEN_REQUERIDO (nivel 1)"]):::externo
    lv2 -- sí --> ok2["cliente de la llave"]:::prisma
    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
```

- `ORIGEN_REQUERIDO` aparece cuando llega, sin `Origin`, la llave de una aplicación de nivel 1: ese nivel solo vale desde un navegador.
- Con `Origin` y llave, la llave debe pertenecer a la aplicación dueña del origen; si es de otra, `LLAVE_INVALIDA`.
- *Sin cliente* no es un rechazo todavía: el paso de firma identifica al cliente por el `cliente_id` de un documento firmado, o responde `CREDENCIALES_REQUERIDAS`.

**Fuentes en el código:**

- internal/seguridad/autorizacion.go (`Identificar`, `ClienteDeOrigen`, `verificarLlave`, `desactivado`)

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
