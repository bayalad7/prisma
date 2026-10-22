# Solicitudes de origen

¿Qué pasa con un origen desconocido que intenta imprimir?

```mermaid
stateDiagram-v2
    [*] --> pendiente: un origen desconocido intenta imprimir
    pendiente --> pendiente: nuevo intento (se cuenta)
    pendiente --> aprobada: el dueño la aprueba
    pendiente --> rechazada: el dueño la rechaza
    rechazada --> rechazada: nuevo intento (se cuenta, sin avisar)
    pendiente --> [*]: el dueño la olvida
    rechazada --> [*]: el dueño la olvida
    pendiente --> [*]: 30 días sin intentos (purga horaria)
    rechazada --> [*]: 90 días sin intentos (purga horaria)
    pendiente --> [*]: reemplazada al llegar al tope de 200
    aprobada --> [*]: el origen pasa a la aplicación y la solicitud se borra
```

Al llegar al tope, un origen nuevo reemplaza a la pendiente con la actividad más antigua. Si todas son rechazadas, el origen nuevo no se registra.

**Basado en:**

- internal/almacen/solicitudes.go (`Registrar`, `Aprobar`, `Rechazar`, `Eliminar`, `Purgar`, `MaxSolicitudes`)
- internal/admin/clientes.go (`aprobarSolicitud`, `rechazarSolicitud`, `olvidarSolicitud`)
- cmd/prisma/ejecutar.go (`mantenimiento`)
