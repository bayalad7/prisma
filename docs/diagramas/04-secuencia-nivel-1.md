---
titulo: "Imprimir en el nivel 1 (origen)"
descripcion: "Cómo imprime una página web aprobada solo por su dirección"
audiencia: [integrador]
pantallas: [aplicaciones]
secciones_manual: [integradores/niveles-de-seguridad]
---
# Imprimir en el nivel 1 (origen)

¿Qué intercambian el navegador y el puente cuando una página sin llave imprime? Para integradores que publican una página web sin backend.

```mermaid
sequenceDiagram
    autonumber
    box rgb(232,240,254) navegador
        participant pag as página web
        participant nav as navegador
    end
    box rgb(230,244,234) PRISMA
        participant srv as internal/servidor
    end
    box rgb(253,232,232) seguridad
        participant seg as internal/seguridad
    end
    box rgb(255,244,224) datos
        participant alm as internal/almacen
    end

    pag->>nav: fetch POST /v1/imprimir
    nav->>srv: OPTIONS con Origin (preflight)
    srv->>seg: ClienteDeOrigen(origen)
    seg->>alm: cliente dueño del origen
    alt origen aprobado
        srv-->>nav: 204, Allow-Origin exacto, Max-Age 600
        nav->>srv: POST con Origin, sin Authorization
        srv->>seg: Identificar(origen, sin llave)
        seg-->>srv: cliente de nivel 1
        srv-->>nav: 200 impreso, Allow-Origin exacto
        nav-->>pag: resultado
    else origen desconocido
        seg->>alm: registrar solicitud pendiente
        srv-->>nav: 403 sin cabeceras CORS
        nav-->>pag: TypeError: Failed to fetch
        pag->>srv: GET /v1/salud sin CORS (diagnóstico del SDK)
        pag-->>pag: ORIGEN_NO_AUTORIZADO
    end
```

- El navegador envía `Origin` y no deja que la página lo cambie; por eso el nivel 1 protege contra páginas maliciosas, pero no contra curl.
- Con un origen desconocido el puente responde 403 **sin** CORS: la página no puede leer la respuesta y solo ve un error de red. El SDK lo distingue pidiendo `/v1/salud` sin CORS: si el puente responde, el problema es el origen.
- El origen desconocido ya quedó en *Solicitudes* del panel, tanto por el preflight como por la petición.
- Los pasos posteriores (validación, render, cola) son los de [03-flujo-impresion.md](03-flujo-impresion.md).

**Fuentes en el código:**

- internal/servidor/servidor.go (`preflight`, `permitirCORS`)
- internal/seguridad/autorizacion.go (`ClienteDeOrigen`, `Identificar`)
- internal/almacen/solicitudes.go (`Registrar`)
- sdk/js/src/prisma.mjs (`_diagnosticarNavegador`)

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
