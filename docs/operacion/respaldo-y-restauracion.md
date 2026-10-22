# Respaldo y restauración

Todo lo que se configura en el panel vive en un solo archivo: `prisma.db`, en la carpeta de datos ([dónde está](README.md#dónde-está-cada-cosa)). El certificado autofirmado está aparte, en `tls/`, y `config.toml` también.

## Qué contiene un respaldo

| Contiene | No contiene |
|---|---|
| Impresoras y sus perfiles · aplicaciones, orígenes, permisos, hashes de llaves y llaves públicas · solicitudes de origen · historial de trabajos (sin el contenido de los tickets) · imágenes guardadas · ajustes, incluido el hash de la contraseña del panel | `config.toml` · el certificado y su llave (`tls/`) · las bitácoras |

El respaldo contiene datos sensibles del negocio (hashes de llaves y de la contraseña). Guárdalo en un lugar privado y fuera del equipo del puente.

## Respaldar

- **Desde el panel:** *Ajustes → Respaldo → Descargar*. Hace una copia consistente con el puente en marcha, sin detener nada.
- **Automático antes de migrar:** si una versión nueva cambia la base, al arrancar se copia a `respaldos/prisma-esquema<N>-<versión>-<fecha>.db` y se conservan las 5 más recientes.
- **Copiando el archivo:** solo con el servicio **detenido** (`prisma servicio detener`). Con el puente en marcha, una copia directa de `prisma.db` puede quedar incompleta (la base usa WAL: también hay `prisma.db-wal`), y abrir la base con otro programa puede bloquearla (`PUENTE_OCUPADO`).

Para no perder el certificado (y tener que reinstalarlo en cada caja), respalda también la carpeta `tls/` y `config.toml`.

## Restaurar

1. *Ajustes → Restaurar* y elige el archivo.
2. El puente lo valida (una base SQLite íntegra, de PRISMA y con un esquema que esta versión conozca), reemplaza la base con una pausa breve, recarga impresoras y ajustes, y **cierra todas las sesiones**.
3. Entra de nuevo con la contraseña **que tenía el respaldo**.

La base anterior queda junto a la nueva como `prisma.db.antes-de-restaurar`, por si hay que deshacer.

### Restaurar sin el panel

Si el panel no abre (por ejemplo, la base está dañada):

1. Detén el servicio: `prisma servicio detener`.
2. Renombra `prisma.db` (y `prisma.db-wal`, `prisma.db-shm` si existen) para conservarla.
3. Copia el respaldo como `prisma.db` en la carpeta de datos, **con el mismo usuario del servicio** como dueño:
   - Linux: `sudo install -o prisma -g prisma -m 0600 respaldo.db /var/lib/prisma/prisma.db`
   - macOS: `sudo install -o _prisma -g _prisma -m 0600 respaldo.db "/Library/Application Support/PRISMA/datos/prisma.db"`
   - Windows: cópialo en `C:\ProgramData\PRISMA\datos\prisma.db` desde una PowerShell de administrador.
4. Inicia el servicio: `prisma servicio iniciar`.

Si el respaldo es de una versión anterior, el puente lo migra al arrancar (y antes guarda una copia en `respaldos/`). Si es de una versión **más nueva**, el puente se niega a abrirlo: instala esa versión o una posterior.
