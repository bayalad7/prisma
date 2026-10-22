# Sesión del panel

¿Cómo se entra al panel y cómo se protege la sesión?

```mermaid
sequenceDiagram
    participant N as Navegador del dueño
    participant P as Panel /admin
    participant E as Equipo del puente (CLI)

    N->>P: GET /admin/api/sesion
    alt el panel aún no tiene contraseña
        alt desde el propio equipo
            N->>P: configuracion-inicial con contraseña nueva
        else desde otro equipo
            E->>E: prisma admin codigo-inicial
            E-->>N: código de un solo uso (30 min)
            N->>P: configuracion-inicial con código y contraseña
            Note over P: 10 códigos incorrectos anulan el código
        end
    else ya tiene contraseña
        N->>P: entrar con la contraseña
        Note over P: 5 fallos desde una IP la bloquean 5 min
    end
    P-->>N: cookie de sesión HttpOnly, Secure, SameSite=Strict, y token CSRF
    N->>P: cambios con X-CSRF-Token y el mismo Origin
    Note over P: la sesión vence a los 30 min sin actividad,<br/>a las 12 h, al reiniciar o al cambiar la contraseña
```

**Basado en:**

- internal/admin/sesiones.go
- internal/admin/admin.go (`sesion`, `entrar`, `configuracionInicial`, `conSesion`)
- cmd/prisma/admin.go
