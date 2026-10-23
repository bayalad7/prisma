# Política de seguridad

PRISMA controla impresoras y cajones de dinero en negocios reales. Si encuentras una vulnerabilidad, repórtala en privado para que se pueda corregir antes de que se conozca.

## Versiones con soporte

| Versión | Soporte |
|---|---|
| 0.1.x | ✅ Recibe correcciones de seguridad |
| < 0.1 | ❌ No se publicó |

Mientras PRISMA esté en 0.x, solo la última versión menor recibe correcciones. Actualiza con `prisma actualizar` ([operacion/actualizacion.md](docs/operacion/actualizacion.md)).

## Cómo reportar una vulnerabilidad

**No abras un issue público.** Usa el reporte privado de GitHub: en el repositorio, *Security → Report a vulnerability* ([enlace directo](https://github.com/bayalad7/prisma/security/advisories/new)).

Incluye, si puedes:

- la versión (`prisma version`) y el sistema operativo;
- qué componente afecta: API `/v1`, panel, SDK, firmadores, instaladores o `prisma actualizar`;
- los pasos para reproducirlo y qué puede lograr un atacante;
- si ya lo comunicaste a alguien más.

No incluyas llaves `pk_`, contraseñas ni contenido de tickets reales.

## Qué esperar

| Paso | Plazo objetivo |
|---|---|
| Confirmación de recibido | 3 días hábiles |
| Evaluación inicial y severidad | 10 días hábiles |
| Corrección publicada (severidad alta o crítica) | 30 días |

Te mantendremos al tanto del avance y, si quieres, te daremos crédito en el aviso de seguridad y en el [registro de cambios](CHANGELOG.md).

## Alcance

**Dentro del alcance:**

- saltarse la autorización por niveles (origen, llave, firma) o los permisos de impresoras y operaciones (`abrir_cajon`, `raw`);
- entrar al panel sin la contraseña, CSRF, secuestro de sesión;
- leer llaves, contraseñas o datos de la base sin permisos de administrador del equipo;
- hacer que `prisma actualizar` o el instalador de Linux acepten un binario no firmado por el proyecto;
- registrar contenido de tickets o credenciales en las bitácoras;
- inyectar comandos ESC/POS a través de los campos de texto del documento.

**Fuera del alcance** (son riesgos aceptados y documentados en el [modelo de amenazas](docs/seguridad/modelo-de-amenazas.md)):

- imprimir con una aplicación de nivel 1 enviando un `Origin` inventado desde fuera del navegador;
- usar una llave `pk_` obtenida del código de una página web;
- imprimir directo en el puerto 9100 de una impresora de red, sin pasar por PRISMA;
- ataques que requieren ser administrador del equipo del puente;
- la falta de firma de código de los instaladores de 0.1.0;
- denegación de servicio por volumen desde la red local, más allá de los límites configurables.

## Para quien mantiene el proyecto

Si se filtra la llave privada de las versiones, sigue [seguridad/politicas.md](docs/seguridad/politicas.md#llave-de-las-versiones).
