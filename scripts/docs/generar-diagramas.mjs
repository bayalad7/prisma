// Genera los diagramas que muestra el panel desde docs/diagramas/NN-*.md (la única fuente): para
// cada bloque mermaid, un SVG claro y uno oscuro, saneados y comprimidos con gzip, y el manifiesto
// internal/admin/ayuda/diagramas/manifiesto.json (metadatos del front-matter, hash del bloque y
// archivos). Lo usan los mantenedores con Node 18+; `go build` no lo necesita, porque los SVG se
// versionan. TestDiagramasVigentes falla si un bloque cambió y no se volvió a generar.
//
// Uso (desde la raíz del repositorio): node scripts/docs/generar-diagramas.mjs
// (o scripts/docs/generar-diagramas.sh / .ps1). Descarga @mermaid-js/mermaid-cli con npx.
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import zlib from 'node:zlib';

const MERMAID_CLI = '@mermaid-js/mermaid-cli@11.4.2';
const raiz = process.cwd();
const origen = path.join(raiz, 'docs', 'diagramas');
const destino = path.join(raiz, 'internal', 'admin', 'ayuda', 'diagramas');
const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'prisma-diagramas-'));

// Tema de Mermaid con los tokens del panel (panel.css). htmlLabels desactivado: sin foreignObject.
const fuente = 'system-ui, -apple-system, "Segoe UI", Roboto, sans-serif';
const temas = {
  claro: { background: '#ffffff', primaryColor: '#e6f4ea', primaryTextColor: '#141922', primaryBorderColor: '#2f7d32',
    lineColor: '#5b6573', textColor: '#141922', mainBkg: '#e6f4ea', noteBkgColor: '#fff4e0', noteTextColor: '#3d2400',
    actorBkg: '#ffffff', actorBorder: '#5b6573', actorTextColor: '#141922', signalColor: '#141922', signalTextColor: '#141922',
    labelBoxBkgColor: '#ffffff', labelTextColor: '#141922', fontFamily: fuente, fontSize: '14px' },
  oscuro: { background: '#161a22', primaryColor: '#16301f', primaryTextColor: '#e6e9ef', primaryBorderColor: '#5fbf6b',
    lineColor: '#a0a9b8', textColor: '#e6e9ef', mainBkg: '#16301f', noteBkgColor: '#3a2a12', noteTextColor: '#ffe7c2',
    actorBkg: '#1e232d', actorBorder: '#a0a9b8', actorTextColor: '#e6e9ef', signalColor: '#e6e9ef', signalTextColor: '#e6e9ef',
    labelBoxBkgColor: '#1e232d', labelTextColor: '#e6e9ef', edgeLabelBackground: '#29303c', clusterBkg: '#1e232d',
    tertiaryColor: '#29303c', secondaryColor: '#29303c', loopTextColor: '#e6e9ef', sequenceNumberColor: '#161a22',
    clusterBorder: '#6c7686', fontFamily: fuente, fontSize: '14px', darkMode: true },
};
// En la variante oscura, los colores comunes (classDef y cajas de las secuencias) pasan a su par oscuro.
const oscuros = [
  ['fill:#e8f0fe,stroke:#3b6db3,color:#102a43', 'fill:#1d2b45,stroke:#7aa2e3,color:#dbe7ff'],
  ['fill:#e6f4ea,stroke:#2f7d32,color:#0b2e13', 'fill:#16301f,stroke:#5fbf6b,color:#d7f5dc'],
  ['fill:#fff4e0,stroke:#b26a00,color:#3d2400', 'fill:#3a2a12,stroke:#e0a040,color:#ffe7c2'],
  ['fill:#fde8e8,stroke:#c62828,color:#4a0d0d', 'fill:#3d1a1a,stroke:#f07a7a,color:#ffdcdc'],
  ['rgb(232,240,254)', 'rgb(29,43,69)'], ['rgb(230,244,234)', 'rgb(22,48,31)'],
  ['rgb(255,244,224)', 'rgb(58,42,18)'], ['rgb(253,232,232)', 'rgb(61,26,26)'],
];

// Saneado: el panel muestra los SVG como imágenes, pero igual no deben llevar código ni recursos de
// afuera (TestDiagramasSaneados lo comprueba).
function sanear(svg) {
  return svg
    .replace(/<script[\s\S]*?<\/script>/gi, '')
    .replace(/<foreignObject[\s\S]*?<\/foreignObject>/gi, '')
    .replace(/\s+on[a-z]+\s*=\s*("[^"]*"|'[^']*')/gi, '')
    .replace(/\s+(xlink:)?href\s*=\s*("(?!#)[^"]*"|'(?!#)[^']*')/gi, '')
    .replace(/@import[^;]+;/gi, '');
}

function frontMatter(texto) {
  const m = /^---\n([\s\S]*?)\n---\n/.exec(texto);
  const meta = {};
  if (!m) return meta;
  for (const linea of m[1].split('\n')) {
    const c = /^([a-z_]+):\s*(.*)$/.exec(linea);
    if (!c) continue;
    const v = c[2].trim();
    meta[c[1]] = v.startsWith('[') ? v.slice(1, -1).split(',').map((x) => x.trim()).filter(Boolean) : v.replace(/^"(.*)"$/, '$1');
  }
  return meta;
}

const archivos = fs.readdirSync(origen).filter((f) => /^\d\d-.+\.md$/.test(f)).sort();
const diagramas = [];
for (const archivo of archivos) {
  const texto = fs.readFileSync(path.join(origen, archivo), 'utf8').replace(/\r\n/g, '\n');
  const bloque = (/```mermaid\n([\s\S]*?)\n```/.exec(texto) || [])[1];
  if (!bloque) continue;
  const id = archivo.replace(/\.md$/, '');
  diagramas.push({ id, archivo, bloque, meta: frontMatter(texto) });
}

// Un markdown por variante con todos los bloques: mmdc los dibuja en una sola sesión del navegador.
fs.rmSync(destino, { recursive: true, force: true });
fs.mkdirSync(destino, { recursive: true });
for (const [variante, tema] of Object.entries(temas)) {
  const md = diagramas.map((d) => {
    let b = d.bloque;
    if (variante === 'oscuro') for (const [claro, oscuro] of oscuros) b = b.split(claro).join(oscuro);
    return '```mermaid\n' + b + '\n```\n';
  }).join('\n');
  const entrada = path.join(tmp, `${variante}.md`);
  fs.writeFileSync(entrada, md);
  const config = path.join(tmp, `${variante}.json`);
  fs.writeFileSync(config, JSON.stringify({ theme: 'base', securityLevel: 'strict', htmlLabels: false,
    flowchart: { htmlLabels: false }, fontFamily: fuente, themeVariables: tema }));
  console.log(`Dibujando ${diagramas.length} diagramas (${variante})…`);
  // Fondo transparente: el visor pone el fondo del tema del panel.
  execFileSync('npx', ['--yes', '-p', MERMAID_CLI, 'mmdc', '-q', '-b', 'transparent', '-p', path.join(raiz, 'scripts', 'puppeteer.json'),
    '-c', config, '-i', entrada, '-o', path.join(tmp, `${variante}-salida.md`)], { stdio: 'inherit', shell: process.platform === 'win32' });
  diagramas.forEach((d, k) => {
    const svg = sanear(fs.readFileSync(path.join(tmp, `${variante}-salida-${k + 1}.svg`), 'utf8'));
    fs.writeFileSync(path.join(destino, `${d.id}.${variante}.svg.gz`), zlib.gzipSync(svg, { level: 9 }));
  });
}

const manifiesto = {
  generador: 'scripts/docs/generar-diagramas.mjs',
  mermaid_cli: MERMAID_CLI,
  diagramas: diagramas.map((d) => ({
    id: d.id,
    fuente: `diagramas/${d.archivo}`,
    titulo: d.meta.titulo || '',
    descripcion: d.meta.descripcion || '',
    audiencia: d.meta.audiencia || [],
    pantallas: d.meta.pantallas || [],
    secciones_manual: d.meta.secciones_manual || [],
    alt: [d.meta.titulo, d.meta.descripcion].filter(Boolean).join(': '),
    hash: createHash('sha256').update(d.bloque).digest('hex'),
    claro: `${d.id}.claro.svg.gz`,
    oscuro: `${d.id}.oscuro.svg.gz`,
  })),
};
fs.writeFileSync(path.join(destino, 'manifiesto.json'), JSON.stringify(manifiesto, null, 2) + '\n');
fs.rmSync(tmp, { recursive: true, force: true });
console.log(`Listo: ${diagramas.length} diagramas en ${path.relative(raiz, destino)}`);
