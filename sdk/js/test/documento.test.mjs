import { test } from 'node:test';
import assert from 'node:assert/strict';
import { Documento, PrismaError, generarId, aBase64, pngABase64, validarDocumento } from '../src/prisma.mjs';

const PNG_1X1 = 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAAAAAA6fptVAAAACklEQVR4nGNgAAAAAgABSK+kcQAAAABJRU5ErkJggg==';

test('el ejemplo del README produce operaciones v1 válidas', () => {
  const doc = new Documento('caja')
    .texto('MI NEGOCIO', { alineacion: 'centro', negrita: true, tamano: [2, 2] })
    .separador()
    .par('2 x Taco', '$60.00')
    .par('TOTAL', '$60.00', { negrita: true })
    .qr('https://minegocio.com/p/812')
    .cortar()
    .json();
  assert.equal(doc.version, '1');
  assert.match(doc.id, /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
  assert.deepEqual(doc.operaciones, [
    { tipo: 'texto', valor: 'MI NEGOCIO', alineacion: 'centro', negrita: true, tamano: [2, 2] },
    { tipo: 'separador' },
    { tipo: 'par', izquierda: '2 x Taco', derecha: '$60.00' },
    { tipo: 'par', izquierda: 'TOTAL', derecha: '$60.00', negrita: true },
    { tipo: 'qr', valor: 'https://minegocio.com/p/812' },
    { tipo: 'cortar' },
  ]);
  assert.equal(validarDocumento(doc), null);
});

test('todas las operaciones y el sobre', () => {
  const doc = new Documento('cocina', { referencia: 'PED-1' })
    .copias(2)
    .fila(['1', { texto: 'Pan de caja', ancho: 'auto' }, { texto: '$62.50', ancho: 25, alineacion: 'derecha' }], { negrita: true })
    .salto(2)
    .codigoBarras('4006381333931', 'EAN13', { alto: 60 })
    .imagen('logo-tienda', { ancho_max: 60 })
    .imagen(PNG_1X1)
    .imagen(`data:image/png;base64,${PNG_1X1}`)
    .imagen(Uint8Array.from(Buffer.from(PNG_1X1, 'base64')))
    .abrirCajon(5)
    .raw(new Uint8Array([0x1b, 0x40]))
    .cortar({ parcial: true })
    .json();
  assert.equal(doc.copias, 2);
  assert.equal(doc.referencia, 'PED-1');
  assert.deepEqual(doc.operaciones[0].columnas[0], { texto: '1' });
  assert.deepEqual(doc.operaciones[3], { tipo: 'imagen', recurso: 'logo-tienda', ancho_max: 60 });
  for (const i of [4, 5, 6]) assert.equal(doc.operaciones[i].base64, PNG_1X1);
  assert.deepEqual(doc.operaciones.at(-2), { tipo: 'raw', base64: 'G0A=' });
});

test('cada operación se valida al agregarla y el error señala el campo', () => {
  const casos = [
    [(d) => d.texto('x', { alineacion: 'center' }), 'DOCUMENTO_INVALIDO', '/operaciones/0/alineacion'],
    [(d) => d.texto('Hola\u001b@'), 'DOCUMENTO_INVALIDO', '/operaciones/0/valor'],
    [(d) => d.texto('x', { tamano: [9, 1] }), 'DOCUMENTO_INVALIDO', '/operaciones/0/tamano/0'],
    [(d) => d.texto('x', { color: 'rojo' }), 'DOCUMENTO_INVALIDO', '/operaciones/0'],
    [(d) => d.separador('=='), 'DOCUMENTO_INVALIDO', '/operaciones/0/caracter'],
    [(d) => d.par('a\nb', 'c'), 'DOCUMENTO_INVALIDO', '/operaciones/0/izquierda'],
    [(d) => d.fila([{ texto: 'a', ancho: 70 }, { texto: 'b', ancho: 50 }]), 'FILA_NO_CABE', '/operaciones/0/columnas'],
    [(d) => d.codigoBarras('4006381333932', 'EAN13'), 'CODIGO_BARRAS_INVALIDO', '/operaciones/0/valor'],
    [(d) => d.codigoBarras('abc', 'CODE39'), 'DOCUMENTO_INVALIDO', '/operaciones/0/valor'],
    [(d) => d.imagen('https://ejemplo.com/logo.png'), 'DOCUMENTO_INVALIDO', '/operaciones/0/recurso'],
    [(d) => d.imagen(`iVBORw0KGgo${'A'.repeat(140000)}`), 'DOCUMENTO_INVALIDO', '/operaciones/0/base64'],
    [(d) => d.salto(11), 'DOCUMENTO_INVALIDO', '/operaciones/0/lineas'],
    [(d) => d.abrirCajon(3), 'DOCUMENTO_INVALIDO', '/operaciones/0/pin'],
    [(d) => d.operacion({ tipo: 'sonido' }), 'OPERACION_DESCONOCIDA', '/operaciones/0/tipo'],
  ];
  for (const [agregar, codigo, ruta] of casos) {
    const d = new Documento('caja');
    assert.throws(() => agregar(d), (e) => {
      assert.ok(e instanceof PrismaError);
      assert.equal(e.codigo, codigo, `${e.codigo}: ${e.mensaje}`);
      assert.equal(e.status, 0);
      assert.ok(e.sugerencia);
      assert.ok(e.detalles.some((x) => x.ruta === ruta), `${ruta} no está en ${JSON.stringify(e.detalles)}`);
      return true;
    });
  }
});

test('el sobre se valida al pedir el JSON', () => {
  assert.throws(() => new Documento('192.168.1.50').texto('x').json(), (e) => e.detalles[0].ruta === '/impresora');
  assert.throws(() => new Documento('caja').json(), (e) => e.detalles[0].regla === 'minItems');
  assert.throws(() => new Documento('caja').copias(6).texto('x').json(), (e) => e.detalles[0].ruta === '/copias');
  assert.throws(() => new Documento('caja', { id: 'no-es-uuid' }).texto('x').json(), (e) => e.detalles[0].ruta === '/id');
});

test('los valores undefined se omiten en lugar de enviarse', () => {
  const doc = new Documento('caja').texto('x', { negrita: undefined }).separador(undefined).salto().json();
  assert.deepEqual(doc.operaciones, [{ tipo: 'texto', valor: 'x' }, { tipo: 'separador' }, { tipo: 'salto' }]);
});

test('JSON.stringify de un Documento produce el documento', () => {
  const d = new Documento('caja', { id: '3f2b8c1e-5a4d-4e7b-9c2a-1d0e6f8a7b90' }).texto('x');
  assert.equal(JSON.stringify(d), '{"version":"1","id":"3f2b8c1e-5a4d-4e7b-9c2a-1d0e6f8a7b90","impresora":"caja","operaciones":[{"tipo":"texto","valor":"x"}]}');
  assert.equal(d.id, '3f2b8c1e-5a4d-4e7b-9c2a-1d0e6f8a7b90');
});

test('utilidades: ids únicos v4 y base64', async () => {
  const ids = new Set(Array.from({ length: 1000 }, generarId));
  assert.equal(ids.size, 1000);
  for (const n of [0, 1, 2, 3, 4, 5, 100]) {
    const b = Uint8Array.from({ length: n }, (_, i) => (i * 37) & 255);
    assert.equal(aBase64(b), Buffer.from(b).toString('base64'));
  }
  assert.equal(await pngABase64(new Blob([Buffer.from(PNG_1X1, 'base64')])), PNG_1X1);
});
