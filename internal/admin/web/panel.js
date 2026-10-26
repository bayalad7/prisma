// PRISMA — panel de administración. JavaScript sin dependencias.
// Toda la información llega de la API interna /admin/api/* (o de datos de ejemplo en el prototipo).
// El texto que viene del servidor se inserta siempre como texto, nunca como HTML.
'use strict';
(() => {
  const raiz = document.getElementById('raiz');

  // Utilidades de DOM y componentes del sistema de diseño (sistema.js).
  const { h, agregar, vaciar, logo, aviso, avisoError, dialogo, confirmar, copiar, botonCopiar, conEspera } = window.PrismaUI;
  const UI = window.PrismaUI;
  const TV = window.PRISMA_TEXTOS.virtual;

  // ---------------------------------------------------------------------------------------------
  // Formatos
  // ---------------------------------------------------------------------------------------------
  const fmtFecha = new Intl.DateTimeFormat('es', { day: 'numeric', month: 'short' });
  const fmtHora = new Intl.DateTimeFormat('es', { hour: '2-digit', minute: '2-digit' });
  const fmtNum = new Intl.NumberFormat('es');
  function fechaHora(iso) {
    if (!iso) return '—';
    const d = new Date(iso);
    const hoy = new Date();
    const mismoDia = d.toDateString() === hoy.toDateString();
    return mismoDia ? `hoy ${fmtHora.format(d)}` : `${fmtFecha.format(d)} ${fmtHora.format(d)}`;
  }
  function hace(iso) {
    if (!iso) return '—';
    const s = Math.round((Date.now() - new Date(iso).getTime()) / 1000);
    if (s < 60) return 'hace un momento';
    if (s < 3600) return `hace ${Math.round(s / 60)} min`;
    if (s < 86400) return `hace ${Math.round(s / 3600)} h`;
    return `hace ${Math.round(s / 86400)} d`;
  }
  function tamanoArchivo(b) { return b < 1024 ? `${b} B` : `${fmtNum.format(Math.round(b / 102.4) / 10)} KB`; }
  function duracion(ms) {
    if (ms === null || ms === undefined) return '—';
    return ms < 1000 ? `${ms} ms` : `${fmtNum.format(Math.round(ms / 100) / 10)} s`;
  }

  const PAGINAS = [
    { pagina: 'CP437', comando: 0, etiqueta: 'PC437 (EE. UU.)' },
    { pagina: 'CP850', comando: 2, etiqueta: 'PC850 (Multilingüe)' },
    { pagina: 'WPC1252', comando: 16, etiqueta: 'WPC1252 (Windows)' },
    { pagina: 'CP858', comando: 19, etiqueta: 'PC858 (Multilingüe con €)' },
  ];
  const PERFILES = { 58: { columnas: 32, columnas_fuente_b: 42, ancho_px: 384 }, 80: { columnas: 48, columnas_fuente_b: 64, ancho_px: 576 } };
  const TIPOS_CONEXION = { red: 'Red', usb: 'USB', bluetooth: 'Bluetooth', archivo: 'Archivo', virtual: TV.tipo };
  const insigniaVirtual = (imp) => imp.virtual && UI.insignia('virtual', TV.insignia);

  // ---------------------------------------------------------------------------------------------
  // API interna
  // ---------------------------------------------------------------------------------------------
  class ErrorApi extends Error {
    constructor(d, estadoHttp) {
      super((d && d.mensaje) || 'No se pudo completar la operación.');
      this.codigo = d && d.codigo; this.sugerencia = d && d.sugerencia; this.estado = estadoHttp;
      this.detalles = d && d.detalles;
    }
  }
  const api = {
    csrf: '',
    ultimaActividad: Date.now(), // cada petición al puente renueva la sesión
    // { sondeo: true }: petición automática (el rollo en vivo); no renueva la sesión.
    async pedir(metodo, ruta, cuerpo, { sondeo = false } = {}) {
      if (window.PRISMA_DEMO) return window.PRISMA_DEMO.pedir(metodo, ruta, cuerpo);
      const op = { method: metodo, credentials: 'same-origin', headers: { Accept: 'application/json' } };
      if (sondeo) op.headers['X-Prisma-Sondeo'] = '1';
      if (metodo !== 'GET') op.headers['X-CSRF-Token'] = api.csrf;
      if (cuerpo instanceof FormData) op.body = cuerpo;
      else if (cuerpo !== undefined) { op.headers['Content-Type'] = 'application/json'; op.body = JSON.stringify(cuerpo); }
      let r;
      try { r = await fetch(ruta.startsWith('/admin/') ? ruta : '/admin/api' + ruta, op); if (!sondeo) api.ultimaActividad = Date.now(); } catch {
        throw new ErrorApi({ mensaje: 'No hay conexión con el puente.', sugerencia: 'Revisa que PRISMA siga en marcha.' });
      }
      const tipo = r.headers.get('Content-Type') || '';
      const datos = tipo.includes('application/json') ? await r.json().catch(() => null) : await r.blob();
      if (r.status === 401 && datos && datos.codigo === 'SESION_REQUERIDA') {
        estado.sesion = null; arrancar(); throw new ErrorApi(datos, 401);
      }
      if (!r.ok || (datos && datos.ok === false)) throw new ErrorApi(datos, r.status);
      return datos;
    },
    get: (ruta, opciones) => api.pedir('GET', ruta, undefined, opciones),
    post: (ruta, cuerpo) => api.pedir('POST', ruta, cuerpo === undefined ? {} : cuerpo),
    put: (ruta, cuerpo) => api.pedir('PUT', ruta, cuerpo),
    borrar: (ruta) => api.pedir('DELETE', ruta),
    url: (ruta) => (window.PRISMA_DEMO && window.PRISMA_DEMO.url) ? window.PRISMA_DEMO.url(ruta) : '/admin/api' + ruta,
  };
  const enc = encodeURIComponent;

  // ---------------------------------------------------------------------------------------------
  // Estado, acceso y enrutado
  // ---------------------------------------------------------------------------------------------
  const estado = { sesion: null, pendientes: 0, nombrePuente: 'PRISMA', direccion: '', version: '' };
  // El nombre que se muestra: con el predeterminado, «Puente de impresión» en lugar de repetir «PRISMA».
  const NOMBRE_PREDETERMINADO = 'PRISMA';
  const nombreVisible = (n) => (!n || n === NOMBRE_PREDETERMINADO ? TN.sinNombre : n);
  const TN = window.PRISMA_TEXTOS.nav;
  // Secciones del panel: cada una tiene su ruta bajo /admin/ (el puente sirve el panel en todas), su
  // grupo en la navegación y la tecla de su atajo (g y la letra).
  const SECCIONES = [
    { id: 'inicio', ruta: '', icono: 'inicio', grupo: 'operacion', tecla: 'i' },
    { id: 'impresoras', ruta: 'impresoras', icono: 'impresora', grupo: 'operacion', tecla: 'p' },
    { id: 'solicitudes', ruta: 'solicitudes', icono: 'solicitud', grupo: 'operacion', tecla: 's' },
    { id: 'trabajos', ruta: 'trabajos', icono: 'trabajos', grupo: 'operacion', tecla: 't' },
    { id: 'pruebas', ruta: 'pruebas', icono: 'probar', grupo: 'integracion', tecla: 'r' },
    { id: 'aplicaciones', ruta: 'aplicaciones', icono: 'aplicacion', grupo: 'integracion', tecla: 'a' },
    { id: 'imagenes', ruta: 'imagenes', icono: 'imagen', grupo: 'integracion', tecla: 'm' },
    { id: 'ajustes', ruta: 'ajustes', icono: 'ajustes', grupo: 'sistema', tecla: 'j' },
    { id: 'ayuda', ruta: 'ayuda', icono: 'ayuda', grupo: 'sistema', tecla: 'y' },
  ];
  // En 0.1.0 las secciones eran #clientes, #recursos, #impresoras…: esos enlaces siguen funcionando.
  const HASH_VIEJOS = { clientes: 'aplicaciones', recursos: 'imagenes' };
  const rutaDe = (id) => `/admin/${SECCIONES.find((s) => s.id === id).ruta}`;
  const seccionDe = (ruta) => {
    const r = ruta.replace(/^\/admin\/?/, '').replace(/\/$/, '');
    if (/^pruebas\/(permisos|rollo\/[^/]+)$/.test(r)) return 'pruebas';
    if (/^ayuda\/([a-z]+\/[A-Za-z0-9-]+\.md|no-imprime|diagramas)$/.test(r)) return 'ayuda';
    return SECCIONES.find((s) => s.ruta === r)?.id;
  };

  async function arrancar() {
    try {
      estado.sesion = await api.get('/sesion');
      api.csrf = estado.sesion.csrf || '';
    } catch (e) {
      const reintentar = UI.boton({ texto: TAc0.reintentar, variante: 'primario', icono: 'recargar' });
      reintentar.addEventListener('click', arrancar);
      pantallaAcceso(TAc0.sinConexion, TAc0.sinConexionTexto, h('div', {}, reintentar));
      return;
    }
    const s = estado.sesion;
    if (!s.configurado) return pantallaConfiguracionInicial();
    if (!s.autenticado) return pantallaEntrar();
    if (s.debe_cambiar) return pantallaCambioObligatorio();
    mostrarApp();
  }

  function marca(subtitulo) {
    return h('div', { class: 'marca' }, logo(32), h('div', { class: 'marca-texto' },
      h('span', { class: 'marca-nombre' }, 'PRISMA'), h('span', { class: 'marca-puente' }, subtitulo)));
  }
  function cintaDemo() {
    if (!window.PRISMA_DEMO) return null;
    const ir = (pantalla) => { window.PRISMA_DEMO.verPantalla(pantalla); arrancar(); };
    return h('div', { class: 'demo-cinta' }, h('strong', {}, 'Prototipo con datos de ejemplo.'), 'Ver también:',
      h('button', { type: 'button', onclick: () => ir('entrar') }, 'inicio de sesión'),
      h('button', { type: 'button', onclick: () => ir('configuracion') }, 'configuración inicial'),
      h('button', { type: 'button', onclick: () => ir('cambio') }, 'cambio obligatorio de contraseña'),
      h('button', { type: 'button', onclick: () => ir('panel') }, 'panel'));
  }

  const TAc0 = window.PRISMA_TEXTOS.acceso;
  // Campo de contraseña con botón para mostrarla (evita errores al escribirla) y error en línea.
  function campoClave(id, etiqueta, { auto = 'current-password', ayuda, validar } = {}) {
    const input = UI.entrada({ tipo: 'password', autocomplete: auto, name: id });
    const ver = UI.boton({ icono: 'ojo', etiqueta: TAc0.mostrar, variante: 'texto', chico: true, 'aria-pressed': 'false' });
    ver.addEventListener('click', () => {
      const visible = input.type === 'password';
      input.type = visible ? 'text' : 'password';
      ver.setAttribute('aria-pressed', String(visible));
      ver.setAttribute('aria-label', visible ? TAc0.ocultar : TAc0.mostrar);
      ver.title = visible ? TAc0.ocultar : TAc0.mostrar;
    });
    const campo = UI.campo({ id, etiqueta, control: input, ayuda, validar });
    const caja = h('div', { class: 'clave' });
    input.replaceWith(caja);
    caja.append(input, ver);
    return { input, campo };
  }
  function pantallaAcceso(titulo, texto, ...contenido) {
    vaciar(raiz, h('main', { class: 'acceso' }, h('div', { class: 'tarjeta' },
      cintaDemo(), marca(TAc0.subtitulo), h('h1', {}, titulo), texto && h('p', { class: 'tenue' }, texto), contenido)));
  }

  function pantallaEntrar() {
    const clave = campoClave('contrasena', TAc0.contrasena, { validar: (v) => (v ? '' : TAc0.falta) });
    const error = h('div', { 'aria-live': 'assertive' });
    const boton = UI.boton({ texto: TAc0.botonEntrar, variante: 'primario', tipo: 'submit' });
    const form = h('form', { class: 'pila-amplia', novalidate: true, onsubmit: async (ev) => {
      ev.preventDefault();
      vaciar(error);
      if (!UI.revisarCampos([clave.campo])) return;
      try {
        await conEspera(boton, async () => {
          const r = await api.post('/entrar', { contrasena: clave.input.value });
          api.csrf = r.csrf;
        });
        arrancar();
      } catch (e) {
        vaciar(error, UI.nota({ tipo: 'error', titulo: e.message, texto: e.sugerencia }));
        clave.input.select();
      }
    } }, clave.campo, error, boton);
    pantallaAcceso(TAc0.entrar, null, form,
      h('details', { class: 'avanzado' }, h('summary', {}, TAc0.olvido), h('div', {}, h('p', { class: 'tenue pequeno' }, TAc0.olvidoTexto))));
    clave.input.focus();
  }

  function formularioNuevaContrasena({ titulo, texto, conCodigo = false, ruta, boton: textoBoton }) {
    const codigo = conCodigo && UI.entrada({ mono: true, autocomplete: 'one-time-code' });
    const cCodigo = codigo && UI.campo({ id: 'codigo', etiqueta: TAc0.codigo, control: codigo, ayuda: TAc0.codigoAyuda, validar: (v) => (v.trim() ? '' : TAc0.codigoFalta) });
    const nueva = campoClave('nueva', TAc0.nueva, { auto: 'new-password', ayuda: TAc0.nuevaAyuda, validar: (v) => (v.length >= 10 ? '' : TAc0.corta) });
    const repetir = campoClave('repetir', TAc0.repetir, { auto: 'new-password', validar: (v) => (v === nueva.input.value ? '' : TAc0.noCoinciden) });
    const error = h('div', { 'aria-live': 'assertive' });
    const boton = UI.boton({ texto: textoBoton, variante: 'primario', tipo: 'submit' });
    const campos = [cCodigo, nueva.campo, repetir.campo].filter(Boolean);
    const form = h('form', { class: 'pila-amplia', novalidate: true, onsubmit: async (ev) => {
      ev.preventDefault();
      vaciar(error);
      if (!UI.revisarCampos(campos)) return;
      try {
        await conEspera(boton, async () => {
          const r = await api.post(ruta, { contrasena: nueva.input.value, codigo: codigo ? codigo.value.trim() : undefined });
          if (r.csrf) api.csrf = r.csrf;
        });
        aviso(TAc0.guardada, 'ok');
        arrancar();
      } catch (e) {
        if (cCodigo && /^CODIGO_/.test(e.codigo || '')) { cCodigo.ponerError([e.message, e.sugerencia].filter(Boolean).join(' ')); codigo.focus(); return; }
        vaciar(error, UI.nota({ tipo: 'error', titulo: e.message, texto: e.sugerencia }));
      }
    } }, campos, error, boton);
    pantallaAcceso(titulo, texto, form);
    (codigo || nueva.input).focus();
  }
  function pantallaConfiguracionInicial() {
    formularioNuevaContrasena({ titulo: TAc0.inicialTitulo, texto: TAc0.inicialTexto, conCodigo: !estado.sesion.local,
      ruta: '/configuracion-inicial', boton: TAc0.inicialBoton });
  }
  function pantallaCambioObligatorio() {
    formularioNuevaContrasena({ titulo: TAc0.cambioTitulo, texto: TAc0.cambioTexto, ruta: '/contrasena', boton: TAc0.cambioBoton });
  }

  let app;
  function mostrarApp() {
    const viejo = location.hash.slice(1);
    if (viejo && location.pathname === '/admin/' && (HASH_VIEJOS[viejo] || SECCIONES.some((x) => x.id === viejo))) history.replaceState(null, '', rutaDe(HASH_VIEJOS[viejo] || viejo));
    const grupos = Object.entries(TN.grupos).map(([g, titulo]) => ({
      titulo,
      secciones: SECCIONES.filter((x) => x.grupo === g).map((x) => ({ id: x.id, texto: TN.secciones[x.id], icono: x.icono, href: rutaDe(x.id) })),
    }));
    const salir = UI.boton({ texto: TN.cerrarSesion, icono: 'salir', chico: true });
    salir.addEventListener('click', async () => {
      try { await api.post('/salir'); } catch { /* la sesión ya no existe */ }
      estado.sesion = null; arrancar();
    });
    app = UI.marco({
      marca: h('a', { class: 'marca', href: rutaDe('inicio') }, logo(30), h('span', { class: 'marca-texto' },
        h('span', { class: 'marca-nombre' }, 'PRISMA'), h('span', { class: 'marca-puente', id: 'nombre-puente' }, nombreVisible(estado.nombrePuente)))),
      grupos, actual: seccionActual(), inferior: ['inicio', 'impresoras', 'solicitudes', 'trabajos', 'ayuda'],
      pie: [UI.selectorTema(), h('span', { id: 'version' }, estado.version && TN.version(estado.version)), botonAtajos(), salir],
    });
    vaciar(raiz, app);
    UI.configurarAyuda({ abrir: (destino) => abrirAyuda(destino).catch(() => {}), deError: (codigo) => TAy.errores[codigo] });
    refrescarPendientes();
    navegar();
  }
  function seccionActual() { return seccionDe(location.pathname) || 'inicio'; }
  function ir(id, consulta = '') { irRuta(rutaDe(id) + consulta); }
  function irRuta(destino) {
    if (location.pathname + location.search + location.hash !== destino) history.pushState(null, '', destino);
    navegar();
  }
  async function navegar() {
    const id = seccionActual();
    app.marcar(id);
    document.title = TN.tituloPagina(TN.secciones[id], nombreVisible(estado.nombrePuente));
    const vista = h('div', { class: 'vista' }, cintaDemo(), UI.esqueleto({ lineas: 4 }));
    vaciar(app.principal, vista);
    try {
      await VISTAS[id](vista);
      vista.querySelector('.esqueleto')?.remove();
    } catch (e) {
      if (e.estado === 401) return;
      vaciar(vista, cintaDemo(), h('div', { class: 'nota error' }, e.message, ' ', e.sugerencia || ''));
    }
    // Con un ancla (una sección del manual), la vista va a ese título.
    const ancla = location.hash.length > 1 && document.getElementById(decodeURIComponent(location.hash.slice(1)));
    if (ancla && vista.contains(ancla)) { enfocarAncla(ancla); return; }
    // El foco va al título de la pantalla: el lector de pantalla anuncia dónde se está.
    const titulo = vista.querySelector('h1');
    if (titulo) { titulo.tabIndex = -1; titulo.focus({ preventScroll: true }); } else app.principal.focus({ preventScroll: true });
    window.scrollTo(0, 0);
  }
  function encabezado(titulo, subtitulo, ...acciones) {
    const destino = ayudaDePantalla();
    const bAyuda = destino && UI.boton({ texto: TAy.botonPantalla, icono: 'ayuda', variante: 'texto', 'aria-haspopup': 'dialog',
      'aria-label': TAy.botonPantallaEtiqueta, class: 'boton texto boton-ayuda',
      onclick: async () => { await cargarDiagramas().catch(() => null); abrirAyuda(destino, { pantalla: clavePantalla() }).catch(avisoError); } });
    return h('div', { class: 'encabezado' }, h('div', {}, h('h1', {}, titulo), subtitulo && h('p', { class: 'subtitulo' }, subtitulo)),
      (acciones.length > 0 || bAyuda) && h('div', { class: 'acciones' }, acciones, bAyuda));
  }
  function recargar() { navegar(); }
  function actualizarContador() {
    if (app) app.contador('solicitudes', estado.pendientes, TN.pendientes(estado.pendientes));
  }
  // Pide el resumen al entrar y al volver a la pestaña. El Inicio además se actualiza solo, como
  // sondeo (X-Prisma-Sondeo): una petición normal cada tanto mantendría viva la sesión.
  async function refrescarPendientes() {
    try {
      const r = await api.get('/resumen');
      estado.pendientes = r.solicitudes_pendientes;
      estado.nombrePuente = r.puente.nombre;
      estado.version = r.puente.version;
      const nombre = document.getElementById('nombre-puente');
      if (nombre) nombre.textContent = nombreVisible(r.puente.nombre);
      const version = document.getElementById('version');
      if (version) version.textContent = TN.version(r.puente.version);
    } catch { /* sin cambios */ }
    actualizarContador();
  }
  const enElPanel = () => app && app.isConnected;
  document.addEventListener('visibilitychange', () => { if (document.visibilityState === 'visible' && enElPanel()) refrescarPendientes(); });
  window.addEventListener('popstate', () => { if (enElPanel()) navegar(); });
  // Los enlaces internos cambian de pantalla sin recargar la página.
  document.addEventListener('click', (ev) => {
    const a = ev.target.closest('a[href^="/admin/"]');
    if (!a || !enElPanel() || ev.defaultPrevented || ev.button !== 0 || ev.metaKey || ev.ctrlKey || ev.shiftKey || ev.altKey) return;
    if (a.target || a.hasAttribute('download')) return;
    const destino = new URL(a.href);
    if (!seccionDe(destino.pathname)) return;
    ev.preventDefault();
    irRuta(destino.pathname + destino.search + destino.hash);
  });

  // Atajos: g y la letra de la sección; ? muestra la ayuda. Se pueden desactivar (WCAG 2.1.4).
  const CLAVE_ATAJOS = 'prisma.atajos';
  const atajosActivos = () => { try { return localStorage.getItem(CLAVE_ATAJOS) !== 'no'; } catch { return true; } };
  let pulsoG = 0;
  document.addEventListener('keydown', (ev) => {
    if (!enElPanel() || !atajosActivos() || ev.ctrlKey || ev.metaKey || ev.altKey) return;
    if (ev.target.closest('input, textarea, select, [contenteditable]') || document.querySelector('dialog[open]')) return;
    if (ev.key === '?') { ev.preventDefault(); mostrarAtajos(); return; }
    if (ev.key === 'g') { pulsoG = Date.now(); return; }
    if (Date.now() - pulsoG < 1500) {
      pulsoG = 0;
      const destino = SECCIONES.find((x) => x.tecla === ev.key.toLowerCase());
      if (destino) { ev.preventDefault(); ir(destino.id); }
    }
  });
  function mostrarAtajos() {
    const activar = UI.interruptor({ id: 'atajos-activos', texto: TN.atajos.activar, ayuda: TN.atajos.activarAyuda, marcado: atajosActivos() });
    activar.input.addEventListener('change', () => {
      try { localStorage.setItem(CLAVE_ATAJOS, activar.input.checked ? 'si' : 'no'); } catch { /* solo en esta página */ }
    });
    dialogo({ titulo: TN.atajos.titulo, cuerpo: h('div', { class: 'pila-amplia' }, h('p', { class: 'tenue pequeno' }, TN.atajos.ayuda),
      h('dl', { class: 'datos' }, SECCIONES.map((x) => [h('dt', {}, h('kbd', {}, 'g'), ' ', h('kbd', {}, x.tecla)), h('dd', {}, TN.secciones[x.id])]),
        h('dt', {}, h('kbd', {}, '?')), h('dd', {}, TN.atajos.mostrar)),
      activar) });
  }
  function botonAtajos() {
    const b = UI.boton({ texto: TN.atajos.boton, variante: 'texto', chico: true, class: 'boton texto chico atajos-boton' });
    b.addEventListener('click', mostrarAtajos);
    return b;
  }

  // ---------------------------------------------------------------------------------------------
  // 1. Inicio: el estado de un vistazo, con una acción directa en cada tarjeta, y «Primeros pasos»
  // hasta que el puente esté listo. Los pasos salen de los datos del puente (persisten entre
  // equipos); ocultar la lista es una comodidad de cada navegador.
  // ---------------------------------------------------------------------------------------------
  const TI = window.PRISMA_TEXTOS.inicio;
  const CLAVE_PASOS_OCULTOS = 'prisma.primerosPasos.ocultos';
  const pasosOcultos = () => { try { return localStorage.getItem(CLAVE_PASOS_OCULTOS) === '1'; } catch { return false; } };
  const ocultarPasos = (si) => { try { localStorage.setItem(CLAVE_PASOS_OCULTOS, si ? '1' : '0'); } catch { /* solo esta vez */ } };
  const ESTADO_INSIGNIA_IMPRESORA = { en_linea: 'ok', fuera_de_linea: 'error', ocupada: 'info', desconocido: 'desconocido', inactiva: 'desconocido' };
  const insigniaImpresora = (estadoImp) => UI.insignia(ESTADO_INSIGNIA_IMPRESORA[estadoImp] || 'desconocido', TI.impresoras.estados[estadoImp] || estadoImp);
  const enlaceBoton = (seccion, texto, variante = '', consulta = '') => h('a', { class: `boton chico ${variante}`, href: rutaDe(seccion) + consulta }, texto);

  // Cada cuánto se actualiza el Inicio mientras se ve (con la cabecera de sondeo: no renueva la sesión).
  const INICIO_CADA_MS = 15_000;
  const CONTROLES = 'a[href], button, input, select, textarea, summary, [contenteditable], [tabindex]:not([tabindex="-1"])';

  async function vistaInicio(vista) {
    const caja = h('div', { class: 'contenido-vivo' });
    let ultimo = '';
    const pintar = (r) => {
      ultimo = JSON.stringify(r);
      estado.pendientes = r.solicitudes_pendientes;
      estado.direccion = r.puente.direccion;
      actualizarContador();
      vaciar(caja, contenidoInicio(r));
    };
    pintar(await api.get('/resumen'));
    agregar(vista, caja);

    // Se vuelve a dibujar solo si algo cambió, con la pestaña visible y sin estorbar: no mientras
    // la persona usa un control de la pantalla o tiene un diálogo abierto.
    let ocupado = false;
    const revisar = async () => {
      if (ocupado || document.visibilityState !== 'visible' || document.querySelector('dialog[open]')) return;
      const enfocado = caja.contains(document.activeElement) ? document.activeElement : null;
      if (enfocado && enfocado.matches(CONTROLES)) return;
      ocupado = true;
      try {
        const r = await api.get('/resumen', { sondeo: true });
        if (!vista.isConnected || JSON.stringify(r) === ultimo) return;
        pintar(r);
        // El foco estaba en el título (al llegar a la pantalla): sigue en el título nuevo.
        if (enfocado && enfocado.tagName === 'H1') caja.querySelector('h1')?.focus({ preventScroll: true });
      } catch { /* se intenta en la siguiente vuelta */ } finally { ocupado = false; }
    };
    const reloj = setInterval(() => {
      if (!vista.isConnected) { clearInterval(reloj); document.removeEventListener('visibilitychange', alVolver); return; }
      revisar();
    }, INICIO_CADA_MS);
    function alVolver() { if (vista.isConnected) revisar(); }
    document.addEventListener('visibilitychange', alVolver);
  }

  function contenidoInicio(r) {
    const imps = r.impresoras;
    const listas = imps.filter((i) => i.estado === 'en_linea').length;
    const sinRespuesta = imps.filter((i) => i.estado === 'fuera_de_linea');
    const cert = r.certificado_actual;
    const certMal = cert && cert.estado !== 'vigente';
    const porRevisar = sinRespuesta.length + (r.solicitudes_pendientes > 0 ? 1 : 0) + (r.trabajos_hoy.fallidos > 0 ? 1 : 0) + (certMal ? 1 : 0);

    return [
      encabezado(nombreVisible(r.puente.nombre), TI.subtitulo, porRevisar === 0 ? UI.insignia('ok', TI.todoBien) : UI.insignia('aviso', TI.porRevisar(porRevisar)),
        r.puente.nombre === NOMBRE_PREDETERMINADO && h('a', { class: 'boton chico texto', href: rutaDe('ajustes'), 'aria-label': TI.ponerNombreEtiqueta }, UI.icono('lapiz'), TI.ponerNombre)),
      r.certificado && UI.nota({
        tipo: r.certificado.estado === 'vencido' ? 'error' : 'aviso',
        titulo: r.certificado.estado === 'vencido' ? TI.certificado.avisoVencido(r.certificado.vence)
          : TI.certificado.avisoVence(TI.certificado.dias(r.certificado.dias_restantes), r.certificado.vence),
        texto: r.certificado.autofirmado ? TI.certificado.avisoAutofirmado : TI.certificado.avisoArchivos,
        acciones: enlaceBoton('ajustes', TI.certificado.irAjustes, 'primario', '?pestana=https'),
      }),
      r.actualizacion && UI.nota({
        tipo: 'info', titulo: TI.actualizacion.titulo(r.actualizacion.version), texto: TI.actualizacion.texto(r.puente.version),
        acciones: h('a', { class: 'boton chico', href: r.actualizacion.url, target: '_blank', rel: 'noopener noreferrer' }, UI.icono('externo'), TI.actualizacion.ver),
      }),
      primerosPasos(r),
      h('div', { class: 'rejilla de-cuatro' },
        tarjetaCifra('impresora', sinRespuesta.length > 0 ? 'error' : 'ok', TI.impresoras.titulo,
          imps.length ? TI.impresoras.cifra(listas, imps.length) : '0', imps.length ? TI.impresoras.texto : TI.impresoras.ninguna,
          imps.length > 0 && h('ul', { class: 'lista-impresoras' }, imps.map((i) => h('li', {},
            h('span', { class: 'lista-impresoras-nombre' }, h('span', { class: 'mono' }, i.alias), ' ', h('span', { class: 'tenue pequeno' }, i.nombre)),
            h('span', { class: 'fila' }, i.pendientes > 0 && h('span', { class: 'tenue pequeno numeros' }, TI.impresoras.enCola(i.pendientes)),
              insigniaVirtual(i), insigniaImpresora(i.estado))))),
          sinRespuesta.length > 0 ? enlaceBoton('impresoras', TI.impresoras.revisar(sinRespuesta[0].alias))
            : imps.length === 0 ? botonCrearVirtual() : enlaceBoton('impresoras', TI.impresoras.verTodas, 'texto')),
        tarjetaCifra('solicitud', r.solicitudes_pendientes > 0 ? 'aviso' : 'ok', TI.solicitudes.titulo, String(r.solicitudes_pendientes),
          r.solicitudes_pendientes > 0 ? TI.solicitudes.texto(r.solicitudes_pendientes) : TI.solicitudes.ninguna, null,
          r.solicitudes_pendientes > 0 && enlaceBoton('solicitudes', TI.solicitudes.revisar, 'primario')),
        tarjetaHoy(r.trabajos_hoy),
        tarjetaCertificado(cert, r.puente.modo_tls)),
      h('section', { class: 'pila', 'aria-labelledby': 't-ultimos' },
        h('div', { class: 'fila-entre' }, h('h2', { id: 't-ultimos' }, TI.ultimos.titulo), r.ultimos_trabajos.length > 0 && h('a', { href: rutaDe('trabajos'), class: 'pequeno' }, TI.ultimos.verTodos)),
        tablaUltimos(r.ultimos_trabajos)),
    ];
  }

  function tarjetaCifra(ic, tipo, titulo, cifra, texto, detalle, accion) {
    return UI.tarjeta({ clase: `estado-${tipo}`, titulo, nivel: 2 },
      h('div', { class: 'cifra-estado' }, UI.icono(ic), h('div', {}, h('div', { class: 'cifra numeros' }, cifra), h('div', { class: 'tenue pequeno' }, texto))),
      detalle, accion && h('div', {}, accion));
  }
  function tarjetaHoy(hoy) {
    const total = hoy.completados + hoy.fallidos;
    return tarjetaCifra(hoy.fallidos > 0 ? 'error' : 'trabajos', hoy.fallidos > 0 ? 'error' : 'ok', TI.hoy.titulo,
      total === 0 ? '0' : TI.hoy.cifra(hoy.fallidos), TI.hoy.texto(hoy.fallidos, total),
      hoy.pruebas > 0 && h('p', { class: 'tenue pequeno' }, TI.hoy.pruebas(hoy.pruebas)),
      hoy.fallidos > 0 ? enlaceBoton('trabajos', TI.hoy.verQuePaso, '', '?estado=fallido') : enlaceBoton('trabajos', TI.hoy.verTrabajos, 'texto'));
  }
  function tarjetaCertificado(cert, modo) {
    if (!cert) return tarjetaCifra('candado', 'info', TI.certificado.titulo, TI.certificado.sinHttps, TI.certificado.sinHttpsTexto);
    const vencido = cert.estado === 'vencido';
    return tarjetaCifra('candado', vencido ? 'error' : (cert.estado === 'por_vencer' ? 'aviso' : 'info'), TI.certificado.titulo,
      vencido ? TI.certificado.vencido : TI.certificado.dias(cert.dias_restantes),
      vencido ? TI.certificado.vencidoTexto(cert.vence) : TI.certificado.texto, null,
      cert.estado !== 'vigente' ? enlaceBoton('ajustes', TI.certificado.renovar, '', '?pestana=https')
        : modo === 'autofirmado' && enlaceBoton('ayuda', TI.certificado.instalar, 'texto'));
  }

  function primerosPasos(r) {
    const p = r.primeros_pasos;
    const conImpresora = p.impresoras_fisicas + p.impresoras_virtuales > 0;
    const pasos = [
      { hecho: true, titulo: TI.pasos.contrasena },
      { hecho: conImpresora, titulo: TI.pasos.impresora, texto: TI.pasos.impresoraTexto,
        accion: () => h('div', { class: 'acciones' }, enlaceBoton('impresoras', TI.pasos.impresoraBoton, 'primario'), botonCrearVirtual()) },
      { hecho: p.prueba_impresa, titulo: TI.pasos.prueba, texto: TI.pasos.pruebaTexto, accion: () => conImpresora && botonPrueba(r.impresoras) },
      p.impresoras_fisicas > 0 && { hecho: p.acentos_probados, titulo: TI.pasos.acentos, texto: TI.pasos.acentosTexto,
        accion: () => enlaceBoton('impresoras', TI.pasos.acentosBoton) },
      { hecho: p.aplicaciones > 0, titulo: TI.pasos.aplicacion, texto: TI.pasos.aplicacionTexto,
        accion: () => enlaceBoton('aplicaciones', TI.pasos.aplicacionBoton, 'primario') },
      p.impresoras_fisicas === 0 && p.impresoras_virtuales > 0 && { hecho: false, titulo: TI.pasos.fisica, texto: TI.pasos.fisicaTexto,
        accion: () => enlaceBoton('impresoras', TI.pasos.fisicaBoton) },
    ].filter(Boolean);
    const hechos = pasos.filter((x) => x.hecho).length;
    const caja = h('div', {});
    const pintar = () => {
      if (pasosOcultos()) {
        const ver = UI.boton({ texto: TI.pasos.ver(hechos, pasos.length), variante: 'texto', chico: true, icono: 'check' });
        ver.addEventListener('click', () => { ocultarPasos(false); pintar(); });
        vaciar(caja, h('div', {}, ver));
        return;
      }
      const ocultar = UI.boton({ texto: TI.pasos.ocultar, variante: 'texto', chico: true });
      ocultar.addEventListener('click', () => { ocultarPasos(true); pintar(); });
      if (hechos === pasos.length) {
        vaciar(caja, UI.nota({ tipo: 'ok', titulo: TI.pasos.listo, acciones: ocultar }));
        return;
      }
      const barra = h('span');
      barra.style.width = `${(hechos / pasos.length) * 100}%`;
      vaciar(caja, UI.tarjeta({ titulo: TI.pasos.titulo, subtitulo: TI.pasos.avance(hechos, pasos.length), icono: 'check', acciones: ocultar },
        h('div', { class: 'progreso', role: 'progressbar', 'aria-valuemin': '0', 'aria-valuemax': String(pasos.length), 'aria-valuenow': String(hechos), 'aria-label': TI.pasos.titulo },
          h('div', { class: 'progreso-barra' }, barra)),
        h('ol', { class: 'lista-pasos' }, pasos.map((x) => h('li', { class: x.hecho ? 'hecho' : '' },
          h('span', { class: 'paso-marca', 'aria-hidden': 'true' }, x.hecho ? UI.icono('check') : ''),
          h('div', {}, h('p', { class: 'paso-titulo' }, x.titulo, x.hecho && h('span', { class: 'solo-lector' }, ` ${TI.pasos.hecho}`)),
            !x.hecho && x.texto && h('p', { class: 'tenue pequeno' }, x.texto)),
          !x.hecho && x.accion && x.accion())))));
    };
    pintar();
    return caja;
  }

  // Imprime el ticket de diagnóstico en la impresora más adecuada (una física en línea, si no la
  // primera) y muestra cómo salió.
  function botonPrueba(imps) {
    const imp = imps.find((i) => !i.virtual && i.estado === 'en_linea') || imps.find((i) => i.estado !== 'fuera_de_linea') || imps[0];
    const b = UI.boton({ texto: `${TI.pasos.pruebaBoton} (${imp.alias})`, icono: 'ticket', variante: 'primario', chico: true });
    b.addEventListener('click', async () => {
      try {
        const r = await conEspera(b, () => api.post(`/impresoras/${enc(imp.alias)}/diagnostico`));
        await dialogo({ titulo: TI.diagnostico.titulo, ancho: 520, cuerpo: h('div', { class: 'pila' },
          h('p', { class: 'tenue pequeno' }, imp.virtual ? TV.diagnosticoVirtual : TV.diagnosticoFisica),
          UI.ticketPapel({ imagen: r.imagen, anchoMm: imp.ancho_mm || 80 })) });
        recargar();
      } catch (e) { avisoError(e); }
    });
    return b;
  }

  // Últimos trabajos: tabla que en celular se vuelve tarjetas (el detalle llega en la entrega de Trabajos).
  function tablaUltimos(lista) {
    if (lista.length === 0) return UI.vacio({ icono: 'trabajos', titulo: TI.ultimos.vacio, texto: TI.ultimos.vacioTexto });
    const C = TI.ultimos.columnas, E = TI.ultimos.estados;
    const insigniaTrabajo = (t) => t.estado === 'fallido' ? UI.insignia('error', E.fallido)
      : t.estado === 'completado' ? ((t.advertencias || []).length ? UI.insignia('aviso', E.conAviso) : UI.insignia('ok', E.completado))
        : UI.insignia(t.estado === 'imprimiendo' ? 'info' : 'acento', E[t.estado] || t.estado);
    return UI.tabla({
      etiqueta: TI.ultimos.titulo,
      columnas: [
        { titulo: C.referencia, celda: (t) => h('span', { class: 'mono' }, t.referencia || '—') },
        { titulo: C.fecha, celda: (t) => fechaHora(t.creado), clase: 'numeros' },
        { titulo: C.aplicacion, celda: (t) => (t.cliente_id === 'cli_panel' ? TI.ultimos.prueba : t.cliente_nombre || t.cliente_id) },
        { titulo: C.impresora, celda: (t) => h('span', { class: 'fila' }, h('span', { class: 'mono' }, t.impresora_alias), insigniaVirtual(t)) },
        { titulo: C.estado, celda: insigniaTrabajo },
        { titulo: C.detalle, celda: (t) => {
          if (t.codigo_error) return h('span', { class: 'chip aviso' }, t.codigo_error);
          const adv = t.advertencias || [];
          return adv.length ? h('span', { class: 'fila' }, adv.map((a) => h('span', { class: 'chip' }, a))) : '—';
        } },
      ],
      filas: lista,
    });
  }

  // ---------------------------------------------------------------------------------------------
  // 2. Impresoras
  // ---------------------------------------------------------------------------------------------
  const TP = window.PRISMA_TEXTOS.impresoras;
  async function vistaImpresoras(vista) {
    const lista = await api.get('/impresoras');
    const botonAgregar = () => {
      const b = UI.boton({ texto: TP.agregar, variante: 'primario', icono: 'mas' });
      b.addEventListener('click', () => asistenteImpresora(vista));
      return b;
    };
    agregar(vista, encabezado(TP.titulo, TP.subtitulo, botonAgregar()),
      lista.length === 0
        ? UI.vacio({ icono: 'impresora', titulo: TP.vacio, texto: TP.vacioTexto, accion: [botonAgregar(), botonCrearVirtual()] })
        : h('div', { class: 'lista' }, lista.map((imp) => tarjetaImpresora(imp, vista))));
  }

  // Tarjeta de una impresora: la acción principal depende del estado (probar si no responde,
  // imprimir una prueba si está lista, activar si está desactivada).
  function tarjetaImpresora(imp, vista) {
    const A = TP.acciones, D = TP.datos, C = TP.capacidades;
    const ruta = `/impresoras/${enc(imp.alias)}`;
    const resultado = h('div', { 'aria-live': 'polite' });
    const ok = (titulo, tipo = 'ok') => vaciar(resultado, UI.nota({ tipo, titulo, cerrable: true }));
    const fallo = (e) => vaciar(resultado, UI.nota({ tipo: 'error', titulo: e.message, texto: e.sugerencia, cerrable: true }));
    const sinRespuesta = imp.activa && imp.estado === 'fuera_de_linea';
    const accion = (texto, variante, icono, alPulsar) => {
      const b = UI.boton({ texto, variante, icono, chico: true });
      b.addEventListener('click', () => alPulsar(b));
      return b;
    };
    const bProbar = imp.activa && accion(A.probar, sinRespuesta ? 'primario' : '', 'red', async (b) => {
      try { await conEspera(b, () => api.post(`${ruta}/probar`)); ok(TP.respondio); } catch (e) { fallo(e); }
    });
    const bPrueba = imp.activa && accion(A.prueba, sinRespuesta ? '' : 'primario', 'ticket', async (b) => {
      try {
        const r = await conEspera(b, () => api.post(`${ruta}/diagnostico`));
        ok(r.aviso ? TP.pruebaConAviso(r.aviso) : TP.pruebaEnviada, r.aviso ? 'aviso' : 'ok');
        await dialogo({ titulo: TI.diagnostico.titulo, ancho: 520, cuerpo: h('div', { class: 'pila' },
          h('p', { class: 'tenue pequeno' }, imp.virtual ? TV.diagnosticoVirtual : TV.diagnosticoFisica),
          UI.ticketPapel({ imagen: r.imagen, anchoMm: imp.ancho_mm })) });
        if (imp.virtual) recargar();
      } catch (e) { fallo(e); }
    });
    const bAcentos = imp.activa && !imp.virtual && accion(TAc.boton, '', null, async (b) => { if (await probarAcentos(imp, b)) recargar(); });
    const bTickets = imp.virtual && accion(TV.verTickets(imp.tickets_recibidos), '', 'ticket', () => verRollo(imp));
    const bActivar = accion(imp.activa ? A.desactivar : A.activar, imp.activa ? '' : 'primario', null, async (b) => {
      if (imp.activa && !(await confirmar(TP.desactivarTitulo(imp.alias), TP.desactivarTexto,
        { boton: A.desactivar, peligro: true, consecuencias: TP.desactivarConsecuencias }))) return;
      try {
        await conEspera(b, () => api.put(ruta, { ...imp, activa: !imp.activa }));
        aviso(imp.activa ? TP.desactivadaAviso(imp.alias) : TP.activada(imp.alias), 'ok');
        recargar();
      } catch (e) { fallo(e); }
    });
    const bEliminar = accion(A.eliminar, 'peligro', 'papelera', async () => {
      // Dice qué aplicaciones se quedan sin esta impresora.
      let usan = [];
      try { usan = (await api.get('/clientes')).filter((c) => c.todas_las_impresoras || c.impresoras.includes(imp.alias)).map((c) => c.nombre); } catch { /* se confirma sin la lista */ }
      const E = window.PRISMA_TEXTOS.eliminarImpresora;
      if (!(await confirmar(E.titulo(imp.alias), E.texto, { boton: E.boton, peligro: true,
        consecuencias: [usan.length ? E.usan(usan) : E.ninguna, E.historial] }))) return;
      try { await api.borrar(ruta); aviso(TP.eliminada(imp.alias), 'ok'); recargar(); } catch (e) { avisoError(e); }
    });
    const pagina = PAGINAS.find((x) => x.pagina === imp.pagina_codigos);
    const capacidades = [
      imp.soporta_corte ? C.corta : C.noCorta, imp.soporta_cajon && C.cajon, imp.soporta_qr_nativo ? C.qr : C.qrImagen,
      !imp.soporta_imagen && C.sinImagenes, imp.confirmar_impresion && C.confirma,
    ].filter(Boolean).join(' · ');
    return UI.tarjeta({ titulo: imp.alias, subtitulo: imp.nombre, icono: imp.virtual ? 'virtual' : 'impresora',
      acciones: h('span', { class: 'fila' }, insigniaVirtual(imp),
        imp.activa ? insigniaImpresora(imp.estado) : UI.insignia('desconocido', TP.desactivada)) },
    h('dl', { class: 'datos' },
      h('dt', {}, D.conexion), imp.virtual
        ? h('dd', {}, TV.conexion(imp.tickets_recibidos, imp.tickets_en_memoria),
          imp.simulacion !== 'normal' && [' · ', h('strong', {}, TV.simulacionActiva[imp.simulacion])])
        : h('dd', {}, `${TA.conexion.tipos[imp.tipo] || imp.tipo} · `, h('span', { class: 'mono' }, imp.destino)),
      h('dt', {}, D.papel), h('dd', {}, TP.papel(imp.ancho_mm, imp.columnas)),
      !imp.virtual && [h('dt', {}, D.acentos), h('dd', {}, pagina ? pagina.etiqueta : imp.pagina_codigos)],
      h('dt', {}, D.capacidades), h('dd', {}, capacidades),
      imp.ultimo_error && [h('dt', {}, D.problema), h('dd', {}, (TT.codigos[imp.ultimo_error] || { que: imp.ultimo_error }).que)]),
    h('div', { class: 'acciones' }, imp.activa ? [bPrueba, bProbar] : bActivar, bAcentos, bTickets,
      accion(A.editar, '', 'lapiz', () => formularioImpresora(vista, imp)), imp.activa && bActivar, bEliminar),
    resultado);
  }

  // Crea la impresora virtual «prueba» (o «prueba-2», …) con el perfil típico de 80 mm.
  function botonCrearVirtual() {
    const b = UI.boton({ texto: TV.crearBoton, icono: 'virtual' });
    b.addEventListener('click', async () => {
      try {
        await conEspera(b, async () => {
          const existentes = new Set((await api.get('/impresoras')).map((x) => x.alias));
          let alias = 'prueba';
          for (let n = 2; existentes.has(alias); n++) alias = `prueba-${n}`;
          await api.post('/impresoras', { alias, nombre: TV.nombre, tipo: 'virtual', destino: '', ancho_mm: 80, ...PERFILES[80],
            pagina_codigos: 'CP858', comando_pagina_codigos: 19, soporta_corte: true, soporta_cajon: true, soporta_qr_nativo: true,
            soporta_codigo_barras_nativo: true, soporta_imagen: true, activa: true });
          aviso(TV.creada(alias), 'ok');
        });
        ir('impresoras');
      } catch (e) { avisoError(e); }
    });
    return b;
  }

  // Tickets que recibió una impresora virtual (solo en la memoria del puente).
  // Prueba de acentos: imprime la línea de prueba con las 4 páginas de códigos, una por renglón, y
  // la persona elige el renglón que se lee bien. Devuelve la impresora actualizada, o null.
  const TAc = window.PRISMA_TEXTOS.acentos;
  async function probarAcentos(imp, b) {
    try { await conEspera(b, () => api.post(`/impresoras/${enc(imp.alias)}/paginas-codigos`)); } catch (e) { avisoError(e); return null; }
    const opciones = h('div', { class: 'opciones', role: 'radiogroup', 'aria-label': TAc.pregunta }, PAGINAS.map((p, k) => h('label', { class: 'opcion' },
      h('input', { type: 'radio', name: 'renglon', value: p.pagina, checked: p.pagina === imp.pagina_codigos }),
      h('span', { class: 'titulo-opcion' }, TAc.renglon(k + 1)), h('span', { class: 'tenue mono' }, TAc.empieza(`ESC t ${p.comando} ${p.pagina}`)))));
    const ninguno = h('div', { hidden: true }, UI.nota({ tipo: 'info', texto: TAc.ningunoTexto }));
    const eleccion = await dialogo({
      titulo: TAc.titulo, ancho: 640,
      cuerpo: h('div', { class: 'pila-amplia' }, h('p', {}, TAc.pregunta), opciones, ninguno),
      acciones: [
        { texto: TAc.ninguno, clase: 'texto', valor: null, antes: () => { ninguno.hidden = false; return false; } },
        { texto: 'Cancelar', valor: null },
        { texto: TAc.usar, clase: 'primario', valor: 'usar', antes: () => {
          if (opciones.querySelector('input:checked')) return true;
          aviso(TAc.elegirUno, 'aviso');
          return false;
        } },
      ],
    });
    if (eleccion !== 'usar') return null;
    const p = PAGINAS.find((x) => x.pagina === opciones.querySelector('input:checked').value);
    try {
      const actualizada = await api.put(`/impresoras/${enc(imp.alias)}`, { ...imp, pagina_codigos: p.pagina, comando_pagina_codigos: p.comando });
      aviso(TAc.guardado(imp.alias), 'ok');
      return actualizada;
    } catch (e) { avisoError(e); return null; }
  }

  // ---------------------------------------------------------------------------------------------
  // Asistente para agregar una impresora: 1) qué impresora es, 2) cómo está conectada y su nombre,
  // 3) ticket de prueba con las preguntas «¿salió?» y «¿se leen los acentos?».
  // ---------------------------------------------------------------------------------------------
  const TA = window.PRISMA_TEXTOS.asistente;
  // Impresoras del sistema que la detección encuentra pero no son de tickets (PDF, XPS, fax…).
  const NO_TICKETS = /pdf|xps|onenote|fax|document writer|send to|enviar a|microsoft/i;
  const ICONO_CONEXION = { red: 'red', usb: 'usb', bluetooth: 'bluetooth', archivo: 'carpeta' };
  const PATRON_ALIAS = /^[a-z0-9][a-z0-9_-]{0,31}$/;
  // Propone un alias a partir del nombre: «Caja principal» → caja-principal (único).
  function aliasDe(nombre, usados) {
    const base = nombre.normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase()
      .replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 28) || 'impresora';
    let alias = base;
    for (let n = 2; usados.has(alias); n++) alias = `${base}-${n}`;
    return alias;
  }

  async function asistenteImpresora(vista) {
    const usados = new Set((await api.get('/impresoras')).map((x) => x.alias));
    const w = { tipo: '80' };
    const indicador = UI.pasos(TA.pasos, 0);
    const cuerpo = h('div', { class: 'pila-amplia' });
    vaciar(vista, cintaDemo(), encabezado(TA.titulo, TA.subtitulo, h('a', { class: 'boton', href: rutaDe('impresoras') }, TA.cancelar)), indicador, cuerpo);
    const titulo = vista.querySelector('h1');
    titulo.tabIndex = -1;
    const irPaso = (n, pintar) => { indicador.ir(n); vaciar(cuerpo, pintar()); titulo.focus({ preventScroll: true }); window.scrollTo(0, 0); };
    const boton = (texto, variante, alPulsar, extra = {}) => {
      const b = UI.boton({ texto, variante, ...extra });
      b.addEventListener('click', () => alPulsar(b));
      return b;
    };

    function pasoTipo() {
      const opcion = (valor, ic, t, d) => h('label', { class: 'opcion' },
        h('input', { type: 'radio', name: 'tipo-imp', value: valor, checked: w.tipo === valor }),
        UI.icono(ic), h('span', { class: 'titulo-opcion' }, t), h('span', { class: 'tenue' }, d));
      const grupo = h('div', { class: 'opciones' },
        opcion('80', 'impresora', TA.tipo.ochenta, TA.tipo.ochentaTexto),
        opcion('58', 'ticket', TA.tipo.cincuenta, TA.tipo.cincuentaTexto),
        opcion('virtual', 'virtual', TA.tipo.virtual, TA.tipo.virtualTexto));
      return h('div', { class: 'pila-amplia' },
        h('fieldset', {}, h('legend', {}, TA.tipo.pregunta), grupo),
        h('p', { class: 'tenue pequeno' }, TA.tipo.ayuda),
        h('div', { class: 'fila-entre' }, h('span'), boton(TA.continuar, 'primario', () => {
          w.tipo = grupo.querySelector('input:checked').value;
          irPaso(1, pasoConexion);
        }, { icono: 'flecha-der' })));
    }

    function pasoConexion() {
      const virtual = w.tipo === 'virtual';
      const nombre = UI.entrada({ value: w.nombre || (virtual ? TV.nombre : ''), autocomplete: 'off' });
      const alias = UI.entrada({ mono: true, autocomplete: 'off', value: w.alias || (virtual ? aliasDe('prueba', usados) : '') });
      let aliasEditado = Boolean(w.alias);
      alias.addEventListener('input', () => { aliasEditado = true; });
      nombre.addEventListener('input', () => { if (!aliasEditado) alias.value = aliasDe(nombre.value, usados); });
      const cNombre = UI.campo({ id: 'as-nombre', etiqueta: TA.conexion.nombre, control: nombre, ayuda: TA.conexion.nombreAyuda,
        validar: (v) => (v.trim() ? '' : TA.conexion.nombreFalta) });
      const cAlias = UI.campo({ id: 'as-alias', etiqueta: TA.conexion.alias, control: alias, ayuda: TA.conexion.aliasAyuda,
        info: { texto: TA.conexion.aliasInfo, enlace: { destino: TAy.campos.alias } },
        validar: (v) => (!v ? TA.conexion.aliasFalta : !PATRON_ALIAS.test(v) ? TA.conexion.aliasMal : usados.has(v) ? TA.conexion.aliasUsado : '') });
      const nombres = h('div', { class: 'campos' }, cNombre, cAlias);
      const resultado = h('div', { 'aria-live': 'polite' });
      const atras = boton(TA.atras, '', () => irPaso(0, pasoTipo), { icono: 'flecha-izq' });
      const datosBase = () => {
        const mm = virtual ? 80 : Number(w.tipo);
        return { alias: alias.value.trim(), nombre: nombre.value.trim(), ancho_mm: mm, ...PERFILES[mm], pagina_codigos: 'CP858',
          comando_pagina_codigos: 19, soporta_corte: true, soporta_cajon: false, soporta_qr_nativo: true, soporta_codigo_barras_nativo: true,
          soporta_imagen: true, activa: true, confirmar_impresion: false };
      };
      const guardar = async (b, datos) => {
        w.nombre = datos.nombre; w.alias = datos.alias;
        try {
          w.imp = await conEspera(b, () => api.post('/impresoras', datos));
          usados.add(w.imp.alias);
          irPaso(2, pasoProbar);
        } catch (e) {
          if (e.codigo === 'ALIAS_DUPLICADO') { usados.add(datos.alias); cAlias.revisar(); alias.focus(); return; }
          vaciar(resultado, UI.nota({ tipo: 'error', titulo: e.message, texto: e.sugerencia }));
        }
      };

      if (virtual) {
        return h('div', { class: 'pila-amplia' }, UI.nota({ tipo: 'info', titulo: TV.crearTitulo, texto: TV.crearTexto }), nombres, resultado,
          h('div', { class: 'fila-entre' }, atras, boton(TA.conexion.guardarVirtual, 'primario', (b) => {
            if (UI.revisarCampos([cNombre, cAlias])) guardar(b, { ...datosBase(), tipo: 'virtual', destino: '' });
          })));
      }

      // Impresora física: detectadas, escaneo de red o dirección escrita a mano.
      let elegida = w.conexion || null;
      const falta = h('p', { class: 'campo-error', hidden: true }, UI.icono('error'), h('span', {}, TA.conexion.faltaDestino));
      const listaDet = h('div', { class: 'opciones', role: 'radiogroup', 'aria-label': TA.conexion.detectadas }, h('p', { class: 'tenue' }, TA.conexion.buscando));
      let detectadas = [];
      const opcionDetectada = (d) => {
        const tickets = !NO_TICKETS.test(`${d.destino} ${d.detalle || ''}`);
        const input = h('input', { type: 'radio', name: 'detectada', checked: Boolean(elegida && elegida.destino === d.destino && elegida.tipo === d.tipo) });
        input.addEventListener('change', () => {
          elegida = { tipo: d.tipo, destino: d.destino };
          destinoManual.value = ''; falta.hidden = true;
          if (!nombre.value.trim()) { nombre.value = d.destino; nombre.dispatchEvent(new Event('input')); }
        });
        return { tickets, el: h('label', { class: 'opcion' }, input,
          h('span', { class: 'fila' }, UI.icono(ICONO_CONEXION[d.tipo] || 'impresora'), h('span', { class: 'titulo-opcion' }, d.destino),
            !tickets && UI.insignia('desconocido', TA.conexion.noTickets, { icono: 'info' })),
          h('span', { class: 'tenue' }, [TA.conexion.tipos[d.tipo], d.detalle].filter(Boolean).join(' · '))) };
      };
      const pintarDetectadas = () => {
        const ops = detectadas.map(opcionDetectada).sort((a, b) => Number(b.tickets) - Number(a.tickets));
        vaciar(listaDet, ops.length ? ops.map((o) => o.el) : h('p', { class: 'tenue' }, TA.conexion.ninguna));
      };
      api.get('/detectar').then((l) => { detectadas = l; pintarDetectadas(); })
        .catch((e) => vaciar(listaDet, UI.nota({ tipo: 'error', titulo: e.message, texto: e.sugerencia })));
      const bRed = boton(TA.conexion.buscarRed, '', async (b) => {
        try {
          const l = await conEspera(b, () => api.get('/detectar?red=1'));
          const nuevas = l.filter((x) => !detectadas.some((y) => y.tipo === x.tipo && y.destino === x.destino));
          detectadas = [...detectadas, ...nuevas];
          pintarDetectadas();
          if (!l.some((x) => x.tipo === 'red')) aviso(TA.conexion.redNinguna);
        } catch (e) { avisoError(e); }
      }, { icono: 'buscar', chico: true, title: TA.conexion.buscarRedTexto });

      const tipoManual = UI.selector({ opciones: Object.entries(TA.conexion.tipos), valor: elegida ? elegida.tipo : 'red' });
      const destinoManual = UI.entrada({ mono: true, autocomplete: 'off' });
      const cTipo = UI.campo({ id: 'as-tipo', etiqueta: TA.conexion.tipo, control: tipoManual });
      const cDestino = UI.campo({ id: 'as-destino', etiqueta: TA.conexion.destino, control: destinoManual, ayuda: TA.conexion.ayudas[tipoManual.value] });
      tipoManual.addEventListener('change', () => { cDestino.querySelector('.ayuda').textContent = TA.conexion.ayudas[tipoManual.value]; });
      destinoManual.addEventListener('input', () => {
        if (!destinoManual.value.trim()) return;
        elegida = null; falta.hidden = true;
        for (const i of listaDet.querySelectorAll('input')) i.checked = false;
      });
      const manual = h('details', { class: 'avanzado' }, h('summary', {}, TA.conexion.manual), h('div', {}, h('div', { class: 'campos' }, cTipo, cDestino)));
      const conexion = () => (destinoManual.value.trim() ? { tipo: tipoManual.value, destino: destinoManual.value.trim() } : elegida);

      const casilla = (id, texto, marcado) => UI.casilla({ id, texto, marcado });
      const caps = {
        corte: casilla('as-corte', TA.conexion.cuchilla, true), cajon: casilla('as-cajon', TA.conexion.cajon, false),
        qr: casilla('as-qr', TA.conexion.qr, true), barras: casilla('as-barras', TA.conexion.barras, true), imagen: casilla('as-imagen', TA.conexion.imagenes, true),
      };
      const confirmarImp = UI.interruptor({ id: 'as-confirmar', texto: TA.conexion.confirmar, ayuda: TA.conexion.confirmarAyuda, marcado: false });
      const avanzadas = h('details', { class: 'avanzado' }, h('summary', {}, TA.conexion.avanzadas),
        h('div', {}, h('p', { class: 'tenue pequeno' }, TA.conexion.avanzadasTexto), h('fieldset', {}, Object.values(caps)), confirmarImp));

      const bProbar = boton(TA.conexion.probar, '', async (b) => {
        const c = conexion();
        if (!c) { falta.hidden = false; return; }
        try {
          await conEspera(b, () => api.post('/impresoras/probar-destino', { tipo: c.tipo, destino: c.destino, baudios: 0 }));
          vaciar(resultado, UI.nota({ tipo: 'ok', titulo: TA.conexion.probarOk }));
        } catch (e) { vaciar(resultado, UI.nota({ tipo: 'error', titulo: e.message, texto: e.sugerencia })); }
      }, { icono: 'red' });
      const bGuardar = boton(TA.conexion.guardar, 'primario', (b) => {
        const c = conexion();
        const nombresOk = UI.revisarCampos([cNombre, cAlias]);
        if (!c) { falta.hidden = false; if (nombresOk) listaDet.scrollIntoView({ block: 'center' }); return; }
        if (!nombresOk) return;
        w.conexion = c;
        guardar(b, { ...datosBase(), tipo: c.tipo, destino: c.destino, soporta_corte: caps.corte.input.checked,
          soporta_cajon: caps.cajon.input.checked, soporta_qr_nativo: caps.qr.input.checked, soporta_codigo_barras_nativo: caps.barras.input.checked,
          soporta_imagen: caps.imagen.input.checked, confirmar_impresion: confirmarImp.input.checked });
      });

      return h('div', { class: 'pila-amplia' },
        UI.tarjeta({ titulo: TA.conexion.detectadas, subtitulo: TA.conexion.detectadasTexto, icono: 'buscar', acciones: bRed }, listaDet, manual, falta),
        nombres, avanzadas, resultado,
        h('div', { class: 'fila-entre' }, atras, h('div', { class: 'acciones' }, bProbar, bGuardar)));
    }

    function pasoProbar() {
      const imp = w.imp;
      const caja = h('div', { class: 'pila-amplia' });
      const fallo = () => h('div', { class: 'acciones' },
        boton(TA.probar.reintentar, 'primario', imprimir, { icono: 'recargar' }),
        boton(TA.probar.editar, '', () => formularioImpresora(vista, w.imp), { icono: 'lapiz' }));
      const final = () => UI.nota({ tipo: 'ok', titulo: TA.probar.listo(w.imp.alias), texto: TA.probar.listoTexto, acciones: [
        h('a', { class: 'boton chico primario', href: rutaDe('aplicaciones') }, TA.probar.autorizar),
        h('a', { class: 'boton chico', href: rutaDe('impresoras') }, TA.probar.irImpresoras),
        boton(TA.probar.otra, 'texto', () => asistenteImpresora(vista), { chico: true }),
      ] });
      function mostrar(r) {
        const ticket = UI.tarjeta({ titulo: imp.virtual ? TA.probar.enviadoVirtual : TA.probar.asiDebeVerse, icono: 'ticket' },
          UI.ticketPapel({ imagen: r.imagen, anchoMm: imp.ancho_mm }));
        if (imp.virtual) {
          vaciar(caja, h('div', { class: 'rejilla' }, ticket, h('div', {}, final())));
          return;
        }
        const preguntas = h('div', { class: 'pila-amplia' });
        const preguntarAcentos = () => vaciar(preguntas, h('fieldset', {}, h('legend', {}, TA.probar.acentos),
          h('p', { class: 'tenue pequeno' }, TA.probar.acentosTexto),
          h('div', { class: 'acciones' },
            boton(TA.probar.acentosSi, 'primario', () => {
              // Lo comprobó en el papel: cuenta como «Revisar acentos» de Primeros pasos.
              api.post(`/impresoras/${enc(w.imp.alias)}/acentos-correctos`).catch(() => {});
              vaciar(preguntas, final());
            }),
            boton(TA.probar.acentosNo, '', async (b) => {
              const actualizada = await probarAcentos(w.imp, b);
              if (actualizada) { w.imp = actualizada; vaciar(preguntas, final()); }
            }))));
        vaciar(preguntas, h('fieldset', {}, h('legend', {}, TA.probar.salio), h('div', { class: 'acciones' },
          boton(TA.probar.si, 'primario', preguntarAcentos),
          boton(TA.probar.no, '', () => vaciar(preguntas, UI.nota({ tipo: 'aviso', titulo: TA.probar.noSalio, acciones: fallo() },
            h('ul', {}, TA.probar.ayudaNoSalio.map((t) => h('li', {}, t)))))))));
        vaciar(caja, UI.nota({ tipo: r.aviso ? 'aviso' : 'ok', titulo: TA.probar.enviado, texto: r.aviso }),
          h('div', { class: 'rejilla' }, ticket, UI.tarjeta({ titulo: TA.probar.revisa, icono: 'check' }, preguntas)));
      }
      async function imprimir() {
        vaciar(caja, h('p', { class: 'tenue', role: 'status' }, TA.probar.imprimiendo), UI.esqueleto({ lineas: 4 }));
        try {
          mostrar(await api.post(`/impresoras/${enc(imp.alias)}/diagnostico`));
        } catch (e) {
          vaciar(caja, UI.nota({ tipo: 'error', titulo: TA.probar.noSalio, texto: [e.message, e.sugerencia].filter(Boolean).join(' '), acciones: fallo() }));
        }
      }
      imprimir();
      return caja;
    }

    irPaso(0, pasoTipo);
  }

  // Formulario para editar el perfil de una impresora (las altas usan el asistente).
  function formularioImpresora(vista, existente) {
    const i = { ...existente };
    const campo = (id, etiqueta, control, ayuda) => h('label', { class: 'campo', for: id }, h('span', {}, etiqueta), control, ayuda && h('span', { class: 'ayuda' }, ayuda));
    const alias = h('input', { type: 'text', id: 'alias', class: 'mono', value: i.alias, required: true, pattern: '[a-z0-9][a-z0-9_\\-]{0,31}', autocomplete: 'off', placeholder: 'caja' });
    const nombre = h('input', { type: 'text', id: 'nombre', value: i.nombre, placeholder: 'Caja principal' });
    const tipo = h('select', { id: 'tipo' }, Object.entries(TIPOS_CONEXION).map(([v, t]) => h('option', { value: v, selected: v === i.tipo }, t)));
    const destino = h('input', { type: 'text', id: 'destino', class: 'mono', value: i.destino, required: true });
    const columnas = h('input', { type: 'number', id: 'columnas', min: 1, max: 200, value: i.columnas });
    const columnasB = h('input', { type: 'number', id: 'columnas-b', min: 1, max: 255, value: i.columnas_fuente_b });
    const anchoPx = h('input', { type: 'number', id: 'ancho-px', min: 8, max: 2048, value: i.ancho_px });
    const comando = h('input', { type: 'number', id: 'comando', min: 0, max: 255, value: i.comando_pagina_codigos });
    const pagina = h('select', { id: 'pagina', onchange: () => {
      const p = PAGINAS.find((x) => x.pagina === pagina.value); if (p) comando.value = p.comando;
    } }, PAGINAS.map((p) => h('option', { value: p.pagina, selected: p.pagina === i.pagina_codigos }, `${p.pagina} — ${p.etiqueta}`)));
    const baudios = h('input', { type: 'number', id: 'baudios', min: 0, value: i.baudios || 9600 });
    const filaBaudios = campo('baudios', 'Baudios', baudios, 'Velocidad del puerto serie. Casi todas las impresoras Bluetooth usan 9600.');
    const casilla = (id, texto, valor) => h('label', { class: 'casilla', for: id }, h('input', { type: 'checkbox', id, checked: valor }), h('span', {}, texto));
    const caps = {
      corte: casilla('cap-corte', 'Tiene cuchilla (corta el papel)', i.soporta_corte),
      cajon: casilla('cap-cajon', 'Tiene un cajón de dinero conectado', i.soporta_cajon),
      qr: casilla('cap-qr', 'Imprime códigos QR por comando', i.soporta_qr_nativo),
      barras: casilla('cap-barras', 'Imprime códigos de barras por comando', i.soporta_codigo_barras_nativo),
      imagen: casilla('cap-imagen', 'Imprime imágenes (logos)', i.soporta_imagen),
      activa: casilla('activa', 'Activa (las aplicaciones pueden usarla)', i.activa),
    };
    const confirmarImp = casilla('confirmar', 'Confirmar impresión: preguntar a la impresora si recibió cada ticket', i.confirmar_impresion);
    const filaConfirmar = h('div', { class: 'pila' }, confirmarImp, h('p', { class: 'ayuda' },
      'Sin esto, un ticket corto se da por impreso aunque la impresora se haya apagado mientras lo recibía. ',
      'Actívalo si la impresora responde a la consulta de estado (casi todas las ESC/POS): pruébalo con «Ticket de diagnóstico». ',
      'Si el diagnóstico falla con TIEMPO_AGOTADO, tu impresora no la admite: desactívalo.'));
    const ancho = h('div', { class: 'segmentos', role: 'group', 'aria-label': 'Ancho del papel' }, [58, 80].map((mm) =>
      h('button', { type: 'button', 'aria-pressed': String(i.ancho_mm === mm), onclick: (ev) => {
        i.ancho_mm = mm;
        for (const b of ancho.children) b.setAttribute('aria-pressed', String(b === ev.currentTarget));
        columnas.value = PERFILES[mm].columnas; columnasB.value = PERFILES[mm].columnas_fuente_b; anchoPx.value = PERFILES[mm].ancho_px;
      } }, `${mm} mm`)));
    const filaDestino = campo('destino', 'Destino', destino);
    const simulacion = h('select', { id: 'simulacion' }, Object.entries(TV.simulaciones).map(([v, t]) => h('option', { value: v, selected: v === i.simulacion }, t)));
    const retraso = h('input', { type: 'number', id: 'sim-retraso', min: 0, max: 30000, step: 500, value: i.simulacion_retraso_ms });
    const falla = h('input', { type: 'number', id: 'sim-falla', min: 0, max: 100, value: i.simulacion_falla_pct });
    const tickets = h('input', { type: 'number', id: 'sim-tickets', min: 1, max: 200, value: i.tickets_en_memoria });
    const filaRetraso = campo('sim-retraso', TV.retraso, retraso);
    const filaFalla = campo('sim-falla', TV.falla, falla);
    const bloqueVirtual = h('section', { class: 'tarjeta' }, h('h2', {}, TV.simulacionTitulo), h('p', { class: 'tenue pequeno' }, TV.simulacionAyuda),
      campo('simulacion', TV.simulacionEtiqueta, simulacion), h('div', { class: 'campos' }, filaRetraso, filaFalla),
      campo('sim-tickets', TV.tickets, tickets, TV.ticketsAyuda));
    const actualizarTipo = () => {
      const virtual = tipo.value === 'virtual';
      filaBaudios.hidden = tipo.value !== 'bluetooth';
      filaConfirmar.hidden = tipo.value !== 'red' && tipo.value !== 'bluetooth';
      filaDestino.hidden = virtual; destino.required = !virtual;
      bloqueVirtual.hidden = !virtual;
      filaRetraso.hidden = simulacion.value !== 'lenta';
      filaFalla.hidden = simulacion.value !== 'falla_aleatoria';
    };
    tipo.addEventListener('change', actualizarTipo);
    simulacion.addEventListener('change', actualizarTipo);
    actualizarTipo();

    const leer = () => ({
      alias: alias.value.trim(), nombre: nombre.value.trim() || alias.value.trim(), tipo: tipo.value, destino: destino.value.trim(),
      ancho_mm: i.ancho_mm, columnas: +columnas.value, columnas_fuente_b: +columnasB.value, ancho_px: +anchoPx.value,
      pagina_codigos: pagina.value, comando_pagina_codigos: +comando.value,
      soporta_corte: caps.corte.querySelector('input').checked, soporta_cajon: caps.cajon.querySelector('input').checked,
      soporta_qr_nativo: caps.qr.querySelector('input').checked, soporta_codigo_barras_nativo: caps.barras.querySelector('input').checked,
      soporta_imagen: caps.imagen.querySelector('input').checked, activa: caps.activa.querySelector('input').checked,
      baudios: tipo.value === 'bluetooth' ? +baudios.value : 0,
      confirmar_impresion: (tipo.value === 'red' || tipo.value === 'bluetooth') && confirmarImp.querySelector('input').checked,
      simulacion: simulacion.value, simulacion_retraso_ms: +retraso.value, simulacion_falla_pct: +falla.value, tickets_en_memoria: +tickets.value,
    });
    const resultado = h('p', { class: 'resultado', hidden: true });
    const bProbar = h('button', { class: 'boton', type: 'button', onclick: async () => {
      const d = leer();
      try {
        const r = await conEspera(bProbar, () => api.post('/impresoras/probar-destino', { tipo: d.tipo, destino: d.destino, baudios: d.baudios }));
        resultado.className = 'resultado nota ok'; resultado.textContent = r.mensaje || 'La impresora respondió.';
      } catch (e) { resultado.className = 'resultado nota error'; resultado.textContent = `${e.message} ${e.sugerencia || ''}`; }
      resultado.hidden = false;
    } }, 'Probar conexión');
    const bGuardar = h('button', { class: 'boton primario', type: 'submit' }, 'Guardar cambios');
    const form = h('form', { class: 'pila', onsubmit: async (ev) => {
      ev.preventDefault();
      if (!alias.checkValidity()) { aviso('El alias solo admite minúsculas, dígitos, - y _ (hasta 32 caracteres).', 'error'); alias.focus(); return; }
      const d = leer();
      try {
        await conEspera(bGuardar, () => api.put(`/impresoras/${enc(existente.alias)}`, d));
        aviso('Cambios guardados.', 'ok');
        ir('impresoras');
      } catch (e) { avisoError(e); }
    } },
    h('section', { class: 'tarjeta' }, h('h2', {}, 'Identificación'),
      h('div', { class: 'campos' },
        campo('alias', 'Alias', alias, 'Con este nombre la eligen las aplicaciones: minúsculas, dígitos, - o _.'),
        campo('nombre', 'Nombre descriptivo', nombre)),
      h('div', { class: 'campos' }, campo('tipo', 'Tipo de conexión', tipo), filaDestino, filaBaudios)),
    bloqueVirtual,
    h('section', { class: 'tarjeta' }, h('h2', {}, 'Papel'),
      h('div', { class: 'fila' }, ancho, h('span', { class: 'tenue pequeno' }, '58 mm: 32 columnas · 80 mm: 48 columnas')),
      h('details', { class: 'avanzado' }, h('summary', {}, 'Opciones avanzadas'),
        h('div', {},
          h('div', { class: 'campos' }, campo('pagina', 'Página de códigos', pagina, 'Si los acentos salen mal, usa «Probar acentos».'), campo('comando', 'Comando ESC t', comando)),
          h('div', { class: 'campos' }, campo('columnas', 'Columnas (fuente A)', columnas), campo('columnas-b', 'Columnas (fuente B)', columnasB), campo('ancho-px', 'Puntos por línea', anchoPx))))),
    h('section', { class: 'tarjeta' }, h('h2', {}, 'Capacidades'),
      h('fieldset', {}, h('legend', { class: 'tenue pequeno' }, 'Lo que no tenga, PRISMA lo resuelve de otra forma o lo omite con un aviso.'),
        caps.corte, caps.cajon, caps.qr, caps.barras, caps.imagen, caps.activa), filaConfirmar),
    resultado,
    h('div', { class: 'acciones' }, bProbar, bGuardar));

    vaciar(vista, cintaDemo(), encabezado(`Editar «${i.alias}»`, null, h('button', { class: 'boton', type: 'button', onclick: recargar }, 'Cancelar')), form);
    alias.focus();
  }

  // ---------------------------------------------------------------------------------------------
  // 3. Aplicaciones (en la API y la CLI se llaman «clientes»)
  // ---------------------------------------------------------------------------------------------
  const TAp = window.PRISMA_TEXTOS.aplicaciones;
  const TS = window.PRISMA_TEXTOS.solicitudes;
  const CDN_SDK = 'https://cdn.jsdelivr.net/gh/bayalad7/prisma@v0.2.0/sdk/js/dist/prisma.umd.js';
  const ICONO_NIVEL = { 1: 'aplicacion', 2: 'llave', 3: 'escudo' };
  const insigniaNivel = (n) => UI.insignia('acento', `${TAp.nivel(n)} · ${TAp.niveles[n].corto}`, { icono: ICONO_NIVEL[n] });
  // Tarjetas de opción para elegir el nivel; el recomendado (2) va primero.
  const opcionesNivel = (nombre, actual, niveles = [2, 3, 1]) => h('div', { class: 'opciones' }, niveles.map((n) => h('label', { class: 'opcion' },
    h('input', { type: 'radio', name: nombre, value: n, checked: actual === n }), UI.icono(ICONO_NIVEL[n]),
    h('span', { class: 'titulo-opcion' }, TAp.niveles[n].titulo, n === 2 && h('span', { class: 'tenue' }, ` (${TAp.recomendado})`)),
    h('span', { class: 'tenue' }, TAp.niveles[n].texto))));

  async function vistaClientes(vista) {
    const [clientes, impresoras] = await Promise.all([api.get('/clientes'), api.get('/impresoras')]);
    const botonAgregar = () => {
      const b = UI.boton({ texto: TAp.agregar, variante: 'primario', icono: 'mas' });
      b.addEventListener('click', () => formularioCliente(vista, null, impresoras));
      return b;
    };
    agregar(vista, encabezado(TAp.titulo, TAp.subtitulo, botonAgregar()),
      clientes.length === 0
        ? UI.vacio({ icono: 'aplicacion', titulo: TAp.vacio, texto: TAp.vacioTexto, accion: botonAgregar() })
        : h('div', { class: 'lista' }, clientes.map((c) => tarjetaCliente(c, vista, impresoras))));
  }

  function tarjetaCliente(c, vista, impresoras) {
    const ruta = `/clientes/${enc(c.id)}`;
    const D = TAp.datos, A = TAp.acciones, K = TAp.confirmar;
    const chips = (lista, f = (x) => x, clase = 'chip') => h('span', { class: 'fila' }, lista.map((x) => h('span', { class: clase }, f(x))));
    const accion = (texto, variante, alPulsar, icono) => {
      const b = UI.boton({ texto, variante, chico: true, icono });
      b.addEventListener('click', () => alPulsar(b));
      return b;
    };
    const hacer = async (b, f, mensaje) => {
      try { await conEspera(b, f); aviso(mensaje, 'ok'); recargar(); } catch (e) { avisoError(e); }
    };
    return UI.tarjeta({ titulo: c.nombre, icono: 'aplicacion',
      acciones: h('span', { class: 'fila' }, insigniaNivel(c.nivel), c.activo ? UI.insignia('ok', TAp.activa) : UI.insignia('desconocido', TAp.desactivada)) },
    h('dl', { class: 'datos' },
      // El id: para soporte, la CLI (prisma clientes …) y, en el nivel 3, para que su backend firme.
      h('dt', {}, D.id), h('dd', {}, h('span', { class: 'fila' }, h('span', { class: 'mono' }, c.id), botonCopiar(c.id)),
        c.nivel === 3 && h('span', { class: 'tenue pequeno' }, D.idNivel3)),
      h('dt', {}, D.origenes), h('dd', {}, c.origenes.length ? chips(c.origenes) : h('span', { class: 'tenue' }, D.origenesNinguna)),
      h('dt', {}, D.impresoras), h('dd', {}, c.todas_las_impresoras ? D.todas : (c.impresoras.length ? chips(c.impresoras) : h('span', { class: 'tenue' }, D.ninguna))),
      h('dt', {}, D.llave), h('dd', {}, c.llave_prefijo ? h('span', { class: 'mono' }, `${c.llave_prefijo}_…`)
        : h('span', { class: 'tenue' }, c.nivel === 2 ? D.llaveQuitada : D.sinLlave)),
      c.nivel === 3 && [h('dt', {}, D.llavePublica), h('dd', { class: 'mono pequeno' }, c.huella_publica || D.registrada)],
      h('dt', {}, D.especiales), h('dd', {}, c.operaciones_sensibles.length
        ? chips(c.operaciones_sensibles, (o) => TAp.especiales[o] || o, 'chip aviso') : h('span', { class: 'tenue' }, D.ningunPermiso))),
    h('div', { class: 'acciones' },
      accion(A.codigo, 'primario', () => codigoIntegracion(c, impresoras), 'codigo'),
      accion(A.editar, '', () => formularioCliente(vista, c, impresoras), 'lapiz'),
      accion(A.comprobar, '', () => irRuta(`/admin/pruebas/permisos?aplicacion=${enc(c.id)}`), 'escudo'),
      (c.nivel === 2 || c.llave_prefijo) && accion(c.llave_prefijo ? A.cambiarLlave : A.darLlave, '', async (b) => {
        if (c.llave_prefijo && !(await confirmar(K.cambiarTitulo(c.nombre), K.cambiarTexto, { boton: K.cambiarBoton, consecuencias: K.cambiarConsecuencias }))) return;
        try {
          const r = await conEspera(b, () => api.post(`${ruta}/llave`));
          await mostrarLlave(c, r.llave, impresoras);
          recargar();
        } catch (e) { avisoError(e); }
      }, 'llave'),
      c.llave_prefijo && accion(A.quitarLlave, 'peligro', async (b) => {
        if (!(await confirmar(K.quitarTitulo(c.nombre), c.nivel === 2 ? K.quitarTexto2 : K.quitarTexto3, { boton: K.quitarBoton, peligro: true }))) return;
        hacer(b, () => api.borrar(`${ruta}/llave`), TAp.hecho.llaveQuitada);
      }),
      accion(c.activo ? A.desactivar : A.activar, '', async (b) => {
        if (c.activo && !(await confirmar(K.desactivarTitulo(c.nombre), K.desactivarTexto,
          { boton: K.desactivarBoton, peligro: true, consecuencias: K.desactivarConsecuencias }))) return;
        hacer(b, () => api.put(ruta, { ...c, activo: !c.activo }), c.activo ? TAp.hecho.desactivada : TAp.hecho.activada);
      }),
      accion(A.eliminar, 'peligro', async (b) => {
        if (!(await confirmar(K.eliminarTitulo(c.nombre), K.eliminarTexto, { boton: K.eliminarBoton, peligro: true, consecuencias: K.eliminarConsecuencias }))) return;
        hacer(b, () => api.borrar(ruta), TAp.hecho.eliminada);
      }, 'papelera')));
  }

  function formularioCliente(vista, existente, impresoras) {
    const F = TAp.form;
    const c = existente ? { ...existente }
      : { nombre: '', nivel: 2, origenes: [], todas_las_impresoras: true, impresoras: [], operaciones_sensibles: [], exigir_llave: false, activo: true };
    const nombre = UI.entrada({ value: c.nombre, autocomplete: 'off' });
    const cNombre = UI.campo({ id: 'cli-nombre', etiqueta: F.nombre, control: nombre, ayuda: F.nombreAyuda, validar: (v) => (v.trim() ? '' : F.nombreFalta) });
    const niveles = opcionesNivel('nivel', c.nivel);
    const nivelElegido = () => Number(niveles.querySelector('input:checked').value);
    const cualElijo = h('details', { class: 'avanzado' }, h('summary', {}, TAp.cualElijo), h('div', {}, h('ul', {}, TAp.cualElijoTexto.map((t) => h('li', {}, t)))));

    // Páginas web (orígenes): solo esquema, nombre y puerto.
    const origenes = [...c.origenes];
    const chips = h('div', { class: 'fila' });
    const nuevoOrigen = UI.entrada({ tipo: 'url', mono: true, autocomplete: 'off', placeholder: 'https://pos.mitienda.mx' });
    const cOrigen = UI.campo({ id: 'cli-origen', etiqueta: F.origen, control: nuevoOrigen,
      info: { texto: TAy.origenInfo, enlace: { destino: TAy.campos.origen } } });
    const pintarOrigenes = () => vaciar(chips, origenes.length
      ? origenes.map((o, k) => h('span', { class: 'chip' }, o, h('button', { type: 'button', 'aria-label': F.quitarOrigen(o), onclick: () => { origenes.splice(k, 1); pintarOrigenes(); } }, '×')))
      : h('span', { class: 'tenue pequeno' }, F.origenesVacio));
    const normalizar = (v) => {
      let u;
      try { u = new URL(v); } catch { return null; }
      if (!/^https?:$/.test(u.protocol) || (u.pathname !== '/' && u.pathname !== '') || u.search || u.hash || v.includes('*')) return null;
      return u.origin;
    };
    // Agrega lo escrito; devuelve false si no es un origen válido.
    const agregarOrigen = () => {
      const v = nuevoOrigen.value.trim();
      if (!v) return true;
      const o = normalizar(v);
      if (!o) { cOrigen.ponerError(F.origenMal); nuevoOrigen.focus(); return false; }
      cOrigen.quitarError();
      if (!origenes.includes(o)) origenes.push(o);
      nuevoOrigen.value = '';
      pintarOrigenes();
      return true;
    };
    nuevoOrigen.addEventListener('keydown', (ev) => { if (ev.key === 'Enter') { ev.preventDefault(); agregarOrigen(); } });
    pintarOrigenes();
    const bAgregarOrigen = UI.boton({ texto: F.agregarOrigen, icono: 'mas' });
    bAgregarOrigen.addEventListener('click', agregarOrigen);

    // Permisos
    const todas = UI.interruptor({ id: 'cli-todas', texto: F.impresorasTodas, marcado: c.todas_las_impresoras });
    const listaImpresoras = h('div', { class: 'pila' }, impresoras.map((i) => UI.casilla({ id: `cli-imp-${i.alias}`, value: i.alias,
      marcado: c.impresoras.includes(i.alias),
      texto: h('span', { class: 'fila' }, h('span', {}, h('span', { class: 'mono' }, i.alias), ' ', h('span', { class: 'tenue' }, i.nombre)), insigniaVirtual(i)) })));
    const actualizarTodas = () => { listaImpresoras.hidden = todas.input.checked; };
    todas.input.addEventListener('change', actualizarTodas);
    actualizarTodas();
    const especiales = Object.entries(TAp.especiales).map(([op, texto]) => UI.casilla({ id: `cli-op-${op}`, value: op, texto, marcado: c.operaciones_sensibles.includes(op) }));

    // Nivel 3: llave pública del servidor de la aplicación.
    const llavePublica = h('textarea', { placeholder: '-----BEGIN PUBLIC KEY-----\n…\n-----END PUBLIC KEY-----' });
    const cLlave = UI.campo({ id: 'cli-publica', etiqueta: existente && existente.nivel === 3 ? F.llavePublicaReemplazo : F.llavePublica,
      control: llavePublica, ayuda: F.llavePublicaAyuda });
    const estadoLlave = h('div', { 'aria-live': 'polite' });
    let temporizador;
    llavePublica.addEventListener('input', () => {
      clearTimeout(temporizador);
      temporizador = setTimeout(async () => {
        const v = llavePublica.value.trim();
        vaciar(estadoLlave);
        cLlave.quitarError();
        if (!v) return;
        try {
          const r = await api.post('/validar-llave-publica', { llave: v });
          vaciar(estadoLlave, UI.nota({ tipo: 'ok', texto: F.llaveValida(r.formato, r.huella) }));
        } catch (e) { cLlave.ponerError(e.message); }
      }, 300);
    });
    const exigir = UI.casilla({ id: 'cli-exigir', texto: F.exigirLlave, marcado: c.exigir_llave });
    const bloqueFirma = UI.tarjeta({ titulo: F.seccionFirma, icono: 'escudo' }, cLlave, estadoLlave, exigir);
    const actualizarNivel = () => { bloqueFirma.hidden = nivelElegido() !== 3; };
    niveles.addEventListener('change', actualizarNivel);
    actualizarNivel();

    const resultado = h('div', { 'aria-live': 'polite' });
    const bGuardar = UI.boton({ texto: existente ? F.guardar : F.crear, variante: 'primario', tipo: 'submit' });
    const form = h('form', { class: 'pila-amplia', novalidate: true, onsubmit: async (ev) => {
      ev.preventDefault();
      vaciar(resultado);
      let valido = UI.revisarCampos([cNombre]);
      if (!agregarOrigen()) valido = false;
      const nivel = nivelElegido();
      if (nivel === 1 && origenes.length === 0) { cOrigen.ponerError(F.origenFaltaNivel1); if (valido) nuevoOrigen.focus(); valido = false; }
      if (nivel === 3 && !(existente && existente.nivel === 3) && !llavePublica.value.trim()) {
        cLlave.ponerError(F.llavePublicaFalta); if (valido) llavePublica.focus(); valido = false;
      }
      if (!valido) return;
      const datos = {
        nombre: nombre.value.trim(), nivel, origenes, todas_las_impresoras: todas.input.checked,
        impresoras: [...listaImpresoras.querySelectorAll('input:checked')].map((x) => x.value),
        operaciones_sensibles: especiales.filter((e) => e.input.checked).map((e) => e.input.value),
        llave_publica: llavePublica.value.trim() || undefined, exigir_llave: exigir.input.checked, activo: c.activo,
      };
      try {
        const r = await conEspera(bGuardar, () => (existente ? api.put(`/clientes/${enc(existente.id)}`, datos) : api.post('/clientes', datos)));
        if (r.llave) await mostrarLlave(r.cliente, r.llave, impresoras);
        else if (!existente && r.cliente.nivel === 3) await mostrarId(r.cliente, impresoras);
        else aviso(existente ? TAp.hecho.guardada : TAp.hecho.creada, 'ok');
        ir('aplicaciones');
      } catch (e) {
        if (e.codigo === 'ORIGEN_DUPLICADO') { cOrigen.ponerError([e.message, e.sugerencia].filter(Boolean).join(' ')); nuevoOrigen.focus(); return; }
        if (e.codigo === 'LLAVE_PUBLICA_INVALIDA') { cLlave.ponerError(e.message); llavePublica.focus(); return; }
        vaciar(resultado, UI.nota({ tipo: 'error', titulo: e.message, texto: e.sugerencia }));
      }
    } },
    UI.tarjeta({ titulo: F.seccionApp, icono: 'aplicacion' }, cNombre, h('fieldset', {}, h('legend', {}, F.nivel), niveles), cualElijo),
    bloqueFirma,
    UI.tarjeta({ titulo: F.seccionOrigenes, icono: 'red' }, h('p', { class: 'tenue pequeno' }, F.origenesTexto),
      h('div', { class: 'fila-campo' }, cOrigen, bAgregarOrigen), chips),
    UI.tarjeta({ titulo: F.seccionPermisos, icono: 'candado' },
      h('fieldset', {}, h('legend', {}, F.impresorasLeyenda), todas, listaImpresoras),
      h('hr', { class: 'separador' }),
      h('fieldset', {}, h('legend', {}, F.especialesLeyenda), h('p', { class: 'tenue pequeno' }, F.especialesAviso), especiales)),
    resultado,
    h('div', { class: 'acciones' }, h('a', { class: 'boton', href: rutaDe('aplicaciones') }, F.cancelar), bGuardar));

    vaciar(vista, cintaDemo(), encabezado(existente ? F.tituloEditar(c.nombre) : F.tituloNueva, existente ? null : F.subtituloNueva), form);
    nombre.focus();
  }

  // La llave se muestra una sola vez: con copiar, descargar como archivo y qué hacer con ella.
  async function mostrarLlave(cliente, llave, impresoras) {
    const L = TAp.llave;
    const url = estado.direccion || location.origin;
    const nivel3 = cliente.nivel === 3;
    const href = URL.createObjectURL(new Blob([L.archivo(cliente.nombre, llave, url, nivel3 && cliente.id)], { type: 'text/plain;charset=utf-8' }));
    await dialogo({
      titulo: L.titulo(cliente.nombre), ancho: 700,
      cuerpo: h('div', { class: 'pila-amplia' },
        UI.nota({ tipo: 'aviso', titulo: L.aviso, texto: L.avisoTexto }),
        h('div', { class: 'llave-caja' }, h('code', {}, llave), botonCopiar(llave)),
        // En el nivel 3 el backend necesita también el id para firmar.
        nivel3 && h('div', { class: 'pila' }, h('h3', {}, L.id), cajaId(cliente.id), h('p', { class: 'tenue pequeno' }, L.idTexto)),
        h('div', {}, h('a', { class: 'boton chico', href, download: `llave-${aliasDe(cliente.nombre, new Set())}.txt` }, UI.icono('descargar'), L.descargar)),
        h('div', { class: 'pila' }, h('h3', {}, L.queHago), h('p', { class: 'tenue pequeno' }, L.queHagoTexto)),
        bloqueCodigo(cliente, impresoras, llave)),
      acciones: [{ texto: L.listo, clase: 'primario', valor: true }],
    });
    URL.revokeObjectURL(href);
  }

  const cajaId = (id) => h('div', { class: 'llave-caja' }, h('code', {}, id), botonCopiar(id));

  // Nivel 3 sin llave: al crearla, el id que necesita su backend para firmar y qué hacer con él.
  async function mostrarId(cliente, impresoras) {
    const I = TAp.idCreado;
    await dialogo({
      titulo: I.titulo(cliente.nombre), ancho: 700,
      cuerpo: h('div', { class: 'pila-amplia' },
        h('p', {}, I.texto),
        cajaId(cliente.id),
        h('div', { class: 'pila' }, h('h3', {}, I.queHago), h('p', { class: 'tenue pequeno' }, I.queHagoTexto)),
        bloqueCodigo(cliente, impresoras, null)),
      acciones: [{ texto: I.listo, clase: 'primario', valor: true }],
    });
  }

  // Código para integrar la aplicación, con la impresora elegida y la llave si el nivel la usa.
  function bloqueCodigo(cliente, impresoras, llave) {
    const K = TAp.codigo;
    const url = estado.direccion || location.origin;
    const permitidas = impresoras.filter((i) => cliente.todas_las_impresoras || cliente.impresoras.includes(i.alias));
    const conLlave = cliente.nivel === 2 || (cliente.nivel === 3 && cliente.exigir_llave);
    const valorLlave = llave || K.llaveAqui;
    const preferida = permitidas.find((i) => i.estado === 'en_linea' && !i.virtual) || permitidas.find((i) => i.estado === 'en_linea') || permitidas[0];
    const caja = h('div', {});
    // Nivel 3: el servidor firma con el id real de la aplicación y la página o curl entregan el
    // documento firmado sin tocarlo (más la llave, si la aplicación la exige).
    const variantesFirma = (alias, saludo) => ({
      [K.pestanas.firma]: [
        'import { readFileSync } from "node:fs";',
        'import { firmar } from "./firmador.mjs"; // sdk/firmadores/node',
        '',
        `// ${K.enElServidor}`,
        'const firmado = firmar({',
        '  version: "1",',
        `  impresora: "${alias}",`,
        `  operaciones: [{ tipo: "texto", valor: "${saludo}" }, { tipo: "cortar" }],`,
        '}, {',
        `  clienteId: "${cliente.id}",`,
        '  llavePrivada: readFileSync("/ruta/segura/privada.pem", "utf8"),',
        '});',
      ],
      [K.pestanas.web]: [
        `<script src="${CDN_SDK}"></script>`,
        '<script>',
        '  const prisma = new Prisma({',
        `    url: "${url}",`,
        conLlave && `    llave: "${valorLlave}",`,
        '  });',
        `  // ${K.enLaPagina}`,
        '  fetch("/api/ticket-firmado", { method: "POST" })',
        '    .then((r) => r.json())',
        '    .then((firmado) => prisma.imprimirDocumento(firmado));',
        '</script>',
      ],
      [K.pestanas.fetch]: [
        `// ${K.firmadoDelServidor}`,
        `const respuesta = await fetch("${url}/v1/imprimir", {`,
        '  method: "POST",',
        '  headers: {',
        '    "Content-Type": "application/json",',
        conLlave && `    "Authorization": "Bearer ${valorLlave}",`,
        '  },',
        '  body: JSON.stringify(firmado),',
        '});',
        'const resultado = await respuesta.json();',
      ],
      [K.pestanas.curl]: [
        `prisma dev firmar documento.json --llave privada.pem --cliente ${cliente.id} --nuevo-id -o firmado.json`,
        `curl --cacert certificado.pem ${url}/v1/imprimir \\`,
        conLlave && `  -H "Authorization: Bearer ${valorLlave}" \\`,
        '  -H "Content-Type: application/json" --data-binary @firmado.json',
      ],
    });
    const pintar = (alias) => {
      const saludo = K.saludo(cliente.nombre.replace(/["\\]/g, ''));
      const variantes = cliente.nivel === 3 ? variantesFirma(alias, saludo) : {
        [K.pestanas.web]: [
          `<script src="${CDN_SDK}"></script>`,
          '<script>',
          '  const prisma = new Prisma({',
          `    url: "${url}",`,
          conLlave && `    llave: "${valorLlave}",`,
          '  });',
          `  prisma.imprimir("${alias}", (doc) => doc`,
          `    .texto("${saludo}", { alineacion: "centro" })`,
          '    .cortar());',
          '</script>',
        ],
        [K.pestanas.fetch]: [
          `const respuesta = await fetch("${url}/v1/imprimir", {`,
          '  method: "POST",',
          '  headers: {',
          '    "Content-Type": "application/json",',
          conLlave && `    "Authorization": "Bearer ${valorLlave}",`,
          '  },',
          '  body: JSON.stringify({',
          '    version: "1",',
          '    id: crypto.randomUUID(),',
          `    impresora: "${alias}",`,
          `    operaciones: [{ tipo: "texto", valor: "${saludo}" }, { tipo: "cortar" }],`,
          '  }),',
          '});',
          'const resultado = await respuesta.json();',
        ],
        [K.pestanas.curl]: [
          `curl --cacert certificado.pem ${url}/v1/imprimir \\`,
          conLlave && `  -H "Authorization: Bearer ${valorLlave}" \\`,
          '  -H "Content-Type: application/json" \\',
          '  --data-binary @documento.json',
        ],
      };
      const textos = Object.fromEntries(Object.entries(variantes).map(([k, ls]) => [k, ls.filter((l) => typeof l === 'string').join('\n')]));
      vaciar(caja, UI.codigo(textos, { etiqueta: K.titulo }));
    };
    const idSelector = `cod-imp-${cliente.id}`;
    const selector = permitidas.length > 1 && UI.selector({ opciones: permitidas.map((i) => [i.alias, i.virtual ? `${i.alias} (${TV.insignia})` : i.alias]), valor: preferida.alias });
    if (selector) selector.addEventListener('change', () => pintar(selector.value));
    pintar(preferida ? preferida.alias : 'caja');
    return h('div', { class: 'pila-amplia' }, h('h3', {}, K.titulo),
      permitidas.length === 0 && UI.nota({ tipo: 'aviso', texto: K.sinImpresoras }),
      selector && UI.campo({ id: idSelector, etiqueta: K.impresora, control: selector }),
      caja,
      h('p', { class: 'tenue pequeno' }, K.certificado, h('a', { href: rutaDe('ayuda'), target: '_blank', rel: 'noopener' }, K.certificadoEnlace), '.'),
      cliente.nivel === 1 && h('p', { class: 'tenue pequeno' }, K.nivel1),
      cliente.nivel === 3 && h('p', { class: 'tenue pequeno' }, K.nivel3));
  }

  function codigoIntegracion(c, impresoras) {
    dialogo({ titulo: TAp.codigo.dialogo(c.nombre), ancho: 720, cuerpo: bloqueCodigo(c, impresoras, null) });
  }

  // ---------------------------------------------------------------------------------------------
  // 4. Solicitudes: páginas web desconocidas que intentaron imprimir.
  // ---------------------------------------------------------------------------------------------
  const esEsteEquipo = (host) => host === 'localhost' || /^127\./.test(host) || host === '[::1]';
  const esRedLocal = (host) => /^(10|192\.168|169\.254)\./.test(host) || /^172\.(1[6-9]|2\d|3[01])\./.test(host) || /\.(local|lan|home)$/.test(host);
  // Dónde está la página y qué riesgos tiene aprobarla.
  function contextoOrigen(origen) {
    let u;
    try { u = new URL(origen); } catch { return { insignias: [], riesgos: [], host: origen }; }
    const host = u.hostname;
    const local = esEsteEquipo(host);
    const red = !local && esRedLocal(host);
    const sinCifrar = u.protocol === 'http:' && !local;
    return {
      host,
      insignias: [
        local ? UI.insignia('info', TS.esteEquipo, { icono: 'inicio' }) : red ? UI.insignia('info', TS.red, { icono: 'red' }) : UI.insignia('aviso', TS.internet, { icono: 'externo' }),
        sinCifrar && UI.insignia('aviso', TS.sinCifrar, { icono: 'candado' }),
      ].filter(Boolean),
      riesgos: [!local && !red && TS.dialogo.riesgoInternet, sinCifrar && TS.dialogo.riesgoHttp].filter(Boolean),
    };
  }

  async function vistaSolicitudes(vista) {
    const [solicitudes, clientes, impresoras] = await Promise.all([api.get('/solicitudes'), api.get('/clientes'), api.get('/impresoras')]);
    const pendientes = solicitudes.filter((s) => s.estado === 'pendiente');
    const rechazadas = solicitudes.filter((s) => s.estado === 'rechazada');
    estado.pendientes = pendientes.length;
    actualizarContador();
    const tarjeta = (s, rechazada) => {
      const ctx = contextoOrigen(s.origen);
      const accion = (texto, variante, f) => {
        const b = UI.boton({ texto, variante, chico: true });
        b.addEventListener('click', () => f(b));
        return b;
      };
      const quitar = (b, ruta, mensaje, opciones) => async () => {
        try { await conEspera(b, () => api.post(ruta, { origen: s.origen })); aviso(mensaje, 'ok', opciones); refrescarPendientes(); recargar(); } catch (e) { avisoError(e); }
      };
      // Rechazar no pide confirmación (se puede revertir); el aviso ofrece deshacerlo durante 10 s, y
      // después sigue en «Rechazadas».
      const deshacer = { duracion: 10_000, accion: { texto: TS.deshacer, alPulsar: async () => {
        try { await api.post('/solicitudes/restaurar', { origen: s.origen }); aviso(TS.restaurada(s.origen), 'ok'); refrescarPendientes(); if (enElPanel()) recargar(); } catch (e) { avisoError(e); }
      } } };
      return h('article', { class: 'tarjeta' },
        h('div', { class: 'fila-entre' }, h('span', { class: 'mono solicitud-origen' }, s.origen), h('span', { class: 'fila' }, ctx.insignias)),
        h('p', { class: 'tenue pequeno numeros' }, TS.intentos(s.intentos, hace(s.ultima_vez))),
        h('div', { class: 'acciones' },
          accion(TS.aprobar, 'primario', () => aprobar(s, clientes, impresoras)),
          !rechazada && accion(TS.rechazar, '', (b) => quitar(b, '/solicitudes/rechazar', TS.rechazada(s.origen), deshacer)()),
          rechazada && accion(TS.olvidar, 'texto', (b) => quitar(b, '/solicitudes/olvidar', TS.olvidada)())));
    };
    agregar(vista, encabezado(TS.titulo, TS.subtitulo),
      // El puente recuerda hasta 200; más de 150 suele indicar que alguien de la red registra orígenes inventados.
      solicitudes.length >= 150 && UI.nota({ tipo: 'aviso', texto: TS.muchas(solicitudes.length) }),
      pendientes.length === 0
        ? UI.vacio({ icono: 'solicitud', titulo: TS.vacio, texto: TS.vacioTexto })
        : h('section', { class: 'pila-amplia', 'aria-labelledby': 's-pendientes' },
          h('h2', { id: 's-pendientes' }, TS.pendientes(pendientes.length)),
          h('details', { class: 'avanzado' }, h('summary', {}, TS.queEs), h('div', {}, h('p', {}, TS.queEsTexto))),
          h('div', { class: 'lista' }, pendientes.map((s) => tarjeta(s, false)))),
      rechazadas.length > 0 && h('details', { class: 'avanzado' }, h('summary', {}, TS.rechazadas(rechazadas.length)),
        h('div', { class: 'lista' }, rechazadas.map((s) => tarjeta(s, true)))));
  }

  async function aprobar(s, clientes, impresoras) {
    const D = TS.dialogo;
    const ctx = contextoOrigen(s.origen);
    const nombre = UI.entrada({ value: ctx.host, autocomplete: 'off' });
    const cNombre = UI.campo({ id: 'apr-nombre', etiqueta: D.nombre, control: nombre, validar: (v) => (v.trim() ? '' : TAp.form.nombreFalta) });
    const niveles = opcionesNivel('apr-nivel', 2, [2, 1]);
    const existente = UI.selector({ opciones: clientes.map((c) => [c.id, `${c.nombre} (${TAp.niveles[c.nivel].corto})`]) });
    const modo = h('div', { class: 'opciones' },
      h('label', { class: 'opcion' }, h('input', { type: 'radio', name: 'modo', value: 'nuevo', checked: true }),
        h('span', { class: 'titulo-opcion' }, D.nueva), h('span', { class: 'tenue' }, D.nuevaTexto)),
      clientes.length > 0 && h('label', { class: 'opcion' }, h('input', { type: 'radio', name: 'modo', value: 'existente' }),
        h('span', { class: 'titulo-opcion' }, D.existente), h('span', { class: 'tenue' }, D.existenteTexto)));
    const nuevoBloque = h('div', { class: 'pila-amplia' }, cNombre, h('fieldset', {}, h('legend', {}, D.nivel), niveles), h('p', { class: 'tenue pequeno' }, D.nota));
    const existenteBloque = h('div', { hidden: true }, UI.campo({ id: 'apr-cliente', etiqueta: D.aplicacion, control: existente }));
    modo.addEventListener('change', () => {
      const v = modo.querySelector('input:checked').value;
      nuevoBloque.hidden = v !== 'nuevo';
      existenteBloque.hidden = v !== 'existente';
    });
    let respuesta;
    const ok = await dialogo({
      titulo: D.titulo, ancho: 660,
      cuerpo: h('div', { class: 'pila-amplia' },
        h('div', { class: 'fila' }, h('span', { class: 'chip' }, s.origen), ctx.insignias),
        h('p', {}, D.texto(s.origen)),
        ctx.riesgos.length > 0 && UI.nota({ tipo: 'aviso', texto: ctx.riesgos.join(' ') }),
        modo, nuevoBloque, existenteBloque),
      acciones: [{ texto: D.cancelar, valor: false }, { texto: D.aprobar, clase: 'primario', valor: true, antes: async () => {
        const v = modo.querySelector('input:checked').value;
        if (v === 'nuevo' && !cNombre.revisar()) return false;
        const cuerpo = v === 'nuevo'
          ? { origen: s.origen, nuevo: { nombre: nombre.value.trim(), nivel: Number(niveles.querySelector('input:checked').value) } }
          : { origen: s.origen, cliente_id: existente.value };
        respuesta = await api.post('/solicitudes/aprobar', cuerpo);
        return true;
      } }],
    });
    if (!ok) return;
    aviso(D.aprobada(s.origen), 'ok');
    if (respuesta && respuesta.llave) await mostrarLlave(respuesta.cliente, respuesta.llave, impresoras);
    await refrescarPendientes();
    recargar();
  }

  // ---------------------------------------------------------------------------------------------
  // 5. Trabajos: el historial, con el detalle de cada trabajo (qué pasó y qué hacer).
  // ---------------------------------------------------------------------------------------------
  const TT = window.PRISMA_TEXTOS.trabajos;
  const conAviso = (t) => t.estado === 'completado' && (t.advertencias || []).length > 0;
  function insigniaTrabajo(t) {
    const E = TT.estados;
    if (t.estado === 'fallido') return UI.insignia('error', E.fallido);
    if (t.estado === 'completado') return conAviso(t) ? UI.insignia('aviso', E.conAviso) : UI.insignia('ok', E.completado);
    return UI.insignia(t.estado === 'imprimiendo' ? 'info' : 'acento', E[t.estado] || t.estado);
  }
  const nombreAplicacion = (t) => (t.cliente_id === 'cli_panel' ? TT.panel : t.cliente_id === 'cli_consola' ? TT.consola : t.cliente_nombre || t.cliente_id);
  const referenciaDe = (t) => t.referencia || t.id;

  async function vistaTrabajos(vista) {
    cargarDiagramas().catch(() => {}); // para «Ver el diagrama» en el detalle de un trabajo
    const [clientes, impresoras, ajustes] = await Promise.all([api.get('/clientes'), api.get('/impresoras'), api.get('/ajustes').catch(() => ({}))]);
    const consulta = new URLSearchParams(location.search);
    // Estado: los del historial más «con aviso», que se filtra aquí (son trabajos impresos).
    const estados = [['', TT.filtros.todos], ['fallido', TT.filtros.fallidos], ['aviso', TT.filtros.conAviso], ['encolado', TT.filtros.enCola]];
    let estadoActual = estados.some(([v]) => v === consulta.get('estado')) ? consulta.get('estado') : '';
    const segmentos = h('div', { class: 'segmentos', role: 'group', 'aria-label': TT.filtros.etiqueta }, estados.map(([v, t]) => {
      const b = h('button', { type: 'button', 'aria-pressed': String(v === estadoActual) }, t);
      b.addEventListener('click', () => {
        estadoActual = v;
        for (const x of segmentos.children) x.setAttribute('aria-pressed', String(x === b));
        cargar();
      });
      return b;
    }));
    const selector = (id, etiqueta, opciones) => UI.campo({ id, etiqueta, control: UI.selector({ opciones: [['', TT.todas], ...opciones] }) });
    const filtros = {
      cliente: selector('f-cliente', TT.aplicacion, clientes.map((c) => [c.id, c.nombre]).concat([['cli_panel', TT.panel], ['cli_consola', TT.consola]])),
      impresora: selector('f-impresora', TT.impresora, impresoras.map((i) => [i.alias, i.virtual ? `${i.alias} (${TV.insignia})` : i.alias])),
      desde: UI.campo({ id: 'f-desde', etiqueta: TT.desde, control: UI.entrada({ tipo: 'date' }) }),
      hasta: UI.campo({ id: 'f-hasta', etiqueta: TT.hasta, control: UI.entrada({ tipo: 'date' }) }),
    };
    const mas = h('details', { class: 'avanzado' }, h('summary', {}, TT.masFiltros), h('div', {}, h('div', { class: 'campos' }, Object.values(filtros))));
    mas.addEventListener('change', () => cargar());
    const resultados = h('div', { class: 'pila-amplia', 'aria-live': 'polite' });
    async function cargar() {
      const q = new URLSearchParams();
      for (const [k, el] of Object.entries(filtros)) { const v = el.querySelector('select, input').value; if (v) q.set(k, v); }
      if (estadoActual === 'fallido' || estadoActual === 'encolado') q.set('estado', estadoActual);
      if (estadoActual === 'aviso') q.set('estado', 'completado');
      const filtrando = [...q.keys()].length > 0 || estadoActual !== '';
      vaciar(resultados, UI.esqueleto({ lineas: 4 }));
      try {
        let lista = await api.get(`/trabajos?${q}`);
        const enLimite = lista.length >= 100;
        if (estadoActual === 'aviso') lista = lista.filter(conAviso);
        const fallidos = lista.filter((t) => t.estado === 'fallido').length;
        vaciar(resultados,
          fallidos > 0 && estadoActual !== 'fallido' && UI.nota({ tipo: 'error', titulo: TT.hayFallidos(fallidos), texto: TT.hayFallidosTexto }),
          lista.length === 0
            ? UI.vacio({ icono: 'trabajos', titulo: filtrando ? TT.vacio : TT.vacioTodo, texto: filtrando ? TT.vacioTexto : TT.vacioTodoTexto })
            : tablaTrabajos(lista, impresoras),
          lista.length > 0 && h('p', { class: 'tenue pequeno' }, TT.cuenta(lista.length, ajustes.retencion_dias || 30), enLimite ? ` ${TT.limite}` : ''));
      } catch (e) { vaciar(resultados, UI.nota({ tipo: 'error', titulo: e.message, texto: e.sugerencia })); }
    }
    agregar(vista, encabezado(TT.titulo, TT.subtitulo), h('div', { class: 'pila' }, segmentos, mas), resultados);
    await cargar();
  }

  function tablaTrabajos(lista, impresoras) {
    const C = TT.columnas;
    return UI.tabla({
      etiqueta: TT.titulo, nombreFila: (t) => TT.fila(referenciaDe(t)), alAbrir: (t) => detalleTrabajo(t, impresoras),
      columnas: [
        { titulo: C.referencia, celda: (t) => h('span', { class: 'mono' }, t.referencia || TT.sinReferencia) },
        { titulo: C.fecha, celda: (t) => fechaHora(t.creado), clase: 'numeros' },
        { titulo: C.aplicacion, celda: nombreAplicacion },
        { titulo: C.impresora, celda: (t) => h('span', { class: 'fila' }, h('span', { class: 'mono' }, t.impresora_alias), insigniaVirtual(t)) },
        { titulo: C.estado, celda: insigniaTrabajo },
        { titulo: C.detalle, celda: (t) => {
          if (t.codigo_error) return (TT.codigos[t.codigo_error] || TT.detalle.desconocido).que;
          const adv = t.advertencias || [];
          return adv.length ? (TT.avisos[adv[0]] || { que: adv[0] }).que : '—';
        }, clase: 'envolver' },
        { titulo: C.duracion, celda: (t) => duracion(t.duracion_ms), clase: 'numeros' },
      ],
      filas: lista,
    });
  }

  // Detalle de un trabajo: para uno fallido, qué pasó, por qué suele pasar y qué hacer, con la acción
  // directa (probar la impresora); para uno con aviso, cada aviso explicado.
  function detalleTrabajo(t, impresoras) {
    const D = TT.detalle;
    const ref = referenciaDe(t);
    const imp = impresoras.find((i) => i.alias === t.impresora_alias);
    const titulo = t.estado === 'fallido' ? D.fallido(ref) : conAviso(t) ? D.conAviso(ref) : t.estado === 'completado' ? D.completado(ref) : D.pendiente(ref);
    const resultado = h('div', { 'aria-live': 'polite' });
    const botonProbar = () => {
      const b = UI.boton({ texto: D.probar, variante: 'primario', icono: 'red', chico: true });
      b.addEventListener('click', async () => {
        try {
          await conEspera(b, () => api.post(`/impresoras/${enc(imp.alias)}/probar`));
          vaciar(resultado, UI.nota({ tipo: 'ok', titulo: D.respondio }));
        } catch (e) { vaciar(resultado, UI.nota({ tipo: 'error', titulo: e.message, texto: e.sugerencia })); }
      });
      return b;
    };
    const irImpresoras = () => h('a', { class: 'boton chico', href: rutaDe('impresoras') }, D.irImpresoras);
    const lista = (titulo2, items, ordenada) => items.length > 0 && h('div', { class: 'pila' }, h('h3', {}, titulo2),
      h(ordenada ? 'ol' : 'ul', {}, items.map((x) => h('li', {}, x))));
    let cuerpo;
    if (t.estado === 'fallido') {
      const c = TT.codigos[t.codigo_error] || D.desconocido;
      cuerpo = [
        h('div', { class: 'pila' }, h('h3', {}, D.que), h('p', {}, c.que)),
        lista(D.porque, c.porque, false),
        lista(D.hacer, c.hacer, true),
        h('div', { class: 'acciones' }, imp && botonProbar(), irImpresoras(), enlaceManual(t.codigo_error),
          TAy.diagramasDeError[t.codigo_error] && botonDiagrama(TAy.diagramasDeError[t.codigo_error])),
        resultado,
        t.codigo_error !== 'PUENTE_REINICIADO' && h('p', { class: 'tenue pequeno' }, D.reintento),
      ];
    } else {
      const avisos = (t.advertencias || []).map((a) => ({ codigo: a, ...(TT.avisos[a] || { que: a, hacer: '' }) }));
      cuerpo = [
        avisos.length > 0 && h('div', { class: 'pila' }, h('h3', {}, D.avisos), h('ul', {}, avisos.map((a) => h('li', {}, a.que, a.hacer && [' ', h('span', { class: 'tenue' }, a.hacer)])))),
        avisos.some((a) => a.codigo === 'CARACTER_SUSTITUIDO') && imp && !imp.virtual && h('div', { class: 'acciones' }, (() => {
          const b = UI.boton({ texto: D.probarAcentos, chico: true });
          b.addEventListener('click', () => probarAcentos(imp, b));
          return b;
        })()),
      ];
      // Si la impresora es virtual y el ticket sigue en su rollo, se muestra.
      if (t.virtual && t.estado === 'completado' && imp && imp.virtual) {
        const img = h('img', { alt: D.verTicket, src: api.url(`/impresoras/${enc(imp.alias)}/rollo/${enc(t.id)}/imagen`) });
        const figura = h('figure', { class: 'ticket' }, h('div', { class: 'ticket-papel' }, img), h('figcaption', { class: 'tenue pequeno' }, D.verTicket));
        img.addEventListener('error', () => figura.remove());
        cuerpo.push(figura);
      }
    }
    dialogo({
      titulo, ancho: 640,
      cuerpo: h('div', { class: 'pila-amplia' },
        h('div', { class: 'fila' }, insigniaTrabajo(t), h('span', { class: 'tenue pequeno' }, `${fechaHora(t.creado)} · ${nombreAplicacion(t)} · `,
          h('span', { class: 'mono' }, t.impresora_alias)), insigniaVirtual(t)),
        t.virtual && h('p', { class: 'tenue pequeno' }, D.virtual),
        cuerpo,
        h('dl', { class: 'datos tenue pequeno' },
          h('dt', {}, D.trabajo), h('dd', { class: 'mono' }, t.id),
          t.duracion_ms !== null && t.duracion_ms !== undefined && [h('dt', {}, D.duracion), h('dd', {}, duracion(t.duracion_ms))],
          (t.codigo_error || (t.advertencias || []).length > 0) && [h('dt', {}, D.codigo), h('dd', { class: 'mono' }, [t.codigo_error, ...(t.advertencias || [])].filter(Boolean).join(', '))])),
    });
  }

  // ---------------------------------------------------------------------------------------------
  // 6. Imágenes (recursos)
  // ---------------------------------------------------------------------------------------------
  const TIm = window.PRISMA_TEXTOS.imagenes;
  async function vistaRecursos(vista) {
    const lista = await api.get('/recursos');
    let anchoVista = 576;
    let archivo = null;
    const nombre = UI.entrada({ mono: true, autocomplete: 'off' });
    const cNombre = UI.campo({ id: 'rec-nombre', etiqueta: TIm.nombre, control: nombre, ayuda: TIm.nombreAyuda,
      validar: (v) => (/^[a-z0-9][a-z0-9_-]{0,63}$/.test(v) ? '' : TIm.nombreMal) });
    const entrada = h('input', { type: 'file', id: 'rec-archivo', accept: 'image/png', class: 'solo-lector' });
    const errorArchivo = h('p', { class: 'campo-error', hidden: true }, UI.icono('error'), h('span'));
    const marcarArchivo = (m) => { errorArchivo.hidden = !m; errorArchivo.lastChild.textContent = m || ''; };
    const previa = h('div', { class: 'recurso-imagen' }, h('span', { class: 'tenue pequeno' }, TIm.previaVacia));
    const zona = h('label', { class: 'soltar', for: 'rec-archivo' }, UI.icono('imagen'), h('strong', {}, TIm.elegir), h('span', { class: 'pequeno' }, TIm.soltar));
    const anchoSel = h('div', { class: 'segmentos', role: 'group', 'aria-label': TIm.previaAncho }, [[384, '58 mm'], [576, '80 mm']].map(([px, t]) => {
      const b = h('button', { type: 'button', 'aria-pressed': String(px === anchoVista) }, t);
      b.addEventListener('click', () => {
        anchoVista = px;
        for (const x of anchoSel.children) x.setAttribute('aria-pressed', String(x === b));
        vistaPrevia();
      });
      return b;
    }));
    const vistaPrevia = async () => {
      if (!archivo) return;
      const fd = new FormData();
      fd.append('archivo', archivo); fd.append('ancho', anchoVista);
      try {
        const blob = await api.pedir('POST', '/recursos/vista-previa', fd);
        vaciar(previa, h('img', { src: URL.createObjectURL(blob), alt: TIm.previaTitulo }));
      } catch (e) { vaciar(previa, UI.nota({ tipo: 'error', titulo: e.message, texto: e.sugerencia })); }
    };
    const tomar = (f) => {
      if (!f) return;
      if (f.type !== 'image/png') { marcarArchivo(TIm.soloPng); return; }
      marcarArchivo('');
      archivo = f;
      if (!nombre.value) nombre.value = f.name.replace(/\.png$/i, '').toLowerCase().replace(/[^a-z0-9_-]+/g, '-').replace(/^-+/, '').slice(0, 64);
      vistaPrevia();
    };
    entrada.addEventListener('change', () => tomar(entrada.files[0]));
    zona.addEventListener('dragover', (ev) => { ev.preventDefault(); zona.classList.add('encima'); });
    zona.addEventListener('dragleave', () => zona.classList.remove('encima'));
    zona.addEventListener('drop', (ev) => { ev.preventDefault(); zona.classList.remove('encima'); tomar(ev.dataTransfer.files[0]); });
    const bGuardar = UI.boton({ texto: TIm.guardar, variante: 'primario', tipo: 'submit' });
    const form = h('form', { class: 'pila-amplia', novalidate: true, onsubmit: async (ev) => {
      ev.preventDefault();
      const nombreOk = UI.revisarCampos([cNombre]);
      if (!archivo) { marcarArchivo(TIm.falta); return; }
      if (!nombreOk) return;
      const fd = new FormData();
      fd.append('archivo', archivo); fd.append('nombre', nombre.value);
      try { await conEspera(bGuardar, () => api.pedir('POST', '/recursos', fd)); aviso(TIm.guardada(nombre.value), 'ok'); recargar(); } catch (e) { avisoError(e); }
    } },
    h('div', { class: 'rejilla' },
      h('div', { class: 'pila-amplia' }, h('div', { class: 'pila' }, zona, entrada, errorArchivo), cNombre),
      h('div', { class: 'pila' }, h('div', { class: 'fila-entre' }, h('span', { class: 'pequeno tenue' }, TIm.previaTitulo), anchoSel), previa)),
    h('div', {}, bGuardar));

    agregar(vista,
      encabezado(TIm.titulo, TIm.subtitulo),
      UI.tarjeta({ titulo: TIm.subir, icono: 'imagen' }, form),
      lista.length === 0 ? UI.vacio({ icono: 'imagen', titulo: TIm.vacio, texto: TIm.vacioTexto }) : h('div', { class: 'recursos' }, lista.map((r) => {
        const uso = `{"tipo": "imagen", "recurso": "${r.nombre}"}`;
        const eliminar = UI.boton({ texto: TIm.eliminar, variante: 'peligro', icono: 'papelera', chico: true });
        eliminar.addEventListener('click', async () => {
          if (!(await confirmar(TIm.eliminarTitulo(r.nombre), TIm.eliminarTexto, { boton: TIm.eliminar, peligro: true, consecuencias: TIm.eliminarConsecuencias }))) return;
          try { await api.borrar(`/recursos/${enc(r.nombre)}`); aviso(TIm.eliminada, 'ok'); recargar(); } catch (e) { avisoError(e); }
        });
        return h('article', { class: 'tarjeta' },
          h('div', { class: 'recurso-imagen' }, h('img', { src: api.url(`/recursos/${enc(r.nombre)}/imagen?ancho=576`), alt: TIm.alt(r.nombre), loading: 'lazy' })),
          h('div', { class: 'pila' }, h('span', { class: 'alias' }, r.nombre),
            h('span', { class: 'tenue pequeno numeros' }, TIm.medidas(r.ancho_px, r.alto_px, tamanoArchivo(r.bytes)))),
          h('div', { class: 'acciones' }, botonCopiar(uso, TIm.copiarUso), eliminar));
      })));
  }

  // ---------------------------------------------------------------------------------------------
  // Probar impresión: plantillas o JSON, vista previa con el raster de la impresora y envío.
  // ---------------------------------------------------------------------------------------------
  const TPr = window.PRISMA_TEXTOS.pruebas;
  const nuevoUUID = () => {
    if (crypto.randomUUID) return crypto.randomUUID();
    const b = crypto.getRandomValues(new Uint8Array(16));
    b[6] = (b[6] & 0x0f) | 0x40; b[8] = (b[8] & 0x3f) | 0x80;
    const x = [...b].map((n) => n.toString(16).padStart(2, '0')).join('');
    return `${x.slice(0, 8)}-${x.slice(8, 12)}-${x.slice(12, 16)}-${x.slice(16, 20)}-${x.slice(20)}`;
  };
  // "/operaciones/3/valor" → «Operación 4 (texto) · valor».
  function rutaLegible(ruta, doc) {
    const m = /^\/operaciones\/(\d+)(?:\/(.+))?$/.exec(ruta || '');
    if (!m) return ruta ? ruta.slice(1).replace(/\//g, ' · ') : '';
    const op = doc && Array.isArray(doc.operaciones) ? doc.operaciones[Number(m[1])] : null;
    return TPr.ruta(Number(m[1]) + 1, op && op.tipo, m[2] && m[2].replace(/\//g, ' · '));
  }
  // Renglón y columna de un error de JSON.parse (Chromium y Firefox los dan distinto).
  function errorSintaxis(texto, e) {
    const lc = /line (\d+) column (\d+)/.exec(e.message);
    const pos = /position (\d+)/.exec(e.message);
    let linea, columna;
    if (lc) [linea, columna] = [Number(lc[1]), Number(lc[2])];
    else if (pos) {
      const antes = texto.slice(0, Number(pos[1])).split('\n');
      [linea, columna] = [antes.length, antes[antes.length - 1].length + 1];
    }
    const mensaje = e.message.replace(/^JSON\.parse: /, '').replace(/ in JSON at position.*$| at line \d+ column \d+.*$/, '');
    return linea ? TPr.sintaxis(linea, columna, mensaje) : TPr.sintaxisSinLugar(mensaje);
  }
  const explicacion = (codigo) => TT.avisos[codigo] || TT.codigos[codigo] || null;
  const bloqueRespuesta = (datos) => h('details', { class: 'avanzado' }, h('summary', {}, TPr.respuesta),
    h('pre', { class: 'codigo', tabindex: '0', 'aria-label': TPr.respuesta }, h('code', {}, JSON.stringify(datos, null, 2))));

  async function vistaPruebas(vista) {
    const rollo = /^\/admin\/pruebas\/rollo\/([^/]+)$/.exec(location.pathname);
    if (rollo) return vistaRollo(vista, decodeURIComponent(rollo[1]));
    if (location.pathname === '/admin/pruebas/permisos') return vistaPermisos(vista);
    const impresoras = (await api.get('/impresoras')).filter((i) => i.activa);
    const bPermisos = UI.boton({ texto: TPr.permisos.boton, icono: 'escudo' });
    bPermisos.addEventListener('click', () => irRuta('/admin/pruebas/permisos'));
    agregar(vista, encabezado(TPr.titulo, TPr.subtitulo, bPermisos));
    if (impresoras.length === 0) {
      const agregarImp = UI.boton({ texto: TP.agregar, variante: 'primario', icono: 'mas' });
      agregarImp.addEventListener('click', () => ir('impresoras'));
      agregar(vista, UI.vacio({ icono: 'impresora', titulo: TPr.sinImpresoras, texto: TPr.sinImpresorasTexto, accion: [agregarImp, botonCrearVirtual()] }));
      return;
    }
    const consulta = new URLSearchParams(location.search);
    const enLinea = (i) => i.estado === 'en_linea';
    let imp = impresoras.find((i) => i.alias === consulta.get('impresora')) || impresoras.find((i) => i.virtual && enLinea(i))
      || impresoras.find(enLinea) || impresoras.find((i) => i.virtual) || impresoras[0];
    const plantillas = await api.get(`/pruebas/plantillas?impresora=${enc(imp.alias)}`);

    // El texto del editor es la única fuente del documento: las plantillas lo escriben y el modo
    // JSON lo edita, así cambiar de modo nunca lo pierde.
    let texto = '';
    let doc = null;
    const conImpresora = (d) => ({ ...d, impresora: imp.alias });

    const editor = h('textarea', { id: 'pr-json', class: 'editor-json', spellcheck: 'false', autocomplete: 'off', rows: '22',
      'aria-describedby': 'pr-json-ayuda pr-estado' });
    const estadoDoc = h('div', { id: 'pr-estado', 'aria-live': 'polite' });
    const previa = h('div', { class: 'previa-prueba' }, UI.vacio({ icono: 'ticket', titulo: TPr.vistaPreviaVacia }));
    const advertencias = h('div');
    const resultado = h('div', { 'aria-live': 'polite', class: 'pila' });
    const bEnviar = UI.boton({ texto: TPr.enviar(imp.alias), variante: 'primario', icono: 'ticket' });
    const bCodigo = UI.boton({ texto: TPr.copiarCodigo, icono: 'codigo' });
    const falta = h('div');
    const habilitar = (si) => { bEnviar.disabled = !si; bCodigo.disabled = !si; };

    const selImp = UI.selector({ opciones: impresoras.map((i) => [i.alias, i.virtual ? `${i.alias} (${TV.insignia})` : i.alias]), valor: imp.alias });
    const cImp = UI.campo({ id: 'pr-impresora', etiqueta: TPr.impresora, control: selImp, ayuda: TPr.impresoraAyuda });
    selImp.addEventListener('change', () => {
      imp = impresoras.find((i) => i.alias === selImp.value);
      bEnviar.lastChild.textContent = TPr.enviar(imp.alias);
      vaciar(resultado);
      if (doc) { doc.impresora = imp.alias; escribir(JSON.stringify(doc, null, 2)); }
      else revisar();
    });

    let turno = 0;
    let temporizador;
    async function revisar() {
      const mio = ++turno;
      vaciar(advertencias);
      if (!texto.trim()) { doc = null; vaciar(estadoDoc); editor.removeAttribute('aria-invalid'); habilitar(false); return; }
      try {
        // Si el texto salió del constructor, se conserva el mismo objeto: el constructor lo edita.
        if (!doc || texto !== JSON.stringify(doc, null, 2)) doc = JSON.parse(texto);
        if (!doc || typeof doc !== 'object' || Array.isArray(doc)) throw Object.assign(new Error(TPr.noObjeto), { propio: true });
      } catch (e) {
        doc = null;
        editor.setAttribute('aria-invalid', 'true');
        habilitar(false);
        vaciar(estadoDoc, h('p', { class: 'campo-error' }, UI.icono('error'), h('span', {}, e.propio ? e.message : errorSintaxis(texto, e))));
        return;
      }
      vaciar(estadoDoc, h('p', { class: 'tenue pequeno' }, TPr.revisando));
      try {
        const r = await api.post('/pruebas/vista-previa', { ...doc, id: doc.id || nuevoUUID() });
        if (mio !== turno) return;
        editor.removeAttribute('aria-invalid');
        habilitar(true);
        vaciar(estadoDoc, h('p', { class: 'estado-valido pequeno' }, UI.icono('ok'), h('span', {}, `${TPr.valido} ${TPr.bytes(r.bytes)}`)));
        vaciar(previa, UI.ticketPapel({ imagen: r.imagen, anchoMm: imp.ancho_mm }));
        if (r.advertencias.length) {
          vaciar(advertencias, UI.nota({ tipo: 'aviso', titulo: TPr.advertencias, texto: h('ul', { class: 'lista-avisos' }, r.advertencias.map((a) => {
            const ex = explicacion(a.codigo);
            return h('li', {}, ex ? ex.que : a.mensaje, ex && ex.hacer ? ` ${[].concat(ex.hacer).join(' ')}` : '',
              a.ruta ? h('span', { class: 'tenue' }, ` (${rutaLegible(a.ruta, doc)})`) : '');
          })) }));
        }
      } catch (e) {
        if (mio !== turno) return;
        editor.setAttribute('aria-invalid', 'true');
        habilitar(false);
        const detalles = (e.detalles || []).map((d) => h('li', {}, d.ruta ? h('strong', {}, `${rutaLegible(d.ruta, doc)}: `) : '', d.mensaje));
        vaciar(estadoDoc, UI.nota({ tipo: 'error', titulo: e.message, texto: h('div', {}, e.sugerencia ? h('p', {}, e.sugerencia) : '',
          detalles.length ? h('ul', { class: 'lista-avisos' }, detalles) : '') }));
        vaciar(previa, UI.vacio({ icono: 'ticket', titulo: TPr.vistaPreviaError }));
      }
    }
    function escribir(nuevo) {
      texto = nuevo;
      editor.value = nuevo;
      clearTimeout(temporizador);
      revisar();
    }

    const listaPlantillas = h('div', { class: 'opciones', role: 'radiogroup', 'aria-label': TPr.modos.plantillas }, plantillas.map((p) => {
      const [nombre, descripcion] = TPr.plantillas[p.id] || [p.id, ''];
      const radio = h('input', { type: 'radio', name: 'pr-plantilla', value: p.id });
      radio.addEventListener('change', () => {
        const irImagenes = UI.boton({ texto: TPr.irImagenes, chico: true });
        irImagenes.addEventListener('click', () => ir('imagenes'));
        vaciar(falta, p.faltan && p.faltan.length ? UI.nota({ tipo: 'aviso', texto: TPr.falta(p.faltan), acciones: irImagenes }) : '');
        vaciar(resultado);
        escribir(JSON.stringify(conImpresora(p.documento), null, 2));
      });
      return h('label', { class: 'opcion' }, radio, h('span', { class: 'titulo-opcion' }, nombre), h('span', { class: 'tenue' }, descripcion));
    }));
    editor.addEventListener('input', () => {
      texto = editor.value;
      for (const r of listaPlantillas.querySelectorAll('input')) r.checked = false;
      vaciar(falta);
      clearTimeout(temporizador);
      temporizador = setTimeout(revisar, 400);
    });

    // Constructor visual: edita doc.operaciones por índice y reescribe el JSON, así los tres modos
    // comparten un solo documento. Solo se vuelve a dibujar al agregar, mover o quitar.
    const K = TPr.constructor;
    const constructor = h('div', { class: 'pila constructor' });
    let recursos = null;
    const abiertas = new Set();
    function cambiado() {
      texto = JSON.stringify(doc, null, 2);
      editor.value = texto;
      for (const r of listaPlantillas.querySelectorAll('input')) r.checked = false;
      vaciar(falta);
      clearTimeout(temporizador);
      temporizador = setTimeout(revisar, 400);
    }
    // Pone un campo de la operación i (o lo quita si queda vacío, para que valga el de por defecto).
    const poner = (i, clave, valor) => {
      const op = doc.operaciones[i];
      if (valor === undefined || valor === '' || valor === false) delete op[clave]; else op[clave] = valor;
      cambiado();
      const r = document.querySelector(`#pr-op-${i}-resumen .operacion-resumen`);
      if (r) r.textContent = resumen(op);
    };
    const idCampo = (i, clave) => `pr-op-${i}-${clave}`;
    const campoTexto = (i, clave, etiqueta, { area, maximo, ayuda, numero, min, max } = {}) => {
      const op = doc.operaciones[i];
      const control = area ? h('textarea', { rows: '2', maxlength: maximo }) : UI.entrada({ tipo: numero ? 'number' : 'text', maxlength: maximo, min, max });
      control.value = op[clave] ?? '';
      control.addEventListener('input', () => poner(i, clave, numero ? (control.value === '' ? undefined : Number(control.value)) : control.value));
      return UI.campo({ id: idCampo(i, clave), etiqueta, control, ayuda });
    };
    const campoLista = (i, clave, etiqueta, opciones, porDefecto) => {
      const op = doc.operaciones[i];
      const control = UI.selector({ opciones, valor: op[clave] ?? porDefecto });
      control.addEventListener('change', () => poner(i, clave, control.value === porDefecto ? undefined : (/^\d+$/.test(control.value) ? Number(control.value) : control.value)));
      return UI.campo({ id: idCampo(i, clave), etiqueta, control });
    };
    const casillaOp = (i, clave, etiqueta) => {
      const c = UI.casilla({ id: idCampo(i, clave), texto: etiqueta, marcado: !!doc.operaciones[i][clave] });
      c.input.addEventListener('change', () => poner(i, clave, c.input.checked));
      return c;
    };
    const alineacion = (i) => campoLista(i, 'alineacion', K.alineacion, Object.entries(K.alineaciones), 'izquierda');
    const casillas = (...cs) => h('div', { class: 'fila' }, cs);

    function camposTexto(i) {
      const op = doc.operaciones[i];
      const actual = JSON.stringify(op.tamano || [1, 1]);
      const opciones = K.tamanos.map(([t, n]) => [JSON.stringify(t), n]);
      if (!opciones.some(([v]) => v === actual)) opciones.push([actual, K.tamanoOtro(...(op.tamano || [1, 1]))]);
      const tam = UI.selector({ opciones, valor: actual });
      tam.addEventListener('change', () => poner(i, 'tamano', tam.value === '[1,1]' ? undefined : JSON.parse(tam.value)));
      return [campoTexto(i, 'valor', K.valor, { area: true, maximo: 4096 }),
        h('div', { class: 'rejilla-campos' }, alineacion(i), UI.campo({ id: idCampo(i, 'tamano'), etiqueta: K.tamano, control: tam })),
        casillas(casillaOp(i, 'negrita', K.negrita), casillaOp(i, 'subrayado', K.subrayado), casillaOp(i, 'invertido', K.invertido))];
    }
    function camposFila(i) {
      const op = doc.operaciones[i];
      const columnas = op.columnas || (op.columnas = []);
      const lista = columnas.map((c, k) => {
        const id = (x) => `pr-op-${i}-c${k}-${x}`;
        const t = UI.entrada({ maxlength: 1024 });
        t.value = c.texto ?? '';
        t.addEventListener('input', () => { c.texto = t.value; cambiado(); document.querySelector(`#pr-op-${i}-resumen .operacion-resumen`).textContent = resumen(op); });
        const a = UI.entrada({ tipo: 'number', min: 1, max: 100, placeholder: 'auto' });
        a.value = typeof c.ancho === 'number' ? c.ancho : '';
        a.addEventListener('input', () => { if (a.value === '') delete c.ancho; else c.ancho = Number(a.value); cambiado(); });
        const al = UI.selector({ opciones: Object.entries(K.alineaciones), valor: c.alineacion || 'izquierda' });
        al.addEventListener('change', () => { if (al.value === 'izquierda') delete c.alineacion; else c.alineacion = al.value; cambiado(); });
        const quitar = UI.boton({ icono: 'papelera', etiqueta: K.quitarColumna(k + 1), variante: 'texto', chico: true });
        quitar.disabled = columnas.length === 1;
        quitar.addEventListener('click', () => { columnas.splice(k, 1); cambiado(); pintarConstructor(`pr-op-${i}-agregar-columna`); });
        return h('fieldset', { class: 'columna-fila' }, h('legend', {}, K.columna(k + 1)),
          UI.campo({ id: id('texto'), etiqueta: K.contenido, control: t }),
          h('div', { class: 'rejilla-campos' }, UI.campo({ id: id('ancho'), etiqueta: K.ancho, control: a, ayuda: K.anchoAyuda }),
            UI.campo({ id: id('alineacion'), etiqueta: K.alineacion, control: al })),
          h('div', {}, quitar));
      });
      const mas = UI.boton({ texto: K.agregarColumna, icono: 'mas', chico: true, id: `pr-op-${i}-agregar-columna` });
      mas.disabled = columnas.length >= 8;
      mas.addEventListener('click', () => { columnas.push({ texto: K.nuevo.columna }); cambiado(); pintarConstructor(`pr-op-${i}-c${columnas.length - 1}-texto`); });
      return [lista, casillas(casillaOp(i, 'negrita', K.negrita)), h('div', {}, mas)];
    }
    function camposImagen(i) {
      const op = doc.operaciones[i];
      if (op.base64) return [h('p', { class: 'tenue pequeno' }, K.incluida), h('div', { class: 'rejilla-campos' }, alineacion(i), campoTexto(i, 'ancho_max', K.anchoMax, { numero: true, min: 1, max: 100 }))];
      const nombres = (recursos || []).map((r) => r.nombre);
      if (op.recurso && !nombres.includes(op.recurso)) nombres.unshift(op.recurso);
      return [nombres.length ? campoLista(i, 'recurso', K.recurso, nombres.map((n) => [n, n]), null) : h('p', { class: 'tenue pequeno' }, K.sinRecursos),
        h('div', { class: 'rejilla-campos' }, alineacion(i), campoTexto(i, 'ancho_max', K.anchoMax, { numero: true, min: 1, max: 100 }))];
    }
    const CAMPOS = {
      texto: camposTexto,
      separador: (i) => [campoTexto(i, 'caracter', K.caracter, { maximo: 1, ayuda: K.caracterAyuda })],
      par: (i) => [h('div', { class: 'rejilla-campos' }, campoTexto(i, 'izquierda', K.izquierda, { maximo: 1024 }), campoTexto(i, 'derecha', K.derecha, { maximo: 1024 })),
        casillas(casillaOp(i, 'negrita', K.negrita))],
      fila: camposFila,
      salto: (i) => [campoTexto(i, 'lineas', K.lineas, { numero: true, min: 1, max: 10 })],
      qr: (i) => [campoTexto(i, 'valor', K.contenido, { maximo: 700 }),
        h('div', { class: 'rejilla-campos' }, alineacion(i), campoTexto(i, 'tamano', K.tamanoQr, { numero: true, min: 1, max: 16 }))],
      codigo_barras: (i) => [campoTexto(i, 'valor', K.contenido, { maximo: 255 }),
        h('div', { class: 'rejilla-campos' }, campoLista(i, 'simbologia', K.simbologia, ['CODE128', 'CODE39', 'EAN13', 'EAN8', 'UPCA', 'ITF'].map((s) => [s, s]), null),
          campoLista(i, 'texto', K.textoBarras, Object.entries(K.textosBarras), 'abajo'), alineacion(i))],
      imagen: camposImagen,
      cortar: (i) => [casillas(casillaOp(i, 'parcial', K.parcial))],
      abrir_cajon: (i) => [campoLista(i, 'pin', K.pin, [['2', 'Pin 2'], ['5', 'Pin 5']], '2')],
      raw: () => [h('p', { class: 'tenue pequeno' }, K.raw)],
    };
    // Resumen de una línea para la operación plegada.
    const resumen = (op) => {
      const t = op.valor ?? (op.izquierda !== undefined ? `${op.izquierda} · ${op.derecha ?? ''}` : null)
        ?? (op.columnas ? op.columnas.map((c) => c.texto).join(' | ') : null) ?? op.recurso ?? op.caracter ?? '';
      return String(t).replace(/\s+/g, ' ').slice(0, 48);
    };
    const NUEVAS = {
      texto: () => ({ tipo: 'texto', valor: K.nuevo.texto }),
      separador: () => ({ tipo: 'separador' }),
      par: () => ({ tipo: 'par', izquierda: K.nuevo.izquierda, derecha: K.nuevo.derecha }),
      fila: () => ({ tipo: 'fila', columnas: [{ texto: `${K.nuevo.columna} 1` }, { texto: `${K.nuevo.columna} 2`, alineacion: 'derecha' }] }),
      salto: () => ({ tipo: 'salto', lineas: 1 }),
      qr: () => ({ tipo: 'qr', valor: K.nuevo.qr, alineacion: 'centro' }),
      codigo_barras: () => ({ tipo: 'codigo_barras', valor: K.nuevo.barras, simbologia: 'CODE128', alineacion: 'centro' }),
      imagen: () => ({ tipo: 'imagen', recurso: recursos[0].nombre, alineacion: 'centro' }),
      cortar: () => ({ tipo: 'cortar' }),
      abrir_cajon: () => ({ tipo: 'abrir_cajon', pin: 2 }),
    };

    async function pintarConstructor(enfocar) {
      if (recursos === null) { try { recursos = await api.get('/recursos'); } catch { recursos = []; } }
      if (!doc) {
        const irJson = UI.boton({ texto: K.irJson, chico: true });
        irJson.addEventListener('click', () => modos.elegir('json'));
        vaciar(constructor, UI.nota({ tipo: 'aviso', titulo: K.sinDocumento, texto: K.sinDocumentoTexto, acciones: irJson }));
        return;
      }
      const ops = Array.isArray(doc.operaciones) ? doc.operaciones : (doc.operaciones = []);
      const referencia = UI.entrada({ maxlength: 64 });
      referencia.value = doc.referencia ?? '';
      referencia.addEventListener('input', () => { if (referencia.value) doc.referencia = referencia.value; else delete doc.referencia; cambiado(); });
      const copias = UI.entrada({ tipo: 'number', min: 1, max: 5 });
      copias.value = doc.copias ?? 1;
      copias.addEventListener('input', () => { if (copias.value === '' || copias.value === '1') delete doc.copias; else doc.copias = Number(copias.value); cambiado(); });

      const mover = (i, d, b) => {
        [ops[i], ops[i + d]] = [ops[i + d], ops[i]];
        const a = abiertas.has(i), z = abiertas.has(i + d);
        abiertas.delete(i); abiertas.delete(i + d);
        if (a) abiertas.add(i + d);
        if (z) abiertas.add(i);
        cambiado();
        pintarConstructor(`pr-op-${i + d}-${b}`);
      };
      const tarjetas = ops.map((op, i) => {
        const nombre = K.tipos[op.tipo] || op.tipo;
        const subir = UI.boton({ icono: 'abajo', etiqueta: K.subir(i + 1), variante: 'texto', chico: true, id: `pr-op-${i}-subir` });
        const bajar = UI.boton({ icono: 'abajo', etiqueta: K.bajar(i + 1), variante: 'texto', chico: true, id: `pr-op-${i}-bajar` });
        const quitar = UI.boton({ icono: 'papelera', etiqueta: K.eliminar(i + 1), variante: 'texto', chico: true });
        subir.classList.add('girar');
        subir.disabled = i === 0; bajar.disabled = i === ops.length - 1;
        subir.addEventListener('click', () => mover(i, -1, 'subir'));
        bajar.addEventListener('click', () => mover(i, 1, 'bajar'));
        quitar.addEventListener('click', () => {
          ops.splice(i, 1);
          const nuevas = [...abiertas].filter((k) => k !== i).map((k) => (k > i ? k - 1 : k));
          abiertas.clear(); nuevas.forEach((k) => abiertas.add(k));
          cambiado();
          pintarConstructor(ops.length ? `pr-op-${Math.min(i, ops.length - 1)}-resumen` : 'pr-agregar-tipo');
        });
        const campos = (CAMPOS[op.tipo] || (() => [h('p', { class: 'tenue pequeno' }, K.sinCampos)]))(i);
        const det = h('details', { class: 'operacion', open: abiertas.has(i) },
          h('summary', { id: `pr-op-${i}-resumen` }, h('span', { class: 'operacion-numero numeros' }, String(i + 1)), h('strong', {}, nombre),
            h('span', { class: 'tenue operacion-resumen' }, resumen(op))),
          h('div', { class: 'pila operacion-cuerpo' }, campos, h('div', { class: 'fila operacion-mover' }, subir, bajar, quitar)));
        det.addEventListener('toggle', () => { if (det.open) abiertas.add(i); else abiertas.delete(i); });
        return det;
      });

      const tipo = UI.selector({ opciones: Object.entries(NUEVAS).filter(([t]) => t !== 'imagen' || recursos.length).map(([t]) => [t, K.tipos[t]]), valor: 'texto' });
      const cTipo = UI.campo({ id: 'pr-agregar-tipo', etiqueta: K.agregarEtiqueta, control: tipo });
      const bAgregar = UI.boton({ texto: K.agregar, icono: 'mas' });
      bAgregar.addEventListener('click', () => {
        ops.push(NUEVAS[tipo.value]());
        abiertas.add(ops.length - 1);
        cambiado();
        pintarConstructor(`pr-op-${ops.length - 1}-resumen`);
      });
      vaciar(constructor,
        h('div', { class: 'rejilla-campos' }, UI.campo({ id: 'pr-referencia', etiqueta: K.referencia, control: referencia, ayuda: K.referenciaAyuda }),
          UI.campo({ id: 'pr-copias', etiqueta: K.copias, control: copias })),
        h('p', { class: 'tenue pequeno' }, K.operaciones(ops.length)),
        h('div', { class: 'operaciones' }, tarjetas),
        h('div', { class: 'agregar-operacion' }, cTipo, bAgregar));
      if (enfocar) {
        const el = document.getElementById(enfocar);
        if (el) { el.focus(); el.scrollIntoView({ block: 'nearest' }); }
      }
    }

    const modos = UI.pestanas([
      { id: 'plantillas', titulo: TPr.modos.plantillas, contenido: () => h('div', { class: 'pila' }, listaPlantillas, falta) },
      { id: 'constructor', titulo: TPr.modos.constructor, contenido: () => constructor },
      { id: 'json', titulo: TPr.modos.json, contenido: () => h('div', { class: 'campo' },
        h('label', { for: 'pr-json' }, TPr.jsonEtiqueta), editor, h('span', { class: 'ayuda', id: 'pr-json-ayuda' }, TPr.jsonAyuda)) },
    ], { etiqueta: TPr.modos.etiqueta, inicial: consulta.get('modo') || 'plantillas',
      alCambiar: (id) => { if (id === 'constructor') pintarConstructor(); } });

    bEnviar.addEventListener('click', async () => {
      if (!doc) return;
      vaciar(resultado);
      const enviada = imp;
      try {
        const r = await conEspera(bEnviar, () => api.post('/pruebas/imprimir', { ...doc, id: nuevoUUID() }));
        const { imagen, vista: _vista, aviso: _aviso, ...respuesta } = r;
        const verTrabajos = UI.boton({ texto: TPr.verTrabajos, chico: true, variante: 'texto' });
        verTrabajos.addEventListener('click', () => ir('trabajos'));
        const verTicket = enviada.virtual && UI.boton({ texto: TPr.verTicket, chico: true, icono: 'ticket' });
        if (verTicket) verTicket.addEventListener('click', () => dialogo({ titulo: TPr.verTicket, ancho: 520, cuerpo: UI.ticketPapel({ imagen, anchoMm: enviada.ancho_mm }) }));
        agregar(resultado,
          UI.nota({ tipo: r.advertencias.length ? 'aviso' : 'ok',
            titulo: enviada.virtual ? TPr.impresoVirtual(enviada.alias) : TPr.impreso(enviada.alias, r.duracion_ms),
            texto: r.advertencias.length ? h('ul', { class: 'lista-avisos' }, r.advertencias.map((a) => h('li', {}, (explicacion(a.codigo) || { que: a.mensaje }).que))) : null,
            acciones: [verTicket, enviada.virtual && Object.assign(UI.boton({ texto: TV.rollo.verRollo, chico: true, variante: 'texto' }),
            { onclick: () => verRollo(enviada) }), verTrabajos].filter(Boolean) }),
          bloqueRespuesta(respuesta));
      } catch (e) {
        const ex = explicacion(e.codigo);
        const hacer = (ex && ex.hacer) || e.sugerencia;
        // Los pasos de «qué hacer» citan «Probar la impresora»: el botón está aquí mismo.
        const probar = UI.boton({ texto: TP.acciones.probar, chico: true, icono: 'red' });
        probar.addEventListener('click', async () => {
          try { await conEspera(probar, () => api.post(`/impresoras/${enc(enviada.alias)}/probar`)); aviso(TP.respondio, 'ok'); } catch (e2) { avisoError(e2); }
        });
        agregar(resultado,
          UI.nota({ tipo: 'error', titulo: TPr.noImpreso(enviada.alias), texto: h('div', {}, h('p', {}, ex ? ex.que : e.message),
            Array.isArray(hacer) ? h('ol', { class: 'lista-avisos' }, hacer.map((x) => h('li', {}, x))) : hacer ? h('p', {}, hacer) : ''),
          acciones: [(e.codigo === 'IMPRESORA_INALCANZABLE' || e.codigo === 'TIEMPO_AGOTADO') && probar, enlaceManual(e.codigo)].filter(Boolean) }),
          bloqueRespuesta({ codigo: e.codigo, mensaje: e.message, sugerencia: e.sugerencia }));
      }
    });

    bCodigo.addEventListener('click', () => {
      if (!doc) return;
      const url = estado.direccion || location.origin;
      const { version: _version, id: _id, impresora, operaciones = [], ...opciones } = doc;
      const ops = operaciones.map((o) => `  ${JSON.stringify(o)},`).join('\n');
      const opc = Object.keys(opciones).length ? `, ${JSON.stringify(opciones).replace(/"(\w+)":/g, '$1: ')}` : '';
      const variantes = {
        [TPr.codigoPestanas.sdk]: [
          `const prisma = new Prisma({ url: "${url}", llave: "pk_..." });`,
          `const resultado = await prisma.imprimir("${impresora}", [`, ops, `]${opc});`,
        ].join('\n'),
        [TPr.codigoPestanas.curl]: [
          `curl --cacert prisma.crt ${url}/v1/imprimir \\`,
          '  -H "Content-Type: application/json" \\',
          '  -H "Authorization: Bearer pk_..." \\',
          "  --data-binary @- <<'FIN'",
          JSON.stringify({ ...doc, id: nuevoUUID() }, null, 2),
          'FIN',
        ].join('\n'),
      };
      dialogo({ titulo: TPr.codigoTitulo, ancho: 760, cuerpo: h('div', { class: 'pila' }, h('p', { class: 'tenue pequeno' }, TPr.codigoTexto), UI.codigo(variantes)) });
    });
    habilitar(false);

    agregar(vista, h('div', { class: 'area-pruebas' },
      UI.tarjeta({ titulo: TPr.documento, icono: 'codigo' }, cImp, modos),
      UI.tarjeta({ titulo: TPr.vistaPrevia, icono: 'ticket', clase: 'tarjeta-previa' }, estadoDoc, previa, advertencias,
        h('div', { class: 'acciones' }, bEnviar, bCodigo), resultado)));

    // Con una virtual se empieza por una venta; con una física, por su diagnóstico.
    const inicial = plantillas.find((p) => p.id === (imp.virtual ? 'venta-simple' : 'diagnostico')) || plantillas[0];
    const radio = listaPlantillas.querySelector(`input[value="${inicial.id}"]`);
    radio.checked = true;
    radio.dispatchEvent(new Event('change'));
  }

  // Rollo en vivo de una impresora virtual (/admin/pruebas/rollo/{alias}): los tickets llegan solos.
  // Se sondea cada 2 s solo mientras la pantalla está abierta y visible, con la cabecera que no
  // renueva la sesión: dejarla abierta no impide el cierre por inactividad.
  const TR = TV.rollo;
  const verRollo = (imp) => irRuta(`/admin/pruebas/rollo/${enc(imp.alias)}`);
  async function vistaRollo(vista, alias) {
    const virtuales = (await api.get('/impresoras')).filter((i) => i.virtual);
    const imp = virtuales.find((i) => i.alias === alias);
    const elegir = virtuales.length > 0 && UI.selector({ opciones: virtuales.map((i) => [i.alias, i.alias]), valor: alias });
    if (elegir) elegir.addEventListener('change', () => verRollo({ alias: elegir.value }));
    const cElegir = elegir && UI.campo({ id: 'ro-impresora', etiqueta: TR.impresora, control: elegir });
    const irPruebas = UI.boton({ texto: TR.irPruebas, icono: 'probar' });
    irPruebas.addEventListener('click', () => ir('pruebas', imp ? `?impresora=${enc(imp.alias)}` : ''));
    if (!imp) {
      agregar(vista, encabezado(TPr.titulo, null),
        virtuales.length ? h('div', { class: 'pila-amplia' }, UI.nota({ tipo: 'aviso', titulo: TR.noExiste(alias), texto: TR.noExisteTexto }), cElegir)
          : UI.vacio({ icono: 'virtual', titulo: TR.sinVirtuales, texto: TR.sinVirtualesTexto, accion: botonCrearVirtual() }));
      return;
    }
    document.title = TN.tituloPagina(TR.titulo(alias), nombreVisible(estado.nombrePuente));
    const ruta = `/impresoras/${enc(alias)}/rollo`;
    const sondeo = { sondeo: true };

    const estadoVivo = h('span', { class: 'en-vivo' }, h('span', { class: 'punto', 'aria-hidden': 'true' }), h('span', {}, TR.enVivo));
    const anuncio = h('div', { class: 'solo-lector', 'aria-live': 'polite' });
    let pausado = false;
    const bPausa = UI.boton({ texto: TR.pausar, chico: true, 'aria-pressed': 'false' });
    bPausa.addEventListener('click', () => {
      pausado = !pausado;
      bPausa.setAttribute('aria-pressed', String(pausado));
      bPausa.lastChild.textContent = pausado ? TR.reanudar : TR.pausar;
      pintarEstado();
      if (!pausado) revisar();
    });
    const pintarEstado = () => {
      const oculto = document.visibilityState !== 'visible';
      estadoVivo.classList.toggle('pausado', pausado || oculto);
      estadoVivo.lastChild.textContent = pausado ? TR.pausado : oculto ? TR.oculto : TR.enVivo;
    };
    const bVaciar = UI.boton({ texto: TV.vaciar, variante: 'peligro', icono: 'papelera', chico: true });
    bVaciar.addEventListener('click', async () => {
      if (!(await confirmar(TV.vaciarTitulo, TV.vaciarTexto, { boton: TV.vaciar, peligro: true, consecuencias: TV.vaciarConsecuencias }))) return;
      try { await api.borrar(ruta); aviso(TV.vaciado, 'ok'); revisar(); } catch (e) { avisoError(e); }
    });

    const rollo = h('div', { class: 'rollo' });
    const vacio = UI.vacio({ icono: 'ticket', titulo: TV.rolloVacio, texto: TV.rolloVacioTexto, accion: irPruebas.cloneNode(true) });
    vacio.querySelector('button')?.addEventListener('click', () => ir('pruebas', `?impresora=${enc(alias)}`));
    const tarjetas = new Map();
    const urls = [];

    const tarjeta = (t) => {
      const base = `${ruta}/${enc(t.trabajo_id)}`;
      const papel = h('div', { class: 'previa-prueba' }, UI.esqueleto({ lineas: 3 }));
      let imagen = '';
      // La imagen se pide con la misma cabecera del sondeo: que llegue un ticket no es actividad.
      api.pedir('GET', `${base}/imagen`, undefined, sondeo).then((blob) => {
        imagen = URL.createObjectURL(blob);
        urls.push(imagen);
        vaciar(papel, UI.ticketPapel({ imagen, anchoMm: imp.ancho_mm }));
      }).catch((e) => vaciar(papel, h('p', { class: 'tenue pequeno' }, e.message)));
      const ampliar = UI.boton({ texto: TR.ampliar, icono: 'acercar', chico: true });
      ampliar.addEventListener('click', () => imagen && dialogo({ titulo: [t.cliente_nombre, t.referencia].filter(Boolean).join(' · ') || t.trabajo_id,
        ancho: 760, cuerpo: UI.ticketPapel({ imagen, anchoMm: imp.ancho_mm, anchoPx: imp.ancho_mm === 58 ? 384 : 576 }) }));
      const traza = h('pre', { class: 'codigo', tabindex: '0', 'aria-label': TV.verTraza }, '…');
      const detalles = h('details', { class: 'avanzado' }, h('summary', {}, TV.verTraza), h('div', {}, traza));
      detalles.addEventListener('toggle', async () => {
        if (!detalles.open || traza.dataset.listo) return;
        try { traza.textContent = (await api.get(`${base}/traza`)).traza; traza.dataset.listo = '1'; } catch (e) { traza.textContent = e.message; }
      });
      const avisos = t.advertencias.length > 0 && UI.nota({ tipo: 'aviso', titulo: TR.advertencias,
        texto: h('ul', { class: 'lista-avisos' }, t.advertencias.map((c) => h('li', {}, (explicacion(c) || { que: c }).que))) });
      return h('article', { class: 'tarjeta ticket-rollo', 'aria-label': [t.cliente_nombre, t.referencia, fechaHora(t.recibido)].filter(Boolean).join(' · ') },
        h('div', { class: 'fila-entre' },
          h('div', { class: 'pila' }, h('strong', {}, [t.cliente_nombre, t.referencia].filter(Boolean).join(' · ') || t.trabajo_id),
            h('span', { class: 'tenue pequeno numeros' }, `${fechaHora(t.recibido)} · `, h('span', { class: 'mono' }, t.trabajo_id), ` · ${TV.bytes(t.bytes)}`)),
          t.advertencias.length > 0 && UI.insignia('aviso', String(t.advertencias.length))),
        avisos, papel,
        h('div', { class: 'acciones' }, ampliar, enlaceDescarga(`${base}/imagen?descargar=1`, TV.descargarPNG), enlaceDescarga(`${base}/bin`, TV.descargarBin)),
        detalles);
    };

    let primera = true;
    let ocupado = false;
    async function revisar() {
      if (ocupado || !vista.isConnected) return;
      ocupado = true;
      try {
        const lista = await api.get(ruta, sondeo);
        const ids = new Set(lista.map((t) => t.trabajo_id));
        for (const [id, el] of tarjetas) if (!ids.has(id)) { el.remove(); tarjetas.delete(id); }
        // La lista llega del más reciente al más antiguo: los nuevos van arriba, en orden.
        const nuevos = lista.filter((t) => !tarjetas.has(t.trabajo_id));
        for (const t of [...nuevos].reverse()) {
          const el = tarjeta(t);
          tarjetas.set(t.trabajo_id, el);
          rollo.prepend(el);
          if (!primera) el.classList.add('recien');
        }
        if (!primera && nuevos.length) anuncio.textContent = TR.llego(nuevos[0].cliente_nombre);
        vacio.hidden = tarjetas.size > 0;
        bVaciar.disabled = tarjetas.size === 0;
        primera = false;
      } catch (e) {
        if (e.estado !== 401) estadoVivo.lastChild.textContent = e.message;
      } finally { ocupado = false; }
    }
    const reloj = setInterval(() => {
      if (!vista.isConnected) { clearInterval(reloj); urls.forEach(URL.revokeObjectURL); document.removeEventListener('visibilitychange', alCambiarVisibilidad); return; }
      if (!pausado && document.visibilityState === 'visible') revisar();
    }, 2000);
    function alCambiarVisibilidad() { pintarEstado(); if (document.visibilityState === 'visible' && !pausado) revisar(); }
    document.addEventListener('visibilitychange', alCambiarVisibilidad);

    agregar(vista,
      encabezado(TR.titulo(alias), TR.subtitulo, irPruebas),
      h('div', { class: 'barra-rollo' },
        virtuales.length > 1 && cElegir,
        h('div', { class: 'pila' }, h('span', { class: 'fila' }, h('span', {}, TR.alias(alias)), UI.botonCopiar(alias, TR.copiarAlias)),
          h('span', { class: 'fila' }, estadoVivo, bPausa, bVaciar))),
      UI.nota({ tipo: 'info', texto: `${TV.rolloAviso} ${TR.sesion}` }),
      anuncio, vacio, rollo);
    pintarEstado();
    await revisar();
  }

  // Comprobador de permisos (/admin/pruebas/permisos): recorre, sin imprimir, los pasos de
  // autorización de /v1/imprimir para la situación descrita y dice dónde fallaría y qué hacer.
  const TPe = TPr.permisos;
  // A qué pantalla llevar según el código de rechazo.
  const ARREGLO = {
    ORIGEN_NO_AUTORIZADO: 'aplicaciones', CLIENTE_DESACTIVADO: 'aplicaciones', LLAVE_REQUERIDA: 'aplicaciones', LLAVE_INVALIDA: 'aplicaciones',
    ORIGEN_REQUERIDO: 'aplicaciones', FIRMA_REQUERIDA: 'aplicaciones', FIRMA_INVALIDA: 'aplicaciones', CREDENCIALES_REQUERIDAS: 'aplicaciones',
    IMPRESORA_NO_PERMITIDA: 'aplicaciones', OPERACION_NO_PERMITIDA: 'aplicaciones', IMPRESORA_NO_ENCONTRADA: 'impresoras',
  };
  async function vistaPermisos(vista) {
    const [clientes, impresoras] = await Promise.all([api.get('/clientes'), api.get('/impresoras'), cargarDiagramas().catch(() => null)]);
    agregar(vista, encabezado(TPe.titulo, TPe.subtitulo));
    if (clientes.length === 0) {
      const irAp = UI.boton({ texto: TPe.irAplicaciones, variante: 'primario' });
      irAp.addEventListener('click', () => ir('aplicaciones'));
      agregar(vista, UI.vacio({ icono: 'aplicacion', titulo: TPe.sinAplicaciones, texto: TPe.sinAplicacionesTexto, accion: irAp }));
      return;
    }
    const pedida = new URLSearchParams(location.search).get('aplicacion');
    const selApp = UI.selector({ opciones: clientes.map((c) => [c.id, c.activo ? c.nombre : TPe.desactivada(c.nombre)]),
      valor: (clientes.find((c) => c.id === pedida) || clientes[0]).id });
    const cApp = UI.campo({ id: 'pe-app', etiqueta: TPe.aplicacion, control: selApp });
    const nivel = h('p', { class: 'tenue pequeno' });
    const desde = h('div', { class: 'pila' });
    const otra = UI.entrada({ tipo: 'url', placeholder: 'https://', autocomplete: 'off' });
    const cOtra = UI.campo({ id: 'pe-otra', etiqueta: TPe.otraPaginaEtiqueta, control: otra, ayuda: TPe.otraPaginaAyuda });
    const llave = UI.casilla({ id: 'pe-llave', texto: TPe.conLlave });
    const firmado = UI.casilla({ id: 'pe-firmado', texto: TPe.firmado });
    const selImp = UI.selector({ opciones: impresoras.map((i) => [i.alias, i.activa ? i.alias : TPe.impresoraInactiva(i.alias)]) });
    const cImp = UI.campo({ id: 'pe-impresora', etiqueta: TPe.impresora, control: selImp });
    const cajon = UI.casilla({ id: 'pe-cajon', texto: TPe.cajon });
    const raw = UI.casilla({ id: 'pe-raw', texto: TPe.raw });
    const resultado = h('div', { class: 'pila', 'aria-live': 'polite' });

    const app = () => clientes.find((c) => c.id === selApp.value);
    // Al cambiar de aplicación se propone su caso típico: desde su página, con llave si la usa y
    // con firma si es de nivel 3.
    function proponer() {
      const c = app();
      nivel.textContent = TPe.nivel(c.nivel);
      const opcion = (valor, texto, marcado) => h('label', { class: 'casilla' },
        h('input', { type: 'radio', name: 'pe-desde', value: valor, checked: marcado }), h('span', {}, texto));
      vaciar(desde, c.origenes.map((o, k) => opcion(o, TPe.suPagina(o), k === 0)),
        opcion('otra', TPe.otraPagina, false), opcion('', TPe.servidor, c.origenes.length === 0));
      llave.input.checked = c.nivel === 2 || (c.nivel === 3 && c.exigir_llave);
      firmado.input.checked = c.nivel === 3;
      const permitida = impresoras.find((i) => i.activa && (c.todas_las_impresoras || c.impresoras.includes(i.alias))) || impresoras[0];
      if (permitida) selImp.value = permitida.alias;
      comprobar();
    }
    const origenElegido = () => {
      const v = desde.querySelector('input:checked')?.value ?? '';
      cOtra.hidden = v !== 'otra';
      return v === 'otra' ? otra.value.trim() : v;
    };

    let turno = 0;
    async function comprobar() {
      const mio = ++turno;
      const origen = origenElegido();
      if (cOtra.hidden === false && !origen) { vaciar(resultado); return; }
      const datos = { cliente_id: selApp.value, origen, con_llave: llave.input.checked, firmado: firmado.input.checked, impresora: selImp.value,
        operaciones: ['texto', cajon.input.checked && 'abrir_cajon', raw.input.checked && 'raw'].filter(Boolean) };
      let r;
      try { r = await api.post('/pruebas/permisos', datos); } catch (e) { if (mio === turno) vaciar(resultado, UI.nota({ tipo: 'error', titulo: e.message, texto: e.sugerencia })); return; }
      if (mio !== turno) return;
      const elegida = app();
      const otraApp = r.identificado && r.identificado.id !== elegida.id ? r.identificado.nombre : null;
      const origenDeOtra = r.origen_de && r.origen_de.id !== elegida.id ? r.origen_de.nombre : null;
      const ORDEN = ['origen', 'llave', 'firma', 'impresora', 'operaciones', 'disponible'];
      const lista = h('ol', { class: 'pasos-permisos' }, ORDEN.map((id) => {
        const p = r.pasos.find((x) => x.id === id);
        const estadoPaso = p ? p.estado : 'pendiente';
        const ic = { ok: 'ok', falla: 'error', no_aplica: 'menos', pendiente: 'desconocido' }[estadoPaso];
        let texto;
        if (!p) texto = TPe.noLlego;
        else if (p.estado === 'falla') texto = h('span', {}, h('strong', {}, p.mensaje), ' ', h('span', { class: 'mono pequeno' }, p.codigo));
        else if (p.estado === 'no_aplica') texto = TPe.noAplica[id];
        else if (id === 'origen' && origenDeOtra) texto = TPe.ok.origenOtra(origenDeOtra);
        else texto = typeof TPe.ok[id] === 'function' ? TPe.ok[id](selImp.value) : TPe.ok[id];
        return h('li', { class: `paso-permiso paso-${estadoPaso}` }, UI.icono(ic),
          h('div', { class: 'pila' }, h('span', { class: 'titulo-paso' }, TPe.pasos[id]), h('span', { class: 'pequeno' }, texto)));
      }));
      const falla = r.pasos.find((p) => p.estado === 'falla');
      let resumen;
      if (!falla) {
        resumen = UI.nota({ tipo: otraApp ? 'aviso' : 'ok', titulo: TPe.todoBien(r.identificado.nombre, selImp.value), texto: otraApp ? TPe.comoOtra(otraApp) : null });
      } else {
        const destino = ARREGLO[falla.codigo];
        const boton = destino && UI.boton({ texto: TPe.acciones[destino], chico: true });
        if (boton) boton.addEventListener('click', () => ir(destino));
        resumen = UI.nota({ tipo: 'error', titulo: TPe.rechazo(r.estado_http, r.codigo),
          texto: h('p', {}, h('strong', {}, `${TPe.resolver}: `), TPe.sugerencias[falla.codigo] ? TPe.sugerencias[falla.codigo](elegida.nombre) : falla.sugerencia), acciones: boton });
      }
      const niveles = UI.boton({ texto: TPe.verManual, icono: 'libro', chico: true, variante: 'texto' });
      niveles.addEventListener('click', () => abrirAyuda('integradores/niveles-de-seguridad.md').catch(avisoError));
      const delError = falla && TAy.diagramasDeError[falla.codigo];
      vaciar(resultado, resumen, lista, h('div', { class: 'acciones' },
        botonDiagrama(delError || '07-orden-autorizacion'), delError && delError !== '07-orden-autorizacion' && botonDiagrama('07-orden-autorizacion'), niveles));
    }

    selApp.addEventListener('change', proponer);
    desde.addEventListener('change', comprobar);
    let espera;
    otra.addEventListener('input', () => { clearTimeout(espera); espera = setTimeout(comprobar, 400); });
    for (const x of [llave.input, firmado.input, selImp, cajon.input, raw.input]) x.addEventListener('change', comprobar);

    agregar(vista, h('div', { class: 'area-pruebas' },
      UI.tarjeta({ titulo: TPe.situacion, icono: 'aplicacion' }, cApp, nivel,
        h('fieldset', { class: 'grupo-campos' }, h('legend', {}, TPe.desde), desde, cOtra),
        h('div', { class: 'pila' }, llave, firmado), cImp,
        h('fieldset', { class: 'grupo-campos' }, h('legend', {}, TPe.sensibles), cajon, raw)),
      UI.tarjeta({ titulo: TPe.resultado, icono: 'escudo', clase: 'tarjeta-previa' }, resultado)));
    proponer();
  }

  // ---------------------------------------------------------------------------------------------
  // 7. Ajustes: cuatro pestañas (General, Seguridad y acceso, HTTPS, Respaldo); la pestaña va en
  // la dirección (?pestana=https) para enlazarla desde otras pantallas.
  // ---------------------------------------------------------------------------------------------
  const TAj = window.PRISMA_TEXTOS.ajustes;
  function enlaceDescarga(ruta, texto) {
    const a = h('a', { class: 'boton', href: api.url(ruta) }, UI.icono('descargar'), texto);
    if (window.PRISMA_DEMO) a.addEventListener('click', (ev) => { ev.preventDefault(); aviso('En el prototipo no se descargan archivos.'); });
    return a;
  }
  const entero = (min, max, mensaje) => (v) => (/^\d+$/.test(v.trim()) && Number(v) >= min && Number(v) <= max ? '' : mensaje);

  async function vistaAjustes(vista) {
    const a = await api.get('/ajustes');
    const pestana = new URLSearchParams(location.search).get('pestana');
    // Una tarjeta con formulario: valida sus campos, envía y avisa.
    const formulario = (titulo, icono, texto, contenido, validables, enviar, textoBoton = TAj.guardar) => {
      const b = UI.boton({ texto: textoBoton, variante: 'primario', chico: true, tipo: 'submit' });
      const form = h('form', { class: 'pila-amplia', novalidate: true, onsubmit: async (ev) => {
        ev.preventDefault();
        if (!UI.revisarCampos(validables)) return;
        try { await conEspera(b, enviar); } catch (e) { avisoError(e); }
      } }, contenido, h('div', {}, b));
      return UI.tarjeta({ titulo, icono }, texto && h('p', { class: 'tenue pequeno' }, texto), form);
    };
    const guardado = () => aviso(TAj.guardado, 'ok');

    function general() {
      const G = TAj.general;
      const nombre = UI.entrada({ value: a.nombre_puente });
      const cNombre = UI.campo({ id: 'aj-nombre', etiqueta: G.nombre, control: nombre, ayuda: G.nombreAyuda,
        validar: (v) => (v.trim() && [...v.trim()].length <= 60 ? '' : G.nombreMal) });
      const limIP = UI.entrada({ tipo: 'number', value: a.limite_ip, min: 0, max: 100000 });
      const limCli = UI.entrada({ tipo: 'number', value: a.limite_cliente, min: 0, max: 100000 });
      const cIP = UI.campo({ id: 'aj-ip', etiqueta: G.porEquipo, control: limIP, validar: entero(0, 100000, G.limiteMal) });
      const cCli = UI.campo({ id: 'aj-cliente', etiqueta: G.porAplicacion, control: limCli, validar: entero(0, 100000, G.limiteMal) });
      const dias = UI.entrada({ tipo: 'number', value: a.retencion_dias, min: 1, max: 3650 });
      const cDias = UI.campo({ id: 'aj-retencion', etiqueta: G.dias, control: dias, validar: entero(1, 3650, G.diasMal) });
      const buscar = UI.interruptor({ id: 'aj-actualizaciones', texto: G.buscar, marcado: a.buscar_actualizaciones });
      return h('div', { class: 'rejilla' },
        formulario(G.puente, 'inicio', null, [cNombre], [cNombre], async () => {
          await api.put('/ajustes', { nombre_puente: nombre.value.trim() });
          guardado(); refrescarPendientes();
        }),
        formulario(G.limites, 'ajustes', G.limitesTexto, [h('div', { class: 'campos' }, cIP, cCli)], [cIP, cCli], async () => {
          await api.put('/ajustes', { limite_ip: Number(limIP.value), limite_cliente: Number(limCli.value) });
          guardado();
        }),
        formulario(G.historial, 'trabajos', G.historialTexto, [cDias], [cDias], async () => {
          await api.put('/ajustes', { retencion_dias: Number(dias.value) });
          guardado();
        }),
        formulario(G.actualizaciones, 'recargar', G.actualizacionesTexto, [buscar], [], async () => {
          await api.put('/ajustes', { buscar_actualizaciones: buscar.input.checked });
          guardado();
        }));
    }

    function seguridad() {
      const S = TAj.seguridad;
      const clave = (id, etiqueta, auto, ayuda, validar) => {
        const input = UI.entrada({ tipo: 'password', autocomplete: auto });
        return { input, campo: UI.campo({ id, etiqueta, control: input, ayuda, validar }) };
      };
      const actual = clave('aj-actual', S.actual, 'current-password', null, (v) => (v ? '' : S.actualFalta));
      const nueva = clave('aj-nueva', S.nueva, 'new-password', S.nuevaAyuda, (v) => (v.length >= 10 ? '' : S.nuevaCorta));
      const repetir = clave('aj-repetir', S.repetir, 'new-password', null, (v) => (v === nueva.input.value ? '' : S.noCoinciden));
      const cambio = formulario(S.contrasena, 'candado', null, [actual.campo, nueva.campo, repetir.campo], [actual.campo, nueva.campo, repetir.campo], async () => {
        try {
          const r = await api.post('/contrasena', { actual: actual.input.value, contrasena: nueva.input.value });
          if (r && r.csrf) api.csrf = r.csrf;
          actual.input.value = nueva.input.value = repetir.input.value = '';
          aviso(S.cambiada, 'ok');
        } catch (e) {
          if (e.codigo === 'CONTRASENA_INCORRECTA') { actual.campo.ponerError(e.message); actual.input.focus(); return; }
          throw e;
        }
      }, S.cambiar);
      // El acceso se guarda al cambiar el interruptor; quitarlo pide confirmación.
      const soloRed = UI.interruptor({ id: 'aj-privada', texto: S.soloRed, ayuda: S.soloRedAyuda, marcado: a.solo_red_privada });
      soloRed.input.addEventListener('change', async () => {
        const valor = soloRed.input.checked;
        if (!valor && !(await confirmar(S.quitarTitulo, S.quitarTexto, { boton: S.quitarBoton, peligro: true, consecuencias: S.quitarConsecuencias }))) {
          soloRed.input.checked = true;
          return;
        }
        try { await api.put('/ajustes', { solo_red_privada: valor }); a.solo_red_privada = valor; guardado(); } catch (e) { soloRed.input.checked = !valor; avisoError(e); }
      });
      return h('div', { class: 'rejilla' }, cambio,
        UI.tarjeta({ titulo: S.acceso, icono: 'escudo' }, soloRed, UI.nota({ tipo: 'info', titulo: S.olvido, texto: S.olvidoTexto })));
    }

    function https() {
      const T = TAj.https;
      const cert = a.tls.certificado;
      // Desde 0.3.1, una autoridad local firma el certificado del servidor (ADR 0021): regenerarlo
      // con la misma autoridad no obliga a reinstalar nada en los equipos.
      const autoridad = a.tls.autoridad;
      const regenerar = a.tls.modo === 'autofirmado' && UI.boton({ texto: T.regenerar, variante: 'peligro', icono: 'recargar' });
      if (regenerar) {
        regenerar.addEventListener('click', async () => {
          if (!(await confirmar(T.regenerarTitulo, T.regenerarTexto, { boton: T.regenerar, peligro: !autoridad,
            consecuencias: autoridad ? T.regenerarConsecuenciasAutoridad : T.regenerarConsecuencias }))) return;
          try {
            const r = await conEspera(regenerar, () => api.post('/tls/regenerar'));
            // Solo hay algo que hacer si la autoridad es nueva: entonces el aviso se queda.
            aviso(r.autoridad_nueva ? T.regeneradoAutoridadNueva : T.regenerado, 'ok', { persistente: r.autoridad_nueva });
            ir('ajustes', '?pestana=https');
          } catch (e) { avisoError(e); }
        });
      }
      return UI.tarjeta({ titulo: T.titulo, icono: 'candado' },
        h('dl', { class: 'datos' },
          h('dt', {}, T.modo), h('dd', {}, T.modos[a.tls.modo] || a.tls.modo),
          h('dt', {}, T.direccion), h('dd', { class: 'mono' }, a.direccion),
          cert && [
            h('dt', {}, T.vence), h('dd', { class: 'fila' }, cert.vence,
              cert.estado === 'vencido' ? UI.insignia('error', T.vencido) : cert.estado === 'por_vencer' ? UI.insignia('aviso', T.venceEn(cert.dias_restantes)) : null),
            h('dt', {}, T.nombres), h('dd', { class: 'fila' }, cert.nombres.map((n) => h('span', { class: 'chip' }, n))),
            h('dt', {}, T.ips), h('dd', { class: 'fila' }, cert.ips.map((n) => h('span', { class: 'chip' }, n))),
            autoridad
              ? [h('dt', {}, T.autoridad), h('dd', {}, T.autoridadSi(autoridad.vence)),
                h('dt', {}, T.huellaAutoridad), h('dd', { class: 'mono pequeno' }, autoridad.huella)]
              : [h('dt', {}, T.huella), h('dd', { class: 'mono pequeno' }, cert.huella),
                cert.limitado !== undefined && [h('dt', {}, T.autoridad), h('dd', {}, T.autoridadNo)]]]),
        cert && cert.limitado === false && UI.nota({ tipo: 'aviso', titulo: T.sinLimites, texto: T.sinLimitesTexto }),
        cert && cert.ips_sin_cubrir && cert.ips_sin_cubrir.length > 0 && UI.nota({ tipo: 'aviso', texto: T.ipsFaltan(cert.ips_sin_cubrir.join(', ')) }),
        a.tls.modo === 'archivos' && UI.nota({ tipo: 'info', texto: T.archivos }),
        h('div', { class: 'acciones' }, cert && enlaceDescarga('/tls/certificado', T.descargar), regenerar),
        cert && h('p', { class: 'tenue pequeno' }, T.instalar, h('a', { href: rutaDe('ayuda') }, T.instalarEnlace), '.'));
    }

    function respaldo() {
      const R = TAj.respaldo;
      const archivo = h('input', { type: 'file', accept: '.db,application/octet-stream' });
      const cArchivo = UI.campo({ id: 'aj-restaurar', etiqueta: R.archivo, control: archivo, validar: () => (archivo.files[0] ? '' : R.archivoFalta) });
      const confirmacion = UI.entrada({ mono: true, autocomplete: 'off' });
      const cConfirmar = UI.campo({ id: 'aj-confirmar', etiqueta: R.confirmar, control: confirmacion, validar: (v) => (v === 'RESTAURAR' ? '' : R.confirmarMal) });
      const restaurar = UI.boton({ texto: R.boton, variante: 'peligro-lleno', tipo: 'submit' });
      const form = h('form', { class: 'pila-amplia', novalidate: true, onsubmit: async (ev) => {
        ev.preventDefault();
        if (!UI.revisarCampos([cArchivo, cConfirmar])) return;
        const fd = new FormData();
        fd.append('archivo', archivo.files[0]);
        try {
          await conEspera(restaurar, () => api.pedir('POST', '/restaurar', fd));
          aviso(R.hecho, 'ok');
          estado.sesion = null;
          setTimeout(arrancar, 1500);
        } catch (e) { avisoError(e); }
      } }, cArchivo, cConfirmar, h('div', {}, restaurar));
      return h('div', { class: 'rejilla' },
        UI.tarjeta({ titulo: R.titulo, icono: 'descargar' }, h('p', { class: 'tenue pequeno' }, R.texto), h('div', {}, enlaceDescarga('/respaldo', R.descargar))),
        UI.tarjeta({ titulo: R.restaurar, icono: 'aviso' }, UI.nota({ tipo: 'aviso', titulo: R.cuidado, texto: R.antes }), form));
    }

    const P = TAj.pestanas;
    agregar(vista, encabezado(TAj.titulo, TAj.subtitulo),
      UI.pestanas([
        { id: 'general', titulo: P.general, contenido: general },
        { id: 'seguridad', titulo: P.seguridad, contenido: seguridad },
        { id: 'https', titulo: P.https, contenido: https },
        { id: 'respaldo', titulo: P.respaldo, contenido: respaldo },
      ], { etiqueta: TAj.titulo, inicial: pestana || 'general',
        alCambiar: (id) => history.replaceState(null, '', `${rutaDe('ajustes')}?pestana=${id}`) }));
  }

  // ---------------------------------------------------------------------------------------------
  // Aviso de sesión: la sesión se cierra tras 30 minutos sin actividad. 2 minutos antes se avisa;
  // solo «Seguir conectado» (que pulsa la persona) la renueva: el panel no la mantiene viva solo.
  // ---------------------------------------------------------------------------------------------
  const TSe = window.PRISMA_TEXTOS.sesion;
  const INACTIVIDAD = 30 * 60 * 1000;
  const AVISAR_ANTES = 2 * 60 * 1000;
  let avisoSesion = null;
  function cerrarAvisoSesion() {
    if (avisoSesion) { avisoSesion.dialogo.close(); avisoSesion.dialogo.remove(); avisoSesion = null; }
  }
  function mostrarAvisoSesion(resta) {
    const texto = h('p', {}, TSe.texto(Math.ceil(resta / 1000)));
    const seguir = UI.boton({ texto: TSe.seguir, variante: 'primario' });
    const salir = UI.boton({ texto: TSe.salir });
    const d = h('dialog', { role: 'alertdialog', 'aria-labelledby': 'sesion-titulo', 'aria-describedby': 'sesion-texto' },
      h('div', { class: 'dialogo' }, h('h2', { id: 'sesion-titulo' }, TSe.titulo), h('div', { id: 'sesion-texto' }, texto),
        h('div', { class: 'dialogo-pie' }, salir, seguir)));
    const renovar = async () => {
      cerrarAvisoSesion();
      try { if (!(await api.get('/sesion')).autenticado) { estado.sesion = null; arrancar(); } } catch { /* se avisará otra vez */ }
    };
    seguir.addEventListener('click', renovar);
    d.addEventListener('cancel', (ev) => { ev.preventDefault(); renovar(); });
    salir.addEventListener('click', async () => {
      cerrarAvisoSesion();
      try { await api.post('/salir'); } catch { /* la sesión ya no existe */ }
      estado.sesion = null; arrancar();
    });
    document.body.append(d);
    d.showModal();
    seguir.focus();
    avisoSesion = { dialogo: d, texto };
  }
  setInterval(() => {
    if (!enElPanel()) { cerrarAvisoSesion(); return; }
    const resta = INACTIVIDAD - (Date.now() - api.ultimaActividad);
    if (resta <= 0) {
      // El puente ya la cerró por inactividad; se pide de todos modos, por si el reloj del navegador va adelantado.
      cerrarAvisoSesion();
      estado.sesion = null;
      aviso(TSe.cerrada, 'aviso', { persistente: true });
      api.post('/salir').catch(() => {}).finally(arrancar);
    } else if (resta <= AVISAR_ANTES) {
      if (!avisoSesion) mostrarAvisoSesion(resta);
      else avisoSesion.texto.textContent = TSe.texto(Math.ceil(resta / 1000));
    }
  }, 1000);

  // ---------------------------------------------------------------------------------------------
  // 8. Ayuda
  // ---------------------------------------------------------------------------------------------
  // La documentación de la versión instalada: la etiqueta v<versión> si es una versión publicada, o
  // main en una compilación de desarrollo (0.0.0-dev) o intermedia (git describe con -N-gabc).

  // ---------------------------------------------------------------------------------------------
  // 9. Ayuda: el manual de docs/ (docs.Manual), dibujado por manual.js sin innerHTML.
  // ---------------------------------------------------------------------------------------------
  const TAy = window.PRISMA_TEXTOS.ayuda;
  let manual = null; // { grupos, documentos: Map(ruta → { titulo, grupo }), orden: [ruta] }
  async function cargarManual() {
    if (manual) return manual;
    const { grupos } = await api.get('/manual');
    const documentos = new Map();
    const orden = [];
    for (const g of grupos) for (const d of g.documentos) { documentos.set(d.ruta, { ...d, grupo: g }); orden.push(d.ruta); }
    manual = { grupos, documentos, orden };
    return manual;
  }
  const rutaAyuda = (ruta, ancla = '') => `/admin/ayuda/${ruta}${ancla}`;
  // Resuelve un destino relativo al documento actual: «../referencia/cli.md#x» desde
  // «administracion/README.md» es «referencia/cli.md» con el ancla «#x». Fuera de docs/ empieza con «../».
  function resolverRuta(destino, desde) {
    const u = new URL(destino, `https://x/docs/${desde}`);
    const ruta = u.pathname.startsWith('/docs/') ? u.pathname.slice(6) : `../${u.pathname.slice(1)}`;
    return { ruta: decodeURIComponent(ruta), ancla: u.hash };
  }

  const textosManual = new Map();
  async function leerDocumento(ruta) {
    if (!textosManual.has(ruta)) textosManual.set(ruta, await (await api.pedir('GET', '/admin/manual/' + ruta.split('/').map(enc).join('/'))).text());
    return textosManual.get(ruta);
  }
  // Diagramas dibujados de antemano (ADR 0018): el manifiesto, por id y por documento de origen.
  let diagramas = null;
  async function cargarDiagramas() {
    if (diagramas) return diagramas;
    const { diagramas: lista } = await api.get('/diagramas');
    diagramas = { lista, porId: new Map(lista.map((d) => [d.id, d])), porFuente: new Map(lista.map((d) => [d.fuente, d])) };
    return diagramas;
  }
  // El visor de un diagrama, con sus dos variantes (cambian con el tema del panel).
  function visorDe(d, { mermaid, explicacion } = {}) {
    const url = (v, descarga) => `/admin/diagramas/${d.id}.${v}.svg${descarga ? '?descargar=1' : ''}`;
    return UI.visorDiagrama({ src: { claro: url('claro'), oscuro: url('oscuro') }, descargar: { claro: url('claro', true), oscuro: url('oscuro', true) },
      alt: d.alt, titulo: d.titulo, descripcion: d.descripcion, publico: d.audiencia, mermaid, explicacion, enlaces: enlacesDeDiagrama(d) });
  }
  // Los nodos con enlace (front-matter «enlaces»): un documento del manual o una pantalla del panel.
  function enlacesDeDiagrama(d) {
    return (d.enlaces || []).map((e) => {
      if (e.destino.startsWith('/admin/')) {
        const seccion = e.destino.slice('/admin/'.length).split('/')[0] || 'inicio';
        return { caja: e.caja, href: e.destino, etiqueta: window.PRISMA_TEXTOS.ui.visor.enlacePantalla(e.texto, TN.secciones[seccion] || seccion) };
      }
      const [ruta, ancla] = e.destino.split('#');
      return { caja: e.caja, href: rutaAyuda(`${ruta}.md`, ancla ? `#${ancla}` : ''), etiqueta: window.PRISMA_TEXTOS.ui.visor.enlaceManual(e.texto) };
    });
  }

  // El repositorio en GitHub, en la versión instalada (o main si es de desarrollo).
  const repoGithub = () => `https://github.com/bayalad7/prisma/blob/${/^\d+\.\d+\.\d+(-rc\.\d+)?$/.test(estado.version || '') ? `v${estado.version}` : 'main'}/`;
  // Un enlace que sale del panel: otra pestaña, con su aviso para el lector de pantalla.
  const externo = (href, hijos, nota = TAy.externo) => h('a', { href, target: '_blank', rel: 'noopener noreferrer', class: 'enlace-externo' },
    hijos, UI.icono('externo'), h('span', { class: 'solo-lector' }, ` ${nota}`));

  // Adónde lleva cada enlace de un documento del manual (para manual.js).
  function contextoManual(ruta, documentos, dibujos) {
    const ref = repoGithub();
    return {
      textos: TAy,
      enlace(destino, hijos) {
        if (/^(https?:|mailto:)/.test(destino)) return externo(destino, hijos);
        if (destino.startsWith('#')) return h('a', { href: destino, 'data-ancla': '1' }, hijos);
        let { ruta: r, ancla } = resolverRuta(destino, ruta);
        if (r.endsWith('/') || r === '') r += 'README.md';
        if (documentos.has(r)) return h('a', { href: rutaAyuda(r, ancla) }, hijos);
        // Fuera del manual (código, ADR, desarrollo): el repositorio en GitHub, con Internet.
        const enRepo = r.startsWith('../') ? r.slice(3) : `docs/${r}`;
        return externo(`${ref}${enRepo}${ancla}`, hijos, TAy.requiereInternet);
      },
      imagen(destino, alt) {
        if (/^https?:/.test(destino)) return h('span', { class: 'tenue' }, TAy.imagenExterna(alt));
        const { ruta: r } = resolverRuta(destino, ruta);
        return h('img', { src: `/admin/manual/${r}`, alt, loading: 'lazy', class: 'imagen-manual' });
      },
      codigo(lenguaje, texto, pre) {
        if (lenguaje !== 'mermaid') return pre;
        // En la página de un diagrama, su bloque se ve dibujado; otros bloques mermaid, como código.
        const d = dibujos && dibujos.porFuente.get(ruta);
        if (d) return visorDe(d, { mermaid: texto });
        return h('details', { class: 'avanzado' }, h('summary', {}, TAy.verDiagrama), pre);
      },
      // {{diagrama:id}}: el visor de ese diagrama, con un enlace a su explicación.
      diagrama(id) {
        const d = dibujos && dibujos.porId.get(id);
        if (!d) return UI.nota({ tipo: 'aviso', titulo: TAy.diagramaNoExiste(id) });
        return visorDe(d, { explicacion: h('a', { href: rutaAyuda(d.fuente) }, TAy.verExplicacion) });
      },
    };
  }
  // En la página de un diagrama, «Fuentes en el código» (para desarrolladores) va plegado.
  function plegarFuentes(nodos) {
    const i = nodos.findIndex((n) => n.tagName === 'P' && n.textContent.startsWith('Fuentes en el código'));
    if (i < 0 || !nodos[i + 1] || nodos[i + 1].tagName !== 'UL') return nodos;
    const plegado = h('details', { class: 'avanzado' }, h('summary', {}, TAy.fuentes), nodos[i + 1]);
    return [...nodos.slice(0, i), plegado, ...nodos.slice(i + 2)];
  }

  // Panel lateral de ayuda: una sección del manual («ruta#ancla») sin salir de la pantalla, así no se
  // pierde lo que se estaba haciendo. Los enlaces al panel lo cierran y llevan a su pantalla.
  async function abrirAyuda(destino, { pantalla } = {}) {
    const [ruta, ancla = ''] = destino.split('#');
    const { documentos } = await cargarManual();
    document.querySelector('dialog.panel-ayuda[open]')?.close();
    const doc = documentos.get(ruta);
    if (!doc) return;
    const dibujos = await cargarDiagramas().catch(() => null);
    const { nodos } = window.PrismaManual.renderizar(await leerDocumento(ruta), contextoManual(ruta, documentos, dibujos));
    let titulo = doc.titulo;
    let cuerpo = nodos.slice(1);
    const i = ancla ? nodos.findIndex((n) => n.id === ancla) : -1;
    if (i >= 0) {
      const nivel = Number(nodos[i].tagName[1]);
      let j = i + 1;
      while (j < nodos.length && !(/^H[1-6]$/.test(nodos[j].tagName) && Number(nodos[j].tagName[1]) <= nivel)) j++;
      titulo = nodos[i].textContent;
      cuerpo = nodos.slice(i + 1, j);
    }
    const idTitulo = UI.nuevoId('ayuda');
    const d = h('dialog', { class: 'panel-ayuda', 'aria-labelledby': idTitulo });
    d.addEventListener('close', () => d.remove());
    d.addEventListener('click', (ev) => {
      if (ev.target === d) { d.close(); return; } // clic fuera del panel
      const a = ev.target.closest('a');
      if (!a) return;
      if (a.dataset.ancla) {
        ev.preventDefault();
        d.querySelector(`[id="${CSS.escape(decodeURIComponent(a.getAttribute('href').slice(1)))}"]`)?.scrollIntoView({ block: 'start' });
      } else if (a.getAttribute('href').startsWith('/admin/')) d.close();
    });
    agregar(d,
      h('div', { class: 'panel-ayuda-cabeza' },
        h('div', {}, i >= 0 && h('p', { class: 'tenue pequeno' }, doc.titulo), h('h2', { id: idTitulo }, titulo)),
        UI.boton({ icono: 'cerrar', variante: 'texto', etiqueta: TAy.panelCerrar, onclick: () => d.close() })),
      h('div', { class: 'documento-manual' }, cuerpo, relacionadosDe(pantalla, ruta)),
      h('p', { class: 'pequeno' }, h('a', { href: rutaAyuda(ruta, ancla ? `#${ancla}` : '') }, TAy.abrirEnAyuda, UI.icono('derecha'))));
    document.body.append(d);
    d.showModal();
  }
  // La pantalla actual, como clave de ayuda.pantallas (las subpantallas de Probar impresión tienen la suya).
  function clavePantalla() {
    const r = location.pathname.replace(/^\/admin\/?/, '');
    if (r.startsWith('pruebas/rollo/')) return 'rollo';
    if (r === 'pruebas/permisos') return 'permisos';
    return seccionActual();
  }
  const ayudaDePantalla = () => TAy.pantallas[clavePantalla()];
  // «Diagramas de esta pantalla»: los que la nombran en su front-matter, primero los de dueños.
  function relacionadosDe(pantalla, rutaActual) {
    if (!pantalla || !diagramas) return null;
    const lista = diagramas.lista.filter((d) => d.pantallas.includes(pantalla) && d.fuente !== rutaActual).sort(ordenDiagramas);
    if (!lista.length) return null;
    return h('section', { class: 'relacionados' }, h('h3', {}, TAy.relacionados),
      h('ul', { class: 'lista-manual' }, lista.map((d) => {
        const b = h('button', { type: 'button', class: 'enlace' }, d.titulo);
        b.addEventListener('click', () => abrirAyuda(d.fuente, { pantalla }).catch(avisoError));
        return h('li', {}, b, h('span', { class: 'tenue pequeno' }, ` · ${d.descripcion}`));
      })));
  }
  // Un enlace «Leer en el manual» para un código de error o de aviso, si el manual lo explica.
  function enlaceManual(codigo) {
    const destino = TAy.errores[codigo];
    if (!destino) return null;
    const b = UI.boton({ texto: TAy.leerManual, chico: true, variante: 'texto', icono: 'libro' });
    b.addEventListener('click', () => abrirAyuda(destino).catch(avisoError));
    return b;
  }

  // «Mi impresora no imprime»: preguntas paso a paso armadas desde el diagrama 23 (la única fuente),
  // hasta la solución del manual. manual.js (flujo) lee el diagrama; TestArbolDeDiagnostico exige que
  // el árbol esté completo.
  const DIAGRAMA_NO_IMPRIME = 'diagramas/23-diagnostico-problemas.md';
  // Primera letra en mayúscula, también después de «¿» (los diagramas van en minúsculas).
  const mayuscula = (t) => t.replace(/^([¿¡]?)(\p{L})/u, (_, signo, letra) => signo + letra.toUpperCase());
  async function vistaNoImprime(vista) {
    const T = TAy.noImprime;
    const f = window.PrismaManual.flujo(await leerDocumento(DIAGRAMA_NO_IMPRIME));
    const destino = (id) => {
      const e = f.enlaces[id];
      if (!e) return null;
      const { ruta, ancla } = resolverRuta(e.destino, DIAGRAMA_NO_IMPRIME);
      return { texto: e.texto, destino: `${ruta}${decodeURIComponent(ancla)}` };
    };
    const botonManual = (enlace, texto) => {
      const b = UI.boton({ texto, icono: 'libro', chico: true });
      b.addEventListener('click', () => abrirAyuda(enlace.destino).catch(avisoError));
      return b;
    };
    const recorrido = []; // [{ pregunta, respuesta }] ya contestadas
    let actual = f.respuestas[f.inicio][0].a;
    const zona = h('div', { class: 'pila-amplia' });
    function pintar(enfocar) {
      const historial = recorrido.length > 0 && h('ol', { class: 'recorrido-arbol' }, recorrido.map((paso, k) => {
        const cambiar = UI.boton({ texto: T.cambiar, variante: 'texto', chico: true, 'aria-label': T.cambiarEtiqueta(f.nodos[paso.pregunta].texto) });
        cambiar.addEventListener('click', () => { actual = paso.pregunta; recorrido.splice(k); pintar(true); });
        return h('li', {}, h('span', {}, mayuscula(f.nodos[paso.pregunta].texto)), h('strong', {}, ` ${mayuscula(paso.respuesta.texto)}`), cambiar);
      }));
      const nodo = f.nodos[actual];
      const idTitulo = UI.nuevoId('arbol');
      let paso;
      if (nodo.pregunta) {
        const ayuda = destino(actual);
        paso = h('section', { class: 'tarjeta pregunta-arbol', 'aria-labelledby': idTitulo },
          h('h2', { id: idTitulo, tabindex: '-1' }, mayuscula(nodo.texto)),
          h('div', { class: 'acciones' }, f.respuestas[actual].map((r) => {
            const b = UI.boton({ texto: mayuscula(r.texto), variante: 'secundario' });
            b.addEventListener('click', () => { recorrido.push({ pregunta: actual, respuesta: r }); actual = r.a; pintar(true); });
            return b;
          })),
          ayuda && h('p', { class: 'pequeno' }, T.comoSaber, ' ', botonManual(ayuda, ayuda.texto)));
      } else {
        const solucion = destino(actual);
        const reiniciar = UI.boton({ texto: T.reiniciar, variante: 'texto', icono: 'recargar' });
        reiniciar.addEventListener('click', () => { recorrido.length = 0; actual = f.respuestas[f.inicio][0].a; pintar(true); });
        paso = h('section', { class: 'tarjeta', 'aria-labelledby': idTitulo },
          h('h2', { id: idTitulo, tabindex: '-1' }, T.probable),
          UI.nota({ tipo: 'info', titulo: mayuscula(nodo.texto) }),
          h('div', { class: 'acciones' }, solucion && botonManual(solucion, T.leerSolucion(solucion.texto)), reiniciar));
      }
      vaciar(zona, historial, paso);
      if (enfocar) zona.querySelector('h2').focus();
    }
    agregar(vista, h('a', { href: rutaDe('ayuda'), class: 'volver' }, UI.icono('flecha-izq'), TAy.volver),
      encabezado(T.titulo, T.subtitulo), zona,
      h('p', { class: 'pequeno tenue' }, T.fuente, ' ', h('a', { href: rutaAyuda(DIAGRAMA_NO_IMPRIME) }, T.verDiagrama)));
    pintar(false);
  }

  // Galería «Cómo funciona PRISMA» (/admin/ayuda/diagramas): los diagramas por público, con búsqueda.
  // «Para el negocio» (dueños e instaladores) va primero, para no abrumar con los técnicos.
  const PUBLICOS = { negocio: ['usuario', 'instalador'], tecnicos: ['integrador', 'desarrollador'] };
  // Orden de los diagramas: primero los de dueños; luego, el orden curado del manual (docs/manual.json).
  const ordenDiagramas = (a, b) => {
    const dueno = Number(b.audiencia.includes('usuario')) - Number(a.audiencia.includes('usuario'));
    const lugar = (d) => (manual ? manual.orden.indexOf(d.fuente) : 0);
    return dueno || lugar(a) - lugar(b);
  };
  const plegarTexto = (t) => t.toLowerCase().normalize('NFD').replace(/[̀-ͯ]/g, '');
  async function vistaGaleria(vista) {
    const T = TAy.galeria;
    const [{ lista }] = await Promise.all([cargarDiagramas(), cargarManual()]);
    const busqueda = UI.entrada({ tipo: 'search', autocomplete: 'off' });
    const cBusqueda = UI.campo({ id: 'ga-buscar', etiqueta: T.buscar, control: busqueda });
    const cuenta = h('p', { class: 'tenue pequeno', 'aria-live': 'polite' });
    const rejilla = h('div', { class: 'galeria-diagramas' });
    let publico = 'negocio';
    const tarjeta = (d) => {
      const url = (v) => `/admin/diagramas/${d.id}.${v}.svg`;
      return h('a', { class: 'tarjeta tarjeta-diagrama', href: rutaAyuda(d.fuente) },
        h('div', { class: 'miniatura' }, h('img', { class: 'solo-claro', src: url('claro'), alt: '', loading: 'lazy' }),
          h('img', { class: 'solo-oscuro', src: url('oscuro'), alt: '', loading: 'lazy' })),
        h('strong', {}, d.titulo), h('span', { class: 'tenue pequeno' }, d.descripcion),
        h('span', { class: 'fila' }, d.audiencia.map((a) => h('span', { class: 'chip' }, window.PRISMA_TEXTOS.ui.visor.audiencias[a] || a))));
    };
    const pintar = () => {
      const q = plegarTexto(busqueda.value.trim());
      const elegidos = lista
        .filter((d) => d.audiencia.some((a) => PUBLICOS[publico].includes(a)))
        .filter((d) => !q || plegarTexto(`${d.titulo} ${d.descripcion}`).includes(q))
        .sort(ordenDiagramas);
      cuenta.textContent = T.cuenta(elegidos.length);
      vaciar(rejilla, elegidos.length ? elegidos.map(tarjeta) : UI.vacio({ icono: 'diagrama', titulo: T.vacio, texto: T.vacioTexto }));
    };
    busqueda.addEventListener('input', pintar);
    const pestanas = UI.pestanas([
      { id: 'negocio', titulo: T.negocio, contenido: () => h('div') },
      { id: 'tecnicos', titulo: T.tecnicos, contenido: () => h('div') },
    ], { etiqueta: T.publico, alCambiar: (id) => { publico = id; pintar(); } });
    agregar(vista, h('a', { href: rutaDe('ayuda'), class: 'volver' }, UI.icono('flecha-izq'), TAy.volver),
      encabezado(T.titulo, T.subtitulo), cBusqueda, pestanas, cuenta, rejilla);
    pintar();
  }

  // Un botón que abre un diagrama en el panel lateral de ayuda.
  function botonDiagrama(id, { chico = true } = {}) {
    const d = diagramas && diagramas.porId.get(id);
    if (!d) return null;
    const b = UI.boton({ texto: TAy.verDiagramaDe(d.titulo), icono: 'diagrama', chico, variante: 'texto' });
    b.addEventListener('click', () => abrirAyuda(d.fuente).catch(avisoError));
    return b;
  }

  async function vistaAyuda(vista) {
    const { grupos, documentos, orden } = await cargarManual();
    const ruta = decodeURIComponent(location.pathname.replace(/^\/admin\/ayuda\/?/, ''));
    if (ruta === 'no-imprime') return vistaNoImprime(vista);
    if (ruta === 'diagramas') return vistaGaleria(vista);
    if (!ruta) {
      const guiar = h('a', { class: 'boton primario', href: rutaAyuda('no-imprime') }, UI.icono('impresora'), TAy.noImprime.empezar);
      const galeria = h('a', { class: 'boton', href: rutaAyuda('diagramas') }, UI.icono('diagrama'), TAy.galeria.abrir);
      agregar(vista, encabezado(TAy.titulo, TAy.subtitulo),
        UI.nota({ tipo: 'info', titulo: TAy.noImprime.tarjeta, texto: TAy.noImprime.tarjetaTexto, acciones: guiar }),
        UI.nota({ tipo: 'info', titulo: TAy.galeria.titulo, texto: TAy.galeria.tarjetaTexto, acciones: galeria }),
        buscadorManual(),
        h('div', { class: 'grupos-manual' }, grupos.map((g) => UI.tarjeta({ titulo: g.titulo, icono: 'libro', nivel: 2 },
          h('ul', { class: 'lista-manual' }, g.documentos.map((d) => h('li', {}, h('a', { href: rutaAyuda(d.ruta) }, d.titulo))))))));
      return;
    }
    const doc = documentos.get(ruta);
    const indice = h('nav', { class: 'indice-manual', 'aria-label': TAy.indice },
      h('details', { class: 'avanzado', open: matchMedia('(min-width: 1024px)').matches },
        h('summary', {}, TAy.indice),
        buscadorManual(true),
        grupos.map((g) => h('div', { class: 'pila' }, h('p', { class: 'grupo-manual' }, g.titulo),
          h('ul', { class: 'lista-manual' }, g.documentos.map((d) => h('li', {},
            h('a', { href: rutaAyuda(d.ruta), 'aria-current': d.ruta === ruta ? 'page' : undefined }, d.titulo))))))));
    const volver = h('a', { href: rutaDe('ayuda'), class: 'volver' }, UI.icono('flecha-izq'), TAy.volver);
    if (!doc) {
      agregar(vista, h('div', { class: 'manual' }, indice, h('article', { class: 'documento-manual' }, volver,
        UI.vacio({ icono: 'libro', titulo: TAy.noExiste, texto: TAy.noExisteTexto }))));
      return;
    }
    const texto = await leerDocumento(ruta);
    const contexto = contextoManual(ruta, documentos, await cargarDiagramas().catch(() => null));
    const { nodos: dibujados, titulos } = window.PrismaManual.renderizar(texto, contexto);
    const nodos = ruta.startsWith('diagramas/') ? plegarFuentes(dibujados) : dibujados;
    const secciones = titulos.filter((t) => t.nivel === 2);
    const enEsta = secciones.length >= 3 && h('details', { class: 'avanzado en-esta-pagina' }, h('summary', {}, TAy.enEstaPagina),
      h('ul', { class: 'lista-manual' }, secciones.map((t) => h('li', {}, h('a', { href: `#${t.id}`, 'data-ancla': '1' }, t.texto)))));
    const k = orden.indexOf(ruta);
    const vecino = (j, etiqueta, icono) => orden[j] && h('a', { href: rutaAyuda(orden[j]), class: `vecino-manual vecino-${icono}` },
      h('span', { class: 'tenue pequeno' }, etiqueta), h('span', {}, documentos.get(orden[j]).titulo));
    const articulo = h('article', { class: 'documento-manual' },
      h('div', { class: 'fila-entre' }, volver, h('span', { class: 'tenue pequeno' }, doc.grupo.titulo)),
      nodos.slice(0, 1), enEsta, nodos.slice(1),
      h('nav', { class: 'vecinos-manual', 'aria-label': TAy.indice }, vecino(k - 1, TAy.anterior, 'izq'), vecino(k + 1, TAy.siguiente, 'der')),
      h('p', { class: 'pequeno' }, externo(`${repoGithub()}docs/${ruta}`, TAy.enGithub, TAy.requiereInternet)));
    // Anclas dentro de la página: desplazan sin volver a dibujar.
    articulo.addEventListener('click', (ev) => {
      const a = ev.target.closest('a[data-ancla]');
      if (!a) return;
      const destino = document.getElementById(decodeURIComponent(a.getAttribute('href').slice(1)));
      if (!destino) return;
      ev.preventDefault();
      history.replaceState(null, '', a.getAttribute('href'));
      enfocarAncla(destino);
    });
    document.title = TN.tituloPagina(doc.titulo, nombreVisible(estado.nombrePuente));
    agregar(vista, h('div', { class: 'manual' }, indice, articulo));
  }
  // Buscador del manual. En la portada de la Ayuda muestra los resultados al escribir (y deja la
  // búsqueda en la dirección, ?q=…); en una página, lleva a la portada con la búsqueda.
  function buscadorManual(compacto = false) {
    const id = compacto ? 'ay-buscar-indice' : 'ay-buscar';
    const entrada = UI.entrada({ tipo: 'search', autocomplete: 'off', value: compacto ? '' : new URLSearchParams(location.search).get('q') || '' });
    const resultados = h('div', { class: 'pila', 'aria-live': 'polite' });
    const campo = UI.campo({ id, etiqueta: TAy.buscar, control: entrada, ayuda: compacto ? undefined : TAy.buscarAyuda });
    if (compacto) {
      return h('form', { class: 'buscar-manual', role: 'search', onsubmit: (ev) => {
        ev.preventDefault();
        if (entrada.value.trim()) ir('ayuda', `?q=${enc(entrada.value.trim())}`);
      } }, campo);
    }
    let turno = 0;
    let espera;
    const buscar = async () => {
      const q = entrada.value.trim();
      const mio = ++turno;
      history.replaceState(null, '', rutaDe('ayuda') + (q ? `?q=${enc(q)}` : ''));
      if (q.length < 2) { vaciar(resultados); return; }
      let lista;
      try { lista = await api.get(`/manual/buscar?q=${enc(q)}`); } catch (e) { if (mio === turno) vaciar(resultados, UI.nota({ tipo: 'error', titulo: e.message })); return; }
      if (mio !== turno) return;
      const palabras = q.toLowerCase().normalize('NFD').replace(/[̀-ͯ]/g, '').split(/[^\p{L}\p{N}_]+/u).filter((p) => p.length >= 2);
      // Resalta las palabras buscadas en el extracto, sin acentos al comparar.
      const resaltar = (texto) => {
        const plano = texto.toLowerCase().normalize('NFD').replace(/[̀-ͯ]/g, '');
        if (plano.length !== texto.length || !palabras.length) return texto;
        const marcas = [];
        for (const p of palabras) { let i = plano.indexOf(p); while (i >= 0) { marcas.push([i, i + p.length]); i = plano.indexOf(p, i + p.length); } }
        marcas.sort((a, b) => a[0] - b[0]);
        const nodos = [];
        let desde = 0;
        for (const [a, b] of marcas) { if (a < desde) continue; nodos.push(texto.slice(desde, a), h('mark', {}, texto.slice(a, b))); desde = b; }
        nodos.push(texto.slice(desde));
        return nodos;
      };
      vaciar(resultados, lista.length === 0 ? h('p', { class: 'tenue' }, TAy.sinResultados(q)) : [
        h('p', { class: 'tenue pequeno' }, TAy.resultados(lista.length)),
        h('ol', { class: 'resultados-manual' }, lista.map((r) => h('li', {},
          h('a', { href: rutaAyuda(r.ruta, r.ancla ? `#${r.ancla}` : '') }, r.seccion || r.titulo),
          r.seccion && h('span', { class: 'tenue pequeno' }, ` · ${r.titulo}`),
          h('p', { class: 'pequeno' }, resaltar(r.extracto))))),
      ]);
    };
    entrada.addEventListener('input', () => { clearTimeout(espera); espera = setTimeout(buscar, 250); });
    if (entrada.value) setTimeout(buscar);
    return h('section', { class: 'tarjeta buscar-manual', role: 'search', 'aria-label': TAy.buscar }, campo, resultados);
  }
  // Lleva la vista a un título del manual y le da el foco (lo anuncia el lector de pantalla).
  function enfocarAncla(el) {
    el.tabIndex = -1;
    el.focus({ preventScroll: true });
    el.scrollIntoView({ block: 'start' });
  }

  const VISTAS = {
    inicio: vistaInicio, impresoras: vistaImpresoras, aplicaciones: vistaClientes, solicitudes: vistaSolicitudes,
    trabajos: vistaTrabajos, pruebas: vistaPruebas, imagenes: vistaRecursos, ajustes: vistaAjustes, ayuda: vistaAyuda,
  };

  arrancar();
})();
