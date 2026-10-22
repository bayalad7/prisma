// Fase 3, entrega 5: Trabajos con detalle. Uso: node fase3e5.mjs <carpeta-salida> <llave pk_ de «Mi app en desarrollo»>
import { chromium } from 'playwright';
import { AxeBuilder } from '@axe-core/playwright';
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';

const [SALIDA, LLAVE] = process.argv.slice(2);
const B = 'https://127.0.0.1:18443';
const CLAVE = 'contrasena-demo-2026';
process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';
fs.mkdirSync(SALIDA, { recursive: true });
const espera = (ms) => new Promise((r) => setTimeout(r, ms));
const informe = { axe: {}, desborde: {}, consola: [], comprobaciones: [] };
const comprobar = (nombre, ok, detalle = '') => informe.comprobaciones.push(`${ok ? 'ok   ' : 'FALLA'} ${nombre} ${detalle}`);
const VISTAS = {
  escritorio: { viewport: { width: 1366, height: 768 } },
  celular: { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 },
};

// Un ticket nuevo en la impresora virtual «prueba-2», para ver su imagen en el detalle.
const ref = `VIR-${Date.now() % 10000}`;
const r = await fetch(`${B}/v1/imprimir`, { method: 'POST', headers: { 'Content-Type': 'application/json', Origin: 'https://dev.ejemplo.mx', Authorization: `Bearer ${LLAVE}` },
  body: JSON.stringify({ version: '1', id: crypto.randomUUID(), impresora: 'prueba-2', referencia: ref, operaciones: [{ tipo: 'texto', valor: 'Café con acentos: ñandú', alineacion: 'centro' }, { tipo: 'cortar' }] }) });
comprobar('ticket de prueba en la virtual', r.ok, String(r.status));

async function revisar(page, clave) {
  const a = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze();
  informe.axe[clave] = a.violations.map((v) => ({ id: v.id, impacto: v.impact, nodos: v.nodes.length, ej: v.nodes.slice(0, 2).map((x) => x.target.join(' ')) }));
  informe.desborde[clave] = await page.evaluate(() => document.documentElement.scrollWidth - innerWidth);
}
async function listo(page) {
  await page.waitForSelector('.esqueleto', { state: 'detached', timeout: 15000 }).catch(() => {});
  await page.waitForLoadState('networkidle'); await espera(400);
}
async function abrir(browser, vp, ruta) {
  const ctx = await browser.newContext({ ...VISTAS[vp], ignoreHTTPSErrors: true, locale: 'es-MX' });
  const page = await ctx.newPage();
  page.on('console', (m) => { if (['error', 'warning'].includes(m.type())) informe.consola.push(`${vp}: ${m.text()}`); });
  page.on('pageerror', (e) => informe.consola.push(`${vp} pageerror: ${e.message}`));
  await page.goto(`${B}${ruta}`); await espera(800);
  if (await page.locator('#contrasena').count()) { await page.fill('#contrasena', CLAVE); await page.getByRole('button', { name: 'Entrar' }).click(); }
  await listo(page);
  return { ctx, page };
}
const cap = (page, nombre, completa = true) => page.screenshot({ path: path.join(SALIDA, `${nombre}.png`), fullPage: completa });
const cerrar = async (page) => { await page.keyboard.press('Escape'); await page.waitForSelector('dialog', { state: 'detached' }).catch(() => {}); };

const browser = await chromium.launch();
{
  const { ctx, page } = await abrir(browser, 'escritorio', '/admin/');
  await page.getByRole('link', { name: 'Ver qué pasó' }).click(); await listo(page);
  comprobar('«Ver qué pasó» abre los fallidos', page.url().endsWith('/admin/trabajos?estado=fallido') &&
    await page.locator('.vista .segmentos button[aria-pressed="true"]').textContent() === 'Fallidos', page.url());
  await cap(page, '01-fallidos-escritorio');
  await revisar(page, 'fallidos-escritorio');
  await page.getByRole('button', { name: /Ver el detalle de el trabajo/ }).first().click();
  await page.waitForSelector('dialog'); await espera(300);
  comprobar('el detalle explica qué pasó y qué hacer', await page.locator('dialog').getByText('Qué hacer').isVisible());
  const probar = page.locator('dialog').getByRole('button', { name: 'Probar la impresora' });
  if (await probar.count()) { await probar.click(); await page.locator('dialog .nota-caja').waitFor({ timeout: 15000 }).catch(() => {}); }
  comprobar('«Probar la impresora» muestra el resultado ahí mismo', await page.locator('dialog .nota-caja').count() > 0);
  await cap(page, '02-detalle-fallido-escritorio', false);
  await revisar(page, 'detalle-fallido-escritorio');
  await cerrar(page);

  await page.goto(`${B}/admin/trabajos`); await listo(page);
  await cap(page, '03-todos-escritorio');
  await revisar(page, 'todos-escritorio');
  await page.getByRole('button', { name: 'Con aviso' }).click(); await listo(page);
  const filas = await page.locator('tbody tr').count();
  comprobar('«Con aviso» filtra los trabajos impresos con aviso', filas > 0, `${filas} filas`);
  await page.getByRole('button', { name: /Ver el detalle de el trabajo/ }).first().click();
  await page.waitForSelector('dialog'); await espera(300);
  comprobar('el aviso se explica en lenguaje llano', await page.locator('dialog').getByText(/no existe en la impresora/).isVisible());
  await cap(page, '04-detalle-aviso-escritorio', false);
  await cerrar(page);

  await page.getByRole('button', { name: 'Todos' }).click(); await listo(page);
  await page.getByRole('button', { name: `Ver el detalle de el trabajo ${ref}` }).click();
  await page.waitForSelector('dialog'); await espera(800);
  comprobar('el detalle de un trabajo virtual muestra su ticket', await page.locator('dialog .ticket img').evaluate((i) => i.complete && i.naturalWidth > 0).catch(() => false));
  await cap(page, '05-detalle-virtual-escritorio', false);
  await cerrar(page);
  await ctx.close();
}
{
  const { ctx, page } = await abrir(browser, 'celular', '/admin/trabajos?estado=fallido');
  await cap(page, '06-fallidos-celular');
  await revisar(page, 'fallidos-celular');
  await page.getByRole('button', { name: /Ver el detalle de el trabajo/ }).first().click();
  await page.waitForSelector('dialog'); await espera(300);
  await cap(page, '07-detalle-celular', false);
  await revisar(page, 'detalle-celular');
  await ctx.close();
}
await browser.close();
fs.writeFileSync(path.join(SALIDA, 'informe.json'), JSON.stringify(informe, null, 2));
console.log(informe.comprobaciones.join('\n'));
const viol = Object.entries(informe.axe).filter(([, v]) => v.length);
console.log('axe:', viol.length ? JSON.stringify(viol) : 'sin violaciones');
console.log('desbordes:', JSON.stringify(Object.entries(informe.desborde).filter(([, v]) => v > 0)));
console.log('consola:', informe.consola.length ? informe.consola.join('\n') : 'sin errores');
