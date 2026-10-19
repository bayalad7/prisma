# Primeros pasos: PRISMA en 5 minutos

Esta guía compila PRISMA y lo usa para imprimir un ticket en una **impresora virtual**. No hace falta hardware. Los comandos sirven en Windows (PowerShell), Linux y macOS. Donde cambian, se muestran ambas versiones.

## 1. Compilar

Necesitas Go 1.26 o superior y Git.

```sh
git clone https://github.com/bayalad7/prisma.git
cd prisma
git switch develop
```

En Windows:

```powershell
.\build.ps1 build        # deja el binario en dist\prisma.exe
```

En Linux o macOS:

```sh
make build               # deja el binario en dist/prisma
```

Comprueba que funciona:

```sh
./dist/prisma version
```

## 2. Crear una configuración de desarrollo

Para no tocar las rutas del sistema, usa una configuración propia con una carpeta de datos local. Crea `config.dev.toml` en la raíz del repositorio. Este archivo y la carpeta `datos-dev/` están en `.gitignore`.

En Windows:

```toml
[servidor]
escuchar = "127.0.0.1"

[datos]
ruta = 'C:\ruta\al\repo\prisma\datos-dev'   # absoluta, con comillas simples

[dev]
modo_desarrollo = true
```

En Linux o macOS:

```toml
[servidor]
escuchar = "127.0.0.1"

[datos]
ruta = "/home/tu-usuario/prisma/datos-dev"

[dev]
modo_desarrollo = true
```

Valídala:

```sh
./dist/prisma config validar --config config.dev.toml
```

## 3. Ver un ticket sin imprimir

```sh
./dist/prisma dev previsualizar ejemplos/validos/venta-simple.json
./dist/prisma dev previsualizar ejemplos/validos/venta-logo-qr.json --ancho 58
./dist/prisma dev previsualizar ejemplos/validos/venta-logo-qr.json --html vista.html   # ábrelo en el navegador
```

## 4. Imprimir en la impresora virtual

Abre **dos terminales**.

**Terminal 1** — la impresora virtual escucha en el puerto 9100 y muestra cada ticket que recibe:

```sh
./dist/prisma dev impresora-virtual --puerto 9100 --ancho 80
```

**Terminal 2** — registra la impresora en el puente e imprime:

```sh
./dist/prisma impresoras agregar --config config.dev.toml --alias caja --tipo red --destino 127.0.0.1:9100 --cajon
./dist/prisma impresoras probar caja --config config.dev.toml
./dist/prisma imprimir --config config.dev.toml --impresora caja ejemplos/validos/venta-simple.json
```

En la terminal 1 aparecen la traza de los comandos ESC/POS y la previsualización del ticket. Cada trabajo también se guarda como `.bin`.

## 5. Imprimir en una impresora real

1. Busca la impresora: `./dist/prisma impresoras detectar` (o `--red` para buscar en la red local).
2. Agrégala con el `TIPO` y el `DESTINO` que aparecen en la lista. Por ejemplo:

   ```sh
   ./dist/prisma impresoras agregar --config config.dev.toml --alias termica --tipo usb --destino "POS-80"
   ./dist/prisma impresoras probar termica --config config.dev.toml
   ./dist/prisma imprimir --config config.dev.toml --impresora termica ejemplos/validos/venta-simple.json
   ```

3. Si algo falla, el error trae un código y una sugerencia. Los detalles de cada tipo de conexión están en [conectores.md](conectores.md).

Si los acentos salen mal o las columnas no cuadran, ajusta `--pagina`, `--comando-pagina` o `--columnas`. Ver [solucion-de-problemas.md](solucion-de-problemas.md).

## 6. Imprimir por la API, como una aplicación

Con la impresora virtual de la sección 4 todavía abierta:

**Terminal 2** — registra una aplicación de nivel 2 y anota la llave `pk_…` que muestra el comando:

```sh
./dist/prisma clientes agregar --config config.dev.toml --nombre "Prueba" --nivel 2 --origen https://localhost
```

**Terminal 3** — arranca el puente. Escucha en `https://127.0.0.1:8443` y la primera vez genera su certificado autofirmado:

```sh
./dist/prisma ejecutar --config config.dev.toml
```

**Terminal 2** — imprime con curl. `-k` acepta el certificado autofirmado:

```sh
curl -k https://127.0.0.1:8443/v1/salud
curl -k https://127.0.0.1:8443/v1/imprimir -H "Authorization: Bearer pk_…" --data-binary @ejemplos/validos/venta-simple.json
```

Si repites el mismo comando, el puente responde `"duplicado": true` y no imprime de nuevo, porque el documento tiene el mismo `id`.

Para el nivel 3 (documentos firmados) y para llamar desde un navegador, sigue los ejemplos de [api-v1.md](api-v1.md#ejemplos). Para que el navegador confíe en el certificado, ver [tls.md](tls.md).

## 7. Configurar todo desde el panel

Lo mismo que hiciste con la CLI se puede hacer desde el navegador. Con el puente de la sección 6 en marcha (`prisma ejecutar` debe seguir abierto; Ctrl+C lo detiene):

1. Abre `https://localhost:8443/admin` en el navegador de **ese mismo equipo**.
2. El navegador avisa que el certificado no es de confianza, porque es autofirmado. Elige *Configuración avanzada → Continuar a localhost*. Para que no vuelva a avisar, instálalo como explica [tls.md](tls.md).
3. La primera vez el panel pide crear la contraseña de administrador (al menos 10 caracteres). Desde el propio equipo no hace falta ningún código.
4. Prueba el recorrido completo:
   - En *Impresoras → Agregar impresora → Escribir a mano*, usa el tipo **Red** con destino `127.0.0.1:9100` (la impresora virtual de la sección 4) o el tipo **Archivo** con una carpeta.
   - Pulsa **Ticket de diagnóstico**. La regla de columnas debe ocupar una sola línea.
   - En *Aplicaciones*, crea una de nivel 2 y copia la llave `pk_…`, que se muestra una sola vez.

Con `escuchar = "127.0.0.1"`, como en `config.dev.toml`, el panel solo se abre desde este equipo. Para entrar desde un celular u otra computadora de la red:

1. Cambia a `escuchar = "0.0.0.0"` y reinicia el puente.
2. Si Windows lo pregunta, permite el acceso en el Firewall.
3. Abre `https://<ip-de-este-equipo>:8443/admin`.
4. Si el panel aún no tiene contraseña, pedirá un código. Genéralo en este equipo con `./dist/prisma admin codigo-inicial --config config.dev.toml`.

Si olvidas la contraseña: `./dist/prisma admin restablecer-contrasena --config config.dev.toml`. Todo lo demás del panel está en [panel.md](panel.md).

## Siguientes pasos

- Escribe tus propios documentos con el [contrato](contrato-documento-v1.md). La carpeta [ejemplos/validos](../ejemplos/validos/) tiene un ejemplo de cada caso.
- Para contribuir, lee [desarrollo.md](desarrollo.md).
- Para instalar en un negocio, lee [instalacion.md](instalacion.md).
