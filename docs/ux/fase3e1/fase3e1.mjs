// Fase 3, entrega 1: navegación. Rutas reales, enlaces viejos con #, atrás, atajos, menú en celular,
// tema oscuro, axe, desborde y consola. Uso: node fase3e1.mjs <carpeta-salida>
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

async function listo(page) {
  await page.waitForSelector('.vista .esqueleto', { state: 'detached', timeout: 15000 }).catch(() => {});
  await page.waitForLoadState('networkidle'); await espera(300);
}
async function revisar(page, clave) {
  const r = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze();
  informe.axe[clave] = r.violations.map((v) => ({ id: v.id, impacto: v.impact, nodos: v.nodes.length, ej: v.nodes.slice(0, 2).map((n) => n.target.join(' ')) }));
  informe.desborde[clave] = await page.evaluate(() => document.documentElement.scrollWidth - innerWidth);
}

const browser = await chromium.launch();
for (const [vp, op] of Object.entries({
  escritorio: { viewport: { width: 1366, height: 768 } },
  celular: { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 },
})) {
  const ctx = await browser.newContext({ ...op, ignoreHTTPSErrors: true, locale: 'es-MX' });
  const page = await ctx.newPage();
  page.on('console', (m) => { if (['error', 'warning'].includes(m.type())) informe.consola.push(`${vp}: ${m.text()}`); });
  page.on('pageerror', (e) => informe.consola.push(`${vp} pageerror: ${e.message}`));

  // Enlace viejo de 0.1.0: entra y termina en la ruta nueva.
  await page.goto(`${B}/admin/#clientes`); await espera(800);
  if (await page.locator('#contrasena').count()) { await page.fill('#contrasena', CLAVE); await page.getByRole('button', { name: 'Entrar' }).click(); }
  await listo(page);
  comprobar(`${vp}: #clientes redirige`, page.url().endsWith('/admin/aplicaciones'), page.url());
  comprobar(`${vp}: título de la página`, (await page.title()).startsWith('Aplicaciones'), await page.title());
  comprobar(`${vp}: el foco queda en el h1`, await page.evaluate(() => document.activeElement?.tagName === 'H1'));

  await page.goto(`${B}/admin/`); await listo(page);
  await page.screenshot({ path: path.join(SALIDA, `01-inicio-${vp}.png`), fullPage: vp === 'escritorio' });
  await revisar(page, `inicio-${vp}`);
  const primero = await page.evaluate(() => document.querySelector('#raiz a[href], #raiz button')?.className);
  await page.locator('a.saltar').focus();
  const visible = await page.evaluate(() => document.querySelector('a.saltar').getBoundingClientRect().top >= 0);
  comprobar(`${vp}: «Saltar al contenido» es lo primero y se ve con el foco`, primero === 'saltar' && visible, primero);
  if (vp === 'escritorio') await page.screenshot({ path: path.join(SALIDA, `06-saltar-${vp}.png`) });

  if (vp === 'celular') {
    await page.getByRole('button', { name: 'Menú' }).click(); await espera(250);
    await page.screenshot({ path: path.join(SALIDA, `02-menu-${vp}.png`) });
    await revisar(page, `menu-${vp}`);
    await page.keyboard.press('Escape'); await espera(150);
    comprobar(`${vp}: Escape cierra el menú`, await page.getByRole('button', { name: 'Menú' }).getAttribute('aria-expanded') === 'false');
    await page.locator('.nav-inferior a', { hasText: 'Trabajos' }).click();
  } else {
    await page.locator('.nav a', { hasText: 'Trabajos' }).click();
  }
  await listo(page);
  comprobar(`${vp}: el menú lleva a /admin/trabajos`, page.url().endsWith('/admin/trabajos'), page.url());
  await page.screenshot({ path: path.join(SALIDA, `03-trabajos-${vp}.png`) });
  await revisar(page, `trabajos-${vp}`);
  await page.goBack(); await listo(page);
  comprobar(`${vp}: Atrás vuelve a Inicio`, new URL(page.url()).pathname === '/admin/' && await page.locator('h1').first().isVisible());

  // Carga directa de una ruta y recarga.
  await page.goto(`${B}/admin/imagenes`); await listo(page);
  comprobar(`${vp}: carga directa de /admin/imagenes`, await page.locator('h1', { hasText: 'Imágenes' }).isVisible());
  comprobar(`${vp}: marca la sección actual`, await page.locator('.nav a[aria-current="page"]').first().getAttribute('data-seccion') === 'imagenes');

  if (vp === 'escritorio') {
    await page.keyboard.press('g'); await page.keyboard.press('p'); await listo(page);
    comprobar('atajo g p abre Impresoras', page.url().endsWith('/admin/impresoras'), page.url());
    await page.keyboard.press('?'); await espera(300);
    await page.screenshot({ path: path.join(SALIDA, `04-atajos-${vp}.png`) });
    await revisar(page, `atajos-${vp}`);
    await page.locator('#atajos-activos').uncheck({ force: true });
    await page.keyboard.press('Escape'); await espera(200);
    await page.keyboard.press('g'); await page.keyboard.press('t'); await espera(500);
    comprobar('con los atajos desactivados, g t no navega', page.url().endsWith('/admin/impresoras'), page.url());
    await page.locator('.segmentos.tema button[aria-label="Oscuro"]').click(); await espera(300);
    await page.goto(`${B}/admin/solicitudes`); await listo(page);
    await page.screenshot({ path: path.join(SALIDA, `05-solicitudes-oscuro-${vp}.png`) });
    await revisar(page, `solicitudes-oscuro-${vp}`);
    comprobar('el contador de solicitudes se ve', await page.locator('.nav [data-seccion="solicitudes"] .contador').isVisible());
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
