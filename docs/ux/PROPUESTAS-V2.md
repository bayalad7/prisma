# Propuestas para después de 0.2.0

Cambios al puente que salieron del trabajo de experiencia del panel pero quedan fuera de lo acordado para 0.2.0 (la única excepción de backend fue la impresora virtual y el área de pruebas). Ninguno está implementado.

## El panel comparte el límite de peticiones por IP con la API

- **Qué pasa:** `peticiones_ip_por_minuto` (120 por defecto) cuenta todas las peticiones de una IP al puente, también las del panel. Abrir el panel cuesta unas 11 (la página, sus archivos, la sesión, el resumen); navegar dentro de él, una o dos por pantalla. En uso normal no se nota, pero si el equipo del dueño es también el punto de venta que imprime por `/v1`, las dos cosas se restan del mismo límite: muchas impresiones seguidas pueden dejar el panel sin cargar (los archivos llegan como error `DEMASIADAS_PETICIONES`) y al revés.
- **Cómo se encontró:** el recorrido de la Fase 4 abría las 63 páginas del manual recargando el panel en cada una y superó el límite en menos de un minuto.
- **Propuesta:** no contar los archivos estáticos del panel (son públicos e iguales para todos) y dar al panel con sesión un límite propio, separado del de `/v1`. Requiere decidir los valores y una prueba en `internal/servidor`.
