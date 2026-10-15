# Configuración

PRISMA tiene dos niveles de configuración:

| Dónde | Qué contiene | Cómo se cambia |
|---|---|---|
| `config.toml` | Solo lo necesario para **arrancar**: red, TLS, ruta de datos y modo desarrollo. | Editando el archivo y reiniciando. |
| Base de datos (`prisma.db`) | Todo lo demás: impresoras, clientes, orígenes, permisos, recursos, retención y contraseña del panel. | ⏳ Desde el panel. Hoy, con la CLI (`prisma impresoras …`, `prisma admin …`). |

## `config.toml`

**Ubicación por defecto:**
- Windows: `%ProgramData%\PRISMA\config.toml`
- Linux: `/etc/prisma/config.toml`
- macOS: `/Library/Application Support/PRISMA/config.toml`

Puedes usar otra con `--config ruta`. Si el archivo por defecto no existe, PRISMA usa los valores por defecto. En [empaquetado/config.toml](../empaquetado/config.toml) hay un ejemplo comentado.

```toml
[servidor]
escuchar = "0.0.0.0"
puerto = 8443
modo_tls = "autofirmado"
certificado = ""
llave = ""

[datos]
ruta = ""

[dev]
modo_desarrollo = false
```

### `[servidor]` ⏳

Estas claves ya se validan, pero se usarán cuando exista el servidor HTTPS.

| Clave | Por defecto | Valores |
|---|---|---|
| `escuchar` | `"0.0.0.0"` | Dirección IP o `"localhost"`. `0.0.0.0` acepta conexiones de toda la red local; `127.0.0.1`, solo de este equipo. |
| `puerto` | `8443` | De 1 a 65535. |
| `modo_tls` | `"autofirmado"` | Ver la tabla siguiente. |
| `certificado`, `llave` | `""` | Rutas a archivos PEM. **Solo** con `modo_tls = "archivos"`; las rutas relativas se resuelven desde la carpeta de `config.toml`. |

| `modo_tls` | Uso |
|---|---|
| `autofirmado` | El puente genera y guarda su propio certificado. Es lo recomendado para empezar. |
| `archivos` | Usa `certificado` y `llave`. `config validar` comprueba que existan y que formen un par válido. |
| `solo_localhost_http` | HTTP sin cifrar. Solo se acepta si `escuchar` es `127.0.0.1`, `::1` o `localhost`; nunca se puede usar en la red. |

### `[datos]`

| Clave | Por defecto | Valores |
|---|---|---|
| `ruta` | `""` | Carpeta de la base de datos. Vacío significa la ruta del sistema (ver `prisma config ruta`). Si se indica, debe ser absoluta. En Windows, usa comillas simples para no escapar las barras: `ruta = 'D:\PRISMA\datos'`. |

### `[dev]`

| Clave | Por defecto | Valores |
|---|---|---|
| `modo_desarrollo` | `false` | Con `true` se habilita `prisma imprimir`. **Nunca lo actives en un negocio.** |

### Validación

`prisma config validar` (y cada arranque) revisa el archivo y reporta **todos** los problemas juntos, en español:

- claves desconocidas (por ejemplo, `servidor.purto: clave desconocida`);
- errores de sintaxis, con su número de línea;
- valores fuera de rango o incompatibles entre sí.

```
Error: /etc/prisma/config.toml: la configuración tiene 2 problema(s):
  - servidor.puerto: debe estar entre 1 y 65535 (valor: 0)
  - servidor.modo_tls: "x" no es válido; usa "autofirmado", "archivos" o "solo_localhost_http"
```

## Ajustes en la base de datos

| Clave | Por defecto | Significado |
|---|---|---|
| `trabajos.retencion_dias` | 30 | Días que se conserva el historial de trabajos. |
| `admin.contrasena_hash` | — | Hash argon2id de la contraseña del panel. Se crea con `prisma admin restablecer-contrasena`. |
| `admin.debe_cambiar` | — | `1` obliga a cambiar la contraseña al entrar al panel. |

⏳ El panel permitirá editarlos. Las tablas completas están en [arquitectura.md](arquitectura.md#datos).

## Perfil de una impresora

Cada impresora tiene un perfil que el renderizador usa para que el ticket salga bien. Se define con `prisma impresoras agregar` (ver [cli.md](cli.md#impresoras-agregar)).

| Campo | Para qué sirve | Cómo averiguarlo |
|---|---|---|
| Columnas (fuente A / B) | Partir y alinear texto, filas y pares. | La hoja de autoprueba suele indicarlo («Font A: 48 columns»). Si el texto se parte antes de tiempo o se desborda, este valor está mal. |
| Puntos por línea (`ancho_px`) | Tamaño de imágenes, QR y códigos de barras. | 576 en 80 mm y 384 en 58 mm en la mayoría de los modelos. |
| Página de códigos y comando | Acentos, ñ, ¿, ¡ y €. | Las tablas de la autoprueba o el manual. Ver [solucion-de-problemas.md](solucion-de-problemas.md#acentos-o-ñ-salen-mal). |
| Capacidades | Decidir si el QR o el código de barras se envía por comando o como imagen, y qué hacer sin cuchilla o sin cajón. | Prueba con `ejemplos/validos/todas-las-operaciones.json`. |
