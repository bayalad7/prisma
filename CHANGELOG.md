# Registro de cambios

Todos los cambios notables de PRISMA. El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/) y las versiones, [SemVer](https://semver.org/lang/es/). Mientras la versión sea 0.x, una versión menor puede traer cambios incompatibles; se marcarán como tales.

Hay tres versiones independientes: la del **programa** (este registro), la de la **API** (`/v1/`) y la del **documento** (`"version": "1"`).

## [Sin publicar]

### Agregado

- Licencia Apache-2.0 (`LICENSE`, `NOTICE`). El SDK pasa de `UNLICENSED` a `Apache-2.0`.
- Documentación reorganizada por tipo de lector: `instalacion/`, `administracion/`, `operacion/`, `integradores/`, `seguridad/`, `arquitectura/`, `adr/`, `diagramas/`, `referencia/` y `desarrollo/`.
- Referencia formal de la API en OpenAPI 3.1 (`docs/referencia/openapi.yaml`) y una tabla con todos los códigos de error y advertencias.
- Modelo de amenazas, políticas de seguridad, runbook de operación y 14 registros de decisiones (ADR).
- 27 diagramas Mermaid numerados en `docs/diagramas/`, cada uno con las fuentes en el código en que se basa, y `scripts/validar-diagramas.sh` (y `.ps1`), que los compila con mermaid-cli. El CI falla si alguno no compila.
- Pruebas que mantienen la documentación fiel al código (`go test ./docs`): rutas de la API, códigos de error y enlaces.
- `CONTRIBUTING.md` y `SECURITY.md`.
- Sistema de diseño del panel: tokens de color con contraste AA verificado por pruebas, tema oscuro con selector manual, componentes accesibles (`sistema.js`), textos en un solo archivo (`textos.js`) y un sprite de iconos propios. Con `modo_desarrollo`, `/admin/estilos` muestra la galería de componentes y los prototipos de las pantallas.
- Análisis de experiencia del panel en `docs/ux/` (hallazgos, flujos, accesibilidad e inventario de diagramas).
- **Impresora virtual** (tipo `virtual`): vive dentro del puente, recibe los mismos bytes que una impresora física y muestra cada ticket en el panel como saldría en el papel (203 ppp), con descarga del PNG y del `.bin` y los comandos ESC/POS. Sus tickets se guardan solo en memoria (los últimos 20 por defecto), nunca en la base ni en la bitácora. Puede simular fallas (fuera de línea, lenta, falla aleatoria) con los códigos que la API ya informa. Se crea desde el panel o con `prisma impresoras agregar --tipo virtual`.
- `GET /v1/impresoras` incluye el campo `virtual` (aditivo). Los trabajos del historial indican si fueron a una impresora virtual.
- El ticket de diagnóstico del panel muestra la imagen del ticket en lugar de una vista de texto que se cortaba.
- Migración 0005 de la base de datos (se respalda sola antes de aplicarse).
- Dependencia nueva: `golang.org/x/image`, por la fuente Go Mono del raster de tickets.
- **Probar impresión** (`/admin/pruebas`): plantillas (los ejemplos del repositorio y el diagnóstico de la impresora), constructor visual (agregar, editar, reordenar y eliminar operaciones) o JSON, con validación en vivo contra el contrato v1, vista previa al ancho de la impresora, envío a cualquier impresora activa por el mismo camino que `/v1/imprimir` (a nombre de «Panel») con la respuesta que recibiría una aplicación, y «Copiar como código» con el SDK de JavaScript o curl.
- **¿Por qué mi app no puede imprimir?** (`/admin/pruebas/permisos`, y *Comprobar permisos* en cada aplicación): simula sin imprimir la autorización de `/v1/imprimir` (origen, llave, firma, impresora, operaciones sensibles e impresora activa) y dice en qué paso fallaría, con el código, el estado HTTP y cómo resolverlo. No pide la llave ni registra solicitudes; una prueba lo compara con la API en 336 combinaciones.
- Guía **Probar sin impresora** (`docs/administracion/probar-sin-impresora.md`), para dueños e integradores, con capturas generadas por un script, y diagrama 29 de la impresora virtual (el primero con metadatos para el panel).
- Una prueba nueva comprueba que nada de lo que se prueba desde el panel (vista previa, envío a la impresora virtual, documentos con errores, rollo) deja el contenido del ticket en la bitácora, la base o el respaldo.
- **Rollo en vivo** de las impresoras virtuales (`/admin/pruebas/rollo/{alias}`): los tickets de cualquier aplicación aparecen solos, con la aplicación, la referencia, el `trabajo_id` y las advertencias; se pueden ampliar, descargar (PNG y `.bin`) y ver sus comandos. Consulta cada 2 s solo mientras se ve, con la cabecera `X-Prisma-Sondeo`, que no renueva la sesión del panel.
- Tarjetas de impresoras con insignia de estado, datos en palabras (conexión, papel, acentos, lo que sabe hacer, último problema) y la acción principal según el estado; *Activar* y *Desactivar* desde la tarjeta, con confirmación. Imágenes con vista previa en 58 u 80 mm y errores junto al campo. Las pantallas de acceso permiten mostrar la contraseña, marcan los errores junto al campo, recuerdan cómo restablecerla y, sin conexión con el puente, ofrecen volver a intentar.
- Ajustes en cuatro pestañas (General, Seguridad y acceso, HTTPS, Respaldo) con validación junto al campo y confirmaciones que dicen qué pasará (desactivar el acceso solo desde la red local, regenerar el certificado, restaurar). El panel avisa 2 minutos antes de cerrar la sesión por inactividad y permite seguir conectado.
- Trabajos con explicación: al tocar un trabajo fallido, el panel dice qué pasó, por qué suele pasar y qué hacer, con *Probar la impresora* ahí mismo; los avisos también se explican y un trabajo de una impresora virtual muestra su ticket. Filtros rápidos (fallidos, con aviso, en cola); «Ver qué pasó» en el Inicio abre directamente los fallidos. Una prueba exige que cada código que puede quedar en un trabajo tenga su explicación.
- Aplicaciones y Solicitudes en lenguaje llano: los niveles se explican sin jerga («Dirección y llave secreta», recomendado) con una guía «¿Cuál elijo?»; la llave se muestra una sola vez con copiar, descargar como archivo y qué hacer con ella; el código para integrar usa el SDK real y deja elegir la impresora; las solicitudes dicen si vienen de este equipo, de la red o de Internet y si van sin cifrar, y advierten el riesgo al aprobarlas. Desactivar una aplicación pide confirmación, y eliminar una impresora dice qué aplicaciones la usan.
- Asistente nuevo para agregar impresoras en 3 pasos reales: tipo (80 mm, 58 mm o virtual), conexión con las impresoras detectadas (las que no son de tickets van marcadas) y un alias propuesto a partir del nombre, y un ticket de prueba con las preguntas «¿salió?» y «¿se leen bien los acentos?». La prueba de acentos pide elegir el renglón que se lee bien, sin hablar de páginas de códigos. Los errores de los formularios aparecen junto al campo.
- Inicio nuevo: el estado de un vistazo con una acción en cada tarjeta, y **Primeros pasos** (agregar una impresora física o virtual, imprimir una prueba, revisar los acentos, autorizar una aplicación) hasta que el puente esté listo. Los tickets de hoy ya no cuentan las pruebas del panel.
- Navegación nueva del panel: secciones agrupadas (Operación, Integración, Sistema) con iconos, barra inferior en celular, enlace para saltar al contenido, selector de tema y atajos de teclado (`g` y una letra; se pueden desactivar). Cada sección tiene su dirección (`/admin/impresoras`, `/admin/aplicaciones`…); los enlaces viejos con `#` siguen funcionando.

### Cambiado

- **Ayuda en cada pantalla:** el botón *Ayuda* abre su sección del manual en un panel lateral, sin salir de la pantalla ni perder lo escrito. Los avisos de error ofrecen *Cómo resolverlo*, el detalle de un trabajo fallido y *Probar impresión* enlazan la sección que lo resuelve, y los campos técnicos tienen *Saber más*. Una prueba exige ayuda para cada pantalla y cada código de error. Guías nuevas de Inicio, Trabajos, Probar impresión e Imágenes.
- Manual para el dueño: **Primeros pasos** (de recién instalado al primer ticket y la primera aplicación, con capturas generadas por un script), **Conceptos básicos**, **Tareas frecuentes**, **Solución de problemas** (cada síntoma lleva a su solución) y **Preguntas frecuentes**. La Ayuda busca en todo el manual sin Internet, sin distinguir acentos, y resalta las palabras en cada resultado.
- En el asistente de impresoras, responder «Sí, se leen bien» a los acentos completa ese paso de *Primeros pasos* (antes lo pedía otra vez).
- La *Ayuda* del panel muestra el manual de `docs/` dentro del puente, sin Internet y de la versión instalada: índice por grupos, una dirección por página y enlaces entre páginas sin recargar. Lo dibuja un renderizador propio que no usa `innerHTML` (ADR 0017); una prueba falla si un documento usa Markdown que no admite.

### Corregido

- En celular, Trabajos, Ayuda y el asistente de impresoras ya no se salen de la pantalla a lo ancho.
- El foco del teclado se ve en las tarjetas de opción (nivel de seguridad, página de códigos) y los bloques de código desplazables reciben el foco.
- Los bordes de los campos de formulario tienen contraste suficiente (3:1).
- Los mensajes que citan la documentación usan las rutas nuevas. Las rutas viejas siguen existiendo como páginas que redirigen.

## [0.1.0] — 2026-10-22

Primera versión publicada.

### Agregado

- **Puente** para Windows (amd64, 386), Linux (amd64, arm64, incluida Raspberry Pi) y macOS (amd64, arm64): un binario en Go sin CGO, con SQLite en Go puro.
- **API pública `/v1`** por HTTPS: `salud`, `impresoras`, `imprimir` y `previsualizar`.
- **Contrato «documento v1»** con JSON Schema 2020-12 embebido y ejemplos válidos e inválidos: texto, separador, fila, par, salto, QR, código de barras, imagen, cortar, abrir cajón y `raw`.
- **Renderizador ESC/POS** por perfil de impresora: páginas de códigos CP437, CP850, CP858 y WPC1252; QR y códigos de barras por comando o como imagen; imágenes en 1 bit.
- **Conectores** de red (TCP 9100), USB (winspool, CUPS, `/dev/usb`), Bluetooth (puerto serie) y archivo, con detección de impresoras. Opción *Confirmar impresión* (`DLE EOT`) en red y Bluetooth.
- **Cola** FIFO por impresora, con estado y pruebas periódicas.
- **Seguridad** por niveles: origen, llave `pk_` (argon2id) y firma Ed25519 sobre JCS; idempotencia persistente; CORS con eco exacto; límites por IP y por aplicación; solicitudes de origen que el dueño aprueba; `abrir_cajon` y `raw` deshabilitadas por defecto.
- **Panel de administración** en `/admin`: impresoras, aplicaciones, solicitudes, historial, imágenes, ajustes, certificado, respaldo y restauración.
- **CLI** `prisma`: `ejecutar`, `servicio`, `actualizar`, `admin`, `impresoras`, `clientes`, `tls`, `config`, `imprimir` y `dev` (previsualizar, renderizar, decodificar, impresora virtual, llaves y firmar).
- **SDK JavaScript** para navegador y Node.js 18+, con validación local, reintentos idempotentes y diagnóstico de conexión; firmadores de nivel 3 en PHP, Node.js y Python.
- **Instalación como servicio** en los tres sistemas; instaladores para Windows (Inno Setup), Debian y Raspberry Pi (`.deb` e instalador de una línea) y macOS (`.pkg`).
- **Actualizaciones** con `prisma actualizar`, verificadas con la firma Ed25519 de `checksums.txt` y el SHA-256 de cada archivo; aviso de versión nueva en el panel.
- Certificado autofirmado de 825 días, con aviso desde 60 días antes de vencer y regeneración automática si venció.

### Problemas conocidos

- El SDK no está publicado en npm. Se carga desde `cdn.jsdelivr.net/gh/bayalad7/prisma@v0.1.0/sdk/js/dist/…` o copiando `sdk/js/dist/` al proyecto.
- Los instaladores no tienen firma de código: Windows muestra SmartScreen y macOS pide *Abrir de todas formas*. Verifica los archivos con `checksums.txt`.
- Los trabajos que llegan por la API pasan directo de `encolado` a `completado` o `fallido`; el estado `imprimiendo` solo lo usa `prisma imprimir`.
- La *Ayuda* del panel enlaza la documentación de la rama `develop`.

[Sin publicar]: https://github.com/bayalad7/prisma/compare/v0.1.0...develop
[0.1.0]: https://github.com/bayalad7/prisma/releases/tag/v0.1.0
