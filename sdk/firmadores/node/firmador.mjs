// Firmador de referencia de PRISMA (nivel 3) para Node.js 18+. Sin dependencias.
//
//   import { firmar } from './firmador.mjs';
//   const firmado = firmar(documento, { clienteId: 'cli_…', llavePrivada: pem });
//
// Firma con Ed25519 la forma canónica JCS (RFC 8785) del documento sin «firma», como verifica el
// puente (docs/contrato-documento-v1.md §7).
import { createPrivateKey, randomUUID, sign } from 'node:crypto';

const VIGENCIA_MAXIMA = 120;

/**
 * Forma canónica JCS (RFC 8785): claves ordenadas por unidades UTF-16, sin espacios y con el escape
 * mínimo de JSON. JSON.stringify ya serializa cadenas y números como pide el RFC.
 */
export function canonicalizar(valor) {
  if (valor === null || typeof valor === 'boolean' || typeof valor === 'string') return JSON.stringify(valor);
  if (typeof valor === 'number') {
    if (!Number.isFinite(valor)) throw new TypeError(`JCS no admite el número ${valor}`);
    return JSON.stringify(valor);
  }
  if (Array.isArray(valor)) return `[${valor.map(canonicalizar).join(',')}]`;
  if (typeof valor === 'object') {
    // sort() sin comparador ordena por unidades de código UTF-16, exactamente lo que pide JCS.
    const claves = Object.keys(valor).filter((k) => valor[k] !== undefined).sort();
    return `{${claves.map((k) => `${JSON.stringify(k)}:${canonicalizar(valor[k])}`).join(',')}}`;
  }
  throw new TypeError(`JCS no admite valores de tipo ${typeof valor}`);
}

const fechaRFC3339 = (d) => d.toISOString().replace(/\.\d{3}Z$/, 'Z');

/**
 * Completa y firma un documento v1. Devuelve un objeto nuevo con cliente_id, emitido, expira y
 * firma; si el documento no trae id, le asigna un UUID v4. Ignora una firma anterior.
 *
 * @param {object} documento   documento v1 (sin firmar o firmado antes)
 * @param {object} opciones
 * @param {string} opciones.clienteId     id del cliente de nivel 3 en el puente (cli_…)
 * @param {string|import('node:crypto').KeyObject} opciones.llavePrivada  PEM PKCS#8 de la llave Ed25519
 * @param {number} [opciones.vigencia=90] segundos hasta que expira (máximo 120)
 * @param {Date}   [opciones.ahora]       hora de emisión (para pruebas)
 */
export function firmar(documento, { clienteId, llavePrivada, vigencia = 90, ahora = new Date() }) {
  if (!Number.isInteger(vigencia) || vigencia < 1 || vigencia > VIGENCIA_MAXIMA) {
    throw new RangeError(`La vigencia debe ser de 1 a ${VIGENCIA_MAXIMA} segundos.`);
  }
  if (!/^cli_[A-Za-z0-9]{8,40}$/.test(clienteId ?? '')) throw new TypeError('clienteId debe tener la forma cli_… (el id que muestra el panel).');
  const llave = typeof llavePrivada === 'string' ? createPrivateKey(llavePrivada) : llavePrivada;
  if (llave.asymmetricKeyType !== 'ed25519') throw new TypeError('La llave privada debe ser Ed25519.');

  const emitido = new Date(Math.floor(ahora.getTime() / 1000) * 1000);
  const doc = { ...documento, id: documento.id ?? randomUUID(), cliente_id: clienteId,
    emitido: fechaRFC3339(emitido), expira: fechaRFC3339(new Date(emitido.getTime() + vigencia * 1000)) };
  delete doc.firma;
  const firma = sign(null, Buffer.from(canonicalizar(doc), 'utf8'), llave);
  return { ...doc, firma: firma.toString('base64') };
}
