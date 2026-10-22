# Conectores

¿Qué conector usa cada tipo de impresora y por qué mecanismo llega a ella en cada sistema? Para instaladores y desarrolladores.

```mermaid
flowchart LR
    t{"tipo del perfil"}:::prisma -- red --> red["red: TCP al puerto 9100"]:::prisma
    t -- usb --> u{"sistema"}:::prisma
    u -- windows --> win["winspool: documento RAW<br/>en la cola de Windows"]:::prisma
    u -- "linux y macOS, destino /dev/…" --> dis["dispositivo: escribe<br/>en el archivo del dispositivo"]:::prisma
    u -- "linux y macOS, nombre de cola" --> cups["cups: lp -d cola -o raw"]:::prisma
    t -- bluetooth --> ser["serie: puerto serie, 9600 baudios<br/>por defecto"]:::prisma
    t -- archivo --> arc["archivo: .bin y vista .txt<br/>en una carpeta"]:::datos
    t -- virtual --> vir["virtual: rollo en la memoria<br/>del puente, sin papel"]:::datos
    vir --> sim["simulación opcional: fuera de línea,<br/>lenta o falla aleatoria"]:::prisma
    vir --> panel["panel: imagen del ticket,<br/>.bin y comandos"]:::externo
    red --> conf["Confirmar impresión opcional:<br/>DLE EOT 1"]:::seguridad
    ser --> conf
    red --> imp["impresora"]:::externo
    win --> imp
    dis --> imp
    cups --> imp
    ser --> imp
    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
```

- Todos implementan `Conector{Enviar, Probar}`; `conector.Nuevo` elige según el tipo del perfil, y `nuevoUSB` existe en dos versiones por build tag (Windows y el resto).
- Red: 3 s para conectar y 10 s para escribir; reintenta una vez solo si no envió ningún byte.
- Winspool: si vence el plazo, cancela el trabajo con `AbortPrinter` ([25-confirmar-impresion.md](25-confirmar-impresion.md) explica por qué USB no confirma).
- Confirmar impresión solo existe en red y serie: winspool, CUPS y `/dev/usb` no devuelven la respuesta de la impresora.
- El conector de archivo es para desarrollo y auditoría: guarda los bytes exactos y su previsualización.
- El conector virtual vive dentro del puente: recibe los mismos bytes que una impresora física y los guarda solo en memoria (los últimos 20 por defecto, como mucho 16 MB por impresora) para que el panel los dibuje a 203 ppp. Un ticket que falla por la simulación no se guarda, y su `id` se libera como en cualquier trabajo fallido.

**Fuentes en el código:**

- internal/conector/conector.go (`Nuevo`)
- internal/conector/red.go
- internal/conector/serie.go
- internal/conector/usb_windows.go
- internal/conector/usb_unix.go (`nuevoUSB`, `Dispositivo`, `CUPS`)
- internal/conector/archivo.go
- internal/conector/virtual.go (`Virtual`, `Rollos`)
- internal/dev/raster.go (`Rasterizar`)

**Última verificación:** PRISMA 0.2.0 en desarrollo (rama `develop`), 22 de octubre de 2026.
