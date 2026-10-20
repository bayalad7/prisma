// Los archivos de dist/ funcionan en cada forma de carga: <script> (global), CommonJS y módulo ES.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import vm from 'node:vm';

const umd = readFileSync(new URL('../dist/prisma.umd.js', import.meta.url), 'utf8');

const usar = (Prisma) => {
  const prisma = new Prisma({ url: 'https://192.168.1.10:8443', llave: 'pk_x' });
  const doc = prisma.documento('caja').texto('Hola').cortar().json();
  assert.equal(doc.operaciones.length, 2);
  assert.equal(typeof Prisma.Documento, 'function');
  assert.equal(typeof Prisma.validarDocumento, 'function');
  assert.equal(new Prisma.PrismaError({ codigo: 'X' }).name, 'PrismaError'); // en vm, Error es de otro realm
};

test('UMD con <script>: define window.Prisma', () => {
  const ventana = { crypto: globalThis.crypto, TextEncoder, URL, AbortController, setTimeout, clearTimeout };
  ventana.globalThis = ventana;
  vm.runInNewContext(umd, ventana);
  usar(ventana.Prisma);
});

test('UMD con require (CommonJS)', () => {
  const Prisma = createRequire(import.meta.url)('../dist/prisma.umd.js');
  usar(Prisma);
  const { Prisma: mismo, Documento } = Prisma;
  assert.equal(mismo, Prisma);
  assert.equal(typeof Documento, 'function');
});

test('módulo ES', async () => {
  const m = await import('../dist/prisma.mjs');
  usar(m.default);
  assert.equal(m.Prisma, m.default);
  assert.equal(m.VERSION_SDK, JSON.parse(readFileSync(new URL('../package.json', import.meta.url), 'utf8')).version);
});

test('el paquete npm resuelve por nombre a los mismos archivos', async () => {
  const pkg = JSON.parse(readFileSync(new URL('../package.json', import.meta.url), 'utf8'));
  assert.equal(pkg.exports['.'].import.default, './dist/prisma.mjs');
  assert.equal(pkg.exports['.'].require.default, './dist/prisma.umd.js');
  assert.equal(pkg.jsdelivr, './dist/prisma.umd.js');
  assert.deepEqual(pkg.dependencies, undefined, 'el SDK no tiene dependencias');
});
