# 0021. Una autoridad local firma el certificado del servidor

- **Estado:** aceptada, en desarrollo para PRISMA 0.3.1
- **Fecha:** 2026-10-26
- **Reemplaza en parte a:** [0010](0010-certificado-autofirmado.md) (un solo certificado autofirmado) y [0020](0020-confiar-en-el-certificado-del-puente-al-instalar.md) (en qué se confía)

## Contexto

Hasta 0.3.0, el puente presentaba un solo certificado autofirmado que era a la vez autoridad (para que los almacenes de confianza lo aceptaran al importarlo) y certificado del sitio. En 0.3.0, el instalador lo agregó a las raíces de confianza de Windows. Edge y Chrome abrieron el panel sin aviso, pero Firefox no: rechaza que una autoridad se use como certificado del sitio (`MOZILLA_PKIX_ERROR_CA_CERT_USED_AS_END_ENTITY`).

Además, cada vez que se regeneraba el certificado (cambio de IP o vencimiento), todos los equipos debían volver a confiar en él.

## Decisión

El modelo de mkcert:

- **Autoridad local** (`tls/autoridad.pem` y `tls/autoridad-llave.pem`, la llave solo para el usuario del servicio):
  - ECDSA P-256, válida 10 años, `MaxPathLen 0` (no firma otras autoridades);
  - limitada a la red local con restricciones de nombre críticas (ADR 0020);
  - se llama «PRISMA autoridad local», seguido del nombre del equipo y de un identificador de su llave (desde 0.3.4). Dos autoridades del mismo equipo nunca tienen el mismo sujeto: con el mismo, Firefox arma la cadena con la que no es y falla (`SEC_ERROR_UNKNOWN`). Pasa cuando la vieja sigue entre las raíces, por ejemplo tras reinstalar en macOS o con un puente de prueba.

  Es lo único en lo que confían los equipos: la instalan `prisma tls confiar` y los instaladores, y la entrega *Descargar certificado* del panel.
- **Certificado del servidor** (`tls/certificado.pem` y `tls/llave.pem`):
  - no es autoridad, dura 825 días y lleva los nombres y las IPs del equipo;
  - lo firma la autoridad, y `certificado.pem` guarda la cadena (servidor y autoridad) para que el puente la presente completa.
- **Regenerar** (panel o `prisma tls regenerar`) emite otro certificado del servidor con la **misma** autoridad. Los equipos que ya confían en ella no tienen que hacer nada.
- **Se crea una autoridad nueva solo si:**
  - no había (puentes de antes de 0.3.1);
  - ya no cubre los nombres del equipo (cambió su nombre);
  - está por vencer;
  - o se pide con `--nueva-autoridad`.

  En esos casos la CLI y el panel avisan que hay que volver a instalarla.
- **Al arrancar**, si el certificado del servidor venció, está por vencer o no incluye alguna IP actual, y la autoridad lo cubre, el puente lo renueva solo.

## Consecuencias

- Firefox, Edge, Chrome y Safari abren el panel sin aviso en el equipo del puente.
- Un cambio de IP o un vencimiento ya no obliga a reinstalar nada en las cajas.
- **Desde 0.2.0 o 0.3.0:**
  - el certificado anterior sigue funcionando, pero no tiene autoridad;
  - el panel y `prisma tls info` lo dicen, y `prisma tls confiar` pide regenerarlo;
  - ese regenerar crea la autoridad, y hay que instalarla una vez en cada equipo.
  - En Windows, `tls confiar` reemplaza el certificado anterior de PRISMA en las raíces.
- **Pruebas que lo cubren:**
  - `TestAutoridadLocalYCertificadoDelServidor` comprueba la cadena y los límites de la autoridad.
  - `TestRegenerarConservaLaAutoridad` y `TestCargarRenuevaConLaMismaAutoridad` comprueban que se conserva al regenerar y al arrancar, y que una autoridad nueva tiene otro sujeto.
  - En el CI de Linux, `e2e/certificado-firefox.spec.mjs` abre el puente con un **Firefox real**. Sin la autoridad en su base NSS, falla (`SEC_ERROR_UNKNOWN_ISSUER`); con ella, abre.
- La carpeta `tls/` tiene una llave más que proteger (la de la autoridad), con los mismos permisos que la del certificado.
