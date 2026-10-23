# Documentación de PRISMA

PRISMA es un puente de impresión. Se instala en un equipo de la red local de un negocio, recibe documentos JSON con las operaciones de un ticket y los imprime en impresoras térmicas por red, USB o Bluetooth. Esta documentación corresponde a la versión **0.1.0** ([registro de cambios](../CHANGELOG.md)).

## Según quién eres

### Integras PRISMA en tu aplicación

| Para… | Lee |
|---|---|
| Imprimir tu primer ticket en 5 minutos | [integradores/inicio-rapido.md](integradores/inicio-rapido.md) |
| Elegir el nivel de seguridad (origen, llave, firma) | [integradores/niveles-de-seguridad.md](integradores/niveles-de-seguridad.md) |
| Usar el SDK JavaScript | [integradores/sdk.md](integradores/sdk.md) |
| Firmar documentos desde tu backend (nivel 3) | [integradores/firmar-documentos.md](integradores/firmar-documentos.md) |
| Llamar a la API sin el SDK | [referencia/api-v1.md](referencia/api-v1.md) · [openapi.yaml](referencia/openapi.yaml) |
| Saber qué lleva un documento | [referencia/contrato-documento-v1.md](referencia/contrato-documento-v1.md) |
| Resolver un error de conexión, certificado o CORS | [integradores/problemas-comunes.md](integradores/problemas-comunes.md) |

Todo para integradores: [integradores/](integradores/README.md).

### Instalas PRISMA en un negocio

| Para… | Lee |
|---|---|
| Instalarlo en Windows, Raspberry Pi, Linux o Mac | [instalacion/](instalacion/README.md) |
| Preparar impresoras USB, de red o Bluetooth | [instalacion/impresoras.md](instalacion/impresoras.md) |
| Que los equipos confíen en el certificado HTTPS | [instalacion/https-y-certificados.md](instalacion/https-y-certificados.md) |
| Resolver un problema de instalación o de arranque | [instalacion/problemas-comunes.md](instalacion/problemas-comunes.md) |

### Administras un puente (dueño del negocio)

| Para… | Lee |
|---|---|
| Usar el panel: primer acceso y secciones | [administracion/](administracion/README.md) |
| Agregar y probar impresoras | [administracion/impresoras.md](administracion/impresoras.md) |
| Autorizar aplicaciones y revisar solicitudes | [administracion/aplicaciones-y-solicitudes.md](administracion/aplicaciones-y-solicitudes.md) |
| Ajustes, certificado y respaldo | [administracion/ajustes.md](administracion/ajustes.md) |
| Resolver un problema de impresión o del panel | [administracion/problemas-comunes.md](administracion/problemas-comunes.md) |
| Respaldar, actualizar, leer la bitácora y recuperarte de una falla | [operacion/](operacion/README.md) |

### Desarrollas PRISMA

| Para… | Lee |
|---|---|
| Verlo funcionando sin impresora, desde el código | [desarrollo/primeros-pasos.md](desarrollo/primeros-pasos.md) |
| Compilar, probar y las herramientas de desarrollo | [desarrollo/](desarrollo/README.md) · [desarrollo/pruebas.md](desarrollo/pruebas.md) |
| Contribuir: ramas, commits y pull requests | [CONTRIBUTING.md](../CONTRIBUTING.md) |
| Entender cómo está construido | [arquitectura/](arquitectura/README.md) · [diagramas/](diagramas/README.md) |
| Saber por qué se decidió algo | [adr/](adr/README.md) |
| Publicar una versión | [desarrollo/publicar.md](desarrollo/publicar.md) |
| Probar antes de publicar (matriz manual y reportes) | [desarrollo/pruebas/](desarrollo/pruebas/README.md) |

## Seguridad

El modelo de amenazas, las políticas y cómo reportar una vulnerabilidad: [seguridad/](seguridad/README.md) y [SECURITY.md](../SECURITY.md).

## Referencia

| Documento | Contenido |
|---|---|
| [referencia/api-v1.md](referencia/api-v1.md) · [openapi.yaml](referencia/openapi.yaml) | API pública `/v1`. |
| [referencia/contrato-documento-v1.md](referencia/contrato-documento-v1.md) | El documento JSON y sus reglas. |
| [referencia/codigos-de-error.md](referencia/codigos-de-error.md) | Todos los códigos de error y advertencias. |
| [referencia/cli.md](referencia/cli.md) | Todos los comandos y opciones de `prisma`. |
| [referencia/configuracion.md](referencia/configuracion.md) | `config.toml`. |
| [referencia/glosario.md](referencia/glosario.md) | Los términos de PRISMA. |

## Estado

| Parte | Estado |
|---|---|
| Puente, API `/v1`, panel, CLI, conectores y seguridad por niveles | ✅ Publicado en 0.1.0 |
| Instaladores (Windows, `.deb`, Raspberry Pi, `.pkg`) y `prisma actualizar` | ✅ Publicado en 0.1.0, sin firma de código |
| SDK JavaScript y firmadores de nivel 3 | ✅ Publicado en 0.1.0, en el repositorio y por CDN |
| SDK en npm | ⏳ Pendiente |
| Negociación de versiones del documento | ⏳ Política futura ([contrato §10](referencia/contrato-documento-v1.md)) |
| Certificados automáticos (ACME) | ⏳ Previsto ([https-y-certificados.md](instalacion/https-y-certificados.md)) |

La documentación marca con **⏳** lo que todavía no existe.
