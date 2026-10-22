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

**Fuentes en el código:**

- internal/conector/conector.go (`Nuevo`)
- internal/conector/red.go
- internal/conector/serie.go
- internal/conector/usb_windows.go
- internal/conector/usb_unix.go (`nuevoUSB`, `Dispositivo`, `CUPS`)
- internal/conector/archivo.go

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
