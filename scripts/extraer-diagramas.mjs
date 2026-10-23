// Extrae cada bloque ```mermaid de los Markdown del repositorio a archivos .mmd en la carpeta
// indicada, y escribe en ella indice.tsv (archivo .mmd → Markdown y línea de origen).
// Lo usan scripts/validar-diagramas.sh y scripts/validar-diagramas.ps1.
import { mkdirSync, readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { join, relative, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const raiz = join(fileURLToPath(new URL('.', import.meta.url)), '..');
const salida = process.argv[2];
if (!salida) {
  console.error('uso: node scripts/extraer-diagramas.mjs <carpeta-de-salida>');
  process.exit(2);
}
mkdirSync(salida, { recursive: true });

const ignorar = new Set(['.git', 'node_modules', 'dist', 'key', '__pycache__']);
function* markdowns(dir) {
  for (const e of readdirSync(dir, { withFileTypes: true })) {
    if (e.isDirectory()) {
      if (!ignorar.has(e.name)) yield* markdowns(join(dir, e.name));
    } else if (e.name.endsWith('.md')) {
      yield join(dir, e.name);
    }
  }
}

const indice = [];
for (const archivo of markdowns(raiz)) {
  const lineas = readFileSync(archivo, 'utf8').split('\n');
  const rel = relative(raiz, archivo).split(sep).join('/');
  let bloque = null;
  let inicio = 0;
  let n = 0;
  lineas.forEach((linea, i) => {
    if (bloque === null && linea.trim() === '```mermaid') {
      bloque = [];
      inicio = i + 1;
    } else if (bloque !== null && linea.trim() === '```') {
      const nombre = `${rel.replace(/[/.]/g, '_')}-${n++}.mmd`;
      writeFileSync(join(salida, nombre), bloque.join('\n') + '\n');
      indice.push(`${nombre}\t${rel}:${inicio}`);
      bloque = null;
    } else if (bloque !== null) {
      bloque.push(linea);
    }
  });
}
writeFileSync(join(salida, 'indice.tsv'), indice.join('\n') + '\n');
console.log(`${indice.length} diagramas extraídos`);
