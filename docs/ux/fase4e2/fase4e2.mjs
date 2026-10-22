// Fase 4, entrega 2: contenido nuevo del manual y búsqueda local. Uso: node fase4e2.mjs <carpeta-salida>
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
  const ctx = await browser.newContext({ ...VISTAS[vp], ignoreHTTPSErrors: true, locale: 'es-MX' });
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
const cap = (page, nombre, completa = false) => page.screenshot({ path: path.join(SALIDA, `${nombre}.png`), fullPage: completa });

const browser = await chromium.launch();
{
  const { ctx, page } = await abrir(browser, 'escritorio', '/admin/ayuda');
  comprobar('el manual empieza por «Primeros pasos»', (await page.locator('.grupos-manual .tarjeta').first().locator('a').first().innerText()) === 'Primeros pasos');
  await page.locator('#ay-buscar').fill('olvide contrasena'); await espera(900);
  comprobar('buscar sin acentos encuentra la sección', await page.locator('.resultados-manual a', { hasText: 'Olvidé la contraseña del panel' }).first().isVisible());
  comprobar('resalta las palabras en el extracto', await page.locator('.resultados-manual mark').count() > 0);
  comprobar('la búsqueda queda en la dirección', page.url().includes('?q=olvide'));
  await cap(page, '01-busqueda-escritorio', true);
  await revisar(page, 'busqueda-escritorio');
  await page.locator('.resultados-manual a', { hasText: 'Olvidé la contraseña del panel' }).first().click(); await listo(page); await espera(500);
  comprobar('un resultado lleva a su sección', page.url().includes('#olvid') && await page.evaluate(() => /^H[2-4]$/.test(document.activeElement?.tagName || '')), page.url());
  await page.goBack(); await listo(page); await espera(600);
  comprobar('«atrás» vuelve a la búsqueda con sus resultados', (await page.locator('#ay-buscar').inputValue()) === 'olvide contrasena'
    && await page.locator('.resultados-manual li').count() > 0);
  await page.locator('#ay-buscar').fill('palabraquenoexiste'); await espera(900);
  comprobar('sin resultados lo dice', await page.getByText(/No hay resultados para «palabraquenoexiste»/).isVisible());

  for (const ruta of ['primeros-pasos', 'conceptos', 'tareas-frecuentes', 'solucion-de-problemas', 'preguntas-frecuentes']) {
    await irA(page, `/admin/ayuda/administracion/${ruta}.md`);
    const r = await page.evaluate(() => ({ h1: document.querySelector('.documento-manual h1')?.textContent, imgs: [...document.querySelectorAll('.imagen-manual')].length }));
    comprobar(`«${r.h1}» se dibuja`, !!r.h1, `${r.imgs} imágenes`);
  }
  await irA(page, '/admin/ayuda/administracion/primeros-pasos.md');
  await page.waitForFunction(() => [...document.querySelectorAll('.imagen-manual')].every((i) => i.complete), null, { timeout: 10000 }).catch(() => {});
  comprobar('las 10 capturas de Primeros pasos se cargan', await page.evaluate(() => {
    const imgs = [...document.querySelectorAll('.imagen-manual')];
    return imgs.length === 10 && imgs.every((i) => i.naturalWidth > 0);
  }));
  await cap(page, '02-primeros-pasos-escritorio', true);
  await revisar(page, 'primeros-pasos-escritorio');
  await irA(page, '/admin/ayuda/administracion/solucion-de-problemas.md');
  await cap(page, '03-solucion-escritorio');
  await revisar(page, 'solucion-escritorio');

  // Buscador del índice lateral: lleva a la portada con la búsqueda.
  await page.locator('#ay-buscar-indice').fill('bluetooth'); await page.locator('#ay-buscar-indice').press('Enter'); await listo(page); await espera(900);
  comprobar('el buscador de una página lleva a los resultados', page.url().includes('/admin/ayuda?q=bluetooth') && await page.locator('.resultados-manual li').count() > 0);
  await ctx.close();
}
{
  const { ctx, page } = await abrir(browser, 'celular', '/admin/ayuda?q=certificado');
  await espera(900);
  comprobar('celular: abrir con ?q= muestra los resultados', await page.locator('.resultados-manual li').count() > 0);
  await cap(page, '04-busqueda-celular');
  await revisar(page, 'busqueda-celular');
  await irA(page, '/admin/ayuda/administracion/conceptos.md');
  await cap(page, '05-conceptos-celular');
  await revisar(page, 'conceptos-celular');
  await ctx.close();
}
await browser.close();
fs.writeFileSync(path.join(SALIDA, 'informe.json'), JSON.stringify(informe, null, 2));
console.log(informe.comprobaciones.join('\n'));
const viol = Object.entries(informe.axe).filter(([, v]) => v.length);
console.log('axe:', viol.length ? JSON.stringify(viol) : 'sin violaciones');
console.log('desbordes:', JSON.stringify(Object.entries(informe.desborde).filter(([, v]) => v > 0)));
console.log('consola:', informe.consola.length ? informe.consola.slice(0, 8).join('\n') : 'sin errores');
