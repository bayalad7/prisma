# Detección de impresoras

¿Cómo encuentra el puente las impresoras en cada sistema operativo? Para instaladores que agregan impresoras con el panel o con `prisma impresoras detectar`.

```mermaid
flowchart TD
    d["detectar<br/>internal/conector"]:::prisma --> so{"sistema operativo"}:::prisma
    so -- windows --> w["EnumPrinters: impresoras<br/>locales y conectadas"]:::externo
    so -- "linux y macOS" --> l1["lpstat -v y -e: colas de CUPS"]:::externo
    so -- "linux" --> l2["/dev/usb/lp*: dispositivos"]:::externo
    d --> s["GetPortsList: puertos serie<br/>(Bluetooth SPP o serial)"]:::externo
    d --> q{"¿se pidió escanear la red?"}:::prisma
    q -- sí --> r["probar el puerto 9100 en las subredes<br/>privadas: 64 a la vez, 400 ms por equipo,<br/>15 s en total, /24 como máximo"]:::externo
    w --> res["lista de detectadas: tipo, destino y detalle"]:::datos
    l1 --> res
    l2 --> res
    s --> res
    r --> res
    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
```

- La detección es igual en Linux y macOS (mismo archivo `usb_unix.go`); en macOS no suele haber `/dev/usb/lp*`, así que aparecen solo las colas de CUPS.
- Los puertos serie se listan en los tres sistemas: `COMn` en Windows, `/dev/rfcommN` o `/dev/ttyUSBN` en Linux y `/dev/cu.*` en macOS.
- El escaneo de red nunca es automático: solo con `--red` en la CLI o el botón del panel. Una subred mayor que /24 se reduce al /24 de este equipo.
- Detectar no agrega nada: el dueño elige la impresora y completa su perfil.

**Fuentes en el código:**

- internal/conector/deteccion.go (`Detectar`, `EscanearRed`, `SubredesLocales`)
- internal/conector/usb_windows.go (`impresorasSistema`, `impresorasWindows`)
- internal/conector/usb_unix.go (`impresorasSistema`)
- internal/admin/impresoras.go (`detectar`)
- cmd/prisma/impresoras.go

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
