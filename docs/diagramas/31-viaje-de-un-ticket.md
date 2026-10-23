---
titulo: "El viaje de un ticket"
descripcion: "Qué le pasa a un ticket desde que tu aplicación lo manda hasta que sale o falla"
audiencia: [usuario]
pantallas: [trabajos, solicitudes]
secciones_manual: [administracion/trabajos, administracion/solucion-de-problemas]
enlaces: [permiso=administracion/aplicaciones-y-solicitudes, rechazo=/admin/solicitudes, aviso=administracion/trabajos#qué-pasó-con-un-trabajo, fallo=administracion/trabajos#qué-pasó-con-un-trabajo]
---
# El viaje de un ticket

¿Por dónde pasa un ticket y qué significa lo que muestra *Trabajos*? Para el dueño. La versión técnica, con los estados y los códigos, es [08-ciclo-vida-trabajo.md](08-ciclo-vida-trabajo.md).

```mermaid
flowchart TD
    llega["tu aplicación manda un ticket"]:::externo --> permiso{"¿la aplicación tiene<br/>permiso para esa impresora?"}:::seguridad
    permiso -- "no" --> rechazo["no se imprime: si es una página nueva,<br/>aparece en Solicitudes para que decidas"]:::seguridad
    permiso -- "sí" --> turno["espera su turno: cada impresora<br/>imprime un ticket a la vez"]:::prisma
    turno --> recibe{"¿la impresora lo recibe?"}:::prisma
    recibe -- "sí" --> impreso["Impreso"]:::datos
    recibe -- "sí, pero con algo distinto<br/>(sin cuchilla, un acento)" --> aviso["Impreso con aviso:<br/>Trabajos explica qué ajustar"]:::datos
    recibe -- "no: apagada<br/>o desconectada" --> fallo["Falló: Trabajos dice qué revisar;<br/>se puede reenviar sin que salga dos veces"]:::externo
    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
```

- «Impreso» quiere decir que la impresora recibió el ticket. Si la impresora avisa que tiene la tapa abierta o no tiene papel, queda «Impreso con aviso» y sale en cuanto lo resuelves: no hace falta reenviarlo.
- Un ticket que falló no salió, así que la aplicación puede volver a mandarlo. Y si reenvía uno que sí salió (el mismo ticket, con su mismo identificador), PRISMA no lo imprime otra vez.
- PRISMA guarda el resultado de cada ticket los días que indiques en *Ajustes*, nunca su contenido.

**Fuentes en el código:**

- internal/servidor/api.go (`imprimir`, `finalizar`)
- internal/cola/cola.go
- internal/almacen/trabajos.go
- internal/seguridad/autorizacion.go (`VerificarPermisos`)

**Última verificación:** PRISMA 0.2.0 en desarrollo (rama `develop`), 22 de octubre de 2026.
