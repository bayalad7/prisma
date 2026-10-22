# Glosario

Los términos que usan el código, la API, el panel y esta documentación. Cuando el panel y la API usan nombres distintos para lo mismo, se indican los dos.

| Término | Significado |
|---|---|
| **Puente** | El programa `prisma` corriendo en un equipo del negocio. Recibe documentos por HTTPS y los imprime. |
| **Panel** | La página web de administración del puente, en `https://<equipo>:8443/admin`. |
| **Dueño** | Quien administra el puente desde el panel. Es el único que autoriza aplicaciones. |
| **Impresora** | Un dispositivo configurado en el puente, identificado por un **alias**. |
| **Alias** | El nombre con el que las aplicaciones eligen una impresora (`caja`, `cocina`): minúsculas, dígitos, `-` y `_`, hasta 32 caracteres. Las aplicaciones nunca conocen IPs, puertos ni nombres del sistema. |
| **Perfil** | La configuración de una impresora: tipo de conexión, destino, ancho del papel, columnas, puntos por línea, página de códigos y capacidades. |
| **Destino** | Dónde está la impresora: IP y puerto, nombre en Windows, cola de CUPS, `/dev/usb/lpN`, puerto serie o carpeta. |
| **Capacidades** | Lo que la impresora puede hacer: cuchilla, cajón, QR y códigos de barras por comando, imágenes y *Confirmar impresión*. Lo que no tenga, el puente lo resuelve de otra forma o lo omite con una advertencia. |
| **Conector** | La parte del puente que entrega los bytes a una impresora: red, USB, Bluetooth (puerto serie) o archivo. |
| **Cola** | La fila de trabajos de una impresora: uno a la vez, en orden de llegada, hasta 20. |
| **Cliente** · **aplicación** | Un programa autorizado a imprimir, con un nivel de seguridad, orígenes, impresoras permitidas y operaciones permitidas. La API y la CLI dicen *cliente*; el panel dice *aplicación*. Su id tiene la forma `cli_…`. |
| **Nivel** | Cómo se identifica un cliente: 1 (origen), 2 (llave, por defecto) o 3 (firma). Ver [integradores/niveles-de-seguridad.md](../integradores/niveles-de-seguridad.md). |
| **Origen** | El esquema, host y puerto de una página web (`https://caja.example`), que el navegador envía en el encabezado `Origin`. |
| **Solicitud de origen** | Un origen desconocido que intentó imprimir y quedó registrado para que el dueño lo apruebe, rechace u olvide. |
| **Llave** | La credencial de nivel 2, `pk_…`. El puente guarda solo su hash argon2id; se muestra una sola vez al crearla. |
| **Llave pública** | La mitad pública del par Ed25519 de un cliente de nivel 3. El puente nunca conoce la privada. |
| **Firma** | La firma Ed25519 que el backend de un cliente de nivel 3 calcula sobre la forma canónica JCS del documento. |
| **Operaciones sensibles** | `abrir_cajon` y `raw`. Deshabilitadas para cada cliente hasta que el dueño las habilita. |
| **Documento** | El JSON con las operaciones de un ticket, según el contrato «documento v1». |
| **Contrato** | La especificación del documento: [contrato-documento-v1.md](contrato-documento-v1.md) y el esquema `schema/documento-v1.schema.json`. |
| **Operación** | Cada paso del ticket: `texto`, `separador`, `fila`, `par`, `salto`, `qr`, `codigo_barras`, `imagen`, `cortar`, `abrir_cajon` o `raw`. No dependen unas de otras: cada una trae su propio estilo. |
| **Recurso** | Una imagen PNG guardada en el puente desde el panel, que un documento usa por nombre (`"recurso": "logo"`). |
| **Trabajo** | Un documento aceptado para una impresora, con su resultado. Su id tiene la forma `trb_…`. Se guarda en el historial **sin** el contenido del ticket. |
| **Advertencia** | Algo que no impidió imprimir, pero que el ticket no salió exactamente como se pidió (por ejemplo, `CARACTER_SUSTITUIDO`). Ver [codigos-de-error.md](codigos-de-error.md#advertencias). |
| **Idempotencia** | La garantía de que reenviar el mismo documento (mismo `id`) no lo imprime dos veces. El puente recuerda cada `id` 10 minutos. |
| **Confirmar impresión** | Opción del perfil de las impresoras de red y Bluetooth: después de cada ticket, el puente pregunta a la impresora (`DLE EOT 1`) si lo recibió. |
| **ESC/POS** | El lenguaje de comandos de las impresoras térmicas, que el puente genera a partir de las operaciones. |
| **Página de códigos** | La tabla de caracteres de la impresora (CP437, CP850, CP858, WPC1252), que determina cómo salen los acentos. |
| **Previsualización** | La simulación del ticket a partir de los mismos bytes que recibiría la impresora (`/v1/previsualizar`, `prisma dev previsualizar`). |
| **Certificado autofirmado** | El certificado HTTPS que el puente genera para sí mismo. Cada equipo que imprime debe confiar en él. |
| **Modo desarrollo** | `[dev] modo_desarrollo = true` en `config.toml`. Habilita `prisma imprimir`. Nunca lo actives en un negocio. |
| **Llave de las versiones** | El par Ed25519 del proyecto que firma `checksums.txt` de cada versión publicada. `prisma actualizar` solo instala lo que verifica con su mitad pública. |
| **Versión** | Hay tres, independientes: la del **programa** (0.1.0), la de la **API** (`/v1/`) y la del **documento** (`"version": "1"`). |
