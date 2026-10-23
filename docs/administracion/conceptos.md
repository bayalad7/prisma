# Conceptos básicos

Las ideas que usa el panel, sin tecnicismos. El [glosario](../referencia/glosario.md) tiene las definiciones exactas, para quien programa.

| Concepto | Qué es |
|---|---|
| **Puente** | PRISMA, el programa que corre en un equipo de tu negocio. Recibe los tickets de tus aplicaciones y los manda a la impresora. Funciona sin Internet. |
| **Panel** | La página donde configuras el puente: `https://<equipo-del-puente>:8443/admin`. Solo tú entras, con tu contraseña, y solo desde la red del negocio. |
| **Impresora** | Una impresora de tickets que agregaste al puente. Puede estar conectada por red, USB o Bluetooth. |
| **Nombre corto (alias)** | Cómo eligen las aplicaciones la impresora: `caja`, `cocina`. Las aplicaciones nunca saben su dirección ni cómo está conectada; si cambias la impresora, solo actualizas el puente. |
| **Impresora virtual** | Una impresora de mentira que vive dentro del puente: recibe los tickets igual que una real y los muestra en pantalla. Sirve para probar sin papel ([Probar sin impresora](probar-sin-impresora.md)). |
| **Aplicación** | Un programa o página que tiene tu permiso para imprimir: tu punto de venta, tu menú digital. Cada una tiene sus impresoras permitidas. En la API y en la terminal se llama *cliente*. |
| **Página web (origen)** | La dirección de la página que imprime, como aparece en el navegador: `https://pos.mitienda.mx`. El puente solo acepta tickets de las páginas que aprobaste. |
| **Llave** | Una contraseña larga de la aplicación (`pk_…`). La aplicación la manda con cada ticket; así, ningún otro equipo de la red puede imprimir haciéndose pasar por ella. Se muestra una sola vez. |
| **Nivel de seguridad** | Cuánto se protege una aplicación: 1, solo la dirección de la página; 2, dirección y llave (el recomendado); 3, el servidor de la aplicación firma cada ticket. [Qué nivel elegir](../integradores/niveles-de-seguridad.md). |
| **Solicitud** | Una página que intentó imprimir sin permiso. Queda en *Solicitudes* para que la apruebes o la rechaces. |
| **Ticket o documento** | Lo que una aplicación manda a imprimir: textos, renglones, totales, códigos QR, logos y el corte. |
| **Trabajo** | Cada ticket que el puente recibió, con su resultado: impreso, con aviso o fallido, y por qué. El puente guarda el resultado, nunca el contenido. |
| **Imágenes** | Logos guardados en el puente para imprimirlos por su nombre. En la API se llaman *recursos*. |
| **Certificado** | Lo que permite que los navegadores hablen con el puente de forma cifrada (HTTPS). El puente crea el suyo; cada equipo que imprime debe confiar en él una vez. |
| **Página de códigos** | La tabla que usa la impresora para los acentos y la ñ. Si salen símbolos raros, se elige otra con *Probar acentos*. |

Cómo encaja todo, en un dibujo: [Contexto del sistema](../diagramas/01-contexto.md).
