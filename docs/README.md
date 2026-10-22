# Documentación de PRISMA

PRISMA es un puente de impresión. Se instala en un equipo de la red local de un negocio, recibe documentos JSON con las operaciones de un ticket y los imprime en impresoras térmicas por red, USB o Bluetooth.

## Por dónde empezar

| Si quieres… | Lee |
|---|---|
| Ver PRISMA funcionando en 5 minutos, sin impresora | [primeros-pasos.md](primeros-pasos.md) |
| Instalarlo en el equipo de un negocio (Windows, Raspberry Pi, Linux, Mac) | [instalacion.md](instalacion.md) |
| Configurar impresoras y aplicaciones desde el navegador | [panel.md](panel.md) |
| Conectar una impresora (red, USB, Bluetooth) | [conectores.md](conectores.md) |
| Conocer todos los comandos y opciones | [cli.md](cli.md) |
| Configurar `config.toml` | [configuracion.md](configuracion.md) |
| Saber cómo está construido por dentro | [arquitectura.md](arquitectura.md) |
| Contribuir al código | [desarrollo.md](desarrollo.md) |
| Publicar una versión (llaves, CI, instaladores) | [publicar.md](publicar.md) |
| Probar antes de publicar (matriz manual y último reporte) | [pruebas/](pruebas/README.md) |
| Integrar PRISMA en tu aplicación (SDK, niveles, firmas) | [integradores/](integradores/README.md) |
| Generar documentos JSON desde tu aplicación | [contrato-documento-v1.md](contrato-documento-v1.md) |
| Imprimir desde tu aplicación (endpoints, autorización, CORS, errores) | [api-v1.md](api-v1.md) |
| Configurar HTTPS y confiar en el certificado | [tls.md](tls.md) |
| Resolver un problema | [solucion-de-problemas.md](solucion-de-problemas.md) |

## Estado actual

PRISMA está en desarrollo. Esto es lo que ya funciona y lo que falta:

| Parte | Estado |
|---|---|
| Contrato «documento v1»: esquema, validación y ejemplos | ✅ Listo |
| Renderizador ESC/POS: páginas de códigos, QR, códigos de barras e imágenes | ✅ Listo |
| Conectores (red, USB, Bluetooth, archivo) y detección de impresoras | ✅ Listo |
| Cola por impresora y estado (en línea / fuera de línea / ocupada) | ✅ Listo |
| Base de datos, configuración y CLI | ✅ Listo |
| Herramientas de desarrollo: impresora virtual, previsualización y decodificador | ✅ Listo |
| Servidor HTTPS (autofirmado, archivos o solo localhost) y API pública `/v1` | ✅ Listo |
| Autorización: orígenes, llaves `pk_` (argon2id), firma Ed25519, permisos, idempotencia, CORS y límites | ✅ Listo |
| Gestión de clientes desde la CLI (`prisma clientes`) | ✅ Listo |
| Panel de administración web en `/admin` ([panel.md](panel.md)) | ✅ Listo |
| SDK JavaScript, firmadores de nivel 3 (PHP, Node.js, Python) y guías para integradores ([integradores/](integradores/README.md)) | ✅ Listo |
| Servicio del sistema (`prisma servicio`), instaladores (Inno Setup, .deb, .pkg), `prisma actualizar` y CI | ✅ Listo |
| Verificación integral previa a la publicación ([reporte](pruebas/reporte-verificacion.md)) | ✅ Hecha; críticos corregidos, el resto pendiente de decisión |
| Primera versión publicada en GitHub Releases y SDK en npm | ⏳ Pendiente |

Las aplicaciones ya pueden imprimir por la red con la [API v1](api-v1.md), y el dueño configura todo desde el [panel](panel.md) sin usar la CLI. La documentación marca con **⏳** lo que todavía no existe.

## Glosario

| Término | Significado |
|---|---|
| **Puente** | El programa `prisma` corriendo en un equipo del negocio. |
| **Impresora** | Un dispositivo configurado en el puente, identificado por un **alias** (`caja`, `cocina`). Tiene un **perfil**: tipo de conexión, destino, ancho, columnas, página de códigos y capacidades. |
| **Destino** | Dónde está la impresora: IP, nombre en Windows, cola de CUPS, puerto serie o carpeta. |
| **Cliente** | Una aplicación autorizada a imprimir, con un nivel de seguridad (1: origen, 2: llave, 3: firma). |
| **Documento** | El JSON con las operaciones de un ticket (contrato «documento v1»). |
| **Operación** | Cada paso del ticket: `texto`, `fila`, `par`, `qr`, `cortar`, etc. No dependen unas de otras: cada una trae su propio estilo. |
| **Trabajo** | Un documento aceptado para una impresora, con su resultado. Se guarda en el historial sin el contenido del ticket. |
| **ESC/POS** | El lenguaje de comandos de las impresoras térmicas, que PRISMA genera a partir de las operaciones. |
| **Modo desarrollo** | `[dev] modo_desarrollo = true` en `config.toml`. Habilita `prisma imprimir`. Nunca lo actives en un negocio. |
