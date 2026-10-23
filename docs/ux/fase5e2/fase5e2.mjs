// Fase 5, entrega 2: SVG servidos, visor de diagramas y diagramas dibujados en el manual. Uso: node fase5e2.mjs <carpeta-salida>
import { chromium } from 'playwright';
import { AxeBuilder } from '@axe-core/playwright';
import fs from 'node:fs';
import path from 'node:path';

const SALIDA = process.argv[2];
const B = 'https://127.0.0.1:18443';
const CLAVE = 'contrasena-demo-2026';
fs.mkdirSync(SALIDA, { recursive: true });
const espera = (ms) => new Promise((r) => setTimeout(r, ms));
const informe = { axe: {}, desborde: {}, consola: [], comprobaciones: [] };
const comprobar = (nombre, ok, detalle = '') => informe.comprobaciones.push(`${ok ? 'ok   ' : 'FALLA'} ${nombre} ${detalle}`);
const VISTAS = {
  escritorio: { viewport: { width: 1366, height: 768 } },
  celular: { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 },
};
async function revisar(page, clave) {
  const a = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze();
  informe.axe[clave] = a.violations.map((v) => ({ id: v.id, impacto: v.impact, nodos: v.nodes.length, ej: v.nodes.slice(0, 2).map((x) => x.target.join(' ')) }));
  informe.desborde[clave] = await page.evaluate(() => document.documentElement.scrollWidth - innerWidth);
}
async function listo(page) {
  await page.waitForSelector('.vista .esqueleto', { state: 'detached', timeout: 15000 }).catch(() => {});
  await page.waitForLoadState('networkidle'); await espera(300);
}
async function abrir(browser, vp, ruta) {
  const ctx = await browser.newContext({ ...VISTAS[vp], ignoreHTTPSErrors: true, locale: 'es-MX', colorScheme: 'light' });
  const page = await ctx.newPage();
  page.on('console', (m) => { if (['error', 'warning'].includes(m.type()) && !/Failed to load resource/.test(m.text())) informe.consola.push(`${vp}: ${m.text()}`); });
  page.on('pageerror', (e) => informe.consola.push(`${vp} pageerror: ${e.message}`));
  await page.goto(`${B}${ruta}`); await espera(800);
  if (await page.locator('#contrasena').count()) { await page.fill('#contrasena', CLAVE); await page.getByRole('button', { name: 'Entrar', exact: true }).click(); }
  await listo(page);
  return { ctx, page };
}
async function irA(page, ruta) {
  await page.evaluate((r) => { history.pushState(null, '', r); dispatchEvent(new PopStateEvent('popstate')); }, ruta);
  await espera(700); await listo(page);
}
const imagenLista = (page, sel = '.visor-lienzo img') => page.waitForFunction((s) => {
  const i = document.querySelector(s); return i && i.complete && i.naturalWidth > 0;
}, sel, { timeout: 8000 }).then(() => true).catch(() => false);
const cap = (page, nombre, completa = false) => page.screenshot({ path: path.join(SALIDA, `${nombre}.png`), fullPage: completa });

const browser = await chromium.launch();
{
  const { ctx, page } = await abrir(browser, 'escritorio', '/admin/ayuda/diagramas/07-orden-autorizacion.md');
  comprobar('la página del diagrama lo muestra dibujado', await imagenLista(page) && await page.locator('.documento-manual > pre.codigo').count() === 0
    && (await page.locator('.visor-lienzo img').getAttribute('src')).endsWith('.claro.svg'));
  comprobar('sin front-matter a la vista y con «Fuentes en el código» plegado', !(await page.locator('.documento-manual').innerText()).includes('secciones_manual')
    && await page.locator('.documento-manual details summary', { hasText: 'Fuentes en el código' }).count() === 1);
  comprobar('dice para quién es', await page.locator('.visor-publico .chip').count() >= 1);
  await cap(page, '01-diagrama-claro-escritorio');
  await revisar(page, 'diagrama-claro');

  // Zoom y movimiento con botones y teclado.
  await page.getByRole('button', { name: 'Acercar' }).click(); await espera(100);
  const t1 = await page.locator('.visor-lienzo img').evaluate((i) => i.style.transform);
  await page.locator('.visor-lienzo').focus(); await page.keyboard.press('+'); await page.keyboard.press('ArrowLeft'); await espera(100);
  const t2 = await page.locator('.visor-lienzo img').evaluate((i) => i.style.transform);
  await page.keyboard.press('0'); await espera(100);
  const t3 = await page.locator('.visor-lienzo img').evaluate((i) => i.style.transform);
  comprobar('acercar, mover con el teclado y volver al tamaño original', t1.includes('scale(1.25)') && t2.includes('scale(1.5625)') && t2.includes('translate(40px') && t3.includes('scale(1)'), `${t1} | ${t2} | ${t3}`);

  // Ver en grande y salir con Escape.
  await page.getByRole('button', { name: 'Ver en grande' }).click(); await espera(300);
  comprobar('«Ver en grande» ocupa la ventana', await page.locator('figure.visor.visor-grande').count() === 1
    && await page.locator('figure.visor-grande .visor-lienzo').evaluate((e) => e.getBoundingClientRect().height > 500));
  await cap(page, '02-ver-en-grande-escritorio');
  await page.keyboard.press('Escape'); await espera(200);
  comprobar('Escape vuelve al tamaño normal', await page.locator('figure.visor.visor-grande').count() === 0);

  // Descargar y copiar el código.
  const href = await page.locator('.visor-barra a[download]').getAttribute('href');
  const descarga = await page.evaluate((u) => fetch(u).then((r) => ({ s: r.status, t: r.headers.get('content-type'), d: r.headers.get('content-disposition') })), href);
  comprobar('«Descargar SVG» baja el archivo', descarga.s === 200 && descarga.t === 'image/svg+xml' && (descarga.d || '').includes('attachment'), JSON.stringify(descarga));
  comprobar('ofrece copiar el código Mermaid', await page.getByRole('button', { name: 'Copiar código Mermaid' }).count() === 1);

  // El tema oscuro cambia la imagen sola.
  await page.getByRole('button', { name: /Oscuro/ }).first().click(); await espera(300);
  comprobar('en tema oscuro usa la variante oscura', (await page.locator('.visor-lienzo img').getAttribute('src')).endsWith('.oscuro.svg') && await imagenLista(page));
  await cap(page, '03-diagrama-oscuro-escritorio');
  await revisar(page, 'diagrama-oscuro');
  await page.getByRole('button', { name: /Claro/ }).first().click(); await espera(300);

  // Los 27 diagramas se dibujan.
  const { diagramas } = await page.evaluate(() => fetch('/admin/api/diagramas').then((r) => r.json()));
  let bien = 0; const mal = [];
  for (const d of diagramas) {
    await irA(page, `/admin/ayuda/${d.fuente}`);
    if (await imagenLista(page)) bien++; else mal.push(d.id);
  }
  comprobar(`los ${diagramas.length} diagramas se dibujan en su página`, bien === diagramas.length, mal.join(' '));

  // {{diagrama:id}} dentro de una guía.
  await irA(page, '/admin/ayuda/administracion/trabajos.md');
  comprobar('{{diagrama:…}} pone el visor en la guía', await imagenLista(page) && !(await page.locator('.documento-manual').innerText()).includes('{{diagrama')
    && await page.getByRole('link', { name: 'Ver la explicación del diagrama' }).count() === 1);
  await page.locator('figure.visor').scrollIntoViewIfNeeded(); await espera(200);
  await cap(page, '04-inclusion-escritorio');
  await revisar(page, 'inclusion');

  // También en el panel lateral de ayuda.
  await irA(page, '/admin/trabajos');
  await page.getByRole('button', { name: 'Ayuda de esta pantalla' }).click(); await page.waitForSelector('dialog.panel-ayuda[open]'); await espera(400);
  comprobar('el panel de ayuda muestra el diagrama', await imagenLista(page, 'dialog.panel-ayuda .visor-lienzo img'));
  await page.keyboard.press('Escape'); await espera(200);

  // Si la imagen no llega, el código.
  await page.route('**/admin/diagramas/**', (r) => r.abort());
  await irA(page, '/admin/ayuda/diagramas/24-idempotencia.md');
  await espera(800);
  comprobar('si la imagen no carga, muestra el código Mermaid', await page.getByText('No se encontró la imagen del diagrama').isVisible()
    && (await page.locator('.visor-lienzo pre').innerText()).includes('flowchart'));
  await page.unroute('**/admin/diagramas/**');
  await ctx.close();
}
{
  const { ctx, page } = await abrir(browser, 'celular', '/admin/ayuda/diagramas/29-impresora-virtual.md');
  comprobar('celular: el diagrama cabe a lo ancho', await imagenLista(page));
  await cap(page, '05-diagrama-celular');
  await revisar(page, 'diagrama-celular');
  await page.getByRole('button', { name: 'Ver en grande' }).click(); await espera(300);
  await cap(page, '06-grande-celular');
  await revisar(page, 'grande-celular');
  await ctx.close();
}
await browser.close();
fs.writeFileSync(path.join(SALIDA, 'informe.json'), JSON.stringify(informe, null, 2));
console.log(informe.comprobaciones.join('\n'));
const viol = Object.entries(informe.axe).filter(([, v]) => v.length);
console.log('axe:', viol.length ? JSON.stringify(viol) : 'sin violaciones');
console.log('desbordes:', JSON.stringify(Object.entries(informe.desborde).filter(([, v]) => v > 0)));
console.log('consola:', informe.consola.length ? informe.consola.slice(0, 8).join('\n') : 'sin errores');
