---
titulo: "Solicitudes de origen"
descripcion: "Qué pasa con una página desconocida que intenta imprimir"
audiencia: [usuario, desarrollador]
pantallas: [solicitudes]
secciones_manual: [administracion/aplicaciones-y-solicitudes]
---
# Solicitudes de origen

¿Qué pasa con un origen desconocido que intenta imprimir? Para el dueño que revisa *Solicitudes* y para quien audita que la lista no se pueda llenar.

```mermaid
stateDiagram-v2
    [*] --> pendiente: un origen desconocido intenta imprimir
    pendiente --> pendiente: nuevo intento, se cuenta
    pendiente --> aprobado: el dueño la aprueba
    pendiente --> rechazado: el dueño la rechaza
    rechazado --> rechazado: nuevo intento, se cuenta sin avisar
    aprobado --> [*]: el origen pasa a la aplicación y la solicitud se borra
    pendiente --> [*]: el dueño la olvida, 30 días sin intentos o reemplazada
    rechazado --> [*]: el dueño la olvida o 90 días sin intentos

    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
    class pendiente externo
    class aprobado prisma
    class rechazado seguridad
```

- *Aprobar* crea una aplicación nueva (nivel 1 o 2) o agrega el origen a una existente; la fila de la solicitud se borra en la misma transacción.
- La purga es horaria. Caben 200 solicitudes: un origen nuevo **reemplaza** a la pendiente con la actividad más antigua; si todas son rechazadas, el nuevo no se registra.
- Con más de 150, el panel muestra un aviso: muchos orígenes desconocidos suelen indicar que alguien está probando el puente.
- *Olvidar* la borra; si el origen vuelve a intentarlo, aparece de nuevo como pendiente.

**Fuentes en el código:**

- internal/almacen/solicitudes.go (`Registrar`, `Aprobar`, `Rechazar`, `Eliminar`, `Purgar`, `MaxSolicitudes`)
- internal/admin/clientes.go (`aprobarSolicitud`, `rechazarSolicitud`, `olvidarSolicitud`)
- cmd/prisma/ejecutar.go (`mantenimiento`)
- internal/seguridad/autorizacion.go (`ClienteDeOrigen`)

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
