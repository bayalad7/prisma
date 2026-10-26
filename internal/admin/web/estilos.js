// PRISMA — galería del sistema de diseño y prototipos (/admin/estilos, solo en modo desarrollo).
// Muestra cada componente de sistema.js en sus estados, y tres pantallas clave con datos de ejemplo
// para aprobar el diseño antes de aplicarlo al panel. No llama a la API.
// Los textos de los prototipos son borradores: en la Fase 3 pasan a textos.js con las pantallas reales.
'use strict';
(() => {
  const UI = window.PrismaUI;
  const { h, agregar, vaciar, icono, boton, insignia, tarjeta, nota, vacio, esqueleto, pasos, pestanas, codigo, tabla,
    campo, entrada, selector, interruptor, casilla, revisarCampos, aviso, confirmar, dialogo, selectorTema, marco, logo,
    visorDiagrama, ticketPapel } = UI;
  const raiz = document.getElementById('raiz');

  const encabezado = (titulo, subtitulo, ...acciones) => h('div', { class: 'encabezado' },
    h('div', {}, h('h1', {}, titulo), subtitulo && h('p', { class: 'subtitulo' }, subtitulo)),
    acciones.length > 0 && h('div', { class: 'acciones' }, acciones));
  const seccion = (titulo, ...contenido) => tarjeta({ titulo }, h('div', { class: 'muestra' }, contenido));
  const fila = (...hijos) => h('div', { class: 'muestra-fila' }, hijos);

  // ---------------------------------------------------------------------------------------------
  // Galería
  // ---------------------------------------------------------------------------------------------
  const TOKENS = ['fondo', 'superficie', 'superficie-2', 'superficie-3', 'tinta', 'tenue', 'borde', 'borde-control',
    'acento', 'acento-fuerte', 'acento-suave', 'ok', 'ok-suave', 'aviso', 'aviso-suave', 'error', 'error-suave',
    'info', 'info-suave', 'desconocido', 'desconocido-suave', 'virtual', 'virtual-suave', 'papel'];

  async function nombresIconos() {
    const r = await fetch('/admin/iconos.svg');
    const doc = new DOMParser().parseFromString(await r.text(), 'image/svg+xml');
    return [...doc.querySelectorAll('symbol')].map((s) => s.id.replace(/^i-/, ''));
  }

  function vistaFundamentos(v) {
    const paleta = h('div', { class: 'paleta' }, TOKENS.map((t) => {
      const m = h('div', { class: 'color-muestra' });
      m.style.background = `var(--${t})`;
      const valor = h('span', { class: 'mono tenue' });
      requestAnimationFrame(() => { valor.textContent = getComputedStyle(document.documentElement).getPropertyValue(`--${t}`).trim(); });
      return h('div', { class: 'color' }, m, h('span', { class: 'mono' }, `--${t}`), valor);
    }));
    const iconos = h('div', { class: 'paleta' });
    nombresIconos().then((lista) => agregar(iconos, lista.map((n) => h('div', { class: 'fila' }, icono(n), h('span', { class: 'mono pequeno' }, n)))));
    const espacio = (n) => { const b = h('span'); b.style.cssText = ''; b.style.display = 'inline-block'; b.style.height = '12px'; b.style.background = 'var(--acento)'; b.style.width = `var(--e-${n})`; return h('div', { class: 'fila' }, b, h('span', { class: 'mono pequeno' }, `--e-${n}`)); };
    const radio = (n) => { const b = h('div', { class: 'color-muestra' }); b.style.width = '64px'; b.style.background = 'var(--superficie-2)'; b.style.borderRadius = `var(--${n})`; return h('div', { class: 'color' }, b, h('span', { class: 'mono' }, `--${n}`)); };
    const sombra = (n) => { const b = h('div', { class: 'color-muestra' }); b.style.width = '96px'; b.style.background = 'var(--superficie)'; b.style.boxShadow = `var(--${n})`; return h('div', { class: 'color' }, b, h('span', { class: 'mono' }, `--${n}`)); };
    agregar(v,
      encabezado('Fundamentos', 'Tokens de color, tipografía, espaciado, radios, sombras e iconos. Cambia el tema para ver el modo oscuro.', selectorTema()),
      seccion('Colores', paleta, h('p', { class: 'tenue pequeno' }, 'Una prueba de Go verifica el contraste de cada par de texto y fondo (4.5:1) y de los bordes de controles (3:1), en claro y en oscuro.')),
      seccion('Estados', fila(['ok', 'aviso', 'error', 'info', 'desconocido', 'virtual'].map((e) => insignia(e))),
        h('p', { class: 'tenue pequeno' }, 'Siempre color + icono + texto: nunca solo color.')),
      seccion('Tipografía', h('h1', {}, 'Título de pantalla'), h('h2', {}, 'Título de tarjeta'), h('h3', {}, 'Subtítulo'),
        h('p', {}, 'Texto normal: la impresora de la caja está lista para imprimir.'), h('p', { class: 'pequeno tenue' }, 'Texto pequeño y tenue para ayudas.'),
        h('p', {}, h('code', {}, 'caja'), ' — monoespaciada para alias, direcciones y llaves.')),
      h('div', { class: 'rejilla' },
        seccion('Espaciado', [1, 2, 3, 4, 5, 6, 8].map(espacio)),
        seccion('Radios y sombras', fila(['radio-chico', 'radio', 'radio-grande'].map(radio)), fila(['sombra-1', 'sombra-2'].map(sombra)))),
      seccion('Iconos (un solo sprite: iconos.svg)', iconos));
  }

  function vistaComponentes(v) {
    const espera = boton({ texto: 'Probar conexión', icono: 'red' });
    espera.addEventListener('click', () => UI.conEspera(espera, () => new Promise((r) => setTimeout(r, 1500))));
    agregar(v,
      encabezado('Componentes', 'Botones, insignias, tarjetas, tablas, pestañas, pasos, código y diálogos.'),
      seccion('Botones',
        fila(boton({ texto: 'Guardar impresora', variante: 'primario' }), boton({ texto: 'Cancelar' }), boton({ texto: 'Eliminar', variante: 'peligro', icono: 'papelera' }),
          boton({ texto: 'Eliminar para siempre', variante: 'peligro-lleno' }), boton({ texto: 'Ver detalles', variante: 'texto' })),
        fila(boton({ texto: 'Chico', chico: true }), boton({ texto: 'Copiar', icono: 'copiar', chico: true }), boton({ icono: 'puntos', etiqueta: 'Más acciones' }),
          boton({ texto: 'Deshabilitado', disabled: true }), espera)),
      seccion('Tarjetas',
        h('div', { class: 'rejilla' },
          tarjeta({ titulo: 'caja', subtitulo: 'Caja principal · Red 192.168.1.50', icono: 'impresora', acciones: insignia('ok', 'En línea') }, h('p', { class: 'tenue' }, 'Contenido de la tarjeta.')),
          tarjeta({ titulo: 'prueba', subtitulo: 'Impresora virtual · 80 mm', icono: 'virtual', acciones: insignia('virtual') }, h('p', { class: 'tenue' }, 'No imprime en papel: muestra el ticket en pantalla.')))),
      seccion('Tabla que se vuelve tarjetas en celular', tablaTrabajos(TRABAJOS.slice(0, 4), (t) => aviso(`Abrir ${t.ref}`))),
      seccion('Pestañas (flechas para moverse)', pestanas([
        { id: 'a', titulo: 'Windows', contenido: h('p', {}, 'Pasos para Windows.') },
        { id: 'b', titulo: 'macOS', contenido: h('p', {}, 'Pasos para macOS.') },
        { id: 'c', titulo: 'Android', contenido: h('p', {}, 'Pasos para Android.') }], { etiqueta: 'Sistema operativo' })),
      seccion('Pasos de asistente', pasos(['Tipo de impresora', 'Conexión', 'Probar'], 1)),
      seccion('Bloque de código', codigo({ SDK: CODIGO_SDK, curl: CODIGO_CURL }, { etiqueta: 'Código de integración' })),
      seccion('Diálogos',
        fila(boton({ texto: 'Confirmar una acción destructiva', variante: 'peligro', onclick: () => confirmar('¿Eliminar la impresora «cocina»?',
          'La impresora deja de estar disponible de inmediato.', { boton: 'Eliminar', peligro: true,
            consecuencias: ['«Punto de venta web» y «Menú digital» ya no podrán imprimir en ella.', 'El historial de trabajos se conserva.'] }) }),
        boton({ texto: 'Diálogo informativo', onclick: () => dialogo({ titulo: 'Ticket de diagnóstico', cuerpo: ticketPapel({ imagen: ticketDeEjemplo(48), anchoMm: 80 }) }) }))));
  }

  function vistaFormularios(v) {
    const alias = campo({ id: 'f-alias', etiqueta: 'Nombre corto para las aplicaciones (alias)', control: entrada({ mono: true, autocomplete: 'off' }),
      ayuda: 'Minúsculas, números, guion o guion bajo. Por ejemplo: caja, cocina.',
      info: { texto: 'Las aplicaciones eligen la impresora por este nombre, nunca por su dirección. No se puede cambiar después.', enlace: { href: '#', texto: 'Saber más' } },
      validar: (val) => !val ? 'Escribe un nombre corto, por ejemplo «caja».' : (/^[a-z0-9][a-z0-9_-]{0,31}$/.test(val) ? '' : 'Usa solo minúsculas, números, guion (-) o guion bajo (_), sin espacios.') });
    const direccion = campo({ id: 'f-dir', etiqueta: 'Dirección de la impresora', control: entrada({ mono: true, placeholder: 'p. ej. 192.168.1.50' }), ayuda: 'La IP que aparece en el ticket de configuración de la impresora.',
      validar: (val) => val ? '' : 'Escribe la dirección de la impresora.' });
    const conexion = campo({ id: 'f-tipo', etiqueta: 'Cómo está conectada', control: selector({ opciones: [['red', 'Por red (cable o wifi)'], ['usb', 'Por USB'], ['bluetooth', 'Por Bluetooth']], valor: 'red' }) });
    const notas = campo({ id: 'f-notas', etiqueta: 'Notas', opcional: true, control: h('textarea', {}) });
    const form = h('form', { class: 'pila-amplia', novalidate: true, onsubmit: (ev) => {
      ev.preventDefault();
      if (revisarCampos([alias, direccion])) aviso('Formulario válido.', 'ok');
    } },
    h('div', { class: 'campos' }, alias, direccion), conexion, notas,
    interruptor({ id: 'f-confirmar', texto: 'Preguntar a la impresora si recibió cada ticket', ayuda: 'Recomendado si tu impresora lo admite: lo comprobamos al imprimir la prueba.', marcado: true }),
    h('fieldset', {}, h('legend', {}, 'Lo que tiene la impresora'),
      casilla({ id: 'f-cuchilla', texto: 'Corta el papel sola (cuchilla)', marcado: true }), casilla({ id: 'f-cajon', texto: 'Tiene un cajón de dinero conectado' })),
    h('fieldset', {}, h('legend', {}, 'Ancho del papel'), h('div', { class: 'opciones' },
      [['58', '58 mm', 'Rollo angosto, 32 letras por renglón'], ['80', '80 mm', 'Rollo estándar, 48 letras por renglón']].map(([val, t, d]) =>
        h('label', { class: 'opcion' }, h('input', { type: 'radio', name: 'papel', value: val, checked: val === '80' }), h('span', { class: 'titulo-opcion' }, t), h('span', { class: 'tenue' }, d))))),
    h('div', { class: 'acciones' }, boton({ texto: 'Cancelar' }), boton({ texto: 'Guardar', variante: 'primario', tipo: 'submit' })));
    agregar(v, encabezado('Formularios', 'Campos con ayuda, explicación y error en línea. Pulsa «Guardar» vacío o escribe un alias con espacios.'), tarjeta({}, form));
  }

  function vistaEstados(v) {
    agregar(v,
      encabezado('Estados y avisos', 'Avisos en la página, avisos flotantes, estados vacíos y de carga.'),
      seccion('Avisos en la página',
        nota({ tipo: 'info', titulo: 'Tu impresora es virtual', texto: 'Los tickets se ven aquí, no salen en papel. Cuando conectes la impresora física, agrégala desde Impresoras.' }),
        nota({ tipo: 'ok', titulo: 'La impresora respondió', texto: 'Está lista para imprimir.' }),
        nota({ tipo: 'aviso', titulo: 'El certificado vence en 12 días', texto: 'Regenéralo antes y vuelve a instalarlo en cada equipo que imprime.', acciones: boton({ texto: 'Ir a HTTPS', chico: true }) }),
        nota({ tipo: 'error', titulo: 'La impresora no respondió', texto: 'Revisa que esté encendida y conectada a la misma red que este equipo.', cerrable: true })),
      seccion('Avisos flotantes', fila(
        boton({ texto: 'Información', onclick: () => aviso('Se copió la llave.') }),
        boton({ texto: 'Correcto', onclick: () => aviso('Impresora «caja» agregada.', 'ok') }),
        boton({ texto: 'Advertencia', onclick: () => aviso('La impresora recibió el ticket, pero no tiene papel.', 'aviso') }),
        boton({ texto: 'Error', onclick: () => aviso('No hay conexión con el puente. Revisa que PRISMA siga en marcha.', 'error') }))),
      h('div', { class: 'rejilla' },
        seccion('Estado vacío', vacio({ icono: 'aplicacion', titulo: 'Todavía no hay aplicaciones', texto: 'Una aplicación es un sistema o página web que puede imprimir en este puente, como tu punto de venta.',
          accion: boton({ texto: 'Autorizar una aplicación', variante: 'primario', icono: 'mas' }) })),
        seccion('Cargando', esqueleto({ lineas: 4 }))));
  }

  function vistaContenido(v) {
    agregar(v,
      encabezado('Diagramas y tickets', 'Bases del visor de diagramas (Fase 5) y de la vista de ticket en papel térmico (Fase 6).'),
      seccion('Visor de diagramas', visorDiagrama({ src: DIAGRAMA_EJEMPLO, alt: 'Diagrama: la aplicación envía el ticket al puente y el puente lo imprime.',
        titulo: 'Cómo viaja un ticket', descripcion: 'La aplicación envía el documento al puente, que revisa el permiso y lo manda a la impresora.',
        mermaid: 'flowchart LR\n  app[Aplicación] --> puente[PRISMA] --> imp[Impresora]' })),
      h('div', { class: 'rejilla' },
        seccion('Ticket de 80 mm', ticketPapel({ imagen: ticketDeEjemplo(48), anchoMm: 80 })),
        seccion('Ticket de 58 mm', ticketPapel({ imagen: ticketDeEjemplo(32), anchoMm: 58 }))));
  }

  // ---------------------------------------------------------------------------------------------
  // Datos de ejemplo
  // ---------------------------------------------------------------------------------------------
  const CODIGO_SDK = 'import { Prisma } from "@bayalad7/prisma";\n\nconst prisma = new Prisma({\n  url: "https://192.168.1.10:8443",\n  llave: "pk_…",\n});\n\nawait prisma.imprimir("caja", [\n  { tipo: "texto", valor: "¡Hola desde mi app!", alineacion: "centro" },\n  { tipo: "cortar" },\n]);';
  const CODIGO_CURL = 'curl --cacert prisma.crt https://192.168.1.10:8443/v1/imprimir \\\n  -H "Authorization: Bearer pk_…" \\\n  -H "Content-Type: application/json" \\\n  --data-binary @documento.json';
  const TRABAJOS = [
    { hora: 'hoy 13:42', app: 'Punto de venta web', imp: 'caja', estado: 'ok', ref: 'VTA-1042', dur: '0.4 s' },
    { hora: 'hoy 13:40', app: 'Menú digital', imp: 'cocina', estado: 'aviso', ref: 'PED-0187', dur: '0.9 s', adv: 'Un carácter no existe en la impresora' },
    { hora: 'hoy 13:31', app: 'Punto de venta web', imp: 'barra', estado: 'error', ref: 'VTA-1041', dur: '3.0 s', err: 'La impresora no respondió' },
    { hora: 'hoy 13:30', app: 'Mi app en desarrollo', imp: 'prueba', estado: 'virtual', ref: 'DEV-7', dur: '0.1 s' },
    { hora: 'hoy 13:12', app: 'Punto de venta web', imp: 'caja', estado: 'ok', ref: 'VTA-1040', dur: '0.4 s' },
    { hora: 'hoy 12:58', app: 'Panel (prueba)', imp: 'caja', estado: 'ok', ref: 'Diagnóstico', dur: '0.5 s' },
  ];
  const ESTADO_TRABAJO = { ok: ['ok', 'Impreso'], aviso: ['aviso', 'Impreso con aviso'], error: ['error', 'Falló'], virtual: ['ok', 'Impreso'] };
  function tablaTrabajos(lista, alAbrir) {
    return tabla({
      etiqueta: 'Trabajos', alAbrir, nombreFila: (t) => `el trabajo ${t.ref}`,
      columnas: [
        { titulo: 'Referencia', celda: (t) => t.ref },
        { titulo: 'Fecha', celda: (t) => t.hora, clase: 'numeros' },
        { titulo: 'Aplicación', celda: (t) => t.app },
        { titulo: 'Impresora', celda: (t) => h('span', { class: 'fila' }, h('span', { class: 'mono' }, t.imp), t.estado === 'virtual' && insignia('virtual')) },
        { titulo: 'Estado', celda: (t) => insignia(...ESTADO_TRABAJO[t.estado]) },
        { titulo: 'Detalle', celda: (t) => t.err || t.adv || '—' },
      ],
      filas: lista,
    });
  }
  const DIAGRAMA_EJEMPLO = 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(
    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 560 120" width="560" height="120" font-family="sans-serif" font-size="15">' +
    '<rect x="10" y="35" width="140" height="50" rx="8" fill="#e8f0fe" stroke="#3b6db3"/><text x="80" y="65" text-anchor="middle" fill="#102a43">Aplicación</text>' +
    '<rect x="210" y="35" width="140" height="50" rx="8" fill="#e6f4ea" stroke="#2f7d32"/><text x="280" y="65" text-anchor="middle" fill="#0b2e13">PRISMA</text>' +
    '<rect x="410" y="35" width="140" height="50" rx="8" fill="#e8f0fe" stroke="#3b6db3"/><text x="480" y="65" text-anchor="middle" fill="#102a43">Impresora</text>' +
    '<path d="M150 60h52M350 60h52" stroke="#444" stroke-width="2" marker-end="url(#f)"/>' +
    '<defs><marker id="f" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto"><path d="M0 0L10 5L0 10z" fill="#444"/></marker></defs></svg>');

  // Ticket aproximado dibujado en canvas (la Fase 6 lo reemplaza por el raster real de la impresora virtual).
  function ticketDeEjemplo(columnas) {
    const celda = 12, alto = 24, ancho = columnas * celda;
    const lineas = [['LA ESPERANZA', 'c', true], ['Cafetería · Colima', 'c'], ['='.repeat(columnas)], ['Café americano', 'par', '$35.00'],
      ['Pan dulce x2', 'par', '$28.00'], ['Jugo de naranja', 'par', '$30.00'], ['-'.repeat(columnas)], ['TOTAL', 'par', '$93.00', true],
      [''], ['¡Gracias por su visita!', 'c'], ['QR'], ['']];
    const c = document.createElement('canvas');
    c.width = ancho; c.height = (lineas.length + 5) * alto;
    const x = c.getContext('2d');
    x.fillStyle = '#fff'; x.fillRect(0, 0, c.width, c.height); x.fillStyle = '#000'; x.textBaseline = 'top';
    let y = 6;
    for (const [t, modo, der, negrita] of lineas) {
      x.font = `${negrita || (modo === 'c' && t === 'LA ESPERANZA') ? 'bold ' : ''}20px monospace`;
      if (t === 'QR') {
        const lado = 6 * alto, x0 = (ancho - lado) / 2;
        for (let i = 0; i < 21; i++) for (let j = 0; j < 21; j++) if ((i * 7 + j * 13 + i * j) % 3 === 0 || ((i < 7 || i > 13) && (j < 7 || j > 13) && (i % 6 === 0 || j % 6 === 0 || (i % 6 > 1 && i % 6 < 5 && j % 6 > 1 && j % 6 < 5)))) x.fillRect(x0 + i * lado / 21, y + j * lado / 21, lado / 21, lado / 21);
        y += lado + 8; continue;
      }
      if (modo === 'c') { x.textAlign = 'center'; x.fillText(t, ancho / 2, y); } else { x.textAlign = 'left'; x.fillText(t, 0, y); }
      if (modo === 'par') { x.textAlign = 'right'; x.fillText(der, ancho, y); }
      y += alto;
    }
    x.textAlign = 'left'; x.font = '16px monospace'; x.fillText('✂ - - - - - - - - - - - - - - - - - - - - - - -', 0, c.height - alto);
    return c.toDataURL('image/png');
  }

  // ---------------------------------------------------------------------------------------------
  // Prototipos: estructura del panel propuesta en la Fase 1
  // ---------------------------------------------------------------------------------------------
  const GRUPOS_PANEL = [
    { titulo: 'Operación', secciones: [
      { id: 'proto-inicio', texto: 'Inicio', icono: 'inicio' },
      { id: 'proto-impresora', texto: 'Impresoras', icono: 'impresora' },
      { id: 'proto-solicitudes', texto: 'Solicitudes', icono: 'solicitud', contador: 2, etiquetaContador: '2 pendientes' },
      { id: 'proto-trabajos', texto: 'Trabajos', icono: 'trabajos' }] },
    { titulo: 'Integración', secciones: [
      { id: 'proto-aplicaciones', texto: 'Aplicaciones', icono: 'aplicacion' },
      { id: 'proto-pruebas', texto: 'Probar impresión', textoCorto: 'Probar', icono: 'probar' },
      { id: 'proto-imagenes', texto: 'Imágenes', icono: 'imagen' }] },
    { titulo: 'Sistema', secciones: [
      { id: 'proto-ajustes', texto: 'Ajustes', icono: 'ajustes' },
      { id: 'proto-ayuda', texto: 'Ayuda', icono: 'ayuda' }] },
  ];
  const marcaPanel = () => h('a', { class: 'marca', href: '#proto-inicio' }, logo(30), h('span', { class: 'marca-texto' },
    h('span', { class: 'marca-nombre' }, 'PRISMA'), h('span', { class: 'marca-puente' }, 'La Esperanza')));

  function protoInicio(v) {
    const hechos = 3, total = 5;
    const paso = (hecho, titulo, texto, accion) => h('li', { class: hecho ? 'hecho' : '' },
      h('span', { class: 'paso-marca', 'aria-hidden': 'true' }, hecho ? icono('check') : ''),
      h('div', {}, h('p', { class: 'paso-titulo' }, titulo, hecho && h('span', { class: 'solo-lector' }, ' (hecho)')), texto && h('p', { class: 'tenue pequeno' }, texto)),
      !hecho && accion);
    const barra = h('span'); barra.style.width = `${(hechos / total) * 100}%`;
    const cifra = (estado, ic, valor, texto, accion) => tarjeta({ clase: `estado-${estado}` },
      h('div', { class: 'cifra-estado' }, icono(ic), h('div', {}, h('div', { class: 'cifra' }, valor), h('div', { class: 'tenue pequeno' }, texto))), accion);
    agregar(v,
      encabezado('La Esperanza', 'Estado del puente de impresión de este negocio.'),
      tarjeta({ titulo: 'Primeros pasos', subtitulo: `${hechos} de ${total} pasos completados`, icono: 'check', acciones: boton({ texto: 'Ocultar', variante: 'texto', chico: true }) },
        h('div', { class: 'progreso', role: 'progressbar', 'aria-valuemin': '0', 'aria-valuemax': String(total), 'aria-valuenow': String(hechos), 'aria-label': 'Primeros pasos' }, h('div', { class: 'progreso-barra' }, barra)),
        h('ol', { class: 'lista-pasos' },
          paso(true, 'Crear la contraseña del panel'),
          paso(true, 'Agregar una impresora', 'Agregaste la impresora virtual «prueba».'),
          paso(true, 'Imprimir un ticket de prueba'),
          paso(false, 'Autorizar tu primera aplicación', 'Para que tu punto de venta o tu página web puedan imprimir.', boton({ texto: 'Autorizar', variante: 'primario', chico: true })),
          paso(false, 'Conectar tu impresora de tickets', 'Por ahora solo tienes la impresora virtual: los tickets se ven en pantalla, no en papel.', boton({ texto: 'Agregar', chico: true, onclick: () => { location.hash = '#proto-impresora'; } })))),
      h('div', { class: 'rejilla de-cuatro' },
        cifra('ok', 'impresora', '2 de 3', 'impresoras listas', h('div', { class: 'pila' },
          h('div', { class: 'fila-entre' }, h('span', {}, h('span', { class: 'mono' }, 'caja'), ' ', h('span', { class: 'tenue pequeno' }, 'Caja principal')), insignia('ok', 'En línea')),
          h('div', { class: 'fila-entre' }, h('span', {}, h('span', { class: 'mono' }, 'prueba'), ' ', h('span', { class: 'tenue pequeno' }, 'Para probar')), insignia('virtual')),
          h('div', { class: 'fila-entre' }, h('span', {}, h('span', { class: 'mono' }, 'barra'), ' ', h('span', { class: 'tenue pequeno' }, 'Barra de bebidas')), insignia('error', 'No responde')),
          h('div', {}, boton({ texto: 'Revisar «barra»', chico: true, icono: 'derecha' })))),
        cifra('aviso', 'solicitud', '2', 'páginas web piden permiso para imprimir', h('div', {}, boton({ texto: 'Revisar solicitudes', variante: 'primario', chico: true }))),
        cifra('error', 'error', '1', 'trabajo falló hoy (de 42)', h('div', {}, boton({ texto: 'Ver qué pasó', chico: true, onclick: () => { location.hash = '#proto-trabajos'; } }))),
        cifra('info', 'candado', '21 días', 'para que venza el certificado HTTPS', h('div', {}, boton({ texto: 'Ver cómo renovarlo', variante: 'texto', chico: true })))),
      h('section', { class: 'pila', 'aria-labelledby': 'p-ultimos' }, h('div', { class: 'fila-entre' }, h('h2', { id: 'p-ultimos' }, 'Últimos trabajos'), h('a', { href: '#proto-trabajos', class: 'pequeno' }, 'Ver todos')),
        tablaTrabajos(TRABAJOS.slice(0, 4), abrirTrabajo)));
  }

  function protoImpresora(v) {
    const indicador = pasos(['Tipo de impresora', 'Conexión', 'Probar'], 0);
    const cuerpo = h('div', { class: 'pila-amplia' });
    const ir = (n) => { indicador.ir(n); vaciar(cuerpo, [pasoTipo, pasoConexion, pasoProbar][n]()); v.querySelector('h1')?.focus(); };
    const opcion = (valor, ic, titulo, texto, marcado) => h('label', { class: 'opcion' },
      h('input', { type: 'radio', name: 'tipo-imp', value: valor, checked: marcado }), icono(ic), h('span', { class: 'titulo-opcion' }, titulo), h('span', { class: 'tenue' }, texto));
    const pie = (atras, siguiente, texto = 'Continuar') => h('div', { class: 'fila-entre' },
      atras !== null ? boton({ texto: 'Atrás', icono: 'flecha-izq', onclick: () => ir(atras) }) : h('span'),
      boton({ texto, variante: 'primario', onclick: () => ir(siguiente) }));
    function pasoTipo() {
      return h('div', { class: 'pila-amplia' },
        h('fieldset', {}, h('legend', {}, '¿Qué impresora vas a agregar?'), h('div', { class: 'opciones' },
          opcion('80', 'impresora', 'De tickets de 80 mm', 'La más común en cajas y cocinas. Rollo estándar.', true),
          opcion('58', 'ticket', 'De tickets de 58 mm', 'Rollo angosto, como las portátiles o Bluetooth.'),
          opcion('virtual', 'virtual', 'Virtual, para probar', 'No necesita impresora: el ticket se ve en pantalla.'))),
        h('p', { class: 'tenue pequeno' }, '¿No sabes el ancho? Mide el rollo de papel: casi siempre es de 80 mm. Podrás cambiarlo después.'),
        pie(null, 1));
    }
    function pasoConexion() {
      const detectada = (ic, nombre, detalle, tickets) => h('label', { class: 'opcion' }, h('input', { type: 'radio', name: 'det', checked: nombre === 'EPSON TM-T20III' }),
        h('span', { class: 'fila' }, icono(ic), h('span', { class: 'titulo-opcion' }, nombre), !tickets && insignia('desconocido', 'No parece de tickets', { icono: 'info' })),
        h('span', { class: 'tenue' }, detalle));
      return h('div', { class: 'pila-amplia' },
        tarjeta({ titulo: 'Encontramos estas impresoras', subtitulo: 'Conectadas a este equipo o en tu red.', icono: 'buscar' },
          h('div', { class: 'opciones' },
            detectada('red', 'EPSON TM-T20III', 'En la red · 192.168.1.50', true),
            detectada('usb', 'POS-80C', 'Por USB · instalada en Windows', true),
            detectada('usb', 'Microsoft Print to PDF', 'Impresora de Windows para guardar PDF', false)),
          h('div', {}, boton({ texto: 'Buscar otra vez', icono: 'recargar', chico: true }))),
        h('details', { class: 'avanzado' }, h('summary', {}, '¿No aparece? Escribe la dirección'),
          h('div', {}, h('div', { class: 'campos' },
            campo({ id: 'p-tipo', etiqueta: 'Cómo está conectada', control: selector({ opciones: [['red', 'Por red (cable o wifi)'], ['usb', 'Por USB'], ['bluetooth', 'Por Bluetooth']], valor: 'red' }) }),
            campo({ id: 'p-dir', etiqueta: 'Dirección de la impresora', control: entrada({ mono: true, placeholder: 'p. ej. 192.168.1.50' }), ayuda: 'Aparece en el ticket de configuración que imprime la impresora al encenderla con el botón de avance presionado.' })))),
        h('div', { class: 'campos' },
          campo({ id: 'p-nombre', etiqueta: 'Nombre', control: entrada({ value: 'Caja principal' }), ayuda: 'Para reconocerla en el panel.' }),
          campo({ id: 'p-alias', etiqueta: 'Nombre corto para las aplicaciones (alias)', control: entrada({ mono: true, value: 'caja-principal' }), ayuda: 'Lo propusimos a partir del nombre. Puedes cambiarlo ahora; después ya no.',
            info: { texto: 'Las aplicaciones eligen la impresora por este nombre, nunca por su dirección.', enlace: { href: '#', texto: 'Saber más' } } })),
        pie(0, 2, 'Probar e imprimir'));
    }
    function pasoProbar() {
      const fila = (n, texto) => h('label', { class: 'opcion' }, h('input', { type: 'radio', name: 'acentos', checked: n === 4 }),
        h('span', { class: 'titulo-opcion' }, `Renglón ${n}`), h('span', { class: 'tenue mono' }, texto));
      return h('div', { class: 'pila-amplia' },
        nota({ tipo: 'ok', titulo: 'Se imprimió el ticket de prueba', texto: 'La impresora respondió en 0.4 s y confirmó que lo recibió.' }),
        h('div', { class: 'rejilla' },
          tarjeta({ titulo: 'Así debe verse', icono: 'ticket' }, ticketPapel({ imagen: ticketDeEjemplo(48), anchoMm: 80 })),
          tarjeta({ titulo: 'Revisa el papel', icono: 'check' },
            h('fieldset', {}, h('legend', {}, '¿Salió el ticket?'), h('div', { class: 'fila' }, boton({ texto: 'Sí, salió' }), boton({ texto: 'No salió', variante: 'texto' }))),
            h('fieldset', {}, h('legend', {}, 'Abajo se imprimieron 4 renglones numerados. ¿En cuál se leen bien á é í ó ú ñ ¿ €?'),
              h('div', { class: 'opciones' }, fila(1, 'áéíóú ñ ¿ ?'), fila(2, 'áéíóú ñ ¿ ?'), fila(3, 'áéíóú ñ ¿ €'), fila(4, 'áéíóú ñ ¿ €')),
              boton({ texto: 'En ninguno se ven bien', variante: 'texto', chico: true })))),
        pie(1, 2, 'Terminar'));
    }
    agregar(v, encabezado('Agregar impresora', 'Tres pasos: qué impresora es, cómo está conectada y una prueba en papel.', boton({ texto: 'Cancelar' })), indicador, cuerpo);
    v.querySelector('h1').tabIndex = -1;
    const paso = Number(new URLSearchParams(location.hash.split('?')[1] || '').get('paso') || 0);
    indicador.ir(paso);
    agregar(cuerpo, [pasoTipo, pasoConexion, pasoProbar][paso]());
  }

  function abrirTrabajo(t) {
    if (t.estado !== 'error') { aviso(`El trabajo ${t.ref} se imprimió bien.`, 'ok'); return; }
    dialogo({ titulo: ['El trabajo ', h('span', { class: 'ref' }, t.ref), ' no se imprimió'], ancho: 620, cuerpo: h('div', { class: 'pila-amplia' },
      h('div', { class: 'fila' }, insignia('error', 'Falló'), h('span', { class: 'tenue pequeno' }, `${t.hora} · ${t.app} · impresora `, h('span', { class: 'mono' }, t.imp))),
      h('div', { class: 'pila' }, h('h3', {}, 'Qué pasó'), h('p', {}, 'PRISMA intentó conectarse con la impresora «barra» durante 3 segundos y no respondió.')),
      h('div', { class: 'pila' }, h('h3', {}, 'Por qué suele pasar'), h('ul', {}, h('li', {}, 'La impresora está apagada o sin cable de red.'), h('li', {}, 'Cambió su dirección (por ejemplo, después de reiniciar el módem).'), h('li', {}, 'Otra computadora la está usando en este momento.'))),
      h('div', { class: 'pila' }, h('h3', {}, 'Qué hacer'), h('ol', {}, h('li', {}, 'Revisa que esté encendida y con papel.'), h('li', {}, 'Pulsa «Probar la impresora».'), h('li', {}, 'Si sigue sin responder, sigue la guía «Mi impresora no imprime».'))),
      h('p', { class: 'tenue pequeno' }, 'Tu aplicación puede reintentar este trabajo: como no se imprimió, no saldrá dos veces. Código: ', h('code', {}, 'IMPRESORA_INALCANZABLE')),
      h('div', { class: 'acciones' }, boton({ texto: 'Probar la impresora', variante: 'primario', icono: 'red' }), boton({ texto: 'Mi impresora no imprime', icono: 'diagrama' }), boton({ texto: 'Abrir el manual', variante: 'texto', icono: 'libro' }))) });
  }

  function protoTrabajos(v) {
    const filtroEstado = h('div', { class: 'segmentos', role: 'group', 'aria-label': 'Estado' }, ['Todos', 'Fallidos', 'Con aviso'].map((t, k) =>
      h('button', { type: 'button', 'aria-pressed': String(k === 0) }, t)));
    agregar(v,
      encabezado('Trabajos', 'Lo que se ha impreso. Por privacidad no se guarda el contenido de los tickets.'),
      h('div', { class: 'fila-entre' }, filtroEstado,
        h('div', { class: 'fila' }, h('label', { class: 'solo-lector', for: 'f-imp' }, 'Impresora'), selector({ id: 'f-imp', opciones: [['', 'Todas las impresoras'], ['caja', 'caja'], ['cocina', 'cocina'], ['barra', 'barra'], ['prueba', 'prueba (virtual)']] }))),
      nota({ tipo: 'error', titulo: '1 trabajo falló hoy', texto: 'La impresora «barra» no respondió. Toca el trabajo para ver qué hacer.' }),
      tablaTrabajos(TRABAJOS, abrirTrabajo),
      h('p', { class: 'tenue pequeno' }, '6 trabajos · se conservan 30 días'));
  }

  function protoPendiente(v, nombre) {
    agregar(v, encabezado(nombre, 'Esta pantalla no forma parte del prototipo.'),
      vacio({ icono: 'lapiz', titulo: 'Se rediseña en la Fase 3', texto: 'El prototipo muestra Inicio, Agregar impresora y Trabajos.', accion: boton({ texto: 'Ir a Inicio', onclick: () => { location.hash = '#proto-inicio'; } }) }));
  }

  // ---------------------------------------------------------------------------------------------
  // Enrutado
  // ---------------------------------------------------------------------------------------------
  const GALERIA = [
    { titulo: 'Sistema de diseño', secciones: [
      { id: 'fundamentos', texto: 'Fundamentos', icono: 'diagrama', vista: vistaFundamentos },
      { id: 'componentes', texto: 'Componentes', icono: 'aplicacion', vista: vistaComponentes },
      { id: 'formularios', texto: 'Formularios', icono: 'lapiz', vista: vistaFormularios },
      { id: 'estados', texto: 'Estados y avisos', icono: 'aviso', vista: vistaEstados },
      { id: 'contenido', texto: 'Diagramas y tickets', icono: 'ticket', vista: vistaContenido }] },
    { titulo: 'Prototipos', secciones: [
      { id: 'proto-inicio', texto: 'Inicio', icono: 'inicio' },
      { id: 'proto-impresora', texto: 'Agregar impresora', icono: 'impresora' },
      { id: 'proto-trabajos', texto: 'Trabajos', icono: 'trabajos' }] },
  ];
  const PROTOTIPOS = { 'proto-inicio': protoInicio, 'proto-impresora': protoImpresora, 'proto-trabajos': protoTrabajos };

  function pintar() {
    const id = location.hash.slice(1).split('?')[0] || 'fundamentos';
    const esProto = id.startsWith('proto-');
    const app = esProto
      ? marco({ marca: marcaPanel(), grupos: GRUPOS_PANEL, actual: id, inferior: ['proto-inicio', 'proto-impresora', 'proto-trabajos', 'proto-pruebas', 'proto-ayuda'],
        pie: [h('a', { href: '#fundamentos', class: 'pequeno' }, '← Volver a la galería'), selectorTema(), h('span', {}, 'Versión 0.2.0 (prototipo)')] })
      : marco({ marca: h('a', { class: 'marca', href: '#fundamentos' }, logo(30), h('span', { class: 'marca-texto' }, h('span', { class: 'marca-nombre' }, 'PRISMA'), h('span', { class: 'marca-puente' }, 'Sistema de diseño'))),
        grupos: GALERIA, actual: id, pie: [selectorTema(), h('span', {}, 'Solo en modo desarrollo')] });
    const vista = h('div', { class: 'vista' });
    if (esProto) vista.append(nota({ tipo: 'info', titulo: 'Prototipo con datos de ejemplo', texto: 'Así se verá esta pantalla con el sistema de diseño. Los botones no guardan nada.' }));
    app.principal.append(vista);
    vaciar(raiz, app);
    const seccion = GALERIA.flatMap((g) => g.secciones).find((s) => s.id === id);
    if (esProto) (PROTOTIPOS[id] || ((v) => protoPendiente(v, GRUPOS_PANEL.flatMap((g) => g.secciones).find((s) => s.id === id)?.texto || 'Pantalla')))(vista);
    else (seccion?.vista || vistaFundamentos)(vista);
    document.title = `${seccion?.texto || 'Prototipo'} · Estilos del panel PRISMA`;
    app.principal.focus({ preventScroll: true });
    window.scrollTo(0, 0);
  }
  window.addEventListener('hashchange', pintar);
  pintar();
})();
