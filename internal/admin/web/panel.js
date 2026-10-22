// PRISMA — panel de administración. JavaScript sin dependencias.
// Toda la información llega de la API interna /admin/api/* (o de datos de ejemplo en el prototipo).
// El texto que viene del servidor se inserta siempre como texto, nunca como HTML.
'use strict';
(() => {
  const raiz = document.getElementById('raiz');

  // Utilidades de DOM y componentes del sistema de diseño (sistema.js).
  const { h, agregar, vaciar, logo, aviso, avisoError, dialogo, confirmar, copiar, botonCopiar, conEspera } = window.PrismaUI;

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
  const ESTADOS_TRABAJO = {
    completado: ['ok', 'Impreso'], fallido: ['error', 'Falló'], encolado: ['acento', 'En cola'], imprimiendo: ['aviso', 'Imprimiendo'],
  };
  function pastilla(mapa, clave) {
    const [clase, texto] = mapa[clave] || ['', clave];
    return h('span', { class: `pastilla ${clase}` }, texto);
  }
  const NIVELES = {
    1: { nombre: 'Origen', texto: 'Se identifica por la página web (encabezado Origin). Para páginas sin backend en una red de confianza.' },
    2: { nombre: 'Llave', texto: 'Origen aprobado más una llave pk_ en cada petición. Recomendado para la mayoría.' },
    3: { nombre: 'Firma', texto: 'El backend firma cada documento con Ed25519. Para aplicaciones que no deben poder ser suplantadas.' },
  };
  const PAGINAS = [
    { pagina: 'CP437', comando: 0, etiqueta: 'PC437 (EE. UU.)' },
    { pagina: 'CP850', comando: 2, etiqueta: 'PC850 (Multilingüe)' },
    { pagina: 'WPC1252', comando: 16, etiqueta: 'WPC1252 (Windows)' },
    { pagina: 'CP858', comando: 19, etiqueta: 'PC858 (Multilingüe con €)' },
  ];
  const PERFILES = { 58: { columnas: 32, columnas_fuente_b: 42, ancho_px: 384 }, 80: { columnas: 48, columnas_fuente_b: 64, ancho_px: 576 } };
  const TIPOS_CONEXION = { red: 'Red', usb: 'USB', bluetooth: 'Bluetooth', archivo: 'Archivo' };

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
      try { r = await fetch('api' + ruta, op); } catch {
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
    url: (ruta) => (window.PRISMA_DEMO && window.PRISMA_DEMO.url) ? window.PRISMA_DEMO.url(ruta) : 'api' + ruta,
  };
  const enc = encodeURIComponent;

  // ---------------------------------------------------------------------------------------------
  // Estado, acceso y enrutado
  // ---------------------------------------------------------------------------------------------
  const estado = { sesion: null, pendientes: 0, nombrePuente: 'PRISMA', direccion: '' };
  const SECCIONES = [
    ['inicio', 'Inicio'], ['impresoras', 'Impresoras'], ['clientes', 'Aplicaciones'], ['solicitudes', 'Solicitudes'],
    ['trabajos', 'Trabajos'], ['recursos', 'Imágenes'], ['ajustes', 'Ajustes'], ['ayuda', 'Ayuda'],
  ];

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

  let barra, principal;
  function mostrarApp() {
    const nav = h('nav', { class: 'nav', 'aria-label': 'Secciones' });
    const menu = h('button', { class: 'boton chico menu-boton', type: 'button', 'aria-expanded': 'false', onclick: () => {
      const abierta = barra.classList.toggle('abierta');
      menu.setAttribute('aria-expanded', String(abierta));
    } }, 'Menú');
    barra = h('header', { class: 'barra' },
      h('a', { class: 'marca', href: '#inicio' }, logo(30), h('span', { class: 'marca-texto' },
        h('span', { class: 'marca-nombre' }, 'PRISMA'), h('span', { class: 'marca-puente', id: 'nombre-puente' }, estado.nombrePuente))),
      menu, nav,
      h('div', { class: 'barra-pie' }, h('span', { id: 'version' }),
        h('button', { class: 'boton chico', type: 'button', onclick: async () => {
          try { await api.post('/salir'); } catch { /* la sesión ya no existe */ }
          estado.sesion = null; arrancar();
        } }, 'Cerrar sesión')));
    principal = h('main', { class: 'contenido', id: 'principal', tabindex: '-1' });
    vaciar(raiz, h('div', { class: 'app' }, barra, principal));
    pintarNav();
    window.onhashchange = navegar;
    navegar();
  }
  function pintarNav() {
    const actual = seccionActual();
    vaciar(barra.querySelector('.nav'), SECCIONES.map(([id, nombre]) =>
      h('a', { href: `#${id}`, 'aria-current': id === actual ? 'page' : false },
        h('span', {}, nombre), id === 'solicitudes' && estado.pendientes > 0 && h('span', { class: 'contador', 'aria-label': `${estado.pendientes} pendientes` }, estado.pendientes))));
  }
  function seccionActual() {
    const id = location.hash.slice(1);
    return SECCIONES.some(([s]) => s === id) ? id : 'inicio';
  }
  async function navegar() {
    barra.classList.remove('abierta');
    pintarNav();
    const id = seccionActual();
    const vista = h('div', { class: 'vista' }, cintaDemo(), h('p', { class: 'tenue', id: 'cargando' }, 'Cargando…'));
    vaciar(principal, vista);
    try {
      await VISTAS[id](vista);
      vista.querySelector('#cargando')?.remove();
    } catch (e) {
      if (e.estado === 401) return;
      vaciar(vista, cintaDemo(), h('div', { class: 'nota error' }, e.message, ' ', e.sugerencia || ''));
    }
    principal.focus({ preventScroll: true });
    window.scrollTo(0, 0);
  }
  function encabezado(titulo, subtitulo, ...acciones) {
    return h('div', { class: 'encabezado' }, h('div', {}, h('h1', {}, titulo), subtitulo && h('p', { class: 'subtitulo' }, subtitulo)),
      acciones.length > 0 && h('div', { class: 'acciones' }, acciones));
  }
  function recargar() { navegar(); }
  async function refrescarPendientes() {
    try { estado.pendientes = (await api.get('/resumen')).solicitudes_pendientes; } catch { /* sin cambios */ }
    pintarNav();
  }

  // ---------------------------------------------------------------------------------------------
  // 1. Inicio
  // ---------------------------------------------------------------------------------------------
  async function vistaInicio(vista) {
    const r = await api.get('/resumen');
    estado.pendientes = r.solicitudes_pendientes;
    estado.nombrePuente = r.puente.nombre;
    estado.direccion = r.puente.direccion;
    document.getElementById('nombre-puente').textContent = r.puente.nombre;
    document.getElementById('version').textContent = `Versión ${r.puente.version}`;
    pintarNav();

    const enLinea = r.impresoras.filter((i) => i.estado === 'en_linea').length;
    agregar(vista,
      encabezado(r.puente.nombre, 'Estado del puente de impresión de este negocio.'),
      r.solicitudes_pendientes > 0 && h('div', { class: 'aviso-caja', role: 'status' },
        h('div', {}, h('strong', {}, r.solicitudes_pendientes === 1 ? '1 sitio web quiere imprimir' : `${r.solicitudes_pendientes} sitios web quieren imprimir`),
          h('div', { class: 'pequeno' }, 'Intentaron imprimir desde un origen que no está aprobado. No imprimen hasta que los apruebes.')),
        h('a', { class: 'boton primario', href: '#solicitudes' }, 'Revisar solicitudes')),
      r.certificado && h('div', { class: 'aviso-caja', role: 'alert' },
        h('div', {},
          h('strong', {}, r.certificado.estado === 'vencido'
            ? `El certificado HTTPS venció el ${r.certificado.vence}: las aplicaciones no pueden imprimir`
            : `El certificado HTTPS vence en ${r.certificado.dias_restantes} días (${r.certificado.vence})`),
          h('div', { class: 'pequeno' }, r.certificado.autofirmado
            ? 'Regenéralo en Ajustes → HTTPS y vuelve a instalarlo en cada equipo que imprime.'
            : 'Renueva los archivos del certificado configurados en config.toml; el puente los toma sin reiniciar.')),
        h('a', { class: 'boton primario', href: '#ajustes' }, 'Ir a Ajustes')),
      r.actualizacion && h('div', { class: 'aviso-caja', role: 'status' },
        h('div', {}, h('strong', {}, `Hay una versión nueva de PRISMA: ${r.actualizacion.version}`),
          h('div', { class: 'pequeno' }, `Tienes la ${r.puente.version}. En el equipo del puente, como administrador, ejecuta «prisma actualizar», o descarga el instalador de la página de la versión.`)),
        h('a', { class: 'boton', href: r.actualizacion.url, target: '_blank', rel: 'noopener noreferrer' }, 'Ver la versión')),
      h('div', { class: 'rejilla' },
        h('section', { class: 'tarjeta', 'aria-labelledby': 't-puente' }, h('h2', { id: 't-puente' }, 'Puente'),
          h('dl', { class: 'datos' },
            h('dt', {}, 'Dirección'), h('dd', { class: 'mono' }, r.puente.direccion),
            h('dt', {}, 'HTTPS'), h('dd', {}, { autofirmado: 'Certificado autofirmado', archivos: 'Certificado propio', solo_localhost_http: 'HTTP solo en este equipo' }[r.puente.modo_tls] || r.puente.modo_tls),
            h('dt', {}, 'En marcha'), h('dd', {}, `desde ${fechaHora(r.puente.iniciado)}`),
            h('dt', {}, 'Versión'), h('dd', { class: 'mono' }, r.puente.version))),
        h('section', { class: 'tarjeta', 'aria-labelledby': 't-imp' },
          h('div', { class: 'tarjeta-cabeza' }, h('h2', { id: 't-imp' }, 'Impresoras'),
            h('span', { class: 'tenue pequeno numeros' }, `${enLinea} de ${r.impresoras.length} en línea`)),
          r.impresoras.length === 0
            ? h('p', { class: 'tenue' }, 'Todavía no hay impresoras. ', h('a', { href: '#impresoras' }, 'Agrega la primera.'))
            : h('div', { class: 'pila' }, r.impresoras.map((i) => h('div', { class: 'fila-entre' },
              h('span', {}, h('span', { class: 'mono' }, i.alias), ' ', h('span', { class: 'tenue pequeno' }, i.nombre)),
              h('span', { class: 'fila' }, i.pendientes > 0 && h('span', { class: 'tenue pequeno numeros' }, `${i.pendientes} en cola`), pastilla(ESTADOS_IMPRESORA, i.estado)))))),
        h('section', { class: 'tarjeta', 'aria-labelledby': 't-hoy' }, h('h2', { id: 't-hoy' }, 'Hoy'),
          h('div', { class: 'cifras' },
            h('div', {}, h('div', { class: 'cifra' }, fmtNum.format(r.trabajos_hoy.completados)), h('div', { class: 'tenue pequeno' }, 'tickets impresos')),
            h('div', {}, h('div', { class: 'cifra' }, fmtNum.format(r.trabajos_hoy.fallidos)), h('div', { class: 'tenue pequeno' }, 'con error'))),
          h('a', { href: '#trabajos', class: 'pequeno' }, 'Ver todos los trabajos'))),
      h('section', { class: 'pila', 'aria-labelledby': 't-ultimos' }, h('h2', { id: 't-ultimos' }, 'Últimos trabajos'), tablaTrabajos(r.ultimos_trabajos)));
  }

  function tablaTrabajos(lista) {
    if (lista.length === 0) return h('div', { class: 'tabla-caja' }, h('p', { class: 'vacio' }, 'No hay trabajos con estos filtros.'));
    return h('div', { class: 'tabla-caja' }, h('table', {},
      h('thead', {}, h('tr', {}, ['Fecha', 'Aplicación', 'Impresora', 'Estado', 'Referencia', 'Error / avisos', 'Duración'].map((t) => h('th', { scope: 'col' }, t)))),
      h('tbody', {}, lista.map((t) => h('tr', {},
        h('td', { class: 'numeros' }, fechaHora(t.creado)),
        h('td', {}, t.cliente_nombre || h('span', { class: 'mono' }, t.cliente_id)),
        h('td', { class: 'mono' }, t.impresora_alias),
        h('td', {}, pastilla(ESTADOS_TRABAJO, t.estado)),
        h('td', { class: 'mono' }, t.referencia || '—'),
        h('td', {}, t.codigo_error && h('span', { class: 'chip aviso' }, t.codigo_error), (t.advertencias || []).map((a) => h('span', { class: 'chip' }, a)), !t.codigo_error && !(t.advertencias || []).length && '—'),
        h('td', { class: 'numeros' }, duracion(t.duracion_ms)))))));
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
        h('button', { class: 'boton primario', type: 'button', onclick: () => asistenteImpresora(vista) }, 'Agregar la primera')));
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
        dialogo({ titulo: 'Ticket de diagnóstico', ancho: 460, cuerpo: h('div', { class: 'pila' },
          h('p', { class: 'tenue pequeno' }, 'Así se ve el ticket que se acaba de imprimir. Compara la regla de columnas y los acentos con el papel.'),
          h('pre', { class: 'papel', tabindex: '0', 'aria-label': 'Vista del ticket' }, r.vista)) });
      } catch (e) { mostrar(`${e.codigo ? e.codigo + ': ' : ''}${e.message} ${e.sugerencia || ''}`, 'error'); }
    } }, 'Ticket de diagnóstico');
    const bPaginas = h('button', { class: 'boton chico', type: 'button', onclick: () => probarPaginas(imp, bPaginas, vista) }, 'Probar páginas de códigos');
    const perfil = `${imp.ancho_mm} mm · ${imp.columnas} columnas · ${imp.pagina_codigos} (ESC t ${imp.comando_pagina_codigos})`;
    const capacidades = [
      imp.soporta_corte ? 'cuchilla' : 'sin cuchilla', imp.soporta_cajon && 'cajón',
      imp.soporta_qr_nativo ? 'QR nativo' : 'QR como imagen', !imp.soporta_imagen && 'sin imágenes',
      imp.confirmar_impresion && 'confirma la impresión',
    ].filter(Boolean).join(' · ');
    return h('article', { class: 'tarjeta' },
      h('div', { class: 'tarjeta-cabeza' },
        h('div', { class: 'impresora-cabeza' }, h('span', { class: 'alias' }, imp.alias), h('span', { class: 'tenue' }, imp.nombre),
          !imp.activa && h('span', { class: 'pastilla sin-punto' }, 'Desactivada')),
        pastilla(ESTADOS_IMPRESORA, imp.activa ? imp.estado : 'desconocido')),
      h('dl', { class: 'datos' },
        h('dt', {}, 'Conexión'), h('dd', {}, `${TIPOS_CONEXION[imp.tipo]} · `, h('span', { class: 'mono' }, imp.destino)),
        h('dt', {}, 'Perfil'), h('dd', {}, perfil),
        h('dt', {}, 'Capacidades'), h('dd', {}, capacidades),
        imp.ultimo_error && [h('dt', {}, 'Último error'), h('dd', {}, h('span', { class: 'chip aviso' }, imp.ultimo_error))]),
      h('div', { class: 'acciones' }, bProbar, bDiag, bPaginas,
        h('button', { class: 'boton chico', type: 'button', onclick: () => formularioImpresora(vista, imp) }, 'Editar'),
        h('button', { class: 'boton chico peligro', type: 'button', onclick: async () => {
          if (!(await confirmar(`¿Eliminar «${imp.alias}»?`, 'Las aplicaciones que imprimen en esta impresora empezarán a recibir IMPRESORA_NO_ENCONTRADA. El historial de trabajos se conserva.', { boton: 'Eliminar', peligro: true }))) return;
          try { await api.borrar(ruta); aviso(`Impresora «${imp.alias}» eliminada.`, 'ok'); recargar(); } catch (e) { avisoError(e); }
        } }, 'Eliminar')),
      resultado);
  }

  async function probarPaginas(imp, boton, vista) {
    let r;
    try { r = await conEspera(boton, () => api.post(`/impresoras/${enc(imp.alias)}/paginas-codigos`)); } catch (e) { avisoError(e); return; }
    const opciones = h('div', { class: 'opciones' }, PAGINAS.map((p) => h('label', { class: 'opcion' },
      h('input', { type: 'radio', name: 'pagina', value: p.pagina, checked: p.pagina === imp.pagina_codigos }),
      h('span', { class: 'titulo-opcion mono' }, `ESC t ${p.comando}`), h('span', { class: 'tenue' }, p.etiqueta))));
    const elegida = await dialogo({
      titulo: 'Elige la página de códigos', ancho: 620,
      cuerpo: h('div', { class: 'pila' },
        h('p', {}, 'Se imprimió la misma línea con cuatro páginas de códigos. Busca en el papel la que muestra bien todos los caracteres y elígela.'),
        h('pre', { class: 'papel', tabindex: '0', 'aria-label': 'Vista del ticket' }, r.vista), opciones),
      acciones: [{ texto: 'Cancelar', valor: null }, { texto: 'Usar la elegida', clase: 'primario', valor: 'elegir' }],
    });
    if (elegida !== 'elegir') return;
    const sel = opciones.querySelector('input:checked');
    if (!sel) return;
    const p = PAGINAS.find((x) => x.pagina === sel.value);
    try {
      await api.put(`/impresoras/${enc(imp.alias)}`, { ...imp, pagina_codigos: p.pagina, comando_pagina_codigos: p.comando });
      aviso(`«${imp.alias}» ahora usa ${p.etiqueta}.`, 'ok'); recargar();
    } catch (e) { avisoError(e); }
  }

  // Alta guiada: 1) cómo está conectada, 2) perfil, 3) probar y guardar.
  function asistenteImpresora(vista) {
    const borrador = { tipo: 'red', destino: '', baudios: 0 };
    const pasos = h('div', { class: 'pasos', 'aria-label': 'Pasos' },
      h('span', { 'aria-current': 'step' }, '1. Conexión'), h('span', {}, '2. Perfil'), h('span', {}, '3. Probar y guardar'));
    const cuerpo = h('div', { class: 'pila' });
    vaciar(vista, cintaDemo(),
      encabezado('Agregar impresora', 'Elige cómo está conectada la impresora a este equipo.',
        h('button', { class: 'boton', type: 'button', onclick: recargar }, 'Cancelar')),
      pasos, cuerpo);

    const elegir = (tipo, destino, baudios = 0) => {
      Object.assign(borrador, { tipo, destino, baudios });
      pasos.children[0].removeAttribute('aria-current'); pasos.children[1].setAttribute('aria-current', 'step');
      formularioImpresora(vista, null, borrador, pasos);
    };
    const filaDetectada = (d) => h('button', { class: 'opcion', type: 'button', onclick: () => elegir(d.tipo, d.destino) },
      h('span', { class: 'fila' }, h('span', { class: 'pastilla sin-punto acento' }, TIPOS_CONEXION[d.tipo]), h('span', { class: 'titulo-opcion mono' }, d.destino)),
      h('span', { class: 'tenue' }, d.detalle || ''));

    const detectadas = h('div', { class: 'opciones' }, h('p', { class: 'tenue' }, 'Buscando…'));
    const redResultados = h('div', { class: 'opciones' });
    const bRed = h('button', { class: 'boton', type: 'button', onclick: async () => {
      try {
        const lista = await conEspera(bRed, () => api.get('/detectar?red=1'));
        vaciar(redResultados, lista.length ? lista.map(filaDetectada) : h('p', { class: 'tenue' }, 'Ningún equipo de la red respondió en el puerto 9100.'));
      } catch (e) { avisoError(e); }
    } }, 'Buscar en la red (puerto 9100)');

    const tipoManual = h('select', { id: 'tipo-manual' }, Object.entries(TIPOS_CONEXION).map(([v, t]) => h('option', { value: v }, t)));
    const destinoManual = h('input', { type: 'text', id: 'destino-manual', class: 'mono', placeholder: '192.168.1.50:9100' });
    const ejemplos = { red: '192.168.1.50:9100', usb: 'Nombre en Windows, /dev/usb/lp0 o cola de CUPS', bluetooth: 'COM5, /dev/rfcomm0 o /dev/cu.Impresora', archivo: 'Carpeta, p. ej. C:\\tickets' };
    tipoManual.addEventListener('change', () => { destinoManual.placeholder = ejemplos[tipoManual.value]; });

    agregar(cuerpo,
      h('section', { class: 'tarjeta' }, h('h2', {}, 'Detectadas en este equipo'),
        h('p', { class: 'tenue pequeno' }, 'Impresoras instaladas (USB) y puertos serie (Bluetooth emparejado).'), detectadas),
      h('section', { class: 'tarjeta' }, h('h2', {}, 'Impresoras de red'),
        h('p', { class: 'tenue pequeno' }, 'Revisa las direcciones de la red local que aceptan conexiones en el puerto 9100. Tarda hasta 15 segundos.'),
        h('div', {}, bRed), redResultados),
      h('section', { class: 'tarjeta' }, h('h2', {}, 'Escribir la dirección'),
        h('div', { class: 'campos' },
          h('label', { class: 'campo', for: 'tipo-manual' }, h('span', {}, 'Tipo de conexión'), tipoManual),
          h('label', { class: 'campo', for: 'destino-manual' }, h('span', {}, 'Destino'), destinoManual)),
        h('div', {}, h('button', { class: 'boton primario', type: 'button', onclick: () => {
          if (!destinoManual.value.trim()) { destinoManual.focus(); aviso('Escribe el destino de la impresora.', 'error'); return; }
          elegir(tipoManual.value, destinoManual.value.trim());
        } }, 'Continuar'))));

    api.get('/detectar').then((lista) => {
      vaciar(detectadas, lista.length ? lista.map(filaDetectada) : h('p', { class: 'tenue' }, 'No se encontraron impresoras USB ni puertos serie.'));
    }).catch((e) => vaciar(detectadas, h('p', { class: 'nota error' }, e.message)));
  }

  // Formulario de perfil: alta (después del asistente) o edición.
  function formularioImpresora(vista, existente, borrador, pasos) {
    const i = existente ? { ...existente } : {
      alias: '', nombre: '', tipo: borrador.tipo, destino: borrador.destino, ancho_mm: 80, ...PERFILES[80],
      pagina_codigos: 'CP858', comando_pagina_codigos: 19, soporta_corte: true, soporta_cajon: false, soporta_qr_nativo: true,
      soporta_codigo_barras_nativo: true, soporta_imagen: true, baudios: borrador.baudios || 0, activa: true,
      confirmar_impresion: false,
    };
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
    const actualizarTipo = () => {
      filaBaudios.hidden = tipo.value !== 'bluetooth';
      filaConfirmar.hidden = tipo.value !== 'red' && tipo.value !== 'bluetooth';
    };
    tipo.addEventListener('change', actualizarTipo);
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
    const bGuardar = h('button', { class: 'boton primario', type: 'submit' }, existente ? 'Guardar cambios' : 'Guardar impresora');
    const form = h('form', { class: 'pila', onsubmit: async (ev) => {
      ev.preventDefault();
      if (!alias.checkValidity()) { aviso('El alias solo admite minúsculas, dígitos, - y _ (hasta 32 caracteres).', 'error'); alias.focus(); return; }
      const d = leer();
      try {
        await conEspera(bGuardar, () => existente ? api.put(`/impresoras/${enc(existente.alias)}`, d) : api.post('/impresoras', d));
        aviso(existente ? 'Cambios guardados.' : `Impresora «${d.alias}» agregada. Imprime el ticket de diagnóstico para comprobarla.`, 'ok');
        location.hash = '#impresoras'; recargar();
      } catch (e) { avisoError(e); }
    } },
    h('section', { class: 'tarjeta' }, h('h2', {}, 'Identificación'),
      h('div', { class: 'campos' },
        campo('alias', 'Alias', alias, 'Con este nombre la eligen las aplicaciones: minúsculas, dígitos, - o _.'),
        campo('nombre', 'Nombre descriptivo', nombre)),
      h('div', { class: 'campos' }, campo('tipo', 'Tipo de conexión', tipo), campo('destino', 'Destino', destino), filaBaudios)),
    h('section', { class: 'tarjeta' }, h('h2', {}, 'Papel'),
      h('div', { class: 'fila' }, ancho, h('span', { class: 'tenue pequeno' }, '58 mm: 32 columnas · 80 mm: 48 columnas')),
      h('details', { class: 'avanzado' }, h('summary', {}, 'Opciones avanzadas'),
        h('div', {},
          h('div', { class: 'campos' }, campo('pagina', 'Página de códigos', pagina, 'Si los acentos salen mal, usa «Probar páginas de códigos».'), campo('comando', 'Comando ESC t', comando)),
          h('div', { class: 'campos' }, campo('columnas', 'Columnas (fuente A)', columnas), campo('columnas-b', 'Columnas (fuente B)', columnasB), campo('ancho-px', 'Puntos por línea', anchoPx))))),
    h('section', { class: 'tarjeta' }, h('h2', {}, 'Capacidades'),
      h('fieldset', {}, h('legend', { class: 'tenue pequeno' }, 'Lo que no tenga, PRISMA lo resuelve de otra forma o lo omite con un aviso.'),
        caps.corte, caps.cajon, caps.qr, caps.barras, caps.imagen, caps.activa), filaConfirmar),
    resultado,
    h('div', { class: 'acciones' }, bProbar, bGuardar));

    if (pasos) {
      vaciar(vista, cintaDemo(), encabezado('Agregar impresora', 'Revisa el perfil y prueba la conexión antes de guardar.',
        h('button', { class: 'boton', type: 'button', onclick: recargar }, 'Cancelar')), pasos, form);
      bProbar.addEventListener('click', () => { pasos.children[1].removeAttribute('aria-current'); pasos.children[2].setAttribute('aria-current', 'step'); });
    } else {
      vaciar(vista, cintaDemo(), encabezado(`Editar «${i.alias}»`, null, h('button', { class: 'boton', type: 'button', onclick: recargar }, 'Cancelar')), form);
    }
    alias.focus();
  }

  // ---------------------------------------------------------------------------------------------
  // 3. Aplicaciones (clientes)
  // ---------------------------------------------------------------------------------------------
  async function vistaClientes(vista) {
    const [clientes, impresoras] = await Promise.all([api.get('/clientes'), api.get('/impresoras')]);
    agregar(vista,
      encabezado('Aplicaciones', 'Sitios web y sistemas autorizados a imprimir, con su nivel de seguridad y permisos.',
        h('button', { class: 'boton primario', type: 'button', onclick: () => formularioCliente(vista, null, impresoras) }, 'Agregar aplicación')),
      clientes.length === 0
        ? h('div', { class: 'tarjeta vacio' }, h('p', {}, 'Todavía no hay aplicaciones autorizadas.'))
        : h('div', { class: 'lista' }, clientes.map((c) => tarjetaCliente(c, vista, impresoras))));
  }

  function tarjetaCliente(c, vista, impresoras) {
    const ruta = `/clientes/${enc(c.id)}`;
    const llave = c.llave_prefijo ? `${c.llave_prefijo}_…` : null;
    return h('article', { class: 'tarjeta' },
      h('div', { class: 'tarjeta-cabeza' },
        h('div', { class: 'cliente-cabeza' }, h('h2', {}, c.nombre), h('span', { class: 'mono tenue pequeno' }, c.id)),
        h('div', { class: 'fila' }, h('span', { class: 'pastilla sin-punto acento' }, `Nivel ${c.nivel} · ${NIVELES[c.nivel].nombre}`),
          h('span', { class: `pastilla ${c.activo ? 'ok' : ''}` }, c.activo ? 'Activa' : 'Desactivada'))),
      h('dl', { class: 'datos' },
        h('dt', {}, 'Orígenes'), h('dd', { class: 'fila' }, c.origenes.length ? c.origenes.map((o) => h('span', { class: 'chip' }, o)) : h('span', { class: 'tenue' }, 'Ninguno (solo desde servidores)')),
        h('dt', {}, 'Impresoras'), h('dd', {}, c.todas_las_impresoras ? 'Todas' : (c.impresoras.length ? c.impresoras.map((a) => h('span', { class: 'chip' }, a)) : h('span', { class: 'tenue' }, 'Ninguna'))),
        h('dt', {}, 'Llave'), h('dd', {}, llave ? h('span', { class: 'mono' }, llave) : h('span', { class: 'tenue' }, c.nivel === 2 ? 'Revocada' : 'Sin llave'), c.exigir_llave && ' (obligatoria con la firma)'),
        c.nivel === 3 && [h('dt', {}, 'Llave pública'), h('dd', { class: 'mono pequeno' }, c.huella_publica || 'Registrada')],
        h('dt', {}, 'Sensibles'), h('dd', {}, c.operaciones_sensibles.length ? c.operaciones_sensibles.map((o) => h('span', { class: 'chip aviso' }, o)) : h('span', { class: 'tenue' }, 'Ninguna'))),
      h('div', { class: 'acciones' },
        h('button', { class: 'boton chico', type: 'button', onclick: () => codigoIntegracion(c, impresoras) }, 'Código de integración'),
        h('button', { class: 'boton chico', type: 'button', onclick: () => formularioCliente(vista, c, impresoras) }, 'Editar'),
        (c.nivel === 2 || c.llave_prefijo) && h('button', { class: 'boton chico', type: 'button', onclick: async () => {
          if (!(await confirmar('¿Generar una llave nueva?', `La llave actual de «${c.nombre}» deja de funcionar en cuanto confirmes. Tendrás que poner la nueva en la aplicación.`, { boton: 'Generar llave nueva' }))) return;
          try { const r = await api.post(`${ruta}/llave`); await mostrarLlave(c, r.llave, impresoras); recargar(); } catch (e) { avisoError(e); }
        } }, c.llave_prefijo ? 'Rotar llave' : 'Generar llave'),
        c.llave_prefijo && h('button', { class: 'boton chico peligro', type: 'button', onclick: async () => {
          if (!(await confirmar('¿Revocar la llave?', c.nivel === 2 ? `«${c.nombre}» dejará de poder imprimir hasta que generes una llave nueva.` : `«${c.nombre}» seguirá imprimiendo con documentos firmados, sin llave.`, { boton: 'Revocar', peligro: true }))) return;
          try { await api.borrar(`${ruta}/llave`); aviso('Llave revocada.', 'ok'); recargar(); } catch (e) { avisoError(e); }
        } }, 'Revocar llave'),
        h('button', { class: 'boton chico', type: 'button', onclick: async () => {
          try { await api.put(ruta, { ...c, activo: !c.activo }); aviso(c.activo ? 'Aplicación desactivada.' : 'Aplicación activada.', 'ok'); recargar(); } catch (e) { avisoError(e); }
        } }, c.activo ? 'Desactivar' : 'Activar'),
        h('button', { class: 'boton chico peligro', type: 'button', onclick: async () => {
          if (!(await confirmar(`¿Eliminar «${c.nombre}»?`, 'Se borran sus orígenes, su llave y sus permisos. Su historial de trabajos se conserva.', { boton: 'Eliminar', peligro: true }))) return;
          try { await api.borrar(ruta); aviso('Aplicación eliminada.', 'ok'); recargar(); } catch (e) { avisoError(e); }
        } }, 'Eliminar')));
  }

  function formularioCliente(vista, existente, impresoras, prellenado = {}) {
    const c = existente ? { ...existente } : {
      nombre: '', nivel: 2, origenes: [], todas_las_impresoras: true, impresoras: [], operaciones_sensibles: [], exigir_llave: false, activo: true, ...prellenado,
    };
    const nombre = h('input', { type: 'text', id: 'cli-nombre', value: c.nombre, required: true, placeholder: 'Tienda en línea' });
    const niveles = h('div', { class: 'opciones', role: 'radiogroup', 'aria-label': 'Nivel de seguridad' }, [1, 2, 3].map((n) =>
      h('label', { class: 'opcion' }, h('input', { type: 'radio', name: 'nivel', value: n, checked: c.nivel === n }),
        h('span', { class: 'titulo-opcion' }, `Nivel ${n} · ${NIVELES[n].nombre}`, n === 2 ? ' (recomendado)' : ''), h('span', { class: 'tenue' }, NIVELES[n].texto))));

    const origenes = [...c.origenes];
    const chips = h('div', { class: 'fila' });
    const nuevoOrigen = h('input', { type: 'url', id: 'cli-origen', class: 'mono', placeholder: 'https://tienda.example', autocomplete: 'off' });
    const pintarOrigenes = () => vaciar(chips, origenes.length ? origenes.map((o, k) => h('span', { class: 'chip' }, o,
      h('button', { type: 'button', 'aria-label': `Quitar ${o}`, onclick: () => { origenes.splice(k, 1); pintarOrigenes(); } }, '×'))) : h('span', { class: 'tenue pequeno' }, 'Sin orígenes.'));
    const agregarOrigen = () => {
      const v = nuevoOrigen.value.trim();
      let u;
      try { u = new URL(v); } catch { aviso('Escribe un origen completo, por ejemplo https://tienda.example', 'error'); return; }
      if (!/^https?:$/.test(u.protocol) || (u.pathname !== '/' && u.pathname !== '') || u.search || u.hash || v.includes('*')) {
        aviso('Un origen es solo esquema, nombre y puerto: https://tienda.example o http://localhost:5173 (sin rutas ni comodines).', 'error'); return;
      }
      if (!origenes.includes(u.origin)) origenes.push(u.origin);
      nuevoOrigen.value = ''; pintarOrigenes();
    };
    nuevoOrigen.addEventListener('keydown', (ev) => { if (ev.key === 'Enter') { ev.preventDefault(); agregarOrigen(); } });
    pintarOrigenes();

    const todas = h('input', { type: 'checkbox', id: 'cli-todas', checked: c.todas_las_impresoras });
    const listaImpresoras = h('div', { class: 'pila' }, impresoras.map((i) => h('label', { class: 'casilla' },
      h('input', { type: 'checkbox', value: i.alias, checked: c.impresoras.includes(i.alias) }), h('span', {}, h('span', { class: 'mono' }, i.alias), ' ', h('span', { class: 'tenue' }, i.nombre)))));
    const actualizarTodas = () => { listaImpresoras.hidden = todas.checked; };
    todas.addEventListener('change', actualizarTodas); actualizarTodas();

    const sensible = (op, texto) => h('label', { class: 'casilla' }, h('input', { type: 'checkbox', value: op, checked: c.operaciones_sensibles.includes(op) }),
      h('span', {}, h('span', { class: 'mono' }, op), ' — ', texto));
    const sensibles = h('fieldset', {}, h('legend', {}, 'Operaciones sensibles'),
      h('p', { class: 'nota aviso' }, 'Habilítalas solo si la aplicación las necesita: abrir_cajon abre el cajón de dinero y raw envía comandos sin revisar a la impresora.'),
      sensible('abrir_cajon', 'abrir el cajón de dinero'), sensible('raw', 'enviar bytes ESC/POS sin revisar'));

    const llavePublica = h('textarea', { id: 'cli-publica', placeholder: '-----BEGIN PUBLIC KEY-----\n…\n-----END PUBLIC KEY-----\n\no los 32 bytes en base64' });
    const estadoLlave = h('p', { class: 'pequeno', 'aria-live': 'polite' });
    let temporizador;
    llavePublica.addEventListener('input', () => {
      clearTimeout(temporizador);
      temporizador = setTimeout(async () => {
        const v = llavePublica.value.trim();
        if (!v) { estadoLlave.textContent = ''; return; }
        try {
          const r = await api.post('/validar-llave-publica', { llave: v });
          estadoLlave.className = 'pequeno nota ok'; estadoLlave.textContent = `Llave Ed25519 válida (${r.formato}). Huella: ${r.huella}`;
        } catch (e) { estadoLlave.className = 'pequeno nota error'; estadoLlave.textContent = e.message; }
      }, 300);
    });
    const exigir = h('input', { type: 'checkbox', id: 'cli-exigir', checked: c.exigir_llave });
    const bloqueFirma = h('section', { class: 'tarjeta' }, h('h2', {}, 'Llave pública (nivel 3)'),
      h('label', { class: 'campo', for: 'cli-publica' }, h('span', {}, existente ? 'Reemplazar la llave pública (opcional)' : 'Pega la llave pública Ed25519 del backend'), llavePublica,
        h('span', { class: 'ayuda' }, 'Solo la llave pública: la privada nunca sale del servidor de la aplicación. Para pruebas: ', h('code', {}, 'prisma dev llaves'), '.')),
      estadoLlave,
      h('label', { class: 'casilla', for: 'cli-exigir' }, exigir, h('span', {}, 'Exigir también la llave pk_ en cada petición')));
    const nivelElegido = () => +niveles.querySelector('input:checked').value;
    const actualizarNivel = () => { bloqueFirma.hidden = nivelElegido() !== 3; };
    niveles.addEventListener('change', actualizarNivel); actualizarNivel();

    const bGuardar = h('button', { class: 'boton primario', type: 'submit' }, existente ? 'Guardar cambios' : 'Crear aplicación');
    const form = h('form', { class: 'pila', onsubmit: async (ev) => {
      ev.preventDefault();
      if (nuevoOrigen.value.trim()) agregarOrigen();
      const datos = {
        nombre: nombre.value.trim(), nivel: nivelElegido(), origenes, todas_las_impresoras: todas.checked,
        impresoras: [...listaImpresoras.querySelectorAll('input:checked')].map((x) => x.value),
        operaciones_sensibles: [...sensibles.querySelectorAll('input:checked')].map((x) => x.value),
        llave_publica: llavePublica.value.trim() || undefined, exigir_llave: exigir.checked, activo: c.activo,
      };
      if (!datos.nombre) { aviso('Escribe el nombre de la aplicación.', 'error'); nombre.focus(); return; }
      if (datos.nivel === 1 && datos.origenes.length === 0) { aviso('El nivel 1 se identifica solo por el origen: agrega al menos uno.', 'error'); nuevoOrigen.focus(); return; }
      if (datos.nivel === 3 && !(existente && existente.nivel === 3) && !datos.llave_publica) { aviso('El nivel 3 necesita la llave pública.', 'error'); llavePublica.focus(); return; }
      try {
        const r = await conEspera(bGuardar, () => existente ? api.put(`/clientes/${enc(existente.id)}`, datos) : api.post('/clientes', datos));
        if (r.llave) await mostrarLlave(r.cliente, r.llave, impresoras);
        else aviso(existente ? 'Cambios guardados.' : 'Aplicación creada.', 'ok');
        location.hash = '#clientes'; recargar();
      } catch (e) { avisoError(e); }
    } },
    h('section', { class: 'tarjeta' }, h('h2', {}, 'Aplicación'),
      h('label', { class: 'campo', for: 'cli-nombre' }, h('span', {}, 'Nombre'), nombre),
      h('fieldset', {}, h('legend', {}, 'Nivel de seguridad'), niveles)),
    bloqueFirma,
    h('section', { class: 'tarjeta' }, h('h2', {}, 'Orígenes permitidos'),
      h('p', { class: 'tenue pequeno' }, 'La dirección exacta de la página web que imprime: esquema, nombre y puerto. Sin comodines.'),
      h('div', { class: 'fila' }, h('div', { class: 'campo', style: null }, nuevoOrigen), h('button', { class: 'boton', type: 'button', onclick: agregarOrigen }, 'Agregar')),
      chips),
    h('section', { class: 'tarjeta' }, h('h2', {}, 'Permisos'),
      h('fieldset', {}, h('legend', {}, 'Impresoras'), h('label', { class: 'casilla', for: 'cli-todas' }, todas, h('span', {}, 'Todas las impresoras, también las que se agreguen después')), listaImpresoras),
      h('hr', { class: 'separador' }), sensibles),
    h('div', { class: 'acciones' }, h('button', { class: 'boton', type: 'button', onclick: recargar }, 'Cancelar'), bGuardar));

    vaciar(vista, cintaDemo(), encabezado(existente ? `Editar «${c.nombre}»` : 'Agregar aplicación',
      existente ? null : 'Registra un sitio web o sistema que imprimirá en este puente.'), form);
    nombre.focus();
  }

  async function mostrarLlave(cliente, llave, impresoras) {
    const caja = h('div', { class: 'llave-caja' }, h('code', {}, llave), botonCopiar(llave));
    await dialogo({
      titulo: 'Llave de la aplicación', ancho: 640,
      cuerpo: h('div', { class: 'pila' },
        h('p', {}, `Esta es la llave de «${cliente.nombre}». `, h('strong', {}, 'Cópiala ahora: no se volverá a mostrar.'), ' El puente solo guarda una huella de ella.'),
        caja,
        h('p', { class: 'tenue pequeno' }, 'La aplicación la envía en cada petición con el encabezado Authorization: Bearer <llave>.'),
        bloqueCodigo(cliente, impresoras, llave)),
      acciones: [{ texto: 'Ya la copié', clase: 'primario', valor: true }],
    });
  }

  function bloqueCodigo(cliente, impresoras, llave) {
    const url = estado.direccion || location.origin;
    const permitidas = cliente.todas_las_impresoras ? impresoras.map((i) => i.alias) : cliente.impresoras;
    const alias = permitidas[0] || 'caja';
    const valorLlave = llave || (cliente.llave_prefijo ? `${cliente.llave_prefijo}_…` : '');
    const conLlave = cliente.nivel === 2 || (cliente.nivel === 3 && cliente.exigir_llave) || (llave && cliente.nivel !== 1);
    const variantes = {
      SDK: [
        `import { Prisma } from "./prisma.js";`,
        ``,
        `const prisma = new Prisma({`,
        `  url: "${url}",`,
        conLlave && `  llave: "${valorLlave}",`,
        `});`,
        ``,
        `await prisma.imprimir({`,
        `  impresora: "${alias}",`,
        `  operaciones: [`,
        `    { tipo: "texto", valor: "Hola desde ${cliente.nombre.replace(/"/g, '')}", alineacion: "centro" },`,
        `    { tipo: "cortar" },`,
        `  ],`,
        `});`,
      ],
      fetch: [
        `const respuesta = await fetch("${url}/v1/imprimir", {`,
        `  method: "POST",`,
        `  headers: {`,
        `    "Content-Type": "application/json",`,
        conLlave && `    "Authorization": "Bearer ${valorLlave}",`,
        `  },`,
        `  body: JSON.stringify({`,
        `    version: "1",`,
        `    id: crypto.randomUUID(),`,
        `    impresora: "${alias}",`,
        `    operaciones: [{ tipo: "texto", valor: "Hola" }, { tipo: "cortar" }],`,
        `  }),`,
        `});`,
        `const resultado = await respuesta.json();`,
      ],
      curl: cliente.nivel === 3 && !conLlave ? [
        `prisma dev firmar documento.json --llave privada.pem --cliente ${cliente.id} --nuevo-id -o firmado.json`,
        `curl --cacert certificado.pem ${url}/v1/imprimir \\`,
        `  -H "Content-Type: application/json" --data-binary @firmado.json`,
      ] : [
        `curl --cacert certificado.pem ${url}/v1/imprimir \\`,
        conLlave && `  -H "Authorization: Bearer ${valorLlave}" \\`,
        `  -H "Content-Type: application/json" \\`,
        `  --data-binary @documento.json`,
      ],
    };
    const pre = h('pre', { class: 'codigo', tabindex: '0', 'aria-label': 'Código de integración' });
    const bCopiar = h('button', { class: 'boton chico', type: 'button', onclick: () => copiar(pre.textContent, bCopiar) }, 'Copiar código');
    const pestanas = h('div', { class: 'pestanas', role: 'tablist' });
    const elegir = (nombre) => {
      pre.textContent = variantes[nombre].filter((l) => l !== false && l !== null && l !== undefined).join('\n');
      for (const b of pestanas.children) b.setAttribute('aria-selected', String(b.textContent === nombre));
    };
    agregar(pestanas, Object.keys(variantes).map((n) => h('button', { type: 'button', role: 'tab', onclick: () => elegir(n) }, n)));
    elegir('SDK');
    return h('div', { class: 'pila' }, h('div', { class: 'fila-entre' }, h('h3', {}, 'Código de integración'), bCopiar), pestanas, pre,
      h('p', { class: 'tenue pequeno' }, 'El SDK (prisma.js) está en camino; mientras tanto puedes usar fetch. ',
        cliente.nivel === 1 && 'En el nivel 1 la página debe servirse desde un origen aprobado. ',
        cliente.nivel === 3 && 'En el nivel 3 el backend firma cada documento antes de enviarlo (ver docs/integradores/firmar-documentos.md).'));
  }

  function codigoIntegracion(c, impresoras) {
    dialogo({ titulo: `Integrar «${c.nombre}»`, ancho: 680, cuerpo: bloqueCodigo(c, impresoras, null) });
  }

  // ---------------------------------------------------------------------------------------------
  // 4. Solicitudes de origen
  // ---------------------------------------------------------------------------------------------
  async function vistaSolicitudes(vista) {
    const [solicitudes, clientes, impresoras] = await Promise.all([api.get('/solicitudes'), api.get('/clientes'), api.get('/impresoras')]);
    const pendientes = solicitudes.filter((s) => s.estado === 'pendiente');
    const rechazadas = solicitudes.filter((s) => s.estado === 'rechazada');
    estado.pendientes = pendientes.length; pintarNav();
    const tabla = (lista, rechazada) => lista.length === 0
      ? h('div', { class: 'tabla-caja' }, h('p', { class: 'vacio' }, rechazada ? 'No hay orígenes rechazados.' : 'No hay solicitudes pendientes.'))
      : h('div', { class: 'tabla-caja' }, h('table', {},
        h('thead', {}, h('tr', {}, ['Origen', 'Intentos', 'Primera vez', 'Última vez', ''].map((t) => h('th', { scope: 'col' }, t)))),
        h('tbody', {}, lista.map((s) => h('tr', {},
          h('td', { class: 'mono' }, s.origen), h('td', { class: 'numeros' }, fmtNum.format(s.intentos)),
          h('td', { class: 'numeros' }, fechaHora(s.primera_vez)), h('td', { class: 'numeros' }, hace(s.ultima_vez)),
          h('td', {}, h('div', { class: 'acciones' },
            h('button', { class: 'boton chico primario', type: 'button', onclick: () => aprobar(s, clientes, impresoras) }, 'Aprobar'),
            !rechazada && h('button', { class: 'boton chico', type: 'button', onclick: async () => {
              try { await api.post('/solicitudes/rechazar', { origen: s.origen }); aviso(`${s.origen} rechazado. Sus intentos se seguirán contando sin avisarte.`, 'ok'); recargar(); } catch (e) { avisoError(e); }
            } }, 'Rechazar'),
            rechazada && h('button', { class: 'boton chico', type: 'button', onclick: async () => {
              try { await api.post('/solicitudes/olvidar', { origen: s.origen }); aviso('Solicitud eliminada.', 'ok'); recargar(); } catch (e) { avisoError(e); }
            } }, 'Olvidar'))))))));
    agregar(vista,
      encabezado('Solicitudes de origen', 'Páginas web que intentaron imprimir sin estar aprobadas. No imprimen hasta que las apruebes.'),
      // El puente recuerda hasta 200; más de 150 suele indicar que alguien de la red registra orígenes inventados.
      solicitudes.length >= 150 && h('p', { class: 'nota aviso' },
        `Hay ${solicitudes.length} solicitudes guardadas (el máximo es 200). Muchas solicitudes de orígenes desconocidos pueden indicar que un equipo de la red intenta imprimir sin permiso. Al llegar al máximo, cada solicitud nueva reemplaza a la pendiente más antigua; las que no tienen actividad se olvidan solas a los 30 días.`),
      h('section', { class: 'pila' }, h('h2', {}, `Pendientes (${pendientes.length})`), tabla(pendientes, false)),
      h('section', { class: 'pila' }, h('h2', {}, `Rechazadas (${rechazadas.length})`), tabla(rechazadas, true)));
  }

  async function aprobar(s, clientes, impresoras) {
    let host = s.origen;
    try { host = new URL(s.origen).hostname; } catch { /* se usa el origen completo */ }
    const nombre = h('input', { type: 'text', id: 'apr-nombre', value: host });
    const nivel = h('select', { id: 'apr-nivel' }, [1, 2].map((n) => h('option', { value: n, selected: n === 2 }, `Nivel ${n} · ${NIVELES[n].nombre}`)));
    const existente = h('select', { id: 'apr-cliente' }, clientes.map((c) => h('option', { value: c.id }, `${c.nombre} (nivel ${c.nivel})`)));
    const modo = h('div', { class: 'opciones' },
      h('label', { class: 'opcion' }, h('input', { type: 'radio', name: 'modo', value: 'nuevo', checked: true }),
        h('span', { class: 'titulo-opcion' }, 'Crear una aplicación nueva'), h('span', { class: 'tenue' }, 'Con este origen como su página.')),
      clientes.length > 0 && h('label', { class: 'opcion' }, h('input', { type: 'radio', name: 'modo', value: 'existente' }),
        h('span', { class: 'titulo-opcion' }, 'Agregar a una existente'), h('span', { class: 'tenue' }, 'Otra página de una aplicación ya registrada.')));
    const nuevoBloque = h('div', { class: 'campos' },
      h('label', { class: 'campo', for: 'apr-nombre' }, h('span', {}, 'Nombre'), nombre),
      h('label', { class: 'campo', for: 'apr-nivel' }, h('span', {}, 'Nivel'), nivel));
    const existenteBloque = h('label', { class: 'campo', for: 'apr-cliente', hidden: true }, h('span', {}, 'Aplicación'), existente);
    modo.addEventListener('change', () => {
      const v = modo.querySelector('input:checked').value;
      nuevoBloque.hidden = v !== 'nuevo'; existenteBloque.hidden = v !== 'existente';
    });
    let respuesta;
    const ok = await dialogo({
      titulo: 'Aprobar origen', ancho: 600,
      cuerpo: h('div', { class: 'pila' },
        h('p', {}, h('span', { class: 'chip' }, s.origen), ' podrá imprimir con los permisos de la aplicación a la que pertenezca.'),
        modo, nuevoBloque, existenteBloque,
        h('p', { class: 'tenue pequeno' }, 'Una aplicación nueva puede imprimir en todas las impresoras y sin operaciones sensibles. Ajústalo después en Aplicaciones.')),
      acciones: [{ texto: 'Cancelar', valor: false }, { texto: 'Aprobar', clase: 'primario', valor: true, antes: async () => {
        const v = modo.querySelector('input:checked').value;
        const cuerpo = v === 'nuevo' ? { origen: s.origen, nuevo: { nombre: nombre.value.trim() || host, nivel: +nivel.value } } : { origen: s.origen, cliente_id: existente.value };
        respuesta = await api.post('/solicitudes/aprobar', cuerpo);
      } }],
    });
    if (!ok) return;
    aviso(`${s.origen} aprobado.`, 'ok');
    if (respuesta && respuesta.llave) await mostrarLlave(respuesta.cliente, respuesta.llave, impresoras);
    await refrescarPendientes();
    recargar();
  }

  // ---------------------------------------------------------------------------------------------
  // 5. Trabajos
  // ---------------------------------------------------------------------------------------------
  async function vistaTrabajos(vista) {
    const [clientes, impresoras] = await Promise.all([api.get('/clientes'), api.get('/impresoras')]);
    const sel = (id, etiqueta, opciones) => h('label', { class: 'campo', for: id }, h('span', {}, etiqueta),
      h('select', { id }, h('option', { value: '' }, 'Todos'), opciones.map(([v, t]) => h('option', { value: v }, t))));
    const filtros = {
      cliente: sel('f-cliente', 'Aplicación', clientes.map((c) => [c.id, c.nombre]).concat([['cli_consola', 'Consola (CLI)'], ['cli_panel', 'Panel']])),
      impresora: sel('f-impresora', 'Impresora', impresoras.map((i) => [i.alias, i.alias])),
      estado: sel('f-estado', 'Estado', Object.entries(ESTADOS_TRABAJO).map(([v, [, t]]) => [v, t])),
      desde: h('label', { class: 'campo', for: 'f-desde' }, h('span', {}, 'Desde'), h('input', { type: 'date', id: 'f-desde' })),
      hasta: h('label', { class: 'campo', for: 'f-hasta' }, h('span', {}, 'Hasta'), h('input', { type: 'date', id: 'f-hasta' })),
    };
    const resultados = h('div', {});
    const cargar = async () => {
      const q = new URLSearchParams();
      for (const [k, el] of Object.entries(filtros)) { const v = el.querySelector('select, input').value; if (v) q.set(k, v); }
      try { vaciar(resultados, tablaTrabajos(await api.get(`/trabajos?${q}`))); } catch (e) { vaciar(resultados, h('p', { class: 'nota error' }, e.message)); }
    };
    const form = h('form', { class: 'tarjeta', onchange: cargar, onsubmit: (ev) => ev.preventDefault() },
      h('div', { class: 'campos' }, Object.values(filtros)));
    agregar(vista,
      encabezado('Trabajos', 'Bitácora de impresión. Por privacidad no guarda el contenido de los tickets.'),
      form, resultados);
    await cargar();
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
      h('p', { class: 'tenue pequeno' }, 'Cómo instalarlo en cada dispositivo: ', h('a', { href: '#ayuda' }, 'Ayuda'), '.'));

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
          h('li', {}, 'Agrega la aplicación en ', h('a', { href: '#clientes' }, 'Aplicaciones'), ' y elige su nivel de seguridad.'),
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
    inicio: vistaInicio, impresoras: vistaImpresoras, clientes: vistaClientes, solicitudes: vistaSolicitudes,
    trabajos: vistaTrabajos, recursos: vistaRecursos, ajustes: vistaAjustes, ayuda: vistaAyuda,
  };

  arrancar();
})();
