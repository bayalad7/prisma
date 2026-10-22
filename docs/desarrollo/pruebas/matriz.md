# Matriz de pruebas manuales

Pruebas para ejecutar a mano antes de publicar una versión, con hardware y dispositivos reales. Complementan las pruebas automáticas (`go test -race ./...`, SDK, firmadores, y la instalación del servicio que hace el CI en Windows, Ubuntu y macOS).

**Cómo usarla:**

- Marca cada fila con ✅ (como se espera), ❌ (anota qué pasó) o ➖ (no aplica o no hay hardware).
- Cada fila dice el **resultado esperado**. En los casos que **no deben funcionar**, el mensaje exacto que debe verse. El SDK muestra `mensaje` y luego `sugerencia`; en el panel aparecen como aviso.
- Anota en *Notas* la versión de PRISMA (`prisma version`), el modelo de impresora y la versión del sistema o del navegador.

Cubrir las 360 combinaciones de {3 sistemas} × {4 conexiones} × {2 anchos} × {5 navegadores} × {3 niveles} no es práctico. Por eso la matriz se organiza así:

| Sección | Qué cubre | Filas |
|---|---|---|
| [1. Instalación](#1-instalación-y-servicio) | Cada sistema: instalar, reiniciar, actualizar y desinstalar | 3 + negativos |
| [2. Impresoras](#2-impresoras-sistema--conexión--ancho) | Sistema × conexión × ancho, completo: es lo que cambia la impresión | 24 + negativos |
| [3. Aplicaciones](#3-aplicaciones-navegador--nivel) | Navegador × nivel, completo: es lo que cambia la autorización | 15 + negativos |
| [4. Combinaciones](#4-combinaciones-cruzadas-por-pares) | Los cinco factores juntos, por pares: cada pareja de valores aparece al menos una vez | 20 |
| [5. Resiliencia](#5-resiliencia-con-hardware-real) | Fallas con hardware real | 8 |

---

## 0. Preparación

**Equipos:**

- un equipo por sistema con PRISMA instalado según [instalacion/](../../instalacion/README.md): PC con Windows, Raspberry Pi con Raspberry Pi OS de 64 bits, y una Mac;
- teléfonos Android e iPhone en la misma red Wi-Fi.

**Impresoras:**

| Conexión | Qué hace falta |
|---|---|
| Red | Térmica Ethernet o Wi-Fi con IP fija, en el puerto 9100. |
| USB | Térmica USB: instalada en Windows; en Linux como `/dev/usb/lp0` o cola de CUPS raw; en macOS como cola de CUPS ([instalacion/impresoras.md](../../instalacion/impresoras.md)). |
| Bluetooth | Térmica Bluetooth emparejada: COM saliente en Windows, `/dev/rfcomm0` en Linux, `/dev/cu.*` en macOS. |
| Virtual | `prisma dev impresora-virtual --puerto 9100 --ancho 58` (o `80`) en el equipo del puente, agregada en el panel como **Red** con destino `127.0.0.1:9100`. |

Si es posible, prueba una impresora de 58 mm y otra de 80 mm por conexión. Si no, cambia el ancho en el perfil y comprueba la regla del ticket de diagnóstico.

**Aplicaciones en el panel** de cada puente:

| Aplicación | Nivel | Configuración |
|---|---|---|
| Prueba N1 | 1 | Origen de la página de prueba, por ejemplo `http://192.168.1.20:3000`. |
| Prueba N2 | 2 | Mismo origen; anota la llave `pk_…`. |
| Prueba N3 | 3 | Llave pública del ejemplo PHP; origen `http://<pc>:8000` ([ejemplo](../../integradores/ejemplos/backend-php-firmado/README.md)). |

**Páginas de prueba**, servidas desde una PC de la red (no desde el equipo del puente), con `npx serve -l 3000`:

- **N1:** [pagina-simple.html](../../integradores/ejemplos/pagina-simple.html) sin `llave`.
- **N2:** la misma página con la llave de Prueba N2.
- **N3:** el ejemplo PHP (`php -S 0.0.0.0:8000`).

Cada dispositivo debe **confiar en el certificado** del puente ([instalacion/https-y-certificados.md](../../instalacion/https-y-certificados.md#confiar-en-el-certificado)). Se comprueba abriendo `https://<puente>:8443/v1/salud` sin advertencia.

---

## 1. Instalación y servicio

| ID | Sistema | Pasos | Esperado | Resultado | Notas |
|---|---|---|---|---|---|
| I-W | Windows 10/11 | Ejecutar `PRISMA-…-windows-amd64-setup.exe`. Al terminar, se abre el panel. Crear la contraseña. Reiniciar Windows **sin iniciar sesión** y abrir el panel desde otra PC. | Servicio «PRISMA - Puente de Impresión Térmica» en *Automático* y *En ejecución*. Regla «PRISMA» en el firewall. `C:\ProgramData\PRISMA\datos` solo para Administradores y SYSTEM. El panel responde después del reinicio. `prisma servicio estado` → «en marcha». | | |
| I-L | Raspberry Pi OS 64 bits | `curl -fsSL …/instalar.sh \| sh`. Abrir el panel desde una PC con el código de `sudo -u prisma prisma admin codigo-inicial`. `sudo reboot`. | Usuario `prisma` en los grupos `lp` y `dialout`. `systemctl is-enabled prisma` → enabled. Tras reiniciar, `systemctl is-active prisma` → active y el panel responde. | | |
| I-M | macOS 12+ | Abrir `PRISMA-…-macos.pkg`. Abrir `https://localhost:8443/admin` y crear la contraseña. Reiniciar la Mac. | Usuario oculto `_prisma`. `sudo launchctl print system/com.github.bayalad7.prisma` → *state = running*. El panel responde tras reiniciar. | | |
| I-A | Los tres | Con una versión anterior instalada: `prisma actualizar` (como administrador). | Verifica la firma y el SHA-256, reemplaza el programa y reinicia el servicio. `prisma version` muestra la versión nueva. Si cambió el esquema, hay una copia en `datos/respaldos`. | | |
| I-D | Los tres | Desinstalar (*Aplicaciones* en Windows, `apt remove prisma`, `prisma servicio desinstalar`). | El servicio desaparece; los datos se conservan. `apt purge` borra todo. | | |

**Lo que no debe funcionar:**

| ID | Caso | Esperado |
|---|---|---|
| I-N1 | `prisma servicio instalar` sin permisos de administrador | Windows: «instalar el servicio requiere permisos de administrador. Abre PowerShell como administrador…». Linux/macOS: «…ejecuta el comando con sudo». |
| I-N2 | Linux: binario en `/home/…` y `sudo prisma servicio instalar` | «el binario está en /home/…, fuera del alcance del servicio; muévelo antes de instalar», con los comandos para moverlo. |
| I-N3 | Instalador de Linux en Raspberry Pi OS de **32 bits** | «este sistema es de 32 bits (armhf). PRISMA necesita Raspberry Pi OS de 64 bits…». No instala nada. |
| I-N4 | Abrir el panel desde **otra** PC sin contraseña creada | Pide el código de configuración; con un código incorrecto: «El código de configuración no es correcto». |
| I-N5 | Panel desde una IP pública (fuera de la red local) | 403 «El panel solo se puede abrir desde la red local de este negocio». |

---

## 2. Impresoras: sistema × conexión × ancho

Para cada fila, en el panel:

1. **Impresoras → Agregar** con el preset del ancho.
2. **Probar conexión**: «La impresora respondió: está en línea».
3. **Ticket de diagnóstico**: sale completo. La **regla de columnas cabe en una sola línea** (32 columnas en 58 mm, 48 en 80 mm). Los acentos `áéíóú ÁÉÍÓÚ ñÑ üÜ ¿¡ € °` se leen bien, y se ven las tres alineaciones, el QR y el corte (o 4 líneas de avance si no hay cuchilla).
4. **Probar acentos**: elegir el renglón que se lee bien deja el perfil actualizado; repetir el diagnóstico para comprobar los acentos.
5. Imprimir [`venta-logo-qr.json`](../../../ejemplos/validos/venta-logo-qr.json) desde una aplicación: logo, QR y 2 copias.

| ID | Sistema | Conexión | Ancho | Resultado | Notas |
|---|---|---|---|---|---|
| C01 | Windows | Red | 58 mm | | |
| C02 | Windows | Red | 80 mm | | |
| C03 | Windows | USB (nombre de la impresora en Windows) | 58 mm | | |
| C04 | Windows | USB | 80 mm | | |
| C05 | Windows | Bluetooth (COM saliente) | 58 mm | | |
| C06 | Windows | Bluetooth | 80 mm | | |
| C07 | Windows | Virtual | 58 mm | | |
| C08 | Windows | Virtual | 80 mm | | |
| C09 | Linux/RPi | Red | 58 mm | | |
| C10 | Linux/RPi | Red | 80 mm | | |
| C11 | Linux/RPi | USB (`/dev/usb/lp0` o cola CUPS) | 58 mm | | |
| C12 | Linux/RPi | USB | 80 mm | | |
| C13 | Linux/RPi | Bluetooth (`/dev/rfcomm0`) | 58 mm | | |
| C14 | Linux/RPi | Bluetooth | 80 mm | | |
| C15 | Linux/RPi | Virtual | 58 mm | | |
| C16 | Linux/RPi | Virtual | 80 mm | | |
| C17 | macOS | Red | 58 mm | | |
| C18 | macOS | Red | 80 mm | | |
| C19 | macOS | USB (cola de CUPS) | 58 mm | | |
| C20 | macOS | USB | 80 mm | | |
| C21 | macOS | Bluetooth (`/dev/cu.*`) | 58 mm | | |
| C22 | macOS | Bluetooth | 80 mm | | |
| C23 | macOS | Virtual | 58 mm | | |
| C24 | macOS | Virtual | 80 mm | | |

**Lo que no debe funcionar** (en cualquier sistema):

| ID | Caso | Esperado |
|---|---|---|
| C-N1 | Red: impresora apagada → *Probar conexión* | `IMPRESORA_INALCANZABLE`: «La impresora en <ip>:9100 rechazó la conexión» o «Falló la comunicación con la impresora…». El estado pasa a *fuera de línea* en ≤30 s. |
| C-N2 | Red: IP equivocada de otra subred | «Falló la comunicación…» o `TIEMPO_AGOTADO` en ≤3 s al conectar. |
| C-N3 | USB: impresora desconectada → ticket de diagnóstico | Windows: la cola acepta el trabajo y el ticket sale al reconectar (documentado en administracion/problemas-comunes.md). Linux con `/dev/usb/lp0`: `IMPRESORA_NO_ENCONTRADA`, «No existe…». |
| C-N4 | Linux USB: usuario `prisma` fuera del grupo `lp` | `PERMISO_DENEGADO`: «No hay permiso para abrir /dev/usb/lp0». |
| C-N5 | Bluetooth: un teléfono conectado a la impresora | `PUERTO_OCUPADO` o `IMPRESORA_INALCANZABLE`; al desconectar el teléfono, vuelve a imprimir. |
| C-N6 | Bluetooth: puerto mal escrito (`COM99`, `/dev/rfcomm9`) | `IMPRESORA_NO_ENCONTRADA`: «No existe el puerto …». |
| C-N7 | Impresora sin cuchilla con `cortar` | Imprime, avanza 4 líneas y el trabajo lleva la advertencia `SIN_CUCHILLA`. |
| C-N8 | Imagen en una impresora marcada «sin imágenes» | Rechazo **antes** de imprimir: `OPERACION_NO_SOPORTADA`. No sale nada. |
| C-N9 | `fila` con porcentajes que no caben en 58 mm | `FILA_NO_CABE`; no sale nada. |

---

## 3. Aplicaciones: navegador × nivel

Para cada fila, abre la página del nivel en el dispositivo y pulsa **Imprimir**. Esperado: «Impreso (trb_…)», el ticket sale una vez y el trabajo aparece en *Trabajos* con el nombre de la aplicación.

| ID | Navegador | Nivel 1 (origen) | Nivel 2 (origen + llave) | Nivel 3 (firma) |
|---|---|---|---|---|
| N-Ch | PC Chrome | | | |
| N-Ed | PC Edge | | | |
| N-Ff | PC Firefox (con el certificado importado en Firefox o `security.enterprise_roots.enabled`) | | | |
| N-An | Android Chrome (página en `http://<pc>:3000`) | | | |
| N-iP | iPhone Safari (perfil instalado **y** confianza activada) | | | |

**Lo que no debe funcionar.** Pruébalo al menos en PC Chrome y en iPhone Safari:

| ID | Caso | Esperado (mensaje del SDK o del panel) |
|---|---|---|
| N-N1 | Página en un origen **no aprobado** | «El puente responde, pero no ha autorizado el origen http://…:3000.» + «El dueño del puente debe aprobar este origen en el panel (Solicitudes): ya quedó registrado como pendiente…». La solicitud aparece en *Solicitudes* con sus intentos. |
| N-N2 | Origen rechazado en *Solicitudes* | Mismo mensaje que N-N1; la solicitud queda «rechazada» y no vuelve a aparecer como pendiente. |
| N-N3 | Dispositivo que **no** confía en el certificado | «No se pudo conectar con el puente en https://…. Si es la primera vez desde este equipo, lo más probable es que el navegador no confíe en su certificado.» + «Abre https://…/v1/salud en una pestaña…». |
| N-N4 | Puente apagado (después de haber impreso) | «El puente https://… dejó de responder.» Tarda unos 4 s: el SDK reintenta 2 veces. |
| N-N5 | Nivel 2 sin llave | «Esta aplicación debe enviar su llave de cliente.» (`LLAVE_REQUERIDA`) |
| N-N6 | Nivel 2 con una llave **regenerada** en el panel (la anterior) | «La llave de cliente no es válida.» (`LLAVE_INVALIDA`) |
| N-N7 | Nivel 2 con la llave de **otra** aplicación | «La llave no corresponde a la aplicación de este origen.» |
| N-N8 | Nivel 1: `curl` a `/v1/imprimir` sin `Origin` ni llave (fuera del navegador) | 401 «La petición no identifica a ninguna aplicación autorizada.» (`CREDENCIALES_REQUERIDAS`). Nadie de la red imprime sin pasar por una página aprobada. |
| N-N8b | Nivel 1: `curl` con `-H "Origin: <origen aprobado>"` | **Imprime.** Es la limitación documentada del nivel 1: fuera del navegador, `Origin` se puede falsificar. Por eso el nivel 2 es el de fábrica. |
| N-N9 | Nivel 3: documento firmado, alterado en las herramientas del navegador antes de enviarlo | «La firma no corresponde al documento: fue alterado o se firmó con otra llave.» (`FIRMA_INVALIDA`). No imprime. |
| N-N10 | Nivel 3: esperar más de 90 s entre firmar y enviar | «El documento ya expiró.» (`FIRMA_EXPIRADA`) |
| N-N11 | Nivel 3: reenviar el mismo documento firmado | Responde `duplicado: true`. **No** imprime otra vez. |
| N-N12 | `abrir_cajon` sin habilitarlo para la aplicación | «La operación "abrir_cajon" está deshabilitada para esta aplicación.» No imprime nada del documento. |
| N-N13 | Impresora no permitida para la aplicación | «La aplicación no tiene permiso para imprimir en "cocina".» |
| N-N14 | Aplicación desactivada en el panel | `CLIENTE_DESACTIVADO`. Desde el navegador, el SDK muestra el mensaje de N-N1, que termina con «…puede estar desactivada». |
| N-N15 | Página `https://` (por ejemplo en Internet) y puente en `http://` | `CONTENIDO_MIXTO`: «Esta página usa https:// y el puente … usa http://; el navegador bloquea esa mezcla.» |
| N-N16 | Página pública de Internet en Chrome/Edge recientes, eligiendo **Bloquear** en el aviso de red local | «El navegador no permite que esta página se conecte a equipos de la red local.» (`PERMISO_RED_LOCAL_DENEGADO`) |
| N-N17 | Documento inválido (por ejemplo `alineacion: "center"`) | Error **antes** de enviar: `DOCUMENTO_INVALIDO`, con `detalles[0].ruta = /operaciones/0/alineacion`. No llega al puente. |
| N-N18 | 130 peticiones en un minuto desde la misma IP | Las que pasan de 120 reciben 429 `DEMASIADAS_PETICIONES` con `Retry-After`. |

---

## 4. Combinaciones cruzadas, por pares

Cada pareja de valores de dos factores cualesquiera (por ejemplo, *Bluetooth + iPhone Safari* o *macOS + nivel 3*) aparece al menos en una fila. Para cada una: agrega la impresora, imprime el diagnóstico y luego imprime desde la aplicación del nivel con el navegador indicado.

| ID | Sistema del puente | Conexión | Ancho | Navegador | Nivel | Resultado | Notas |
|---|---|---|---|---|---|---|---|
| P01 | Windows | Red | 58 mm | PC Chrome | 1 | | |
| P02 | Linux/RPi | USB | 80 mm | PC Chrome | 3 | | |
| P03 | macOS | Bluetooth | 58 mm | PC Chrome | 2 | | |
| P04 | Windows | Virtual | 80 mm | PC Chrome | 1 | | |
| P05 | Linux/RPi | Red | 80 mm | PC Edge | 2 | | |
| P06 | macOS | USB | 58 mm | PC Edge | 1 | | |
| P07 | Windows | Bluetooth | 80 mm | PC Edge | 3 | | |
| P08 | Linux/RPi | Virtual | 58 mm | PC Edge | 2 | | |
| P09 | macOS | Red | 58 mm | PC Firefox | 3 | | |
| P10 | Windows | USB | 80 mm | PC Firefox | 2 | | |
| P11 | Linux/RPi | Bluetooth | 58 mm | PC Firefox | 1 | | |
| P12 | macOS | Virtual | 80 mm | PC Firefox | 3 | | |
| P13 | Windows | Red | 80 mm | Android Chrome | 1 | | |
| P14 | Linux/RPi | USB | 58 mm | Android Chrome | 3 | | |
| P15 | macOS | Bluetooth | 80 mm | Android Chrome | 2 | | |
| P16 | Windows | Virtual | 58 mm | Android Chrome | 1 | | |
| P17 | Linux/RPi | Red | 58 mm | iPhone Safari | 2 | | |
| P18 | macOS | USB | 80 mm | iPhone Safari | 1 | | |
| P19 | Windows | Bluetooth | 58 mm | iPhone Safari | 3 | | |
| P20 | Linux/RPi | Virtual | 80 mm | iPhone Safari | 2 | | |

---

## 5. Resiliencia con hardware real

| ID | Caso | Pasos | Esperado | Resultado |
|---|---|---|---|---|
| R1 | Impresora de red apagada a mitad de un ticket | Imprimir un ticket largo (logo, muchas filas, 3 copias) y apagar la impresora mientras imprime. | Con un ticket que no cabe en los búferes: `IMPRESORA_INALCANZABLE` o `TIEMPO_AGOTADO`, y el reintento al encenderla imprime **una** vez. Con un ticket corto: sin *Confirmar impresión* el puente puede responder «impreso» aunque la impresora no lo haya recibido (M3 del [reporte](reporte-verificacion.md)); **con** *Confirmar impresión*, `TIEMPO_AGOTADO` («no confirmó que recibió el ticket») y el reintento imprime una vez. | |
| R1b | Confirmar impresión con tapa abierta | Activar *Confirmar impresión*, abrir la tapa e imprimir; luego cerrarla. | `200` con la advertencia `IMPRESORA_FUERA_DE_LINEA`; el ticket sale **una** vez al cerrar la tapa. Si la impresora no admite `DLE EOT`, todo ticket falla con `TIEMPO_AGOTADO`: anota el modelo. | |
| R2 | Bluetooth fuera de alcance | Alejar la impresora más de 10 m (o apagarla) e imprimir. | `IMPRESORA_INALCANZABLE` o `TIEMPO_AGOTADO` en ≤15 s. Al volver al alcance, el siguiente ticket sale. | |
| R3 | 10 trabajos a la vez | Desde 2 dispositivos, pulsar Imprimir 5 veces rápido en cada uno. | Salen los 10, de uno en uno, sin mezclarse ni repetirse. | |
| R4 | Cola llena | Con la impresora sin papel, enviar 25 tickets. | Los primeros 20 esperan (se ven como *pendientes* en el panel); los demás reciben `COLA_LLENA`. Al poner papel, salen los 20. | |
| R5 | Reinicio con trabajos en cola | Con la impresora sin papel y 3 tickets en espera, reiniciar el equipo del puente. Al volver, reenviar los mismos 3 documentos. | En *Trabajos* aparecen como fallidos con `PUENTE_REINICIADO`. Al reenviarlos se imprimen (no responden `duplicado`). | |
| R6 | Corte de red | Desconectar el cable del puente durante 1 min e imprimir; reconectar. | El SDK informa `PUENTE_INALCANZABLE` tras sus reintentos. Al reconectar, imprime sin reiniciar nada. | |
| R7 | Certificado | En el panel, *Ajustes → HTTPS → Regenerar*. | Los dispositivos que confiaban en el anterior muestran el mensaje de N-N3 hasta instalar el nuevo. | |
| R8 | Disco casi lleno (Raspberry Pi) | Llenar la tarjeta hasta dejar menos de 50 MB libres e imprimir. | Imprime, o falla con `ERROR_INTERNO` sin corromper la base. Al liberar espacio, todo vuelve a funcionar. | |
