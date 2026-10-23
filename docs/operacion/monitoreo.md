# Monitoreo

PRISMA no tiene métricas ni alertas propias: se vigila con lo que ya expone.

## ¿Está en marcha el puente?

- **Desde la red:** `GET https://<equipo>:8443/v1/salud` responde `{"ok": true, "version": …}` sin autorización. Es la comprobación adecuada para un monitor externo (Uptime Kuma, Zabbix, un script con `curl --cacert prisma.crt -fsS`).
- **En el equipo:** `prisma servicio estado` dice «en marcha», «detenido» o «no instalado».

`/v1/salud` cuenta para el límite de peticiones por IP (120 por minuto por defecto). Un monitor que consulta cada minuto no se acerca.

## ¿Funcionan las impresoras?

- **Panel, *Inicio* e *Impresoras*:** el estado de cada una (*en línea*, *fuera de línea*, *ocupada*, *desconocido*) y su último error.
- **API:** `GET /v1/impresoras` devuelve el `estado` de las impresoras que la aplicación puede usar. Requiere autorización.
- **Bitácora:** el evento `estado de impresora` se registra solo cuando una impresora cambia entre en línea y fuera de línea, con el código del error.

El estado sale de una prueba cada 30 s: abrir la conexión (red, Bluetooth) o consultar el spooler (USB en Windows). Una impresora puede estar *en línea* y aun así sin papel: para detectarlo, activa [Confirmar impresión](../administracion/impresoras.md#confirmar-impresión).

## Señales en la bitácora

| Señal | Qué revisar |
|---|---|
| `trabajo` con `estado: fallido` repetidos para la misma impresora | La impresora: papel, tapa, red, Bluetooth. |
| `trabajos interrumpidos por el reinicio del puente` | Cortes de luz o reinicios del equipo con tickets en cola. |
| `intento fallido de entrar al panel` repetidos | Alguien está probando contraseñas. Revisa la IP. |
| `base de datos ocupada` | Un programa externo abre `prisma.db`. |
| `el certificado HTTPS está por vencer` | Regenerarlo antes de que venza. |
| Muchas solicitudes de origen nuevas en el panel | Un equipo de la red está probando el puente con orígenes inventados. |

El formato y la lista completa de eventos están en [bitacoras.md](bitacoras.md).
