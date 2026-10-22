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

  const ESTADOS_IMPRESORA = {
    en_linea: ['ok', 'En línea'], fuera_de_linea: ['error', 'Fuera de línea'],
    ocupada: ['aviso', 'Imprimiendo'], desconocido: ['', 'Sin probar'],
  };
  function pastilla(mapa, clave) {
    const [clase, texto] = mapa[clave] || ['', clave];
    return h('span', { class: `pastilla ${clase}` }, texto);
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
    }
  }
  const api = {
    csrf: '',
    async pedir(metodo, ruta, cuerpo) {
      if (window.PRISMA_DEMO) return window.PRISMA_DEMO.pedir(metodo, ruta, cuerpo);
      const op = { method: metodo, credentials: 'same-origin', headers: { Accept: 'application/json' } };
      if (metodo !== 'GET') op.headers['X-CSRF-Token'] = api.csrf;
      if (cuerpo instanceof FormData) op.body = cuerpo;
      else if (cuerpo !== undefined) { op.headers['Content-Type'] = 'application/json'; op.body = JSON.stringify(cuerpo); }
      let r;
      try { r = await fetch('/admin/api' + ruta, op); } catch {
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
    get: (ruta) => api.pedir('GET', ruta),
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
  const TN = window.PRISMA_TEXTOS.nav;
  // Secciones del panel: cada una tiene su ruta bajo /admin/ (el puente sirve el panel en todas), su
  // grupo en la navegación y la tecla de su atajo (g y la letra).
  const SECCIONES = [
    { id: 'inicio', ruta: '', icono: 'inicio', grupo: 'operacion', tecla: 'i' },
    { id: 'impresoras', ruta: 'impresoras', icono: 'impresora', grupo: 'operacion', tecla: 'p' },
    { id: 'solicitudes', ruta: 'solicitudes', icono: 'solicitud', grupo: 'operacion', tecla: 's' },
    { id: 'trabajos', ruta: 'trabajos', icono: 'trabajos', grupo: 'operacion', tecla: 't' },
    { id: 'aplicaciones', ruta: 'aplicaciones', icono: 'aplicacion', grupo: 'integracion', tecla: 'a' },
    { id: 'imagenes', ruta: 'imagenes', icono: 'imagen', grupo: 'integracion', tecla: 'm' },
    { id: 'ajustes', ruta: 'ajustes', icono: 'ajustes', grupo: 'sistema', tecla: 'j' },
    { id: 'ayuda', ruta: 'ayuda', icono: 'ayuda', grupo: 'sistema', tecla: 'y' },
  ];
  // En 0.1.0 las secciones eran #clientes, #recursos, #impresoras…: esos enlaces siguen funcionando.
  const HASH_VIEJOS = { clientes: 'aplicaciones', recursos: 'imagenes' };
  const rutaDe = (id) => `/admin/${SECCIONES.find((s) => s.id === id).ruta}`;
  const seccionDe = (ruta) => SECCIONES.find((s) => s.ruta === ruta.replace(/^\/admin\/?/, '').replace(/\/$/, ''))?.id;

  async function arrancar() {
    try {
      estado.sesion = await api.get('/sesion');
      api.csrf = estado.sesion.csrf || '';
    } catch (e) {
      vaciar(raiz, h('div', { class: 'acceso' }, h('div', { class: 'tarjeta' }, h('h1', {}, 'Sin conexión'), h('p', { class: 'tenue' }, e.message))));
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

  function campoContrasena(id, etiqueta, ayuda, auto = 'current-password') {
    const input = h('input', { type: 'password', id, name: id, autocomplete: auto, required: true });
    return { input, el: h('label', { class: 'campo', for: id }, h('span', {}, etiqueta), input, ayuda && h('span', { class: 'ayuda' }, ayuda)) };
  }

  function pantallaEntrar() {
    const pass = campoContrasena('contrasena', 'Contraseña');
    const error = h('p', { class: 'nota error', hidden: true });
    const boton = h('button', { class: 'boton primario', type: 'submit' }, 'Entrar');
    const form = h('form', { class: 'pila', onsubmit: async (ev) => {
      ev.preventDefault();
      error.hidden = true;
      try {
        await conEspera(boton, async () => {
          const r = await api.post('/entrar', { contrasena: pass.input.value });
          api.csrf = r.csrf;
        });
        arrancar();
      } catch (e) {
        error.textContent = e.sugerencia ? `${e.message} ${e.sugerencia}` : e.message;
        error.hidden = false; pass.input.select();
      }
    } }, pass.el, error, boton);
    vaciar(raiz, h('main', { class: 'acceso' }, h('div', { class: 'tarjeta' },
      cintaDemo(), marca('Panel de administración'), h('h1', {}, 'Entrar al panel'), form,
      h('p', { class: 'tenue pequeno' }, '¿Olvidaste la contraseña? En el equipo del puente ejecuta ', h('code', {}, 'prisma admin restablecer-contrasena'), '.'))));
    pass.input.focus();
  }

  function formularioNuevaContrasena({ titulo, texto, conCodigo = false, conActual = false, ruta, boton: textoBoton }) {
    const actual = conActual && campoContrasena('actual', 'Contraseña actual');
    const nueva = campoContrasena('nueva', 'Contraseña nueva', 'Al menos 10 caracteres. Una frase larga es más fácil de recordar.', 'new-password');
    const repetir = campoContrasena('repetir', 'Repite la contraseña nueva', null, 'new-password');
    const codigo = conCodigo && h('input', { type: 'text', id: 'codigo', class: 'mono', autocomplete: 'one-time-code', inputmode: 'text' });
    const error = h('p', { class: 'nota error', hidden: true });
    const boton = h('button', { class: 'boton primario', type: 'submit' }, textoBoton);
    const form = h('form', { class: 'pila', onsubmit: async (ev) => {
      ev.preventDefault();
      error.hidden = true;
      const mostrar = (m) => { error.textContent = m; error.hidden = false; };
      if (nueva.input.value.length < 10) return mostrar('La contraseña debe tener al menos 10 caracteres.');
      if (nueva.input.value !== repetir.input.value) return mostrar('Las dos contraseñas no coinciden.');
      try {
        await conEspera(boton, async () => {
          const r = await api.post(ruta, { contrasena: nueva.input.value, actual: actual ? actual.input.value : undefined, codigo: codigo ? codigo.value.trim() : undefined });
          if (r.csrf) api.csrf = r.csrf;
        });
        aviso('Contraseña guardada.', 'ok');
        arrancar();
      } catch (e) { mostrar(e.sugerencia ? `${e.message} ${e.sugerencia}` : e.message); }
    } },
    codigo && h('label', { class: 'campo', for: 'codigo' }, h('span', {}, 'Código de configuración'), codigo,
      h('span', { class: 'ayuda' }, 'Estás entrando desde otro equipo. En el equipo del puente ejecuta ', h('code', {}, 'prisma admin codigo-inicial'), ' y escribe aquí el código que muestra.')),
    actual && actual.el, nueva.el, repetir.el, error, boton);
    vaciar(raiz, h('main', { class: 'acceso' }, h('div', { class: 'tarjeta' },
      cintaDemo(), marca('Panel de administración'), h('h1', {}, titulo), h('p', { class: 'tenue' }, texto), form)));
  }
  function pantallaConfiguracionInicial() {
    formularioNuevaContrasena({
      titulo: 'Configuración inicial',
      texto: 'Es la primera vez que se abre el panel. Crea la contraseña de administración: con ella se configuran impresoras y aplicaciones.',
      conCodigo: !estado.sesion.local, ruta: '/configuracion-inicial', boton: 'Crear contraseña y entrar',
    });
  }
  function pantallaCambioObligatorio() {
    formularioNuevaContrasena({
      titulo: 'Cambia la contraseña',
      texto: 'Entraste con una contraseña generada por «prisma admin restablecer-contrasena». Elige una propia para continuar.',
      ruta: '/contrasena', boton: 'Guardar y continuar',
    });
  }

  let app;
  function mostrarApp() {
    const viejo = location.hash.slice(1);
    if (viejo && (HASH_VIEJOS[viejo] || SECCIONES.some((x) => x.id === viejo))) history.replaceState(null, '', rutaDe(HASH_VIEJOS[viejo] || viejo));
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
        h('span', { class: 'marca-nombre' }, 'PRISMA'), h('span', { class: 'marca-puente', id: 'nombre-puente' }, estado.nombrePuente))),
      grupos, actual: seccionActual(), inferior: ['inicio', 'impresoras', 'solicitudes', 'trabajos', 'ayuda'],
      pie: [UI.selectorTema(), h('span', { id: 'version' }, estado.version && TN.version(estado.version)), botonAtajos(), salir],
    });
    vaciar(raiz, app);
    refrescarPendientes();
    navegar();
  }
  function seccionActual() { return seccionDe(location.pathname) || 'inicio'; }
  function ir(id, consulta = '') {
    const destino = rutaDe(id) + consulta;
    if (location.pathname + location.search !== destino) history.pushState(null, '', destino);
    navegar();
  }
  async function navegar() {
    const id = seccionActual();
    app.marcar(id);
    document.title = TN.tituloPagina(TN.secciones[id], estado.nombrePuente);
    const vista = h('div', { class: 'vista' }, cintaDemo(), UI.esqueleto({ lineas: 4 }));
    vaciar(app.principal, vista);
    try {
      await VISTAS[id](vista);
      vista.querySelector('.esqueleto')?.remove();
    } catch (e) {
      if (e.estado === 401) return;
      vaciar(vista, cintaDemo(), h('div', { class: 'nota error' }, e.message, ' ', e.sugerencia || ''));
    }
    // El foco va al título de la pantalla: el lector de pantalla anuncia dónde se está.
    const titulo = vista.querySelector('h1');
    if (titulo) { titulo.tabIndex = -1; titulo.focus({ preventScroll: true }); } else app.principal.focus({ preventScroll: true });
    window.scrollTo(0, 0);
  }
  function encabezado(titulo, subtitulo, ...acciones) {
    return h('div', { class: 'encabezado' }, h('div', {}, h('h1', {}, titulo), subtitulo && h('p', { class: 'subtitulo' }, subtitulo)),
      acciones.length > 0 && h('div', { class: 'acciones' }, acciones));
  }
  function recargar() { navegar(); }
  function actualizarContador() {
    if (app) app.contador('solicitudes', estado.pendientes, TN.pendientes(estado.pendientes));
  }
  // Pide el resumen al entrar y al volver a la pestaña (no cada tanto: eso mantendría viva la sesión
  // y anularía el cierre por inactividad).
  async function refrescarPendientes() {
    try {
      const r = await api.get('/resumen');
      estado.pendientes = r.solicitudes_pendientes;
      estado.nombrePuente = r.puente.nombre;
      estado.version = r.puente.version;
      const nombre = document.getElementById('nombre-puente');
      if (nombre) nombre.textContent = r.puente.nombre;
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
    const id = seccionDe(destino.pathname);
    if (!id) return;
    ev.preventDefault();
    ir(id, destino.search);
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

  async function vistaInicio(vista) {
    const r = await api.get('/resumen');
    estado.pendientes = r.solicitudes_pendientes;
    estado.direccion = r.puente.direccion;
    actualizarContador();

    const imps = r.impresoras;
    const listas = imps.filter((i) => i.estado === 'en_linea').length;
    const sinRespuesta = imps.filter((i) => i.estado === 'fuera_de_linea');
    const cert = r.certificado_actual;
    const certMal = cert && cert.estado !== 'vigente';
    const porRevisar = sinRespuesta.length + (r.solicitudes_pendientes > 0 ? 1 : 0) + (r.trabajos_hoy.fallidos > 0 ? 1 : 0) + (certMal ? 1 : 0);

    agregar(vista,
      encabezado(r.puente.nombre, TI.subtitulo, porRevisar === 0 ? UI.insignia('ok', TI.todoBien) : UI.insignia('aviso', TI.porRevisar(porRevisar))),
      r.certificado && UI.nota({
        tipo: r.certificado.estado === 'vencido' ? 'error' : 'aviso',
        titulo: r.certificado.estado === 'vencido' ? TI.certificado.avisoVencido(r.certificado.vence)
          : TI.certificado.avisoVence(TI.certificado.dias(r.certificado.dias_restantes), r.certificado.vence),
        texto: r.certificado.autofirmado ? TI.certificado.avisoAutofirmado : TI.certificado.avisoArchivos,
        acciones: enlaceBoton('ajustes', TI.certificado.irAjustes, 'primario'),
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
        tablaUltimos(r.ultimos_trabajos)));
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
      cert.estado !== 'vigente' ? enlaceBoton('ajustes', TI.certificado.renovar)
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
  async function vistaImpresoras(vista) {
    const lista = await api.get('/impresoras');
    const contenedor = h('div', { class: 'lista' });
    agregar(vista,
      encabezado('Impresoras', 'Cada impresora tiene un alias con el que las aplicaciones la eligen.',
        h('button', { class: 'boton primario', type: 'button', onclick: () => asistenteImpresora(vista) }, 'Agregar impresora')),
      contenedor);
    if (lista.length === 0) {
      agregar(contenedor, h('div', { class: 'tarjeta vacio' }, h('p', {}, 'Todavía no hay impresoras.'),
        h('div', { class: 'acciones' }, h('button', { class: 'boton primario', type: 'button', onclick: () => asistenteImpresora(vista) }, 'Agregar la primera'),
          botonCrearVirtual())));
    }
    for (const imp of lista) agregar(contenedor, tarjetaImpresora(imp, vista));
  }

  function tarjetaImpresora(imp, vista) {
    const resultado = h('p', { class: 'resultado', hidden: true });
    const mostrar = (texto, tipo) => { resultado.className = `resultado nota ${tipo}`; resultado.textContent = texto; resultado.hidden = false; };
    const ruta = `/impresoras/${enc(imp.alias)}`;
    const bProbar = h('button', { class: 'boton chico', type: 'button', onclick: async () => {
      try { const r = await conEspera(bProbar, () => api.post(`${ruta}/probar`)); mostrar(r.mensaje || 'La impresora respondió: está en línea.', 'ok'); }
      catch (e) { mostrar(`${e.codigo ? e.codigo + ': ' : ''}${e.message} ${e.sugerencia || ''}`, 'error'); }
    } }, 'Probar conexión');
    const bDiag = h('button', { class: 'boton chico', type: 'button', onclick: async () => {
      try {
        const r = await conEspera(bDiag, () => api.post(`${ruta}/diagnostico`));
        if (r.aviso) mostrar(`Ticket de diagnóstico enviado (trabajo ${r.trabajo_id}), con un aviso: ${r.aviso}`, 'aviso');
        else mostrar(`Ticket de diagnóstico enviado (trabajo ${r.trabajo_id}).`, 'ok');
        dialogo({ titulo: 'Ticket de diagnóstico', ancho: 520, cuerpo: h('div', { class: 'pila' },
          h('p', { class: 'tenue pequeno' }, imp.virtual ? TV.diagnosticoVirtual : TV.diagnosticoFisica),
          r.imagen ? UI.ticketPapel({ imagen: r.imagen, anchoMm: imp.ancho_mm })
            : h('pre', { class: 'papel', tabindex: '0', 'aria-label': 'Vista del ticket' }, r.vista)) }).then(() => { if (imp.virtual) recargar(); });
      } catch (e) { mostrar(`${e.codigo ? e.codigo + ': ' : ''}${e.message} ${e.sugerencia || ''}`, 'error'); }
    } }, 'Ticket de diagnóstico');
    const bPaginas = UI.boton({ texto: TAc.boton, chico: true });
    bPaginas.addEventListener('click', async () => { if (await probarAcentos(imp, bPaginas)) recargar(); });
    const perfil = `${imp.ancho_mm} mm · ${imp.columnas} columnas · ${imp.pagina_codigos} (ESC t ${imp.comando_pagina_codigos})`;
    const capacidades = [
      imp.soporta_corte ? 'cuchilla' : 'sin cuchilla', imp.soporta_cajon && 'cajón',
      imp.soporta_qr_nativo ? 'QR nativo' : 'QR como imagen', !imp.soporta_imagen && 'sin imágenes',
      imp.confirmar_impresion && 'confirma la impresión',
    ].filter(Boolean).join(' · ');
    return h('article', { class: 'tarjeta' },
      h('div', { class: 'tarjeta-cabeza' },
        h('div', { class: 'impresora-cabeza' }, h('span', { class: 'alias' }, imp.alias), h('span', { class: 'tenue' }, imp.nombre),
          insigniaVirtual(imp), !imp.activa && h('span', { class: 'pastilla sin-punto' }, 'Desactivada')),
        pastilla(ESTADOS_IMPRESORA, imp.activa ? imp.estado : 'desconocido')),
      h('dl', { class: 'datos' },
        h('dt', {}, 'Conexión'), imp.virtual
          ? h('dd', {}, TV.conexion(imp.tickets_recibidos, imp.tickets_en_memoria),
            imp.simulacion !== 'normal' && [' · ', h('strong', {}, TV.simulacionActiva[imp.simulacion])])
          : h('dd', {}, `${TIPOS_CONEXION[imp.tipo]} · `, h('span', { class: 'mono' }, imp.destino)),
        h('dt', {}, 'Perfil'), h('dd', {}, perfil),
        h('dt', {}, 'Capacidades'), h('dd', {}, capacidades),
        imp.ultimo_error && [h('dt', {}, 'Último error'), h('dd', {}, h('span', { class: 'chip aviso' }, imp.ultimo_error))]),
      h('div', { class: 'acciones' }, bProbar, bDiag, !imp.virtual && bPaginas,
        imp.virtual && h('button', { class: 'boton chico', type: 'button', onclick: () => verRollo(imp).catch(avisoError) }, TV.verTickets(imp.tickets_recibidos)),
        h('button', { class: 'boton chico', type: 'button', onclick: () => formularioImpresora(vista, imp) }, 'Editar'),
        h('button', { class: 'boton chico peligro', type: 'button', onclick: async () => {
          // Dice qué aplicaciones se quedan sin esta impresora.
          let usan = [];
          try { usan = (await api.get('/clientes')).filter((c) => c.todas_las_impresoras || c.impresoras.includes(imp.alias)).map((c) => c.nombre); } catch { /* se confirma sin la lista */ }
          const E = window.PRISMA_TEXTOS.eliminarImpresora;
          if (!(await confirmar(E.titulo(imp.alias), E.texto, { boton: E.boton, peligro: true,
            consecuencias: [usan.length ? E.usan(usan) : E.ninguna, E.historial] }))) return;
          try { await api.borrar(ruta); aviso(`Impresora «${imp.alias}» eliminada.`, 'ok'); recargar(); } catch (e) { avisoError(e); }
        } }, 'Eliminar')),
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
  async function verRollo(imp) {
    const ruta = `/impresoras/${enc(imp.alias)}/rollo`;
    const lista = await api.get(ruta);
    const ticket = (t) => {
      const base = `${ruta}/${enc(t.trabajo_id)}`;
      const traza = h('pre', { class: 'codigo', tabindex: '0', 'aria-label': TV.verTraza }, '…');
      const detalles = h('details', { class: 'avanzado' }, h('summary', {}, TV.verTraza), h('div', {}, traza));
      detalles.addEventListener('toggle', async () => {
        if (!detalles.open || traza.dataset.listo) return;
        try { traza.textContent = (await api.get(`${base}/traza`)).traza; traza.dataset.listo = '1'; } catch (e) { traza.textContent = e.message; }
      });
      return h('article', { class: 'tarjeta' },
        h('div', { class: 'fila-entre' },
          h('div', { class: 'pila' }, h('strong', {}, [t.cliente_nombre, t.referencia].filter(Boolean).join(' · ') || t.trabajo_id),
            h('span', { class: 'tenue pequeno numeros' }, `${fechaHora(t.recibido)} · ${t.trabajo_id} · ${TV.bytes(t.bytes)}`)),
          t.advertencias.length > 0 && h('span', { class: 'fila' }, t.advertencias.map((a) => h('span', { class: 'chip aviso' }, a)))),
        UI.ticketPapel({ imagen: api.url(`${base}/imagen`), anchoMm: imp.ancho_mm }),
        h('div', { class: 'acciones' }, enlaceDescarga(`${base}/imagen?descargar=1`, TV.descargarPNG), enlaceDescarga(`${base}/bin`, TV.descargarBin)),
        detalles);
    };
    const cuerpo = h('div', { class: 'pila-amplia' }, UI.nota({ tipo: 'info', texto: TV.rolloAviso }),
      lista.length === 0 ? UI.vacio({ icono: 'ticket', titulo: TV.rolloVacio, texto: TV.rolloVacioTexto }) : lista.map(ticket));
    const accion = await dialogo({ titulo: TV.rolloTitulo(imp.alias), ancho: 720, cuerpo,
      acciones: [lista.length > 0 && { texto: TV.vaciar, clase: 'peligro', valor: 'vaciar' }, { texto: 'Cerrar', valor: null }].filter(Boolean) });
    if (accion !== 'vaciar') return;
    if (!(await confirmar(TV.vaciarTitulo, TV.vaciarTexto, { boton: TV.vaciar, peligro: true, consecuencias: TV.vaciarConsecuencias }))) return;
    await api.borrar(ruta);
    aviso(TV.vaciado, 'ok');
    recargar();
  }

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
        info: { texto: TA.conexion.aliasInfo },
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
            boton(TA.probar.acentosSi, 'primario', () => vaciar(preguntas, final())),
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
  const CDN_SDK = 'https://cdn.jsdelivr.net/gh/bayalad7/prisma@v0.1.0/sdk/js/dist/prisma.umd.js';
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
    const cOrigen = UI.campo({ id: 'cli-origen', etiqueta: F.origen, control: nuevoOrigen });
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
    const href = URL.createObjectURL(new Blob([L.archivo(cliente.nombre, llave, url)], { type: 'text/plain;charset=utf-8' }));
    await dialogo({
      titulo: L.titulo(cliente.nombre), ancho: 700,
      cuerpo: h('div', { class: 'pila-amplia' },
        UI.nota({ tipo: 'aviso', titulo: L.aviso, texto: L.avisoTexto }),
        h('div', { class: 'llave-caja' }, h('code', {}, llave), botonCopiar(llave)),
        h('div', {}, h('a', { class: 'boton chico', href, download: `llave-${aliasDe(cliente.nombre, new Set())}.txt` }, UI.icono('descargar'), L.descargar)),
        h('div', { class: 'pila' }, h('h3', {}, L.queHago), h('p', { class: 'tenue pequeno' }, L.queHagoTexto)),
        bloqueCodigo(cliente, impresoras, llave)),
      acciones: [{ texto: L.listo, clase: 'primario', valor: true }],
    });
    URL.revokeObjectURL(href);
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
    const pintar = (alias) => {
      const saludo = K.saludo(cliente.nombre.replace(/["\\]/g, ''));
      const variantes = {
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
        [K.pestanas.curl]: cliente.nivel === 3 && !conLlave ? [
          `prisma dev firmar documento.json --llave privada.pem --cliente ${cliente.id} --nuevo-id -o firmado.json`,
          `curl --cacert certificado.pem ${url}/v1/imprimir \\`,
          '  -H "Content-Type: application/json" --data-binary @firmado.json',
        ] : [
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
      const quitar = (b, ruta, mensaje) => async () => {
        try { await conEspera(b, () => api.post(ruta, { origen: s.origen })); aviso(mensaje, 'ok'); refrescarPendientes(); recargar(); } catch (e) { avisoError(e); }
      };
      return h('article', { class: 'tarjeta' },
        h('div', { class: 'fila-entre' }, h('span', { class: 'mono solicitud-origen' }, s.origen), h('span', { class: 'fila' }, ctx.insignias)),
        h('p', { class: 'tenue pequeno numeros' }, TS.intentos(s.intentos, hace(s.ultima_vez))),
        h('div', { class: 'acciones' },
          accion(TS.aprobar, 'primario', () => aprobar(s, clientes, impresoras)),
          !rechazada && accion(TS.rechazar, '', (b) => quitar(b, '/solicitudes/rechazar', TS.rechazada(s.origen))()),
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
        h('div', { class: 'acciones' }, imp && botonProbar(), irImpresoras()),
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
  async function vistaRecursos(vista) {
    const lista = await api.get('/recursos');
    let anchoVista = 576;
    let archivo = null;
    const nombre = h('input', { type: 'text', id: 'rec-nombre', class: 'mono', placeholder: 'logo-tienda', pattern: '[a-z0-9][a-z0-9_\\-]{0,63}' });
    const entrada = h('input', { type: 'file', id: 'rec-archivo', accept: 'image/png', hidden: true });
    const previa = h('div', { class: 'recurso-imagen' }, h('span', { class: 'tenue pequeno' }, 'La vista previa aparece aquí, en blanco y negro como se imprimirá.'));
    const zona = h('label', { class: 'soltar', for: 'rec-archivo' }, h('strong', {}, 'Elige un PNG'), h('span', { class: 'pequeno' }, 'o arrástralo aquí. Hasta 512 KB y 4096 px por lado.'));
    const anchoSel = h('div', { class: 'segmentos', role: 'group', 'aria-label': 'Ancho de la vista previa' }, [[384, '58 mm'], [576, '80 mm']].map(([px, t]) =>
      h('button', { type: 'button', 'aria-pressed': String(px === anchoVista), onclick: (ev) => {
        anchoVista = px; for (const b of anchoSel.children) b.setAttribute('aria-pressed', String(b === ev.currentTarget)); vistaPrevia();
      } }, t)));
    const vistaPrevia = async () => {
      if (!archivo) return;
      const fd = new FormData(); fd.append('archivo', archivo); fd.append('ancho', anchoVista);
      try {
        const blob = await api.pedir('POST', '/recursos/vista-previa', fd);
        vaciar(previa, h('img', { src: URL.createObjectURL(blob), alt: 'Vista previa en 1 bit' }));
      } catch (e) { vaciar(previa, h('p', { class: 'nota error' }, e.message)); }
    };
    const tomar = (f) => {
      if (!f) return;
      if (f.type !== 'image/png') { aviso('Solo se aceptan imágenes PNG.', 'error'); return; }
      archivo = f;
      if (!nombre.value) nombre.value = f.name.replace(/\.png$/i, '').toLowerCase().replace(/[^a-z0-9_-]+/g, '-').replace(/^-+/, '').slice(0, 64);
      vistaPrevia();
    };
    entrada.addEventListener('change', () => tomar(entrada.files[0]));
    zona.addEventListener('dragover', (ev) => { ev.preventDefault(); zona.classList.add('encima'); });
    zona.addEventListener('dragleave', () => zona.classList.remove('encima'));
    zona.addEventListener('drop', (ev) => { ev.preventDefault(); zona.classList.remove('encima'); tomar(ev.dataTransfer.files[0]); });
    const bGuardar = h('button', { class: 'boton primario', type: 'submit' }, 'Guardar imagen');
    const form = h('form', { class: 'tarjeta', onsubmit: async (ev) => {
      ev.preventDefault();
      if (!archivo) { aviso('Elige primero un PNG.', 'error'); return; }
      if (!nombre.checkValidity() || !nombre.value) { aviso('El nombre solo admite minúsculas, dígitos, - y _.', 'error'); nombre.focus(); return; }
      const fd = new FormData(); fd.append('archivo', archivo); fd.append('nombre', nombre.value);
      try { await conEspera(bGuardar, () => api.pedir('POST', '/recursos', fd)); aviso(`Imagen «${nombre.value}» guardada.`, 'ok'); recargar(); } catch (e) { avisoError(e); }
    } },
    h('h2', {}, 'Subir una imagen'),
    h('div', { class: 'rejilla' },
      h('div', { class: 'pila' }, zona, entrada,
        h('label', { class: 'campo', for: 'rec-nombre' }, h('span', {}, 'Nombre'), nombre,
          h('span', { class: 'ayuda' }, 'Las aplicaciones la usan con {"tipo": "imagen", "recurso": "nombre"}.'))),
      h('div', { class: 'pila' }, h('div', { class: 'fila-entre' }, h('span', { class: 'pequeno tenue' }, 'Vista previa'), anchoSel), previa)),
    h('div', {}, bGuardar));

    agregar(vista,
      encabezado('Imágenes', 'Logos e imágenes guardados en el puente. Se imprimen con el mismo nombre desde cualquier aplicación.'),
      form,
      lista.length === 0 ? h('p', { class: 'tenue' }, 'Todavía no hay imágenes guardadas.') : h('div', { class: 'recursos' }, lista.map((r) => {
        const uso = `{"tipo": "imagen", "recurso": "${r.nombre}"}`;
        return h('article', { class: 'tarjeta' },
          h('div', { class: 'recurso-imagen' }, h('img', { src: api.url(`/recursos/${enc(r.nombre)}/imagen?ancho=576`), alt: `Imagen ${r.nombre} en 1 bit`, loading: 'lazy' })),
          h('div', { class: 'pila' }, h('span', { class: 'alias' }, r.nombre),
            h('span', { class: 'tenue pequeno numeros' }, `${r.ancho_px} × ${r.alto_px} px · ${tamanoArchivo(r.bytes)}`)),
          h('div', { class: 'acciones' }, botonCopiar(uso, 'Copiar uso'),
            h('button', { class: 'boton chico peligro', type: 'button', onclick: async () => {
              if (!(await confirmar(`¿Eliminar «${r.nombre}»?`, 'Los documentos que la usen recibirán RECURSO_NO_ENCONTRADO.', { boton: 'Eliminar', peligro: true }))) return;
              try { await api.borrar(`/recursos/${enc(r.nombre)}`); aviso('Imagen eliminada.', 'ok'); recargar(); } catch (e) { avisoError(e); }
            } }, 'Eliminar')));
      })));
  }

  // ---------------------------------------------------------------------------------------------
  // 7. Ajustes
  // ---------------------------------------------------------------------------------------------
  function enlaceDescarga(ruta, texto) {
    const a = h('a', { class: 'boton', href: api.url(ruta) }, texto);
    if (window.PRISMA_DEMO) a.addEventListener('click', (ev) => { ev.preventDefault(); aviso('En el prototipo no se descargan archivos.'); });
    return a;
  }

  async function vistaAjustes(vista) {
    const a = await api.get('/ajustes');
    const num = (id, valor, min, max) => h('input', { type: 'number', id, value: valor, min, max });
    const nombre = h('input', { type: 'text', id: 'aj-nombre', value: a.nombre_puente });
    const limIP = num('aj-ip', a.limite_ip, 0, 100000);
    const limCli = num('aj-cliente', a.limite_cliente, 0, 100000);
    const retencion = num('aj-retencion', a.retencion_dias, 1, 3650);
    const privada = h('input', { type: 'checkbox', id: 'aj-privada', checked: a.solo_red_privada });
    const buscar = h('input', { type: 'checkbox', id: 'aj-actualizaciones', checked: a.buscar_actualizaciones });
    const seccion = (id, titulo, texto, campos, guardar) => {
      const b = h('button', { class: 'boton primario chico', type: 'submit' }, 'Guardar');
      return h('form', { class: 'tarjeta', 'aria-labelledby': id, onsubmit: async (ev) => {
        ev.preventDefault();
        try { await conEspera(b, guardar); aviso('Ajustes guardados.', 'ok'); } catch (e) { avisoError(e); }
      } }, h('h2', { id }, titulo), texto && h('p', { class: 'tenue pequeno' }, texto), campos, h('div', {}, b));
    };
    const campo = (id, etiqueta, control, ayuda) => h('label', { class: 'campo', for: id }, h('span', {}, etiqueta), control, ayuda && h('span', { class: 'ayuda' }, ayuda));

    const cert = a.tls.certificado;
    const tls = h('section', { class: 'tarjeta', 'aria-labelledby': 'aj-tls' }, h('h2', { id: 'aj-tls' }, 'HTTPS'),
      h('dl', { class: 'datos' },
        h('dt', {}, 'Modo'), h('dd', {}, { autofirmado: 'Certificado autofirmado', archivos: 'Certificado propio (archivos)', solo_localhost_http: 'HTTP solo en este equipo' }[a.tls.modo]),
        h('dt', {}, 'Dirección'), h('dd', { class: 'mono' }, a.direccion),
        cert && [
          h('dt', {}, 'Vence'), h('dd', {}, cert.vence,
            cert.estado === 'vencido' ? h('span', { class: 'pastilla error' }, 'Vencido')
              : cert.estado === 'por_vencer' ? h('span', { class: 'pastilla aviso' }, `Vence en ${cert.dias_restantes} días`) : null),
          h('dt', {}, 'Nombres'), h('dd', {}, cert.nombres.map((n) => h('span', { class: 'chip' }, n))),
          h('dt', {}, 'IPs'), h('dd', {}, cert.ips.map((n) => h('span', { class: 'chip' }, n))),
          h('dt', {}, 'Huella SHA-256'), h('dd', { class: 'mono pequeno' }, cert.huella)]),
      cert && cert.ips_sin_cubrir && cert.ips_sin_cubrir.length > 0 && h('p', { class: 'nota aviso' },
        `El certificado no incluye las IPs actuales ${cert.ips_sin_cubrir.join(', ')}. Los equipos que usen esas direcciones verán un error de certificado: regenéralo.`),
      h('div', { class: 'acciones' },
        cert && enlaceDescarga('/tls/certificado', 'Descargar certificado'),
        a.tls.modo === 'autofirmado' && h('button', { class: 'boton peligro', type: 'button', onclick: async () => {
          if (!(await confirmar('¿Regenerar el certificado?', 'Se crea uno nuevo con el nombre y las IPs actuales de este equipo. Cada dispositivo que confiaba en el anterior tendrá que confiar en el nuevo.', { boton: 'Regenerar', peligro: true }))) return;
          try { await api.post('/tls/regenerar'); aviso('Certificado regenerado. El puente ya lo usa.', 'ok'); recargar(); } catch (e) { avisoError(e); }
        } }, 'Regenerar certificado')),
      h('p', { class: 'tenue pequeno' }, 'Cómo instalarlo en cada dispositivo: ', h('a', { href: rutaDe('ayuda') }, 'Ayuda'), '.'));

    const actual = h('input', { type: 'password', id: 'aj-actual', autocomplete: 'current-password' });
    const nueva = h('input', { type: 'password', id: 'aj-nueva', autocomplete: 'new-password' });
    const repetir = h('input', { type: 'password', id: 'aj-repetir', autocomplete: 'new-password' });

    const archivoRespaldo = h('input', { type: 'file', id: 'aj-restaurar', accept: '.db,application/octet-stream' });
    const confirmarRestaurar = h('input', { type: 'text', id: 'aj-confirmar', class: 'mono', placeholder: 'RESTAURAR', autocomplete: 'off' });
    const bRestaurar = h('button', { class: 'boton peligro', type: 'button', onclick: async () => {
      if (!archivoRespaldo.files[0]) { aviso('Elige el archivo de respaldo.', 'error'); return; }
      if (confirmarRestaurar.value !== 'RESTAURAR') { aviso('Escribe RESTAURAR para confirmar.', 'error'); confirmarRestaurar.focus(); return; }
      const fd = new FormData(); fd.append('archivo', archivoRespaldo.files[0]);
      try { await conEspera(bRestaurar, () => api.pedir('POST', '/restaurar', fd)); aviso('Respaldo restaurado. Vuelve a entrar al panel.', 'ok'); estado.sesion = null; setTimeout(arrancar, 1500); } catch (e) { avisoError(e); }
    } }, 'Restaurar');

    agregar(vista,
      encabezado('Ajustes', null),
      h('div', { class: 'rejilla' },
        seccion('aj-p', 'Puente', 'El nombre aparece en el panel y en GET /v1/salud.',
          campo('aj-nombre', 'Nombre del puente', nombre), () => api.put('/ajustes', { nombre_puente: nombre.value.trim() })),
        seccion('aj-l', 'Límites de peticiones', 'Peticiones por minuto a la API. Admiten ráfagas de ese tamaño. 0 desactiva el límite.',
          h('div', { class: 'campos' }, campo('aj-ip', 'Por dirección IP', limIP), campo('aj-cliente', 'Por aplicación', limCli)),
          () => api.put('/ajustes', { limite_ip: +limIP.value, limite_cliente: +limCli.value })),
        seccion('aj-h', 'Historial', 'Los trabajos más antiguos se borran solos. Nunca se guarda el contenido de los tickets.',
          campo('aj-retencion', 'Días que se conservan', retencion), () => api.put('/ajustes', { retencion_dias: +retencion.value })),
        seccion('aj-a', 'Acceso al panel', null,
          h('label', { class: 'casilla', for: 'aj-privada' }, privada, h('span', {}, 'Solo desde la red local (direcciones privadas). Recomendado.')),
          () => api.put('/ajustes', { solo_red_privada: privada.checked })),
        seccion('aj-u', 'Actualizaciones', 'Una vez al día, el puente pregunta a GitHub si hay una versión nueva y lo avisa en Inicio. No envía ningún dato del negocio. Nunca se instala nada solo.',
          h('label', { class: 'casilla', for: 'aj-actualizaciones' }, buscar, h('span', {}, 'Buscar versiones nuevas')),
          () => api.put('/ajustes', { buscar_actualizaciones: buscar.checked }))),
      tls,
      h('div', { class: 'rejilla' },
        seccion('aj-c', 'Contraseña del panel', 'Al menos 10 caracteres.',
          h('div', { class: 'pila' }, campo('aj-actual', 'Contraseña actual', actual), campo('aj-nueva', 'Contraseña nueva', nueva), campo('aj-repetir', 'Repite la nueva', repetir)),
          async () => {
            if (nueva.value.length < 10) throw new ErrorApi({ mensaje: 'La contraseña nueva debe tener al menos 10 caracteres.' });
            if (nueva.value !== repetir.value) throw new ErrorApi({ mensaje: 'Las dos contraseñas nuevas no coinciden.' });
            await api.post('/contrasena', { actual: actual.value, contrasena: nueva.value });
            actual.value = nueva.value = repetir.value = '';
          }),
        h('section', { class: 'tarjeta', 'aria-labelledby': 'aj-r' }, h('h2', { id: 'aj-r' }, 'Respaldo'),
          h('p', { class: 'tenue pequeno' }, 'Una copia completa de la base de datos: impresoras, aplicaciones, imágenes, historial y ajustes. Contiene huellas de llaves: guárdala en un lugar seguro.'),
          h('div', {}, enlaceDescarga('/respaldo', 'Descargar respaldo')),
          h('hr', { class: 'separador' }),
          h('h3', {}, 'Restaurar'),
          h('p', { class: 'nota aviso' }, 'Reemplaza todo lo que hay ahora en el puente por el contenido del respaldo.'),
          h('label', { class: 'campo', for: 'aj-restaurar' }, h('span', {}, 'Archivo de respaldo'), archivoRespaldo),
          h('label', { class: 'campo', for: 'aj-confirmar' }, h('span', {}, 'Escribe RESTAURAR para confirmar'), confirmarRestaurar),
          h('div', {}, bRestaurar))));
  }

  // ---------------------------------------------------------------------------------------------
  // 8. Ayuda
  // ---------------------------------------------------------------------------------------------
  // La documentación de la versión instalada: la etiqueta v<versión> si es una versión publicada, o
  // main en una compilación de desarrollo (0.0.0-dev) o intermedia (git describe con -N-gabc).
  const docsDe = (version) => `https://github.com/bayalad7/prisma/blob/${/^\d+\.\d+\.\d+(-rc\.\d+)?$/.test(version || '') ? `v${version}` : 'main'}/docs/`;
  const GUIAS_CERT = {
    Windows: ['Descarga el certificado (Ajustes → HTTPS) y ábrelo con doble clic.', 'Instalar certificado → Equipo local.', 'Colocar todos los certificados en el siguiente almacén → «Entidades de certificación raíz de confianza».', 'Reinicia el navegador. Chrome y Edge usan el almacén de Windows.'],
    macOS: ['Descarga el certificado y ábrelo: se agrega al llavero.', 'En Acceso a Llaveros, ábrelo y en «Confiar» elige «Confiar siempre».', 'O en Terminal: sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain certificado.pem'],
    Linux: ['sudo cp certificado.pem /usr/local/share/ca-certificates/prisma.crt', 'sudo update-ca-certificates', 'Chrome/Chromium: certutil -d sql:$HOME/.pki/nssdb -A -t "C,," -n PRISMA -i certificado.pem'],
    Android: ['Copia el certificado al teléfono.', 'Ajustes → Seguridad → Cifrado y credenciales → Instalar un certificado → Certificado de CA.', 'Chrome para Android confía en los certificados de CA instalados así.'],
    'iPhone / iPad': ['Abre el certificado en Safari: se descarga como perfil.', 'Ajustes → General → VPN y gestión de dispositivos → instala el perfil.', 'Ajustes → General → Información → Ajustes de confianza de certificados → activa PRISMA.'],
    Firefox: ['Ajustes → Privacidad y seguridad → Certificados → Ver certificados → Autoridades → Importar.', 'Marca «Confiar en esta CA para identificar sitios web».'],
  };
  async function vistaAyuda(vista) {
    let huella = '';
    let docs = docsDe('');
    try { huella = (await api.get('/ajustes')).tls.certificado?.huella || ''; } catch { /* la ayuda funciona sin huella */ }
    try { docs = docsDe((await api.get('/resumen')).puente.version); } catch { /* sin versión, enlaza main */ }
    const pasos = h('ol', { class: 'pila' });
    const pestanas = h('div', { class: 'pestanas', role: 'tablist' });
    const elegir = (so) => {
      vaciar(pasos, GUIAS_CERT[so].map((p) => h('li', {}, /^(sudo|certutil|O en Terminal)/.test(p) ? h('code', {}, p) : p)));
      for (const b of pestanas.children) b.setAttribute('aria-selected', String(b.textContent === so));
    };
    agregar(pestanas, Object.keys(GUIAS_CERT).map((so) => h('button', { type: 'button', role: 'tab', onclick: () => elegir(so) }, so)));
    elegir('Windows');
    const enlace = (archivo, texto, detalle) => h('li', {}, h('a', { href: docs + archivo, target: '_blank', rel: 'noopener' }, texto), ' — ', h('span', { class: 'tenue' }, detalle));
    agregar(vista,
      encabezado('Ayuda', null),
      h('section', { class: 'tarjeta' }, h('h2', {}, 'Confiar en el certificado del puente'),
        h('p', {}, 'Los navegadores solo dejan imprimir a las páginas HTTPS si confían en el certificado del puente. Hazlo una vez en cada equipo o teléfono desde el que se imprima.'),
        huella && h('p', { class: 'pequeno' }, 'Antes de instalarlo, comprueba que su huella sea: ', h('span', { class: 'mono' }, huella)),
        h('div', {}, enlaceDescarga('/tls/certificado', 'Descargar certificado')),
        pestanas, pasos),
      h('section', { class: 'tarjeta' }, h('h2', {}, 'Integrar una aplicación'),
        h('ol', { class: 'pila' },
          h('li', {}, 'Agrega la aplicación en ', h('a', { href: rutaDe('aplicaciones') }, 'Aplicaciones'), ' y elige su nivel de seguridad.'),
          h('li', {}, 'Copia su código de integración: incluye la dirección del puente y, si corresponde, su llave.'),
          h('li', {}, 'Arma los tickets como documentos JSON con las operaciones del contrato (texto, filas, QR, imágenes, corte…).')),
        h('ul', { class: 'pila' },
          enlace('administracion/README.md', 'Manual del panel', 'impresoras, aplicaciones, solicitudes y ajustes'),
          enlace('integradores/README.md', 'Guía para integradores', 'SDK, niveles de seguridad y firmas'),
          enlace('referencia/api-v1.md', 'API v1', 'endpoints, autorización, CORS y errores'),
          enlace('referencia/contrato-documento-v1.md', 'Contrato del documento', 'todas las operaciones con ejemplos'),
          enlace('instalacion/https-y-certificados.md', 'HTTPS y certificados', 'modos y cómo confiar en cada sistema'),
          enlace('instalacion/impresoras.md', 'Conectar impresoras', 'red, USB y Bluetooth en cada sistema'),
          enlace('administracion/problemas-comunes.md', 'Solución de problemas', 'errores frecuentes, acentos y columnas'),
          enlace('operacion/README.md', 'Operación', 'respaldo, actualización, bitácoras y recuperación'))));
  }

  const VISTAS = {
    inicio: vistaInicio, impresoras: vistaImpresoras, aplicaciones: vistaClientes, solicitudes: vistaSolicitudes,
    trabajos: vistaTrabajos, imagenes: vistaRecursos, ajustes: vistaAjustes, ayuda: vistaAyuda,
  };

  arrancar();
})();
