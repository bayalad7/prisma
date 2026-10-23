# 0016. Impresora virtual dentro del puente, con tickets solo en memoria

- **Estado:** aceptada, en desarrollo para PRISMA 0.2.0
- **Fecha:** 2026-10-22

## Contexto

Para probar PRISMA hacía falta una impresora física, o la terminal: `prisma dev impresora-virtual` (un servidor TCP aparte) o el conector `archivo` (tickets en una carpeta del equipo del puente). Ninguna de las dos sirve a una persona no técnica ni a quien integra una aplicación desde otro equipo. El análisis del panel ([docs/ux/ANALISIS-PANEL-V1.md](../ux/ANALISIS-PANEL-V1.md)) lo marcó como crítico: nadie podía llegar a «ticket de prueba impreso» sin hardware.

La regla del proyecto es no guardar el contenido de los tickets. Ver el ticket de una impresora virtual exige guardarlo en algún lado.

## Decisión

- Un tipo de conexión nuevo, `virtual` (migración 0005), con su conector en `internal/conector`. Vive dentro del proceso del puente: sin red, puertos ni archivos. Recibe exactamente los bytes que `escpos.Renderizar` daría a una impresora física, por la misma cola.
- Los tickets recibidos se guardan **solo en memoria** (`conector.RollosVirtuales`): los últimos N por impresora (20 por defecto, de 1 a 200) y como mucho 16 MB por impresora. Nunca van a la base, la bitácora ni los respaldos, y se pierden al reiniciar. Es la única excepción a la regla de no guardar tickets, y la decidió el dueño del proyecto. Lo verifican `TestImpresoraVirtualDePrincipioAFin` (por `/v1`) y `TestProbarImpresionNoGuardaElContenido` (por el panel: vista previa, envío, errores, rollo y respaldo descargado).
- El panel los dibuja como saldrían en papel: `dev.Rasterizar` interpreta la página simulada a 203 ppp, con las fuentes A (12×24) y B (9×17), tamaños, negrita, subrayado, invertido, imágenes, QR, códigos de barras y guías de corte y cajón.
- Simulación de fallas por impresora, solo con lo que la API v1 ya informa: «fuera de línea» (`IMPRESORA_INALCANZABLE`), «lenta» (retraso de hasta 30 s) y «falla aleatoria» (porcentaje). Un ticket que falla no se guarda y libera su `id`, como cualquier trabajo fallido.
- `GET /v1/impresoras` agrega el campo `virtual` (aditivo; el contrato no cambia), y los trabajos guardan si fueron a una impresora virtual.
- **Dependencia nueva:** `golang.org/x/image` (BSD, del equipo de Go, Go puro), por la fuente Go Mono y su rasterizador. Dibujar el texto a mano con una fuente de mapa de bits propia habría costado mucho más, sin acentos ni símbolos completos.

## Consecuencias

- Cualquier persona puede ver su primer ticket sin impresora, y una aplicación real puede probarse de punta a punta (autorización incluida) contra una impresora simulada.
- El binario crece por la fuente embebida y su rasterizador: medido en Windows amd64 con `-s -w`, pasó de 17.0 a 17.5 MB (545 KB, incluidos los archivos del panel de la Fase 2).
- Las pruebas verifican que los bytes del rollo sean los del renderizador, que el contenido no llegue a la bitácora ni a la base, y que una falla simulada libere el `id`.
- La CLI (`prisma imprimir`) corre en otro proceso, sin acceso a la memoria del puente: con una impresora virtual, muestra el ticket en la terminal.

## Evidencia en el código

`internal/conector/virtual.go`, `internal/dev/raster.go`, `internal/almacen/migraciones/0005_impresora_virtual.sql`, `internal/admin/virtual.go`, `internal/servidor/api.go` (`impresoraVisible`), `internal/servidor/virtual_test.go`.
