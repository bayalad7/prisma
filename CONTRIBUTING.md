# Contribuir a PRISMA

Gracias por tu interés. PRISMA imprime en negocios reales, así que cada cambio debe compilar, pasar las pruebas en Windows, Linux y macOS, y no romper a los integradores.

## Antes de empezar

- **Reglas del proyecto:** [CLAUDE.md](CLAUDE.md) lista las decisiones técnicas que no se cambian sin acordarlo antes, las invariantes y las convenciones. Las decisiones están explicadas en [docs/adr/](docs/adr/README.md).
- **Entorno:** requisitos, tareas y herramientas en [docs/desarrollo/](docs/desarrollo/README.md).
- **Cambios grandes:** abre primero un issue para discutirlo. Cambios en el contrato del documento, la API pública, el modelo de seguridad o la base de datos necesitan acuerdo antes del código.
- **Vulnerabilidades:** nunca en un issue público. Ver [SECURITY.md](SECURITY.md).

## Ramas

| Rama | Uso |
|---|---|
| `main` | Producción: solo lo que se publica. Se etiqueta `v*` para publicar. |
| `develop` | Integración: aquí se trabaja. |

Crea tu rama desde `develop` y abre el pull request hacia `develop`.

## Commits

En español, con la forma `tipo(ámbito): asunto`:

- **Asunto:** en imperativo y presente («agrega», no «agregado»), sin mayúscula inicial ni punto final.
- **Cuerpo:** explica el porqué y contrástalo con el comportamiento anterior.
- **Líneas:** de 100 caracteres como máximo, también en el cuerpo.
- **Un commit por ámbito.** Cada commit debe compilar y pasar las pruebas por sí solo.

| Tipo | Para |
|---|---|
| `caracteristica` | Funcionalidad nueva. |
| `correccion` | Un error corregido. |
| `documentacion` | Solo documentación. |
| `prueba` | Solo pruebas. |
| `refactorizacion` | Cambio interno sin cambiar el comportamiento. |
| `rendimiento` | Mejora de rendimiento. |
| `estilo` | Formato, sin cambiar código. |
| `construccion` | Compilación, empaquetado, CI. |
| `configuracion` | Archivos de configuración del proyecto. |

**Ámbitos:** el paquete o área completa que cambia: `prisma` (CLI y transversal), `contrato`, `escpos`, `conector`, `cola`, `almacen`, `seguridad`, `servidor`, `admin`, `servicio`, `actualizacion`, `config`, `dev`, `sdk`, `empaquetado`, `ci`.

Ejemplo:

```
correccion(almacen): libera los ids interrumpidos por un reinicio

Al arrancar, el puente marcaba como fallidos los trabajos en cola, pero no liberaba su id: el
reintento recibía «duplicado» y el ticket nunca salía. Ahora los libera en la misma transacción.
```

## Pull requests

Antes de abrirlo:

```sh
go test ./...           # incluye las pruebas de la documentación (go test ./docs)
go vet ./...
make lint               # o .\build.ps1 lint: go vet + staticcheck
make build-todo         # o .\build.ps1 build-todo: compila las 6 plataformas
make test-sdk           # si cambiaste sdk/: SDK y firmadores
```

Además:

- **Si cambiaste el contrato:** actualiza a la vez la especificación (`docs/referencia/contrato-documento-v1.md`), el esquema (`schema/`), los ejemplos y el índice de inválidos.
- **Si cambiaste la API, un código de error o un enlace:** actualiza `docs/referencia/`; las pruebas de `./docs` fallan si no.
- **Si cambiaste el SDK:** `npm run construir` y commitea `sdk/js/dist/`.
- **Si cambiaste la base de datos:** agrega una migración nueva; nunca edites una publicada.
- **Agrega tu cambio** a la sección «Sin publicar» de [CHANGELOG.md](CHANGELOG.md).
- **Nunca registres** en la bitácora el contenido de los tickets ni llaves, contraseñas o códigos.

El CI corre las pruebas con `-race` en los tres sistemas, el SDK, los firmadores y una instalación real del servicio. El pull request se integra con el CI en verde y la revisión aprobada.

## Licencia

PRISMA se distribuye con la [licencia Apache-2.0](LICENSE). Al enviar una contribución aceptas que se distribuya con esa misma licencia (sección 5 de la licencia).
