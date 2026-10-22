# Pantallas del panel

¿Qué pantallas tiene el panel y qué se puede hacer en cada una? Para el dueño del negocio y para soporte.

```mermaid
flowchart LR
    acceso["acceso: crear contraseña, entrar,<br/>cambiar contraseña"]:::seguridad --> inicio
    subgraph operacion["Operación"]
        inicio["/admin/ · inicio: primeros pasos, impresoras,<br/>solicitudes, hoy, certificado, últimos trabajos"]:::prisma
        imp["/admin/impresoras: agregar (física o virtual), probar,<br/>diagnóstico, páginas de códigos, tickets virtuales"]:::prisma
        sol["/admin/solicitudes: aprobar con aplicación<br/>nueva o existente, rechazar, olvidar"]:::seguridad
        trab["/admin/trabajos: historial filtrable, detalle<br/>con qué pasó y qué hacer; sin contenido de tickets"]:::datos
    end
    subgraph integracion["Integración"]
        apps["/admin/aplicaciones: crear, nivel, orígenes, impresoras,<br/>operaciones sensibles, regenerar o revocar llave"]:::seguridad
        rec["/admin/imagenes: vista previa en 1 bit,<br/>guardar, eliminar"]:::datos
    end
    subgraph sistema["Sistema"]
        aj["/admin/ajustes: nombre, límites, retención, red local,<br/>actualizaciones, certificado, respaldo, restaurar"]:::prisma
        ay["/admin/ayuda: confiar en el certificado,<br/>enlaces a la documentación"]:::externo
    end
    inicio --> imp
    inicio --> sol
    inicio --> trab
    sol -. "aprobar" .-> apps
    imp -. "detectar o escanear la red" .-> det["internal/conector<br/>detección"]:::prisma
    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
```

- La navegación agrupa las 8 secciones en *Operación*, *Integración* y *Sistema*; *Inicio* es la de entrada después de *acceso*. En celular, el menú se despliega y una barra inferior lleva a Inicio, Impresoras, Solicitudes (con el número de pendientes), Trabajos y Ayuda.
- Cada sección tiene su ruta: el puente sirve el panel en todas (`vistasPanel`) y `panel.js` dibuja la pantalla sin recargar; Atrás y Adelante funcionan. Los enlaces de 0.1.0 con `#` (`#clientes`, `#recursos`, `#impresoras`…) llevan a la ruta nueva.
- *Primeros pasos* sale de los datos del puente (impresoras, aplicaciones y las marcas `panel.primera_prueba` y `panel.acentos_probados`), así que se ve igual en cualquier equipo; ocultarla solo afecta al navegador.
- Atajos: `g` y la letra de la sección (por ejemplo `g p`, Impresoras); `?` los muestra y permite desactivarlos.
- *Acceso* es el primer acceso (con código desde otro equipo), el inicio de sesión y el cambio de contraseña obligatorio tras restablecerla ([16-panel-sesion.md](16-panel-sesion.md)).
- Toda acción usa la API interna `/admin/api/…`, con sesión y CSRF; no es una API pública.
- *Ayuda* enlaza la documentación de la versión instalada (`blob/v<versión>/docs/`).

**Fuentes en el código:**

- internal/admin/web/panel.js (`SECCIONES`, `HASH_VIEJOS`, `VISTAS`, `mostrarApp`, `vistaInicio`, `vistaAyuda`, `formularioImpresora`)
- internal/admin/web/sistema.js (`marco`)
- internal/admin/admin.go (`Handler`: rutas de `/admin/api`; `vistasPanel`)
- internal/admin/impresoras.go
- internal/admin/clientes.go
- internal/admin/ajustes.go

**Última verificación:** PRISMA 0.2.0 en desarrollo (rama `develop`), 22 de octubre de 2026.
