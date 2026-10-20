// Comprobación de tipos para quien usa require (npm run tipos). No se ejecuta.
import { Prisma, Documento } from '../dist/prisma.umd.js';

const prisma = new Prisma({ url: 'https://192.168.1.10:8443' });
void prisma.imprimir('caja', new Documento('caja').texto('Hola').cortar());
