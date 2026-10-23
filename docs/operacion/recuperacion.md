# Recuperación

Qué hacer ante cada falla grave. Los problemas cotidianos están en [administracion/problemas-comunes.md](../administracion/problemas-comunes.md) y en [instalacion/problemas-comunes.md](../instalacion/problemas-comunes.md).

## Olvidé la contraseña del panel

En el equipo del puente, con el mismo usuario que el servicio:

| Sistema | Comando |
|---|---|
| Windows (PowerShell de administrador) | `prisma admin restablecer-contrasena` |
| Linux | `sudo -u prisma prisma admin restablecer-contrasena` |
| macOS | `sudo -u _prisma prisma admin restablecer-contrasena` |

Muestra una contraseña nueva, cierra las sesiones abiertas y el panel obliga a cambiarla al entrar.

## El certificado venció

Síntoma: todas las aplicaciones dejan de conectarse el mismo día; la bitácora dice `el certificado HTTPS venció`.

- **Autofirmado:** reinicia el servicio (`prisma servicio reiniciar`). Al arrancar, el puente genera un certificado nuevo. Descárgalo del panel (*Ajustes → HTTPS*) y vuelve a instalarlo en **cada** equipo que imprime ([instalacion/https-y-certificados.md](../instalacion/https-y-certificados.md#confiar-en-el-certificado)).
- **Certificado propio (`modo_tls = "archivos"`):** el puente no puede renovarlo. Reemplaza los archivos de `certificado` y `llave` de `config.toml`; el puente los toma en unos segundos, sin reiniciar.

Para que no vuelva a pasar: *Inicio* lo avisa desde 60 días antes.

## Corte de luz o reinicio con tickets en cola

Al volver, el puente marca esos trabajos como fallidos (`PUENTE_REINICIADO`) y libera sus `id`. Las aplicaciones que reintentan el mismo documento lo imprimen una vez. Las que no reintentan muestran el error que recibieron (o ninguno, si la conexión se cortó): revisa en *Trabajos* qué quedó sin imprimir.

## La base de datos está bloqueada (`PUENTE_OCUPADO`)

Otro programa abrió `prisma.db`: un visor de SQLite, un antivirus que la escanea o una herramienta de copias. El puente responde `503 PUENTE_OCUPADO` y no imprime nada hasta que se libera. Cierra ese programa; no hace falta reiniciar. Para respaldar, usa *Ajustes → Respaldo* o copia el archivo con el servicio detenido.

## La base de datos está dañada

Síntoma: el servicio no arranca y la bitácora muestra un error de SQLite al abrir la base.

1. Detén el servicio y conserva la carpeta de datos completa (cópiala aparte).
2. Restaura el respaldo más reciente: un respaldo descargado del panel, o el último de `respaldos/`. Sigue [Restaurar sin el panel](respaldo-y-restauracion.md#restaurar-sin-el-panel).
3. Si no hay ningún respaldo: renombra `prisma.db` y arranca el servicio. El puente crea una base vacía; vuelve a crear la contraseña, las impresoras y las aplicaciones. Las llaves `pk_` anteriores dejan de valer: hay que generar llaves nuevas y entregarlas.

## Falló una restauración desde el panel

La base anterior queda como `prisma.db.antes-de-restaurar`. Si después de restaurar el puente se comporta mal, detén el servicio, renombra `prisma.db.antes-de-restaurar` a `prisma.db` y vuelve a iniciarlo.

Si la restauración falló a mitad (por ejemplo, sin espacio en disco) y el puente no pudo reabrir ninguna base, responde con errores y pide reiniciar el servicio. Al reiniciar abre la base que haya quedado en su lugar; si no arranca, sigue [La base de datos está dañada](#la-base-de-datos-está-dañada).

## Una actualización salió mal

Ver [Volver a la versión anterior](actualizacion.md#volver-a-la-versión-anterior).

## El equipo del puente se perdió o se cambió

1. Instala PRISMA en el equipo nuevo, con la **misma IP** si es posible.
2. Restaura el último respaldo desde el panel.
3. Si respaldaste `tls/`, cópiala a la carpeta de datos antes de arrancar: así las cajas no tienen que volver a instalar el certificado. Si no, instala el certificado nuevo en cada caja.
4. Revisa los destinos de las impresoras: los nombres de impresoras de Windows y los puertos serie cambian de un equipo a otro.
