// PRISMA — renderizador del manual (ADR 0017). Convierte el Markdown de docs/ en nodos DOM con
// createElement y textContent: nunca usa innerHTML, así que no hay HTML que sanear. Admite el
// subconjunto que usa docs/ (encabezados, párrafos, listas anidadas, tablas, código, citas, líneas,
// enlaces, imágenes, negritas, cursivas y código en línea); TestSintaxisDelManual (docs) falla si un
// documento del índice usa otra cosa.
(function () {
  'use strict';
  const { h } = window.PrismaUI;

  // Ancla de un título, como GitHub: minúsculas, sin puntuación salvo - y _, espacios como guiones.
  function anclaDe(texto) {
    return texto.trim().toLowerCase().replace(/[^\p{L}\p{N}\s_-]/gu, '').replace(/\s/g, '-');
  }

  // ---------------------------------------------------------------------------------------------
  // En línea
  // ---------------------------------------------------------------------------------------------
  // Busca el cierre de [texto](destino) a partir de la posición del «[».
  function enlaceEn(s, i) {
    let nivel = 0;
    for (let j = i; j < s.length; j++) {
      if (s[j] === '\\') { j++; continue; }
      if (s[j] === '`') { const k = s.indexOf('`', j + 1); if (k < 0) return null; j = k; continue; }
      if (s[j] === '[') nivel++;
      else if (s[j] === ']' && --nivel === 0) {
        if (s[j + 1] !== '(') return null;
        const k = s.indexOf(')', j + 2);
        if (k < 0) return null;
        return { texto: s.slice(i + 1, j), destino: s.slice(j + 2, k).trim(), fin: k + 1 };
      }
    }
    return null;
  }

  function enLinea(s, ctx) {
    const nodos = [];
    let texto = '';
    const soltar = () => { if (texto) { nodos.push(document.createTextNode(texto)); texto = ''; } };
    for (let i = 0; i < s.length; i++) {
      const c = s[i];
      if (c === '\\' && i + 1 < s.length && /[\\`*_[\]()#+\-.!|<>{}]/.test(s[i + 1])) { texto += s[++i]; continue; }
      if (c === '`') {
        let n = 1;
        while (s[i + n] === '`') n++;
        const cierre = s.indexOf('`'.repeat(n), i + n);
        if (cierre > 0) {
          soltar();
          nodos.push(h('code', {}, s.slice(i + n, cierre).replace(/^ (.*) $/, '$1')));
          i = cierre + n - 1;
          continue;
        }
      }
      if (c === '!' && s[i + 1] === '[') {
        const e = enlaceEn(s, i + 1);
        if (e) { soltar(); nodos.push(ctx.imagen(e.destino, e.texto)); i = e.fin - 1; continue; }
      }
      if (c === '[') {
        const e = enlaceEn(s, i);
        if (e) { soltar(); nodos.push(ctx.enlace(e.destino, enLinea(e.texto, ctx))); i = e.fin - 1; continue; }
      }
      if (c === '*' && s[i + 1] === '*') {
        const cierre = s.indexOf('**', i + 2);
        if (cierre > i + 2) { soltar(); nodos.push(h('strong', {}, enLinea(s.slice(i + 2, cierre), ctx))); i = cierre + 1; continue; }
      }
      if (c === '*' && s[i + 1] !== ' ' && s[i + 1] !== '*') {
        const cierre = s.indexOf('*', i + 1);
        if (cierre > i + 1 && s[cierre - 1] !== ' ') { soltar(); nodos.push(h('em', {}, enLinea(s.slice(i + 1, cierre), ctx))); i = cierre; continue; }
      }
      texto += c;
    }
    soltar();
    return nodos;
  }

  // ---------------------------------------------------------------------------------------------
  // Bloques
  // ---------------------------------------------------------------------------------------------
  const RE = {
    cerca: /^(\s*)(`{3,})\s*([\w-]*)\s*$/,
    titulo: /^(#{1,6})\s+(.*?)\s*#*\s*$/,
    linea: /^ {0,3}(-{3,}|\*{3,})\s*$/,
    lista: /^( *)([-*]|\d+\.)\s+(.*)$/,
    cita: /^ {0,3}>\s?(.*)$/,
    separadorTabla: /^\s*\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)*\|?\s*$/,
  };
  const sangria = (l) => l.match(/^ */)[0].length;
  const vacia = (l) => !l.trim();
  const iniciaBloque = (l) => RE.cerca.test(l) || RE.titulo.test(l) || RE.linea.test(l) || RE.lista.test(l) || RE.cita.test(l) || /^\s*\|/.test(l);

  // Celdas de una fila de tabla: separa por | sin escapar y fuera del código en línea.
  function celdas(fila) {
    const s = fila.trim().replace(/^\|/, '').replace(/\|$/, '');
    const salida = [];
    let actual = '';
    let enCodigo = false;
    for (let i = 0; i < s.length; i++) {
      if (s[i] === '\\' && s[i + 1] === '|') { actual += '|'; i++; continue; }
      if (s[i] === '`') enCodigo = !enCodigo;
      if (s[i] === '|' && !enCodigo) { salida.push(actual.trim()); actual = ''; continue; }
      actual += s[i];
    }
    salida.push(actual.trim());
    return salida;
  }

  function tabla(lineas, ctx) {
    const alineaciones = celdas(lineas[1]).map((c) => (/^:-+:$/.test(c) ? 'center' : /-+:$/.test(c) ? 'right' : ''));
    const celda = (etiqueta, texto, k) => {
      const el = h(etiqueta, { scope: etiqueta === 'th' ? 'col' : undefined }, enLinea(texto, ctx));
      if (alineaciones[k]) el.style.textAlign = alineaciones[k];
      return el;
    };
    return h('div', { class: 'tabla-manual', tabindex: '0', role: 'region', 'aria-label': ctx.textos.tabla },
      h('table', {},
        h('thead', {}, h('tr', {}, celdas(lineas[0]).map((c, k) => celda('th', c, k)))),
        h('tbody', {}, lineas.slice(2).map((f) => h('tr', {}, celdas(f).map((c, k) => celda('td', c, k)))))));
  }

  function codigo(lenguaje, lineas, ctx) {
    const texto = lineas.join('\n');
    const pre = h('pre', { class: 'codigo', tabindex: '0', 'aria-label': lenguaje === 'mermaid' ? ctx.textos.diagrama : ctx.textos.codigo },
      h('code', { class: lenguaje ? `lenguaje-${lenguaje}` : undefined }, texto));
    return ctx.codigo ? ctx.codigo(lenguaje, texto, pre) : pre;
  }

  // Una lista empieza en lineas[i]; devuelve el nodo y la línea siguiente al final de la lista.
  function lista(lineas, i, ctx) {
    const primera = RE.lista.exec(lineas[i]);
    const base = primera[1].length;
    const ordenada = /\d/.test(primera[2]);
    const items = [];
    let ajustada = true;
    while (i < lineas.length) {
      const m = RE.lista.exec(lineas[i]);
      if (!m || m[1].length !== base || /\d/.test(m[2]) !== ordenada) break;
      const contenido = base + m[2].length + 1 + (m[3].length - m[3].trimStart().length);
      const cuerpo = [m[3]];
      i++;
      let vaciaAntes = false;
      while (i < lineas.length) {
        const l = lineas[i];
        if (vacia(l)) { cuerpo.push(''); vaciaAntes = true; i++; continue; }
        if (sangria(l) >= Math.min(contenido, base + 2)) {
          if (vaciaAntes) ajustada = false;
          cuerpo.push(l.slice(Math.min(sangria(l), contenido)));
          vaciaAntes = false;
          i++;
          continue;
        }
        // Continuación perezosa de un párrafo.
        if (!vaciaAntes && !iniciaBloque(l)) { cuerpo.push(l.trim()); i++; continue; }
        break;
      }
      while (cuerpo.length && vacia(cuerpo[cuerpo.length - 1])) cuerpo.pop();
      items.push(cuerpo);
      // Una línea vacía entre ítems hace la lista «suelta» (con párrafos).
      if (vaciaAntes && i < lineas.length && RE.lista.test(lineas[i]) && sangria(lineas[i]) === base) ajustada = false;
    }
    const el = h(ordenada ? 'ol' : 'ul', {}, items.map((cuerpo) => {
      const hijos = bloques(cuerpo, ctx);
      // En una lista ajustada, el párrafo de cada ítem se dibuja sin <p>.
      if (ajustada && hijos[0] && hijos[0].tagName === 'P') hijos.splice(0, 1, ...hijos[0].childNodes);
      return h('li', {}, hijos);
    }));
    if (ordenada) {
      const inicio = parseInt(primera[2], 10);
      if (inicio !== 1) el.setAttribute('start', String(inicio));
    }
    return { el, i };
  }

  function bloques(lineas, ctx) {
    const salida = [];
    let i = 0;
    while (i < lineas.length) {
      const l = lineas[i];
      if (vacia(l)) { i++; continue; }
      let m = RE.cerca.exec(l);
      if (m) {
        const sangriaCerca = m[1].length;
        const marca = m[2];
        const cuerpo = [];
        i++;
        while (i < lineas.length && !(lineas[i].trim().startsWith(marca) && !lineas[i].trim().slice(marca.length).trim())) {
          cuerpo.push(lineas[i].slice(Math.min(sangria(lineas[i]), sangriaCerca)));
          i++;
        }
        i++;
        salida.push(codigo(m[3], cuerpo, ctx));
        continue;
      }
      m = RE.titulo.exec(l);
      if (m) {
        const nivel = m[1].length;
        const el = h(`h${nivel}`, {}, enLinea(m[2], ctx));
        el.id = ctx.idTitulo(el.textContent);
        ctx.titulos.push({ nivel, texto: el.textContent, id: el.id });
        salida.push(el);
        i++;
        continue;
      }
      if (RE.linea.test(l)) { salida.push(h('hr')); i++; continue; }
      if (/^\s*\|/.test(l) && i + 1 < lineas.length && RE.separadorTabla.test(lineas[i + 1])) {
        const filas = [];
        while (i < lineas.length && /^\s*\|/.test(lineas[i])) filas.push(lineas[i++]);
        salida.push(tabla(filas, ctx));
        continue;
      }
      if (RE.cita.test(l)) {
        const cuerpo = [];
        while (i < lineas.length && (RE.cita.test(lineas[i]) || (!vacia(lineas[i]) && !iniciaBloque(lineas[i]) && cuerpo.length))) {
          const c = RE.cita.exec(lineas[i]);
          cuerpo.push(c ? c[1] : lineas[i]);
          i++;
        }
        salida.push(h('blockquote', {}, bloques(cuerpo, ctx)));
        continue;
      }
      if (RE.lista.test(l)) {
        const r = lista(lineas, i, ctx);
        salida.push(r.el);
        i = r.i;
        continue;
      }
      const parrafo = [];
      while (i < lineas.length && !vacia(lineas[i]) && (parrafo.length === 0 || !iniciaBloque(lineas[i]))) parrafo.push(lineas[i++].trim());
      salida.push(h('p', {}, enLinea(parrafo.join(' '), ctx)));
    }
    return salida;
  }

  // renderizar(texto, ctx) → { nodos, titulos, metadatos }. ctx da enlace(destino, hijos) e
  // imagen(destino, alt) (el panel decide adónde lleva cada enlace), textos y, opcionalmente,
  // codigo(lenguaje, texto, pre) para reemplazar bloques (los diagramas, en la Fase 5).
  function renderizar(texto, ctx) {
    let cuerpo = texto.replace(/\r\n/g, '\n');
    const metadatos = {};
    if (cuerpo.startsWith('---\n')) {
      const fin = cuerpo.indexOf('\n---\n', 4);
      if (fin > 0) {
        for (const linea of cuerpo.slice(4, fin).split('\n')) {
          const m = /^([a-z_]+):\s*(.*)$/.exec(linea);
          if (m) metadatos[m[1]] = m[2].replace(/^"(.*)"$/, '$1');
        }
        cuerpo = cuerpo.slice(fin + 5);
      }
    }
    const usados = new Map();
    const contexto = {
      ...ctx,
      titulos: [],
      idTitulo: (t) => {
        const base = anclaDe(t);
        const n = usados.get(base) || 0;
        usados.set(base, n + 1);
        return n ? `${base}-${n}` : base;
      },
    };
    const nodos = bloques(cuerpo.split('\n'), contexto);
    return { nodos, titulos: contexto.titulos, metadatos };
  }

  window.PrismaManual = { renderizar, anclaDe };
})();
