# Estados de una impresora

¿Cuándo una impresora está en línea, fuera de línea, ocupada o en estado desconocido?

```mermaid
stateDiagram-v2
    [*] --> desconocido: se registra en la cola
    desconocido --> en_linea: prueba correcta
    desconocido --> fuera_de_linea: prueba fallida
    en_linea --> ocupada: empieza un trabajo
    fuera_de_linea --> ocupada: empieza un trabajo
    ocupada --> en_linea: trabajo enviado
    ocupada --> fuera_de_linea: error, o Aviso de Confirmar impresión
    en_linea --> fuera_de_linea: prueba fallida (cada 30 s)
    fuera_de_linea --> en_linea: prueba correcta (cada 30 s)
```

La misma goroutine de cada impresora imprime y prueba, así que nunca prueba mientras imprime. Una impresora desactivada se muestra como *inactiva* en el panel y no aparece en `/v1/impresoras`.

**Basado en:**

- internal/cola/cola.go (`cambiar`, `probar`, `imprimir`)
- internal/conector/conector.go (`Conector.Probar`)
