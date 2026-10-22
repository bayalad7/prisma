# Diagramas

Un diagrama Mermaid por archivo. GitHub los muestra directamente. Cada uno dice qué pregunta responde y en qué archivos de código se basa: si ese código cambia, hay que revisar el diagrama.

| Diagrama | Responde | Tipo |
|---|---|---|
| [Contexto del sistema](contexto.md) | ¿Quién habla con el puente y por dónde? Nada más sale a Internet. | `flowchart` |
| [Componentes](componentes.md) | ¿Qué paquete depende de cuál? Solo dependencias entre paquetes del proyecto, según `go list ./...`. | `flowchart` |
| [Secuencia de `POST /v1/imprimir`](secuencia-imprimir.md) | ¿Qué pasa, en orden, desde que llega la petición hasta la respuesta? | `sequenceDiagram` |
| [Cómo decide el puente si acepta una petición](autorizacion.md) | ¿Se acepta esta petición y con qué código se rechaza? | `flowchart` |
| [Qué nivel de seguridad elegir](elegir-nivel.md) | ¿Qué nivel le conviene a mi aplicación? | `flowchart` |
| [Estados de un trabajo](estados-trabajo.md) | ¿Por qué estados pasa un trabajo? | `stateDiagram-v2` |
| [Idempotencia](idempotencia.md) | ¿Qué responde el puente si llega el mismo `id`? | `flowchart` |
| [Estados de una impresora](estados-impresora.md) | ¿Cuándo una impresora está en línea, fuera de línea, ocupada o en estado desconocido? | `stateDiagram-v2` |
| [Confirmar impresión](confirmar-impresion.md) | ¿Cómo confirma el puente que el ticket llegó a la impresora? | `sequenceDiagram` |
| [Firma de nivel 3](firma-nivel3.md) | ¿Quién firma qué en el nivel 3, y qué comprueba el puente? | `sequenceDiagram` |
| [Modelo de datos](modelo-datos.md) | ¿Qué guarda la base y cómo se relacionan las tablas? | `erDiagram` |
| [Solicitudes de origen](solicitudes-origen.md) | ¿Qué pasa con un origen desconocido que intenta imprimir? | `stateDiagram-v2` |
| [Sesión del panel](sesion-panel.md) | ¿Cómo se entra al panel y cómo se protege la sesión? | `sequenceDiagram` |
| [Ciclo del certificado](certificado.md) | ¿Qué hace el puente con su certificado al arrancar y mientras corre? | `flowchart` |
| [Actualización](actualizacion.md) | ¿Cómo se actualiza el puente sin riesgo? | `sequenceDiagram` |
| [Despliegue por sistema](despliegue.md) | ¿Cómo queda instalado el puente en cada sistema? | `flowchart` |
| [Publicación de una versión](publicacion.md) | ¿Qué hace el workflow al empujar una etiqueta `v*`? | `flowchart` |
| [Integración continua](ci.md) | ¿Qué se comprueba en cada push a `main` o `develop` y en cada pull request? | `flowchart` |
| [CORS y red local](cors.md) | ¿Por qué el navegador a veces no ve la respuesta del puente? | `sequenceDiagram` |
