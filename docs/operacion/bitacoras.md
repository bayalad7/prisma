# Bitácoras

El puente escribe una línea JSON por evento (`log/slog`). **Nunca** registra el contenido de los tickets, las llaves `pk_`, las contraseñas, los códigos de configuración ni las firmas.

## Dónde están

| Sistema | Archivo | Además |
|---|---|---|
| Windows | `C:\ProgramData\PRISMA\logs\prisma.log` | El *Visor de eventos* muestra si el servicio no llegó a arrancar. |
| Linux | `/var/log/prisma/prisma.log` | `journalctl -u prisma -f` |
| macOS | `/Library/Logs/PRISMA/prisma.log` | `launchd.err` en la misma carpeta, y la app **Consola**. |

El archivo rota al llegar a 10 MB y conserva los 5 anteriores (`prisma.log.1` … `prisma.log.5`). Con `prisma ejecutar`, la bitácora va a la salida de errores de la terminal.

Para leerla en vivo:

```sh
tail -f /var/log/prisma/prisma.log | jq .                            # Linux y macOS
Get-Content C:\ProgramData\PRISMA\logs\prisma.log -Tail 50 -Wait     # Windows
```

## Formato

```json
{"time":"2026-10-22T14:03:11.52Z","level":"INFO","msg":"trabajo","trabajo":"trb_cb1393f662e11be7","cliente":"cli_4f2k9","impresora":"caja","estado":"completado","codigo_error":"","duracion_ms":412}
```

`time`, `level` (`INFO`, `WARN`, `ERROR`) y `msg` van siempre. Los demás campos dependen del evento.

## Eventos

### Arranque y mantenimiento

| `msg` | Nivel | Campos | Significa |
|---|---|---|---|
| `iniciando PRISMA` | INFO | `version`, `config`, `config_existe`, `modo_desarrollo` | Arrancó. `modo_desarrollo` debe ser `false` en un negocio. |
| `base de datos respaldada antes de migrarla` | INFO | `respaldo` | Una versión nueva va a migrar la base; aquí quedó la copia. |
| `base de datos lista` | INFO | `ruta`, `esquema` | Base abierta y migrada. |
| `trabajos interrumpidos por el reinicio del puente` | WARN | `cantidad` | Había tickets en cola al apagarse; quedaron como `PUENTE_REINICIADO` y se pueden reenviar. |
| `mantenimiento de la base de datos` | INFO | `trabajos_purgados`, `ids_purgados`, `solicitudes_purgadas`, `retencion_dias` | Limpieza horaria. |
| `el panel aún no tiene contraseña…` | WARN | — | Falta el [primer acceso](../administracion/README.md#primer-acceso). |
| `certificado autofirmado` | INFO | `archivo`, `huella_sha256`, `vence` | Certificado en uso. |
| `el certificado no incluye algunas IPs actuales…` | WARN | — | Cambió la IP del equipo: regenera el certificado. |
| `el certificado HTTPS está por vencer…` | WARN | `vence`, `dias` | Quedan menos de 60 días. |
| `el certificado HTTPS venció…` | ERROR | `vencio` | Ningún cliente puede conectarse. Ver [recuperacion.md](recuperacion.md#el-certificado-venció). |
| `API escuchando` | INFO | `url` | Listo para recibir peticiones. |
| `PRISMA detenido` | INFO | — | Se detuvo de forma ordenada. |

### Peticiones e impresión

| `msg` | Nivel | Campos | Significa |
|---|---|---|---|
| `petición` | INFO | `metodo`, `ruta` (sin parámetros), `estado`, `duracion_ms`, `ip`, `cliente` | Una petición HTTP a la API o al panel. |
| `trabajo` | INFO | `trabajo`, `cliente`, `impresora`, `estado`, `codigo_error`, `duracion_ms` | Resultado de un trabajo. |
| `estado de impresora` | INFO / WARN | `alias`, `estado`, `codigo` | Una impresora pasó a en línea o a fuera de línea (solo los cambios). |
| `no se pudo preparar la impresora` | WARN | `alias`, `error` | El perfil tiene un destino inválido. |
| `no se pudo liberar el id del trabajo fallido` | ERROR | `trabajo`, `error` | Falló la base al liberar un `id`. El reintento igual se imprime. |
| `no se pudo guardar el resultado del trabajo` | ERROR | `trabajo`, `error` | La base no aceptó el resultado (por ejemplo, bloqueada). |
| `base de datos ocupada` | WARN | `metodo`, `ruta` | Se respondió `PUENTE_OCUPADO`: otro programa tenía bloqueada la base. |
| `error interno` | ERROR | `metodo`, `ruta`, `error` | Se respondió `ERROR_INTERNO`. El detalle solo queda aquí, nunca en la respuesta. |

### Panel

| `msg` | Nivel | Significa |
|---|---|---|
| `contraseña del panel creada` · `cambiada` | INFO | Primer acceso o cambio de contraseña. |
| `sesión iniciada en el panel` | INFO | Alguien entró (campo `ip`). |
| `intento fallido de entrar al panel` | WARN | Contraseña incorrecta (campo `ip`). Muchos seguidos indican que alguien la está adivinando. |
| `impresora agregada · eliminada desde el panel` | INFO | Cambios de impresoras (campos `alias` y `tipo`). |
| `aplicación creada · actualizada · eliminada desde el panel` | INFO | Cambios de aplicaciones (campo `cliente`). |
| `llave de aplicación regenerada · revocada desde el panel` | INFO | Cambios de llaves. Nunca se registra la llave. |
| `origen aprobado desde el panel` · `con una aplicación nueva` | INFO | Solicitudes aprobadas (campo `cliente`). |
| `imagen guardada desde el panel` | INFO | Recurso nuevo. |
| `ajustes cambiados desde el panel` | INFO | Límites, retención, acceso, etc. |
| `certificado autofirmado regenerado desde el panel` | INFO | `autoridad_nueva`. Con `false`, lo firmó la misma autoridad local y los equipos no tienen que hacer nada; con `true`, hay que instalar la autoridad nueva en cada equipo. |
| `respaldo descargado desde el panel` | INFO | Alguien descargó la base completa. La IP está en la línea `petición` de esa descarga. |
| `base de datos restaurada desde un respaldo` | WARN | Se reemplazó la base; se cerraron todas las sesiones. |

### Actualizaciones

| `msg` | Nivel | Significa |
|---|---|---|
| `hay una versión nueva de PRISMA` | INFO | El panel la avisa en *Inicio*. |
| `no se pudo comprobar si hay una versión nueva` | WARN | Sin Internet o GitHub no respondió. No afecta la impresión. |
