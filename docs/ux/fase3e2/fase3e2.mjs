// Fase 3, entrega 2: Inicio. A) puente recién instalado que sigue los primeros pasos; B) puente con
// datos y problemas por revisar. Uso: node fase3e2.mjs <carpeta-salida>
import { chromium } from 'playwright';
import { AxeBuilder } from '@axe-core/playwright';
import fs from 'node:fs';
import path from 'node:path';

const SALIDA = process.argv[2];
const NUEVO = 'https://127.0.0.1:8444';
const DEMO = 'https://127.0.0.1:18443';
const CLAVE = 'contrasena-demo-2026';
fs.mkdirSync(SALIDA, { recursive: true });
const espera = (ms) => new Promise((r) => setTimeout(r, ms));
const informe = { axe: {}, desborde: {}, consola: [], comprobaciones: [] };
const comprobar = (nombre, ok, detalle = '') => informe.comprobaciones.push(`${ok ? 'ok   ' : 'FALLA'} ${nombre} ${detalle}`);
const VISTAS = {
  escritorio: { viewport: { width: 1366, height: 768 } },
  celular: { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 },
};

async function listo(page) {
  await page.waitForSelector('.vista .esqueleto', { state: 'detached', timeout: 15000 }).catch(() => {});
  await page.waitForLoadState('networkidle'); await espera(400);
}
async function revisar(page, clave) {
  const r = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze();
  informe.axe[clave] = r.violations.map((v) => ({ id: v.id, impacto: v.impact, nodos: v.nodes.length, ej: v.nodes.slice(0, 2).map((n) => n.target.join(' ')) }));
  informe.desborde[clave] = await page.evaluate(() => document.documentElement.scrollWidth - innerWidth);
}
async function abrir(browser, base, vp, tema = 'light') {
  const ctx = await browser.newContext({ ...VISTAS[vp], ignoreHTTPSErrors: true, locale: 'es-MX', colorScheme: tema });
  const page = await ctx.newPage();
  page.on('console', (m) => { if (['error', 'warning'].includes(m.type())) informe.consola.push(`${vp}: ${m.text()}`); });
  page.on('pageerror', (e) => informe.consola.push(`${vp} pageerror: ${e.message}`));
  await page.goto(`${base}/admin/`); await espera(800);
  if (await page.locator('#nueva').count()) { await page.fill('#nueva', CLAVE); await page.fill('#repetir', CLAVE); await page.getByRole('button', { name: 'Crear contraseña y entrar' }).click(); }
  else if (await page.locator('#contrasena').count()) { await page.fill('#contrasena', CLAVE); await page.getByRole('button', { name: 'Entrar' }).click(); }
  await listo(page);
  return { ctx, page };
}
const avance = (page) => page.locator('.tarjeta-titulo p').first().textContent().catch(() => '');

const browser = await chromium.launch();

// A) Recién instalado, en escritorio: sigue los pasos.
{
  const { ctx, page } = await abrir(browser, NUEVO, 'escritorio');
  await page.screenshot({ path: path.join(SALIDA, '01-nuevo-inicio-escritorio.png'), fullPage: true });
  await revisar(page, 'nuevo-inicio-escritorio');
  comprobar('recién instalado: 1 de 4 pasos', (await avance(page)).includes('1 de 4'), await avance(page));
  await page.locator('.lista-pasos').getByRole('button', { name: 'Crear impresora virtual de prueba' }).click();
  await listo(page);
  comprobar('crear la virtual lleva a Impresoras', page.url().endsWith('/admin/impresoras'), page.url());
  await page.goto(`${NUEVO}/admin/`); await listo(page);
  comprobar('con impresora: 2 de 5 pasos (sugiere la física)', (await avance(page)).includes('2 de 5'), await avance(page));
  await page.screenshot({ path: path.join(SALIDA, '02-nuevo-con-virtual-escritorio.png'), fullPage: true });
  await page.getByRole('button', { name: /Imprimir prueba/ }).click();
  await page.waitForSelector('dialog'); await espera(600);
  await page.screenshot({ path: path.join(SALIDA, '03-nuevo-prueba-escritorio.png') });
  await revisar(page, 'nuevo-prueba-escritorio');
  await page.keyboard.press('Escape'); await listo(page);
  comprobar('tras la prueba: 3 de 5 pasos', (await avance(page)).includes('3 de 5'), await avance(page));
  await page.locator('.tarjeta').filter({ hasText: 'Primeros pasos' }).getByRole('button', { name: 'Ocultar' }).click(); await espera(300);
  comprobar('ocultar deja el botón para retomarlos', await page.getByRole('button', { name: /Ver primeros pasos \(3 de 5\)/ }).isVisible());
  await page.screenshot({ path: path.join(SALIDA, '04-nuevo-pasos-ocultos-escritorio.png') });
  await page.reload(); await listo(page);
  comprobar('siguen ocultos al recargar', await page.getByRole('button', { name: /Ver primeros pasos/ }).isVisible());
  await page.getByRole('button', { name: /Ver primeros pasos/ }).click(); await espera(200);
  await ctx.close();
}
// A) Recién instalado, en celular.
{
  const { ctx, page } = await abrir(browser, NUEVO, 'celular');
  await page.screenshot({ path: path.join(SALIDA, '05-nuevo-inicio-celular.png'), fullPage: true });
  await revisar(page, 'nuevo-inicio-celular');
  await ctx.close();
}
// B) Puente con datos y problemas.
for (const [vp, tema] of [['escritorio', 'light'], ['celular', 'light'], ['escritorio', 'dark']]) {
  const { ctx, page } = await abrir(browser, DEMO, vp, tema);
  const sufijo = `${vp}${tema === 'dark' ? '-oscuro' : ''}`;
  await page.screenshot({ path: path.join(SALIDA, `06-demo-inicio-${sufijo}.png`), fullPage: true });
  await revisar(page, `demo-inicio-${sufijo}`);
  if (vp === 'escritorio' && tema === 'light') {
    comprobar('muestra cuántas cosas revisar', await page.locator('.encabezado .insignia').textContent().then((t) => /por revisar/.test(t)));
    await page.getByRole('link', { name: /Revisar «/ }).click(); await listo(page);
    comprobar('«Revisar» de la impresora lleva a Impresoras', page.url().endsWith('/admin/impresoras'));
  }
  await ctx.close();
}
await browser.close();
fs.writeFileSync(path.join(SALIDA, 'informe.json'), JSON.stringify(informe, null, 2));
console.log(informe.comprobaciones.join('\n'));
const viol = Object.entries(informe.axe).filter(([, v]) => v.length);
console.log('axe:', viol.length ? JSON.stringify(viol) : 'sin violaciones');
console.log('desbordes:', JSON.stringify(Object.entries(informe.desborde).filter(([, v]) => v > 0)));
console.log('consola:', informe.consola.length ? informe.consola.join('\n') : 'sin errores');
