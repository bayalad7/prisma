# Imprimir en el nivel 2 (llave)

¿Cómo se comprueba la llave `pk_` de una aplicación? Para integradores con página o backend que usan el nivel por defecto.

```mermaid
sequenceDiagram
    autonumber
    box rgb(232,240,254) aplicación
        participant app as navegador o backend
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

    app->>srv: POST /v1/imprimir, Authorization: Bearer pk_…, Origin si es un navegador
    srv->>seg: Identificar(origen, llave)
    opt trae Origin
        seg->>alm: cliente dueño del origen
    end
    seg->>alm: cliente por prefijo público de la llave
    alt prefijo desconocido o mal formado
        srv-->>app: 401 LLAVE_INVALIDA, sin calcular nada
    end
    seg->>seg: ¿resultado en caché? (10 min)
    opt no está en caché
        seg->>seg: argon2id en tiempo constante, como máximo 2 a la vez
    end
    alt llave válida y del mismo cliente que el origen
        seg-->>srv: cliente de nivel 2
        srv->>srv: límite por cliente, permisos, validación, render, cola
        srv-->>app: 200 impreso
    else sin llave
        srv-->>app: 401 LLAVE_REQUERIDA
    else llave incorrecta o de otra aplicación
        srv-->>app: 401 LLAVE_INVALIDA
    end
```

- Desde un servidor no hay `Origin`: la llave sola identifica al cliente.
- Una llave cuyo prefijo no existe se rechaza **sin** calcular argon2id; el resultado, válido o no, queda en caché 10 minutos. Así una lluvia de llaves inventadas no satura el equipo.
- Si viene `Origin`, la llave debe ser de la **misma** aplicación dueña de ese origen.
- Una llave revocada en el panel responde `LLAVE_REQUERIDA` hasta que se genere otra.

**Fuentes en el código:**

- internal/seguridad/autorizacion.go (`Identificar`, `verificarLlave`, `turnos`, `cache`)
- internal/seguridad/llaves.go
- internal/almacen/clientes.go
- internal/servidor/api.go (`identificar`, `limitarCliente`)

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
