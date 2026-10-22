// PRISMA — textos del panel en español de México.
// Es el único archivo con textos para la persona, para poder traducir el panel más adelante: el resto
// los toma por clave. Los mensajes de error de la API llegan ya escritos desde el puente.
// En la Fase 2 contiene los textos del sistema de diseño; los de cada pantalla se mudan aquí en la Fase 3.
'use strict';
window.PRISMA_TEXTOS = {
  regional: 'es-MX',
  ui: {
    cerrar: 'Cerrar',
    cancelar: 'Cancelar',
    confirmar: 'Confirmar',
    copiar: 'Copiar',
    copiado: 'Copiado',
    copiarCodigo: 'Copiar código',
    noSeCopio: 'No se pudo copiar automáticamente: selecciona el texto y cópialo a mano.',
    cargando: 'Cargando…',
    cerrarAviso: 'Cerrar aviso',
    saltar: 'Saltar al contenido',
    opcional: 'opcional',
    masInformacion: 'Más información sobre',
    saberMas: 'Saber más',
    menu: 'Menú',
    cerrarMenu: 'Cerrar el menú',
    navegacion: 'Secciones del panel',
    navegacionInferior: 'Accesos rápidos',
    consecuencias: 'Esto pasará:',
    codigo: 'Código',
    pasos: 'Avance',
    pasoDe: (n, total) => `Paso ${n} de ${total}`,
    completado: 'completado',
    abrir: 'Abrir',
    abrirDetalle: (que) => `Ver el detalle de ${que}`,
    sinResultados: 'No hay nada que mostrar.',
    tema: { etiqueta: 'Tema del panel', auto: 'Automático (como el sistema)', claro: 'Claro', oscuro: 'Oscuro' },
    estados: { ok: 'Correcto', aviso: 'Advertencia', error: 'Error', info: 'Información', desconocido: 'Sin probar', virtual: 'Virtual' },
    visor: {
      herramientas: 'Herramientas del diagrama',
      acercar: 'Acercar',
      alejar: 'Alejar',
      restablecer: 'Tamaño original',
      grande: 'Ver en grande',
      salirGrande: 'Volver al tamaño normal',
      descargar: 'Descargar SVG',
      copiarMermaid: 'Copiar código Mermaid',
      ayudaTeclado: 'Usa + y − para acercar o alejar, las flechas para moverte y 0 para volver al tamaño original.',
      sinImagen: 'No se encontró la imagen del diagrama. Este es su código:',
    },
    ticket: {
      etiqueta: (mm) => `Vista del ticket en papel de ${mm} mm`,
      papel: (mm) => `Papel de ${mm} mm`,
    },
  },
};
