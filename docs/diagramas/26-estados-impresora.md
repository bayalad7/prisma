# Estados de una impresora

¿Cuándo una impresora está en línea, fuera de línea, ocupada o en estado desconocido? Para el dueño que ve el estado en el panel e integradores que leen `/v1/impresoras`.

```mermaid
stateDiagram-v2
    [*] --> desconocido: se registra en la cola
    desconocido --> en_linea: prueba correcta
    desconocido --> fuera_de_linea: prueba fallida
    en_linea --> ocupada: empieza un trabajo
    fuera_de_linea --> ocupada: empieza un trabajo
    ocupada --> en_linea: trabajo enviado
    ocupada --> fuera_de_linea: error o Aviso
    en_linea --> fuera_de_linea: prueba fallida, cada 30 s
    fuera_de_linea --> en_linea: prueba correcta, cada 30 s

    classDef externo fill:#e8f0fe,stroke:#3b6db3,color:#102a43
    classDef prisma fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13
    classDef datos fill:#fff4e0,stroke:#b26a00,color:#3d2400
    classDef seguridad fill:#fde8e8,stroke:#c62828,color:#4a0d0d
    class desconocido externo
    class en_linea prisma
    class ocupada datos
    class fuera_de_linea seguridad
```

- La misma goroutine de cada impresora imprime y prueba: nunca prueba mientras imprime.
- *Probar* abre la conexión (red, Bluetooth) o consulta el spooler (USB en Windows); no detecta falta de papel. Para eso, Confirmar impresión.
- Una impresora desactivada no está en la cola: el panel la muestra como *inactiva* y no aparece en `/v1/impresoras`.

**Fuentes en el código:**

- internal/cola/cola.go (`cambiar`, `probar`, `imprimir`, `Estado`)
- internal/conector/conector.go (`Conector.Probar`)
- internal/servidor/api.go (`impresoras`)

**Última verificación:** PRISMA 0.1.0 (rama `develop`, commit `5f67e19`), 22 de octubre de 2026.
