# PRISMA — Puente Remoto de Impresión Sobre Múltiples Arquitecturas

## Qué es
Un servicio independiente que se instala en una PC, Raspberry Pi o Mac dentro de la red local de un negocio.
Cualquier aplicación web (de cualquier desarrollador) puede enviarle por HTTPS un documento JSON con las
operaciones de un ticket; PRISMA lo valida, verifica la autorización del cliente, lo convierte a ESC/POS
según el perfil de la impresora y lo imprime por USB, red (TCP 9100) o Bluetooth (puerto serie).

## Principios de producto
- Independiente: NO depende de ninguna API, nube ni cuenta externa para funcionar. Todo se configura
  en su panel de administración web local.
- Fácil para integradores: una app sin backend debe poder imprimir con el SDK JS en minutos.
- Seguro por defecto: ninguna página web ni equipo de la red puede imprimir sin autorización explícita del dueño.
- Multiplataforma: Windows (amd64, 386), Linux (amd64, arm64), macOS (amd64, arm64).

## Conceptos del dominio
- Impresora: dispositivo físico configurado en el puente, identificado por un ALIAS (ej. "caja", "cocina").
  Tiene un perfil: tipo de conexión, destino, ancho, columnas, página de códigos y capacidades.
- Cliente: una aplicación autorizada a imprimir. Tiene nombre, nivel de seguridad, orígenes permitidos,
  credenciales según su nivel, impresoras permitidas y operaciones permitidas.
- Documento: el JSON (contrato "documento v1") con las operaciones del ticket.
- Trabajo: un documento aceptado y encolado para una impresora, con su resultado.

## Modelo de seguridad (configurable por cliente)
- Nivel 1 — Origen: el cliente se identifica por el encabezado Origin del navegador, que debe estar aprobado.
  Protege contra páginas web maliciosas. No protege contra herramientas fuera del navegador (curl).
- Nivel 2 — Llave de cliente (POR DEFECTO): además del origen, cada petición trae una llave (pk_...).
  Protege contra equipos de la red sin llave.
- Nivel 3 — Firma: el backend del cliente firma cada documento con Ed25519; el puente solo guarda la llave
  PÚBLICA del cliente. Protege contra documentos alterados, repetidos o falsificados.
- Operaciones sensibles (abrir_cajon, raw) deshabilitadas por defecto; se habilitan por cliente.
- Orígenes desconocidos que intentan imprimir quedan como "solicitudes pendientes" para que el dueño
  las apruebe o rechace en el panel.

## Decisiones técnicas (no las cambies sin preguntar)
- Go 1.26+, sin CGO, un binario por plataforma.
- Almacenamiento local en SQLite puro Go (modernc.org/sqlite). config.toml solo para arranque
  (puerto, rutas, TLS); todo lo demás se administra desde el panel y vive en la base de datos.
- El puente recibe operaciones JSON (no bytes) y genera el ESC/POS con un renderizador propio.
- Operaciones SIN ESTADO: cada operación trae su propio estilo.
- Los clientes solo envían el ALIAS de la impresora; nunca IPs, puertos ni nombres del sistema.
- API pública versionada bajo /v1/.

## Convenciones
- Código: nombres de dominio en español (impresora, cliente, documento, operacion, trabajo, conector, perfil);
  términos técnicos de Go en inglés.
- API pública y SDK: campos JSON en español y snake_case. Errores con código estable en MAYÚSCULAS
  (ej. IMPRESORA_NO_ENCONTRADA) + mensaje en español + sugerencia.
- Interfaz y mensajes al usuario en español (preparar i18n para inglés en el futuro).
- Logs estructurados (log/slog, JSON). NUNCA registrar el contenido de los tickets ni las llaves.
- `go test ./...` debe pasar en Windows, Linux y macOS. Cambios incrementales; pregunta antes de reestructurar.

## Estructura prevista
cmd/prisma/             CLI y punto de entrada
internal/contrato/      tipos del documento v1 y validación (JSON Schema)
internal/escpos/        renderizador operaciones → bytes, codificación de caracteres
internal/conector/      red, usb, bluetooth, archivo (build tags por sistema)
internal/cola/          cola secuencial por impresora
internal/almacen/       SQLite: impresoras, clientes, orígenes, solicitudes, trabajos, ajustes
internal/seguridad/     niveles, llaves, firma Ed25519 (JCS), anti-repetición, permisos
internal/servidor/      HTTPS, CORS, API pública /v1, panel /admin
internal/admin/         panel de administración (HTML embebido)
internal/servicio/      instalación como servicio (kardianos/service)
internal/dev/           herramientas: impresora virtual, decodificador, firmar
sdk/js/                 SDK JavaScript público (prisma.js)
schema/
ejemplos/
docs/
empaquetado/