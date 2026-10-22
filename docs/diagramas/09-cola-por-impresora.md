# Cola por impresora

¿Qué pasa cuando dos aplicaciones imprimen a la vez, en la misma impresora y en impresoras distintas? Para integradores y desarrolladores que razonan sobre tiempos y concurrencia.

```mermaid
sequenceDiagram
    box rgb(232,240,254) aplicaciones
        participant a as aplicación A
        participant b as aplicación B
    end
    box rgb(230,244,234) internal/cola
        participant qc as goroutine de caja
        participant qk as goroutine de cocina
    end
    box rgb(232,240,254) impresoras
        participant ic as caja
        participant ik as cocina
    end

    a->>qc: trabajo 1 para caja
    b->>qc: trabajo 2 para caja
    b->>qk: trabajo 3 para cocina
    par impresoras distintas, en paralelo
        qc->>ic: trabajo 1
        ic-->>qc: enviado
        qc-->>a: resultado 1
        qc->>ic: trabajo 2 (esperó al 1)
        ic-->>qc: enviado
        qc-->>b: resultado 2
    and
        qk->>ik: trabajo 3
        ik-->>qk: enviado
        qk-->>b: resultado 3
    end
    Note over qc: con 20 pendientes, el siguiente recibe 503 COLA_LLENA
```

- Cada impresora tiene **una goroutine**: imprime en orden de llegada (FIFO) y de a un trabajo. Así dos tickets nunca se mezclan en el papel.
- Impresoras distintas trabajan en paralelo.
- El límite de 20 cuenta los que esperan más el que se imprime; el siguiente recibe `COLA_LLENA` con `Retry-After`.
- Cada trabajo tiene como máximo 60 s. Si la aplicación espera más de 60 s su turno, recibe 202 y el trabajo se imprime igual.
- La misma goroutine prueba la impresora cada 30 s cuando no imprime ([26-estados-impresora.md](26-estados-impresora.md)).

**Fuentes en el código:**

- internal/cola/cola.go (`Registrar`, `Encolar`, `atender`, `imprimir`, `MaxPorImpresora`, `TiempoTrabajo`)
- internal/servidor/api.go (`TiempoRespuesta`)

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
