// node --test sdk/firmadores/node/
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { canonicalizar, firmar } from './firmador.mjs';

const vector = (n) => readFileSync(new URL(`../../../testdata/vector-firma/${n}`, import.meta.url), 'utf8');
const casos = JSON.parse(readFileSync(new URL('../casos-jcs.json', import.meta.url), 'utf8'));
const opciones = { clienteId: 'cli_7Hq2MZp4XwR9', llavePrivada: vector('llave-privada.pem'), ahora: new Date('2026-10-14T18:30:00Z') };

test('forma canónica igual a la del puente (casos-jcs.json)', () => {
  for (const c of casos) assert.equal(canonicalizar(c.entrada), c.canonico, c.nombre);
});

test('vector fijo: misma forma canónica, misma firma y mismo documento', () => {
  const firmado = firmar(JSON.parse(vector('documento.json')), opciones);
  const sinFirma = { ...firmado };
  delete sinFirma.firma;
  assert.equal(canonicalizar(sinFirma), vector('canonico.txt'));
  assert.equal(firmado.firma, vector('firma.txt').trim());
  assert.deepEqual(firmado, JSON.parse(vector('firmado.json')));
});

test('volver a firmar ignora la firma anterior', () => {
  assert.equal(firmar(JSON.parse(vector('firmado.json')), opciones).firma, vector('firma.txt').trim());
});

test('id nuevo si falta, vigencia y llave', () => {
  const doc = JSON.parse(vector('documento.json'));
  delete doc.id;
  const firmado = firmar(doc, { ...opciones, ahora: new Date('2026-10-14T18:30:00.750Z'), vigencia: 120 });
  assert.match(firmado.id, /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
  assert.deepEqual([firmado.emitido, firmado.expira], ['2026-10-14T18:30:00Z', '2026-10-14T18:32:00Z']);
  assert.throws(() => firmar(doc, { ...opciones, vigencia: 121 }), RangeError);
  assert.throws(() => firmar(doc, { ...opciones, clienteId: 'cliente' }), TypeError);
});
