// Comprobación de tipos (npm run tipos): debe compilar con --strict. No se ejecuta.
import Prisma, { Documento, PrismaError, validarDocumento, type DocumentoV1, type ResultadoImpresion } from '../dist/prisma.mjs';

async function ejemplo(): Promise<void> {
  const prisma = new Prisma({ url: 'https://192.168.1.10:8443', llave: 'pk_...' });
  const salud = await prisma.salud();
  const nombre: string = salud.nombre_puente;
  const impresoras = await prisma.impresoras();
  const enLinea = impresoras.filter((i) => i.estado === 'en_linea').map((i) => i.alias);

  const r: ResultadoImpresion = await prisma.imprimir('caja', (doc) => doc
    .texto('MI NEGOCIO', { alineacion: 'centro', negrita: true, tamano: [2, 2] })
    .separador()
    .fila(['1', { texto: 'Taco', ancho: 'auto' }, { texto: '$30', ancho: 25, alineacion: 'derecha' }])
    .par('TOTAL', '$60.00', { negrita: true })
    .codigoBarras('4006381333931', 'EAN13')
    .imagen('logo-tienda', { ancho_max: 60 })
    .qr('https://minegocio.com/p/812', { correccion: 'H' })
    .cortar({ parcial: true }), { referencia: 'VTA-1' });
  const id: string = r.trabajo_id;

  const doc: DocumentoV1 = new Documento('caja').texto('x').json();
  await prisma.imprimirDocumento(doc);
  const html: string = await prisma.previsualizar(doc);

  try {
    await prisma.imprimir('caja', [{ tipo: 'texto', valor: 'Hola' }, { tipo: 'cortar' }]);
  } catch (e) {
    if (e instanceof PrismaError && e.codigo === 'ORIGEN_NO_AUTORIZADO') console.log(e.sugerencia, e.status);
  }
  const d = await prisma.diagnosticar();
  if (!d.ok) console.log(d.error.codigo);
  const err = validarDocumento({});
  if (err) console.log(err.detalles?.[0]?.ruta);

  // @ts-expect-error alineación inexistente
  new Documento('caja').texto('x', { alineacion: 'center' });
  // @ts-expect-error la imagen necesita base64 o recurso, no ambos
  const mala: DocumentoV1['operaciones'][number] = { tipo: 'imagen', base64: 'iVBOR', recurso: 'logo' };
  console.log(nombre, enLinea, id, html, mala);
}
void ejemplo;
