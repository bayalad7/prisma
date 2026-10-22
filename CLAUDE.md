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
- Servicio del sistema con kardianos/service: Windows (LocalSystem, inicio automático, reinicio ante
  fallos), systemd (usuario `prisma`, grupos lp y dialout), launchd (usuario oculto `_prisma`). La unidad
  y el plist salen de plantillas propias en internal/servicio; la unidad del .deb debe ser idéntica
  (lo verifica una prueba).
- Actualizaciones: `prisma actualizar` solo instala lo firmado con la llave Ed25519 del proyecto
  (pública en empaquetado/llave-releases.pub, embebida con -ldflags; privada solo en el secreto
  PRISMA_LLAVE_RELEASES del CI) y verifica el SHA-256. Nunca se instala nada solo.
- Panel: sesiones en memoria, cookie HttpOnly/Secure/SameSite=Strict, CSRF por cabecera, solo IPs
  privadas por defecto, sin frameworks ni CDNs.
- CI y publicación compilan con el último parche de Go 1.26 (`go-version: '1.26.x'`, `check-latest`);
  go.mod declara solo el mínimo (1.26.0). nfpm v2.48.0 exige 1.26.4 para el .deb.
- Licencia Apache-2.0 (LICENSE, NOTICE); el SDK también.
- Cada decisión tiene su registro en docs/adr/; una decisión nueva lleva un ADR nuevo.

## Invariantes (verificadas; no las rompas)
- Idempotencia: un trabajo que NO se imprimió (fallido o interrumpido por un reinicio) libera su id;
  nunca se responde `ok: true` a un reintento de algo que no salió.
- Al revés también: si los datos ya llegaron a la impresora (p. ej. responde «fuera de línea» a
  Confirmar impresión), el trabajo NO falla; se completa con advertencia (`conector.Aviso`),
  porque un reintento lo imprimiría dos veces.
- CORS: eco exacto del origen aprobado, nunca `*`; los orígenes desconocidos no reciben cabeceras.
- Nada costoso (esquema completo, imágenes, render) antes de identificar al cliente.
- Los instaladores nunca activan `modo_desarrollo` (TestConfiguracionDeLosInstaladores).
- Ninguna goroutine de la cola sobrevive a Quitar/Cerrar (TestSinFugasDeGoroutines).
- Los tickets de las impresoras virtuales viven solo en memoria (conector.RollosVirtuales): nunca en la
  base, la bitácora ni los respaldos (TestImpresoraVirtualDePrincipioAFin).
- Permisos: base y llave del certificado 0600; datos solo para el usuario del servicio
  (en Windows, SYSTEM y Administradores).

## Convenciones
- Código: nombres de dominio en español (impresora, cliente, documento, operacion, trabajo, conector, perfil);
  términos técnicos de Go en inglés.
- API pública y SDK: campos JSON en español y snake_case. Errores con código estable en MAYÚSCULAS
  (ej. IMPRESORA_NO_ENCONTRADA) + mensaje en español + sugerencia.
- Interfaz y mensajes al usuario en español (preparar i18n para inglés en el futuro).
- Logs estructurados (log/slog, JSON). NUNCA registrar el contenido de los tickets ni las llaves.
- Documentación: en español, fiel al código (si no coinciden, manda el código y se corrige la doc).
  Versiones distintas: programa (0.1.0), API (/v1) y documento ("1").
- `go test ./...` debe pasar en Windows, Linux y macOS. Cambios incrementales; pregunta antes de reestructurar.
- Ramas: `main` (producción) y `develop` (trabajo). Commits en español: `tipo(ámbito): asunto`, con cuerpo,
  líneas ≤100 caracteres, un commit por ámbito, y cada commit debe compilar y pasar las pruebas.

## Comandos
- Pruebas: `go test ./...` (`-race` corre en el CI; localmente, en un contenedor Linux con CGO).
- Lint: `make lint` / `.\build.ps1 lint` (go vet + staticcheck).
- SDK y firmadores: `make test-sdk` / `.\build.ps1 test-sdk`. Tras cambiar sdk/js/src: `npm run construir`.
- Archivos generados por pruebas: `go test ./internal/dev -actualizar` (golden) y
  `go test ./internal/servicio -run Deb -actualizar` (unidad de systemd del .deb).
- Binarios y paquetes: `make build-todo | deb | pkg | checksums` / `.\build.ps1 build-todo | deb | instalador | checksums`.
- Documentación fiel al código: `go test ./docs` compara las rutas de la API con
  docs/referencia/openapi.yaml, exige cada código de error en docs/referencia/codigos-de-error.md y
  revisa los enlaces y anclas de todos los Markdown. Si cambias rutas, códigos o archivos, actualízalos.
- Diagramas: `scripts/validar-diagramas.sh` / `.\scripts\validar-diagramas.ps1` compila cada bloque Mermaid
  (lo corre el CI). Cada docs/diagramas/NN-*.md cita sus fuentes en el código: si cambias ese código,
  revisa el diagrama y su «Última verificación». Colores comunes: classDef externo/prisma/datos/seguridad.
- Antes de publicar: docs/desarrollo/pruebas/matriz.md (manual) y docs/desarrollo/pruebas/ (reportes);
  pasos en docs/desarrollo/publicar.md. Cada versión mueve «Sin publicar» de CHANGELOG.md.

## Estructura
cmd/prisma/             CLI y punto de entrada (servicio, actualizar, admin, impresoras, clientes, tls, dev)
internal/config/        carga y validación de config.toml
internal/rutas/         rutas por sistema (configuración, datos, bitácoras)
internal/contrato/      tipos del documento v1 y validación (JSON Schema + reglas)
internal/escpos/        renderizador operaciones → bytes, codificación de caracteres
internal/conector/      red, usb, bluetooth, archivo, virtual (build tags por sistema) y detección
internal/cola/          cola secuencial por impresora
internal/almacen/       SQLite: impresoras, clientes, orígenes, solicitudes, trabajos, ajustes, respaldos
internal/seguridad/     niveles, llaves, firma Ed25519 (JCS), anti-repetición, permisos, límites
internal/servidor/      HTTPS, CORS, API pública /v1 (monta el panel /admin)
internal/admin/         panel de administración (HTML/CSS/JS embebidos) y API interna /admin/api
internal/servicio/      instalación como servicio (kardianos/service) y bitácora rotativa
internal/actualizacion/ versiones nuevas: firma, checksums, reemplazo del binario y aviso del panel
internal/dev/           herramientas: impresora virtual, decodificador, simulador, firmar
internal/version/       versión, commit y fecha (-ldflags)
sdk/js/                 SDK JavaScript público (@bayalad7/prisma; dist/ se versiona para el CDN)
sdk/firmadores/         firmadores de nivel 3 (PHP, Node.js, Python) y casos JCS compartidos con el puente
schema/                 JSON Schema del documento (embebido)
ejemplos/               documentos válidos e inválidos (con indice.json); los usan puente, SDK y
                        las plantillas de «Probar impresión» (embebidos)
testdata/               vector de firma público
docs/                   por lector: integradores/ instalacion/ administracion/ operacion/ seguridad/
                        arquitectura/ adr/ diagramas/ referencia/ (API, OpenAPI, contrato, errores, CLI)
                        desarrollo/ (con pruebas/). Las rutas viejas citadas por 0.1.0 son páginas
                        cortas que redirigen: no las borres
empaquetado/            Inno Setup, .deb (nfpm), .pkg, instalador de Raspberry Pi, llave pública de versiones
.github/workflows/      CI (3 sistemas, -race, SDK, servicio real) y publicación de versiones