// Datos de ejemplo para el prototipo del panel. Imita la API interna /admin/api/* en memoria:
// el panel real usa exactamente las mismas rutas y formas de respuesta. No se embebe en el binario.
'use strict';
(() => {
  const ahora = Date.now();
  const min = 60e3, hora = 60 * min, dia = 24 * hora;
  const iso = (t) => new Date(t).toISOString();
  const espera = (ms = 250) => new Promise((r) => setTimeout(r, ms));
  class ErrorDemo extends Error {
    constructor(codigo, mensaje, sugerencia) { super(mensaje); this.codigo = codigo; this.sugerencia = sugerencia; }
  }

  let pantalla = 'panel';
  const impresoras = [
    { alias: 'caja', nombre: 'Caja principal', tipo: 'red', destino: '192.168.1.50:9100', ancho_mm: 80, columnas: 48, columnas_fuente_b: 64, ancho_px: 576,
      pagina_codigos: 'CP858', comando_pagina_codigos: 19, soporta_qr_nativo: true, soporta_corte: true, soporta_cajon: true, soporta_imagen: true,
      soporta_codigo_barras_nativo: true, baudios: 0, activa: true, estado: 'en_linea', pendientes: 0 },
    { alias: 'cocina', nombre: 'Cocina', tipo: 'usb', destino: 'POS-80C', ancho_mm: 80, columnas: 48, columnas_fuente_b: 64, ancho_px: 576,
      pagina_codigos: 'CP850', comando_pagina_codigos: 2, soporta_qr_nativo: false, soporta_corte: true, soporta_cajon: false, soporta_imagen: true,
      soporta_codigo_barras_nativo: true, baudios: 0, activa: true, estado: 'ocupada', pendientes: 2 },
    { alias: 'barra', nombre: 'Barra de bebidas', tipo: 'bluetooth', destino: 'COM5', ancho_mm: 58, columnas: 32, columnas_fuente_b: 42, ancho_px: 384,
      pagina_codigos: 'CP858', comando_pagina_codigos: 19, soporta_qr_nativo: true, soporta_corte: false, soporta_cajon: false, soporta_imagen: true,
      soporta_codigo_barras_nativo: true, baudios: 9600, activa: true, estado: 'fuera_de_linea', ultimo_error: 'IMPRESORA_INALCANZABLE', pendientes: 0 },
  ];
  const clientes = [
    { id: 'cli_znkoec7rsug3dioh', nombre: 'Punto de venta web', nivel: 2, llave_prefijo: 'pk_7ax6nj4r4a', exigir_llave: false,
      origenes: ['https://pos.laesperanza.mx'], todas_las_impresoras: false, impresoras: ['caja'], operaciones_sensibles: ['abrir_cajon'], activo: true },
    { id: 'cli_kc94hlbswbrknuza', nombre: 'Pedidos en línea', nivel: 3, llave_prefijo: null, exigir_llave: false,
      origenes: [], todas_las_impresoras: false, impresoras: ['cocina', 'barra'], operaciones_sensibles: [], activo: true,
      huella_publica: 'SHA-256 4F:9A:1C:77:E2:0B:…:D3' },
    { id: 'cli_m3nu8d1g1tal0x2q', nombre: 'Menú digital', nivel: 1, llave_prefijo: null, exigir_llave: false,
      origenes: ['https://menu.laesperanza.mx'], todas_las_impresoras: false, impresoras: ['cocina'], operaciones_sensibles: [], activo: true },
  ];
  const solicitudes = [
    { origen: 'http://192.168.1.77:5173', primera_vez: iso(ahora - 2 * hora), ultima_vez: iso(ahora - 4 * min), intentos: 12, estado: 'pendiente' },
    { origen: 'https://inventario.laesperanza.mx', primera_vez: iso(ahora - 3 * dia), ultima_vez: iso(ahora - 6 * hora), intentos: 4, estado: 'pendiente' },
    { origen: 'https://ofertas-imprime.example', primera_vez: iso(ahora - 9 * dia), ultima_vez: iso(ahora - 1 * dia), intentos: 37, estado: 'rechazada' },
  ];
  const nombreCliente = (id) => (clientes.find((c) => c.id === id) || {}).nombre || (id === 'cli_consola' ? 'Consola (CLI)' : id === 'cli_panel' ? 'Panel' : id);
  const trabajos = [];
  (() => {
    const plan = [
      ['cli_znkoec7rsug3dioh', 'caja', 'completado', '', [], 380], ['cli_kc94hlbswbrknuza', 'cocina', 'completado', '', [], 920],
      ['cli_m3nu8d1g1tal0x2q', 'cocina', 'completado', '', ['CARACTER_SUSTITUIDO'], 870], ['cli_kc94hlbswbrknuza', 'barra', 'fallido', 'IMPRESORA_INALCANZABLE', [], 3010],
      ['cli_znkoec7rsug3dioh', 'caja', 'completado', '', [], 410], ['cli_kc94hlbswbrknuza', 'cocina', 'imprimiendo', '', [], null],
      ['cli_kc94hlbswbrknuza', 'cocina', 'encolado', '', [], null], ['cli_znkoec7rsug3dioh', 'caja', 'completado', '', [], 395],
      ['cli_kc94hlbswbrknuza', 'barra', 'completado', '', ['SIN_CUCHILLA'], 1840], ['cli_consola', 'caja', 'completado', '', [], 360],
    ];
    let t = ahora - 90 * 1000;
    for (let k = 0; k < 34; k++) {
      const [cliente, alias, est, err, adv, dur] = plan[k % plan.length];
      const estadoFinal = k < 2 ? ['encolado', 'imprimiendo'][k] : (est === 'encolado' || est === 'imprimiendo' ? 'completado' : est);
      trabajos.push({ id: `trb_${(0x9a3f00 + k * 7919).toString(16)}e2c41`, documento_id: '', cliente_id: cliente, cliente_nombre: nombreCliente(cliente),
        impresora_alias: k < 2 ? 'cocina' : alias, referencia: cliente === 'cli_kc94hlbswbrknuza' ? `PED-${(812 - k).toString().padStart(6, '0')}` : (cliente === 'cli_znkoec7rsug3dioh' ? `VTA-${4127 - k}` : ''),
        estado: estadoFinal, codigo_error: estadoFinal === 'fallido' ? err : '', advertencias: adv, duracion_ms: estadoFinal === 'completado' || estadoFinal === 'fallido' ? dur : null,
        creado: iso(t) });
      t -= (k % 3 === 0 ? 47 : 13) * min;
    }
  })();

  // --- Imágenes de ejemplo y conversión a 1 bit (Floyd–Steinberg), como hace el puente ---------
  function lienzo(w, hgt) { const c = document.createElement('canvas'); c.width = w; c.height = hgt; return c; }
  function dibujarLogo() {
    const c = lienzo(480, 200), g = c.getContext('2d');
    g.fillStyle = '#fff'; g.fillRect(0, 0, 480, 200);
    const grad = g.createLinearGradient(0, 0, 480, 0); grad.addColorStop(0, '#222'); grad.addColorStop(1, '#888');
    g.fillStyle = grad; g.beginPath(); g.moveTo(70, 20); g.lineTo(130, 170); g.lineTo(10, 170); g.closePath(); g.fill();
    g.fillStyle = '#111'; g.font = 'bold 54px sans-serif'; g.fillText('LA ESPERANZA', 150, 105);
    g.fillStyle = '#555'; g.font = '26px sans-serif'; g.fillText('Abarrotes desde 1987', 152, 150);
    return c;
  }
  function dibujarQR() {
    const c = lienzo(200, 200), g = c.getContext('2d');
    g.fillStyle = '#fff'; g.fillRect(0, 0, 200, 200); g.fillStyle = '#000';
    let semilla = 7;
    const azar = () => (semilla = (semilla * 16807) % 2147483647) / 2147483647;
    for (let y = 0; y < 25; y++) for (let x = 0; x < 25; x++) if (azar() > 0.55) g.fillRect(x * 8, y * 8, 8, 8);
    for (const [x, y] of [[0, 0], [18, 0], [0, 18]]) {
      g.fillStyle = '#000'; g.fillRect(x * 8, y * 8, 56, 56); g.fillStyle = '#fff'; g.fillRect(x * 8 + 8, y * 8 + 8, 40, 40); g.fillStyle = '#000'; g.fillRect(x * 8 + 16, y * 8 + 16, 24, 24);
    }
    return c;
  }
  function a1Bit(origen, anchoMax) {
    const w = Math.min(origen.width, anchoMax), hgt = Math.max(1, Math.round(origen.height * w / origen.width));
    const c = lienzo(w, hgt), g = c.getContext('2d');
    g.fillStyle = '#fff'; g.fillRect(0, 0, w, hgt); g.drawImage(origen, 0, 0, w, hgt);
    const img = g.getImageData(0, 0, w, hgt), d = img.data, gris = new Float32Array(w * hgt);
    for (let i = 0; i < w * hgt; i++) gris[i] = (0.299 * d[i * 4] + 0.587 * d[i * 4 + 1] + 0.114 * d[i * 4 + 2]) / 255;
    for (let y = 0; y < hgt; y++) for (let x = 0; x < w; x++) {
      const i = y * w + x, viejo = gris[i], nuevo = viejo < 0.5 ? 0 : 1, e = viejo - nuevo;
      gris[i] = nuevo;
      if (x + 1 < w) gris[i + 1] += e * 7 / 16;
      if (y + 1 < hgt) { if (x > 0) gris[i + w - 1] += e * 3 / 16; gris[i + w] += e * 5 / 16; if (x + 1 < w) gris[i + w + 1] += e / 16; }
    }
    for (let i = 0; i < w * hgt; i++) { const v = gris[i] ? 255 : 0; d[i * 4] = d[i * 4 + 1] = d[i * 4 + 2] = v; d[i * 4 + 3] = 255; }
    g.putImageData(img, 0, 0);
    return c;
  }
  const recursos = [
    { nombre: 'logo-tienda', ancho_px: 480, alto_px: 200, bytes: 18432, lienzo: dibujarLogo },
    { nombre: 'qr-encuesta', ancho_px: 200, alto_px: 200, bytes: 2210, lienzo: dibujarQR },
  ];
  const cacheImagen = {};
  function imagenRecurso(nombre) {
    const r = recursos.find((x) => x.nombre === nombre);
    if (!r) return '';
    return cacheImagen[nombre] || (cacheImagen[nombre] = a1Bit(r.lienzo(), 576).toDataURL('image/png'));
  }

  // --- Tickets simulados (el puente real los genera con su impresora virtual) -------------------
  function marco(cols, lineas) {
    const fila = (t = '', al = 'izq') => {
      t = [...t].slice(0, cols).join('');
      const falta = cols - [...t].length;
      const izq = al === 'centro' ? Math.floor(falta / 2) : al === 'der' ? falta : 0;
      return '│' + ' '.repeat(izq) + t + ' '.repeat(falta - izq) + '│';
    };
    return ['┌' + '─'.repeat(cols) + '┐', ...lineas.map((l) => (Array.isArray(l) ? fila(...l) : fila(l))), '└✂ - - - corte ' + '─'.repeat(Math.max(0, cols - 14)) + '┘'].join('\n');
  }
  function ticketDiagnostico(imp) {
    const c = imp.columnas;
    const regla = Array.from({ length: c }, (_, k) => ((k + 1) % 10 === 0 ? String(((k + 1) / 10) % 10) : '·')).join('');
    return marco(c, [
      ['P R U E B A   D E   I M P R E S O R A', 'centro'], ['', ''],
      `Alias:     ${imp.alias}`, `Nombre:    ${imp.nombre}`, `Conexión:  ${imp.tipo} ${imp.destino}`,
      `Papel:     ${imp.ancho_mm} mm, ${c} columnas`, `Página:    ${imp.pagina_codigos} (ESC t ${imp.comando_pagina_codigos})`,
      `Fecha:     ${new Date().toLocaleString('es')}`, '',
      'Regla de columnas (debe ocupar una línea):', regla, '',
      'Acentos: áéíóú ÁÉÍÓÚ ñÑ üÜ ¿? ¡! € °', '',
      ['Izquierda', 'izq'], ['Centro', 'centro'], ['Derecha', 'der'], '',
      imp.soporta_qr_nativo ? ['[ código QR nativo ]', 'centro'] : ['[ código QR como imagen ]', 'centro'],
      ['PRISMA · puente de impresión', 'centro'], '',
    ]);
  }
  function ticketPaginas(imp) {
    const linea = 'áéíóú ñÑ ¿¡ € °';
    return marco(imp.columnas, [
      ['PRUEBA DE PÁGINAS DE CÓDIGOS', 'centro'], '',
      'Elige la línea que se lee bien:', '',
      `ESC t 0  PC437    ${linea}`, `ESC t 2  PC850    ${linea}`, `ESC t 16 WPC1252  ${linea}`, `ESC t 19 PC858    ${linea}`, '',
    ]);
  }

  // --- Rutas ----------------------------------------------------------------------------------
  const llaveNueva = () => {
    const az = 'abcdefghijklmnopqrstuvwxyz234567', b64 = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_';
    const prefijo = 'pk_' + Array.from({ length: 10 }, () => az[Math.floor(Math.random() * 32)]).join('');
    return { prefijo, llave: prefijo + '_' + Array.from({ length: 43 }, () => b64[Math.floor(Math.random() * 64)]).join('') };
  };
  const idCliente = () => 'cli_' + Array.from({ length: 16 }, () => 'abcdefghijklmnopqrstuvwxyz0123456789'[Math.floor(Math.random() * 36)]).join('');
  const ajustes = { nombre_puente: 'Abarrotes La Esperanza', limite_ip: 120, limite_cliente: 60, retencion_dias: 30, solo_red_privada: true, buscar_actualizaciones: true };
  const tls = {
    modo: 'autofirmado',
    certificado: { vence: '17 ene 2029', nombres: ['localhost', 'caja-pc', 'caja-pc.local'], ips: ['127.0.0.1', '::1', '192.168.1.10'], ips_sin_cubrir: [],
      huella: '86:D5:2D:B3:E6:F2:DF:D6:2C:4A:17:8F:25:D6:00:48:F0:94:01:D3:4B:83:41:4D:8C:D9:E9:5B:E5:71:25:16' },
  };
  const direccion = 'https://192.168.1.10:8443';
  const buscarImpresora = (alias) => {
    const i = impresoras.find((x) => x.alias === alias);
    if (!i) throw new ErrorDemo('IMPRESORA_NO_ENCONTRADA', `No existe la impresora «${alias}».`, '');
    return i;
  };

  async function pedir(metodo, ruta, cuerpo) {
    await espera();
    const [camino, consulta] = ruta.split('?');
    const p = camino.split('/').filter(Boolean).map(decodeURIComponent);
    const q = new URLSearchParams(consulta || '');
    const r = `${metodo} /${p[0] || ''}`;

    if (r === 'GET /sesion') return { autenticado: pantalla !== 'entrar' && pantalla !== 'configuracion', configurado: pantalla !== 'configuracion', local: false, debe_cambiar: pantalla === 'cambio', csrf: 'demo' };
    if (r === 'POST /entrar') {
      if (!cuerpo.contrasena) throw new ErrorDemo('CONTRASENA_INCORRECTA', 'La contraseña no es correcta.', 'Te quedan 4 intentos antes de un bloqueo de 5 minutos.');
      pantalla = 'panel'; return { ok: true, csrf: 'demo' };
    }
    if (r === 'POST /configuracion-inicial' || r === 'POST /contrasena') {
      if (r === 'POST /configuracion-inicial' && cuerpo.codigo !== undefined && !/^\d{3}-?\d{3}$/.test(cuerpo.codigo || '')) {
        throw new ErrorDemo('CODIGO_INVALIDO', 'El código de configuración no es válido.', 'En el prototipo cualquier código de 8 caracteres funciona, por ejemplo K7QM-4XWP.');
      }
      pantalla = 'panel'; return { ok: true, csrf: 'demo' };
    }
    if (r === 'POST /salir') { pantalla = 'entrar'; return { ok: true }; }
    if (r === 'GET /resumen') {
      const hoy = trabajos.filter((t) => Date.now() - new Date(t.creado) < dia);
      return {
        puente: { nombre: ajustes.nombre_puente, version: '0.6.0', direccion, modo_tls: tls.modo, iniciado: iso(ahora - 3 * dia - 2 * hora) },
        impresoras: impresoras.map(({ alias, nombre, estado, pendientes }) => ({ alias, nombre, estado, pendientes })),
        actualizacion: ajustes.buscar_actualizaciones ? { version: '0.7.0', url: 'https://github.com/bayalad7/prisma/releases/tag/v0.7.0', fecha: iso(ahora - 2 * dia) } : null,
        solicitudes_pendientes: solicitudes.filter((s) => s.estado === 'pendiente').length,
        trabajos_hoy: { completados: hoy.filter((t) => t.estado === 'completado').length, fallidos: hoy.filter((t) => t.estado === 'fallido').length },
        ultimos_trabajos: trabajos.slice(0, 8),
      };
    }

    if (p[0] === 'detectar') {
      await espera(q.get('red') ? 1600 : 500);
      return q.get('red')
        ? [{ tipo: 'red', destino: '192.168.1.50:9100', detalle: 'responde en el puerto 9100' }, { tipo: 'red', destino: '192.168.1.61:9100', detalle: 'responde en el puerto 9100' }]
        : [{ tipo: 'usb', destino: 'POS-80C', detalle: 'puerto USB001, controlador POS-80C' }, { tipo: 'usb', destino: 'EPSON TM-T20II Receipt', detalle: 'puerto USB002, controlador EPSON TM-T20II' },
          { tipo: 'bluetooth', destino: 'COM5', detalle: 'puerto serie (Bluetooth SPP o serial)' }];
    }
    if (p[0] === 'impresoras') {
      if (metodo === 'GET') return impresoras;
      if (metodo === 'POST' && p[1] === 'probar-destino') {
        await espera(700);
        if (/^192\.168\.1\.6/.test(cuerpo.destino)) throw new ErrorDemo('IMPRESORA_INALCANZABLE', `La impresora en ${cuerpo.destino} rechazó la conexión.`, 'Revisa que esté encendida y que el puerto sea 9100.');
        return { ok: true, mensaje: 'La impresora respondió: está lista para imprimir.' };
      }
      if (metodo === 'POST' && !p[1]) {
        if (impresoras.some((i) => i.alias === cuerpo.alias)) throw new ErrorDemo('ALIAS_DUPLICADO', `Ya existe una impresora con el alias «${cuerpo.alias}».`, 'Elige otro alias.');
        const nueva = { ...cuerpo, estado: 'en_linea', pendientes: 0 }; impresoras.push(nueva); return nueva;
      }
      const imp = buscarImpresora(p[1]);
      if (metodo === 'PUT') { Object.assign(imp, cuerpo); return imp; }
      if (metodo === 'DELETE') { impresoras.splice(impresoras.indexOf(imp), 1); return { ok: true }; }
      if (p[2] === 'probar') {
        await espera(600);
        if (imp.estado === 'fuera_de_linea') throw new ErrorDemo('IMPRESORA_INALCANZABLE', `No se pudo comunicar con la impresora en ${imp.destino}.`, 'Revisa que esté encendida, emparejada y dentro del alcance del Bluetooth.');
        return { ok: true, mensaje: 'La impresora respondió: está en línea.' };
      }
      if (p[2] === 'diagnostico' || p[2] === 'paginas-codigos') {
        await espera(900);
        if (imp.estado === 'fuera_de_linea') throw new ErrorDemo('IMPRESORA_INALCANZABLE', `No se pudo comunicar con la impresora en ${imp.destino}.`, 'Revisa que esté encendida y emparejada.');
        const t = { id: `trb_${Math.random().toString(16).slice(2, 14)}`, cliente_id: 'cli_panel', cliente_nombre: 'Panel', impresora_alias: imp.alias, referencia: p[2] === 'diagnostico' ? 'DIAGNOSTICO' : 'PAGINAS', estado: 'completado', codigo_error: '', advertencias: [], duracion_ms: 640, creado: iso(Date.now()) };
        trabajos.unshift(t);
        return { ok: true, trabajo_id: t.id, vista: p[2] === 'diagnostico' ? ticketDiagnostico(imp) : ticketPaginas(imp) };
      }
    }
    if (p[0] === 'validar-llave-publica') {
      const v = cuerpo.llave.trim();
      if (/^-----BEGIN PUBLIC KEY-----[\s\S]+-----END PUBLIC KEY-----$/.test(v)) return { ok: true, formato: 'PEM', huella: 'SHA-256 B1:0E:52:…:9C' };
      try { if (atob(v).length === 32) return { ok: true, formato: 'base64', huella: 'SHA-256 B1:0E:52:…:9C' }; } catch { /* no es base64 */ }
      throw new ErrorDemo('LLAVE_PUBLICA_INVALIDA', 'No es una llave pública Ed25519: pega el bloque PEM completo o los 32 bytes en base64.', '');
    }
    if (p[0] === 'clientes') {
      if (metodo === 'GET') return clientes;
      if (metodo === 'POST' && !p[1]) {
        const c = { id: idCliente(), llave_prefijo: null, huella_publica: cuerpo.llave_publica ? 'SHA-256 B1:0E:52:…:9C' : undefined, ...cuerpo };
        delete c.llave_publica;
        let llave;
        if (c.nivel === 2 || c.exigir_llave) { const k = llaveNueva(); c.llave_prefijo = k.prefijo; llave = k.llave; }
        clientes.push(c);
        return { cliente: c, llave };
      }
      const c = clientes.find((x) => x.id === p[1]);
      if (!c) throw new ErrorDemo('CLIENTE_NO_ENCONTRADO', 'La aplicación ya no existe.', '');
      if (p[2] === 'llave' && metodo === 'POST') { const k = llaveNueva(); c.llave_prefijo = k.prefijo; return { llave: k.llave }; }
      if (p[2] === 'llave' && metodo === 'DELETE') { c.llave_prefijo = null; return { ok: true }; }
      if (metodo === 'PUT') { const { llave_publica, ...resto } = cuerpo; Object.assign(c, resto); if (llave_publica) c.huella_publica = 'SHA-256 B1:0E:52:…:9C'; return { cliente: c }; }
      if (metodo === 'DELETE') { clientes.splice(clientes.indexOf(c), 1); return { ok: true }; }
    }
    if (p[0] === 'solicitudes') {
      if (metodo === 'GET') return solicitudes;
      const s = solicitudes.find((x) => x.origen === cuerpo.origen);
      if (p[1] === 'rechazar') { s.estado = 'rechazada'; return { ok: true }; }
      if (p[1] === 'olvidar') { solicitudes.splice(solicitudes.indexOf(s), 1); return { ok: true }; }
      if (p[1] === 'aprobar') {
        solicitudes.splice(solicitudes.indexOf(s), 1);
        if (cuerpo.cliente_id) { const c = clientes.find((x) => x.id === cuerpo.cliente_id); c.origenes.push(cuerpo.origen); return { cliente: c }; }
        const c = { id: idCliente(), nombre: cuerpo.nuevo.nombre, nivel: cuerpo.nuevo.nivel, llave_prefijo: null, exigir_llave: false, origenes: [cuerpo.origen],
          todas_las_impresoras: true, impresoras: [], operaciones_sensibles: [], activo: true };
        let llave;
        if (c.nivel === 2) { const k = llaveNueva(); c.llave_prefijo = k.prefijo; llave = k.llave; }
        clientes.push(c);
        return { cliente: c, llave };
      }
    }
    if (p[0] === 'trabajos') {
      return trabajos.filter((t) => (!q.get('cliente') || t.cliente_id === q.get('cliente')) && (!q.get('impresora') || t.impresora_alias === q.get('impresora'))
        && (!q.get('estado') || t.estado === q.get('estado')) && (!q.get('desde') || t.creado.slice(0, 10) >= q.get('desde'))
        && (!q.get('hasta') || t.creado.slice(0, 10) <= q.get('hasta')));
    }
    if (p[0] === 'recursos') {
      if (metodo === 'GET') return recursos.map(({ nombre, ancho_px, alto_px, bytes }) => ({ nombre, ancho_px, alto_px, bytes }));
      if (p[1] === 'vista-previa') {
        const img = await createImageBitmap(cuerpo.get('archivo'));
        const c = lienzo(img.width, img.height); c.getContext('2d').drawImage(img, 0, 0);
        return new Promise((res) => a1Bit(c, +cuerpo.get('ancho')).toBlob(res, 'image/png'));
      }
      if (metodo === 'POST') {
        const archivo = cuerpo.get('archivo'), nombre = cuerpo.get('nombre');
        const img = await createImageBitmap(archivo);
        const original = lienzo(img.width, img.height); original.getContext('2d').drawImage(img, 0, 0);
        const i = recursos.findIndex((x) => x.nombre === nombre); if (i >= 0) recursos.splice(i, 1);
        delete cacheImagen[nombre];
        recursos.push({ nombre, ancho_px: img.width, alto_px: img.height, bytes: archivo.size, lienzo: () => original });
        return { ok: true };
      }
      if (metodo === 'DELETE') { recursos.splice(recursos.findIndex((x) => x.nombre === p[1]), 1); return { ok: true }; }
    }
    if (p[0] === 'ajustes') {
      if (metodo === 'GET') return { ...ajustes, tls, direccion };
      Object.assign(ajustes, cuerpo); return { ok: true };
    }
    if (r === 'POST /tls') { await espera(700); return { ok: true }; }
    if (r === 'POST /restaurar') { await espera(900); pantalla = 'entrar'; return { ok: true }; }
    throw new ErrorDemo('RUTA_NO_ENCONTRADA', `El prototipo no simula ${metodo} ${ruta}.`, '');
  }

  window.PRISMA_DEMO = {
    pedir,
    verPantalla: (p) => { pantalla = p; },
    url: (ruta) => {
      const m = ruta.match(/^\/recursos\/([^/]+)\/imagen/);
      return m ? imagenRecurso(decodeURIComponent(m[1])) : '#';
    },
  };
})();
