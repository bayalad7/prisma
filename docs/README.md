# Documentación de PRISMA

PRISMA es un puente de impresión. Se instala en un equipo de la red local de un negocio, recibe documentos JSON con las operaciones de un ticket y los imprime en impresoras térmicas por red, USB o Bluetooth.

## Por dónde empezar

| Si quieres… | Lee |
|---|---|
| Ver PRISMA funcionando en 5 minutos, sin impresora | [primeros-pasos.md](primeros-pasos.md) |
| Instalarlo en el equipo de un negocio | [instalacion.md](instalacion.md) |
| Conectar una impresora (red, USB, Bluetooth) | [conectores.md](conectores.md) |
| Conocer todos los comandos y opciones | [cli.md](cli.md) |
| Configurar `config.toml` | [configuracion.md](configuracion.md) |
| Saber cómo está construido por dentro | [arquitectura.md](arquitectura.md) |
| Contribuir al código | [desarrollo.md](desarrollo.md) |
| Generar documentos JSON desde tu aplicación | [contrato-documento-v1.md](contrato-documento-v1.md) |
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
| Firma Ed25519 (verificación) | ✅ Listo |
| Servidor HTTPS y API pública `/v1` | ⏳ Pendiente |
| Panel de administración web | ⏳ Pendiente |
| Llaves de cliente, orígenes y permisos en la API | ⏳ Pendiente (la base de datos ya tiene las tablas) |
| SDK JavaScript | ⏳ Pendiente |
| Instalación como servicio (`prisma servicio instalar`) e instaladores por plataforma | ⏳ Pendiente |

Mientras falte el servidor HTTPS, las aplicaciones no pueden imprimir por la red. Hoy se imprime desde la CLI, con `prisma imprimir` en modo desarrollo. La documentación marca con **⏳** lo que todavía no existe.

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
