// Fase 6.3: rollo en vivo. Una app real imprime por /v1 y el ticket aparece solo. Uso: node fase63.mjs <carpeta-salida>
// Requiere NODE_TLS_REJECT_UNAUTHORIZED=0 (puente de prueba con certificado autofirmado).
import { chromium } from 'playwright';
import { AxeBuilder } from '@axe-core/playwright';
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { fileURLToPath } from 'node:url';

const SALIDA = process.argv[2];
const B = 'https://127.0.0.1:18443';
const CLAVE = 'contrasena-demo-2026';
const AQUI = path.dirname(fileURLToPath(import.meta.url));
const LLAVE = fs.readFileSync(path.join(AQUI, 'llave-dev.txt'), 'utf8').trim();
fs.mkdirSync(SALIDA, { recursive: true });
const espera = (ms) => new Promise((r) => setTimeout(r, ms));
const informe = { axe: {}, desborde: {}, consola: [], comprobaciones: [], peticiones: [] };
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
async function abrir(browser, vp, ruta) {
  const ctx = await browser.newContext({ ...VISTAS[vp], ignoreHTTPSErrors: true, locale: 'es-MX' });
  const page = await ctx.newPage();
  page.on('console', (m) => { if (['error', 'warning'].includes(m.type()) && !/Failed to load resource/.test(m.text())) informe.consola.push(`${vp}: ${m.text()}`); });
  page.on('pageerror', (e) => informe.consola.push(`${vp} pageerror: ${e.message}`));
  await page.goto(`${B}${ruta}`); await espera(800);
  if (await page.locator('#contrasena').count()) { await page.fill('#contrasena', CLAVE); await page.getByRole('button', { name: 'Entrar', exact: true }).click(); }
  await page.waitForSelector('.vista .esqueleto', { state: 'detached', timeout: 15000 }).catch(() => {});
  await page.waitForLoadState('networkidle'); await espera(800);
  return { ctx, page };
}
const cap = (page, nombre, completa = true) => page.screenshot({ path: path.join(SALIDA, `${nombre}.png`), fullPage: completa });
// Imprime como lo haría una aplicación real: /v1/imprimir con origen aprobado y llave.
async function imprimirDesdeApp(alias, referencia) {
  const r = await fetch(`${B}/v1/imprimir`, { method: 'POST', headers: { 'Content-Type': 'application/json', Origin: 'https://dev.ejemplo.mx', Authorization: `Bearer ${LLAVE}` },
    body: JSON.stringify({ version: '1', id: crypto.randomUUID(), impresora: alias, referencia, operaciones: [
      { tipo: 'texto', valor: 'MI APP', alineacion: 'centro', negrita: true, tamano: [2, 2] },
      { tipo: 'par', izquierda: '1 x Café americano', derecha: '$38.00' },
      { tipo: 'qr', valor: 'https://dev.ejemplo.mx/p/1', alineacion: 'centro' },
      { tipo: 'cortar' }] }) });
  return { estado: r.status, datos: await r.json() };
}

const browser = await chromium.launch();
{
  const { ctx, page } = await abrir(browser, 'escritorio', '/admin/pruebas/rollo/prueba-2');
  page.on('request', (r) => { if (r.url().includes('/rollo')) informe.peticiones.push(`${r.method()} ${r.headers()['x-prisma-sondeo'] || '-'} ${r.url().replace(B, '')}`); });
  comprobar('la ruta abre el rollo con «Probar impresión» marcado', await page.getByRole('heading', { name: 'Rollo de «prueba-2»' }).isVisible()
    && await page.locator('nav a[aria-current="page"]').first().innerText().then((t) => t.includes('Probar impresión')).catch(() => false));
  comprobar('indica que está en vivo', await page.getByText('En vivo: revisa cada 2 segundos').isVisible());
  const antes = await page.locator('.ticket-rollo').count();
  const r = await imprimirDesdeApp('prueba-2', 'APP-001');
  comprobar('la app imprime por /v1', r.estado === 200 && r.datos.ok === true, JSON.stringify(r.datos).slice(0, 120));
  await page.waitForFunction((n) => document.querySelectorAll('.ticket-rollo').length > n, antes, { timeout: 6000 }).catch(() => {});
  await espera(800);
  const primera = page.locator('.ticket-rollo').first();
  comprobar('el ticket aparece solo, arriba, con la app y la referencia', await primera.getByText('Mi app en desarrollo · APP-001').isVisible(),
    await primera.locator('strong').first().innerText().catch(() => ''));
  comprobar('muestra el trabajo_id', (await primera.innerText()).includes(r.datos.trabajo_id || 'x'));
  comprobar('se anuncia a los lectores de pantalla', (await page.locator('[aria-live="polite"].solo-lector').innerText()).includes('Mi app en desarrollo'));
  comprobar('los sondeos llevan la cabecera que no renueva la sesión', informe.peticiones.filter((p) => p.startsWith('GET 1 /admin/api/impresoras/prueba-2/rollo')).length >= 2,
    `${informe.peticiones.length} peticiones`);
  await cap(page, '01-rollo-escritorio');
  await revisar(page, 'rollo-escritorio');

  await primera.getByRole('button', { name: 'Ver en grande' }).click(); await page.waitForSelector('dialog img'); await espera(300);
  await cap(page, '02-ampliar-escritorio', false);
  await page.keyboard.press('Escape'); await espera(200);
  await primera.getByText('Ver los comandos ESC/POS').click(); await espera(700);
  comprobar('la traza se muestra al abrirla', (await primera.locator('details pre').innerText()).includes('inicializar impresora'));

  await page.getByRole('button', { name: 'Pausar' }).click(); await espera(200);
  const n1 = await page.locator('.ticket-rollo').count();
  await imprimirDesdeApp('prueba-2', 'APP-002'); await espera(3000);
  comprobar('en pausa no llegan tickets nuevos', await page.locator('.ticket-rollo').count() === n1 && await page.getByText('En pausa', { exact: true }).isVisible());
  await page.getByRole('button', { name: 'Reanudar' }).click(); await espera(1500);
  comprobar('al reanudar aparece el pendiente', await page.locator('.ticket-rollo').count() === n1 + 1);

  // Ir a otra pantalla detiene el sondeo.
  await page.getByRole('link', { name: 'Inicio' }).first().click(); await espera(500);
  const cuenta = informe.peticiones.length;
  await espera(4500);
  comprobar('al salir de la pantalla deja de sondear', informe.peticiones.length === cuenta, `${informe.peticiones.length - cuenta}`);
  await page.goBack(); await espera(1500);
  comprobar('volver con «atrás» abre el rollo otra vez', await page.getByRole('heading', { name: 'Rollo de «prueba-2»' }).isVisible());

  await page.getByRole('button', { name: 'Vaciar rollo' }).click(); await page.waitForSelector('dialog'); await espera(200);
  await page.locator('dialog').getByRole('button', { name: 'Vaciar rollo' }).click(); await espera(1500);
  comprobar('vaciar deja el rollo vacío y lo explica', await page.locator('.ticket-rollo').count() === 0 && await page.getByText('Todavía no llegó ningún ticket').isVisible());
  await cap(page, '03-vacio-escritorio');
  await revisar(page, 'vacio-escritorio');

  await page.goto(`${B}/admin/pruebas/rollo/caja`); await espera(1500);
  comprobar('una impresora no virtual lo explica', await page.getByText('No hay una impresora virtual «caja»').isVisible());
  await cap(page, '04-no-virtual-escritorio', false);
  await ctx.close();
}
{
  const { ctx, page } = await abrir(browser, 'celular', '/admin/pruebas/rollo/prueba-2');
  await imprimirDesdeApp('prueba-2', 'APP-003');
  await page.waitForSelector('.ticket-rollo', { timeout: 6000 }).catch(() => {}); await espera(800);
  comprobar('celular: el ticket llega', await page.locator('.ticket-rollo').count() === 1);
  await cap(page, '05-rollo-celular');
  await revisar(page, 'rollo-celular');
  await ctx.close();
}
{
  // Desde la tarjeta de la impresora, «Ver tickets» abre el rollo.
  const { ctx, page } = await abrir(browser, 'escritorio', '/admin/impresoras');
  await page.locator('.lista > .tarjeta').filter({ hasText: 'prueba-2' }).first().getByRole('button', { name: /^Ver tickets/ }).click(); await espera(1200);
  comprobar('«Ver tickets» abre el rollo', page.url().endsWith('/admin/pruebas/rollo/prueba-2'));
  await ctx.close();
}
await browser.close();
fs.writeFileSync(path.join(SALIDA, 'informe.json'), JSON.stringify({ ...informe, peticiones: informe.peticiones.slice(0, 20) }, null, 2));
console.log(informe.comprobaciones.join('\n'));
const viol = Object.entries(informe.axe).filter(([, v]) => v.length);
console.log('axe:', viol.length ? JSON.stringify(viol) : 'sin violaciones');
console.log('desbordes:', JSON.stringify(Object.entries(informe.desborde).filter(([, v]) => v > 0)));
console.log('consola:', informe.consola.length ? informe.consola.join('\n') : 'sin errores');
