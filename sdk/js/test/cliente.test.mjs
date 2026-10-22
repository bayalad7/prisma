// El cliente contra un puente simulado con node:http, y el diagnóstico de red con fetch simulado.
import { test, describe, before, after } from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { Prisma, PrismaError } from '../src/prisma.mjs';

// Puente simulado: cada prueba define cómo responde.
let responder = () => [200, { ok: true }];
const recibidas = [];
let servidor;
let URL_PUENTE;

before(async () => {
  servidor = createServer(async (req, res) => {
    let cuerpo = '';
    for await (const trozo of req) cuerpo += trozo;
    const peticion = { metodo: req.method, ruta: req.url, cabeceras: req.headers, cuerpo: cuerpo ? JSON.parse(cuerpo) : undefined };
    recibidas.push(peticion);
    const [estado, datos, tipo = 'application/json; charset=utf-8'] = await responder(peticion, recibidas.length);
    if (estado === 0) { req.socket.destroy(); return; } // corta la conexión: error de red
    res.writeHead(estado, { 'Content-Type': tipo });
    res.end(typeof datos === 'string' ? datos : JSON.stringify(datos));
  });
  await new Promise((r) => servidor.listen(0, '127.0.0.1', r));
  URL_PUENTE = `http://127.0.0.1:${servidor.address().port}`;
});
after(() => servidor.close());

const nuevo = (op = {}) => new Prisma({ url: URL_PUENTE, llave: 'pk_abcdefghij_xyz', espera_reintento_ms: 5, ...op });
const reiniciar = (f) => { recibidas.length = 0; responder = f; };

describe('peticiones', () => {
  test('salud e impresoras', async () => {
    reiniciar((p) => (p.ruta === '/v1/salud' ? [200, { ok: true, version: '0.6.0', nombre_puente: 'Café' }] : [200, [{ alias: 'caja', estado: 'en_linea' }]]));
    const prisma = nuevo();
    assert.equal((await prisma.salud()).nombre_puente, 'Café');
    assert.deepEqual(await prisma.impresoras(), [{ alias: 'caja', estado: 'en_linea' }]);
    assert.equal(recibidas[1].cabeceras.authorization, 'Bearer pk_abcdefghij_xyz');
  });

  test('imprimir envía un documento v1 válido con un id nuevo', async () => {
    reiniciar(() => [200, { ok: true, trabajo_id: 'trb_1', impresora: 'caja', duracion_ms: 10, advertencias: [] }]);
    const r = await nuevo().imprimir('caja', (d) => d.texto('Hola').cortar(), { referencia: 'VTA-1' });
    assert.equal(r.trabajo_id, 'trb_1');
    const doc = recibidas[0].cuerpo;
    assert.equal(recibidas[0].ruta, '/v1/imprimir');
    assert.equal(recibidas[0].cabeceras['content-type'], 'application/json');
    assert.equal(doc.impresora, 'caja');
    assert.equal(doc.referencia, 'VTA-1');
    assert.deepEqual(doc.operaciones, [{ tipo: 'texto', valor: 'Hola' }, { tipo: 'cortar' }]);
    await nuevo().imprimir('caja', [{ tipo: 'texto', valor: 'Otra' }]);
    assert.notEqual(recibidas[1].cuerpo.id, doc.id, 'cada ticket lleva su propio id');
  });

  test('imprimirDocumento envía el documento firmado sin modificarlo', async () => {
    reiniciar(() => [200, { ok: true, trabajo_id: 'trb_2', impresora: 'caja', advertencias: [] }]);
    const firmado = {
      version: '1', id: '780bf0ea-9a33-4add-86f3-9f4f23000fc4', impresora: 'caja', operaciones: [{ tipo: 'cortar' }],
      cliente_id: 'cli_7Hq2MZp4XwR9', emitido: '2026-10-14T18:30:00Z', expira: '2026-10-14T18:31:30Z', firma: `${'A'.repeat(86)}==`,
    };
    await new Prisma({ url: URL_PUENTE }).imprimirDocumento(firmado);
    assert.deepEqual(recibidas[0].cuerpo, firmado);
    assert.equal(recibidas[0].cabeceras.authorization, undefined, 'sin llave no se envía Authorization');
  });

  test('un documento inválido no sale del navegador', async () => {
    reiniciar(() => [200, { ok: true }]);
    await assert.rejects(nuevo().imprimirDocumento({ version: '1', impresora: 'caja', operaciones: [] }), { codigo: 'DOCUMENTO_INVALIDO', status: 0 });
    await assert.rejects(nuevo().imprimir('caja', (d) => d.texto('x', { tamano: [2] })), { codigo: 'DOCUMENTO_INVALIDO' });
    assert.equal(recibidas.length, 0);
  });

  test('previsualizar devuelve el HTML', async () => {
    reiniciar(() => [200, '<div class="ticket">Hola</div>', 'text/html; charset=utf-8']);
    const html = await nuevo().previsualizar('caja', (d) => d.texto('Hola'));
    assert.equal(html, '<div class="ticket">Hola</div>');
    assert.equal(recibidas[0].ruta, '/v1/previsualizar');
  });

  test('los errores del puente llegan como PrismaError y no se reintentan', async () => {
    for (const [estado, codigo] of [[404, 'IMPRESORA_NO_ENCONTRADA'], [403, 'ORIGEN_NO_AUTORIZADO'], [429, 'DEMASIADAS_PETICIONES'], [503, 'IMPRESORA_INALCANZABLE']]) {
      reiniciar(() => [estado, { ok: false, codigo, mensaje: 'm', sugerencia: 's', trabajo_id: 'trb_9' }]);
      await assert.rejects(nuevo().imprimir('caja', (d) => d.cortar()), (e) => {
        assert.ok(e instanceof PrismaError);
        assert.deepEqual([e.codigo, e.mensaje, e.sugerencia, e.status, e.trabajo_id], [codigo, 'm', 's', estado, 'trb_9']);
        return true;
      });
      assert.equal(recibidas.length, 1, `${codigo} no debe reintentarse`);
    }
  });

  test('una respuesta que no es del puente', async () => {
    reiniciar(() => [200, '<html>router</html>', 'text/html']);
    await assert.rejects(nuevo().salud(), { codigo: 'RESPUESTA_INVALIDA' });
  });
});

describe('reintentos', () => {
  test('un error de red se reintenta con el mismo documento (mismo id)', async () => {
    reiniciar((_, n) => (n < 3 ? [0] : [200, { ok: true, trabajo_id: 'trb_3', impresora: 'caja', advertencias: [] }]));
    const r = await nuevo().imprimir('caja', (d) => d.texto('Una sola vez').cortar());
    assert.equal(r.trabajo_id, 'trb_3');
    assert.equal(recibidas.length, 3);
    assert.equal(new Set(recibidas.map((p) => p.cuerpo.id)).size, 1, 'los reintentos reutilizan el id');
  });

  test('se rinde tras los reintentos configurados', async () => {
    reiniciar(() => [0]);
    await assert.rejects(nuevo({ reintentos: 1 }).salud(), (e) => e.codigo === 'PUENTE_INALCANZABLE' && e.reintentable);
    assert.equal(recibidas.length, 2);
  });

  test('sin respuesta a tiempo es un error de red', async () => {
    reiniciar(async (_, n) => { if (n === 1) await new Promise((r) => setTimeout(r, 300)); return [200, { ok: true }]; });
    assert.deepEqual(await nuevo({ tiempo_espera_ms: 100 }).salud(), { ok: true });
    assert.equal(recibidas.length, 2);
    reiniciar(async () => { await new Promise((r) => setTimeout(r, 300)); return [200, { ok: true }]; });
    await assert.rejects(nuevo({ tiempo_espera_ms: 50, reintentos: 0 }).salud(), { codigo: 'PUENTE_SIN_RESPUESTA' });
  });

  test('puente apagado (conexión rechazada)', async () => {
    const libre = createServer();
    await new Promise((r) => libre.listen(0, '127.0.0.1', r));
    const puerto = libre.address().port;
    await new Promise((r) => libre.close(r));
    const prisma = new Prisma({ url: `http://127.0.0.1:${puerto}`, reintentos: 0 });
    await assert.rejects(prisma.salud(), (e) => e.codigo === 'PUENTE_INALCANZABLE' && /ECONNREFUSED/.test(e.mensaje));
    assert.deepEqual(await prisma.diagnosticar().then((d) => [d.ok, d.error.codigo]), [false, 'PUENTE_INALCANZABLE']);
  });
});

describe('diagnóstico', () => {
  const falloDeRed = (code) => Object.assign(new TypeError('fetch failed'), { cause: Object.assign(new Error(code), { code }) });

  test('Node: certificado no confiable, sin reintentar', async () => {
    let llamadas = 0;
    const prisma = new Prisma({ url: 'https://192.168.1.10:8443', fetch: async () => { llamadas++; throw falloDeRed('DEPTH_ZERO_SELF_SIGNED_CERT'); } });
    await assert.rejects(prisma.salud(), (e) => e.codigo === 'CERTIFICADO_NO_CONFIABLE' && /NODE_EXTRA_CA_CERTS/.test(e.sugerencia));
    assert.equal(llamadas, 1);
  });

  test('configuración inválida', () => {
    assert.throws(() => new Prisma({ url: '192.168.1.10' }), { codigo: 'CONFIGURACION_INVALIDA' });
    assert.throws(() => new Prisma({ url: 'https://x', llave: 'sk_123' }), { codigo: 'CONFIGURACION_INVALIDA' });
    assert.equal(new Prisma({ url: 'https://192.168.1.10:8443/v1/' }).url, 'https://192.168.1.10:8443');
  });

  // Simula el navegador: window, location y navigator. Cada prueba usa una URL distinta porque el
  // SDK recuerda qué puentes ya respondieron desde la página.
  async function enNavegador({ protocolo = 'http:', permiso = 'prompt', enLinea = true }, f) {
    const previo = Object.getOwnPropertyDescriptor(globalThis, 'navigator');
    globalThis.window = { document: {} };
    globalThis.location = { protocol: protocolo, origin: `${protocolo}//tienda.example` };
    Object.defineProperty(globalThis, 'navigator', {
      configurable: true,
      value: { onLine: enLinea, permissions: { query: async ({ name }) => ({ state: name === 'local-network-access' ? permiso : 'prompt' }) } },
    });
    try {
      await f();
    } finally {
      delete globalThis.window;
      delete globalThis.location;
      Object.defineProperty(globalThis, 'navigator', previo);
    }
  }

  // fetch del navegador: con mode "cors" falla como TypeError; con "no-cors" responde si el puente está.
  const fetchNavegador = ({ responde }) => async (_, init) => {
    if (init.mode === 'no-cors' && responde) return new Response(null, { status: 200 });
    throw new TypeError('Failed to fetch');
  };

  test('navegador: el puente responde pero el origen no está aprobado', () => enNavegador({}, async () => {
    const prisma = new Prisma({ url: 'https://10.0.0.1:8443', fetch: fetchNavegador({ responde: true }) });
    await assert.rejects(prisma.imprimir('caja', (d) => d.cortar()), (e) => {
      assert.equal(e.codigo, 'ORIGEN_NO_AUTORIZADO');
      assert.match(e.mensaje, /http:\/\/tienda\.example/);
      assert.match(e.sugerencia, /panel/);
      assert.equal(e.reintentable, false);
      return true;
    });
  }));

  test('navegador: permiso de red local denegado', () => enNavegador({ permiso: 'denied' }, async () => {
    const prisma = new Prisma({ url: 'https://10.0.0.2:8443', fetch: fetchNavegador({ responde: false }) });
    await assert.rejects(prisma.salud(), { codigo: 'PERMISO_RED_LOCAL_DENEGADO' });
  }));

  test('navegador: primera conexión fallida → certificado o puente apagado, sin reintentar', () => enNavegador({}, async () => {
    let intentos = 0;
    const fetchContado = async (url, init) => { if (init.mode !== 'no-cors') intentos++; return fetchNavegador({ responde: false })(url, init); };
    const prisma = new Prisma({ url: 'https://10.0.0.3:8443', espera_reintento_ms: 5, fetch: fetchContado });
    await assert.rejects(prisma.salud(), (e) => {
      assert.equal(intentos, 1, 'no reintenta: el certificado no se arregla esperando');
      assert.equal(e.reintentable, false);
      assert.equal(e.codigo, 'PUENTE_INALCANZABLE');
      assert.deepEqual(e.causas_posibles, ['CERTIFICADO_NO_CONFIABLE', 'PUENTE_APAGADO_O_FUERA_DE_RED']);
      assert.match(e.sugerencia, /https:\/\/10\.0\.0\.3:8443\/v1\/salud/);
      return true;
    });
  }));

  test('navegador: un puente que ya respondió y deja de hacerlo', () => enNavegador({}, async () => {
    let apagado = false;
    const prisma = new Prisma({
      url: 'https://10.0.0.4:8443', reintentos: 0,
      fetch: async (url, init) => {
        if (apagado) throw new TypeError('Failed to fetch');
        return new Response('{"ok":true}', { headers: { 'Content-Type': 'application/json' } });
      },
    });
    await prisma.salud();
    apagado = true;
    await assert.rejects(prisma.salud(), (e) => e.codigo === 'PUENTE_INALCANZABLE' && e.causas_posibles.length === 1);
  }));

  test('navegador: sin conexión a la red', () => enNavegador({ enLinea: false }, async () => {
    const prisma = new Prisma({ url: 'https://10.0.0.5:8443', reintentos: 0, fetch: fetchNavegador({ responde: false }) });
    await assert.rejects(prisma.salud(), (e) => e.codigo === 'PUENTE_INALCANZABLE' && /conexión a la red/.test(e.mensaje));
  }));

  test('navegador: página https con puente http', () => enNavegador({ protocolo: 'https:' }, async () => {
    let llamadas = 0;
    const prisma = new Prisma({ url: 'http://192.168.1.10:8443', fetch: async () => { llamadas++; throw new TypeError('Failed to fetch'); } });
    await assert.rejects(prisma.salud(), { codigo: 'CONTENIDO_MIXTO' });
    assert.equal(llamadas, 0, 'ni siquiera se intenta');
    // En localhost el navegador sí lo permite.
    await assert.rejects(new Prisma({ url: 'http://localhost:8443', reintentos: 0, fetch: fetchNavegador({ responde: false }) }).salud(), { codigo: 'PUENTE_INALCANZABLE' });
  }));
});
