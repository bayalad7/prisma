---
titulo: "Actualización"
descripcion: "Cómo se actualiza el puente sin instalar nada que no venga del proyecto"
audiencia: [usuario, instalador]
pantallas: [inicio, ajustes]
secciones_manual: [operacion/actualizacion]
---
# Actualización

¿Cómo se actualiza el puente sin instalar nada que no venga del proyecto? Para administradores y auditores.

```mermaid
sequenceDiagram
    autonumber
    box rgb(232,240,254) externo
        participant duenio as dueño
        participant gh as GitHub Releases
    end
    box rgb(253,232,232) internal/actualizacion
        participant act as prisma actualizar
    end
    box rgb(230,244,234) PRISMA
        participant svc as internal/servicio
    end

    duenio->>act: prisma actualizar (como administrador)
    act->>gh: última versión estable
    act->>gh: checksums.txt y checksums.txt.sig
    act->>act: verificar la firma Ed25519 con la llave pública embebida
    alt la firma no verifica
        act-->>duenio: error, no se toca nada
    end
    act->>gh: binario de esta plataforma
    act->>act: verificar el SHA-256 contra checksums.txt
    act->>svc: detener el servicio si estaba en marcha
    act->>act: reemplazar el binario
    act->>svc: iniciar
    Note over svc: al arrancar, si hay migraciones, respalda la base antes de migrarla
```

- El panel solo **avisa** de una versión nueva; nada se instala solo.
- Las versiones preliminares (`v1.2.0-rc.1`) no se ofrecen.
- Si no puede reemplazar el binario, deja el anterior y reinicia el servicio. En Windows, el anterior queda como `prisma.exe.old` hasta el siguiente arranque.
- Volver atrás: [operacion/actualizacion.md](../operacion/actualizacion.md#volver-a-la-versión-anterior).

**Fuentes en el código:**

- internal/actualizacion/actualizacion.go
- internal/actualizacion/reemplazar.go
- cmd/prisma/actualizar.go (`actualizarBinario`)
- internal/almacen/respaldo.go (`RespaldarSiHayMigraciones`)
- empaquetado/llave-releases.pub

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
