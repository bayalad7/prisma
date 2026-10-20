/*! @bayalad7/prisma 0.1.0 · SDK de PRISMA · https://github.com/bayalad7/prisma/tree/main/sdk/js */
// Tipos del SDK de PRISMA. Los campos siguen el contrato «documento v1» (español, snake_case).

export declare const VERSION_SDK: string;

export type Alineacion = 'izquierda' | 'centro' | 'derecha';
export type Simbologia = 'CODE128' | 'CODE39' | 'EAN13' | 'EAN8' | 'UPCA' | 'ITF';

// --- Operaciones del contrato -------------------------------------------------------------------

export interface OpcionesTexto {
  alineacion?: Alineacion;
  negrita?: boolean;
  subrayado?: boolean;
  invertido?: boolean;
  /** [ancho, alto], cada uno de 1 a 8. */
  tamano?: [number, number];
  fuente?: 'A' | 'B';
}
export interface OperacionTexto extends OpcionesTexto { tipo: 'texto'; valor: string }
export interface OperacionSeparador { tipo: 'separador'; caracter?: string }
export interface Columna {
  texto: string;
  /** Porcentaje entero de 1 a 100, o "auto". */
  ancho?: number | 'auto';
  alineacion?: Alineacion;
}
export interface OperacionFila { tipo: 'fila'; columnas: Columna[]; negrita?: boolean }
export interface OperacionPar { tipo: 'par'; izquierda: string; derecha: string; negrita?: boolean }
export interface OperacionSalto { tipo: 'salto'; lineas?: number }
export interface OpcionesQR {
  /** Tamaño del módulo en puntos, de 1 a 16 (6 por defecto). */
  tamano?: number;
  correccion?: 'L' | 'M' | 'Q' | 'H';
  alineacion?: Alineacion;
}
export interface OperacionQR extends OpcionesQR { tipo: 'qr'; valor: string }
export interface OpcionesCodigoBarras {
  texto?: 'arriba' | 'abajo' | 'ninguno';
  /** Alto en puntos, de 1 a 255 (80 por defecto). */
  alto?: number;
  alineacion?: Alineacion;
}
export interface OperacionCodigoBarras extends OpcionesCodigoBarras { tipo: 'codigo_barras'; valor: string; simbologia: Simbologia }
export interface OpcionesImagen {
  alineacion?: Alineacion;
  /** Ancho máximo como porcentaje del área imprimible, de 1 a 100. */
  ancho_max?: number;
}
export type OperacionImagen = OpcionesImagen & { tipo: 'imagen' } & ({ base64: string; recurso?: never } | { recurso: string; base64?: never });
export interface OperacionCortar { tipo: 'cortar'; parcial?: boolean }
export interface OperacionAbrirCajon { tipo: 'abrir_cajon'; pin?: 2 | 5 }
export interface OperacionRaw { tipo: 'raw'; base64: string }

export type Operacion =
  | OperacionTexto | OperacionSeparador | OperacionFila | OperacionPar | OperacionSalto | OperacionQR
  | OperacionCodigoBarras | OperacionImagen | OperacionCortar | OperacionAbrirCajon | OperacionRaw;

/** Documento v1 tal como viaja al puente. */
export interface DocumentoV1 {
  version: '1';
  /** UUID v4 en minúsculas; se reutiliza en los reintentos de ese mismo ticket. */
  id: string;
  /** Alias de la impresora en el puente, por ejemplo "caja". */
  impresora: string;
  copias?: number;
  referencia?: string;
  operaciones: Operacion[];
  // Nivel 3: los cuatro van juntos.
  cliente_id?: string;
  emitido?: string;
  expira?: string;
  firma?: string;
}

// --- Errores ------------------------------------------------------------------------------------

export interface DetalleError {
  /** JSON Pointer del campo, por ejemplo "/operaciones/3/alineacion". */
  ruta: string;
  regla: string;
  mensaje: string;
}

/** Códigos que genera el propio SDK (status 0), además de los que responde el puente. */
export type CodigoErrorSDK =
  | 'CONFIGURACION_INVALIDA'
  | 'CERTIFICADO_NO_CONFIABLE'
  | 'PERMISO_RED_LOCAL_DENEGADO'
  | 'PUENTE_INALCANZABLE'
  | 'PUENTE_SIN_RESPUESTA'
  | 'ORIGEN_NO_AUTORIZADO'
  | 'CONTENIDO_MIXTO'
  | 'RESPUESTA_INVALIDA';

export declare class PrismaError extends Error {
  /** Código estable en MAYÚSCULAS, por ejemplo "IMPRESORA_NO_ENCONTRADA". */
  readonly codigo: CodigoErrorSDK | (string & {});
  readonly mensaje: string;
  readonly sugerencia: string;
  /** Estado HTTP de la respuesta del puente; 0 si el error se detectó antes o no hubo respuesta. */
  readonly status: number;
  readonly detalles?: DetalleError[];
  readonly trabajo_id?: string;
  /** Solo en PUENTE_INALCANZABLE desde el navegador, que no distingue certificado de equipo apagado. */
  readonly causas_posibles?: Array<'CERTIFICADO_NO_CONFIABLE' | 'PUENTE_APAGADO_O_FUERA_DE_RED'>;
  /** Cuerpo JSON tal como lo respondió el puente. */
  readonly respuesta?: Record<string, unknown>;
  /** true si el SDK lo considera un error de red transitorio (lo reintenta). */
  readonly reintentable: boolean;
}

// --- Respuestas ---------------------------------------------------------------------------------

export interface Salud { ok: true; version: string; nombre_puente: string }

export interface Impresora {
  alias: string;
  nombre: string;
  ancho_mm: number;
  estado: 'en_linea' | 'fuera_de_linea' | 'ocupada' | 'desconocido';
}

export interface Advertencia { codigo: string; mensaje?: string }

export interface ResultadoImpresion {
  ok: true;
  trabajo_id: string;
  impresora: string;
  advertencias: Advertencia[];
  /** Presente cuando la impresora terminó. */
  duracion_ms?: number;
  /** true si el puente ya había recibido este mismo documento (no se volvió a imprimir). */
  duplicado?: boolean;
  /** "encolado" cuando el puente respondió 202: se imprimirá en cuanto la impresora lo atienda. */
  estado?: 'encolado' | 'imprimiendo' | 'completado' | 'fallido';
}

export type Diagnostico = { ok: true; salud: Salud } | { ok: false; error: PrismaError };

// --- Constructor de documentos ------------------------------------------------------------------

export interface OpcionesDocumento {
  /** UUID v4; si se omite se genera uno. */
  id?: string;
  copias?: number;
  referencia?: string;
}

export declare class Documento {
  constructor(impresora: string, opciones?: OpcionesDocumento);
  readonly id: string;
  copias(n: number): this;
  referencia(texto: string): this;
  operacion(op: Operacion): this;
  texto(valor: string, opciones?: OpcionesTexto): this;
  separador(caracter?: string): this;
  fila(columnas: Array<string | Columna>, opciones?: { negrita?: boolean }): this;
  par(izquierda: string, derecha: string, opciones?: { negrita?: boolean }): this;
  salto(lineas?: number): this;
  qr(valor: string, opciones?: OpcionesQR): this;
  codigoBarras(valor: string, simbologia: Simbologia, opciones?: OpcionesCodigoBarras): this;
  /** Nombre de una imagen guardada en el puente, PNG en base64 (con o sin data:) o los bytes del PNG. */
  imagen(fuente: string | Uint8Array | ArrayBuffer | { recurso: string } | { base64: string }, opciones?: OpcionesImagen): this;
  cortar(opciones?: { parcial?: boolean }): this;
  /** Operación sensible: debe estar habilitada para tu aplicación. */
  abrirCajon(pin?: 2 | 5): this;
  /** Operación sensible: bytes ESC/POS tal cual. */
  raw(datos: string | Uint8Array | ArrayBuffer): this;
  /** El documento validado, listo para JSON.stringify. */
  json(): DocumentoV1;
  toJSON(): DocumentoV1;
}

export type Contenido = ((doc: Documento) => unknown) | Documento | Operacion[];

// --- Cliente ------------------------------------------------------------------------------------

export interface OpcionesPrisma {
  /** Dirección del puente, por ejemplo "https://192.168.1.10:8443". */
  url: string;
  /** Llave de cliente (pk_…), para el nivel 2 o el nivel 3 que la exige. */
  llave?: string;
  /** Reintentos ante errores de red (2 por defecto). Nunca se reintentan errores del puente. */
  reintentos?: number;
  /** Espera antes del primer reintento; se triplica en cada uno (1000 ms por defecto). */
  espera_reintento_ms?: number;
  /** Tiempo máximo por petición (75000 ms por defecto; el puente responde 202 a los 60 s). */
  tiempo_espera_ms?: number;
  /** fetch propio, por ejemplo uno con un agente que confíe en el certificado del puente. */
  fetch?: typeof globalThis.fetch;
}

export declare class Prisma {
  constructor(opciones: OpcionesPrisma);
  /** Origen del puente, sin barra final. */
  readonly url: string;
  salud(): Promise<Salud>;
  impresoras(): Promise<Impresora[]>;
  documento(impresora: string, opciones?: OpcionesDocumento): Documento;
  imprimir(impresora: string, contenido: Contenido, opciones?: OpcionesDocumento): Promise<ResultadoImpresion>;
  /** Imprime un documento completo sin modificarlo (por ejemplo, firmado por tu backend). */
  imprimirDocumento(documento: DocumentoV1 | Documento): Promise<ResultadoImpresion>;
  previsualizar(impresora: string, contenido: Contenido, opciones?: OpcionesDocumento): Promise<string>;
  previsualizar(documento: DocumentoV1 | Documento): Promise<string>;
  /** Nunca lanza: explica por qué no hay conexión con el puente. */
  diagnosticar(): Promise<Diagnostico>;

  static readonly Prisma: typeof Prisma;
  static readonly Documento: typeof Documento;
  static readonly PrismaError: typeof PrismaError;
  static readonly validarDocumento: typeof validarDocumento;
  static readonly generarId: typeof generarId;
  static readonly aBase64: typeof aBase64;
  static readonly pngABase64: typeof pngABase64;
  static readonly VERSION_SDK: string;
}

/** Valida un documento v1 como el puente; null si es válido. */
export declare function validarDocumento(documento: unknown): PrismaError | null;
/** UUID v4 en minúsculas. */
export declare function generarId(): string;
export declare function aBase64(datos: Uint8Array | ArrayBuffer): string;
/** Lee un PNG (File, Blob o bytes) y lo devuelve en base64 para la operación imagen. */
export declare function pngABase64(archivo: Blob | Uint8Array | ArrayBuffer): Promise<string>;

export default Prisma;
