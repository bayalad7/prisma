# Pantallas del panel

¿Qué pantallas tiene el panel y qué se puede hacer en cada una? Para el dueño del negocio y para soporte.

```mermaid
flowchart LR
    acceso["acceso: crear contraseña, entrar,<br/>cambiar contraseña"]:::seguridad --> inicio
    inicio["inicio: impresoras, trabajos de hoy, solicitudes,<br/>avisos de versión y de certificado"]:::prisma
    inicio --> imp["impresoras: agregar en 3 pasos, probar conexión,<br/>diagnóstico, páginas de códigos, editar, eliminar"]:::prisma
    inicio --> apps["aplicaciones: crear, nivel, orígenes, impresoras,<br/>operaciones sensibles, regenerar o revocar llave"]:::seguridad
    inicio --> sol["solicitudes: aprobar con aplicación nueva o<br/>existente, rechazar, olvidar"]:::seguridad
    inicio --> trab["trabajos: historial filtrable,<br/>sin contenido de tickets"]:::datos
    inicio --> rec["imágenes: vista previa en 1 bit,<br/>guardar, eliminar"]:::datos
    inicio --> aj["ajustes: nombre, límites, retención, red local,<br/>actualizaciones, certificado, respaldo, restaurar"]:::prisma
    inicio --> ay["ayuda: confiar en el certificado por sistema,<br/>enlaces a la documentación"]:::externo
    sol -. "aprobar" .-> apps
    imp -. "detectar o escanear la red" .-> det["internal/conector<br/>detección"]:::prisma
    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
```

- La barra de navegación lleva las 8 secciones; *Inicio* es la de entrada después de *acceso*.
- *Acceso* es el primer acceso (con código desde otro equipo), el inicio de sesión y el cambio de contraseña obligatorio tras restablecerla ([16-panel-sesion.md](16-panel-sesion.md)).
- Toda acción usa la API interna `/admin/api/…`, con sesión y CSRF; no es una API pública.
- *Ayuda* enlaza la documentación de la versión instalada (`blob/v<versión>/docs/`).

**Fuentes en el código:**

- internal/admin/web/panel.js (`SECCIONES`, `VISTAS`, `vistaInicio`, `vistaAyuda`, `formularioImpresora`)
- internal/admin/admin.go (`Handler`: rutas de `/admin/api`)
- internal/admin/impresoras.go
- internal/admin/clientes.go
- internal/admin/ajustes.go

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
