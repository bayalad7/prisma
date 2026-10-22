// Fase 4, entrega 1: el manual de docs/ dentro del panel. Uso: node fase4e1.mjs <carpeta-salida>
import { chromium } from 'playwright';
import { AxeBuilder } from '@axe-core/playwright';
import fs from 'node:fs';
import path from 'node:path';

const SALIDA = process.argv[2];
const B = 'https://127.0.0.1:18443';
const CLAVE = 'contrasena-demo-2026';
fs.mkdirSync(SALIDA, { recursive: true });
const espera = (ms) => new Promise((r) => setTimeout(r, ms));
const informe = { axe: {}, desborde: {}, consola: [], comprobaciones: [], documentos: [] };
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
// Navega dentro del panel (como un enlace), sin recargar: una petición por página, al ritmo de una persona.
async function irA(page, ruta) {
  await page.evaluate((r) => { history.pushState(null, '', r); dispatchEvent(new PopStateEvent('popstate')); }, ruta);
  await espera(700); await listo(page);
}
const cap = (page, nombre, completa = false) => page.screenshot({ path: path.join(SALIDA, `${nombre}.png`), fullPage: completa });

const browser = await chromium.launch();
{
  const { ctx, page } = await abrir(browser, 'escritorio', '/admin/ayuda');
  comprobar('la Ayuda muestra los grupos del manual', await page.locator('.grupos-manual .tarjeta').count() >= 9);
  await cap(page, '01-ayuda-escritorio', true);
  await revisar(page, 'ayuda-escritorio');

  // Todos los documentos del índice: sin errores, con título y sin Markdown sin dibujar.
  const indice = await page.evaluate(() => fetch('/admin/api/manual').then((r) => r.json()));
  const rutas = indice.grupos.flatMap((g) => g.documentos.map((d) => d.ruta));
  let problemas = 0;
  for (const ruta of rutas) {
    const errores = informe.consola.length;
    await irA(page, `/admin/ayuda/${ruta}`);
    const r = await page.evaluate(() => {
      const art = document.querySelector('.documento-manual');
      if (!art) return { sinArticulo: true };
      const clon = art.cloneNode(true);
      clon.querySelectorAll('pre, code').forEach((x) => x.remove());
      const t = clon.textContent;
      const restos = (t.match(/\]\(|\*\*|^\s*\|\s*-{3}|^#{1,4} /gm) || []).length;
      const anclasRotas = [...art.querySelectorAll('a[data-ancla]')].filter((a) => !document.getElementById(decodeURIComponent(a.getAttribute('href').slice(1)))).map((a) => a.getAttribute('href'));
      const internos = [...art.querySelectorAll('a[href^="/admin/ayuda/"]')].map((a) => a.getAttribute('href'));
      return { h1: art.querySelector('h1')?.textContent || '', restos, anclasRotas, internos, externos: art.querySelectorAll('a.enlace-externo').length };
    });
    const mal = r.sinArticulo || !r.h1 || r.restos > 0 || r.anclasRotas.length > 0 || informe.consola.length > errores;
    if (mal) problemas++;
    informe.documentos.push({ ruta, ...r, errores: informe.consola.length - errores });
  }
  comprobar(`los ${rutas.length} documentos se dibujan sin errores ni restos de Markdown`, problemas === 0, `${problemas} con problemas`);

  // Enlaces internos: cada /admin/ayuda/… existe en el índice, y su ancla en el documento destino.
  const internos = [...new Set(informe.documentos.flatMap((d) => d.internos || []))];
  let rotos = [];
  for (const href of internos) {
    const [ruta, ancla] = href.replace('/admin/ayuda/', '').split('#');
    if (!rutas.includes(decodeURIComponent(ruta))) { rotos.push(href); continue; }
    if (ancla) {
      await irA(page, href);
      if (!(await page.evaluate((id) => !!document.getElementById(id), decodeURIComponent(ancla)))) rotos.push(href);
    }
  }
  comprobar(`los ${internos.length} enlaces entre páginas del manual llevan a su página y su ancla`, rotos.length === 0, rotos.slice(0, 5).join(' '));

  // Navegar sin recargar, con ancla: de la guía de impresoras virtuales a «Diagnóstico y páginas de códigos».
  await irA(page, '/admin/ayuda/administracion/probar-sin-impresora.md');
  await page.evaluate(() => { window.__sinRecargar = true; });
  await page.locator('.documento-manual a[href*="impresoras.md#"]').first().click(); await listo(page); await espera(400);
  comprobar('un enlace interno abre la otra página sin recargar y va a su sección',
    await page.evaluate(() => window.__sinRecargar === true) && page.url().includes('/admin/ayuda/administracion/impresoras.md#')
    && await page.evaluate(() => document.activeElement?.tagName === 'H2'), page.url());
  await cap(page, '02-ancla-escritorio');
  await page.goBack(); await listo(page);
  comprobar('«atrás» vuelve a la página anterior del manual', page.url().endsWith('/administracion/probar-sin-impresora.md'));
  comprobar('las capturas del manual se cargan', await page.evaluate(() => [...document.querySelectorAll('.imagen-manual')].every((i) => i.complete && i.naturalWidth > 0)));
  await cap(page, '03-con-imagenes-escritorio', true);
  await revisar(page, 'documento-escritorio');

  await irA(page, '/admin/ayuda/referencia/contrato-documento-v1.md');
  comprobar('tablas, listas y código dentro de listas', await page.locator('.documento-manual table').count() > 3
    && await page.locator('.documento-manual li pre').count() >= 1);
  await cap(page, '04-referencia-escritorio');
  await revisar(page, 'referencia-escritorio');
  const ext = page.locator('.documento-manual a.enlace-externo').first();
  comprobar('los enlaces externos abren otra pestaña y lo dicen', await ext.getAttribute('target') === '_blank' && /otra pestaña|Internet/.test(await ext.innerText()));

  await irA(page, '/admin/ayuda/diagramas/29-impresora-virtual.md');
  comprobar('los diagramas muestran su código plegado y sin el front-matter', await page.getByText('Ver el código del diagrama (Mermaid)').isVisible()
    && !(await page.locator('.documento-manual').innerText()).includes('secciones_manual'));
  await irA(page, '/admin/ayuda/administracion/no-existe.md');
  comprobar('una página que no existe lo explica', await page.getByText('Esa página del manual no existe').isVisible());
  await ctx.close();
}
{
  const { ctx, page } = await abrir(browser, 'celular', '/admin/ayuda/administracion/impresoras.md');
  await cap(page, '05-documento-celular');
  await revisar(page, 'documento-celular');
  await irA(page, '/admin/ayuda/referencia/api-v1.md');
  await revisar(page, 'tablas-celular');
  await cap(page, '06-tablas-celular');
  await ctx.close();
}
await browser.close();
fs.writeFileSync(path.join(SALIDA, 'informe.json'), JSON.stringify(informe, null, 2));
console.log(informe.comprobaciones.join('\n'));
const viol = Object.entries(informe.axe).filter(([, v]) => v.length);
console.log('axe:', viol.length ? JSON.stringify(viol) : 'sin violaciones');
console.log('desbordes:', JSON.stringify(Object.entries(informe.desborde).filter(([, v]) => v > 0)));
console.log('consola:', informe.consola.length ? informe.consola.slice(0, 10).join('\n') : 'sin errores');
console.log('documentos con problemas:', JSON.stringify(informe.documentos.filter((d) => d.sinArticulo || !d.h1 || d.restos || (d.anclasRotas || []).length || d.errores).map((d) => [d.ruta, d.restos, d.anclasRotas])));
