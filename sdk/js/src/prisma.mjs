// SDK de PRISMA: imprime en un puente PRISMA desde el navegador o desde Node.js 18+.
// Sin dependencias. Este archivo es la fuente; dist/ se genera con «npm run construir».

export const VERSION_SDK = '0.1.0';

const VERSION_DOCUMENTO = '1';
const MAX_BYTES_DOCUMENTO = 256 * 1024;
const MAX_BYTES_IMAGEN = 100 * 1024;
const MAX_BYTES_RAW = 64 * 1024;
const VIGENCIA_MAXIMA_MS = 120 * 1000;

// ---------------------------------------------------------------------------------------------
// Errores
// ---------------------------------------------------------------------------------------------

/**
 * Error de PRISMA: los que responde el puente y los que detecta el SDK antes de enviar o al no
 * poder conectar. Programa contra `codigo`, que es estable; `mensaje` puede cambiar.
 */
export class PrismaError extends Error {
  constructor({ codigo, mensaje, sugerencia = '', status = 0, detalles, trabajo_id, causas_posibles, respuesta, causa } = {}) {
    super(mensaje || codigo);
    this.name = 'PrismaError';
    this.codigo = codigo;
    this.mensaje = mensaje || codigo;
    this.sugerencia = sugerencia;
    this.status = status; // 0 = el error no viene del puente (validación local o red)
    if (detalles) this.detalles = detalles;
    if (trabajo_id) this.trabajo_id = trabajo_id;
    if (causas_posibles) this.causas_posibles = causas_posibles;
    if (respuesta) this.respuesta = respuesta;
    if (causa) this.cause = causa;
    // Solo los errores de red se reintentan; nunca los de negocio.
    this.reintentable = false;
  }
}

// ---------------------------------------------------------------------------------------------
// Utilidades
// ---------------------------------------------------------------------------------------------

/** Genera un UUID v4 en minúsculas (sirve también en páginas http://, donde no hay randomUUID). */
export function generarId() {
  const b = new Uint8Array(16);
  globalThis.crypto.getRandomValues(b);
  b[6] = (b[6] & 0x0f) | 0x40;
  b[8] = (b[8] & 0x3f) | 0x80;
  const h = Array.from(b, (x) => x.toString(16).padStart(2, '0')).join('');
  return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`;
}

const ALFABETO_B64 = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';

/** Codifica bytes en base64 estándar. */
export function aBase64(datos) {
  const b = datos instanceof Uint8Array ? datos : new Uint8Array(datos instanceof ArrayBuffer ? datos : datos.buffer);
  let s = '';
  for (let i = 0; i < b.length; i += 3) {
    const n = (b[i] << 16) | ((b[i + 1] ?? 0) << 8) | (b[i + 2] ?? 0);
    s += ALFABETO_B64[(n >> 18) & 63] + ALFABETO_B64[(n >> 12) & 63];
    s += i + 1 < b.length ? ALFABETO_B64[(n >> 6) & 63] : '=';
    s += i + 2 < b.length ? ALFABETO_B64[n & 63] : '=';
  }
  return s;
}

/** Decodifica base64 estándar; null si no es válido. */
function deBase64(s) {
  if (typeof s !== 'string' || s.length % 4 !== 0 || !/^[A-Za-z0-9+/]*={0,2}$/.test(s)) return null;
  const relleno = s.endsWith('==') ? 2 : s.endsWith('=') ? 1 : 0;
  const salida = new Uint8Array((s.length / 4) * 3 - relleno);
  let j = 0;
  for (let i = 0; i < s.length; i += 4) {
    const n = (ALFABETO_B64.indexOf(s[i]) << 18) | (ALFABETO_B64.indexOf(s[i + 1]) << 12)
      | ((ALFABETO_B64.indexOf(s[i + 2]) & 63) << 6) | (ALFABETO_B64.indexOf(s[i + 3]) & 63);
    if (j < salida.length) salida[j++] = (n >> 16) & 255;
    if (j < salida.length) salida[j++] = (n >> 8) & 255;
    if (j < salida.length) salida[j++] = n & 255;
  }
  return salida;
}

/** Lee un archivo PNG (File, Blob, ArrayBuffer o bytes) y lo devuelve en base64 para `imagen`. */
export async function pngABase64(archivo) {
  const datos = typeof archivo.arrayBuffer === 'function' ? await archivo.arrayBuffer() : archivo;
  return aBase64(datos);
}

const largo = (s) => [...s].length; // en puntos de código, como el esquema
const bytesUTF8 = (s) => new TextEncoder().encode(s).length;
const esObjeto = (v) => v !== null && typeof v === 'object' && !Array.isArray(v);
const esEntero = (v) => typeof v === 'number' && Number.isInteger(v);

// ---------------------------------------------------------------------------------------------
// Validación local (mismas reglas que schema/documento-v1.schema.json y el puente)
// ---------------------------------------------------------------------------------------------

const PATRON_ID = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
const PATRON_ALIAS = /^[a-z0-9][a-z0-9_-]{0,31}$/;
const PATRON_RECURSO = /^[a-z0-9][a-z0-9_-]{0,63}$/;
const PATRON_CLIENTE = /^cli_[A-Za-z0-9]{8,40}$/;
const PATRON_FIRMA = /^[A-Za-z0-9+/]{86}==$/;
const PATRON_FECHA = /^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?(Z|[+-][0-9]{2}:[0-9]{2})$/;
const CONTROL_LINEA = /[\u0000-\u001f\u007f-\u009f]/;
const CONTROL_MULTILINEA = /[\u0000-\u0009\u000b-\u001f\u007f-\u009f]/;
const ALINEACIONES = ['izquierda', 'centro', 'derecha'];
const TIPOS = ['texto', 'separador', 'fila', 'par', 'salto', 'qr', 'codigo_barras', 'imagen', 'cortar', 'abrir_cajon', 'raw'];
const PATRON_SIMBOLOGIA = {
  EAN13: [/^[0-9]{12,13}$/, '12 o 13 dígitos'],
  EAN8: [/^[0-9]{7,8}$/, '7 u 8 dígitos'],
  UPCA: [/^[0-9]{11,12}$/, '11 o 12 dígitos'],
  ITF: [/^([0-9]{2}){1,40}$/, 'de 2 a 80 dígitos, en número par'],
  CODE39: [/^[0-9A-Z .$/+%-]{1,40}$/, 'de 1 a 40 caracteres de 0-9, A-Z, espacio y - . $ / + %'],
  CODE128: [/^[ -~]{1,80}$/, 'de 1 a 80 caracteres ASCII imprimibles'],
};
const LARGO_GS1 = { EAN13: 13, EAN8: 8, UPCA: 12 };

// Validadores de campo: (valor, ruta, falla) → falla(ruta, regla, mensaje) por cada problema.
const tipo = (esperado) => (v, ruta, falla) => {
  const ok = esperado === 'integer' ? esEntero(v) : esperado === 'array' ? Array.isArray(v) : esperado === 'object' ? esObjeto(v) : typeof v === esperado;
  if (!ok) falla(ruta, 'type', `debe ser ${{ string: 'una cadena', boolean: 'true o false', integer: 'un número entero', array: 'un arreglo', object: 'un objeto' }[esperado]}`);
  return ok;
};
const booleano = tipo('boolean');
const entero = (min, max) => (v, ruta, falla) => {
  if (!tipo('integer')(v, ruta, falla)) return;
  if (v < min) falla(ruta, 'minimum', `debe ser al menos ${min}`);
  if (v > max) falla(ruta, 'maximum', `debe ser como máximo ${max}`);
};
const opciones = (lista) => (v, ruta, falla) => {
  if (!lista.includes(v)) falla(ruta, 'enum', `debe ser uno de: ${lista.join(', ')}`);
};
const cadena = ({ min = 0, max = Infinity, control, patron, explicacion } = {}) => (v, ruta, falla) => {
  if (!tipo('string')(v, ruta, falla)) return;
  const n = largo(v);
  if (n < min) falla(ruta, 'minLength', min === 1 ? 'no puede estar vacío' : `debe tener al menos ${min} caracteres`);
  if (n > max) falla(ruta, 'maxLength', `debe tener como máximo ${max} caracteres`);
  if (control && control.test(v)) falla(ruta, 'pattern', control === CONTROL_MULTILINEA
    ? 'no admite caracteres de control (solo el salto de línea \\n)' : 'no admite caracteres de control ni saltos de línea');
  if (patron && !patron.test(v)) falla(ruta, 'pattern', explicacion);
};
const alineacion = opciones(ALINEACIONES);

const OPERACIONES = {
  texto: {
    requeridos: ['valor'],
    campos: {
      valor: cadena({ max: 4096, control: CONTROL_MULTILINEA }),
      alineacion, negrita: booleano, subrayado: booleano, invertido: booleano,
      tamano: (v, ruta, falla) => {
        if (!tipo('array')(v, ruta, falla)) return;
        if (v.length < 2) falla(ruta, 'minItems', 'debe tener dos enteros: [ancho, alto]');
        if (v.length > 2) falla(ruta, 'maxItems', 'debe tener dos enteros: [ancho, alto]');
        v.slice(0, 2).forEach((x, i) => entero(1, 8)(x, `${ruta}/${i}`, falla));
      },
      fuente: opciones(['A', 'B']),
    },
  },
  separador: { campos: { caracter: cadena({ min: 1, max: 1, control: CONTROL_LINEA }) } },
  fila: {
    requeridos: ['columnas'],
    campos: {
      columnas: (v, ruta, falla) => {
        if (!tipo('array')(v, ruta, falla)) return;
        if (v.length < 1) falla(ruta, 'minItems', 'debe tener al menos una columna');
        if (v.length > 8) falla(ruta, 'maxItems', 'debe tener como máximo 8 columnas');
        v.forEach((c, i) => validarObjeto(c, `${ruta}/${i}`, falla, {
          requeridos: ['texto'],
          campos: {
            texto: cadena({ max: 1024, control: CONTROL_LINEA }),
            ancho: (a, r, f) => {
              if (a !== 'auto' && !(esEntero(a) && a >= 1 && a <= 100)) f(r, 'anyOf', 'debe ser un porcentaje entero de 1 a 100 o "auto"');
            },
            alineacion,
          },
        }));
      },
      negrita: booleano,
    },
  },
  par: {
    requeridos: ['izquierda', 'derecha'],
    campos: { izquierda: cadena({ max: 1024, control: CONTROL_LINEA }), derecha: cadena({ max: 1024, control: CONTROL_LINEA }), negrita: booleano },
  },
  salto: { campos: { lineas: entero(1, 10) } },
  qr: {
    requeridos: ['valor'],
    campos: { valor: cadena({ min: 1, max: 700 }), tamano: entero(1, 16), correccion: opciones(['L', 'M', 'Q', 'H']), alineacion },
  },
  codigo_barras: {
    requeridos: ['valor', 'simbologia'],
    campos: {
      valor: cadena({ min: 1 }),
      simbologia: opciones(Object.keys(PATRON_SIMBOLOGIA)),
      texto: opciones(['arriba', 'abajo', 'ninguno']),
      alto: entero(1, 255),
      alineacion,
    },
    extra: (op, ruta, falla) => {
      const regla = PATRON_SIMBOLOGIA[op.simbologia];
      if (regla && typeof op.valor === 'string' && !regla[0].test(op.valor)) {
        falla(`${ruta}/valor`, 'pattern', `${op.simbologia} admite ${regla[1]}`);
      }
    },
  },
  imagen: {
    campos: {
      base64: cadena({ max: 136536, patron: /^iVBORw0KGgo[A-Za-z0-9+/]*={0,2}$/, explicacion: 'debe ser un PNG en base64 estándar, sin el prefijo data:' }),
      recurso: cadena({ patron: PATRON_RECURSO, explicacion: 'solo admite minúsculas, dígitos, - y _ (hasta 64 caracteres)' }),
      alineacion,
      ancho_max: entero(1, 100),
    },
    extra: (op, ruta, falla) => {
      if (('base64' in op) === ('recurso' in op)) falla(ruta, 'oneOf', 'debe traer exactamente uno de base64 o recurso');
    },
  },
  cortar: { campos: { parcial: booleano } },
  abrir_cajon: { campos: { pin: opciones([2, 5]) } },
  raw: {
    requeridos: ['base64'],
    campos: { base64: cadena({ min: 4, max: 87384, patron: /^[A-Za-z0-9+/]*={0,2}$/, explicacion: 'debe ser base64 estándar' }) },
  },
};

function validarObjeto(v, ruta, falla, { requeridos = [], campos, extra }) {
  if (!tipo('object')(v, ruta, falla)) return;
  for (const r of requeridos) if (!(r in v)) falla(ruta, 'required', `falta el campo obligatorio «${r}»`);
  for (const [k, x] of Object.entries(v)) {
    if (k === 'tipo') continue;
    if (!(k in campos)) falla(ruta, 'additionalProperties', `el campo «${k}» no existe en el contrato`);
    else if (x === null) falla(`${ruta}/${k}`, 'type', 'null no es válido: omite el campo');
    else campos[k](x, `${ruta}/${k}`, falla);
  }
  if (extra) extra(v, ruta, falla);
}

/** Valida una operación; devuelve la lista de detalles (vacía si es válida). */
function validarOperacion(op, ruta) {
  const detalles = [];
  const falla = (r, regla, mensaje) => detalles.push({ ruta: r, regla, mensaje });
  if (!esObjeto(op)) {
    falla(ruta, 'type', 'cada operación debe ser un objeto');
  } else if (!('tipo' in op)) {
    falla(ruta, 'required', 'falta el campo obligatorio «tipo»');
  } else if (!TIPOS.includes(op.tipo)) {
    falla(`${ruta}/tipo`, 'enum', `«${op.tipo}» no es una operación de la versión 1; usa una de: ${TIPOS.join(', ')}`);
  } else {
    validarObjeto(op, ruta, falla, OPERACIONES[op.tipo]);
  }
  return detalles;
}

// Reglas que el esquema no expresa y que el puente comprueba al aceptar el documento.
function validarReglas(op, ruta) {
  const fallas = [];
  const falla = (codigo, r, regla, mensaje, sugerencia) => fallas.push({ codigo, detalle: { ruta: r, regla, mensaje }, sugerencia });
  if (op.tipo === 'fila') {
    const suma = op.columnas.reduce((s, c) => s + (esEntero(c.ancho) ? c.ancho : 0), 0);
    if (suma > 100) {
      falla('FILA_NO_CABE', `${ruta}/columnas`, 'suma_porcentajes', `Los porcentajes de las columnas suman ${suma}; el máximo es 100.`,
        'Reduce los porcentajes o usa "auto" en alguna columna.');
    }
  } else if (op.tipo === 'codigo_barras' && LARGO_GS1[op.simbologia] === op.valor.length) {
    const esperado = digitoControlGS1(op.valor.slice(0, -1));
    if (Number(op.valor.at(-1)) !== esperado) {
      falla('CODIGO_BARRAS_INVALIDO', `${ruta}/valor`, 'digito_verificador',
        `El dígito verificador de ${op.simbologia} ${op.valor} es incorrecto: debería ser ${esperado}.`,
        'Corrige el dígito verificador u omítelo para que lo calcule el puente.');
    }
  } else if (op.tipo === 'imagen' && op.base64 !== undefined) {
    const datos = deBase64(op.base64);
    const problema = !datos ? ['base64', 'el base64 no es válido']
      : datos.length > MAX_BYTES_IMAGEN ? ['tamano_maximo', `mide ${datos.length} bytes decodificada; el máximo es 100 KB`]
        : !esPNG(datos) ? ['png_valido', 'no es un PNG']
          : dimensionesPNG(datos).some((n) => n < 1 || n > 4096) ? ['dimensiones', 'mide más de 4096 px por lado'] : null;
    if (problema) {
      falla('IMAGEN_INVALIDA', `${ruta}/base64`, problema[0], `La imagen no es válida: ${problema[1]}.`,
        'Envía un PNG de hasta 100 KB y 4096 px por lado, o guárdalo en el puente y usa "recurso".');
    }
  } else if (op.tipo === 'raw') {
    const datos = deBase64(op.base64);
    if (!datos) {
      falla('DOCUMENTO_INVALIDO', `${ruta}/base64`, 'base64', 'raw no contiene base64 estándar válido.', 'Codifica los bytes con base64 estándar, con relleno =.');
    } else if (datos.length > MAX_BYTES_RAW) {
      falla('DOCUMENTO_INVALIDO', `${ruta}/base64`, 'tamano_maximo', `raw decodificado mide ${datos.length} bytes; el máximo es 64 KB.`,
        'Divide los bytes en varias operaciones raw o en varios documentos.');
    }
  }
  return fallas;
}

function digitoControlGS1(digitos) {
  let suma = 0;
  for (let i = digitos.length - 1, peso = 3; i >= 0; i--, peso = 4 - peso) suma += Number(digitos[i]) * peso;
  return (10 - (suma % 10)) % 10;
}

const FIRMA_PNG = [0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a];
const esPNG = (b) => b.length >= 24 && FIRMA_PNG.every((x, i) => b[i] === x);
const dimensionesPNG = (b) => {
  const v = new DataView(b.buffer, b.byteOffset, b.byteLength);
  return [v.getUint32(16), v.getUint32(20)];
};

const SUGERENCIAS = {
  VERSION_NO_SOPORTADA: 'Envía "version": "1" (como cadena).',
  OPERACION_DESCONOCIDA: 'Revisa el campo tipo; las operaciones están en docs/referencia/contrato-documento-v1.md §4.',
  DEMASIADAS_OPERACIONES: 'Divide el ticket en varios documentos.',
  DOCUMENTO_INVALIDO: 'Revisa los detalles; cada uno indica el campo y la regla que falló.',
};

function errorDeDocumento(codigo, mensaje, detalles, sugerencia = SUGERENCIAS[codigo]) {
  return new PrismaError({ codigo, mensaje, sugerencia, detalles });
}

/**
 * Valida un documento v1 completo con las mismas reglas que el puente (salvo las que dependen de
 * la impresora). Devuelve null si es válido, o el PrismaError que respondería el puente.
 */
export function validarDocumento(doc) {
  if (!esObjeto(doc)) {
    return errorDeDocumento('DOCUMENTO_INVALIDO', 'El documento no cumple el contrato v1.', [{ ruta: '', regla: 'type', mensaje: 'el documento debe ser un objeto JSON' }]);
  }
  if (doc.version !== VERSION_DOCUMENTO) {
    const detalle = 'version' in doc
      ? { ruta: '/version', regla: 'const', mensaje: 'debe ser la cadena "1"' }
      : { ruta: '', regla: 'required', mensaje: 'falta el campo obligatorio «version»' };
    const e = errorDeDocumento('VERSION_NO_SOPORTADA', `Este SDK genera y valida documentos versión ${VERSION_DOCUMENTO}.`, [detalle]);
    e.versiones_soportadas = [VERSION_DOCUMENTO];
    return e;
  }

  const detalles = [];
  const falla = (ruta, regla, mensaje) => detalles.push({ ruta, regla, mensaje });
  const sobre = {
    id: cadena({ patron: PATRON_ID, explicacion: 'debe ser un UUID v4 en minúsculas' }),
    impresora: cadena({ patron: PATRON_ALIAS, explicacion: 'debe ser el alias de la impresora (minúsculas, dígitos, - y _), nunca una IP' }),
    copias: entero(1, 5),
    referencia: cadena({ min: 1, max: 64, control: CONTROL_LINEA }),
    cliente_id: cadena({ patron: PATRON_CLIENTE, explicacion: 'debe tener la forma cli_…' }),
    emitido: cadena({ patron: PATRON_FECHA, explicacion: 'debe ser una fecha RFC 3339 con zona horaria' }),
    expira: cadena({ patron: PATRON_FECHA, explicacion: 'debe ser una fecha RFC 3339 con zona horaria' }),
    firma: cadena({ patron: PATRON_FIRMA, explicacion: 'debe ser la firma Ed25519 en base64 estándar (88 caracteres)' }),
  };
  for (const r of ['id', 'impresora', 'operaciones']) if (!(r in doc)) falla('', 'required', `falta el campo obligatorio «${r}»`);
  for (const [k, v] of Object.entries(doc)) {
    if (k === 'version' || k === 'operaciones') continue;
    if (!(k in sobre)) falla('', 'additionalProperties', `el campo «${k}» no existe en el contrato`);
    else if (v === null) falla(`/${k}`, 'type', 'null no es válido: omite el campo');
    else sobre[k](v, `/${k}`, falla);
  }
  const firma = ['cliente_id', 'emitido', 'expira', 'firma'];
  const presentes = firma.filter((k) => k in doc);
  if (presentes.length > 0 && presentes.length < 4) {
    falla('', 'dependentRequired', `los campos de firma van juntos: falta ${firma.filter((k) => !(k in doc)).join(', ')}`);
  }

  let codigo = 'DOCUMENTO_INVALIDO';
  const ops = doc.operaciones;
  if ('operaciones' in doc && tipo('array')(ops, '/operaciones', falla)) {
    if (ops.length < 1) falla('/operaciones', 'minItems', 'debe tener al menos una operación');
    if (ops.length > 500) {
      return errorDeDocumento('DEMASIADAS_OPERACIONES', `El documento tiene ${ops.length} operaciones; el máximo es 500.`,
        [{ ruta: '/operaciones', regla: 'maxItems', mensaje: 'debe tener como máximo 500 operaciones' }]);
    }
    ops.forEach((op, i) => {
      const d = validarOperacion(op, `/operaciones/${i}`);
      if (d.some((x) => x.regla === 'enum' && x.ruta.endsWith('/tipo'))) codigo = 'OPERACION_DESCONOCIDA';
      detalles.push(...d);
    });
  }
  if (detalles.length > 0) {
    if (codigo === 'OPERACION_DESCONOCIDA') {
      return errorDeDocumento(codigo, 'El documento usa una operación que no existe en la versión 1.', detalles.filter((d) => d.ruta.endsWith('/tipo')));
    }
    return errorDeDocumento(codigo, 'El documento no cumple el contrato v1.', detalles);
  }

  // Reglas del puente que no dependen de la impresora: se devuelve el código de la primera falla
  // con todos los detalles de ese código, como el puente.
  const fallas = ops.flatMap((op, i) => validarReglas(op, `/operaciones/${i}`));
  if (presentes.length === 4) {
    const vigencia = Date.parse(doc.expira) - Date.parse(doc.emitido);
    if (!(vigencia > 0 && vigencia <= VIGENCIA_MAXIMA_MS)) {
      fallas.push({ codigo: 'FIRMA_INVALIDA', detalle: { ruta: '/expira', regla: 'vigencia_maxima', mensaje: 'expira − emitido debe ser mayor que 0 y de 120 s como máximo' },
        sugerencia: 'Firma cada documento justo antes de enviarlo, con expira = emitido + 120 s o menos.' });
    }
  }
  if (fallas.length > 0) {
    const primera = fallas[0];
    const mismas = fallas.filter((f) => f.codigo === primera.codigo);
    return errorDeDocumento(primera.codigo, mismas.length === 1 ? primera.detalle.mensaje ?? '' : 'El documento no cumple las reglas del contrato v1.',
      mismas.map((f) => f.detalle), primera.sugerencia);
  }

  const bytes = bytesUTF8(JSON.stringify(doc));
  if (bytes > MAX_BYTES_DOCUMENTO) {
    return new PrismaError({ codigo: 'DOCUMENTO_DEMASIADO_GRANDE', mensaje: `El documento mide ${bytes} bytes; el máximo es 262144 (256 KB).`,
      sugerencia: 'Reduce las imágenes en línea o guárdalas en el puente y usa "recurso".' });
  }
  return null;
}

// Los mensajes de validarReglas ya son frases completas; los del esquema se arman en el error.
function exigirValida(doc) {
  const e = validarDocumento(doc);
  if (e) throw e;
}

// ---------------------------------------------------------------------------------------------
// Constructor fluido de documentos
// ---------------------------------------------------------------------------------------------

const sinIndefinidos = (o) => Object.fromEntries(Object.entries(o).filter(([, v]) => v !== undefined));

/**
 * Arma un documento v1 operación por operación. Cada método valida su operación al agregarla, así
 * que un error apunta a la línea que lo causó.
 */
export class Documento {
  constructor(impresora, { id, copias, referencia } = {}) {
    this._sobre = sinIndefinidos({ version: VERSION_DOCUMENTO, id: id ?? generarId(), impresora, copias, referencia });
    this._operaciones = [];
  }

  /** Id (UUID v4) del documento: se reutiliza en los reintentos. */
  get id() { return this._sobre.id; }

  copias(n) { this._sobre.copias = n; return this; }

  referencia(texto) { this._sobre.referencia = texto; return this; }

  /** Agrega una operación tal como la define el contrato, por ejemplo {tipo: "salto"}. */
  operacion(op) {
    const ruta = `/operaciones/${this._operaciones.length}`;
    const limpia = esObjeto(op) ? sinIndefinidos(op) : op;
    const detalles = validarOperacion(limpia, ruta);
    if (detalles.length > 0) {
      const desconocida = detalles.some((d) => d.ruta.endsWith('/tipo'));
      throw errorDeDocumento(desconocida ? 'OPERACION_DESCONOCIDA' : 'DOCUMENTO_INVALIDO',
        `Operación ${esObjeto(op) && op.tipo ? `«${op.tipo}» ` : ''}inválida: ${detalles.map((d) => `${d.ruta} ${d.mensaje}`).join('; ')}.`, detalles);
    }
    const [falla] = validarReglas(limpia, ruta);
    if (falla) throw errorDeDocumento(falla.codigo, falla.detalle.mensaje, [falla.detalle], falla.sugerencia);
    this._operaciones.push(limpia);
    return this;
  }

  texto(valor, opciones = {}) { return this.operacion({ tipo: 'texto', valor, ...opciones }); }

  separador(caracter) { return this.operacion({ tipo: 'separador', caracter }); }

  /** Columnas: textos o {texto, ancho, alineacion}. */
  fila(columnas, opciones = {}) {
    const cols = Array.isArray(columnas) ? columnas.map((c) => (typeof c === 'string' ? { texto: c } : esObjeto(c) ? sinIndefinidos(c) : c)) : columnas;
    return this.operacion({ tipo: 'fila', columnas: cols, ...opciones });
  }

  par(izquierda, derecha, opciones = {}) { return this.operacion({ tipo: 'par', izquierda, derecha, ...opciones }); }

  salto(lineas) { return this.operacion({ tipo: 'salto', lineas }); }

  qr(valor, opciones = {}) { return this.operacion({ tipo: 'qr', valor, ...opciones }); }

  codigoBarras(valor, simbologia, opciones = {}) { return this.operacion({ tipo: 'codigo_barras', valor, simbologia, ...opciones }); }

  /**
   * Imagen: el nombre de una imagen guardada en el puente ("logo-tienda"), un PNG en base64 (con o
   * sin prefijo data:) o los bytes del PNG (Uint8Array o ArrayBuffer).
   */
  imagen(fuente, opciones = {}) {
    let campo;
    if (typeof fuente === 'string') {
      const sinPrefijo = fuente.replace(/^data:image\/png;base64,/, '');
      campo = sinPrefijo.startsWith('iVBOR') ? { base64: sinPrefijo } : { recurso: fuente };
    } else if (fuente instanceof Uint8Array || fuente instanceof ArrayBuffer) {
      campo = { base64: aBase64(fuente) };
    } else {
      campo = fuente; // {recurso} o {base64}
    }
    return this.operacion({ tipo: 'imagen', ...campo, ...opciones });
  }

  cortar(opciones = {}) { return this.operacion({ tipo: 'cortar', ...opciones }); }

  /** Operación sensible: el dueño del puente debe habilitarla para tu aplicación. */
  abrirCajon(pin) { return this.operacion({ tipo: 'abrir_cajon', pin }); }

  /** Operación sensible: bytes ESC/POS tal cual (base64 o Uint8Array). */
  raw(datos) { return this.operacion({ tipo: 'raw', base64: typeof datos === 'string' ? datos : aBase64(datos) }); }

  /** Devuelve el documento como objeto JSON, validado completo. */
  json() {
    const doc = { ...this._sobre, operaciones: [...this._operaciones] };
    exigirValida(doc);
    return doc;
  }

  toJSON() { return this.json(); }
}

// ---------------------------------------------------------------------------------------------
// Cliente
// ---------------------------------------------------------------------------------------------

const CODIGOS_CERTIFICADO = new Set([
  'DEPTH_ZERO_SELF_SIGNED_CERT', 'SELF_SIGNED_CERT_IN_CHAIN', 'UNABLE_TO_VERIFY_LEAF_SIGNATURE', 'UNABLE_TO_GET_ISSUER_CERT_LOCALLY',
  'UNABLE_TO_GET_ISSUER_CERT', 'CERT_HAS_EXPIRED', 'CERT_NOT_YET_VALID', 'ERR_TLS_CERT_ALTNAME_INVALID', 'CERT_UNTRUSTED',
]);

const esNavegador = () => typeof window !== 'undefined' && typeof window.document !== 'undefined';
const esLoopback = (host) => /^(localhost|127(\.\d+){3}|\[::1\])$/i.test(host);
const dormir = (ms) => new Promise((r) => setTimeout(r, ms));
const urlsAlcanzadas = new Set(); // puentes que ya respondieron desde esta página

/**
 * Cliente de un puente PRISMA.
 *
 *   const prisma = new Prisma({ url: "https://192.168.1.10:8443", llave: "pk_..." });
 *   await prisma.imprimir("caja", (d) => d.texto("Hola").cortar());
 */
export class Prisma {
  constructor({ url, llave, reintentos = 2, espera_reintento_ms = 1000, tiempo_espera_ms = 75000, fetch: fetchPropio } = {}) {
    let base;
    try {
      base = new URL(url);
    } catch {
      throw new PrismaError({ codigo: 'CONFIGURACION_INVALIDA', mensaje: `La URL del puente no es válida: ${url}.`,
        sugerencia: 'Usa la dirección completa del puente, por ejemplo https://192.168.1.10:8443.' });
    }
    if (llave !== undefined && !/^pk_/.test(llave)) {
      throw new PrismaError({ codigo: 'CONFIGURACION_INVALIDA', mensaje: 'La llave de cliente debe empezar con pk_.',
        sugerencia: 'Copia la llave completa que mostró el panel del puente al crear la aplicación.' });
    }
    this.url = base.origin;
    this._llave = llave;
    this._reintentos = reintentos;
    this._espera = espera_reintento_ms;
    this._tiempo = tiempo_espera_ms;
    this._fetch = fetchPropio ?? ((...a) => globalThis.fetch(...a));
  }

  /** GET /v1/salud: {ok, version, nombre_puente}. No necesita autorización. */
  async salud() { return this._pedir('GET', '/v1/salud'); }

  /** GET /v1/impresoras: las impresoras que tu aplicación puede usar, con su estado. */
  async impresoras() { return this._pedir('GET', '/v1/impresoras'); }

  /** Crea un documento para la impresora sin enviarlo. */
  documento(impresora, opciones) { return new Documento(impresora, opciones); }

  /**
   * Imprime. `contenido` es una función que recibe el Documento y le agrega operaciones, un
   * Documento ya armado o un arreglo de operaciones del contrato.
   */
  async imprimir(impresora, contenido, opciones) {
    return this._pedir('POST', '/v1/imprimir', { cuerpo: this._armar(impresora, contenido, opciones) });
  }

  /** Imprime un documento ya completo, por ejemplo uno firmado por tu backend (nivel 3). No lo modifica. */
  async imprimirDocumento(documento) {
    const doc = documento instanceof Documento ? documento.json() : documento;
    exigirValida(doc);
    return this._pedir('POST', '/v1/imprimir', { cuerpo: doc });
  }

  /** Devuelve el HTML de la simulación del ticket en esa impresora, sin imprimir. */
  async previsualizar(impresora, contenido, opciones) {
    const doc = esObjeto(impresora) || impresora instanceof Documento ? (impresora instanceof Documento ? impresora.json() : impresora)
      : this._armar(impresora, contenido, opciones);
    exigirValida(doc);
    return this._pedir('POST', '/v1/previsualizar', { cuerpo: doc, html: true });
  }

  /**
   * Comprueba la conexión con el puente y, si falla, explica por qué. Nunca lanza:
   * devuelve {ok: true, salud} o {ok: false, error}.
   */
  async diagnosticar() {
    try {
      const salud = await this._pedir('GET', '/v1/salud', { reintentos: 0 });
      return { ok: true, salud };
    } catch (error) {
      return { ok: false, error };
    }
  }

  _armar(impresora, contenido, opciones = {}) {
    if (contenido instanceof Documento) return contenido.json();
    const doc = new Documento(impresora, opciones);
    if (typeof contenido === 'function') {
      contenido(doc);
    } else if (Array.isArray(contenido)) {
      contenido.forEach((op) => doc.operacion(op));
    } else {
      throw new PrismaError({ codigo: 'CONFIGURACION_INVALIDA', mensaje: 'El contenido debe ser una función, un Documento o un arreglo de operaciones.',
        sugerencia: 'Por ejemplo: prisma.imprimir("caja", (d) => d.texto("Hola").cortar()).' });
    }
    return doc.json();
  }

  async _pedir(metodo, ruta, { cuerpo, html = false, reintentos = this._reintentos } = {}) {
    this._revisarContenidoMixto();
    const cabeceras = { Accept: 'application/json' };
    if (this._llave) cabeceras.Authorization = `Bearer ${this._llave}`;
    if (cuerpo !== undefined) cabeceras['Content-Type'] = 'application/json';
    // El cuerpo se serializa una sola vez: los reintentos envían exactamente el mismo documento (mismo id).
    const body = cuerpo === undefined ? undefined : JSON.stringify(cuerpo);

    for (let intento = 0; ; intento++) {
      let respuesta;
      try {
        respuesta = await this._conTiempo(this.url + ruta, { method: metodo, headers: cabeceras, body, mode: 'cors', credentials: 'omit', cache: 'no-store' });
      } catch (e) {
        const error = await this._errorDeRed(e);
        if (!error.reintentable || intento >= reintentos) throw error;
        await dormir(this._espera * 3 ** intento);
        continue;
      }
      urlsAlcanzadas.add(this.url);
      return this._leer(respuesta, html);
    }
  }

  async _conTiempo(url, init) {
    const control = new AbortController();
    const reloj = setTimeout(() => control.abort(), this._tiempo);
    try {
      return await this._fetch(url, { ...init, signal: control.signal });
    } finally {
      clearTimeout(reloj);
    }
  }

  async _leer(respuesta, html) {
    const tipoContenido = respuesta.headers.get('Content-Type') || '';
    if (tipoContenido.includes('application/json')) {
      let datos;
      try {
        datos = await respuesta.json();
      } catch (causa) {
        throw this._respuestaInvalida(respuesta.status, causa);
      }
      if (!respuesta.ok || (datos && datos.ok === false)) {
        throw new PrismaError({ ...datos, status: respuesta.status, respuesta: datos });
      }
      return datos;
    }
    if (html && respuesta.ok && tipoContenido.includes('text/html')) return respuesta.text();
    throw this._respuestaInvalida(respuesta.status);
  }

  _respuestaInvalida(status, causa) {
    return new PrismaError({ codigo: 'RESPUESTA_INVALIDA', status, causa,
      mensaje: `${this.url} respondió algo que no es un puente PRISMA (HTTP ${status}).`,
      sugerencia: 'Revisa que la URL apunte al puente (https://<ip>:8443) y no a otro servicio.' });
  }

  // Una página https:// no puede llamar a http:// salvo en localhost: el navegador lo bloquea sin avisar.
  _revisarContenidoMixto() {
    if (!esNavegador() || globalThis.location?.protocol !== 'https:') return;
    const destino = new URL(this.url);
    if (destino.protocol === 'http:' && !esLoopback(destino.hostname)) {
      throw new PrismaError({ codigo: 'CONTENIDO_MIXTO', mensaje: `Esta página usa https:// y el puente ${this.url} usa http://; el navegador bloquea esa mezcla.`,
        sugerencia: 'Usa la dirección https:// del puente (modo_tls "autofirmado" o "archivos").' });
    }
  }

  // Convierte una falla de fetch (sin respuesta HTTP) en un error con la causa más probable.
  async _errorDeRed(e) {
    if (e?.name === 'AbortError') {
      const error = new PrismaError({ codigo: 'PUENTE_SIN_RESPUESTA', causa: e,
        mensaje: `El puente ${this.url} no respondió en ${Math.round(this._tiempo / 1000)} s.`,
        sugerencia: 'Revisa la red. Puedes reintentar sin miedo: el mismo documento no se imprime dos veces.' });
      error.reintentable = true;
      return error;
    }
    return esNavegador() ? this._diagnosticarNavegador(e) : this._diagnosticarNode(e);
  }

  _diagnosticarNode(e) {
    const codigo = e?.cause?.code ?? e?.cause?.cause?.code ?? e?.code ?? '';
    if (CODIGOS_CERTIFICADO.has(codigo)) {
      return new PrismaError({ codigo: 'CERTIFICADO_NO_CONFIABLE', causa: e,
        mensaje: `No se confía en el certificado de ${this.url} (${codigo}).`,
        sugerencia: 'Agrega el certificado del puente con NODE_EXTRA_CA_CERTS=prisma.crt (se descarga en el panel, Ajustes → HTTPS).' });
    }
    const error = this._inalcanzable(e, `No se pudo conectar con el puente en ${this.url}${codigo ? ` (${codigo})` : ''}.`,
      'Revisa que el puente esté encendido, que la IP y el puerto sean correctos y que este equipo esté en la misma red.', null);
    error.reintentable = true;
    return error;
  }

  async _diagnosticarNavegador(e) {
    if (globalThis.navigator?.onLine === false) {
      const error = this._inalcanzable(e, 'Este equipo no tiene conexión a la red.', 'Conéctate a la red del negocio y vuelve a intentarlo.', null);
      error.reintentable = true;
      return error;
    }
    if (await this._permisoRedLocalDenegado()) {
      return new PrismaError({ codigo: 'PERMISO_RED_LOCAL_DENEGADO', causa: e,
        mensaje: 'El navegador no permite que esta página se conecte a equipos de la red local.',
        sugerencia: 'Haz clic en el candado junto a la dirección de la página y permite el acceso a la red local; luego recarga.' });
    }
    // Una petición sin CORS solo dice si el puente responde. Si responde, lo que falló fue CORS:
    // el puente no reconoce el origen de esta página.
    if (await this._responde()) {
      const origen = globalThis.location?.origin ?? 'esta página';
      return new PrismaError({ codigo: 'ORIGEN_NO_AUTORIZADO', causa: e,
        mensaje: `El puente responde, pero no ha autorizado el origen ${origen}.`,
        sugerencia: 'El dueño del puente debe aprobar este origen en el panel (Solicitudes): ya quedó registrado como pendiente. Si la aplicación ya estaba aprobada, puede estar desactivada.' });
    }
    if (urlsAlcanzadas.has(this.url)) {
      const error = this._inalcanzable(e, `El puente ${this.url} dejó de responder.`,
        'Revisa que el equipo del puente siga encendido y conectado a la red.', ['PUENTE_APAGADO_O_FUERA_DE_RED']);
      error.reintentable = true;
      return error;
    }
    // El navegador no distingue un certificado no confiable de un equipo apagado: se dan las dos causas.
    const error = this._inalcanzable(e, `No se pudo conectar con el puente en ${this.url}. Si es la primera vez desde este equipo, lo más probable es que el navegador no confíe en su certificado.`,
      `Abre ${this.url}/v1/salud en una pestaña: si aparece una advertencia de certificado, instala el certificado del puente (docs/instalacion/https-y-certificados.md); si la página no carga, revisa que el puente esté encendido y en la misma red.`,
      ['CERTIFICADO_NO_CONFIABLE', 'PUENTE_APAGADO_O_FUERA_DE_RED']);
    // Sin reintentos: lo más probable es el certificado, que no se arregla esperando, y así la
    // persona ve el diagnóstico de inmediato en vez de tras unos segundos.
    return error;
  }

  _inalcanzable(causa, mensaje, sugerencia, causas_posibles) {
    return new PrismaError({ codigo: 'PUENTE_INALCANZABLE', causa, mensaje, sugerencia, causas_posibles: causas_posibles ?? undefined });
  }

  async _permisoRedLocalDenegado() {
    for (const name of ['local-network-access', 'private-network-access']) {
      try {
        const estado = await globalThis.navigator.permissions.query({ name });
        if (estado.state === 'denied') return true;
      } catch {
        // permiso desconocido en este navegador
      }
    }
    return false;
  }

  async _responde() {
    try {
      await this._conTiempoCorto(`${this.url}/v1/salud`, { mode: 'no-cors', cache: 'no-store', credentials: 'omit' });
      return true;
    } catch {
      return false;
    }
  }

  async _conTiempoCorto(url, init) {
    const control = new AbortController();
    const reloj = setTimeout(() => control.abort(), 5000);
    try {
      return await this._fetch(url, { ...init, signal: control.signal });
    } finally {
      clearTimeout(reloj);
    }
  }
}

// Para el build UMD y CommonJS: todo cuelga de la clase Prisma.
Object.assign(Prisma, { Prisma, Documento, PrismaError, validarDocumento, generarId, aBase64, pngABase64, VERSION_SDK, default: Prisma });

export default Prisma;
