# Solución de problemas

Cuando un comando falla, PRISMA muestra un código y una **sugerencia**. Busca el código en esta página o en [conectores.md](conectores.md#errores-de-conexión).

## Instalación y arranque

### «la configuración tiene N problema(s)»

`config.toml` tiene claves mal escritas o valores inválidos. El mensaje lista cada clave con su problema. Corrígelas y comprueba el resultado con `prisma config validar`. La referencia completa está en [configuracion.md](configuracion.md).

### «datos.ruta: debe ser una ruta absoluta»

Usa una ruta completa. En Windows, escríbela entre comillas simples: `ruta = 'D:\PRISMA\datos'`.

### «la base de datos usa el esquema N, pero esta versión de PRISMA solo conoce hasta el M»

Una versión más nueva de PRISMA ya migró la base. Instala esa versión o una posterior, o restaura el respaldo hecho antes de actualizar.

### El servicio no puede abrir `prisma.db` (permiso denegado)

La base la creó otro usuario, normalmente root al ejecutar un comando con `sudo`. Hay que devolvérsela al usuario del servicio:

```sh
sudo chown -R prisma:prisma /var/lib/prisma
```

A partir de entonces, ejecuta los comandos de administración como ese usuario: `sudo -u prisma prisma …`.

### Windows bloquea el ejecutable (SmartScreen) o macOS dice que no se puede abrir

El binario no está firmado:

- **Windows:** en las *Propiedades* del archivo, marca **Desbloquear**.
- **macOS:** ejecuta `sudo xattr -d com.apple.quarantine /usr/local/bin/prisma`.

### «prisma imprimir solo está disponible con [dev] modo_desarrollo = true»

Es intencional: en producción, las aplicaciones imprimen por la API con su autorización. Para pruebas, usa una configuración de desarrollo aparte (`--config config.dev.toml`).

## La impresora no imprime

| Código | Causa típica | Qué hacer |
|---|---|---|
| `IMPRESORA_NO_ENCONTRADA` | El alias no existe en el puente, o el destino (nombre, cola, puerto) no existe en el sistema. | `prisma impresoras listar` y `prisma impresoras detectar`; el destino debe coincidir exactamente. |
| `IMPRESORA_INALCANZABLE` | Está apagada, desconectada, en otra red o marcada «sin conexión» en Windows. | Revisa encendido, cable o red, y su IP. Ver [conectores.md](conectores.md). |
| `TIEMPO_AGOTADO` | Dejó de aceptar datos: sin papel, tapa abierta, atasco o Bluetooth fuera de alcance. | Revisa el papel y la tapa; acércate si es Bluetooth. |
| `PUERTO_OCUPADO` | Otro programa tiene abierto el puerto serie o el dispositivo USB. | Cierra el otro programa; en Linux, usa la cola de CUPS. |
| `PERMISO_DENEGADO` | El usuario del servicio no tiene acceso. | Linux: grupos `lp` y `dialout`. Windows: permiso de impresión. |
| `COLA_LLENA` | Hay 20 trabajos pendientes en esa impresora. | La impresora no avanza: revisa el estado con `prisma impresoras probar`. |

### Windows dice que imprimió, pero no salió nada

En USB, Windows acepta el trabajo en su cola de impresión aunque la impresora esté apagada. Abre la cola en *Configuración > Impresoras* y revisa si hay trabajos atascados o si está «sin conexión». `prisma impresoras probar` también lo detecta, si el controlador lo informa.

### Sale texto raro o símbolos en lugar del ticket

La impresora está interpretando los bytes con un controlador que no es RAW, o no habla ESC/POS:

- **Windows:** usa el controlador del fabricante o «Generic / Text Only».
- **Linux y macOS:** crea la cola con `-m raw`.
- **Impresoras de etiquetas:** las que usan ZPL o TSPL no son ESC/POS y no son compatibles.

## El ticket sale, pero no como se espera

### Acentos o ñ salen mal

La página de códigos del perfil no coincide con la que tiene activa la impresora.

1. Imprime la hoja de autoprueba (enciende la impresora con FEED presionado) y busca su tabla de páginas de códigos.
2. Ajusta `--pagina` y `--comando-pagina`, el `n` de `ESC t n` en tu modelo. En Epson y la mayoría de compatibles:

| Página | `--pagina` | `--comando-pagina` |
|---|---|---|
| PC437 (EE. UU.) | `CP437` | 0 |
| PC850 (Multilingüe) | `CP850` | 2 |
| WPC1252 | `WPC1252` | 16 |
| PC858 (con €) | `CP858` | 19 |

En muchos clones chinos los números cambian; por ejemplo, CP858 puede ser 13. Prueba con `ejemplos/validos/todas-las-operaciones.json` hasta que salgan bien `á é í ó ú ñ ¿ ¡ €`.

Si en las advertencias del trabajo aparece **`CARACTER_SUSTITUIDO`**, el carácter no existe en esa página y se imprimió sin acento o como `?`. Elige una página que lo tenga: CP858 o WPC1252 cubren el español y el €.

### Las columnas no cuadran o el texto se parte antes de tiempo

El número de columnas del perfil no es el de la impresora. Muchas de 80 mm usan 48 columnas con la fuente A, pero otras usan 42. Corrígelo así:

```sh
prisma impresoras eliminar caja
prisma impresoras agregar --alias caja … --columnas 42 --columnas-b 56
```

### No corta o no abre el cajón

- **Sin cuchilla:** registra la impresora con `--sin-corte`. El corte se reemplaza por un avance de papel y queda la advertencia `SIN_CUCHILLA`.
- **No abre el cajón:**
  - registra la impresora con `--cajon`;
  - revisa el pin: la operación usa `pin: 2` por defecto, y algunos cajones usan el 5;
  - ⏳ el cliente debe tener habilitada la operación `abrir_cajon`.

### El QR o el código de barras no aparece

La impresora no tiene QR o códigos de barras por comando, algo común en modelos económicos. Regístrala con `--sin-qr-nativo` o `--sin-barras-nativo`: PRISMA los enviará como imagen.

Si el código de barras no cabe en el papel, el documento se rechaza con `OPERACION_NO_SOPORTADA`. Acorta el valor o usa una simbología más compacta.

### Errores del documento

`FILA_NO_CABE`, `DOCUMENTO_INVALIDO`, `CODIGO_BARRAS_INVALIDO`, etc. son problemas del JSON, no de la impresora. Cada error indica el campo (`/operaciones/3/columnas`) y la regla que falló. Las reglas están en el [contrato](contrato-documento-v1.md), y [ejemplos/invalidos](../ejemplos/invalidos/) tiene un ejemplo de cada error.

Para ver el resultado antes de imprimir:

```sh
prisma dev previsualizar documento.json --ancho 58
```

## Diagnóstico avanzado

- **Ver los bytes exactos:** `prisma dev renderizar doc.json -o t.bin` y luego `prisma dev decodificar t.bin`.
- **Imprimir sin hardware:** usa `prisma dev impresora-virtual`, o una impresora `--tipo archivo`, que guarda el `.bin` y un `.txt` con la vista.
- **Bitácoras:**
  - Linux: `journalctl -u prisma -f`.
  - Windows: `C:\ProgramData\PRISMA\logs\prisma.log`.
  - macOS: `/Library/Logs/PRISMA/prisma.log`.

  Son JSON, una línea por evento, y nunca incluyen el contenido de los tickets.
