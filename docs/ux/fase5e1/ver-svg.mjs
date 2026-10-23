import { chromium } from 'playwright';
import fs from 'node:fs'; import zlib from 'node:zlib'; import path from 'node:path';
const dir = process.argv[2], salida = process.argv[3]; fs.mkdirSync(salida, { recursive: true });
const b = await chromium.launch(); const p = await b.newPage({ viewport: { width: 1100, height: 800 } });
for (const id of process.argv.slice(4)) for (const v of ['claro', 'oscuro']) {
  const svg = zlib.gunzipSync(fs.readFileSync(path.join(dir, `${id}.${v}.svg.gz`))).toString('utf8');
  await p.setContent(`<body style="margin:0;padding:16px;background:${v === 'claro' ? '#ffffff' : '#161a22'}"><img style="max-width:1060px" src="data:image/svg+xml;base64,${Buffer.from(svg).toString('base64')}"></body>`);
  await p.waitForTimeout(300); await p.screenshot({ path: path.join(salida, `${id}-${v}.png`), fullPage: true });
}
await b.close();
