# Pruebas antes de publicar

| Documento | Para qué |
|---|---|
| [matriz.md](matriz.md) | Matriz manual con hardware y dispositivos reales: instalación por sistema, impresoras (sistema × conexión × ancho), aplicaciones (navegador × nivel), combinaciones por pares y resiliencia. Incluye los casos que **no** deben funcionar, con su mensaje. |
| [reporte-verificacion.md](reporte-verificacion.md) | Última verificación integral: hallazgos por severidad, evidencia y estado. |

Las pruebas automáticas (`go test -race ./...`, SDK, firmadores, y la instalación real del servicio en Windows, Ubuntu y macOS) corren en el CI en cada push. Ver [desarrollo.md](../desarrollo.md#tareas).
