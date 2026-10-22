# Contrato «documento v1»

Esta especificación describe el JSON que cualquier aplicación envía a un puente PRISMA para imprimir un ticket.
Va dirigida a quienes integran PRISMA en sus sistemas.

- Esquema formal: [`schema/documento-v1.schema.json`](../../schema/documento-v1.schema.json) (JSON Schema draft 2020-12).
- Ejemplos válidos: [`ejemplos/validos/`](../../ejemplos/validos/).
- Ejemplos inválidos, uno por regla: [`ejemplos/invalidos/`](../../ejemplos/invalidos/), con su índice en `indice.json`.

Las palabras DEBE, NO DEBE y PUEDE tienen el sentido normativo de RFC 2119.

## Contenido

1. [Ideas clave](#1-ideas-clave)
2. [El sobre](#2-el-sobre)
3. [Texto, ancho y partido de líneas](#3-texto-ancho-y-partido-de-líneas)
4. [Operaciones](#4-operaciones)
5. [Límites](#5-límites)
6. [Capacidades de la impresora](#6-capacidades-de-la-impresora)
7. [Firma (nivel 3)](#7-firma-nivel-3)
8. [Idempotencia](#8-idempotencia)
9. [Validación y errores](#9-validación-y-errores)
10. [Política de versiones](#10-política-de-versiones)
11. [Ejemplos](#11-ejemplos)

---

## 1. Ideas clave

- **Envías operaciones, no bytes.** El puente traduce cada operación a ESC/POS según el perfil de la impresora (ancho, columnas, página de códigos, capacidades). El mismo documento funciona en una impresora de 58 mm o de 80 mm.
- **Cada operación lleva su propio estilo.** No hay estado compartido entre operaciones: una `negrita: true` no afecta a la siguiente. Si un atributo se omite, se usa su valor por defecto, nunca el de la operación anterior.
- **Solo nombras la impresora por su alias** (`"caja"`, `"cocina"`). Las IPs, puertos y nombres del sistema son configuración privada del puente.
- **El contrato es cerrado.** Un campo, una operación o una versión que el puente no conoce provoca un rechazo con un código claro. Nada se ignora en silencio.
- **El puente nunca descarga nada.** Las imágenes viajan dentro del documento o ya están guardadas en el puente.
- **El puente no añade operaciones por su cuenta.** Si quieres corte al final, incluye `cortar`.

## 2. El sobre

```json
{
  "version": "1",
  "id": "3f2b8c1e-5a4d-4e7b-9c2a-1d0e6f8a7b90",
  "impresora": "caja",
  "copias": 1,
  "referencia": "PED-000812",
  "operaciones": [
    {"tipo": "texto", "valor": "Hola, mundo"},
    {"tipo": "cortar"}
  ]
}
```

| Campo | Tipo | Obligatorio | Por defecto | Regla |
|---|---|---|---|---|
| `version` | cadena | sí | — | Exactamente `"1"`, como cadena y no como número. |
| `id` | cadena | sí | — | UUID v4 en minúsculas, generado por el cliente para cada ticket. Ver [§8](#8-idempotencia). |
| `impresora` | cadena | sí | — | Alias configurado en el puente: `^[a-z0-9][a-z0-9_-]{0,31}$`. |
| `copias` | entero | no | `1` | De 1 a 5. Cada copia repite **todas** las operaciones, cortes incluidos. |
| `referencia` | cadena | no | — | De 1 a 64 caracteres, sin caracteres de control. Aparece en la bitácora del puente y no se imprime. |
| `operaciones` | arreglo | sí | — | De 1 a 500 operaciones, que se ejecutan en orden. |
| `cliente_id` | cadena | nivel 3 | — | `^cli_[A-Za-z0-9]{8,40}$`. |
| `emitido` | cadena | nivel 3 | — | Fecha y hora RFC 3339 con zona horaria, `T` y `Z` en mayúsculas. |
| `expira` | cadena | nivel 3 | — | Igual que `emitido`, y como máximo 120 s después de él. |
| `firma` | cadena | nivel 3 | — | Base64 estándar de 64 bytes, es decir, 88 caracteres terminados en `==`. |

Reglas generales:

- No se admiten campos adicionales, ni en el sobre ni en las operaciones.
- `null` nunca es un valor válido. Si no quieres enviar un campo opcional, omítelo.
- Los campos `cliente_id`, `emitido`, `expira` y `firma` van **los cuatro juntos o ninguno**.
- Las credenciales **no** forman parte del documento. La llave de cliente del nivel 2 (`pk_...`) viaja en el encabezado HTTP `Authorization: Bearer pk_...`. Gracias a eso, el mismo documento sirve en cualquier nivel de seguridad.
- El JSON DEBE ser I-JSON (RFC 7493): codificado en UTF-8, sin claves duplicadas y sin BOM.

## 3. Texto, ancho y partido de líneas

### Codificación

- Todo el texto viaja en UTF-8. El puente lo normaliza a NFC y lo convierte a la página de códigos del perfil de la impresora.
- Un carácter que no existe en esa página **nunca provoca un error**. Se sustituye y el trabajo lleva la advertencia `CARACTER_SUSTITUIDO`:
  1. Los signos tipográficos pasan a su equivalente ASCII: `“ ”` → `"`, `‘ ’` → `'`, `– —` → `-`, `…` → `.`.
  2. Las letras con acento se imprimen sin él (`Á` → `A` en CP437).
  3. Lo demás se imprime como `?`.
- Tú no tienes que conocer la página de códigos. El puente soporta CP437, CP850, CP858 y WPC1252.

### Caracteres de control prohibidos

Los campos de texto imprimible NO DEBEN contener caracteres de control: C0 (U+0000 a U+001F), DEL (U+007F) ni C1 (U+0080 a U+009F). Hay una sola excepción: `texto.valor` admite el salto de línea `\n`.

Esta regla impide colar comandos ESC/POS dentro de un texto. Quien necesite bytes arbitrarios dispone de la operación sensible `raw`.

### Ancho de una línea

El ancho se mide en **columnas**. Cada carácter, ya convertido a la página de códigos, ocupa una columna. El perfil de la impresora define cuántas columnas caben en una línea con la fuente A; por ejemplo, 48 en una de 80 mm o 32 en una de 58 mm.

| Situación | Columnas útiles |
|---|---|
| `texto` con fuente A y `tamano` `[n, m]` | ⌊C_A / n⌋ |
| `texto` con fuente B y `tamano` `[n, m]` | ⌊C_B / n⌋, donde C_B es el número de columnas de la fuente B según el perfil |
| `fila`, `par` y `separador` | siempre C_A, porque usan la fuente A a tamaño 1×1 |

### Partido de texto largo

Cuando un texto no cabe en el ancho disponible, el puente lo parte así:

1. Primero divide el texto en párrafos por cada `\n` (esto solo ocurre en `texto.valor`).
2. Luego reparte cada párrafo en líneas de forma voraz, palabra por palabra: una palabra se coloca en la línea actual si cabe, y si no, empieza una línea nueva. Las palabras están separadas por espacios.
3. Si una palabra es más larga que la línea completa, se corta en seco en el último carácter que cabe y continúa en la siguiente.
4. Los espacios al inicio y al final de cada línea resultante se eliminan antes de alinearla. Los espacios internos se respetan.
5. Cada línea se alinea por separado según su `alineacion`.

## 4. Operaciones

Cada operación es un objeto con un campo `tipo` y sus atributos. Los atributos opcionales que se omiten toman el valor indicado en la columna «Por defecto».

### 4.1 `texto`

Imprime un texto con estilo.

| Campo | Tipo | Por defecto | Regla |
|---|---|---|---|
| `valor` | cadena | obligatorio | Hasta 4096 caracteres. `\n` fuerza un salto de línea; `""` imprime una línea en blanco. |
| `alineacion` | `izquierda` \| `centro` \| `derecha` | `izquierda` | |
| `negrita` | booleano | `false` | |
| `subrayado` | booleano | `false` | |
| `invertido` | booleano | `false` | Blanco sobre negro. |
| `tamano` | `[ancho, alto]` | `[1, 1]` | Dos enteros, cada uno de 1 a 8. |
| `fuente` | `A` \| `B` | `A` | B es la fuente compacta. |

```json
{"tipo": "texto", "valor": "ABARROTES LA ESPERANZA", "alineacion": "centro", "negrita": true, "tamano": [2, 2]}
```

### 4.2 `separador`

Llena una línea completa repitiendo un carácter.

| Campo | Tipo | Por defecto | Regla |
|---|---|---|---|
| `caracter` | cadena | `"-"` | Exactamente un carácter imprimible. |

```json
{"tipo": "separador", "caracter": "="}
```

### 4.3 `fila`

Coloca varias columnas en el mismo renglón. Si el texto de una celda no cabe en su columna, continúa en renglones adicionales.

| Campo | Tipo | Por defecto | Regla |
|---|---|---|---|
| `columnas` | arreglo | obligatorio | De 1 a 8 columnas. |
| `columnas[].texto` | cadena | obligatorio | Hasta 1024 caracteres, sin `\n`. `""` deja la celda vacía. |
| `columnas[].ancho` | entero \| `"auto"` | `"auto"` | Un porcentaje entero de 1 a 100 del ancho útil, o `"auto"`. |
| `columnas[].alineacion` | `izquierda` \| `centro` \| `derecha` | `izquierda` | |
| `negrita` | booleano | `false` | Se aplica a toda la fila. |

```json
{"tipo": "fila", "columnas": [
  {"texto": "1", "ancho": 12},
  {"texto": "Pan de caja integral grande con ajonjolí", "ancho": "auto"},
  {"texto": "$62.50", "ancho": 25, "alineacion": "derecha"}
]}
```

#### Algoritmo de reparto de columnas

Partimos de estos datos:

- C son las columnas de la línea según el perfil (fuente A).
- n es el número de columnas de la fila.
- Las columnas siempre se separan con **un espacio fijo**.

El puente reparte el ancho así:

1. **Ancho útil:** U = C − (n − 1).
2. **Columnas con porcentaje:** cada una recibe wᵢ = ⌊U × pᵢ / 100⌋.
3. **Sobrante:** R = U − Σwᵢ.
   - Si hay columnas `"auto"`, el sobrante se reparte entre ellas en partes iguales (⌊R / k⌋, donde k es el número de columnas auto). Lo que no se pueda repartir se entrega de a 1 columna, de izquierda a derecha, empezando por la primera columna auto.
   - Si no hay columnas `"auto"`, R se suma a la **última** columna, para que la fila siempre ocupe el ancho completo.
4. **Validación:** si Σpᵢ > 100, o si alguna columna queda con ancho menor que 1, el documento se rechaza con `FILA_NO_CABE`. El esquema no puede detectarlo porque depende de la impresora, pero el puente lo hace **al aceptar** el documento, antes de encolarlo.
5. **Renglones:** el texto de cada celda se parte según su ancho, con las reglas de [§3](#partido-de-texto-largo).
   - La fila ocupa tantos renglones físicos como la celda con más líneas.
   - Las celdas con menos líneas se alinean arriba y se completan con espacios.
   - Cada línea se alinea dentro de su celda.

#### Ejemplo resuelto con C = 48

El ejemplo de arriba tiene 3 columnas:

1. U = 48 − 2 = 46.
2. La columna de 12 % recibe ⌊46 × 12 / 100⌋ = ⌊5.52⌋ = **5**, y la de 25 % recibe ⌊11.5⌋ = **11**.
3. R = 46 − 16 = 30, así que la columna auto recibe **30**.
4. El texto de la columna auto («Pan de caja integral grande con ajonjolí», 40 caracteres) no cabe en 30 y se parte en dos líneas.

```
1     Pan de caja integral grande         $62.50
      con ajonjolí
└─5─┘ └──────────── 30 ────────────┘ └───11────┘
```

### 4.4 `par`

Atajo para el renglón más común de un ticket: una etiqueta a la izquierda y un valor alineado a la derecha.

| Campo | Tipo | Por defecto | Regla |
|---|---|---|---|
| `izquierda` | cadena | obligatorio | Hasta 1024 caracteres, sin `\n`. |
| `derecha` | cadena | obligatorio | Hasta 1024 caracteres, sin `\n`. |
| `negrita` | booleano | `false` | |

```json
{"tipo": "par", "izquierda": "TOTAL", "derecha": "$163.50", "negrita": true}
```

Un `par` equivale a una `fila` de 2 columnas cuyos anchos se calculan así:

- U = C − 1.
- La columna derecha mide d = min(largo(`derecha`), ⌊U / 2⌋) y su texto va alineado a la derecha.
- La columna izquierda mide U − d y su texto va alineado a la izquierda.

Con C = 48, `"TOTAL"` / `"$163.50"` da una columna izquierda de 40 y una derecha de 7. Si alguno de los textos es más largo que su columna, se parte en varias líneas como en `fila`.

### 4.5 `salto`

Avanza el papel dejando líneas en blanco.

| Campo | Tipo | Por defecto | Regla |
|---|---|---|---|
| `lineas` | entero | `1` | De 1 a 10. |

```json
{"tipo": "salto", "lineas": 3}
```

### 4.6 `qr`

Imprime un código QR (modelo 2).

| Campo | Tipo | Por defecto | Regla |
|---|---|---|---|
| `valor` | cadena | obligatorio | De 1 a 700 caracteres, de cualquier tipo, incluidos saltos de línea. |
| `tamano` | entero | `6` | Tamaño del módulo en puntos, de 1 a 16. |
| `correccion` | `L` \| `M` \| `Q` \| `H` | `M` | Nivel de corrección de errores. |
| `alineacion` | `izquierda` \| `centro` \| `derecha` | `centro` | |

Si el código resultante es más ancho que el papel, el puente reduce `tamano` hasta que quepa y anota la advertencia `QR_REDUCIDO`. Si ni con tamaño 1 cabe, rechaza el documento con `OPERACION_NO_SOPORTADA`.

```json
{"tipo": "qr", "valor": "https://facturas.example.com/t/FAC-A-000981", "tamano": 6, "correccion": "M"}
```

### 4.7 `codigo_barras`

Imprime un código de barras unidimensional.

| Campo | Tipo | Por defecto | Regla |
|---|---|---|---|
| `valor` | cadena | obligatorio | Depende de la simbología (ver la tabla siguiente). |
| `simbologia` | `CODE128` \| `CODE39` \| `EAN13` \| `EAN8` \| `UPCA` \| `ITF` | obligatorio | |
| `texto` | `arriba` \| `abajo` \| `ninguno` | `abajo` | Posición del texto legible. |
| `alto` | entero | `80` | Alto en puntos, de 1 a 255. |
| `alineacion` | `izquierda` \| `centro` \| `derecha` | `centro` | |

Valores aceptados por simbología:

| Simbología | `valor` | Dígito verificador |
|---|---|---|
| `EAN13` | 12 o 13 dígitos | Con 12 dígitos lo calcula el puente. Con 13, el puente lo verifica. |
| `EAN8` | 7 u 8 dígitos | Igual que en EAN13. |
| `UPCA` | 11 o 12 dígitos | Igual que en EAN13. |
| `ITF` | De 2 a 80 dígitos, siempre en número par | No se calcula. |
| `CODE39` | De 1 a 40 caracteres de `0-9 A-Z espacio - . $ / + %` | No se calcula. |
| `CODE128` | De 1 a 80 caracteres ASCII imprimibles (U+0020 a U+007E) | Lo calcula siempre el puente. |

Si un dígito verificador no coincide, el puente rechaza el documento con `CODIGO_BARRAS_INVALIDO`.

El puente imprime cada módulo (la barra más delgada) con 3 puntos de ancho, o con 2 si con 3 no cabe en el papel. Si ni con 2 cabe, responde `OPERACION_NO_SOPORTADA`. Esto puede pasar, por ejemplo, con un `CODE128` largo en papel de 58 mm.

```json
{"tipo": "codigo_barras", "valor": "4006381333931", "simbologia": "EAN13", "alto": 80}
```

### 4.8 `imagen`

Imprime una imagen. El documento DEBE traer **exactamente uno** de los campos `base64` o `recurso`. No existe un campo para URLs: el puente nunca descarga contenido externo.

| Campo | Tipo | Por defecto | Regla |
|---|---|---|---|
| `base64` | cadena | — | Un PNG en base64 estándar, sin el prefijo `data:`. Decodificado no puede superar los 100 KB. Por eso el esquema exige que empiece con `iVBORw0KGgo`, que es la firma PNG, y que mida como máximo 136 536 caracteres. |
| `recurso` | cadena | — | El nombre de una imagen guardada en el panel del puente: `^[a-z0-9][a-z0-9_-]{0,63}$`. |
| `alineacion` | `izquierda` \| `centro` \| `derecha` | `centro` | |
| `ancho_max` | entero | `100` | Ancho máximo como porcentaje del área imprimible, de 1 a 100. |

El puente procesa la imagen así:

1. Decodifica el PNG. Si no es un PNG válido o mide más de 4096 píxeles por lado, responde `IMAGEN_INVALIDA`.
2. Trata los píxeles transparentes como blancos.
3. Reduce la imagen, conservando la proporción, hasta que su ancho no supere `ancho_max` del área imprimible. **Nunca la agranda.**
4. La convierte a 1 bit con difusión de error Floyd–Steinberg.

Si el documento menciona un `recurso` que no existe en el puente, la respuesta es `RECURSO_NO_ENCONTRADO`.

> Para un logo que se repite en cada ticket, usa `recurso`: el documento queda más ligero y la imagen se procesa una sola vez.

```json
{"tipo": "imagen", "recurso": "logo-tienda", "ancho_max": 60}
```

### 4.9 `cortar`

Avanza el papel hasta la cuchilla y corta.

| Campo | Tipo | Por defecto | Regla |
|---|---|---|---|
| `parcial` | booleano | `false` | Con `true`, el corte deja una pestaña que une el ticket al rollo. |

```json
{"tipo": "cortar"}
```

### 4.10 `abrir_cajon` (sensible)

Envía el pulso que abre el cajón de dinero.

| Campo | Tipo | Por defecto | Regla |
|---|---|---|---|
| `pin` | `2` \| `5` | `2` | Conector del cajón. |

```json
{"tipo": "abrir_cajon", "pin": 2}
```

### 4.11 `raw` (sensible)

Envía bytes a la impresora tal cual, sin ninguna transformación. Úsalo solo para funciones del modelo que el contrato no cubre.

| Campo | Tipo | Por defecto | Regla |
|---|---|---|---|
| `base64` | cadena | obligatorio | Base64 estándar. Decodificado no puede superar los 64 KB. |

```json
{"tipo": "raw", "base64": "G3AAGfo="}
```

### Operaciones sensibles

`abrir_cajon` y `raw` están **deshabilitadas por defecto** para cada cliente. El dueño del puente las habilita, una por una, en el panel de administración.

Si un cliente envía una operación sensible que no tiene permitida, el puente rechaza **el documento completo**, sin imprimir nada. La respuesta es 403 `OPERACION_NO_PERMITIDA` (ver [api-v1.md](api-v1.md)).

## 5. Límites

| Límite | Valor | Lo verifica |
|---|---|---|
| Tamaño del cuerpo HTTP | 256 KB (262 144 bytes) | el puente (`DOCUMENTO_DEMASIADO_GRANDE`) |
| Operaciones por documento | 500 | el esquema (`DEMASIADAS_OPERACIONES`) |
| Copias | 1 a 5 | el esquema |
| `texto.valor` | 4096 caracteres | el esquema |
| Texto de una celda (`fila`, `par`) | 1024 caracteres | el esquema |
| Columnas por `fila` | 8 | el esquema |
| `qr.valor` | 700 caracteres | el esquema |
| `imagen.base64` | 100 KB decodificado, 4096 px por lado | el esquema (longitud) y el puente (contenido) |
| `raw.base64` | 64 KB decodificado | el esquema y el puente |
| Vigencia de la firma | `expira` − `emitido` ≤ 120 s | el puente |
| Ventana de idempotencia | 10 min | el puente |

En el esquema, «caracteres» significa puntos de código Unicode.

## 6. Capacidades de la impresora

El perfil de cada impresora declara qué puede hacer. Cuando una operación no se puede ejecutar de forma nativa, el puente actúa así:

| Operación | Si la impresora no la soporta |
|---|---|
| `cortar` | Avanza 4 líneas y anota una advertencia en el resultado del trabajo. |
| `qr`, `codigo_barras` | Dibuja el código como imagen (raster). Requiere que la impresora soporte imágenes. |
| `imagen` (o un código que haya que dibujar) sin soporte de imágenes | Rechaza el documento con `OPERACION_NO_SOPORTADA`. |
| `abrir_cajon` sin cajón | Omite el pulso, imprime el resto y anota la advertencia `SIN_CAJON`. |

Los rechazos se deciden al aceptar el documento, de modo que **nunca se imprime un ticket a medias** por falta de una capacidad. `cortar` y `abrir_cajon` no rechazan el documento: el ticket se imprime igual y la advertencia queda en el trabajo (`SIN_CUCHILLA`, `SIN_CAJON`).

## 7. Firma (nivel 3)

En el nivel 3, el backend del cliente firma cada documento con Ed25519. El puente solo guarda la **llave pública** del cliente (32 bytes). Este nivel protege contra documentos alterados, repetidos o falsificados.

### Cómo firmar

1. Arma el documento completo, incluidos `cliente_id`, `emitido` (la hora actual) y `expira` (como máximo `emitido` + 120 s), pero **sin** el campo `firma`.
2. Obtén su forma canónica JCS (RFC 8785):
   - claves ordenadas por unidades de código UTF-16;
   - sin espacios;
   - cadenas con el escape mínimo de JSON;
   - caracteres no ASCII en UTF-8 directo, sin `\u`.
3. Firma los bytes UTF-8 de esa forma canónica con Ed25519 (RFC 8032; sin prehash ni contexto).
4. Codifica los 64 bytes de la firma en base64 estándar, con relleno `=`, y agrégala como `firma`.

La firma cubre **el JSON tal como se envía**, no el documento con los valores por defecto aplicados. Un campo omitido y ese mismo campo con su valor por defecto producen firmas distintas, y ambas son válidas.

### Qué comprueba el puente

| Comprobación | Error si falla |
|---|---|
| El cliente es de nivel 3 y el documento trae `firma`. | `FIRMA_REQUERIDA` |
| `cliente_id` coincide con el cliente autenticado. | `FIRMA_INVALIDA` |
| La firma verifica con la llave pública registrada. | `FIRMA_INVALIDA` |
| 0 < `expira` − `emitido` ≤ 120 s. | `FIRMA_INVALIDA` |
| `emitido` ≤ ahora + 30 s (tolerancia de reloj). | `FIRMA_INVALIDA` |
| `expira` > ahora − 30 s. | `FIRMA_EXPIRADA` |
| El `id` no se usó en los últimos 10 min. | Ver [§8](#8-idempotencia) |

La ventana de idempotencia (10 min) es más larga que la vida máxima de un documento firmado (120 s más la tolerancia de reloj). Por eso un documento firmado no se puede repetir mientras siga vigente.

Para los clientes de nivel 1 y 2, los cuatro campos de firma son opcionales. Si se envían, el puente respeta `expira` (`FIRMA_EXPIRADA`), pero no verifica la firma.

### Ejemplo resuelto

> ⚠️ Esta llave es el vector de prueba 1 de RFC 8032 §7.1. Es **pública**: no la uses fuera de pruebas.

| | |
|---|---|
| Semilla privada (hex) | `9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60` |
| Llave pública (hex) | `d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a` |
| Llave pública (base64), tal como se registra en el panel | `11qYAYKxCrfVS/7TyWQHOg7hcvPapiMlrwIaaPcHURo=` |

Documento enviado ([`ejemplos/validos/firmado.json`](../../ejemplos/validos/firmado.json)):

```json
{
  "version": "1",
  "id": "780bf0ea-9a33-4add-86f3-9f4f23000fc4",
  "impresora": "caja",
  "referencia": "PED-000812",
  "operaciones": [
    {"tipo": "texto", "valor": "Pedido en línea #812", "negrita": true},
    {"tipo": "par", "izquierda": "TOTAL", "derecha": "$249.00"},
    {"tipo": "cortar"}
  ],
  "cliente_id": "cli_7Hq2MZp4XwR9",
  "emitido": "2026-10-14T18:30:00Z",
  "expira": "2026-10-14T18:31:30Z",
  "firma": "PN9IWA7gHPPA7ec2VFaqhtZ9BzBh1nZcspFD/nR3w29PM8Q4tAKJMoNilt0+eF2YU4005EPzRGHAmwY+FP6KBQ=="
}
```

Forma canónica que se firma (una sola línea, en UTF-8):

```
{"cliente_id":"cli_7Hq2MZp4XwR9","emitido":"2026-10-14T18:30:00Z","expira":"2026-10-14T18:31:30Z","id":"780bf0ea-9a33-4add-86f3-9f4f23000fc4","impresora":"caja","operaciones":[{"negrita":true,"tipo":"texto","valor":"Pedido en línea #812"},{"derecha":"$249.00","izquierda":"TOTAL","tipo":"par"},{"tipo":"cortar"}],"referencia":"PED-000812","version":"1"}
```

Ed25519 es determinista: con la misma llave y la misma forma canónica, siempre se obtiene la firma de arriba. Este ejemplo sirve para probar tu implementación. Si lo envías a un puente real, responderá `FIRMA_EXPIRADA`, porque sus fechas ya pasaron.

## 8. Idempotencia

El cliente genera un `id` (UUID v4) nuevo para cada ticket y lo **reutiliza en los reintentos** de ese mismo ticket. El puente recuerda cada pareja (cliente, `id`) durante 10 minutos:

| Situación | Respuesta |
|---|---|
| `id` nuevo | Acepta el documento y lo encola. |
| Mismo `id` y mismo contenido (hash SHA-256 de la forma canónica JCS sin `firma`) | 200 con el trabajo original y `"duplicado": true`. **No se vuelve a imprimir.** |
| Mismo `id` con otro contenido | 409 `ID_REUTILIZADO`. |

Gracias a esto puedes reintentar sin miedo después de un error de red: el ticket no saldrá dos veces.

- **Reintento después de un fallo:** si el trabajo original **falló** (impresora apagada, sin papel…), el puente libera el `id`. Reintentar el mismo documento vuelve a intentar imprimirlo, no se toma como repetido.
- **Nivel 3:** reenviar un documento firmado que sigue vigente produce la respuesta de duplicado, así que una firma capturada no sirve para imprimir dos veces (anti-repetición).

## 9. Validación y errores

### Orden de validación

El puente valida en este orden y se detiene en el primer paso que falla. Nada costoso (esquema completo, imágenes, render) ocurre antes de saber quién envía el documento y si puede imprimir:

1. **Autorización de la petición:** origen y llave de cliente (ver [api-v1.md](api-v1.md)).
2. **Tamaño del cuerpo:** `DOCUMENTO_DEMASIADO_GRANDE`.
3. **JSON válido** (I-JSON, sin claves duplicadas): `JSON_INVALIDO`.
4. **`version`:** `VERSION_NO_SOPORTADA`. Se comprueba antes que el resto, para que un documento de otra versión reciba este código y no un error genérico.
5. **Firma**, en el nivel 3: `FIRMA_REQUERIDA`, `FIRMA_INVALIDA` o `FIRMA_EXPIRADA`. En todos los niveles se respeta `expira` si viene.
6. **Idempotencia:** `ID_REUTILIZADO`, o la respuesta de duplicado.
7. **Permisos del cliente** sobre la impresora y las operaciones sensibles: `IMPRESORA_NO_PERMITIDA`, `OPERACION_NO_PERMITIDA`.
8. **Esquema:** `OPERACION_DESCONOCIDA`, `DEMASIADAS_OPERACIONES` o `DOCUMENTO_INVALIDO`.
9. **Reglas del documento que no dependen de la impresora:**
   - `CODIGO_BARRAS_INVALIDO`: dígito verificador incorrecto.
   - `IMAGEN_INVALIDA`: un PNG en línea dañado o demasiado grande.
   - `FILA_NO_CABE`: porcentajes que suman más de 100.
   - `FIRMA_INVALIDA`: la vigencia de la firma no está entre 0 y 120 s.
10. **Perfil de la impresora:** `IMPRESORA_NO_ENCONTRADA`, `FILA_NO_CABE`, `IMAGEN_INVALIDA`, `RECURSO_NO_ENCONTRADO` u `OPERACION_NO_SOPORTADA`.

Para los pasos 5 a 7, el puente lee del documento solo `id`, `impresora`, los campos de firma y el `tipo` de cada operación. La validación completa llega en el paso 8.

### Forma del error

Todas las respuestas llevan `ok`. Los errores agregan su código, mensaje y sugerencia en el mismo nivel:

```json
{
  "ok": false,
  "codigo": "DOCUMENTO_INVALIDO",
  "mensaje": "El documento no cumple el contrato v1.",
  "sugerencia": "Revisa los detalles; cada uno indica el campo y la regla que falló.",
  "detalles": [
    {
      "ruta": "/operaciones/3/alineacion",
      "regla": "enum",
      "mensaje": "debe ser uno de: izquierda, centro, derecha"
    }
  ]
}
```

- `codigo` es estable. Programa contra él, no contra `mensaje`, que puede cambiar o traducirse.
- `detalles` solo aparece en los errores del documento.
- `ruta` es un JSON Pointer (RFC 6901) que apunta al campo del documento.
- Si el error ocurrió al imprimir, la respuesta incluye además `trabajo_id`.

### Códigos del contrato

| Código | HTTP | Cuándo |
|---|---|---|
| `DOCUMENTO_DEMASIADO_GRANDE` | 413 | El cuerpo supera los 256 KB. |
| `JSON_INVALIDO` | 400 | El cuerpo no es JSON, no es I-JSON o tiene claves duplicadas. |
| `VERSION_NO_SOPORTADA` | 400 | `version` falta, no es una cadena o el puente no la soporta. Incluye `versiones_soportadas`. |
| `OPERACION_DESCONOCIDA` | 400 | Un `tipo` no existe en esta versión. |
| `DEMASIADAS_OPERACIONES` | 400 | Hay más de 500 operaciones. |
| `DOCUMENTO_INVALIDO` | 400 | Cualquier otra violación del esquema. |
| `CODIGO_BARRAS_INVALIDO` | 400 | El dígito verificador no coincide. |
| `IMAGEN_INVALIDA` | 400 | `base64` no es un PNG decodificable o supera los 100 KB. |
| `FIRMA_REQUERIDA` | 401 | Un cliente de nivel 3 envió un documento sin firma. |
| `FIRMA_INVALIDA` | 401 | La firma no verifica, `cliente_id` no coincide o la vigencia es incorrecta. |
| `FIRMA_EXPIRADA` | 401 | `expira` ya pasó. |
| `RECURSO_NO_ENCONTRADO` | 404 | La imagen guardada no existe en el puente. |
| `ID_REUTILIZADO` | 409 | El mismo `id` llegó con otro contenido en los últimos 10 min. |
| `FILA_NO_CABE` | 422 | Las columnas de una `fila` no caben en el ancho de la impresora. |
| `OPERACION_NO_SOPORTADA` | 422 | La impresora no puede ejecutar una operación (ver [§6](#6-capacidades-de-la-impresora)). |

Los errores de autenticación, permisos, orígenes e impresoras inexistentes no forman parte del contrato del documento. Están descritos en [api-v1.md](api-v1.md).

## 10. Política de versiones

### Qué indica `version`

- `version` indica la versión **del contrato del documento**, en la forma `"MAYOR"` o `"MAYOR.MENOR"`. Hoy solo existe `"1"`.
- Es independiente del prefijo `/v1/` de la API HTTP: una API `/v1/` puede llegar a aceptar documentos `"2"`.

### Cómo evoluciona el contrato

| Cambio | Nueva versión | Ejemplos |
|---|---|---|
| Compatible | Se publica la siguiente versión menor: `"1.1"`, `"1.2"`, etc. | Agregar una operación nueva o un campo opcional nuevo. |
| Incompatible | Se publica la siguiente versión mayor: `"2"`. | Quitar o renombrar un campo, o cambiar un valor por defecto o un algoritmo de reparto. |

- Los documentos `"1"` válidos hoy lo seguirán siendo, y se imprimirán igual, en cualquier puente `1.x`.
- Un documento DEBE declarar la versión **mínima** que necesita. Si solo usa lo de la v1, envía `"1"` aunque el puente soporte la 1.3, para que el mismo documento funcione en puentes más antiguos.

### Qué hace un puente con cada versión

- Un puente acepta cualquier versión menor o igual a la suya **dentro del mismo mayor**. Por ejemplo, un puente 1.2 acepta `"1"`, `"1.1"` y `"1.2"`.
- Un puente que recibe una versión que no soporta, como `"2"` o una `"1.3"` en un puente 1.2, responde:

  ```json
  {
    "ok": false,
    "codigo": "VERSION_NO_SOPORTADA",
    "mensaje": "Este puente no soporta documentos versión 2.",
    "sugerencia": "Actualiza PRISMA en el equipo del negocio o envía el documento en una versión soportada.",
    "versiones_soportadas": ["1"]
  }
  ```

- El puente nunca intenta interpretar un documento de una versión que no conoce, ni siquiera parcialmente.

### Negociación y soporte a largo plazo

- **Negociación** (⏳ política futura, **no implementada en 0.1.0**): cuando exista una versión menor (`"1.1"`), el puente anunciará sus `versiones_documento` en una ruta de la API y el SDK generará la más alta que entiendan los dos, sin pasar de la mínima que necesita el documento. Hoy solo existe la versión `"1"` (`contrato.VersionesSoportadas`), `GET /v1/salud` no anuncia versiones y el SDK siempre envía `"1"`.
- **Convivencia:** un puente v2 seguirá aceptando documentos `"1"` durante, al menos, todo el ciclo de vida de la v2. Así, las integraciones existentes no se rompen al actualizar el puente.
- **Esquemas publicados:** cada versión tiene su propio esquema inmutable (`documento-v1.schema.json`, `documento-v1.1.schema.json`, …).

## 11. Ejemplos

### Válidos

| Archivo | Qué muestra |
|---|---|
| [`venta-simple.json`](../../ejemplos/validos/venta-simple.json) | Ticket de venta con filas, pares y corte. |
| [`venta-logo-qr.json`](../../ejemplos/validos/venta-logo-qr.json) | Logo PNG en línea, QR para facturar y 2 copias. |
| [`comanda-cocina.json`](../../ejemplos/validos/comanda-cocina.json) | Comanda para cocina: texto grande, invertido y notas. |
| [`todas-las-operaciones.json`](../../ejemplos/validos/todas-las-operaciones.json) | Cada operación y cada atributo al menos una vez. |
| [`con-raw.json`](../../ejemplos/validos/con-raw.json) | Operaciones sensibles: `raw` y `abrir_cajon`. |
| [`firmado.json`](../../ejemplos/validos/firmado.json) | Documento de nivel 3 con firma real (ver [§7](#7-firma-nivel-3)). |

### Inválidos

[`ejemplos/invalidos/`](../../ejemplos/invalidos/) contiene un archivo por regla, nombrado según la regla que viola. Para cada archivo, [`indice.json`](../../ejemplos/invalidos/indice.json) indica:

- `codigo`: el código de error esperado;
- `detecta`: quién lo detecta (`esquema` si basta el JSON Schema, `puente` si hace falta el puente);
- `ruta` y `regla`: el JSON Pointer y la palabra clave del esquema (o la regla del puente) que falla.

Todos los archivos marcados `puente` pasan el esquema: muestran justamente lo que un validador de JSON Schema no puede ver.
