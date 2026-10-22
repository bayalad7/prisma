import { chromium, firefox, webkit } from 'playwright';
import fs from 'node:fs';
const svg = fs.readFileSync(process.argv[2], 'utf8');
const panel = "default-src 'self'; img-src 'self' data: blob:; style-src 'self'; script-src 'self'";
for (const [nombre, tipo] of [['chromium', chromium], ['firefox', firefox], ['webkit', webkit]]) {
  let b; try { b = await tipo.launch(); } catch (e) { console.log(nombre, 'no disponible'); continue; }
  const p = await b.newPage();
  for (const [caso, csp] of [['csp-panel', panel], ['csp-minima', "default-src 'none'; style-src 'unsafe-inline'"]]) {
    await p.route('**/d.svg', (r) => r.fulfill({ body: svg, headers: { 'content-type': 'image/svg+xml', 'content-security-policy': csp } }));
    await p.route('**/pagina', (r) => r.fulfill({ body: '<!doctype html><img id=i src="/d.svg" width=600>', headers: { 'content-type': 'text/html', 'content-security-policy': panel } }));
    await p.goto('http://local.prueba/pagina'); await p.waitForTimeout(500);
    // Colorea los píxeles: con estilos, los nodos tienen relleno de color; sin estilos, quedan negros.
    const color = await p.evaluate(async () => {
      const i = document.getElementById('i'); await i.decode();
      const c = document.createElement('canvas'); c.width = i.width; c.height = i.height;
      const x = c.getContext('2d'); x.drawImage(i, 0, 0);
      const d = x.getImageData(0, 0, c.width, c.height).data; let negros = 0, color = 0;
      for (let k = 0; k < d.length; k += 4) { if (d[k + 3] < 10) continue; if (d[k] < 40 && d[k + 1] < 40 && d[k + 2] < 40) negros++; else if (Math.abs(d[k] - d[k + 1]) > 8 || Math.abs(d[k + 1] - d[k + 2]) > 8) color++; }
      return { negros, color };
    });
    console.log(nombre, caso, JSON.stringify(color));
    await p.unroute('**/d.svg'); await p.unroute('**/pagina');
  }
  await b.close();
}
