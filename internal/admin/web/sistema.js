// PRISMA — sistema de diseño del panel: utilidades de DOM y componentes accesibles, sin dependencias.
// Lo usan el panel (panel.js) y la galería de componentes (estilos.js, solo en modo desarrollo).
// Se carga en <head> sin defer, después de textos.js, para aplicar el tema guardado antes de pintar.
// El texto se inserta siempre como texto, nunca como HTML, y los estilos dinámicos usan CSSOM
// (el.style), que la CSP del panel permite sin 'unsafe-inline'.
'use strict';
(() => {
  const T = window.PRISMA_TEXTOS.ui;
  const NS = 'http://www.w3.org/2000/svg';
  const ICONOS = '/admin/iconos.svg';

  // ---------------------------------------------------------------------------------------------
  // Tema: automático (prefers-color-scheme), claro u oscuro. La elección es una comodidad de cada
  // navegador, así que vive en localStorage; si no está disponible, el panel queda en automático.
  // ---------------------------------------------------------------------------------------------
  const CLAVE_TEMA = 'prisma.tema';
  function leerTema() {
    try { return localStorage.getItem(CLAVE_TEMA) || 'auto'; } catch { return 'auto'; }
  }
  function aplicarTema(tema) {
    const raiz = document.documentElement;
    if (tema === 'claro') raiz.dataset.theme = 'light';
    else if (tema === 'oscuro') raiz.dataset.theme = 'dark';
    else delete raiz.dataset.theme;
  }
  function guardarTema(tema) {
    try { localStorage.setItem(CLAVE_TEMA, tema); } catch { /* queda solo en esta página */ }
    aplicarTema(tema);
  }
  aplicarTema(leerTema());

  // ---------------------------------------------------------------------------------------------
  // DOM
  // ---------------------------------------------------------------------------------------------
  function h(etiqueta, props, ...hijos) {
    const el = document.createElement(etiqueta);
    for (const [k, v] of Object.entries(props || {})) {
      if (v === undefined || v === null || v === false) continue;
      if (k.startsWith('on') && typeof v === 'function') el.addEventListener(k.slice(2), v);
      else if (k === 'class') el.className = v;
      else if (k === 'value') el.value = v;
      else if (k === 'checked') el.checked = !!v;
      else if (v === true) el.setAttribute(k, '');
      else el.setAttribute(k, v);
    }
    agregar(el, hijos);
    return el;
  }
  function agregar(el, ...hijos) {
    for (const hijo of hijos.flat(Infinity)) {
      if (hijo === null || hijo === undefined || hijo === false) continue;
      el.append(hijo instanceof Node ? hijo : String(hijo));
    }
    return el;
  }
  function vaciar(el, ...hijos) { el.replaceChildren(); return agregar(el, hijos); }
  let siguienteId = 0;
  const nuevoId = (prefijo = 'ui') => `${prefijo}-${++siguienteId}`;

  // Icono del sprite. Sin etiqueta es decorativo (el texto de al lado ya lo dice).
  function icono(nombre, etiqueta) {
    const s = document.createElementNS(NS, 'svg');
    s.setAttribute('class', `icono i-${nombre}`);
    s.setAttribute('focusable', 'false');
    if (etiqueta) { s.setAttribute('role', 'img'); s.setAttribute('aria-label', etiqueta); } else s.setAttribute('aria-hidden', 'true');
    const u = document.createElementNS(NS, 'use');
    u.setAttribute('href', `${ICONOS}#i-${nombre}`);
    s.append(u);
    return s;
  }

  function logo(tam = 28) {
    // Un prisma que separa la luz en tres franjas: el documento entra y sale en tres conexiones.
    const s = document.createElementNS(NS, 'svg');
    s.setAttribute('viewBox', '0 0 32 32'); s.setAttribute('width', tam); s.setAttribute('height', tam);
    s.setAttribute('aria-hidden', 'true');
    const p = (d, color, ancho) => {
      const e = document.createElementNS(NS, 'path');
      e.setAttribute('d', d); e.setAttribute('fill', 'none'); e.setAttribute('stroke', color);
      e.setAttribute('stroke-width', ancho); e.setAttribute('stroke-linejoin', 'round'); e.setAttribute('stroke-linecap', 'round');
      s.append(e);
    };
    p('M3 13 L12 15', 'currentColor', 2);
    p('M12 5 L21 25 L3 25 Z', 'var(--acento)', 2.2);
    p('M16 15 L29 10', 'var(--acento)', 2);
    p('M16.8 16.8 L29 16.8', 'var(--ok)', 2);
    p('M17.6 18.6 L29 23.6', 'var(--aviso)', 2);
    return s;
  }

  // ---------------------------------------------------------------------------------------------
  // Botones
  // ---------------------------------------------------------------------------------------------
  // variante: secundario (por defecto), primario, peligro (contorno), peligro-lleno o texto.
  // Sin texto, etiqueta es obligatoria: es el nombre accesible del botón.
  function boton({ texto, icono: ic, variante = 'secundario', chico, tipo = 'button', etiqueta, ...resto } = {}) {
    const clases = ['boton', variante !== 'secundario' && variante, chico && 'chico', !texto && 'solo-icono'].filter(Boolean).join(' ');
    return h('button', { class: clases, type: tipo, 'aria-label': texto ? undefined : etiqueta, title: texto ? undefined : etiqueta, ...resto },
      ic && icono(ic), texto && h('span', {}, texto));
  }
  // Muestra un indicador en el botón mientras dura la acción y evita el doble clic.
  async function conEspera(b, accion) {
    const contenido = [...b.childNodes];
    b.disabled = true;
    b.setAttribute('aria-busy', 'true');
    vaciar(b, h('span', { class: 'girando', 'aria-hidden': 'true' }), contenido.map((n) => n.cloneNode(true)));
    try { return await accion(); } finally { b.disabled = false; b.removeAttribute('aria-busy'); vaciar(b, contenido); }
  }
  async function copiar(texto, b) {
    try {
      await navigator.clipboard.writeText(texto);
      if (b) {
        const antes = [...b.childNodes];
        vaciar(b, icono('check'), h('span', {}, T.copiado));
        setTimeout(() => vaciar(b, antes), 1600);
      }
    } catch {
      aviso(T.noSeCopio, 'aviso');
    }
  }
  function botonCopiar(texto, etiqueta = T.copiar) {
    const b = boton({ texto: etiqueta, icono: 'copiar', chico: true });
    b.addEventListener('click', () => copiar(typeof texto === 'function' ? texto() : texto, b));
    return b;
  }

  // ---------------------------------------------------------------------------------------------
  // Formularios
  // ---------------------------------------------------------------------------------------------
  function entrada({ tipo = 'text', mono, ...resto } = {}) {
    return h('input', { type: tipo, class: mono ? 'mono' : undefined, ...resto });
  }
  // opciones: [[valor, texto], …]
  function selector({ opciones, valor, ...resto }) {
    return h('select', resto, opciones.map(([v, t]) => h('option', { value: v, selected: String(v) === String(valor) }, t)));
  }

  // Explicación corta junto a un campo técnico, con enlace opcional al manual.
  function botonInfo(tema, { texto, enlace }) {
    const id = nuevoId('info');
    const panel = h('div', { class: 'info-panel', id, hidden: true }, h('p', {}, texto),
      enlace && h('a', { href: enlace.href }, enlace.texto || T.saberMas, icono('derecha')));
    const b = h('button', { type: 'button', class: 'info-boton', 'aria-expanded': 'false', 'aria-controls': id, 'aria-label': `${T.masInformacion} ${tema}` }, icono('info'));
    b.addEventListener('click', () => {
      const abierto = panel.hidden;
      panel.hidden = !abierto;
      b.setAttribute('aria-expanded', String(abierto));
    });
    return { boton: b, panel };
  }

  // Campo con etiqueta, ayuda, explicación opcional y error en línea (aria-invalid + aria-describedby).
  // validar(valor) devuelve el mensaje de error o ''; se revisa al salir del campo y, si ya mostró un
  // error, mientras se escribe.
  function campo({ id, etiqueta, control, ayuda, info, opcional, validar }) {
    control.id = id;
    const idAyuda = `${id}-ayuda`, idError = `${id}-error`;
    const textoError = h('span');
    const error = h('span', { class: 'campo-error', id: idError, hidden: true }, icono('error'), textoError);
    const extra = info && botonInfo(etiqueta, info);
    const describir = () => {
      const ids = [ayuda && idAyuda, !error.hidden && idError].filter(Boolean).join(' ');
      if (ids) control.setAttribute('aria-describedby', ids); else control.removeAttribute('aria-describedby');
    };
    const el = h('div', { class: 'campo' },
      h('div', { class: 'campo-cabeza' }, h('label', { for: id }, etiqueta, opcional && h('span', { class: 'campo-opcional' }, ` (${T.opcional})`)), extra && extra.boton),
      extra && extra.panel, control, ayuda && h('span', { class: 'ayuda', id: idAyuda }, ayuda), error);
    el.ponerError = (mensaje) => {
      if (!mensaje) return el.quitarError();
      textoError.textContent = mensaje; error.hidden = false;
      control.setAttribute('aria-invalid', 'true'); describir();
    };
    el.quitarError = () => { error.hidden = true; control.removeAttribute('aria-invalid'); describir(); };
    el.revisar = () => { const m = validar ? validar(control.value) : ''; el.ponerError(m); return !m; };
    if (validar) {
      control.addEventListener('blur', el.revisar);
      control.addEventListener('input', () => { if (!error.hidden) el.revisar(); });
    }
    describir();
    return el;
  }
  function interruptor({ id, texto, ayuda, marcado, ...resto }) {
    const input = h('input', { type: 'checkbox', role: 'switch', id, checked: marcado, ...resto });
    const idAyuda = ayuda && `${id}-ayuda`;
    if (idAyuda) input.setAttribute('aria-describedby', idAyuda);
    const el = h('label', { class: 'interruptor', for: id }, input, h('span', { class: 'interruptor-pista', 'aria-hidden': 'true' }),
      h('span', { class: 'interruptor-texto' }, texto, ayuda && h('span', { class: 'ayuda', id: idAyuda }, ayuda)));
    el.input = input;
    return el;
  }
  function casilla({ id, texto, marcado, ...resto }) {
    const input = h('input', { type: 'checkbox', id, checked: marcado, ...resto });
    const el = h('label', { class: 'casilla', for: id }, input, h('span', {}, texto));
    el.input = input;
    return el;
  }
  // Primer campo con error: recibe el foco. Devuelve true si todos son válidos.
  function revisarCampos(campos) {
    let primero = null;
    for (const c of campos) if (!c.revisar() && !primero) primero = c;
    if (primero) primero.querySelector('input, select, textarea')?.focus();
    return !primero;
  }

  // ---------------------------------------------------------------------------------------------
  // Contenedores e indicadores
  // ---------------------------------------------------------------------------------------------
  function tarjeta({ titulo, subtitulo, icono: ic, acciones, nivel = 2, clase, etiquetaId }, ...contenido) {
    const id = etiquetaId || (titulo ? nuevoId('t') : undefined);
    return h('section', { class: ['tarjeta', clase].filter(Boolean).join(' '), 'aria-labelledby': id },
      (titulo || acciones) && h('div', { class: 'tarjeta-cabeza' },
        h('div', { class: 'tarjeta-titulo' }, ic && icono(ic), h('div', {}, titulo && h(`h${nivel}`, { id }, titulo), subtitulo && h('p', { class: 'tenue pequeno' }, subtitulo))),
        acciones && h('div', { class: 'acciones' }, acciones)),
      contenido);
  }

  const ICONO_ESTADO = { ok: 'ok', aviso: 'aviso', error: 'error', info: 'info', desconocido: 'desconocido', virtual: 'virtual', acento: 'reloj' };
  // Estado con color, icono y texto: nunca solo color.
  function insignia(estado, texto, { icono: ic } = {}) {
    return h('span', { class: `insignia estado-${estado}` }, icono(ic || ICONO_ESTADO[estado] || 'info'), h('span', {}, texto || T.estados[estado] || estado));
  }
  // Aviso en la página (no flotante). tipo: info, ok, aviso o error.
  function nota({ tipo = 'info', titulo, texto, acciones, cerrable }, ...contenido) {
    const el = h('div', { class: `nota-caja estado-${tipo}`, role: tipo === 'error' ? 'alert' : 'status' },
      icono(ICONO_ESTADO[tipo] || 'info'),
      h('div', { class: 'nota-cuerpo' }, titulo && h('strong', {}, titulo), texto && h('p', {}, texto), contenido,
        acciones && h('div', { class: 'acciones' }, acciones)),
      cerrable && boton({ icono: 'cerrar', variante: 'texto', chico: true, etiqueta: T.cerrarAviso, onclick: () => el.remove() }));
    return el;
  }
  function vacio({ icono: ic = 'info', titulo, texto, accion }) {
    return h('div', { class: 'vacio-estado' }, h('span', { class: 'vacio-icono' }, icono(ic)),
      h('p', { class: 'vacio-titulo' }, titulo), texto && h('p', { class: 'tenue' }, texto), accion && h('div', { class: 'acciones' }, accion));
  }
  // Marcador de carga con la forma aproximada del contenido (no hay saltos al llegar los datos).
  function esqueleto({ lineas = 3, alto } = {}) {
    const el = h('div', { class: 'esqueleto', role: 'status' }, h('span', { class: 'solo-lector' }, T.cargando),
      Array.from({ length: lineas }, (_, k) => h('span', { class: `esqueleto-linea${k === lineas - 1 ? ' corta' : ''}`, 'aria-hidden': 'true' })));
    if (alto) el.style.minHeight = `${alto}px`;
    return el;
  }

  // Indicador de pasos de un asistente. pasos: textos; actual: índice (desde 0).
  function pasos(lista, actual = 0) {
    const ol = h('ol', { class: 'pasos-asistente', 'aria-label': T.pasos });
    const pintar = (n) => {
      vaciar(ol, lista.map((texto, k) => h('li', { class: k < n ? 'hecho' : (k === n ? 'actual' : ''), 'aria-current': k === n ? 'step' : false },
        h('span', { class: 'paso-marca', 'aria-hidden': 'true' }, k < n ? icono('check') : String(k + 1)),
        h('span', { class: 'paso-texto' }, texto, k < n && h('span', { class: 'solo-lector' }, ` (${T.completado})`)))));
      ol.dataset.actual = n;
    };
    pintar(actual);
    ol.ir = pintar;
    return ol;
  }

  // Pestañas con el patrón ARIA: flechas, Inicio y Fin; solo la pestaña activa recibe el foco con Tab.
  // lista: [{ id, titulo, contenido: Node | () => Node }]
  function pestanas(lista, { etiqueta, inicial, alCambiar } = {}) {
    const base = nuevoId('pst');
    const barra = h('div', { class: 'pestanas', role: 'tablist', 'aria-label': etiqueta });
    const paneles = h('div', { class: 'pestanas-paneles' });
    const botones = lista.map((p, k) => {
      const panel = h('div', { role: 'tabpanel', id: `${base}-p${k}`, 'aria-labelledby': `${base}-t${k}`, tabindex: '0', hidden: true });
      paneles.append(panel);
      const b = h('button', { type: 'button', role: 'tab', id: `${base}-t${k}`, 'aria-controls': panel.id, 'aria-selected': 'false', tabindex: '-1' }, p.titulo);
      b.addEventListener('click', () => elegir(k));
      b.addEventListener('keydown', (ev) => {
        const destino = { ArrowRight: k + 1, ArrowLeft: k - 1, Home: 0, End: lista.length - 1 }[ev.key];
        if (destino === undefined) return;
        ev.preventDefault();
        const n = (destino + lista.length) % lista.length;
        elegir(n); botones[n].focus();
      });
      return { b, panel };
    });
    let actual = -1;
    function elegir(k) {
      if (k === actual) return;
      botones.forEach(({ b, panel }, i) => {
        const activa = i === k;
        b.setAttribute('aria-selected', String(activa)); b.tabIndex = activa ? 0 : -1; panel.hidden = !activa;
        if (activa && !panel.firstChild) agregar(panel, typeof lista[i].contenido === 'function' ? lista[i].contenido() : lista[i].contenido);
      });
      actual = k;
      if (alCambiar) alCambiar(lista[k].id);
    }
    agregar(barra, botones.map((x) => x.b));
    const el = h('div', { class: 'pestanas-caja' }, barra, paneles);
    el.elegir = (id) => elegir(Math.max(0, lista.findIndex((p) => p.id === id)));
    el.activa = () => lista[actual].id;
    elegir(Math.max(0, lista.findIndex((p) => p.id === inicial)));
    return el;
  }

  // Bloque de código desplazable y operable con teclado, con botón Copiar. variantes: { nombre: texto }.
  function codigo(variantes, { etiqueta = T.codigo } = {}) {
    const bloque = (texto) => h('pre', { class: 'codigo', tabindex: '0', 'aria-label': etiqueta }, h('code', {}, texto));
    const nombres = Object.keys(variantes);
    if (nombres.length === 1) {
      const pre = bloque(variantes[nombres[0]]);
      return h('div', { class: 'codigo-caja' }, h('div', { class: 'codigo-barra' }, h('span', { class: 'tenue pequeno' }, nombres[0]), botonCopiar(() => pre.textContent, T.copiarCodigo)), pre);
    }
    let caja;
    const copiarActual = () => variantes[caja.activa()];
    caja = pestanas(nombres.map((n) => ({ id: n, titulo: n, contenido: () => bloque(variantes[n]) })), { etiqueta });
    caja.classList.add('codigo-caja');
    caja.querySelector('[role=tablist]').after(h('div', { class: 'codigo-barra' }, botonCopiar(copiarActual, T.copiarCodigo)));
    return caja;
  }

  // Tabla que en pantallas angostas se vuelve una lista de tarjetas (cada celda muestra su columna).
  // columnas: [{ titulo, celda: (fila) => Node | texto, clase }]; alAbrir(fila) hace cada fila abrible.
  function tabla({ columnas, filas, vacio: siVacio, alAbrir, nombreFila = (f) => '', etiqueta }) {
    if (filas.length === 0) return siVacio || vacio({ titulo: T.sinResultados });
    const cuerpo = filas.map((f) => {
      const tr = h('tr', { class: alAbrir ? 'abrible' : undefined }, columnas.map((c, k) => {
        let contenido = c.celda(f);
        if (alAbrir && k === 0) contenido = h('button', { type: 'button', class: 'fila-abrir', 'aria-label': T.abrirDetalle(nombreFila(f)), onclick: () => alAbrir(f) }, contenido);
        return h('td', { class: c.clase, 'data-etiqueta': c.titulo }, contenido);
      }));
      if (alAbrir) tr.addEventListener('click', (ev) => { if (!ev.target.closest('button, a, input, select')) alAbrir(f); });
      return tr;
    });
    return h('div', { class: 'tabla-caja' }, h('table', { class: 'tabla-responsiva', 'aria-label': etiqueta },
      h('thead', {}, h('tr', {}, columnas.map((c) => h('th', { scope: 'col', class: c.clase }, c.titulo)))),
      h('tbody', {}, cuerpo)));
  }

  // ---------------------------------------------------------------------------------------------
  // Avisos flotantes y diálogos
  // ---------------------------------------------------------------------------------------------
  let contAvisos = null;
  function contenedorAvisos() {
    if (!contAvisos || !contAvisos.isConnected) {
      contAvisos = h('div', { class: 'avisos', 'aria-live': 'polite' });
      document.body.append(contAvisos);
    }
    return contAvisos;
  }
  // tipo: '' (información), ok, aviso o error. Los errores duran más; persistente no se cierra solo.
  function aviso(texto, tipo = '', { persistente } = {}) {
    const t = h('div', { class: `toast ${tipo}`, role: tipo === 'error' ? 'alert' : 'status' },
      icono(ICONO_ESTADO[tipo] || 'info'), h('span', { class: 'toast-texto' }, texto),
      boton({ icono: 'cerrar', variante: 'texto', chico: true, etiqueta: T.cerrarAviso, onclick: () => t.remove() }));
    contenedorAvisos().append(t);
    if (!persistente) setTimeout(() => t.remove(), tipo === 'error' ? 9000 : 5000);
    return t;
  }
  function avisoError(e) {
    aviso(e.sugerencia ? `${e.message} ${e.sugerencia}` : e.message, 'error');
  }

  function dialogo({ titulo, cuerpo, acciones = [{ texto: T.cerrar, valor: null }], ancho }) {
    return new Promise((resolver) => {
      const idTitulo = nuevoId('dlg');
      const d = h('dialog', { 'aria-labelledby': idTitulo });
      if (ancho) d.style.width = `min(${ancho}px, calc(100vw - 32px))`;
      const cerrar = (valor) => { d.close(); d.remove(); resolver(valor); };
      const pie = h('div', { class: 'dialogo-pie' }, acciones.map((a) =>
        h('button', { class: `boton ${a.clase || ''}`, type: 'button', onclick: async (ev) => {
          if (a.antes) {
            const b = ev.currentTarget;
            b.disabled = true;
            try { if ((await a.antes()) === false) { b.disabled = false; return; } } catch (e) { avisoError(e); b.disabled = false; return; }
          }
          cerrar(a.valor);
        } }, a.texto)));
      d.append(h('div', { class: 'dialogo' },
        h('div', { class: 'dialogo-cabeza' }, h('h2', { id: idTitulo }, titulo),
          boton({ icono: 'cerrar', variante: 'texto', chico: true, etiqueta: T.cerrar, onclick: () => cerrar(null) })),
        cuerpo, pie));
      d.addEventListener('cancel', (ev) => { ev.preventDefault(); cerrar(null); });
      document.body.append(d);
      d.showModal();
    });
  }
  // Confirmación: dice qué pasará (consecuencias) antes de una acción que no se puede deshacer.
  function confirmar(titulo, texto, { boton: textoBoton = T.confirmar, peligro = false, consecuencias } = {}) {
    return dialogo({
      titulo,
      cuerpo: h('div', { class: 'pila' }, h('p', {}, texto),
        consecuencias && consecuencias.length > 0 && h('div', { class: 'consecuencias' }, h('p', { class: 'pequeno' }, h('strong', {}, T.consecuencias)),
          h('ul', {}, consecuencias.map((c) => h('li', {}, c))))),
      acciones: [{ texto: T.cancelar, valor: false }, { texto: textoBoton, valor: true, clase: peligro ? 'peligro-lleno' : 'primario' }],
    });
  }

  // ---------------------------------------------------------------------------------------------
  // Navegación
  // ---------------------------------------------------------------------------------------------
  function selectorTema() {
    const opciones = [['auto', 'auto', T.tema.auto], ['claro', 'sol', T.tema.claro], ['oscuro', 'luna', T.tema.oscuro]];
    const grupo = h('div', { class: 'segmentos tema', role: 'group', 'aria-label': T.tema.etiqueta });
    const pintar = () => {
      const actual = leerTema();
      for (const b of grupo.children) b.setAttribute('aria-pressed', String(b.dataset.tema === actual));
    };
    agregar(grupo, opciones.map(([valor, ic, texto]) => {
      const b = boton({ icono: ic, etiqueta: texto, chico: true });
      b.dataset.tema = valor;
      b.addEventListener('click', () => { guardarTema(valor); pintar(); });
      return b;
    }));
    pintar();
    return grupo;
  }

  // Estructura de la aplicación: barra lateral en escritorio; en celular, barra superior con menú
  // desplegable y barra inferior con los accesos más usados.
  // grupos: [{ titulo, secciones: [{ id, texto, textoCorto, icono, contador, href }] }]; inferior: ids
  // (en la barra inferior se usa textoCorto, si lo hay, para que quepa en un renglón).
  function marco({ marca, grupos, actual, pie, inferior = [] }) {
    const idNav = nuevoId('nav');
    const nav = h('nav', { class: 'nav', id: idNav, 'aria-label': T.navegacion });
    const todas = grupos.flatMap((g) => g.secciones);
    const enlace = (s, corto) => h('a', { href: s.href || `#${s.id}`, 'aria-current': s.id === actual ? 'page' : false },
      icono(s.icono), h('span', { class: 'nav-texto' }, corto && s.textoCorto ? s.textoCorto : s.texto),
      s.contador > 0 && h('span', { class: 'contador', 'aria-label': s.etiquetaContador || String(s.contador) }, String(s.contador)));
    agregar(nav, grupos.map((g) => h('div', { class: 'nav-grupo' }, g.titulo && h('p', { class: 'nav-titulo', 'aria-hidden': 'true' }, g.titulo),
      h('ul', { 'aria-label': g.titulo }, g.secciones.map((s) => h('li', {}, enlace(s)))))));
    const menu = boton({ texto: T.menu, icono: 'menu', chico: true, class: 'boton chico menu-boton', 'aria-expanded': 'false', 'aria-controls': idNav });
    const barra = h('header', { class: 'barra' }, marca, menu, nav, h('div', { class: 'barra-pie' }, pie));
    const cerrarMenu = () => { barra.classList.remove('abierta'); menu.setAttribute('aria-expanded', 'false'); };
    menu.addEventListener('click', () => {
      const abierta = barra.classList.toggle('abierta');
      menu.setAttribute('aria-expanded', String(abierta));
    });
    barra.addEventListener('keydown', (ev) => { if (ev.key === 'Escape' && barra.classList.contains('abierta')) { cerrarMenu(); menu.focus(); } });
    nav.addEventListener('click', (ev) => { if (ev.target.closest('a')) cerrarMenu(); });
    const principal = h('main', { class: 'contenido', id: 'principal', tabindex: '-1' });
    const abajo = inferior.length > 0 && h('nav', { class: 'nav-inferior', 'aria-label': T.navegacionInferior },
      inferior.map((id) => todas.find((s) => s.id === id)).filter(Boolean).map((s) => enlace(s, true)));
    const el = h('div', { class: 'app' }, h('a', { class: 'saltar', href: '#principal' }, T.saltar), barra, principal, abajo);
    el.principal = principal;
    el.marcar = (id) => {
      for (const a of el.querySelectorAll('.nav a, .nav-inferior a')) {
        if (a.getAttribute('href') === `#${id}`) a.setAttribute('aria-current', 'page'); else a.removeAttribute('aria-current');
      }
    };
    return el;
  }

  // ---------------------------------------------------------------------------------------------
  // Visor de diagramas (base; la Fase 5 agrega metadatos, variante oscura y galería) y vista de
  // ticket en papel térmico (base; la Fase 6 le da la imagen real de la impresora virtual).
  // ---------------------------------------------------------------------------------------------
  function visorDiagrama({ src, alt, titulo, descripcion, mermaid }) {
    const idDesc = nuevoId('vd');
    let escala = 1, x = 0, y = 0;
    const img = h('img', { src, alt, draggable: 'false' });
    const lienzo = h('div', { class: 'visor-lienzo', tabindex: '0', role: 'group', 'aria-label': titulo, 'aria-describedby': idDesc }, img);
    const aplicar = () => { img.style.transform = `translate(${x}px, ${y}px) scale(${escala})`; };
    const zoom = (f) => { escala = Math.min(6, Math.max(0.4, escala * f)); aplicar(); };
    const restablecer = () => { escala = 1; x = 0; y = 0; aplicar(); };
    const grande = boton({ icono: 'expandir', etiqueta: T.visor.grande, chico: true, 'aria-pressed': 'false' });
    const fig = h('figure', { class: 'visor' },
      h('div', { class: 'visor-barra', role: 'toolbar', 'aria-label': T.visor.herramientas },
        boton({ icono: 'acercar', etiqueta: T.visor.acercar, chico: true, onclick: () => zoom(1.25) }),
        boton({ icono: 'alejar', etiqueta: T.visor.alejar, chico: true, onclick: () => zoom(0.8) }),
        boton({ icono: 'recargar', etiqueta: T.visor.restablecer, chico: true, onclick: restablecer }),
        grande,
        h('a', { class: 'boton chico solo-icono', href: src, download: '', 'aria-label': T.visor.descargar, title: T.visor.descargar }, icono('descargar')),
        mermaid && botonCopiar(mermaid, T.visor.copiarMermaid)),
      lienzo,
      h('figcaption', {}, h('strong', {}, titulo), h('p', { id: idDesc, class: 'tenue pequeno' }, descripcion, ' ', h('span', { class: 'solo-lector' }, T.visor.ayudaTeclado))));
    grande.addEventListener('click', () => {
      const activo = fig.classList.toggle('visor-grande');
      grande.setAttribute('aria-pressed', String(activo));
      grande.setAttribute('aria-label', activo ? T.visor.salirGrande : T.visor.grande);
      restablecer();
    });
    lienzo.addEventListener('keydown', (ev) => {
      const paso = 40;
      const accion = { '+': () => zoom(1.25), '=': () => zoom(1.25), '-': () => zoom(0.8), 0: restablecer,
        ArrowLeft: () => { x += paso; aplicar(); }, ArrowRight: () => { x -= paso; aplicar(); },
        ArrowUp: () => { y += paso; aplicar(); }, ArrowDown: () => { y -= paso; aplicar(); },
        Escape: () => { if (fig.classList.contains('visor-grande')) grande.click(); } }[ev.key];
      if (accion) { ev.preventDefault(); accion(); }
    });
    // Arrastrar para mover y pellizcar (dos dedos) para acercar.
    const punteros = new Map();
    let distancia = 0;
    lienzo.addEventListener('pointerdown', (ev) => { lienzo.setPointerCapture(ev.pointerId); punteros.set(ev.pointerId, [ev.clientX, ev.clientY]); });
    lienzo.addEventListener('pointermove', (ev) => {
      const previo = punteros.get(ev.pointerId);
      if (!previo) return;
      punteros.set(ev.pointerId, [ev.clientX, ev.clientY]);
      if (punteros.size === 2) {
        const [[ax, ay], [bx, by]] = [...punteros.values()];
        const d = Math.hypot(ax - bx, ay - by);
        if (distancia) zoom(d / distancia);
        distancia = d;
      } else { x += ev.clientX - previo[0]; y += ev.clientY - previo[1]; aplicar(); }
    });
    const soltar = (ev) => { punteros.delete(ev.pointerId); distancia = 0; };
    lienzo.addEventListener('pointerup', soltar);
    lienzo.addEventListener('pointercancel', soltar);
    lienzo.addEventListener('wheel', (ev) => { if (ev.ctrlKey) { ev.preventDefault(); zoom(ev.deltaY < 0 ? 1.1 : 0.9); } }, { passive: false });
    img.addEventListener('error', () => vaciar(lienzo, h('p', { class: 'tenue pequeno' }, T.visor.sinImagen), h('pre', { class: 'codigo', tabindex: '0' }, mermaid || '')));
    return fig;
  }

  // imagen: URL (normalmente data:image/png) del ticket rasterizado al ancho de la impresora.
  function ticketPapel({ imagen, anchoMm = 80, anchoPx, alt }) {
    const img = h('img', { src: imagen, alt: alt || T.ticket.etiqueta(anchoMm) });
    if (anchoPx) img.style.width = `${anchoPx}px`;
    return h('figure', { class: `ticket ancho-${anchoMm}` }, h('div', { class: 'ticket-papel' }, img),
      h('figcaption', { class: 'tenue pequeno' }, T.ticket.papel(anchoMm)));
  }

  window.PrismaUI = {
    h, agregar, vaciar, nuevoId, icono, logo,
    boton, conEspera, copiar, botonCopiar,
    entrada, selector, campo, interruptor, casilla, revisarCampos,
    tarjeta, insignia, nota, vacio, esqueleto, pasos, pestanas, codigo, tabla,
    aviso, avisoError, dialogo, confirmar,
    selectorTema, marco, visorDiagrama, ticketPapel,
    leerTema, guardarTema,
  };
})();
