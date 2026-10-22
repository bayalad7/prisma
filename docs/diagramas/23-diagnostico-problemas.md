# Diagnóstico: «no imprime»

¿Por qué no salió el ticket y qué hacer? Árbol de decisión para soporte y para el dueño del negocio. El panel lo convierte en preguntas paso a paso en *Ayuda → Mi impresora no imprime*.

```mermaid
flowchart TD
    s["no imprime"]:::externo --> p{"¿sale el Ticket de prueba<br/>del panel (Impresoras)?"}:::prisma
    p -- "no" --> pq{"¿qué dice el panel?"}:::prisma
    pq -- "No responde · IMPRESORA_INALCANZABLE · TIEMPO_AGOTADO" --> c5["impresora apagada, sin papel, tapa abierta,<br/>cable o red desconectados, Bluetooth fuera de alcance"]:::externo
    pq -- "PUERTO_OCUPADO · PERMISO_DENEGADO" --> c6["otro programa usa el puerto,<br/>o faltan grupos lp/dialout"]:::prisma
    pq -- "PUENTE_OCUPADO" --> c7["otro programa abrió prisma.db"]:::datos
    pq -- "dice que salió, pero no salió" --> ok{"¿es una impresora de red?"}:::prisma
    ok -- "sí" --> c8["activar Confirmar impresión"]:::seguridad
    ok -- "no, USB en Windows" --> c9["revisar la cola de Windows:<br/>trabajos atascados o sin conexión"]:::externo
    p -- "sí" --> v{"¿tu aplicación imprime en una<br/>impresora virtual?"}:::prisma
    v -- "sí" --> c10["la aplicación funciona: revisa que use el alias<br/>de esta impresora y que tenga permiso para ella"]:::seguridad
    v -- "no" --> r{"¿la aplicación recibe<br/>una respuesta del puente?"}:::prisma
    r -- "no: error de red" --> red{"¿abre https://ip:8443/v1/salud<br/>sin aviso de certificado?"}:::externo
    red -- "aviso de certificado" --> c1["instalar el certificado<br/>en ese equipo"]:::seguridad
    red -- "no carga" --> c2["puente apagado, otra red o firewall:<br/>prisma servicio estado"]:::prisma
    red -- "sí carga" --> c3["origen no autorizado:<br/>aprobarlo en Solicitudes"]:::seguridad
    r -- "sí, con un error 401 o 403" --> c4["llave, origen o permisos de la aplicación:<br/>Aplicaciones → Comprobar permisos"]:::seguridad
    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
```

- La primera pregunta separa la **impresora** (si el ticket de prueba del panel no sale, el problema está entre el puente y la impresora) de la **aplicación**.
- Con la impresora bien, la [impresora virtual](../administracion/probar-sin-impresora.md) separa los problemas de la aplicación de los de esa impresora: si la aplicación imprime en la virtual, funciona, y lo que falta es el alias o el permiso.
- Desde un navegador, un origen no aprobado se ve como error de red: si `/v1/salud` carga, casi siempre es eso.
- Con la advertencia `IMPRESORA_FUERA_DE_LINEA`, el ticket ya está en la impresora: se imprime al cerrar la tapa o poner papel. **No reenviar.**

**Soluciones en el manual** (el panel enlaza cada respuesta del árbol con su línea; una prueba exige que cada final del árbol tenga la suya):

- `v`: [Probar sin impresora](../administracion/probar-sin-impresora.md)
- `c1`: [Confiar en el certificado](../instalacion/https-y-certificados.md#confiar-en-el-certificado)
- `c2`: [El panel o las aplicaciones no llegan desde otro equipo](../instalacion/problemas-comunes.md#el-panel-o-las-aplicaciones-no-llegan-desde-otro-equipo)
- `c3`: [Origen no autorizado](../integradores/problemas-comunes.md#origen-no-autorizado)
- `c4`: [¿Por qué mi app no puede imprimir?](../administracion/probar-impresion.md#por-qué-mi-app-no-puede-imprimir)
- `c5`: [La impresora no imprime](../administracion/problemas-comunes.md#la-impresora-no-imprime)
- `c6`: [Linux: la impresora USB no responde](../instalacion/problemas-comunes.md#linux-la-impresora-usb-no-responde-permiso-denegado)
- `c7`: [503 PUENTE_OCUPADO](../administracion/problemas-comunes.md#503-puente_ocupado)
- `c8`: [Dice «impreso», pero no salió](../administracion/problemas-comunes.md#dice-impreso-pero-no-salió-impresora-de-red)
- `c9`: [Windows dice que imprimió, pero no salió nada](../administracion/problemas-comunes.md#windows-dice-que-imprimió-pero-no-salió-nada)
- `c10`: [Elegir en qué impresoras imprime una aplicación](../administracion/tareas-frecuentes.md#elegir-en-qué-impresoras-imprime-una-aplicación)

**Fuentes en el código:**

- internal/conector/conector.go (códigos `Err…`)
- internal/conector/red.go
- internal/conector/usb_windows.go
- internal/servidor/servidor.go (`ResponderError`)
- internal/seguridad/autorizacion.go
- internal/admin/web/manual.js (`flujo`) e internal/admin/web/panel.js (`vistaNoImprime`)
- sdk/js/src/prisma.mjs (`_diagnosticarNavegador`)
- docs/referencia/codigos-de-error.md

**Última verificación:** PRISMA 0.2.0 en desarrollo (rama `develop`), 22 de octubre de 2026.
