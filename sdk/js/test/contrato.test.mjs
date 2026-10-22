// El validador local debe coincidir con el puente: los ejemplos válidos pasan y cada ejemplo
// inválido recibe el mismo código y apunta al mismo campo que indica ejemplos/invalidos/indice.json.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, readdirSync } from 'node:fs';
import { validarDocumento } from '../src/prisma.mjs';

const ejemplos = new URL('../../../ejemplos/', import.meta.url);
const leer = (ruta) => readFileSync(new URL(ruta, ejemplos), 'utf8');

// Lo que el SDK no puede ver sin la impresora o sin decodificar el JSON crudo lo detecta el puente.
const SOLO_PUENTE = new Set([
  'json-mal-formado.json', // el SDK trabaja con objetos, no con texto JSON
  'clave-duplicada.json',
  'fila-columna-sin-espacio.json', // depende de las columnas de la impresora
  'imagen-png-corrupto.json', // requiere decodificar el PNG completo
  'firma-alterada.json', // verificar la firma requiere la llave pública, que solo tiene el puente
]);

test('los ejemplos válidos pasan la validación local', () => {
  for (const archivo of readdirSync(new URL('validos/', ejemplos))) {
    const error = validarDocumento(JSON.parse(leer(`validos/${archivo}`)));
    assert.equal(error, null, `${archivo}: ${error?.codigo} ${JSON.stringify(error?.detalles)}`);
  }
});

test('cada ejemplo inválido recibe el código y la ruta del puente', () => {
  const indice = JSON.parse(leer('invalidos/indice.json'));
  assert.ok(indice.length > 60);
  for (const caso of indice) {
    const texto = leer(`invalidos/${caso.archivo}`);
    let doc;
    try {
      doc = JSON.parse(texto);
    } catch {
      assert.ok(SOLO_PUENTE.has(caso.archivo), `${caso.archivo} no es JSON y no está marcado como solo del puente`);
      continue;
    }
    const error = validarDocumento(doc);
    if (SOLO_PUENTE.has(caso.archivo)) {
      assert.equal(error, null, `${caso.archivo}: el SDK no debe rechazar lo que solo el puente puede juzgar (${error?.codigo})`);
      continue;
    }
    assert.ok(error, `${caso.archivo}: debería ser inválido (${caso.codigo})`);
    assert.equal(error.codigo, caso.codigo, `${caso.archivo}: ${JSON.stringify(error.detalles)}`);
    if (error.detalles) {
      assert.ok(error.detalles.some((d) => d.ruta === caso.ruta), `${caso.archivo}: ninguna ruta es ${caso.ruta}: ${JSON.stringify(error.detalles)}`);
    }
  }
});

test('el vector de firma es válido y su vigencia se comprueba', () => {
  const firmado = JSON.parse(readFileSync(new URL('../../../testdata/vector-firma/firmado.json', import.meta.url), 'utf8'));
  assert.equal(validarDocumento(firmado), null);
  assert.equal(validarDocumento({ ...firmado, expira: '2026-10-14T18:33:00Z' }).codigo, 'FIRMA_INVALIDA');
  const sinFirma = { ...firmado };
  delete sinFirma.firma;
  const e = validarDocumento(sinFirma);
  assert.equal(e.codigo, 'DOCUMENTO_INVALIDO');
  assert.equal(e.detalles[0].regla, 'dependentRequired');
});

test('rechaza documentos de más de 256 KB', () => {
  const ops = Array.from({ length: 70 }, () => ({ tipo: 'texto', valor: 'x'.repeat(4000) }));
  const e = validarDocumento({ version: '1', id: '3f2b8c1e-5a4d-4e7b-9c2a-1d0e6f8a7b90', impresora: 'caja', operaciones: ops });
  assert.equal(e.codigo, 'DOCUMENTO_DEMASIADO_GRANDE');
});
