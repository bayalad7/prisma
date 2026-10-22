---
titulo: "Sesión del panel"
descripcion: "Cómo se entra al panel y cómo se protege la sesión"
audiencia: [usuario, desarrollador]
pantallas: [ajustes]
secciones_manual: [administracion/ajustes]
---
# Sesión del panel

¿Cómo se entra al panel y cómo se protege la sesión? Para administradores y auditores.

```mermaid
sequenceDiagram
    autonumber
    box rgb(232,240,254) dueño
        participant nav as navegador
        participant cli as terminal del equipo
    end
    box rgb(253,232,232) internal/admin
        participant pan as panel /admin
    end

    nav->>pan: GET /admin/api/sesion
    alt el panel aún no tiene contraseña
        alt desde el propio equipo
            nav->>pan: configuracion-inicial con contraseña nueva
        else desde otro equipo
            cli->>cli: prisma admin codigo-inicial (30 min)
            nav->>pan: configuracion-inicial con código y contraseña
            Note over pan: 10 códigos incorrectos anulan el código
        end
    else ya tiene contraseña
        nav->>pan: entrar con la contraseña
        Note over pan: 5 fallos desde una IP: bloqueo de 5 min (ACCESO_BLOQUEADO)
    end
    pan-->>nav: cookie HttpOnly, Secure, SameSite=Strict y token CSRF
    nav->>pan: cambios con X-CSRF-Token y el mismo Origin
    alt token o Origin incorrectos
        pan-->>nav: 403 CSRF_INVALIDO
    end
    Note over pan: la sesión vence a los 30 min sin actividad, a las 12 h,<br/>al reiniciar el puente o al cambiar la contraseña (SESION_REQUERIDA)
```

- Sin contraseña, solo el propio equipo puede configurar el panel sin código: nadie de la red se adueña de un puente recién instalado.
- La contraseña usa argon2id (64 MiB, 3 pasadas), con un solo cálculo a la vez.
- El token de sesión son 32 bytes aleatorios; el puente guarda su SHA-256, en memoria.
- Tras `prisma admin restablecer-contrasena`, toda acción responde `CAMBIO_DE_CONTRASENA_REQUERIDO` hasta que se cambie.
- Por defecto el panel solo responde a IPs privadas y al propio equipo (`ACCESO_SOLO_RED_LOCAL`).
- Cualquier petición con sesión renueva la actividad, salvo las que llevan `X-Prisma-Sondeo: 1`: el rollo en vivo de una impresora virtual consulta cada 2 s con esa cabecera, que comprueba la sesión sin renovarla. El panel avisa 2 minutos antes del cierre por inactividad y solo «Seguir conectado» la renueva; ninguna consulta automática anula el cierre.

**Fuentes en el código:**

- internal/admin/sesiones.go (`inactividadMaxima`, `duracionMaxima`, `cabeceraSondeo`, `fallosPermitidos`, `bloqueo`, `vigenciaCodigo`, `fallosCodigo`)
- internal/admin/admin.go (`Handler`, `redPermitida`)
- internal/admin/web/panel.js (`INACTIVIDAD`, `mostrarAvisoSesion`, `vistaRollo`)
- internal/seguridad/contrasena.go
- cmd/prisma/admin.go

**Última verificación:** PRISMA 0.2.0 en desarrollo (rama `develop`), 22 de octubre de 2026.
