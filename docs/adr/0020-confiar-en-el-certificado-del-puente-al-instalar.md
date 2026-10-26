# 0020. El instalador confía en el certificado del puente, limitado a la red local

- **Estado:** aceptada, en desarrollo para PRISMA 0.3.0
- **Fecha:** 2026-10-25
- **Reemplazada en parte por:** [0021](0021-autoridad-local-y-certificado-del-servidor.md). Desde 0.3.1 se confía en una autoridad local que firma el certificado del servidor: Firefox rechazaba el certificado único de 0.3.0 (`MOZILLA_PKIX_ERROR_CA_CERT_USED_AS_END_ENTITY`).
- **Complementa:** [0010](0010-certificado-autofirmado.md)

## Contexto

Al terminar de instalar, el instalador abre `https://localhost:8443/admin`. El certificado autofirmado del puente ([ADR 0010](0010-certificado-autofirmado.md)) no es de confianza en ningún equipo, así que lo primero que ve el dueño es una advertencia de seguridad del navegador (en Firefox, `MOZILLA_PKIX_ERROR_SELF_SIGNED_CERT`). La guía explicaba cómo confiar en él a mano, pero ese es justo el paso que nadie espera al instalar.

Se descartaron dos alternativas:

- **Let's Encrypt:** solo emite para dominios públicos. Cada puente necesitaría un nombre público que apunte a una IP privada, un servicio de DNS en la nube e Internet para renovarlo cada pocos meses. Eso va contra el principio de no depender de ninguna nube ni cuenta externa, y algunos routers bloquean esos nombres (protección contra *DNS rebinding*). Quien tenga su propio dominio ya puede usarlo con `modo_tls = "archivos"`.
- **Abrir el panel por `http://localhost`:** quitaba el aviso solo en el panel; las aplicaciones web de ese mismo equipo seguían viéndolo.

El certificado es una CA (así lo aceptan todos los almacenes al importarlo). Confiar en él como raíz tal como era daba a su llave el poder de firmar certificados para cualquier sitio en ese equipo.

## Decisión

- **El certificado autofirmado nuevo lleva restricciones de nombre críticas** (`nameConstraints`, RFC 5280):
  - nombres: `localhost`, `*.local` y los nombres del equipo;
  - IPs: loopback, redes privadas (RFC 1918, `fc00::/7`), enlace local y, si el equipo tuviera una IP pública, solo esa.
  
  Aunque un equipo confíe en él como raíz, su llave no sirve para hacerse pasar por un sitio de Internet. `TestCertificadoLimitadoALaRedLocal` lo comprueba firmando con esa llave un certificado para un dominio público y para `8.8.8.8`.
- **`prisma tls confiar`** agrega el certificado al almacén de confianza del sistema de ese equipo, después de quitar los que PRISMA haya puesto antes; con `--quitar`, los quita. Requiere administrador.
  - Windows: «Entidades de certificación raíz de confianza» del equipo, con la API de certificados de Windows (sin PowerShell ni `certutil`, cuya salida cambia con el idioma).
  - macOS: el llavero del sistema, con `security add-trusted-cert`. Ahí `confiar` **solo agrega** (y no hace nada si el certificado ya está): quitar una confianza de administrador, con `remove-trusted-cert` o `delete-certificate -t`, espera una autorización en una ventana. Se comprobó en el CI que sin pantalla se queda esperando para siempre, y que después hasta agregar se cuelga. Por eso solo `--quitar` lo hace, desde la Terminal, donde macOS muestra esa ventana.
  - Linux: `ca-certificates`.
  
  Chrome, Edge, Safari y Firefox (que importa las raíces del sistema en Windows y macOS) dejan de avisar en ese equipo.
- **Solo acepta certificados limitados:** se niega con uno anterior a 0.3.0 y pide regenerarlo, porque regenerar obliga a volver a instalarlo en los demás equipos y esa decisión es del dueño. Tampoco se usa con `modo_tls = "archivos"`.
- **Los instaladores lo usan** después de iniciar el servicio (`--esperar`, mientras el puente genera el certificado) y lo quitan al desinstalar. Los demás equipos siguen confiando en él a mano, como hasta ahora.

## Consecuencias

- La primera vez que se abre el panel en el equipo del puente, no hay advertencia.
- Un puente que se actualiza desde 0.2.0 conserva su certificado sin límites. El panel lo dice y propone regenerarlo; `prisma tls info` muestra «limitado a la red local: no».
- Al regenerar el certificado desde el panel no se vuelve a confiar en él solo: en Linux y macOS el servicio no corre como administrador. La CLI y el panel indican que hay que ejecutar `prisma tls confiar`.
- Chrome y Chromium en Linux usan además su propia base (NSS): ahí sigue haciendo falta el paso a mano de la guía.
- Agregar una raíz al sistema es una acción visible: el instalador lo hace solo en el equipo donde se instala y la documentación lo dice.
- En macOS, regenerar el certificado deja la confianza del anterior en el llavero hasta que se ejecute `sudo prisma tls confiar --quitar` desde la Terminal; está limitado a la red local y su llave solo está en ese equipo.
- Ninguna herramienta del sistema puede colgar al instalador: en Linux y macOS cada una tiene un límite de 2 minutos.
