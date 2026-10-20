// Genera dist/ a partir de src/: el módulo ES, el build UMD (navegador con <script>, CommonJS y
// AMD) y los tipos. Sin dependencias. Con --verificar falla si dist/ no coincide con src/.
import { readFileSync, writeFileSync, mkdirSync, existsSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const raiz = join(dirname(fileURLToPath(import.meta.url)), '..');
const leer = (ruta) => readFileSync(join(raiz, ruta), 'utf8').replace(/\r\n/g, '\n');
const paquete = JSON.parse(leer('package.json'));
const fuente = leer('src/prisma.mjs');

if (!fuente.includes(`export const VERSION_SDK = '${paquete.version}';`)) {
  console.error(`VERSION_SDK en src/prisma.mjs no coincide con package.json (${paquete.version}).`);
  process.exit(1);
}

const cabecera = `/*! ${paquete.name} ${paquete.version} · SDK de PRISMA · ${paquete.homepage} */\n`;

// El build UMD es el mismo código sin las palabras export: todo queda colgado de la clase Prisma.
const cuerpo = fuente
  .replace(/^export default Prisma;\n/m, '')
  .replace(/^export (?=(class|function|async function|const) )/gm, '');
if (/^export /m.test(cuerpo)) {
  console.error('src/prisma.mjs tiene una exportación que el build UMD no sabe convertir.');
  process.exit(1);
}
const umd = `${cabecera}(function (raiz, fabrica) {
  if (typeof define === 'function' && define.amd) define([], fabrica);
  else if (typeof module === 'object' && module.exports) module.exports = fabrica();
  else raiz.Prisma = fabrica();
})(typeof globalThis !== 'undefined' ? globalThis : this, function () {
'use strict';
${cuerpo}
return Prisma;
});
`;

const archivos = {
  'dist/prisma.mjs': cabecera + fuente,
  'dist/prisma.umd.js': umd,
  // Un archivo de tipos por formato: TypeScript busca los tipos junto a cada archivo: .d.mts para .mjs y .umd.d.ts para .umd.js.
  'dist/prisma.d.mts': cabecera + leer('src/prisma.d.ts'),
  'dist/prisma.umd.d.ts': cabecera + leer('src/prisma.d.ts'),
};

if (process.argv.includes('--verificar')) {
  const distintos = Object.entries(archivos).filter(([ruta, contenido]) => !existsSync(join(raiz, ruta)) || leer(ruta) !== contenido);
  if (distintos.length > 0) {
    console.error(`dist/ está desactualizado (${distintos.map(([r]) => r).join(', ')}): ejecuta «npm run construir».`);
    process.exit(1);
  }
  console.log('dist/ está al día.');
} else {
  mkdirSync(join(raiz, 'dist'), { recursive: true });
  for (const [ruta, contenido] of Object.entries(archivos)) writeFileSync(join(raiz, ruta), contenido);
  console.log(`Generados: ${Object.keys(archivos).join(', ')}`);
}
