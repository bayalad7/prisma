# Registros de decisiones de arquitectura (ADR)

Cada archivo registra una decisión que ya está tomada en el código: el contexto, la decisión, sus consecuencias y dónde se ve en el código. Las reglas que no se cambian sin preguntar están en [CLAUDE.md](../../CLAUDE.md#decisiones-técnicas-no-las-cambies-sin-preguntar).

| # | Decisión | Fecha |
|---|---|---|
| 0001 | [Go sin CGO, un binario por plataforma](0001-go-sin-cgo-un-binario.md) | 2026-10-14 |
| 0002 | [SQLite en Go puro y `config.toml` solo para el arranque](0002-sqlite-en-go-puro.md) | 2026-10-14 |
| 0003 | [Operaciones JSON sin estado en lugar de bytes ESC/POS](0003-operaciones-json-sin-estado.md) | 2026-10-14 |
| 0004 | [Alias de impresora en lugar de direcciones](0004-alias-en-vez-de-direcciones.md) | 2026-10-14 |
| 0005 | [Tres niveles de seguridad por cliente, con el 2 por defecto](0005-tres-niveles-de-seguridad.md) | 2026-10-15 |
| 0006 | [Firma Ed25519 sobre la forma canónica JCS en el nivel 3](0006-firma-ed25519-jcs.md) | 2026-10-14 |
| 0007 | [Llaves `pk_` con argon2id, prefijo público y caché](0007-llaves-argon2id.md) | 2026-10-15 |
| 0008 | [Idempotencia persistente, con liberación del `id` de lo que no se imprimió](0008-idempotencia-persistente.md) | 2026-10-15 |
| 0009 | [Un trabajo que llegó a la impresora no falla](0009-trabajo-recibido-no-falla.md) | 2026-10-21 |
| 0010 | [Certificado autofirmado de 825 días, con aviso y regeneración](0010-certificado-autofirmado.md) | 2026-10-15 |
| 0011 | [Servicio del sistema con kardianos/service y plantillas propias](0011-servicio-kardianos.md) | 2026-10-20 |
| 0012 | [Actualizaciones firmadas con una llave Ed25519 del proyecto](0012-actualizaciones-firmadas.md) | 2026-10-20 |
| 0013 | [Panel sin frameworks ni CDN, con sesiones en memoria](0013-panel-sin-frameworks.md) | 2026-10-19 |
| 0014 | [CI y publicación con el último parche de Go 1.26](0014-go-ultimo-parche-en-ci.md) | 2026-10-22 |
| 0015 | [Sistema de diseño propio para el panel](0015-sistema-de-diseno-del-panel.md) | 2026-10-22 |
| 0016 | [Impresora virtual dentro del puente, con tickets solo en memoria](0016-impresora-virtual-en-el-puente.md) | 2026-10-22 |

## Cómo agregar una decisión

1. Copia la estructura de cualquier ADR con el número siguiente: `NNNN-titulo-en-minusculas.md`.
2. Escribe el contexto (qué problema había), la decisión, las consecuencias (también las malas) y la evidencia en el código.
3. Una decisión que reemplaza a otra no la borra: la nueva dice «reemplaza a NNNN» y la vieja cambia su estado a «reemplazada por NNNN».
4. Si la decisión está en la lista de CLAUDE.md, actualízala ahí también.
