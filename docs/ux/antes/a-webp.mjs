// Convierte los PNG de una carpeta a WebP (calidad 0.85) con Chromium y borra los PNG.
import { chromium } from 'playwright';
import fs from 'node:fs'; import path from 'node:path';
const dir = process.argv[2];
const br = await chromium.launch(); const p = await br.newPage();
for (const f of fs.readdirSync(dir).filter((f) => f.endsWith('.png'))) {
  const b64 = fs.readFileSync(path.join(dir, f)).toString('base64');
  const webp = await p.evaluate(async (src) => {
    const img = new Image(); img.src = src; await img.decode();
    const c = document.createElement('canvas'); c.width = img.naturalWidth; c.height = img.naturalHeight;
    c.getContext('2d').drawImage(img, 0, 0);
    return c.toDataURL('image/webp', 0.85);
  }, 'data:image/png;base64,' + b64);
  if (!webp.startsWith('data:image/webp')) throw new Error('sin WebP: ' + f);
  fs.writeFileSync(path.join(dir, f.replace(/\.png$/, '.webp')), Buffer.from(webp.split(',')[1], 'base64'));
  fs.unlinkSync(path.join(dir, f));
}
await br.close();
