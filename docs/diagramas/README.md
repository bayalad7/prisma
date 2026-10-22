# Diagramas

Un diagrama Mermaid por archivo, numerado. GitHub los muestra sin plugins. Cada uno dice qué pregunta responde y para quién es, explica lo que no es obvio y cita los archivos de código en que se basa: si ese código cambia, hay que revisar el diagrama.

**Colores** (los mismos en todos):

| Color | Clase | Qué representa |
|---|---|---|
| azul | `externo` | Navegadores, aplicaciones, impresoras y servicios de afuera. |
| verde | `prisma` | Paquetes y componentes del puente. |
| naranja | `datos` | Base de datos, archivos y estado guardado. |
| rojo | `seguridad` | Autorización, firmas, certificados y credenciales. |

Los diagramas de secuencia no admiten clases: usan cajas con los mismos colores. El modelo de datos (diagrama ER) no lleva colores.

| # | Diagrama | Tipo | Para quién |
|---|---|---|---|
| 01 | [Contexto del sistema](01-contexto.md) | `flowchart` | integradores, instaladores, desarrolladores |
| 02 | [Componentes y dependencias](02-componentes.md) | `flowchart` | desarrolladores |
| 03 | [Flujo de impresión](03-flujo-impresion.md) | `flowchart` | integradores, desarrolladores |
| 04 | [Imprimir en el nivel 1 (origen)](04-secuencia-nivel-1.md) | `sequenceDiagram` | integradores |
| 05 | [Imprimir en el nivel 2 (llave)](05-secuencia-nivel-2.md) | `sequenceDiagram` | integradores |
| 06 | [Imprimir en el nivel 3 (firma)](06-secuencia-nivel-3.md) | `sequenceDiagram` | integradores |
| 07 | [Orden de autorización y validación](07-orden-autorizacion.md) | `flowchart` | integradores, desarrolladores |
| 07 | [Identificación del cliente, en detalle](07-orden-autorizacion-detalle.md) | `flowchart` | integradores, desarrolladores |
| 08 | [Ciclo de vida de un trabajo](08-ciclo-vida-trabajo.md) | `stateDiagram-v2` | administradores, integradores, desarrolladores |
| 09 | [Cola por impresora](09-cola-por-impresora.md) | `sequenceDiagram` | integradores, desarrolladores |
| 10 | [Solicitudes de origen](10-solicitudes-origen.md) | `stateDiagram-v2` | administradores, desarrolladores |
| 11 | [Modelo de datos](11-modelo-datos.md) | `erDiagram` | desarrolladores, auditores |
| 12 | [Modos de TLS y ciclo del certificado](12-modos-tls.md) | `flowchart` | instaladores, administradores |
| 14 | [Arranque del servicio](14-arranque-servicio.md) | `sequenceDiagram` | desarrolladores, administradores |
| 15 | [Pantallas del panel](15-panel-navegacion.md) | `flowchart` | administradores |
| 16 | [Sesión del panel](16-panel-sesion.md) | `sequenceDiagram` | administradores, auditores |
| 17 | [Detección de impresoras](17-deteccion-impresoras.md) | `flowchart` | instaladores, administradores |
| 18 | [Conectores](18-conectores.md) | `flowchart` | instaladores, desarrolladores |
| 19 | [Despliegue por plataforma](19-despliegue.md) | `flowchart` | instaladores, desarrolladores |
| 20 | [Integración continua y publicación](20-ci-release.md) | `flowchart` | desarrolladores |
| 23 | [Diagnóstico: «no imprime»](23-diagnostico-problemas.md) | `flowchart` | administradores, soporte |
| 24 | [Idempotencia](24-idempotencia.md) | `flowchart` | integradores, desarrolladores |
| 25 | [Confirmar impresión](25-confirmar-impresion.md) | `sequenceDiagram` | instaladores, integradores |
| 26 | [Estados de una impresora](26-estados-impresora.md) | `stateDiagram-v2` | administradores, integradores |
| 27 | [Qué nivel de seguridad elegir](27-elegir-nivel.md) | `flowchart` | integradores |
| 28 | [Actualización](28-actualizacion.md) | `sequenceDiagram` | administradores, auditores |
| 29 | [Impresora virtual](29-impresora-virtual.md) | `flowchart` | dueños, instaladores, integradores |

## Diagramas que no existen en 0.1.0

La lista original incluía tres diagramas sin base en el código. Como cada diagrama debe dibujar lo que el código hace, no se crearon:

| # | Diagrama pedido | Por qué no está |
|---|---|---|
| 13 | Certificado remoto: descarga, validación, recarga y caída del servidor | No hay modo remoto: `modo_tls` solo admite `autofirmado`, `archivos` y `solo_localhost_http` (`internal/config/config.go`). Los tres modos reales están en [12-modos-tls.md](12-modos-tls.md). |
| 21 | Motor WASM: el mismo renderizador en el binario y en el navegador | No hay compilación a WebAssembly. El SDK valida documentos en el navegador, pero no renderiza ESC/POS. |
| 22 | Integración TEDYC → PIAA → navegador → PRISMA | El repositorio no contiene ni menciona esas aplicaciones. El flujo genérico de una aplicación que firma en su backend está en [06-secuencia-nivel-3.md](06-secuencia-nivel-3.md). |

Si alguno se implementa, su diagrama se agrega con ese número.

## Metadatos

Un diagrama puede empezar con un bloque YAML (front-matter) que el panel usará para la galería y la ayuda de cada pantalla: `titulo`, `descripcion`, `audiencia` (`usuario`, `instalador`, `integrador`, `desarrollador`), `pantallas` y `secciones_manual`, con las listas en una línea (`[a, b]`). GitHub lo muestra como una tabla. `go test ./docs` revisa esos campos cuando el bloque existe.

## Validación

`scripts/validar-diagramas.sh` (o `scripts/validar-diagramas.ps1` en Windows) extrae cada bloque `mermaid` de los Markdown del repositorio y lo compila con `@mermaid-js/mermaid-cli`. El CI lo corre en cada cambio y falla si un diagrama no compila. `go test ./docs` comprueba además que cada diagrama tenga sus secciones y que los archivos que cita existan.
